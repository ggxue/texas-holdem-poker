# 01: 初始化项目 harness

Status: done
Type: task
Blocked by: None
Spec: 用户已同意本会话提出的 harness 初始化方案。

## Scope

建立仓库级 AGENTS.md、固定版本的 Matt Pocock skills、本地 Markdown 任务操作、
领域文档布局、代码审查标准、Go 工具链与验证入口。核对依赖和新会话加载。
保留 main.go 和 go.mod 的原始内容。第一版产品范围由后续 wayfinder 讨论确定。

## Acceptance criteria

- [x] 项目入口能找到任务跟踪、领域文档规则、技能来源及验证命令。
- [x] 核心技能及必需依赖安装完整，模板引用有效，版本与许可证可追溯。
- [x] 仓库本地 Go SDK 可执行，版本和下载 SHA256 与固定配置一致。
- [x] harness 检查和完整 Go 检查均已实际执行，结果及已有失败明确记录。
- [x] 新 Codex 会话加载项目规则并发现 skills，实际集成限制明确记录。
- [x] 没有添加游戏业务代码或猜定产品规则。

## Validation

验证日期：2026-10-08。

| 检查 | 实际结果 |
| --- | --- |
| 技能安装 | 15 个，固定上游 SHA `f3fc5632f401156837ee3872f14fe33ccf1024ea`，保留 MIT 许可证 |
| 元数据验证 | skill-creator 验证器通过；7 个带额外上游字段的技能仅在临时副本上验证支持的字段，安装文件未修改；原始调用策略单独检查 |
| Harness | 结构、依赖、资源哈希、实际本地链接、PowerShell 语法全部通过 |
| 工具链 | 官方 Go 1.27.1 windows/amd64 ZIP 的 SHA256 校验通过，解压后版本核对通过 |
| Bootstrap 重跑 | 使用已安装 SDK 成功返回，不重复下载 |
| Go 格式 | 失败：main.go 的原始 `//TIP` 注释需格式化 |
| Go vet | 失败：main.go:13:2 的 fmt.Println 含 `%s` 格式指令 |
| Go tests | 失败：默认 vet 触发同一诊断；当前没有测试文件 |
| Go build | 通过，产物位于忽略的 artifacts/build/ |
| Race | 未运行；当前没有并发行为或测试，本任务不声明 race 已通过 |
| 新会话自动加载 | 独立 Codex CLI 只读、ephemeral 会话已自动收到 AGENTS.md 与 8 个可隐式调用技能元数据 |
| 显式调用加载 | 独立会话通过 `$wayfinder` 获得完整 SKILL.md，正确识别 5 个 map 区段和 4 类决策任务；未运行规划流程 |

命令：`powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1`。
完整检查返回 1，准确报告 Go 格式、vet、测试失败；没有跳过或放宽检查。
单独的 `-Stage Harness` 返回 0。示例代码问题交给
[02-clean-starter.md](02-clean-starter.md)，状态为 draft。

会话探测证据保存在本地忽略的 `artifacts/harness/`。第一个 CLI 只读探测的
文件读取命令被其默认执行策略拒绝，因此磁盘资源和 YAML 策略由当前会话的
验证脚本核对；自动规则/技能注入与 `$wayfinder` 的实际内容加载另有直接证据。
这不构成对 GoLand UI 技能选择器的操作测试。

## Comments

初始化之前仓库没有 commit 或 remote。初始化建立本地 Git 基线，以便后续实现
记录 baseline SHA 并执行差异审查。main.go 和 go.mod 保持原始内容。

自动审批拒绝删除 `disable-model-invocation` 字段，认为这可能削弱上游调用限制。
采用安全替代：保留全部上游文件，仅使用临时副本辅助元数据检查，原始限制单独验证。
