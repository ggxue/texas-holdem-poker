// 本文件维护控制页WebSocket并广播确认视图；涉及流程：断线重连与控制页接管、视图构造与推送。
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
	id     string
	socket *websocket.Conn
	send   chan notification
	ctx    context.Context
	cancel context.CancelFunc
	pageID string
	// generation 绑定建立连接时的控制代次，过期连接不能继续收发房间状态。
	generation int64
}

type notification struct {
	data     []byte
	terminal bool
}

// 【断线重连与控制页接管#5/12】校验 socket.go:serveSocket
// 上一步：#4 announcements.go:commitAnnouncements；下一步：#6 announcements.go:commitAnnouncements
// 职责：验证控制页并维护连接生命周期。
// 前置条件：身份、座位、页号与控制代次必须匹配；注册当前连接并可能解除连接宽限；握手不符返回401/403/503。
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
		if !a.state.Disconnected[id].IsZero() { // 同控制页直接恢复WS，没有重新入房命令。
			for seat, occupant := range a.state.Seats { // 仅向原在线收件人报告回来。
				if occupant == id { // 不给恢复页补播事件。
					a.state.announce(announcement{Kind: "return", Seat: seat})                     // 不改余额或行动期限。
					a.state.record(handRecordEntry{Kind: "return", ParticipantID: id, Seat: seat}) // 同控制页直接恢复不设100，不补播旧声音。
				}
			}
		}
		delete(a.state.Connecting, id)   // 清除握手阶段期限。
		delete(a.state.Disconnected, id) // 清除已观察断线的期限。
		a.state.Version++                // 连接已建立也是权威状态更新。
		a.commitAnnouncements()          // 与成功控制连接共同确认。
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
	// 【断线重连与控制页接管#10/12】连接关闭清理块：仅当前连接可创建断线宽限。
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

// 【断线重连与控制页接管#9/12】收尾 socket.go:connection.writeLoop
// 上一步：#8 socket.go:queue；下一步：#10 socket.go:serveSocket
// 【视图构造与推送#8/8】收尾 socket.go:connection.writeLoop
// 上一步：#7 socket.go:queue；下一步：流程终点：WebSocket视图已写出
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

// 【断线重连与控制页接管#8/12】广播 socket.go:queue
// 上一步：#7 socket.go:broadcast；下一步：#9 socket.go:connection.writeLoop
// 【视图构造与推送#7/8】广播 socket.go:queue
// 上一步：#6b socket.go:broadcast；下一步：#8 socket.go:connection.writeLoop
// 职责：隔离慢客户端与房间锁。
// 前置条件：调用方持有 App.mu；向单连接队列非阻塞入队；编码失败或队列满时关闭该连接。
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

// 【共用】广播身份裁剪状态。
func (a *App) broadcast(state room) {
	defer func() { a.announcements = nil }() // 广播后丢弃瞬时事件；新连接不补历史。
	for c := range a.clients {
		control := state.Controls[c.id]
		if control.PageID == c.pageID && control.Generation == c.generation && a.active[c.id] == c {
			v := state.visibleTo(c.id)
			v.ServerTime = a.clock.Now().UnixMilli()
			v.Error = a.fault
			v.Announcements = a.announcements // 同一次提交的所有在线收件人共享公开事件。
			a.queue(c, v)
		}
	}
}
