# 02：主动全押真实操作与播报

Status: in-progress
Type: implementation
Blocked by: None
Spec: [规格](../spec.md)
Implementation baseline: `0d71575da9400ff48688d7152198d321da36019d`

## What to build

在真实牌桌增加合法的主动全押，完整支持首注、提高目标、重新应答、短额跟注和既定单池结算；服务端确认后公开播报精确金额。按钮简洁，辅助提示和席位解释状态。

## Acceptance criteria

- [ ] AAC01–14、19：扣款、目标、连续全押、短额、小码全池、唯一行动者、重连、重试、期限、拒绝/回滚、隐私与事件。
- [ ] AAC15–16：只渲染合法动作；金额由服务端确认；按钮不带“（不足全押）”；真实全押语音只读一次。
- [ ] 原四动作、机器人策略、内存规则及公开权限保持；无自由金额输入或原型资源上线。
- [ ] 针对公开行为逐个red→green，记录检查及独立Standards/Spec审查。

## Public test boundaries

公开命令→身份视图、真实HTTP/WS和浏览器；牌序/时间/开局钱包只使用已有离线构造/fixture先例。

## Validation

2026-10-10：实现与针对性检查通过，待双轴审查；整套verify/race及最终浏览器回归在票04记录。

- TDD：AAC01在缺少合法allin动作时预期失败，再通过；AAC08在唯一短额回应者缺少allin时预期失败，再通过。首次旧DTO解码失败是测试准备问题，不计行为red。
- 浏览器red：正式页缺少“全押 99”失败；接入真实按钮后green。媒体实际播放“玩家一全押九十九筹码。”，无重复全押。
- `go test ./internal/poker -run 'TestAAC|TestLargeAllIn' -count=1`通过。公开HTTP覆盖AAC01–14、19；复用离线余额/牌序/时钟fixture，不增加线上控制入口。AAC05/06一次测试初稿在轮推进后检查旧目标，已改为观察动作原始响应，非业务故障。
- `POKER_CDP_URL=http://127.0.0.1:9237 node scripts/browser-allin.mjs http://localhost:18082`通过。独立五真人+机器人，真实提交拦截/待确认、HTTP/WS隐私、媒体播报、池600六人平分、下一局、1280×720/390px布局。
- 通过正常全押/跟注/弃牌形成101/99不同钱包，再下一局真实观察“跟注 98”“全押 98”面对目标100；辅助提示、全押状态、目标不降低和小额分整个池通过。没有注入浏览器余额。
- 视觉已检查`artifacts/allin-production-before.png`、`allin-production-call.png`、`allin-production-six-results.png`、`allin-production-short-mobile.png`；报告`artifacts/allin-browser-report.json`。
- 新女声动作素材只重录`allin-action.wav`，原素材保留。SAPI在沙箱内不可用，授权的本地PowerShell运行后录制成功。
- 原CDP9228被已有进程占用，保留该进程；浏览器工具增加可选`POKER_CDP_URL`，使用专用9237和隔离profile验收。
- 本地通过不代表公网完整验收；旧发布票保持原状态。
