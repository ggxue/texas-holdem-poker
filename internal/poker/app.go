package poker

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// App serves the single room. The caller owns the database pool.
type App struct {
	mu      sync.Mutex
	store   store
	secret  []byte
	clients map[*connection]struct{}
	closed  bool
}

func New(ctx context.Context, db *pgxpool.Pool) (*App, error) {
	s := store{db: db}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	secret, err := s.initialize(ctx)
	if err != nil {
		return nil, err
	}
	return &App{store: s, secret: secret, clients: map[*connection]struct{}{}}, nil
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/api/ws" && r.Method == http.MethodGet {
		a.serveSocket(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	switch {
	case r.URL.Path == "/api/state" && r.Method == http.MethodGet:
		state, err := a.store.read(ctx)
		if err != nil {
			writeJSON(w, 503, view{Error: "restoring"})
			return
		}
		id, credential, err := a.identify(r)
		if err != nil {
			writeJSON(w, 503, view{Error: "restoring"})
			return
		}
		if credential != "" {
			setIdentity(w, r, credential)
		}
		writeJSON(w, 200, state.visibleTo(id))
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
		if err := decoder.Decode(new(any)); err != io.EOF || cmd.Action != "join" || len(cmd.RequestID) == 0 || len(cmd.RequestID) > 128 || cmd.Version < 0 {
			writeJSON(w, 400, view{Error: "invalid_command"})
			return
		}
		id, credential, err := a.identify(r)
		if err != nil || credential != "" {
			writeJSON(w, 401, view{Error: "identity_required"})
			return
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.closed {
			writeJSON(w, 503, view{Error: "restoring"})
			return
		}
		result, state, err := a.store.apply(ctx, id, cmd)
		if err != nil {
			writeJSON(w, 503, view{Error: "restoring"})
			return
		}
		writeJSON(w, result.Status, result.View)
		if state != nil {
			a.broadcast(*state)
		}
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
