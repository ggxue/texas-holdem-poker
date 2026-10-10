# 手机紧凑六席：实施顺序

Status: ready-for-agent
Spec: [已确认规格](spec.md)
Implementation baseline: `9b0bba68d0fbbfeb30adbc12ce5dfe15b6b870cf`

用户授权自行执行 to-tickets、implement 到完成，覆盖分票确认步骤。本轮是一个完整的手机
呈现改动，使用一张实施票，避免把样式、交互与验收拆成无法独立演示的水平切片。

| 顺序 | 票据 | Blocked by | 交付 |
| --- | --- | --- | --- |
| 02 | [手机紧凑六席与常驻行动](issues/02-compact-mobile-table.md) | None；原型01已resolved，规格已确认 | 真实牌局的A布局、固定行动区、完整详情与原功能入口，以及手机/桌面公共验收 |

当前不需要预重构；后端规则与传输契约沿用。架构报告候选不纳入本票。
