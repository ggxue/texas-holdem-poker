# 06: 中途入房、主动退出与房主交接

> 当前覆盖：状态只在单进程内存中；重新入房本人余额设100；原有数据库提交/真实数据库fixture/跨重启持久性条款已撤销，不执行。游戏规则和公开seam沿用修订spec；详见[当前决定](../memory-reset-decision.md)。

Status: done
Type: implementation
Spec: [在线德州扑克首版规格](../spec.md)
Blocked by: [04](04-bet-allin-single-pot.md)
Primary acceptance cases: AC10–AC12、AC31

## What to build

中途入房只看公开进度并等下一局；主动退出立即弃牌离房，已投入留池、剩余余额保存，房主交接、空位可被新身份占用。

## Acceptance criteria

- [x] 开局参赛身份冻结；中途新身份无手牌/行动/领奖资格，下一局才付底注参赛，不能继承旧占座者手牌。
- [x] 主动退出立即释放座位；未结算的全押者也弃牌失去资格，已投入不退，剩余余额按身份保存（AC29主动退出子场景）。
- [x] 房主交给最早仍在房间的真人，原座位不移动、不覆盖在进行的局；无真人时无房主，机器人不自行开下一局。
- [x] 仅真人2时固定顺序跳过空真人1；新占真人1与旧参赛者身份分离；先前游戏继续正确推进/结算。
- [x] 退出、资格、池、余额和房主在内存共同更新成功才通知；页面有退出/等待标识，完整验证AC31含中途待局者的网络隐私。
- [x] 普通断开不冒充主动退出；本票不以关闭页面立即弃牌代替未来08宽限。

## Public test boundaries

规格范围：D2、D6–D9；案例见[可追踪验收案例](../spec.md#可追踪验收案例)。

- 公开入房/退出/牌局命令和各身份视图，验证冻结参赛资格、座位重用和全押退出。
- 进程内存与不同身份HTTP/WebSocket核对资格/余额/房主共同提交；实际消息中新人不能收到旧手牌。

## Validation

Implementation baseline: 56dce87222fbb482319cc69180754da9fed5d79c。

实现及自动化通过；实际浏览器/最终检查待12，不提前标done。
- RED：TestAC11And12、TestAC29Active因leave未支持返回400退出1；中途入房现有冻结名单行为已通过。
- GREEN：scripts/test-go.ps1 -Package ./internal/poker -Run 'TestAC(10|11|29Active)'退出0。包括HTTP/原始WebSocket等待者隐私、下一局资格、座位重用、房主交接、全押离房无退款、结算后奖项不撤销。
- Node --check网页JS与gofmt退出0。
- Standards/Spec均0项问题；审查git diff --cached 56dce87 --。
- 普通关页不等于主动退出；07接管与08宽限尚待后票。完整测试/构建/race/真实浏览器检查在12汇总。
按[项目验证约定](../../../docs/agents/harness.md)记录命令、退出码、目标 RED/GREEN、
可复现浏览器/内存行为证据和限制；格式、vet、测试、构建及适用并发检查缺工具或失败不得静默跳过。
完成前以实施 baseline 和本规格分别审查 Standards/Spec，记录结果；仅验收满足才标 done。

## Comments

- 用户已确认12票拆分，本票据据此发布。后续讨论与证据追加到本票，保持规则与案例引用可追踪。
- 依赖04提供弃牌、单池早结算与全押资格；普通断线/期限由08完成。


## 最终本地验收补充

本票本地功能已完成，先前缺浏览器/C编译器的记录为当时状态。
最终构建已完成真实 Chrome DOM 对局检查和手机尺寸模拟，Go全套及race通过；
逐票审查发现均已修复。复现与AC映射见[当前验收证据](../acceptance-evidence.md)。
实际手机、其他浏览器和云端休眠/HTTPS/WSS留在12，本票不宣称已上线。
