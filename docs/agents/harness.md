# Harness 使用与验证

本仓库使用 Matt Pocock 的 skills、本地 Markdown 任务跟踪和可重复的验证入口。
项目入口是 [AGENTS.md](../../AGENTS.md)。当前游戏规则见[规格](../../.scratch/online-poker/spec.md)，运行仅用内存，不使用数据库。

## 日常入口

在仓库根目录运行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-go.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1
```

仅修改文档或 skills 时：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1 -Stage Harness
```

Go SDK 固定在 [go-toolchain.json](../../scripts/go-toolchain.json)，安装到
`.tools/go1.27.1/go`，下载后校验官方 SHA256。脚本不会修改系统 PATH。
GoLand 的 Go SDK 可以选择这个目录。Go 构建、模块和测试缓存均放在 `.tools/`；
构建产物放在 `artifacts/`，这两个目录均不纳入版本管理。

完整验证依次运行 harness 结构检查、Go 版本核对、格式检查、vet、测试和构建。
任一失败都会返回非零退出码，继续显示其他检查结果。涉及并发行为时使用 `-Race`；
Windows race 检查还需要可用的 C 编译器，缺少时不会被当作通过。

## 技能与会话

新会话应从 `D:/texas-poker` 启动。使用 `$skill-name` 或 `/skills`，集成界面
不支持选择器时，可以明确要求读取仓库里的对应 `SKILL.md`。
新安装的技能会在下一轮发现；若没有显示，重启该 Codex 会话。
7 个显式技能不进入默认的可自动调用元数据列表；实际探测已确认 `$wayfinder`
会加载完整技能内容。默认自动技能列表中的 8 个技能也已通过新会话探测。

后续规划入口示例：

```text
$wayfinder 帮我确定德州扑克后端第一版的范围和关键选择。
先讨论目标与决策，不开始业务实现。
```

讨论达成一致后，使用 `$to-spec` 保存规格，再用 `$to-tickets` 拆可验证的行为切片。
每个实现任务链接规格、验收案例和验证结果。实现前记录 baseline commit SHA，
审查时明确指定基准与规格。`$implement` 的上游流程包含本地 commit。

当前使用 [本地任务跟踪](issue-tracker.md) 和 [单一领域文档布局](domain.md)。
术语表和 ADR 在实际讨论产生内容后再创建。技能来源与兼容方式见
[skills-source.md](skills-source.md)。

## 初始化证据

验证结果记录在 [初始化任务](../../.scratch/harness-init/issues/01-initialize.md)。
这份记录区分 harness 检查、工具链检查、新会话加载和当前示例程序的检查结果。
已有失败必须修复后才可报告完整验证通过，不能通过关闭 vet 或忽略返回码隐藏。

初始化结果：harness 检查和 Go 构建通过；原始 main.go 的格式与 Println 诊断
导致完整 Go 验证失败，已记录为一个 draft 任务。当前只读 CLI 的默认策略限制了
探测会话的 shell 文件读取，磁盘配置由本次执行环境的验证脚本核对。


## 当前实施验证补充

Windows race 使用 scripts/bootstrap-race.ps1 安装校验过的 w64devkit；解压必须等待进程结束，
不能把GUI自解压启动成功当作安装完成。当前 Go 全套和 race 已通过。
真实 Chrome 需要沙箱允许其子进程，使用独立测试配置，不操作日常浏览器。
[验收证据](../../.scratch/online-poker/acceptance-evidence.md)区分本地、手机模拟和未完成云验证。
