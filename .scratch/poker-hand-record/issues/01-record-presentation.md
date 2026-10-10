# 本局记录的呈现方式

Status: open
Type: prototype
Mode: HITL
Blocked by: None
Spec: ../requirements-discussion.md

## Question

在现有桌面右栏与手机“本局”详情里，时间线、账本或阶段展开，哪种结构最便于理解
每位参赛者的每一步行动、公共牌及当时筹码变化？

## Agreed scope

[五项需求决定](../requirements-discussion.md)已确认。
用户明确授权本轮 `$prototype`。仅制作界面原型供审阅；正式实现及部署尚未开始。
保留 B2、紧凑六席及手机操作常驻。新过程可内部滚动；桌面结算继续完整一屏呈现六人。

## Primary source

- Baseline: `dcf4b37245f8d5590131da8e238b4e1cf7e83e0a`。
- Branch: `prototype/poker-hand-record`。
- Commit: `f39763bef774f1c0d16722436682c13bf2624989`
  (`prototype: compare current hand record interfaces`)。
- 原型入口及说明：该分支的 `internal/poker/web/PROTOTYPE-HAND-README.md`。
- 当前 worktree：`artifacts/poker-hand-record-prototype`。
- 在主工作区运行：

```powershell
python artifacts/poker-hand-record-prototype/internal/poker/web/prototype-hand-server.py
```

本机审阅：`http://127.0.0.1:18106/?variant=A`。可切换 `variant=B`、`variant=C`，
并选择手机或桌面 CSS 视口；运行期间 URL 有效。
没有此 worktree 时，从上述来源提交创建独立 worktree，再运行该分支 README 的一个命令。
原型代码留在独立分支，master 仅保留需求和上下文指针。

## Candidates

| 方案 | 结构 | 审阅重点 |
| --- | --- | --- |
| A · 逐步时间线 | 按顺序穿插阶段、公共牌与每一步行动 | 推荐；最贴合“刚才发生了什么”，长局需回看 |
| B · 下注账本 | 玩家/行动、底池、可用筹码纵向对齐 | 金额易核对，手机文字较密 |
| C · 按阶段展开 | 展开每个阶段查看完整步骤，当前阶段默认展开 | 长局易定位，早期步骤需要展开 |

## Validation

- Chrome 155 检查 3 个方案 × 5 个尺寸：390×720、360×640、320×568、1280×720、1920×1080。
  无页面横向溢出，操作可见，手机详情底部在操作区上方。
- 360×640 与 390×720 牌桌无需局部滚动；320×568 保留允许的 57px 局部滚动。
- 三方案回看时追加保持滚动位置，未读数为 1；回到最新后归零。标签切回保持跟随。
- 三方案手机结算包含六人；1280×720 桌面六行完整可见，结算区域内容/可用高度均为 464px，无内部滚动。
- 回退示例与牌桌金额同步；C 的回看开局展开开局阶段并定位顶部。选择控件内方向键不切换方案。
- 旧全押记录可用余额为 0，重入记录及当前钱包为 100；仍为全押。座位换人分开归属。
- 新局示例替换旧过程/结果，记录补给及逐人底注。恢复按钮只模拟已有公开过程重绘。
- 初次发现预览内方向键不能切换，已修正。检查途中页面加载等待超时，独立核查正常；
  加入具体导航完成条件后完整重跑通过，脚本异常为 0。失败没有计为通过。
- 原型 worktree 的 `scripts/verify.ps1 -Stage Harness` 与 `git diff --cached --check` 通过。
- 临时截图及浏览器检查结果位于 ignored `artifacts/hand-record-*.png`、
  `artifacts/hand-record-prototype-inspection.json`、`artifacts/hand-record-controls-inspection.json`、
  `artifacts/hand-record-settlement-inspection.json`。不是生产测试；未编写原型测试。
- 固定示例、内存状态，没有真实 HTTP/WebSocket 游戏写入、声音或 storage。
  不证明正式记录恢复、真实手机或公网功能已通过验收。

## Answer

原型制作已完成，推荐 A；用户尚未选择，决策保持 open。
待用户审阅后记录实际选择，再以已确认需求和选定呈现方式制作正式规格。

## Comments

2026-10-10：用户“做吧 $prototype”。本轮授权为原型审阅，不延用手机布局的旧实施授权。
