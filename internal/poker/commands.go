package poker

import (
	"crypto/sha256"
	"encoding/json"
	"math"
	"net/http"
)

type command struct {
	RequestID string `json:"requestID"`
	Version   int64  `json:"version"`
	Action    string `json:"action"`
}

type outcome struct {
	Status int
	View   view
}

type receipt struct {
	Hash   [32]byte
	Result outcome
}

// Caller holds App.mu; state and retry results live only in this process.
func (a *App) apply(id string, cmd command) (outcome, bool) {
	payload, _ := json.Marshal(cmd)
	hash := sha256.Sum256(payload)
	key := id + ":" + cmd.RequestID
	if saved, ok := a.requests[key]; ok {
		if saved.Hash != hash {
			v := a.state.visibleTo(id)
			v.Error = "request_conflict"
			return outcome{http.StatusConflict, v}, false
		}
		return saved.Result, false
	}
	result := outcome{Status: http.StatusOK}
	rejection := ""
	seat := -1
	for i, occupant := range a.state.Seats {
		if occupant == id {
			seat = i
			break
		}
	}
	if seat < 0 {
		for i, occupant := range a.state.Seats {
			if occupant == "" {
				seat = i
				break
			}
		}
	}
	if cmd.Version != a.state.Version {
		rejection = "stale_state"
	} else if a.state.Version == math.MaxInt64 {
		rejection = "version_exhausted"
	} else if seat < 0 {
		rejection = "room_full"
	} else {
		// A fresh successful entry resets the available balance, including reconnects.
		a.state.Accounts[id] = player{ID: id, Chips: 100}
		a.state.Seats[seat] = id
		if a.state.Host == "" {
			a.state.Host = id
		}
		a.state.Version++
	}
	result.View = a.state.visibleTo(id)
	if rejection != "" {
		result.Status = http.StatusConflict
		result.View.Error = rejection
	}
	a.requests[key] = receipt{Hash: hash, Result: result}
	return result, rejection == ""
}
