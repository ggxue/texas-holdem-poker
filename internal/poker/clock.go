// 本文件提供可替换的权威时钟接口；涉及流程：机器人与超时。
package poker

import "time"

// Clock 是确定性时间边界，仅由应用配置，不能通过 HTTP 选择。
type Clock interface {
	Now() time.Time
	AfterFunc(time.Duration, func()) Timer
}

// Timer 只提供取消能力，保证调度器能使旧唤醒失效。
type Timer interface{ Stop() bool }
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

func (systemClock) AfterFunc(d time.Duration, f func()) Timer { return time.AfterFunc(d, f) }
