package poker

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type store struct{ db *pgxpool.Pool }

type command struct {
	RequestID string `json:"requestID"`
	Version   int64  `json:"version"`
	Action    string `json:"action"`
}

type outcome struct {
	Status int  `json:"status"`
	View   view `json:"view"`
}

func (s store) initialize(ctx context.Context) ([]byte, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	// Serialize first-time initialization, including two simultaneous starts.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(817361900)"); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
CREATE TABLE IF NOT EXISTS poker_room (id integer PRIMARY KEY CHECK (id = 1), state jsonb NOT NULL);
CREATE TABLE IF NOT EXISTS poker_identity_key (id integer PRIMARY KEY CHECK (id = 1), secret bytea NOT NULL);
CREATE TABLE IF NOT EXISTS poker_requests (
 player_id text NOT NULL, request_id text NOT NULL, payload_hash bytea NOT NULL, result jsonb NOT NULL,
 PRIMARY KEY (player_id, request_id)
);`); err != nil {
		return nil, err
	}
	initial := room{Accounts: map[string]player{}, Bot: player{ID: "bot", Chips: 100}}
	data, err := json.Marshal(initial)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO poker_room VALUES (1, $1) ON CONFLICT DO NOTHING", data); err != nil {
		return nil, err
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO poker_identity_key VALUES (1, $1) ON CONFLICT DO NOTHING", secret); err != nil {
		return nil, err
	}
	if err := tx.QueryRow(ctx, "SELECT secret FROM poker_identity_key WHERE id = 1").Scan(&secret); err != nil {
		return nil, err
	}
	if len(secret) != 32 {
		return nil, errors.New("invalid saved identity key")
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return secret, nil
}

func decodeRoom(data []byte) (room, error) {
	var state room
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	if state.Version < 0 || state.Accounts == nil || state.Bot.ID != "bot" || state.Bot.Chips < 0 {
		return state, errors.New("invalid saved room")
	}
	return state, nil
}

func (s store) read(ctx context.Context) (room, error) {
	var data []byte
	if err := s.db.QueryRow(ctx, "SELECT state FROM poker_room WHERE id = 1").Scan(&data); err != nil {
		return room{}, err
	}
	return decodeRoom(data)
}

func (s store) apply(ctx context.Context, id string, cmd command) (outcome, *room, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return outcome{}, nil, err
	}
	defer tx.Rollback(context.Background())
	var data []byte
	if err := tx.QueryRow(ctx, "SELECT state FROM poker_room WHERE id = 1 FOR UPDATE").Scan(&data); err != nil {
		return outcome{}, nil, err
	}
	state, err := decodeRoom(data)
	if err != nil {
		return outcome{}, nil, err
	}
	payload, err := json.Marshal(cmd)
	if err != nil {
		return outcome{}, nil, err
	}
	hash := sha256.Sum256(payload)
	var savedHash, savedResult []byte
	err = tx.QueryRow(ctx, "SELECT payload_hash, result FROM poker_requests WHERE player_id = $1 AND request_id = $2", id, cmd.RequestID).Scan(&savedHash, &savedResult)
	if err == nil {
		if string(savedHash) != string(hash[:]) {
			v := state.visibleTo(id)
			v.Error = "request_conflict"
			return outcome{Status: http.StatusConflict, View: v}, nil, nil
		}
		var result outcome
		if err := json.Unmarshal(savedResult, &result); err != nil {
			return outcome{}, nil, err
		}
		return result, nil, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return outcome{}, nil, err
	}
	result := outcome{Status: http.StatusOK}
	var rejection string
	seat := -1
	for i, occupant := range state.Seats {
		if occupant == id {
			seat = i
			break
		}
	}
	if seat < 0 {
		for i, occupant := range state.Seats {
			if occupant == "" {
				seat = i
				break
			}
		}
	}
	if cmd.Version != state.Version {
		rejection = "stale_state"
	} else if state.Version == math.MaxInt64 {
		rejection = "version_exhausted"
	} else if seat < 0 {
		rejection = "room_full"
	} else {
		if _, exists := state.Accounts[id]; !exists {
			state.Accounts[id] = player{ID: id, Chips: 100}
		}
		state.Seats[seat] = id
		if state.Host == "" {
			state.Host = id
		}
		state.Version++
	}
	result.View = state.visibleTo(id)
	if rejection != "" {
		result.Status = http.StatusConflict
		result.View.Error = rejection
	}
	data, err = json.Marshal(state)
	if err != nil {
		return outcome{}, nil, err
	}
	if _, err := tx.Exec(ctx, "UPDATE poker_room SET state = $1 WHERE id = 1", data); err != nil {
		return outcome{}, nil, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return outcome{}, nil, err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO poker_requests VALUES ($1, $2, $3, $4)", id, cmd.RequestID, hash[:], encoded); err != nil {
		return outcome{}, nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return outcome{}, nil, fmt.Errorf("save room: %w", err)
	}
	if rejection != "" {
		return result, nil, nil
	}
	return result, &state, nil
}
