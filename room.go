package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"sync"
)

// Hub manages all rooms
type Hub struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

func newHub() *Hub {
	return &Hub{rooms: make(map[string]*Room)}
}

func (h *Hub) createRoom() *Room {
	h.mu.Lock()
	defer h.mu.Unlock()
	var id string
	for {
		id = randomID(6, "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
		if _, exists := h.rooms[id]; !exists {
			break
		}
	}
	room := newRoom(id, h)
	h.rooms[id] = room
	go room.run()
	return room
}

func (h *Hub) getRoom(id string) (*Room, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	r, ok := h.rooms[id]
	return r, ok
}

func (h *Hub) removeRoom(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms, id)
}

// PlayerInfo is the JSON-serializable player info
type PlayerInfo struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	IsHost   bool   `json:"is_host"`
}

// Room represents a quiz room
type Room struct {
	id         string
	hub        *Hub
	clients    map[string]*Client
	hostID     string
	register   chan *Client
	unregister chan *Client
	game       *Game
	mu         sync.RWMutex
}

func newRoom(id string, hub *Hub) *Room {
	return &Room{
		id:         id,
		hub:        hub,
		clients:    make(map[string]*Client),
		register:   make(chan *Client, 32),
		unregister: make(chan *Client, 32),
	}
}

func (r *Room) run() {
	for {
		select {
		case c := <-r.register:
			r.mu.Lock()
			r.clients[c.id] = c
			r.mu.Unlock()

		case c := <-r.unregister:
			r.mu.Lock()
			if _, ok := r.clients[c.id]; !ok {
				r.mu.Unlock()
				continue
			}
			delete(r.clients, c.id)
			safeClose(c.send)

			if len(r.clients) == 0 {
				r.mu.Unlock()
				r.hub.removeRoom(r.id)
				return
			}

			// If host left, reassign
			if c.isHost {
				for _, nc := range r.clients {
					nc.isHost = true
					r.hostID = nc.id
					break
				}
			}

			players := r.playerListLocked()
			r.mu.Unlock()

			r.broadcastJSON("player_left", map[string]interface{}{
				"player_id": c.id,
				"nickname":  c.nickname,
				"players":   players,
			})
		}
	}
}

// playerListLocked returns player list; caller must hold r.mu (at least RLock)
func (r *Room) playerListLocked() []PlayerInfo {
	list := make([]PlayerInfo, 0, len(r.clients))
	for _, c := range r.clients {
		list = append(list, PlayerInfo{ID: c.id, Nickname: c.nickname, IsHost: c.isHost})
	}
	return list
}

func (r *Room) playerList() []PlayerInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.playerListLocked()
}

func (r *Room) clientCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

func (r *Room) getClient(id string) (*Client, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.clients[id]
	return c, ok
}

func (r *Room) allClientsExcept(excludeID string) []*Client {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Client, 0)
	for id, c := range r.clients {
		if id != excludeID {
			result = append(result, c)
		}
	}
	return result
}

func (r *Room) broadcastJSON(msgType string, data interface{}) {
	b := mustMarshal(msgType, data)
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, c := range r.clients {
		safeSend(c.send, b)
	}
}

func (r *Room) broadcastExcept(excludeID, msgType string, data interface{}) {
	b := mustMarshal(msgType, data)
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id, c := range r.clients {
		if id != excludeID {
			safeSend(c.send, b)
		}
	}
}

func (r *Room) sendTo(c *Client, msgType string, data interface{}) {
	b := mustMarshal(msgType, data)
	safeSend(c.send, b)
}

func mustMarshal(msgType string, data interface{}) []byte {
	b, err := json.Marshal(map[string]interface{}{"type": msgType, "data": data})
	if err != nil {
		log.Println("marshal error:", err)
		return nil
	}
	return b
}

func safeSend(ch chan []byte, data []byte) {
	if data == nil {
		return
	}
	defer func() { recover() }()
	select {
	case ch <- data:
	default:
	}
}

func safeClose(ch chan []byte) {
	defer func() { recover() }()
	close(ch)
}

func randomID(n int, chars string) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
