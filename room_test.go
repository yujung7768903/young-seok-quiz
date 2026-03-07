package main

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// 실행: go test -race -run TestConcurrentJoin
// 게임 참여 시 고루틴 간 데이터 경쟁 테스트
func TestConcurrentJoin(t *testing.T) {
	roomId := "ROOM01"
	hub := newHub()
	host := newTestClient()
	data, _ := json.Marshal(map[string]string{
		"room_id":  roomId,
		"nickname": "host",
	})
	handleJoinRoom(hub, host, data)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			user := newTestClient()
			data, _ := json.Marshal(map[string]string{
				"room_id":  roomId,
				"nickname": fmt.Sprintf("user%d", i),
			})
			handleJoinRoom(hub, user, data)
		}(i)
	}
	wg.Wait()
	time.Sleep(20 * time.Millisecond)

	room, ok := hub.getRoom(roomId)
	if !ok {
		t.Fatalf("room not found(room id: %s)", roomId)
	}
	if room.clientCount() != 11 {
		t.Fatalf("expected client counts: 11, actual: %d", room.clientCount())
	}
	if room.hostID == "" {
		t.Fatal("hostId shoule not be empty")
	}
}

func newTestClient() *Client {
	return &Client{
		id:   randomID(8, "abcdefghijklmnopqrstuvwxyz0123456789"),
		send: make(chan []byte, 256),
	}
}
