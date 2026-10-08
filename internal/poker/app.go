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
	mu       sync.Mutex
	state    room
	requests map[string]receipt
	secret   []byte
	clients  map[*connection]struct{}
	closed   bool
	deck     func() ([]Card, error)
}

// Options supplies an offline deterministic deck seam; HTTP never accepts cards.
type Options struct{ Deck func() ([]Card, error) }

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
	return &App{state: room{Accounts: map[string]player{}, Bot: player{ID: "bot", Chips: 100}}, requests: map[string]receipt{}, secret: secret, clients: map[*connection]struct{}{}, deck: deck}, nil
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/ws" && r.Method == http.MethodGet {
		a.serveSocket(w, r)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(5 * time.Second))
	switch {
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
		v := a.state.visibleTo(id)
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
		if err := decoder.Decode(new(any)); err != io.EOF || !validAction(cmd.Action) || len(cmd.RequestID) == 0 || len(cmd.RequestID) > 128 || cmd.Version < 0 {
			writeJSON(w, 400, view{Error: "invalid_command"})
			return
		}
		id, credential, err := a.identify(r)
		if err != nil || credential != "" {
			writeJSON(w, 401, view{Error: "identity_required"})
			return
		}
		a.mu.Lock()
		if a.closed {
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
