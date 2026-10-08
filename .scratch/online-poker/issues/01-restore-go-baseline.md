# 01: 恢复 Go 基线检查

Status: done
Type: implementation
Spec: [在线德州扑克首版规格](../spec.md)
Blocked by: None (can start immediately)
Primary acceptance cases: None（前置 Go 基线）

## What to build

恢复原始 starter 的格式、vet、测试和构建检查，使后续 TDD 的失败来自目标行为。本票只处理已记录的检查问题。

## Acceptance criteria

- [x] 统一 Go 格式、vet、测试和构建全部通过，Harness 检查也通过。
- [x] 不关闭 vet、不添加忽略规则、不修改验证逻辑来绕过已知失败。
- [x] 不实施扑克功能，不增加没有可观察行为的占位测试。
- [x] 记录实际检查命令、退出结果及仍存在的环境限制；缺工具或失败不能标为通过。

## Public test boundaries

规格范围：Testing Decisions：完成条件与证据；案例见[可追踪验收案例](../spec.md#可追踪验收案例)。

- 通过项目统一验证入口检验实际结果；本票无游戏测试 seam，也无需为格式/示例修正造业务测试。

## Validation

Implementation baseline: 336c9fc5253b699548a6bc93705a5a5964e62ec1。

2026-10-08 完成前置修复：格式化输出使用 Printf 并保留换行，gofmt 修正 starter 注释格式。

| 阶段 | 命令 | 结果 |
| --- | --- | --- |
| SDK | `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-go.ps1` | exit 0；go1.27.1 windows/amd64 |
| 修复前复现 | `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1` | exit 1；main.go 格式失败，Println 中的 %s 导致 vet/test 失败；Harness/build 通过 |
| 格式修复 | `.tools/go1.27.1/go/bin/gofmt.exe -w main.go` | exit 0 |
| 修复后完整验证 | `powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1` | exit 0；Harness、SDK、格式、vet、test、build 全通过；test 明确报告 no test files |

本票没有游戏实现或并发变更，没有新增测试、无浏览器/数据库验收，也不需要 race 检查。
已知检查失败是本票的修复前证据；不把无测试文件解释为游戏规则已通过。

### Standards review

独立审查结果：0 项发现，无标准违规或新增 smell。
来源：AGENTS.md、CODING_STANDARDS.md、harness 与 tracker 约定。

### Spec review

独立审查结果：0 项实现问题，符合本票前置清理范围。
来源：本票及父规格 Testing Decisions；未添加扑克行为或绕过检查。

两轴审查均以 `git diff 336c9fc5253b699548a6bc93705a5a5964e62ec1 -- main.go .scratch/online-poker/issues/01-restore-go-baseline.md`
覆盖实际未提交修改。提交前 baseline..HEAD 无新提交，因此使用包含工作区的 diff，避免审查空的三点 diff。
审查后的收尾仅更新本票完成状态和验证记录。

## Comments

- 用户已确认12票拆分，本票据据此发布。后续讨论与证据追加到本票，保持规则与案例引用可追踪。
- 承接[已有 starter 问题记录](../../harness-init/issues/02-clean-starter.md)。该来源记录保留，本功能不安排第二次重复修复。
