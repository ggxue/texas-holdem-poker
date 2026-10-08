package poker

import (
	"math"
	"time"
)

// All methods below run under App.mu. Only one cancellable wake owns timer work.
func (a *App) visibleTo(id string) view {
	v := a.state.visibleTo(id)               // 先裁剪该身份有权查看的状态。
	v.ServerTime = a.clock.Now().UnixMilli() // 给页面换算服务器倒计时。
	if a.fault != "" {                       // 自动事件故障不能冒充正常状态。
		v.Error = a.fault // 向公开边界报告故障。
	}
	return v // 快照不会携带私有牌序。
}

func (a *App) ensureDeadline() {
	h := a.state.Hand // 行动期限属于本局的具体机会。
	if h == nil {     // 未开局没有行动计时。
		return // 不创建额外期限。
	}
	if h.Stage == "finished" || h.Actor < 0 { // 已结束没有合法行动机会。
		h.Deadline = time.Time{} // 清空行动倒计时。
		return                   // 断线宽限由独立期限管理。
	}
	if h.deadlineTurn != h.Turn || h.Deadline.IsZero() { // 只有新的行动机会才给30秒。
		h.Deadline = a.clock.Now().Add(30 * time.Second) // 新期限从实际推进时起算。
		h.deadlineTurn = h.Turn                          // 将期限绑定当前行动标识。
	}
}

func (a *App) nextDeadline() time.Time {
	var next time.Time                                        // 暂无待处理期限。
	if h := a.state.Hand; h != nil && h.Stage != "finished" { // 只有未结束牌局存在行动期限。
		next = h.Deadline // 先考虑行动超时。
	}
	for _, deadline := range a.state.Disconnected { // 再比较所有已观察断线宽限。
		if next.IsZero() || deadline.Before(next) { // 选择最早的绝对时间。
			next = deadline // 保存最早期限。
		}
	}
	for _, deadline := range a.state.Connecting { // 入房成功但控制连接尚未建立也必须限时。
		if next.IsZero() || deadline.Before(next) { // 选择更早的连接建立期限。
			next = deadline // 握手失败不能永久占据房间。
		}
	}
	return next // 返回零值表示不需要定时器。
}

func (a *App) scheduleLocked() {
	if a.timer != nil { // 已有唤醒可以被新状态取消。
		a.timer.Stop() // 取消旧定时任务。
	}
	token := new(int)              // 使用不同指针识别本次调度，避免旧回调生效。
	a.wake = token                 // 保存当前有效唤醒标识。
	if a.closed || a.fault != "" { // 关闭或故障时不继续自动变更。
		return // 不留下后台忙循环。
	}
	deadline := a.nextDeadline() // 找到当前最早期限。
	if deadline.IsZero() {       // 无期限无需后台定时器。
		return // 无人牌桌不会自行开局。
	}
	delay := deadline.Sub(a.clock.Now()) // 依据绝对时间计算等待。
	if delay < 0 {                       // 已过期则立即处理。
		delay = 0 // 不把负等待解释为新30秒。
	}
	a.timer = a.clock.AfterFunc(delay, func() { // 一个回调负责所有到期事件。
		a.mu.Lock()                      // 自动事件与HTTP命令共用状态锁。
		defer a.mu.Unlock()              // 自动处理完成后释放锁。
		if a.closed || a.wake != token { // 取消或替换后的旧回调无效。
			return // 不覆盖新的控制状态。
		}
		a.tickLocked()     // 按绝对期限处理当前已到期事件。
		a.scheduleLocked() // 安排下一次最早唤醒。
	})
}

