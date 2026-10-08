package poker

import (
	"crypto/sha256"
	"encoding/json"
	"math"
	"net/http"
	"time"
)

type command struct {
	RequestID string `json:"requestID"`
	Version   int64  `json:"version"`
	Action    string `json:"action"`
	HandID    int64  `json:"handID,omitempty"`
	TurnID    int64  `json:"turnID,omitempty"`
	PageID    string `json:"pageID"`
	Control   int64  `json:"control"`
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
	control := a.state.Controls[id]
	if (control.Seen[cmd.PageID] && control.PageID != cmd.PageID) || (cmd.Action != "join" && (control.PageID != cmd.PageID || control.Generation != cmd.Control)) {
		v := a.visibleTo(id)
		v.Error = "taken_over"
		return outcome{http.StatusConflict, v}, false
	}
	payload, _ := json.Marshal(cmd)
	hash := sha256.Sum256(payload)
	key := id + ":" + cmd.RequestID
	if saved, ok := a.requests[key]; ok {
		if saved.Hash != hash {
			v := a.visibleTo(id)
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
	} else if cmd.Action != "join" {
		original := a.state.clone()
		rejection = a.gameCommand(id, cmd)
		if rejection == "" {
			a.state.Version++
		} else {
			a.state = original
		}
	} else if seat < 0 {
		rejection = "room_full"
	} else if control.Generation == math.MaxInt64 {
		rejection = "control_exhausted"
	} else if control.PageID == cmd.PageID && control.Generation != cmd.Control {
		rejection = "taken_over"
	} else {
		// 每次成功的新入房请求（含重连）将可用余额设100，入房先后另行保留。
		joinedVersion := a.state.Version + 1 // 以串行确认版本记录本次占座先后，不受低号空位影响。
		if a.state.Seats[seat] == id {       // 同身份重连或接管仍是原来的在房成员。
			joinedVersion = a.state.Accounts[id].joinedVersion // 保留原入房先后，不能因设100而改动。
		}
		a.state.Accounts[id] = player{ID: id, Chips: 100, joinedVersion: joinedVersion} // 设100并保存已确定的本次占座顺序。
		a.state.Seats[seat] = id
		if a.state.Host == "" {
			a.state.Host = id
		}
		a.state.Version++
		if control.Seen == nil {
			control.Seen = map[string]bool{}
		}
		control.Seen[cmd.PageID] = true
		control.PageID = cmd.PageID
		control.Generation++
		a.state.Controls[id] = control
		deadline := a.state.Disconnected[id]                                                                         // 重连握手前仍沿用原离房期限。
		if pending := a.state.Connecting[id]; !pending.IsZero() && (deadline.IsZero() || pending.Before(deadline)) { // 已在握手阶段的重试也不能延长期限。
			deadline = pending // 保留两种现有期限中更早的一项。
		}
		if deadline.IsZero() { // 新页没有普通断线期限时建立握手期限。
			deadline = a.clock.Now().Add(30 * time.Second) // 不能让未建立连接的页面永久占座。
		}
		a.state.Connecting[id] = deadline // 等有效WebSocket建立后才解除等待。
		delete(a.state.Disconnected, id)  // 接管本身不制造普通断线状态。
		for c := range a.clients {
			if c.id == id && c.generation != control.Generation {
				a.queue(c, view{Error: "taken_over"})
			}
		}
		delete(a.active, id)
	}
	if rejection == "" {
		a.ensureDeadline()
		a.scheduleLocked()
	}
	result.View = a.visibleTo(id)
	if rejection != "" {
		result.Status = http.StatusConflict
		result.View.Error = rejection
	}
	a.requests[key] = receipt{Hash: hash, Result: result}
	return result, rejection == ""
}
