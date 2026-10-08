package poker

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type connection struct {
	id         string
	socket     *websocket.Conn
	send       chan notification
	ctx        context.Context
	cancel     context.CancelFunc
	pageID     string
	generation int64
}

type notification struct {
	data     []byte
	terminal bool
}

func (a *App) serveSocket(w http.ResponseWriter, r *http.Request) {
	id, credential, err := a.identify(r)
	if err != nil || credential != "" {
		writeJSON(w, 401, view{Error: "identity_required"})
		return
	}
	a.mu.Lock()
	a.tickLocked()
	if a.closed || a.fault != "" {
		a.mu.Unlock()
		writeJSON(w, 503, view{Error: "unavailable"})
		return
	}
	upgrader := websocket.Upgrader{HandshakeTimeout: 5 * time.Second}
	pageID := r.URL.Query().Get("pageID")
	generation, _ := strconv.ParseInt(r.URL.Query().Get("control"), 10, 64)
	control := a.state.Controls[id]
	seated := false
	for _, occupant := range a.state.Seats {
		seated = seated || occupant == id
	}
	if !seated || pageID == "" || control.PageID != pageID || control.Generation != generation {
		a.mu.Unlock()
		writeJSON(w, 403, view{Error: "taken_over"})
		return
	}
	if a.state.Version == math.MaxInt64 {
		a.mu.Unlock()
		writeJSON(w, 503, view{Error: "version_exhausted"})
		return
	}
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		a.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	c := &connection{id: id, socket: socket, send: make(chan notification, 16), ctx: ctx, cancel: cancel, pageID: pageID, generation: generation}
	if old := a.active[id]; old != nil {
		old.cancel()
		_ = old.socket.Close()
	}
	a.active[id] = c
	if !a.state.Connecting[id].IsZero() || !a.state.Disconnected[id].IsZero() { // 有效握手才结束连接建立或重连等待。
		delete(a.state.Connecting, id)   // 清除握手阶段期限。
		delete(a.state.Disconnected, id) // 清除已观察断线的期限。
		a.state.Version++                // 连接已建立也是权威状态更新。
		a.broadcast(a.state)             // 向仍有效的控制页同步恢复。
		a.scheduleLocked()               // 保留原行动期限，取消失效的连接期限。
	}
	a.clients[c] = struct{}{}
	a.queue(c, a.visibleTo(id))
	a.mu.Unlock()
	socket.SetReadLimit(4096)
	_ = socket.SetReadDeadline(time.Now().Add(45 * time.Second))
	socket.SetPongHandler(func(string) error { return socket.SetReadDeadline(time.Now().Add(45 * time.Second)) })
	writerDone := make(chan struct{})
	go func() { defer close(writerDone); c.writeLoop() }()
	defer func() {
		cancel()
		_ = socket.Close()
		<-writerDone
		a.mu.Lock()
		delete(a.clients, c)
		if a.active[id] == c {
			delete(a.active, id)
			a.disconnectLocked(id, c.pageID, c.generation)
		}
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
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.socket.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				return
			}
			if err := c.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case message := <-c.send:
			if err := c.socket.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
				return
			}
			if err := c.socket.WriteMessage(websocket.TextMessage, message.data); err != nil {
				return
			}
			if message.terminal {
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
	case c.send <- notification{data: data, terminal: v.Error == "taken_over"}:
	default:
		c.cancel()
		_ = c.socket.Close()
	}
}

func (a *App) broadcast(state room) {
	for c := range a.clients {
		control := state.Controls[c.id]
		if control.PageID == c.pageID && control.Generation == c.generation && a.active[c.id] == c {
			v := state.visibleTo(c.id)
			v.ServerTime = a.clock.Now().UnixMilli()
			v.Error = a.fault
			a.queue(c, v)
		}
	}
}
