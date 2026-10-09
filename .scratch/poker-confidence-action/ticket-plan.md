# 主动全押实施三票

Status: published
Type: ticket-plan
Spec: [已确认规格](spec.md)
Approval: 用户审过prototype，并授权全自动to-spec、to-tickets、implement；无需重复访谈粒度与既有测试边界。
Baseline: `0d71575da9400ff48688d7152198d321da36019d`

1. [02 主动全押真实操作与播报](issues/02-active-allin.md) — 无阻塞；命令到真实界面、扣款/回应/单池结果及声音。
2. [03 黑桃浏览器图标](issues/03-spade-favicon.md) — 无阻塞；独立静态资源到浏览器标签。
3. [04 整合验收与Render更新交接](issues/04-verification-and-update.md) — 阻塞02/03；真实浏览器、race、双轴审查、可复现证据及更新说明。

一个ticket一次推进，02先行，随后03，再04。prototype审阅票01保留为一手来源，已根据用户明确批准resolved。

2026-10-10：实施票02、03、04均done；全部本地检查、race、真实浏览器及独立Standards/Spec审查通过。功能修订提交`1c8c399b00c99a11b29de0748cbc14c838cc75eb`，详见[验收证据](acceptance-evidence.md)。未推送或部署，旧公网验收状态保持。
