package main

import (
	"fmt"
	"sync"
)

type Event struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type LogHub struct {
	clients map[chan Event]bool
	mu      sync.Mutex
}

var globalHub = &LogHub{
	clients: make(map[chan Event]bool),
}

func (h *LogHub) Register() chan Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan Event, 10)
	h.clients[ch] = true
	return ch
}

func (h *LogHub) Unregister(ch chan Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, ch)
	close(ch)
}

func (h *LogHub) Broadcast(e Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- e:
		default:
			// Client chậm, bỏ qua
		}
	}
}

// Hàm tiện ích để log nhanh từ bất kỳ đâu
func BroadcastLog(payload string, eventType string) {
	fmt.Printf("📡 [Broadcast] %s: %s\n", eventType, payload)
	globalHub.Broadcast(Event{Type: eventType, Payload: payload})
}
