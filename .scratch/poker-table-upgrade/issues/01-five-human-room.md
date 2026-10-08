# 01: 五真人入房、开局与房主交接

Status: done
Type: task
Blocked by: None (can start immediately)
Spec: [升级规格](../spec.md) — U1、U2、U10；UAC01–05、UAC28（本票范围）

**What to build:** 玩家可以在唯一房间中占用五个固定真人座位，与固定一个机器人开始游戏。新增座位、房满、身份接管、中途待局和房主交接均能通过真实页面观察，扩容保留既定筹码与权限规则。

## Acceptance criteria

- [x] UAC01：五个新身份依次占真人1–5，第一个为房主；第六个新身份被拒，页面明确提示房满，不抢占任何已有座位。
- [x] UAC02：分别以1、2、3、4、5真人由房主开局，每局只有一个机器人，所有参赛者各获两张不同手牌；首次扣底注后各99，底池分别2、3、4、5、6。既有归零补给、新局限制和手动开局继续适用。
- [x] UAC03（容量与控制权部分）：房满时同身份新页接管不新增座位，旧页动作无权，旧页关闭不让新页离房；成功重新入房仍将本人可用余额设100，不撤销原局状态或既有真人期限。随机起点及思考期限部分由票02、03交付并在票07组合复核。
- [x] UAC04（名单与资格部分）：局中新增身份、重用离房座位者只看公开状态并等待下一局，不继承旧身份的手牌、动作或领奖资格。随机起点保持部分由票02交付。
- [x] UAC05：真人1、2、3依次入房，1退出后新身份填1号位，再由真人2退出时，房主交给原真人3；接管不改变已有入房先后，不能用最低座位编号替代入房顺序。
- [x] 页面正确区分真人1–5和固定机器人，显示本人、空位及待局状态；本票不要求先完成新牌桌布局。
- [x] UAC28（本票范围）：新增身份的HTTP／WebSocket视图不泄漏暗牌或剩余牌序；同请求重试不重复入房或扣底注，旧控制页不能操作，退出投入不退，重连设100及既有期限行为保持。

## Public test boundaries

- 通过公开入房／开局／退出命令及按身份可见状态验证容量、金额、名单与房主；使用多身份HTTP客户端和真实WebSocket连接检查接管与原始载荷。
- 复用固定牌序和手动时钟；按公开结果断言，不断言座位数组、私有字段或函数调用次数，不新增在线fixture入口。
- 浏览器用真实入房和开局入口确认新增座位及房满提示；检查桌面和手机基本操作仍可用。

## Validation

- Implementation baseline: `bb30780165d285fb5f7529d6ac14f006cfc32851`；2026-10-09在当前master开始。本票之外的既有文档／参考图改动保留，不纳入本票提交。
- 已确认seam：公开命令→按身份可见状态、HTTP／WebSocket及真实页面；复用既定测试配置，不增加线上fixture。采用`tdd`与`codebase-design`指导本票实现。
- 工具链：`powershell -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-go.ps1`退出0，核对本地固定Go SDK。
- RED（容量）：`scripts/test-go.ps1 -Package ./internal/poker -Run '^TestUAC01'`退出1，第三真人被旧实现拒绝为room_full；扩容后同测试退出0。
- RED（房主）：`scripts/test-go.ps1 -Package ./internal/poker -Run '^TestUAC05'`退出1，低号位后来者越过原真人3成为房主；记录本次占座的确认版本并在重连／接管时保留后，UAC01／05退出0。
- 回归：`powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-go.ps1 -Package ./internal/poker -Run '^TestUAC0[1-5]'`退出0。`app_test.go`覆盖UAC01五席与第六人拒绝；`capacity_test.go`覆盖UAC02的1–5真人、底注与全局牌唯一性、HTTP／原始WS身份视图、开局重试，UAC03满房接管和原期限，UAC04待局／座位重用，UAC05交接、接管保序及离房后重新排入，另覆盖八身份并发争五席。所有行为断言经公开视图。
- 接管测试首次因通用WS辅助函数按版本过滤掉无版本的taken_over通知而失败；改为直接读取公开通知后通过，不把这次测试辅助错误计作目标行为RED。
- RED（页面）：`C:/Program Files/nodejs/node.exe scripts/browser-smoke.mjs http://localhost:18081`退出1，实际页面误将真人3标为机器人。页面改按真人席位数量区分机器人，并明确空位后复跑退出0。
- 浏览器GREEN：Chrome155实际DOM操作通过六组路径，包含1真人局、中途等待、五真人入房／第六人房满、两真人下注跟注、五真人加机器人开局／弃牌、满房同Cookie接管、旧页关闭无效及低号位复用后的入房顺序交接。1280／390宽无页面横向溢出，errors为空；已人工检查桌面和手机截图。
- 可复现浏览器入口沿用`browser-smoke.mjs`；独立Chrome测试配置的调试端口9228，服务地址作为参数传入。报告为忽略的`artifacts/browser-report.json`，截图为`artifacts/poker-capacity-desktop.png`、`artifacts/poker-capacity-mobile.png`及原桌面／手机截图，可复跑生成。
- 浏览器环境首轮Start-Process因PATH／Path重复失败，随后独立Node启动器规范化进程环境；普通沙箱中Chrome未就绪，获准在沙箱外启动专用无界面测试配置后通过。后续旧进程停止也受权限限制，已仅重启本次创建的测试服务，不影响日常浏览器。
- `node --check`检查页面脚本及浏览器烟测脚本退出0。
- 全套检查首轮因两处旧两真人余额fixture遍历新增空位而越界、持锁清理卡住，已中断为失败（退出1）；补充空位过滤后，受影响AC14–24测试退出0。
- 最终`powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1 -Race`退出0：Harness、Go版本、格式、vet、全套测试、构建和race全部通过。`git diff --check`及暂存差异检查通过。
- 双轴审查固定baseline为上述完整SHA，审查命令为`git diff --cached bb30780165d285fb5f7529d6ac14f006cfc32851`（实施在提交前暂存，含新增测试和本票，排除其他既有改动；当时HEAD仍为baseline，提交列表为空）。Standards首轮1条中文注释问题、Spec首轮1条旧fixture验收阻碍，均修正并由各自独立代理复核，最终两轴各0条未关闭发现。
- 限制：本票只完成其容量／身份／房主／基础页面范围；UAC03的思考保持和UAC04的随机起点保持仍由02／03交付，07组合复核。手机为模拟尺寸，实机、其他浏览器及真实云未验证；不宣称所有并发交错已穷尽。

## Comments

- 2026-10-09：用户确认七票拆分及依赖后发布。本票只交付扩容切片，不提前宣称随机起点、思考等待或新布局完成。
- 2026-10-09：用户调用`implement`后，从本票前沿实施；本票范围验收满足，按本地提交流程完成。旧云票12保持独立blocked。
