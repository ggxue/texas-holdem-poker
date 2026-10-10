# 02：本局恢复、原身份快照与回看

Status: done
Type: implementation
Blocked by: 01-current-hand-record.md
Spec: ../spec.md
Baseline: `7d375e357d31faf550b91ab61e393a63ebf3fcd0`

## What to build

玩家刷新、重连、中途加入仍能理解当前局过去的公开过程；历史金额与原参与者身份不被新钱包或新占座覆盖。
记录解释重置／补给／离房／超时，回看不抢位置，提示新记录并可回到最新。

## Acceptance criteria

- [x] HRC06–HRC09：生命周期事实及当前局保留／替换／重启清空符合规格；恢复不补播历史语音。
- [x] 钱包设100不改写旧余额0／89或撤销原全押；同席新人有独立归属且无旧手牌泄漏。
- [x] HRC10：回看保持位置与未读、回到最新恢复跟随、切标签不丢状态、新局清理。
- [x] 快照与记录的失败回滚、并发与传输身份回归通过。

## Public test boundaries

既有 HTTP／WebSocket 视图及真实浏览器滚动、标签、重连与手机详情。

## Validation

收尾补查发现手机首次打开未跟到最新，新增公开浏览器断言先失败，再修正隐藏位置保存／显示恢复。
关闭再打开回看0、标签往返后手动收起章节遇到新记录仍保留，也已通过。
修正提交 `d010bcf`，两轴复审无问题；五尺寸结果与手机动效详情再次通过。

生命周期公开测试先因缺失重置／断线／补给事实失败，接入确认事务后通过。HTTP和恢复WS提供同一完整记录，初始公告为空；新服务无旧记录。

通过：test-go.ps1 -Package ./internal/poker -Run ^TestHandRecord；browser-hand-continuity.mjs。浏览器验证回看位置0保持、未读提示、标签返回、历史全押余额0、同席新人原身份标记、新局替换。证据：artifacts/hand-continuity-browser.json（ignored）。race与双轴审查在03票末尾记录。
