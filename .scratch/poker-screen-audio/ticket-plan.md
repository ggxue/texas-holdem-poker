# B2生产版三票

Status: published
Type: ticket-plan
Spec: [已确认规格](spec.md)
Approval: 用户已确认三票、公开页面/HTTP/WebSocket测试范围，授权连续开发、推送所提供GitHub仓库并推进上线。
Baseline: `3eec8843fec6d6e69d05f07e54e7b6977408e532`

1. [03 B2真实牌桌与筹码](issues/03-b2-production-screen.md) — 无阻塞。
2. [04 确认事件与中文播报](issues/04-announcements-and-voice.md) — 阻塞03。
3. [05 整合验收与免费上线](issues/05-release.md) — 阻塞03/04。

逐票red→green、Standards/Spec审查、验证及 scoped local commit。GitHub是源码仓库，
Render Web Service运行Go及HTTPS/WSS；不创建Postgres/付费资源/额外保活。
