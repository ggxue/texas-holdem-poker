// 本文件定义房间权威状态与身份视图；涉及流程：玩家动作、机器人与超时、视图构造与推送。
package poker

import (
	"strconv"
	"time"
)

const humanSeatCount = 5

type player struct {
	ID                string `json:"id"`
	Chips             int64  `json:"chips"`
	DisconnectedUntil int64  `json:"disconnectedUntil,omitempty"`
	ConnectingUntil   int64  `json:"connectingUntil,omitempty"`
	joinedVersion     int64
}

type room struct {
	// Version 是串行确认的房间版本；成功命令和自动状态变更递增。
	Version  int64                  `json:"version"`
	Accounts map[string]player      `json:"accounts"`
	Seats    [humanSeatCount]string `json:"seats"`
	Host     string                 `json:"host"`
	Bot      player                 `json:"bot"`
	Hand     *hand
	Controls map[string]controller
	// Disconnected 是已观察断线宽限，Connecting 是接管后等待有效WebSocket的期限。
	Disconnected  map[string]time.Time
	Connecting    map[string]time.Time
	announcements []announcement
}

type controller struct {
	PageID string
	// Generation 随成功接管递增，旧页面与旧连接不得再控制或广播。
	Generation int64
	// Seen 记录曾用过的页面标识，旧页面不能借 join 请求夺回控制权。
	Seen map[string]bool
}

type view struct {
	Version       int64                   `json:"version"`
	You           string                  `json:"you"`
	Host          string                  `json:"host"`
	Seats         [humanSeatCount]*player `json:"seats"`
	Bot           player                  `json:"bot"`
	Error         string                  `json:"error,omitempty"`
	Hand          *handView               `json:"hand,omitempty"`
	Control       int64                   `json:"control"`
	ServerTime    int64                   `json:"serverTime"`
	Announcements []announcement          `json:"announcements,omitempty"`
}

// 【视图构造与推送#3/8】构造视图 state.go:visibleTo
// 上一步：#2 deadlines.go:visibleTo；下一步：#4 game.go:hand.visibleTo
// 职责：裁剪房间状态视图。
// 前置条件：身份已验证；构造按身份裁剪的房间快照；不改变房间数据。
func (s room) visibleTo(id string) view {
	v := view{Version: s.Version, You: id, Host: s.Host, Bot: s.Bot}
	v.Control = s.Controls[id].Generation
	if s.Hand != nil {
		v.Hand = s.Hand.visibleTo(id)
		if v.Hand.Actor == id { // 只有当前行动者需要实际扣款提示。
			p := s.Hand.Players[s.Hand.Actor]                                                     // 读取本人本轮投入。
			v.Hand.Legal = s.Hand.legal(s.balance(id))                                            // 只有本人当前机会收到合法按钮。
			v.Hand.CallRequired = strconv.FormatInt(s.Hand.Target-p.Street, 10)                   // 欠额与实际扣款分开，不由JS推算。
			v.Hand.CallAmount = strconv.FormatInt(min(s.Hand.Target-p.Street, s.balance(id)), 10) // 大额跟注也精确显示。
			v.Hand.AllInAmount = strconv.FormatInt(s.balance(id), 10)                             // 全押实际金额采用精确十进制文字。
			v.Hand.BetAmount = min(int64(10), s.balance(id))                                      // 固定十枚下注不足时显示全押金额。
		}
	}
	for i, occupant := range s.Seats {
		if occupant != "" {
			p := s.Accounts[occupant]
			if deadline := s.Disconnected[occupant]; !deadline.IsZero() {
				p.DisconnectedUntil = deadline.UnixMilli()
			}
			if deadline := s.Connecting[occupant]; !deadline.IsZero() {
				p.ConnectingUntil = deadline.UnixMilli()
			}
			v.Seats[i] = &p
		}
	}
	return v
}

// 【玩家动作#5/12】回滚边界 state.go:clone
// 上一步：#4 commands.go:apply；下一步：#6 game.go:gameCommand
// 【机器人与超时#5/11】回滚边界 state.go:clone
// 上一步：#4 deadlines.go:tickLocked；下一步：#6 game.go:act
// 职责：原子回滚快照。
// 前置条件：调用方持有 App.mu；复制可变房间状态作为事务快照；无业务拒绝出口。
// clone creates the rollback boundary for one atomic in-memory command.
func (s room) clone() room {
	c := s
	c.announcements = append([]announcement(nil), s.announcements...) // 事件草稿与扣款共用回滚边界。
	c.Accounts = make(map[string]player, len(s.Accounts))
	c.Disconnected = make(map[string]time.Time, len(s.Disconnected))
	c.Connecting = make(map[string]time.Time, len(s.Connecting))
	for id, deadline := range s.Connecting {
		c.Connecting[id] = deadline
	}
	for id, deadline := range s.Disconnected {
		c.Disconnected[id] = deadline
	}
	for id, p := range s.Accounts {
		c.Accounts[id] = p
	}
	if s.Hand != nil {
		h := *s.Hand
		h.Players = append([]participant(nil), s.Hand.Players...)
		h.Board = append([]Card(nil), s.Hand.Board...)
		h.records = copyHandRecord(s.Hand.records) // 草稿、提交标识和历史公共牌共同回滚。
		c.Hand = &h
	}
	return c
}
