// 本文件初始化单房间应用并分发HTTP请求；涉及流程：多条流程（共用）。
package poker

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// App owns the single room and all its process-local state.
type App struct {
	mu              sync.Mutex
	state           room
	requests        map[string]receipt
	secret          []byte
	clients         map[*connection]struct{}
	closed          bool
	deck            func() ([]Card, error)
	actionStart     func([]int) (int, error)
	botThinkSeconds func() (int, error)
	active          map[string]*connection
	clock           Clock
	timer           Timer
	// wake 是当前定时器回调的唯一有效令牌，重调度后旧回调失效。
	wake          *int
	fault         string
	announcements []announcement
}

// Options controls cards, time and action-start randomness offline, never via HTTP.
type Options struct {
	Deck            func() ([]Card, error)
	Clock           Clock
	ActionStart     func([]int) (int, error)
	BotThinkSeconds func() (int, error)
}

func New() (*App, error) { return NewWithOptions(Options{}) }

func NewWithOptions(options Options) (*App, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	deck := options.Deck
	if deck == nil {
		deck = shuffledDeck
	}
	clock := options.Clock
	if clock == nil {
		clock = systemClock{}
	}
	actionStart := options.ActionStart
	if actionStart == nil {
		actionStart = randomActionStart
	}
	botThinkSeconds := options.BotThinkSeconds // 离线配置只替换思考时间抽样，不增加在线入口。
	if botThinkSeconds == nil {                // 正常运行使用一到八秒的无偏抽样。
		botThinkSeconds = randomBotThinkSeconds // 每个新机器人行动机会独立抽样。
	}
	return &App{state: room{Accounts: map[string]player{}, Bot: player{ID: "bot", Chips: 100}, Controls: map[string]controller{}, Disconnected: map[string]time.Time{}, Connecting: map[string]time.Time{}}, requests: map[string]receipt{}, secret: secret, clients: map[*connection]struct{}{}, active: map[string]*connection{}, deck: deck, clock: clock, actionStart: actionStart, botThinkSeconds: botThinkSeconds}, nil // 保存本进程的思考时间来源。
}

// 【开局#1/11】入口 app.go:ServeHTTP
// 链路：1. app.go:ServeHTTP → 2. game.go:validAction → 3. identity.go:identify → 4. commands.go:apply → 5. game.go:gameCommand（开局分支） → 6. game.go:validDeck → 7. game.go:randomActionStart → 8. game.go:advance → 9. deadlines.go:ensureDeadline → 10. announcements.go:commitAnnouncements → 11. socket.go:broadcast
// 下一步：#2 game.go:validAction
// 【玩家动作#1/12】入口 app.go:ServeHTTP
// 链路：1. app.go:ServeHTTP → 2. game.go:validAction → 3. identity.go:identify → 4. commands.go:apply → 5. state.go:clone → 6. game.go:gameCommand（行动者与局/回合校验块） → 7. game.go:act → 8. game.go:legal → 9. game.go:gameCommand（调用 advance 的代码块） → 10. deadlines.go:ensureDeadline → 11. announcements.go:commitAnnouncements → 12. socket.go:broadcast
// 下一步：#2 game.go:validAction
// 【断线重连与控制页接管#1/12】入口 app.go:ServeHTTP
// 链路：1. app.go:ServeHTTP → 2. identity.go:identify → 3. commands.go:apply（接管分支） → 4. announcements.go:commitAnnouncements（确认新控制页） → 5. socket.go:serveSocket（有效握手块） → 6. announcements.go:commitAnnouncements（确认连接恢复块） → 7. socket.go:broadcast → 8. socket.go:queue → 9. socket.go:connection.writeLoop → 10. socket.go:serveSocket（连接关闭清理块） → 11. deadlines.go:disconnectLocked → 12. socket.go:broadcast
// 下一步：#2 identity.go:identify
// 【视图构造与推送#1/8】入口 app.go:ServeHTTP
// 链路：1. app.go:ServeHTTP → 2. deadlines.go:visibleTo → 3. state.go:visibleTo → 4. game.go:hand.visibleTo → 5. hand_record.go:copyHandRecord → 6a. app.go:writeJSON / 6b. socket.go:broadcast → 7. socket.go:queue → 8. socket.go:connection.writeLoop
// 下一步：#2 deadlines.go:visibleTo
// 【加入与离开房间#1/12】入口 app.go:ServeHTTP
// 链路：1. app.go:ServeHTTP → 2. game.go:validAction → 3. identity.go:identify → 4. commands.go:apply → 5a. game.go:gameCommand（离开分支） / 5b. commands.go:apply（加入分支状态块） → 6a. game.go:depart（离开分支） / 6b. announcements.go:commitAnnouncements（加入分支） → 7. game.go:release → 8. game.go:advance → 9. deadlines.go:ensureDeadline → 10. announcements.go:commitAnnouncements → 11. deadlines.go:scheduleLocked → 12. socket.go:broadcast
// 下一步：#2 game.go:validAction
// 职责：HTTP入口与路由分发。
// 前置条件：HTTP路由已完成请求解码；只分发请求，不直接改牌局；输入、身份或服务故障按对应HTTP状态拒绝。
func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/ws" && r.Method == http.MethodGet {
		a.serveSocket(w, r)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Second))
	switch {
	case r.URL.Path == "/healthz" && r.Method == http.MethodGet:
		a.mu.Lock()
		ok := !a.closed && a.fault == ""
		a.mu.Unlock()
		if !ok {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	case r.URL.Path == "/api/state" && r.Method == http.MethodGet:
		id, credential, err := a.identify(r)
		if err != nil {
			writeJSON(w, 503, view{Error: "unavailable"})
			return
		}
		if credential != "" {
			setIdentity(w, r, credential)
		}
		a.mu.Lock()
		a.tickLocked()
		v := a.visibleTo(id)
		if a.closed {
			v.Error = "unavailable"
		}
		a.mu.Unlock()
		if v.Error != "" {
			writeJSON(w, 503, v)
			return
		}
		writeJSON(w, 200, v)
	case r.URL.Path == "/api/command" && r.Method == http.MethodPost:
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Host != r.Host || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
				writeJSON(w, 403, view{Error: "origin_denied"})
				return
			}
		}
		var cmd command
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&cmd); err != nil {
			writeJSON(w, 400, view{Error: "invalid_command"})
			return
		}
		if err := decoder.Decode(new(any)); err != io.EOF || !validAction(cmd.Action) || len(cmd.RequestID) == 0 || len(cmd.RequestID) > 128 || cmd.Version < 0 || len(cmd.PageID) == 0 || len(cmd.PageID) > 128 || cmd.Control < 0 {
			writeJSON(w, 400, view{Error: "invalid_command"})
			return
		}
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
		result, changed := a.apply(id, cmd)
		if changed {
			a.broadcast(a.state) // haifeng: 发送ws更新状态，等待玩家操作
		}
		a.mu.Unlock()
		writeJSON(w, result.Status, result.View)
	default:
		servePage(w, r)
	}
}

func (a *App) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closed = true
	if a.timer != nil {
		a.timer.Stop()
	}
	for c := range a.clients {
		c.cancel()
		_ = c.socket.Close()
	}
}

// 【视图构造与推送#6a/8】收尾 app.go:writeJSON
// 上一步：#5 hand_record.go:copyHandRecord；下一步：流程终点：返回HTTP结果或完成当前广播
// 职责：结束HTTP响应。
// 前置条件：调用方已决定状态码和视图；只写不可缓存JSON响应；不改变房间状态。
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
