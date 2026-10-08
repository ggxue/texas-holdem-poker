package poker

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type connection struct {
	id     string
	socket *websocket.Conn
	send   chan []byte
	ctx    context.Context
	cancel context.CancelFunc
}

func (a *App) serveSocket(w http.ResponseWriter, r *http.Request) {
	id, credential, err := a.identify(r)
	if err != nil || credential != "" {
		writeJSON(w, 401, view{Error: "identity_required"})
		return
	}
	a.mu.Lock()
	if a.closed {
		a.mu.Unlock()
		writeJSON(w, 503, view{Error: "restoring"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	state, err := a.store.read(ctx)
	cancel()
	if err != nil {
		a.mu.Unlock()
		writeJSON(w, 503, view{Error: "restoring"})
		return
	}
	upgrader := websocket.Upgrader{HandshakeTimeout: 5 * time.Second}
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		a.mu.Unlock()
		return
	}
	ctx, cancel = context.WithCancel(r.Context())
	c := &connection{id: id, socket: socket, send: make(chan []byte, 16), ctx: ctx, cancel: cancel}
	a.clients[c] = struct{}{}
	a.queue(c, state.visibleTo(id))
	a.mu.Unlock()
	socket.SetReadLimit(4096)
	writerDone := make(chan struct{})
	go func() { defer close(writerDone); c.writeLoop() }()
	defer func() {
		cancel()
		_ = socket.Close()
		<-writerDone
		a.mu.Lock()
		delete(a.clients, c)
		a.mu.Unlock()
	}()
	for {
		if _, _, err := socket.ReadMessage(); err != nil {
			return
		}
		// Commands use the HTTP endpoint; this socket carries confirmed views.
		a.mu.Lock()
		a.queue(c, view{Error: "invalid_command"})
		a.mu.Unlock()
	}
}

func (c *connection) writeLoop() {
	defer c.cancel()
	defer c.socket.Close()
	for {
		select {
		case <-c.ctx.Done():
			return
		case message := <-c.send:
			if err := c.socket.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				return
			}
			if err := c.socket.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		}
	}
}

// Caller holds the application lock; a slow connection cannot block room commands.
func (a *App) queue(c *connection, v view) {
	data, err := json.Marshal(v)
	if err != nil {
		c.cancel()
		_ = c.socket.Close()
		return
	}
	select {
	case c.send <- data:
	default:
		c.cancel()
		_ = c.socket.Close()
	}
}

func (a *App) broadcast(state room) {
	for c := range a.clients {
		a.queue(c, state.visibleTo(c.id))
	}
}
