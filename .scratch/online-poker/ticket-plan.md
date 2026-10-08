# 在线德州扑克：当前任务索引

Status: published
Type: ticket-plan
Spec: [当前规格](spec.md)
Approval: 原12票已批准；2026-10-08按用户明确覆盖决定修订为9张有效实施票。

## 当前决定与执行

[内存与重连规则](memory-reset-decision.md)覆盖Q26/Q27和旧AC25。
02改为内存入房；03–08状态只在单进程内，成功重新入房设100；
09–11撤销；12只部署免费WebSocket运行服务，不创建数据库。
历史编号保留，撤销票不是完成票，也不阻塞后续上线。
每票按TDD→双轴审查→完整检查→记录证据→提交推进。
用户已授权连续实施全部有效票据；02仅保留AC07/新版AC25测试，后续票按各自已确认AC实施。
自动化通过且实现可用时继续依赖票；实际浏览器/race/云验证缺工具仍记录待执行，不冒充done。

## 有效任务

| 任务 | 直接依赖 | 主要案例 |
| --- | --- | --- |
| [01：恢复Go基线](issues/01-restore-go-baseline.md) | 无；已done | Harness基线 |
| [02：匿名入房与内存筹码](issues/02-persistent-room-entry.md) | 01 | AC07、AC25 |
| [03：开局过牌到摊牌](issues/03-check-to-showdown.md) | 02 | AC01–AC06、AC08、AC13、AC22 |
| [04：四动作、全押与单池结算](issues/04-bet-allin-single-pot.md) | 03 | AC09、AC14–AC23、AC32 |
| [05：跨局累计与归零补给](issues/05-balance-refill.md) | 04 | AC24、AC17 |
| [06：中途入房、退出与房主交接](issues/06-membership-host-transfer.md) | 04 | AC10–AC12、AC31 |
| [07：同身份页面接管](issues/07-page-takeover.md) | 04 | AC26、AC32 |
| [08：断线宽限与行动期限](issues/08-disconnect-action-timers.md) | 06、07 | AC27–AC30、AC38 |
| [12：免费云运行与整体验收](issues/12-free-cloud-launch.md) | 05、08 | AC41–AC43 |

## 已撤销的历史票据

- [09：数据库失败与提交不明](issues/09-storage-failure-commit-resolution.md)：wontfix。
- [10：重启恢复存档](issues/10-restart-recovery.md)：wontfix。
- [11：跨进程存档所有权](issues/11-process-ownership.md)：wontfix。

02内存修订已done；03实现及自动化检查通过，浏览器验收待12；当前frontier是04。其余状态以票据为准。
云账号、部署权限及当前官方免费限制在12核实，历史研究不等于上线验证。
