// 本文件记录并快照单局公开事实；涉及流程：多条流程（共用）。
package poker

import "strconv"

// handRecordEntry是公开事实快照，不携带私人手牌或未来牌序。
type handRecordEntry struct {
	// ID 在事务确认时由版本与局内序号生成；Seq 仅在本局内递增。
	ID     string `json:"id"`
	HandID int64  `json:"handID"`
	Seq    int    `json:"seq"`
	// At 是事实确认时的服务端时间，后续钱包变化不重写已存快照。
	At            int64  `json:"at"`
	Stage         string `json:"stage"`
	Kind          string `json:"kind"`
	ParticipantID string `json:"participantID,omitempty"`
	Seat          int    `json:"seat"`
	Action        string `json:"action,omitempty"`
	Amount        string `json:"amount,omitempty"`
	Pot           string `json:"pot"`
	Balance       string `json:"balance,omitempty"`
	AllIn         bool   `json:"allIn,omitempty"`
	Reason        string `json:"reason,omitempty"`
	Board         []Card `json:"board,omitempty"`
}

// 【阶段推进与结算#9/11】改状态 hand_record.go:record
// 上一步：#8 cards.go:Strength.Compare；下一步：#10 announcements.go:commitAnnouncements
// 职责：暂存公开局内事实。
// 前置条件：记录属于当前内存牌局事务；追加公开快照；无当前牌局时不建立跨局记录。
func (s *room) record(e handRecordEntry) {
	h := s.Hand   // 当前局是记录唯一的内存归属。
	if h == nil { // 无牌局不创建跨局历史。
		return // 房间事件仍由原公告处理。
	}
	e.HandID, e.Seq = h.ID, len(h.records)+1 // 局内顺序不由页面推测。
	if e.Stage == "" {                       // 普通动作使用发生时阶段。
		e.Stage = h.Stage // 开局和结算可显式指定章节。
	}
	if e.Pot == "" { // 默认保存变更后的底池。
		e.Pot = strconv.FormatInt(h.Pot, 10) // 派奖可提供分配后的精确快照。
	}
	if e.ParticipantID != "" { // 余额跟随事件中的原身份，不跟随当前席位。
		e.Balance = strconv.FormatInt(s.balance(e.ParticipantID), 10) // 后续设100不改写本条。
	}
	e.Board = append([]Card(nil), h.Board...) // 仅复制已经公开的公共牌。
	h.records = append(h.records, e)          // 草稿与扣款共用回滚边界。
}

// 【机器人与超时#9/11】推进 hand_record.go:commitHandRecord
// 上一步：#8 deadlines.go:ensureDeadline；下一步：#10 announcements.go:commitAnnouncements
// 职责：确认本次局内记录。
// 前置条件：状态事务已成功确认；仅为新增公开记录赋予稳定ID与权威时间；不会改写已提交快照。
func (a *App) commitHandRecord() {
	h := a.state.Hand // 只有确认事务成功后才赋予标识和时间。
	if h == nil {     // 未开局没有待提交记录。
		return // 保持无跨局历史。
	}
	for i := h.committedRecords; i < len(h.records); i++ { // 仅处理这次事务的新事实。
		e := &h.records[i]                                                           // 独立切片不会与回滚快照共用写入空间。
		e.ID = strconv.FormatInt(a.state.Version, 10) + ":record:" + strconv.Itoa(i) // HTTP与WS共用身份。
		e.At = a.clock.Now().UnixMilli()                                             // 使用同一服务端权威时钟。
	}
	h.committedRecords = len(h.records) // 查询与重试不再赋予新身份。
}

// 【视图构造与推送#5/8】构造视图 hand_record.go:copyHandRecord
// 上一步：#4 game.go:hand.visibleTo；下一步：#6a app.go:writeJSON
// 职责：隔离记录快照切片。
// 前置条件：输入记录属于当前只读快照；深拷贝记录及公共牌切片；不改写源数据。
func copyHandRecord(records []handRecordEntry) []handRecordEntry {
	copy := append([]handRecordEntry(nil), records...) // 事务与输出都取得独立切片。
	for i := range copy {                              // 公共牌切片也不能被未来变更覆盖。
		copy[i].Board = append([]Card(nil), records[i].Board...) // 没有暗牌可复制。
	}
	return copy // 返回不可被之后状态写入的快照。
}
