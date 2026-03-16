package api

import (
	"fmt"
	"net/http"
	"sync"
)

// EventHub manages Server-Sent Events connections.
type EventHub struct {
	mu      sync.RWMutex
	clients map[chan string]struct{}
}

// NewEventHub creates a new SSE event hub.
func NewEventHub() *EventHub {
	return &EventHub{
		clients: make(map[chan string]struct{}),
	}
}

// Broadcast sends an event to all connected SSE clients.
func (h *EventHub) Broadcast(eventType, data string) {
	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
			// Client buffer full, skip.
		}
	}
}

// ServeHTTP handles SSE connections (GET /api/events).
func (h *EventHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan string, 16)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
		close(ch)
	}()

	// Send initial connected event.
	fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			fmt.Fprint(w, msg)
			flusher.Flush()
		}
	}
}
