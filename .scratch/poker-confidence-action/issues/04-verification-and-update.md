# 04：整合验收与Render更新交接

Status: ready-for-agent
Type: implementation
Blocked by: 02-active-allin.md, 03-spade-favicon.md
Spec: [规格](../spec.md)
Implementation baseline: pending

## What to build

验证真实全押、声音、满员结果及图标，完成本地发布提交，向用户提供将新版本更新到已有Render服务的步骤。

## Acceptance criteria

- [ ] AAC17及真实媒体/按钮/隐私整合；记录桌面/模拟手机证据及局限。
- [ ] 仓库bootstrap、verify含race、JS语法、diff检查通过，缺工具不得跳过。
- [ ] Standards与Spec独立审查包含全部工作；correctness、状态转换、并发问题解决。
- [ ] scoped local commits完成；记录确切本地SHA和Render更新/验证步骤。
- [ ] 不猜URL，不把本地测试当公网验收，不自行标旧云票完成或执行部署。

## Public test boundaries

沿用02/03测试范围，真实浏览器使用回环fixture；云真实验证需实际URL，保持独立记录。

## Validation

Pending.
