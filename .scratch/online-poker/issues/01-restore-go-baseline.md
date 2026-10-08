# 01: 恢复 Go 基线检查

Status: ready-for-agent
Type: implementation
Spec: [在线德州扑克首版规格](../spec.md)
Blocked by: None (can start immediately)
Primary acceptance cases: None（前置 Go 基线）

## What to build

恢复原始 starter 的格式、vet、测试和构建检查，使后续 TDD 的失败来自目标行为。本票只处理已记录的检查问题。

## Acceptance criteria

- [ ] 统一 Go 格式、vet、测试和构建全部通过，Harness 检查也通过。
- [ ] 不关闭 vet、不添加忽略规则、不修改验证逻辑来绕过已知失败。
- [ ] 不实施扑克功能，不增加没有可观察行为的占位测试。
- [ ] 记录实际检查命令、退出结果及仍存在的环境限制；缺工具或失败不能标为通过。

## Public test boundaries

规格范围：Testing Decisions：完成条件与证据；案例见[可追踪验收案例](../spec.md#可追踪验收案例)。

- 通过项目统一验证入口检验实际结果；本票无游戏测试 seam，也无需为格式/示例修正造业务测试。

## Validation

Implementation baseline: 待实施开始、任何代码编辑前记录 SHA。

尚未实施，本次发布不构成游戏验收通过。
按[项目验证约定](../../../docs/agents/harness.md)记录命令、退出码、目标 RED/GREEN、
可复现浏览器/数据库证据和限制；格式、vet、测试、构建及适用并发检查缺工具或失败不得静默跳过。
完成前以实施 baseline 和本规格分别审查 Standards/Spec，记录结果；仅验收满足才标 done。
本票为已知检查修复，不制造业务 RED/GREEN 或占位测试。

## Comments

- 用户已确认12票拆分，本票据据此发布。后续讨论与证据追加到本票，保持规则与案例引用可追踪。
- 承接[已有 starter 问题记录](../../harness-init/issues/02-clean-starter.md)。该来源记录保留，本功能不安排第二次重复修复。

