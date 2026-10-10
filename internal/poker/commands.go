// 本文件校验、去重并原子提交客户端命令；涉及流程：多条流程（共用）。
package poker

import (
	"crypto/sha256"
	"encoding/json"
	"math"
	"net/http"
	"time"
)

// command 是已解码的客户端请求；版本与控制代次用于拒绝过期操作。
type command struct {
	// RequestID 在同身份内标识幂等请求；相同标识但不同负载会被拒绝。
	RequestID string `json:"requestID"`
	// Version 必须匹配房间当前版本，成功命令才推进该值。
	Version int64  `json:"version"`
	Action  string `json:"action"`
	// HandID/TurnID 必须匹配当前局与行动机会，防止旧动作重放。
	HandID int64  `json:"handID,omitempty"`
	TurnID int64  `json:"turnID,omitempty"`
	PageID string `json:"pageID"`
	// Control 必须匹配当前页面控制代次。
	Control int64 `json:"control"`
}

type outcome struct {
	Status int
	View   view
}

type receipt struct {
	Hash   [32]byte
	Result outcome
}

// 【共用】幂等与原子命令事务。
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
		original := a.state.clone()        // 动作、扣款和期限共用回滚边界。
		rejection = a.gameCommand(id, cmd) // 执行公开游戏命令。
		if rejection == "" {               // 推进成功后才能抽样新机会期限。
			rejection = a.ensureDeadline() // 无效随机源不能留下半完成的动作。
		}
		if rejection == "" { // 状态和期限共同成功才确认版本。
			a.state.Version++ // 确认本次原子命令。
		} else { // 失败时恢复所有扣款和牌局。
			a.state = original // 原机会期限同时保留。
		}
	} else if seat < 0 {
		rejection = "room_full"
	} else if control.Generation == math.MaxInt64 {
		rejection = "control_exhausted"
	} else if control.PageID == cmd.PageID && control.Generation != cmd.Control {
		rejection = "taken_over"
	} else {
		// 【加入与离开房间#5b/12】改状态：加入分支占座并建立控制页代次。
		kind := "join"                 // 首次占用空位。
		if a.state.Seats[seat] == id { // 同身份成功重新入房。
			kind = "return" // 不重新播历史事件。
		}
		a.state.announce(announcement{Kind: kind, Seat: seat}) // 与已确认入房事务一起发布。
		// 每次成功的新入房请求（含重连）将可用余额设100，入房先后另行保留。
		joinedVersion := a.state.Version + 1 // 以串行确认版本记录本次占座先后，不受低号空位影响。
		if a.state.Seats[seat] == id {       // 同身份重连或接管仍是原来的在房成员。
			joinedVersion = a.state.Accounts[id].joinedVersion // 保留原入房先后，不能因设100而改动。
		}
		a.state.Accounts[id] = player{ID: id, Chips: 100, joinedVersion: joinedVersion} // 设100并保存已确定的本次占座顺序。
		a.state.Seats[seat] = id
		a.state.record(handRecordEntry{Kind: kind, ParticipantID: id, Seat: seat}) // 当前局公开过程恢复，不把新身份当旧参赛者。
		if kind == "return" {                                                      // 成功重新入房单独解释钱包重置。
			a.state.record(handRecordEntry{Kind: "reset", ParticipantID: id, Seat: seat, Amount: "100", Reason: "reconnect"}) // 旧历史快照不改写。
		}
		if a.state.Host == "" {
			a.state.Host = id
			a.state.announce(announcement{Kind: "host", Seat: seat}) // 第一名玩家也明确房主身份。
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
		a.commitAnnouncements() // 不为回滚或重复请求创建新事件。
		a.scheduleLocked()
	}
	result.View = a.visibleTo(id)
	if rejection == "" { // HTTP与WebSocket带相同事件身份，页面自行去重。
		result.View.Announcements = append([]announcement(nil), a.announcements...)
	}
	if rejection != "" {
		result.Status = http.StatusConflict
		result.View.Error = rejection
	}
	a.requests[key] = receipt{Hash: hash, Result: result}
	return result, rejection == ""
}
