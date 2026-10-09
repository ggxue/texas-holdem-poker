# 04：整合验收与Render更新交接

Status: in-progress
Type: implementation
Blocked by: 02-active-allin.md, 03-spade-favicon.md
Spec: [规格](../spec.md)
Implementation baseline: `253caecc76a8ed15ea1bf7a53d8cdc8b9f2d6c71`

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

2026-10-10：完整检查与真实浏览器全部通过，最终修订提交后复核独立审查，详见[验收证据](../acceptance-evidence.md)。

- bootstrap固定Go1.27.1成功；`verify.ps1 -Race`通过Harness、格式、vet、全部Go测试、构建及race。初次UAC09旧合法列表失败已按已确认allin规则修正，完整重跑通过，无race告警。
- `node --check`检查app.js、voice.js、browser-client.mjs、browser-allin.mjs；`git diff --check`通过。
- Chrome155独立profile/CDP9237：`browser-voice.mjs ... --casino`通过82WAV/14组媒体检查；`browser-smoke.mjs ... --casino --layout --cards --results --pending --full-hand --tie-six`通过13组旧流程/布局检查；`browser-allin.mjs ... --icon`全部通过，真实短额跟注/全押、六人单池分奖及本地SVG。每套用新fixture进程，报告无JS错误。
- 1280×720六人结算/390px短额手机截图和16/32pxfavicon已人工查看。模拟手机和本地回环限制已记录，不声称真机/公网验收。
- 初轮Standards和Spec零发现；追加favicon审查Spec零发现，Standards仅票02状态名称（resolved应为done），已修复。
- 代码保持本地master；Render官方Manual Deploy步骤已核验并更新部署文档。旧云票不改完成状态，无推送/部署或猜测URL。
- 全功能最终审查范围与提交SHA：待本票最后复核补录。
