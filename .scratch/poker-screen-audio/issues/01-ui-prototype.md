# UI原型：整屏牌桌与中文播报

Status: open
Type: prototype
Mode: HITL
Blocked by: None
Spec: [已确认需求讨论](../requirements-discussion.md)
Prototype status: ready-for-feedback
Implementation baseline: 3eec8843fec6d6e69d05f07e54e7b6977408e532

## Question

三种布局中，哪一种能在1280×720可用视口下，完整、清楚地展示六席牌桌、
十类牌型参考、合法操作及满员结算？固定普通话女声是否顺耳？

## Primary source

- 独立分支：`prototype/poker-screen-audio`，不进入master业务代码。
- 原型提交：`cc55bf4`（全部三方案、启动脚本与62条固定普通话录音）。
- 本地工作树：`artifacts/poker-screen-audio-prototype/`。
- 文件：分支内 `internal/poker/web/prototype-screen.js`、`prototype-screen.css`、
  `prototype-audio/*.wav` 及同目录 `PROTOTYPE-SCREEN-README.md`。
- 单命令启动（从主仓库根目录）：

  ```powershell
  python artifacts/poker-screen-audio-prototype/scripts/start-screen-audio-prototype.py
  ```

- 预览：`http://localhost:18081/?variant=A`，B/C采用相同路径和对应参数。
- A：底部结算表；B：右侧结算栏；C：底部六人结算卡片。
- 六种场景及真人1房主/真人2视角，可从底部切换；方向键可切方案。
- 中文试听、连播被本人提醒打断、声音开关及音量可操作。

## Boundaries

- 可丢弃的UI原型，只在回环地址启用；未修改主工作树的正式前后端业务代码。
- 不连接真实HTTP写操作或WebSocket，按钮切换预设视图，不实现完整扑克状态机。
- 仅内存状态；正式应用的声音偏好持久化在此仅展示控件与切方案时保留状态。
- 机器人思考为预设场景，不是已实现的实时调度；锁屏/后台提醒不作为此原型验收承诺。
- 录音已生成，实际听感待用户试听；自动媒体播放观测不等同人耳音质验收。

## Validation

- Headless Chrome 155，独立开发预览配置，不使用用户浏览器配置。
- 1280×720及1920×900：三方案文档宽高均等于视口，无页面滚动，
  各有10条参考、6座位及6人结果；关键区域无内部溢出。
- 390×844模拟手机：无页面横向溢出，参考入口、两列真人座位、机器人、操作与结果可纵向查看。
- 本地WAV可加载并播放，优先演示已切换到提醒录音且播放时间推进，静音后暂停；
  方案切换更新URL，房主视角显示开局按钮，手机参考展开后仍为390宽。
- `verify.ps1 -Stage Harness`、两工作树 `git diff --check` 及原型JS语法检查通过。
- 截图及尺寸观测：主工作树忽略目录 `artifacts/poker-screen-{A,B,C}-{1280,1920,390}.png`
  和 `artifacts/poker-screen-audio-measurements.json`。
- 真机、其他浏览器及女声音质待用户反馈；本票不是正式游戏或云部署验收。

## Answer

用户明确选择 **B · 对局工作台**。该布局选择已定，不能再按代理原先推荐的C推进。
用户随后提出称呼改为“玩家”、在线玩家面前增加筹码图片、深红色老钱赌场主题及
关键道具质感。见[本轮细化访谈](../visual-refinement-discussion.md)，随后更新原型。
第一版原型提交cc55bf4保留为原始对比依据；本次尚未授权正式游戏实施。
用户尚未明确评价女声实际听感，不能把UI选择当作声音质量验收。
