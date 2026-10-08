# 牌桌升级：已确认实施票据索引

Status: done
Type: ticket-plan
Spec: [升级规格](spec.md)
Breakdown approved: 2026-10-09 (Asia/Singapore)
Implementation status: done (01–07 local scope complete)

用户已确认七票拆分及以下阻塞关系；本索引发布时只做规划。随后用户调用`implement`并授权持续完成全部票，01–07已完成本地实施、整合验收与分票提交；不启动云部署。
父规格保持原状。原[交接](handoff.md)是拆票前快照，不应把其中的下一步说明当成票据尚未发布。

## Tickets and blocking edges

| 票据 | 直接阻塞于 | 主要验收 |
| --- | --- | --- |
| [01 五真人入房、开局与房主交接](issues/01-five-human-room.md) | 无 | UAC01–05的容量、身份、名单和房主；UAC28局部回归 |
| [02 每局随机起点与统一零头顺序](issues/02-random-start-and-odd-chips.md) | 01 | UAC06–12；UAC03–04起点保持 |
| [03 机器人真实思考与页面倒计时](issues/03-bot-thinking-and-countdown.md) | 无 | UAC13–21；UAC03思考保持 |
| [04 桌面与手机六座位布局及操作区](issues/04-responsive-six-seat-table.md) | 01 | UAC22、24、25；UAC27布局与操作 |
| [05 简洁牌面、公共牌区与十类参考栏](issues/05-cards-and-hand-reference.md) | 04 | UAC23；UAC27牌面与参考 |
| [06 独立结算结果区](issues/06-settlement-results.md) | 04 | UAC26；UAC27结果 |
| [07 整合验收与升级证据归档](issues/07-integrated-acceptance.md) | 02、03、05、06 | UAC28及UAC01–27组合与证据核对 |

## Execution notes

- 01–07全部done；各票逐一实施、验证、双轴审查并本地提交。升级本地范围已完成，外部实机／其他浏览器与云验收限制详见证据。
- 各票实施前已读取票据和相关规格，Validation记录独立baseline。本地升级七票收口，云票12仍独立blocked。
- 03可以在现有容量及顺序下交付；01、02、04后续不得破坏已交付思考时序，07验证五身份、随机起点与最终UI的组合。
- 04的版面先交付，05补齐牌面与参考，06交付结果；05与06互不阻塞。每票自行验证，07不承接前票遗漏的实现或必需测试。
- 复用公开命令／按身份可见状态、HTTP／WebSocket、固定牌序与手动时钟；随机控制随对应功能票进入同一离线配置入口。无独立前置重构票，无新增线上fixture。
- 适用基线：[首版规格](../online-poker/spec.md)和[内存决定](../online-poker/memory-reset-decision.md)。覆盖冲突以升级规格为准，不恢复已撤销的持久化与恢复保证。
- 旧云票12及真实云验收独立保留；本批票据不依赖云访问，不创建云资源或部署。

## Publication validation

- 2026-10-09：`powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1 -Stage Harness`退出0，Harness通过。
- `git diff --check`退出0；七张票据及本索引的本地链接、尾部换行和空白格式检查通过。
- 七票连续编号、ready-for-agent状态、必需章节及已批准的直接阻塞边检查通过；所有阻塞票编号小于被阻塞票，无循环。01–06包含UAC01–28的对应引用，07负责整体核对；引用检查不等于验收通过。
- 拆票发布轮当时未执行游戏RED／GREEN、Go、race或新UI浏览器验收；当时未实施业务代码、未提交、未部署。上述结果只验证文档发布，不证明任何游戏行为已经通过。

## Comments

- 2026-10-09：用户回复“可以，确认拆分”，按`to-tickets`发布七个独立本地票据，保留既有未提交文档和父规格。
- 2026-10-09：票01完成，提交`569bb74`；完整验证含race、Chrome桌面／手机模拟及双轴复审通过，详见本票Validation。索引更新保留在原未提交规划文档中，不混入票01的代码提交。

- 2026-10-09：02提交33e1f13、03提交94e229e、04提交13edbf1、05提交82cdccd、06提交af317c2；对应票据各记录本地检查和双轴审查。07以af317c2继续组合验收，详见[独立升级证据](acceptance-evidence.md)。

- 2026-10-09：07完成；新增公开组合、真实HTTP待确认RED／GREEN、最终普通Go／race与production随机／离线3s完整浏览器均通过；Standards 0、Spec 0。全部七票done，完整回链和外部限制见独立升级证据。
