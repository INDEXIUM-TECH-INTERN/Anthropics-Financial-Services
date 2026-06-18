package pubsub

import (
	"fmt"
	"sync"
)

type Event struct {
	Type     string                 `json:"type"`
	Payload  string                 `json:"payload"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type LogHub struct {
	clients map[chan Event]bool
	mu      sync.Mutex
}

var GlobalHub = &LogHub{
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

// BroadcastEvent allows sending structured metadata
func BroadcastEvent(payload string, eventType string, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	fmt.Printf("📡 [Event:%s] %s %v\n", eventType, payload, metadata)
	GlobalHub.Broadcast(Event{Type: eventType, Payload: payload, Metadata: metadata})
}

// BroadcastLog is kept for legacy unstructured logs
func BroadcastLog(payload string, eventType string) {
	BroadcastEvent(payload, eventType, nil)
}
