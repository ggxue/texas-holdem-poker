# 07: 同身份新页面接管

> 当前覆盖：状态只在单进程内存中；重新入房本人余额设100；原有数据库提交/真实数据库fixture/跨重启持久性条款已撤销，不执行。游戏规则和公开seam沿用修订spec；详见[当前决定](../memory-reset-decision.md)。

Status: in-progress
Type: implementation
Spec: [在线德州扑克首版规格](../spec.md)
Blocked by: [04](04-bet-allin-single-pot.md)
Primary acceptance cases: AC32（完整组合）

## What to build

同一身份打开新标签页，新页接管原座位和牌局；旧页显示已在其他页面打开并停止自动重连。旧页面和连接的迟到事件不影响新页。

## Acceptance criteria

- [ ] 接管保留同一身份、座位、入房先后、房主和本局资格；成功重新入房余额按AC25设100，不重复占席（AC26接管子场景）。
- [ ] 有效控制代际保存在当前进程内存，旧页动作、重连或关闭事件不能改变新状态；接管不是确认离房/弃牌/房主交接。
- [ ] 新页只收到自身合法私有视图，旧页明确提示接管并停止自动重连。
- [ ] 真实两页与迟到请求覆盖旧局/旧回合/旧控制权及伪造余额的拒绝，整合AC32先前金额/回合子场景，拒绝后牌/池/余额不变。

## Public test boundaries

规格范围：D2 控制权、D8–D9；案例见[可追踪验收案例](../spec.md#可追踪验收案例)。

- 同一Cookie的两个真实HTTP/WebSocket页面、公开牌局命令和身份视图，验证一次控制权及旧连接迟到事件。
- 进程内存核对代际与房间在内存共同更新，不测试私有连接字段或锁结构。

## Validation

Implementation baseline: 4ba2717a80af0e88892fa18a1d5ca027bdc87fc9。

实现及自动化通过，实际浏览器/最终检查待12，不提前标done。
- RED：TestAC26NewPage因旧WebSocket没有taken_over消息而读超时退出1。
- GREEN：scripts/test-go.ps1 -Package ./internal/poker -Run 'TestAC(07|25|26|32|08|22|14|10)'退出0；两个同Cookie HTTP页与真实WS验证接管、旧动作/旧页重连拒绝及旧关闭不影响新页；保留已有02仅两条测试。
- PageID每次页面加载生成，control按成功入房递增；POST需pageID/control，WS查询同样绑定。旧登记页不能把迟到join变为新接管。
- Standards/Spec均0项问题；审查git diff --cached 4ba2717 --；JS语法与gofmt通过。
- 08补齐真实期限后验证不延长；race/完整测试/电脑手机浏览器在12汇总。
按[项目验证约定](../../../docs/agents/harness.md)记录命令、退出码、目标 RED/GREEN、
可复现浏览器/内存行为证据和限制；格式、vet、测试、构建及适用并发检查缺工具或失败不得静默跳过。
完成前以实施 baseline 和本规格分别审查 Standards/Spec，记录结果；仅验收满足才标 done。

## Comments

- 用户已确认12票拆分，本票据据此发布。后续讨论与证据追加到本票，保持规则与案例引用可追踪。
- 08引入行动期限后验证接管保留原期限，AC26完整组合作为08验收；本票不通过重置时间制造假成功。
- 直接阻塞只有04，不额外依赖05补给或06主动退出。
