package poker

import "strconv"

// announcement只携带确认的公开事实；金额用十进制字符串避免JS精度损失。
type announcement struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Seat   int    `json:"seat"`
	Action string `json:"action,omitempty"`
	Amount string `json:"amount,omitempty"`
	AllIn  bool   `json:"allIn,omitempty"`
	Reason string `json:"reason,omitempty"`
	At     int64  `json:"at"`
	HandID int64  `json:"handID,omitempty"`
	TurnID int64  `json:"turnID,omitempty"`
}

func (s *room) announce(e announcement) {
	if s.Hand != nil { // 标识生成事件时的局与机会，而非最终快照的机会。
		e.HandID, e.TurnID = s.Hand.ID, s.Hand.Turn // 不添加手牌或未来随机结果。
	}
	s.announcements = append(s.announcements, e) // 草稿与状态共同回滚。
}

func (a *App) commitAnnouncements() {
	a.commitHandRecord()                      // 文字记录与瞬时公告在同一次成功提交中确认。
	for i, e := range a.state.announcements { // 只有成功提交后才赋予事件身份。
		e.ID = strconv.FormatInt(a.state.Version, 10) + ":" + strconv.Itoa(i) // 同一进程内的版本与序号唯一。
		e.At = a.clock.Now().UnixMilli()                                      // 使用同一权威时钟。
		a.announcements = append(a.announcements, e)                          // 支持同批自动变化按提交顺序广播。
	}
	a.state.announcements = nil // 状态查询不携带事件历史。
}
