# 筹码投入与派奖的动效呈现

Status: open
Type: prototype
Mode: HITL
Blocked by: None
Spec: ../requirements-discussion.md

## Question

在深红 B2 牌桌与紧凑六席手机布局中，哪种筹码流向编排既刺激又便于追踪实际投入和派奖？
本局记录固定沿用已选 C · 按阶段展开；本问题的 A/B/C 只比较动画。

## Agreed scope

用户确认[需求讨论](../requirements-discussion.md)七项全部按推荐，明确要求更新 `$prototype`。
底注、下注、跟注、主动全押与派奖采用实际确认金额；数字和操作立即更新。
单赢家收池、平局按真实奖项分束、短队列压缩、无历史补播、手机详情遮挡、
关闭动效及减少动态、无新增音效均为已确认范围。真实游戏实施与部署尚未开始。

## Primary source

- Main context baseline: `f6a3bfa77c099a826b4cb98584a08bb230920fab`。
- 原型继承本局记录来源：`f39763bef774f1c0d16722436682c13bf2624989`。
- Branch: `prototype/poker-chip-motion`。
- Commit: `ad0b4c7ff35a3df8456a43c806150799c0bfd845`
  (`prototype: compare chip transfer animations`)。
- 入口与说明：该分支 `internal/poker/web/PROTOTYPE-CHIP-MOTION-README.md`。
- Worktree: `artifacts/poker-chip-motion-prototype`。
- 主工作区运行一个命令：

```powershell
python artifacts/poker-chip-motion-prototype/internal/poker/web/prototype-hand-server.py
```

本机入口：`http://127.0.0.1:18108/?variant=A`；改为 B/C 或用底部箭头切换。
可选择手机 390×720、360×640、320×568、桌面 1280×720、1920×1080。
原始 C 记录原型仍独立保留在 `prototype/poker-hand-record`，端口 18106。
没有本 worktree 时，从上述固定提交创建独立 worktree，再运行 README 的一个命令。

## Candidates

| 方案 | 结构区别 | 审阅重点 |
| --- | --- | --- |
| A · 弧线抛注 | 个体筹码沿弧线错峰抛出，派奖先聚拢再成束返回 | 推荐，清楚、力度适中 |
| B · 整堆推送 | 保留整叠沿桌面推送，蓄力、短促落点回弹 | 更稳、更有整堆重量感 |
| C · 汇聚爆发 | 先扇形展开再汇聚，派奖以放射流束爆发 | 更刺激，同时飞行视觉密度更高 |

## Validation

- 原生 Chrome 155 检查 3 方案 × 5 尺寸。页面无横向或外部纵向溢出；操作区始终可见。
  各方案记录结构均为 C，手机默认关闭详情，打开详情后仍在操作区上方。
- 每组检查六席底注、普通下注、全押、600 收池、204/203 平局、密集投入→派奖。
  数字先更新；全押发射点在余额 0 时仍可定位。手机减少筹码数量，飞行不拦截点击。
- 600 示例最终底池 0、赢家余额 600，总量 600；407 示例最终底池 0，赢家 204/203、
  弃牌者余 99，总量 506。固定示例余额无负数；私牌与结算按原型既有可见性展示。
- 同批两项平局奖项在一次追加中同时更新钱包和飞行；重复事件不重复生成动画。
  新局清理并替换过程；模拟恢复与回看不自动补播旧动效。
- 关闭动效无飞行但金额更新；减少动态无筹码飞行，只展示静态 +金额。
  手机详情打开时跳过飞行、保留详情与记录；窗口变化清理动画，自然结束后节点与计时器为 0。
- 底部点击与预览内方向键都能切换并更新 URL；选择控件方向键不切换。
  原型状态侧栏展示完整金额、阶段与动效状态；预览不被底部审阅栏遮挡。
- 15 组布局与场景检查、交互检查最终均无脚本异常。期间检查脚本自身曾引用未定义变量，
  以及向 Document 而非真实按键目标发送合成键事件；修正检查脚本后重跑通过，失败未计为通过。
- `node --check` 三个变更脚本、`git diff --check`、原型 worktree 的
  `scripts/verify.ps1 -Stage Harness` 与提交前 staged diff 检查通过；commit-msg hook 启用，
  已验证提交标题有类型前缀。
- 临时截图及浏览器检查记录：ignored `artifacts/chip-motion-*.png`、
  `artifacts/chip-motion-inspection.json`、`artifacts/chip-motion-controls-inspection.json`。
  没有编写原型测试，没有真实游戏 HTTP/WebSocket 写入、声音、偏好持久化。
- 此证据只支持原型审阅，不证明真实手机帧率、功耗、Safari、真实身份／事件恢复或公网验收。
  正式代码应按后续规格重写；本局记录与动效均未进入正式游戏。

## Answer

待用户比较动效 A/B/C。C · 按阶段展开的记录结构已确认，不是本问题待选动画的答案。

## Comments

2026-10-10：用户“可以，全部按推荐，加下来更新 $prototype 给我”。
七项范围已确认；动画原型授权明确，正式实施与部署仍未开始。
