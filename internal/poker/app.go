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
	wake            *int
	fault           string
	announcements   []announcement
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
			a.broadcast(a.state)
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

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