func (a *App) tickLocked() {
	if a.closed || a.fault != "" { // 不处理已经关闭或故障的应用。
		return // 保留故障前权威状态。
	}
	changed := false     // 仅变更后广播。
	now := a.clock.Now() // 本批次实际处理时间。
	for {                // 顺序处理所有已创建且到期的事件。
		at := a.nextDeadline()            // 获取最早绝对期限。
		if at.IsZero() || at.After(now) { // 尚无到期事件。
			break // 新机会不会回填停顿期间不存在的回合。
		}
		if a.state.Version == math.MaxInt64 { // 状态版本也不能溢出。
			a.fault = "version_exhausted" // 报告无法继续推进。
			break                         // 停止自动变更。
		}
		original := a.state.clone()        // 自动事件同样保留原子回滚边界。
		departed := false                  // 同刻离房优先于行动超时。
		rejection := ""                    // 保存本批事件的错误。
		for _, id := range a.state.Seats { // 同刻多个离房按固定座位批量移除。
			if id != "" && (a.state.Disconnected[id].Equal(at) || a.state.Connecting[id].Equal(at)) { // 同刻断线与建立连接失败都先确认离房。
				rejection = a.state.release(id) // 先撤销所有同刻离房资格，不中途派奖。
				departed = true                 // 标记本刻已发生离房。
				if rejection != "" {            // 遇到状态错误不能继续。
					break // 留给统一回滚。
				}
			}
		}
		if rejection == "" && departed { // 所有同刻离房完成后统一推进。
			if a.state.Hand != nil && a.state.Hand.Stage != "finished" { // 已结算奖项不撤销。
				rejection = a.state.advance() // 跳过离房玩家，必要时结算。
			}
		} else if rejection == "" { // 没有同刻离房则处理行动超时。
			h := a.state.Hand                         // 最早期限属于当前行动机会。
			action := "check"                         // 不欠注时自动过牌。
			if h.Players[h.Actor].Street < h.Target { // 欠注时不能自动跟注。
				action = "fold" // 自动弃牌而不是替玩家付钱。
			}
			rejection = a.state.act(action) // 用同一合法动作规则处理超时。
			if rejection == "" {            // 动作成功后才推进。
				rejection = a.state.advance() // 机器人和下一阶段立即执行。
			}
		}
		if rejection != "" { // 出错时不保留部分扣款或资格变更。
			a.state = original  // 回滚整个自动事件。
			a.fault = rejection // 明确报告故障并停止继续自动调度。
			break               // 避免对已过期失败事件不断重试。
		}
		a.ensureDeadline() // 新机会从本次真实处理时间开始30秒。
		a.state.Version++  // 自动状态变更也有权威版本。
		changed = true     // 本批需要通知客户端。
	}
	if changed || a.fault != "" { // 只发送已共同更新的状态或明确故障。
		a.broadcast(a.state) // 每名收件人仍按身份裁剪。
		a.scheduleLocked()   // 使旧定时回调失效。
	}
}

func (a *App) disconnectLocked(id, pageID string, generation int64) {
	a.tickLocked()                                                                                 // 先按已创建的绝对期限推进。
	control := a.state.Controls[id]                                                                // 核对断线是否属于当前控制页。
	if a.closed || a.fault != "" || control.Generation != generation || control.PageID != pageID { // 旧控制页关闭不得创建新宽限。
		return // 忽略关闭或过期的连接事件。
	}
	seated := false                          // 确认身份仍在房间。
	for _, occupant := range a.state.Seats { // 不给已经离房者留下孤立期限。
		if occupant == id { // 找到当前座位。
			seated = true // 该身份仍有断线保留资格。
		}
	}
	if !seated { // 已确认离房不能重新占座。
		return // 忽略迟到的连接关闭。
	}
	if a.state.Version == math.MaxInt64 { // 版本计数同样必须保持精确。
		a.fault = "version_exhausted" // 报告无法继续更新。
		a.broadcast(a.state)          // 向客户端发送明确故障。
		a.scheduleLocked()            // 取消后续自动调度。
		return                        // 不允许版本回绕。
	}
	a.state.Disconnected[id] = a.clock.Now().Add(30 * time.Second) // 已观察断线创建独立30秒宽限。
	a.state.Version++                                              // 断线标记也是确认的权威状态。
	a.broadcast(a.state)                                           // 公开断线状态，不改变手牌权限。
	a.scheduleLocked()                                             // 同时考虑原行动期限与新宽限。
}
