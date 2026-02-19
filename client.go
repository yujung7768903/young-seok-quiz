package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Client represents a connected WebSocket user
type Client struct {
	id       string
	nickname string
	room     *Room
	conn     *websocket.Conn
	send     chan []byte
	isHost   bool
}

// InMsg is an incoming WebSocket message from the client
type InMsg struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	c := &Client{
		id:   randomID(8, "abcdefghijklmnopqrstuvwxyz0123456789"),
		conn: conn,
		send: make(chan []byte, 256),
	}
	go c.writePump()
	go c.readPump(hub)
}

func (c *Client) readPump(hub *Hub) {
	defer func() {
		if c.room != nil {
			c.room.unregister <- c
		}
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		var m InMsg
		if err := json.Unmarshal(msg, &m); err != nil {
			log.Println("JSON unmarshal error:", err)
			continue
		}
		handleMessage(hub, c, m)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ──────────────────────────────────────────────
// Message Handler
// ──────────────────────────────────────────────

func handleMessage(hub *Hub, c *Client, m InMsg) {
	log.Printf("message type: %s", m.Type)
	switch m.Type {
	case "create_room":
		handleCreateRoom(hub, c, m.Data)
	case "join_room":
		handleJoinRoom(hub, c, m.Data)
	case "start_game":
		handleStartGame(c)
	case "type1_submit_ranking":
		handleType1SubmitRanking(c, m.Data)
	case "type1_submit_guess":
		handleType1SubmitGuess(c, m.Data)
	case "ready_start":
		handleReadyStart(hub, c, m.Data)
	case "ready_next":
		handleReadyNext(c)
	case "type2_submit_answer":
		handleType2SubmitAnswer(c, m.Data)
	default:
		log.Printf("unknown message type: %s", m.Type)
	}
}

func handleCreateRoom(hub *Hub, c *Client, data json.RawMessage) {
	var d struct {
		Nickname string `json:"nickname"`
	}
	if err := json.Unmarshal(data, &d); err != nil || d.Nickname == "" {
		sendError(c, "닉네임을 입력해주세요.")
		return
	}
	room := hub.createRoom()
	c.nickname = d.Nickname
	c.room = room
	c.isHost = true
	room.hostID = c.id
	room.register <- c

	room.sendTo(c, "room_created", map[string]interface{}{
		"room_id":   room.id,
		"player_id": c.id,
		"players":   []PlayerInfo{{ID: c.id, Nickname: c.nickname, IsHost: true}},
	})
}

func handleJoinRoom(hub *Hub, c *Client, data json.RawMessage) {
	var d struct {
		RoomID   string `json:"room_id"`
		Nickname string `json:"nickname"`
	}
	if err := json.Unmarshal(data, &d); err != nil || d.Nickname == "" {
		sendError(c, "닉네임을 입력해주세요.")
		return
	}
	room, ok := hub.getRoom(d.RoomID)
	if !ok {
		sendError(c, "방을 찾을 수 없습니다.")
		return
	}
	if room.game != nil {
		sendError(c, "이미 게임이 시작된 방입니다.")
		return
	}

	// Build player list before registering
	existingPlayers := room.playerList()
	newPlayer := PlayerInfo{ID: c.id, Nickname: d.Nickname, IsHost: false}
	allPlayers := append(existingPlayers, newPlayer)

	c.nickname = d.Nickname
	c.room = room
	c.isHost = false
	room.register <- c

	room.sendTo(c, "room_joined", map[string]interface{}{
		"room_id":   room.id,
		"player_id": c.id,
		"players":   allPlayers,
		"is_host":   false,
	})
	room.broadcastExcept(c.id, "player_joined", map[string]interface{}{
		"player":  newPlayer,
		"players": allPlayers,
	})
}

func handleStartGame(c *Client) {
	log.Println("handleStartGame start!")
	if c.room == nil {
		sendError(c, "예상치 못한 문제로 게임을 시작할 수 없습니다. 새로운 게임을 생성해주세요.")
		return
	}
	if c.room.clientCount() < 2 {
		sendError(c, "2명 이상이어야 게임을 시작할 수 있습니다.")
		return
	}
	if c.room.game != nil {
		sendError(c, "이미 게임이 진행 중입니다.")
		return
	}
	game := newGame(c.room)
	c.room.game = game
	game.start()
}

func handleType1SubmitRanking(c *Client, data json.RawMessage) {
	if c.room == nil || c.room.game == nil {
		return
	}
	var d struct {
		Ranking []int `json:"ranking"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return
	}
	c.room.game.onType1QuestionerRanking(c, d.Ranking)
}

func handleType1SubmitGuess(c *Client, data json.RawMessage) {
	if c.room == nil || c.room.game == nil {
		return
	}
	var d struct {
		Ranking []int `json:"ranking"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return
	}
	c.room.game.onType1PlayerGuess(c, d.Ranking)
}

func handleReadyStart(hub *Hub, c *Client, data json.RawMessage) {
	log.Println("[handleReadyStart] start!")
	var d struct {
		RoomId string `json:"room_id"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		sendError(c, "방을 찾을 수 없습니다.")
		return
	}
	room, ok := hub.getRoom(d.RoomId)
	if !ok {
		sendError(c, "방을 찾을 수 없습니다.")
		return
	}

	if room.game != nil {
		sendError(c, "이미 게임이 시작된 방입니다.")
		return
	}

	room.mu.Lock()
	room.startReadyPlayers[c.id] = true
	room.mu.Unlock()

	total := room.clientCount()
	log.Printf("[handleReadyStart] ready_count: %d", len(room.startReadyPlayers))
	room.broadcastJSON("ready_start_update", map[string]interface{}{
		"ready_count": len(room.startReadyPlayers),
		"total_count": total,
	})
	if len(room.startReadyPlayers) >= total {
		handleStartGame(c)
	}
}

func handleReadyNext(c *Client) {
	if c.room == nil || c.room.game == nil {
		return
	}
	c.room.game.onReadyNext(c)
}

func handleType2SubmitAnswer(c *Client, data json.RawMessage) {
	if c.room == nil || c.room.game == nil {
		return
	}
	var d struct {
		Answer string `json:"answer"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return
	}
	c.room.game.onType2Answer(c, d.Answer)
}

func sendError(c *Client, msg string) {
	b, _ := json.Marshal(map[string]interface{}{
		"type": "error",
		"data": map[string]string{"message": msg},
	})
	safeSend(c.send, b)
}
