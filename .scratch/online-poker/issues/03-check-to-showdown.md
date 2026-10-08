# 03: 从开局过牌到摊牌

> 当前覆盖：状态只在单进程内存中；重新入房本人余额设100；原有数据库提交/真实数据库fixture/跨重启持久性条款已撤销，不执行。游戏规则和公开seam沿用修订spec；详见[当前决定](../memory-reset-decision.md)。

Status: in-progress
Type: implementation
Spec: [在线德州扑克首版规格](../spec.md)
Blocked by: [02](02-persistent-room-entry.md)
Primary acceptance cases: AC01–AC06、AC08、AC13、AC22

## What to build

房主以一名或两名真人加机器人开局，每人底注1；按固定顺序过牌走完四轮，摊牌并分配底池，查看结果后由房主手动开始下一局。

## Acceptance criteria

- [ ] 非房主开局、进行中重开拒绝；同一已成功开局请求重试不重复收底注或发牌（AC09相关子场景）。
- [ ] 两人/三人局底池2/3，每人两张手牌；全过牌按0→3→4→5公共牌推进，每局只收一次底注，本轮投入从0开始。
- [ ] 按规格固定牌例识别十类牌型、完整点数、A特殊顺子、花色破平局；Best Five允许0/1/2手牌，共用最佳公共牌仍平局。
- [ ] 单池最强/平局分配及21零头例准确；结算只入账一次，结果保留至下一局开始。
- [ ] 机器人能过牌就立即过牌；游戏流程及机器人策略每行逻辑中文注释，不等待动画。
- [ ] 牌序、参赛身份、投入、阶段、底池与结果每次在内存共同更新后广播；暗牌只给本人，摊牌公开未弃牌者（AC31相应子场景）。
- [ ] 中文电脑/手机页面可完成该完整路径；未交付的下注动作不可显示为可成功操作，不将该中间版本视作最终上线。

## Public test boundaries

规格范围：D2 开局、D3 底注、D4–D6、D8–D9；案例见[可追踪验收案例](../spec.md#可追踪验收案例)。

- 主要 seam 为公开牌局命令→按身份视图，受控牌序复现发牌、轮次、结算及平局；比牌使用规格独立固定牌例，不复制算法算预期值。
- 进程内存 和 HTTP/WebSocket 验证内存更新后通知、一次开局/派奖及网络载荷无其他暗牌或剩余牌序。

## Validation

Implementation baseline: 31a11cdd2a17c76741a9fde2f908b91732d49dc8。

实现及自动化检查已完成；电脑/手机实际浏览器操作仍待12整体验收，不将本票标done。

- RED：TestAC01RoyalFlush、TestAC01TenCategoriesInOrder分别因牌型缺失失败；TestAC08And13因start未支持返回400失败。
- GREEN：scripts/test-go.ps1 -Package ./internal/poker -Run 'TestAC(01|02|03|04|05|08|09|22|31Socket)'，退出0；覆盖牌型点数/花色/A规则/最佳五张、1/2真人开局、重复请求、四轮过牌、21零头与原始WebSocket权限载荷。
- SDK gofmt、go vet ./...、go build -o artifacts/build/texas-poker.exe .退出0。Harness验证退出0。全套测试按implement约定留到全部本地实施结束运行。
- Standards：1项测试载荷断言问题；已改为检查原始消息字段并重跑TestAC31Socket退出0。
- Spec：同一载荷断言问题及页面未显示当前行动者；均已修复。后续功能不计入03验收。
- 审查范围：git diff 31a11cd...5787226 -- internal/poker .scratch/online-poker/issues/03-check-to-showdown.md，并复核后续修复。期间出现外部本地提交5787226，保留不改写。
- 限制：当前无可用浏览器自动化和C编译器；真实电脑/手机验收、race未通过且明确待执行。实现依赖可供04继续，人工/环境验收未冒充通过。
按[项目验证约定](../../../docs/agents/harness.md)记录命令、退出码、目标 RED/GREEN、
可复现浏览器/内存行为证据和限制；格式、vet、测试、构建及适用并发检查缺工具或失败不得静默跳过。
完成前以实施 baseline 和本规格分别审查 Standards/Spec，记录结果；仅验收满足才标 done。

## Comments

- 用户已确认12票拆分，本票据据此发布。后续讨论与证据追加到本票，保持规则与案例引用可追踪。
- AC22用可控的结算输入验证整数分配与零头；04补齐下注形成底池的真实端到端路径。
- 本票完成全部过牌路径及机器人Check；下注/跟注/弃牌/全押由04补齐，归零补给由05补齐。
