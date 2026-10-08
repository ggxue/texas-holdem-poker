# 08: 30 秒断线宽限与行动期限

> 当前覆盖：状态只在单进程内存中；重新入房本人余额设100；原有数据库提交/真实数据库fixture/跨重启持久性条款已撤销，不执行。游戏规则和公开seam沿用修订spec；详见[当前决定](../memory-reset-decision.md)。

Status: in-progress
Type: implementation
Spec: [在线德州扑克首版规格](../spec.md)
Blocked by: [06](06-membership-host-transfer.md)、[07](07-page-takeover.md)
Primary acceptance cases: AC09、AC26–AC29（完整组合）；AC30正常运行部分

## What to build

短暂断线可回原座位与牌局，行动超时自动过牌或弃牌；断线宽限到期离房并交接房主。重连/接管不重置行动计时，宽限期间不开始新局。

## Acceptance criteria

- [ ] 普通控制连接丢失保留座位/房主30秒；10秒重连时原新行动期限只剩约20秒，重连恢复原已确认状态。
- [ ] 行动已用25秒后断线，约5秒后按能Check则Check否则Fold；之后宽限内重连不撤销动作或恢复旧期限。
- [ ] 在线行动超时不等于离房；主动退出立即离房；全押者宽限到期离房也失去未结算资格，投入不退、剩余余额保存。
- [ ] 期限保存为绝对时间，行动和断线宽限独立；同刻到期离房先于行动超时（AC38相应子场景）。
- [ ] 普通宽限期间拒绝新开局，不能给离线者误补/收底注；所有自动事件在内存共同更新成功才生效。
- [ ] 同身份新页接管不启动离房，保留原行动期限；旧连接和旧计时事件不能改变新控制状态，整合AC26/AC09已完成子场景。
- [ ] 中文页面显示明确倒计时、断线/恢复状态和合法按钮，电脑/手机均可操作。

## Public test boundaries

规格范围：D7 正常运行、D8–D9；案例见[可追踪验收案例](../spec.md#可追踪验收案例)。

- 公开行动/重连/退出命令及按身份视图，用可控制时间复现30秒、25+5秒和同刻到期，不sleep等待真实30秒。
- 真实HTTP/WebSocket断开/重连及进程内期限更新，覆盖与接管、全押、房主交接的并发交错。

## Validation

Implementation baseline: f0814f36f929e08798629130ab537bcbe8e81733。

实现及自动化/race通过，浏览器整体验收与完整检查待12，不提前标done。
- RED：AC27期限0、AC28/30/38无断线标记、AC29未自动弃牌，目标缺失退出1；审查发现握手重试延长期限后，TestAC30Entry10秒重试复现退出1。
- GREEN：scripts/test-go.ps1 -Package ./internal/poker -Run 'TestAC(27|28|29Owing|29Expired|30|38|26Takeover|26NewPage|08|07|25)'退出0；握手重试修复后TestAC30Entry退出0。
- 使用公开Clock控制时间而非sleep；实际HTTP/WS断开、旧连接关闭、全押到期离房、25+5超时及8并发迟到命令覆盖。
- C工具已按官方SHA256安装portable w64devkit2.10.0/GCC16.2，libsynchronization.a存在。CC绝对路径、CGO_ENABLED=1，以test-go.ps1 -Run 'TestAC(26|27|28|29|30|38)' -Race退出0，无数据竞争报告。
- Standards首次0项、Spec连接生命周期发现1项；Connecting握手期限与保留最早期限修复后双轴复核0项剩余问题。审查git diff --cached f0814f3 --。
- WS应用心跳仅用于真实在线连接故障检测，不创建额外保活客户端；关闭时释放ticker/单定时器/连接。新控制WS握手成功才清除连接等待，原行动绝对期限保持。
- 当前代码与测试可供12继续，完整格式/vet/构建/全套测试和真实Chrome电脑/手机尺寸验收随后汇总。
按[项目验证约定](../../../docs/agents/harness.md)记录命令、退出码、目标 RED/GREEN、
可复现浏览器/内存行为证据和限制；格式、vet、测试、构建及适用并发检查缺工具或失败不得静默跳过。
完成前以实施 baseline 和本规格分别审查 Standards/Spec，记录结果；仅验收满足才标 done。

## Comments

- 用户已确认12票拆分，本票据据此发布。后续讨论与证据追加到本票，保持规则与案例引用可追踪。
- 06提供离房资格/交接，07提供控制代际，二者均为真实前提。
- 本票只验证进程存活时的期限与重连；应用重启清空内存，09–11已撤销，没有数据库写失败或故障恢复依赖。
