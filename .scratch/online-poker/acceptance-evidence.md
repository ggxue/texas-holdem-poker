# 当前验收证据（2026-10-08）

范围为当前 spec 和 memory-reset-decision；数据库历史要求不执行。
02的 app_test.go 仍只有 AC07、AC25 两个测试函数，没有扩充02边界测试。
03–08的逐票 RED/GREEN、baseline、双轴审查见各票 Validation。

| 案例 | 证据与状态 |
| --- | --- |
| AC01–AC06 | cards_test.go、cards_cases_test.go：独立固定牌例、点数、花色、A、最佳五张及平局 |
| AC07、AC25 | app_test.go：公开HTTP/WS入房、满房、同身份重入设100；restart-smoke 补充新进程清空 |
| AC08–AC13、AC22 | game_test.go、membership_test.go：开局/阶段/等待/离房/交接/整数平分 |
| AC14–AC23、AC24 | betting_test.go、refill_test.go：四动作、短全押、单池、机器人及归零补给 |
| AC26–AC30、AC38 | takeover_test.go、timers_test.go：真实WS配合可控时钟、接管、原期限、超时、同刻离房先行及并发 |
| AC31、AC32、AC34 | socket_game_test.go 的原始消息权限断言；takeover_test.go 的旧页/旧局/旧回合；betting_test.go 的同请求重试及换内容拒绝 |
| AC41 | 本地 Chrome155 实际DOM操作：一人/两人加机器人、等待玩家、满房、四轮、下注跟注、结算、接管、退出、手动下一局；1280电脑和390手机尺寸无横向溢出，截图已人工检查。手机为模拟，实机和其他浏览器尚未验证 |
| AC42 | 本地新进程实验通过；实际云 HTTPS/WSS、休眠/重部署、两个不同浏览器尚未验证 |
| AC43 | 当前官方研究及显式免费 render.yaml 已准备；实际账号无绑卡、地域、额度和冷启动尚未核实 |
| AC33、AC35–AC37、AC39–AC40 | 已撤销，不计入有效范围 |

复现命令见[开发说明](../../docs/development.md)。
bootstrap-go 和 bootstrap-race 退出0。完整 verify -Race 中 vet、全套测试、
构建和race退出0；首轮仅 game_test.go 格式失败，随后 gofmt修正，最终 verify -Race 全部通过、退出0。
浏览器 scripts/browser-smoke.mjs 退出0、errors为空；
scripts/restart-smoke.mjs 退出0，旧Cookie新身份、无旧局/席位/房主、余额100。
报告/截图为忽略的 artifacts/browser-report.json、restart-report.json、
poker-desktop.png、poker-mobile.png；复跑可重新生成。

最终审查baseline为2536ef6be944db7c8531f53885f7a3c6efe663af，范围为票12及本地验收补充。
Standards：1项握手拒绝会挂起的P2，已增加绝对超时及response/error/close失败处理；
正常重启退出0，临时副本真实403退出1并清理子进程，复核剩余0。
Spec：0项实质发现，明确云端未完成。浏览器既有路径另验证倒计时与下注/跟注10标签，退出0。

只覆盖执行的测试与浏览器路径，不宣称所有交错已穷尽。
03–08本地功能已交付；票12和整个云上线仍未完成。
没有 Git remote、Render访问或实际云URL，不能把本地健康检查当作云端验收。
