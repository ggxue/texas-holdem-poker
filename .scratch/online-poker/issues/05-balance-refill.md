# 05: 累计余额与归零补给的下一局

> 当前覆盖：状态只在单进程内存中；重新入房本人余额设100；原有数据库提交/真实数据库fixture/跨重启持久性条款已撤销，不执行。游戏规则和公开seam沿用修订spec；详见[当前决定](../memory-reset-decision.md)。

Status: done
Type: implementation
Spec: [在线德州扑克首版规格](../spec.md)
Blocked by: [04](04-bet-allin-single-pot.md)
Primary acceptance cases: AC24

## What to build

多局余额连续，只有下一局开局前归零的本局参与者免费补到100；机器人同样适用，页面显示准确余额和底注。

## Acceptance criteria

- [x] 结算余额0/37/180，新局先只补第一人到100，扣底注后99/36/179；正余额不补。
- [x] 正余额1不补，支付底注后全押；途中全押到0不立刻补或恢复行动（组合AC17）。
- [x] 机器人余额跨局累计，不每局重置；只有本次实际参赛者接受必要补给及底注。
- [x] 补给、开局名单、底注和牌局同一内存状态更新，稳定请求重试不重复执行开局补给；刷新成功重新入房按AC25设100；初始发放与新增补给区分，页面只显示确认余额。

## Public test boundaries

规格范围：D3 余额与补给、D8；案例见[可追踪验收案例](../spec.md#可追踪验收案例)。

- 公开下一局命令和本人/公共筹码视图验证0/37/180案例及局中不补。
- 进程内存查读重访和重复开局的可见结果；测试余额控制不得暴露为线上接口。

## Validation

Implementation baseline: b38dad8489260368d9dba2072781d2bb2348fa4d。

实现及自动化通过，实际浏览器/最终检查待12，不提前标done。
- RED：TestAC24在0真人或机器人开局返回insufficient_chips，退出1。
- GREEN：scripts/test-go.ps1 -Package ./internal/poker -Run TestAC24退出0；0/37/180扣底注后99/36/179，机器人归零同样补100，1不补且底注全押，稳定请求重试不重复补给。
- Standards与Spec均0项实质问题；审查git diff --cached b38dad8 --。
- gofmt通过；全套格式/vet/测试/构建与浏览器证据在12汇总，race缺C编译器仍待处理。
按[项目验证约定](../../../docs/agents/harness.md)记录命令、退出码、目标 RED/GREEN、
可复现浏览器/内存行为证据和限制；格式、vet、测试、构建及适用并发检查缺工具或失败不得静默跳过。
完成前以实施 baseline 和本规格分别审查 Standards/Spec，记录结果；仅验收满足才标 done。

## Comments

- 用户已确认12票拆分，本票据据此发布。后续讨论与证据追加到本票，保持规则与案例引用可追踪。
- 本票验证同进程中的跨局余额及开局补给；重新入房按AC25设100，不涉及数据库失败或重启恢复。


## 最终本地验收补充

本票本地功能已完成，先前缺浏览器/C编译器的记录为当时状态。
最终构建已完成真实 Chrome DOM 对局检查和手机尺寸模拟，Go全套及race通过；
逐票审查发现均已修复。复现与AC映射见[当前验收证据](../acceptance-evidence.md)。
实际手机、其他浏览器和云端休眠/HTTPS/WSS留在12，本票不宣称已上线。
