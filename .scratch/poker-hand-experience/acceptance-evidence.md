# 本局记录 C 与筹码动效 A：本地验收

Status: done
Baseline: `7d375e357d31faf550b91ab61e393a63ebf3fcd0`
Spec: [spec.md](spec.md)

## 已通过

- Go 公开 HTTP／WebSocket 记录测试：完整顺序、稳定事件身份与服务端时间、实际金额、
  查询／重试／拒绝／回滚、公开公共牌、离房资格、原身份与同席新人、超时及机器人原因、
  直接连接恢复、重新入房100、补给、新局失败／成功与重启清空。
- 单赢家提前结束：机器人收底池3，可用102，没有摊牌记录。
  合法不等额全押形成底池7、三人同花大顺：奖项3／2／2、剩余池4／2／0，零头顺序准确。
- `browser-hand-record.mjs`：真实五真人加机器人，C过程与六人结算，
  1280×720、1920×1080、390×720、360×640、320×568，操作区与详情几何。
  完整六人桌面结算首次溢出3px／1px，修正后无内部滚动。
- `browser-hand-continuity.mjs`：实际重连与同席新人，原快照、回看位置、未读、
  标签往返、回到最新和下一局重置；手机首次打开跟随最新、关闭再打开保留回看，
  标签往返后手动收起章节在新记录／实际重连后保持。
- `browser-chip-motion.mjs`：真实底注、全押99、0余额发射，六人各得100，
  HTTP／WS去重、手机详情跳过飞行、关闭偏好刷新恢复、系统减少动态和自然清理。
- `browser-chip-edges.mjs`：实际下注／跟注10、A弧线抬高中点、旋转尺寸清理、
  实际重新入房100不追播、不改下注余额89；原生冻结／恢复且重新可见后无缓冲动作追播，
  当前新派奖仍正常飞行；存储get／set受限不阻塞、单赢家收完整56，过牌／弃牌／重设无飞行。
- `browser-chip-odd-payout.mjs`：五真人加机器人，通过合法下注、四人弃牌、实际重连100、
  再主动全押100和机器人不足额跟89形成单底池215。机器人按既有起点获108、本人107；
  确认奖项与可见+金额各一份、剩余池107／0；原生飞行时序证明末次跟89先落池再出奖项束。
- 既有 `browser-mobile.mjs`、`browser-allin.mjs`、`browser-smoke.mjs`、
  `browser-voice.mjs`、`browser-voice-restart.mjs` 全部通过。
  smoke使用 --casino --layout --cards --results --chips --tie-six；
  回归涵盖手机多尺寸、B2、六席、牌型、满员、弃牌隐私、房主转移／接管、全押、
  中文实际媒体播放及真实进程重启后新身份／空房／声音恢复。
- `verify.ps1 -Race`：Harness、固定SDK、gofmt、vet、全套测试、构建与race全部通过。

## TDD 与证据

第一票公开传输测试首先因记录缺失失败，呈现首先因过程标签缺失失败；
第二票首先因恢复／离房事实缺失失败；第三票首先因正式动效开关缺失失败。
这些失败均先于相应实现。后续新增边界回归不假称曾先失败。
收尾新增的手机首次打开跟随最新断言先失败，修正隐藏状态的位置保存／恢复后通过。
同一修正补齐章节手动状态在标签／尺寸变动后保留，初始化展开不冒充手动选择。
修正后再次通过 continuity、五尺寸完整结果及动效／手机详情浏览器检查。

可重复入口为仓库中的 Go 测试与上述浏览器脚本。浏览器使用固定牌序的离线 fixture、
独立身份上下文和实际 Chromium，页面请求经过真实HTTP／WS。
通过浏览器原生 animate 的透明观察适配器记录几何帧和真实精确金额，不替代动画执行。
离线控制输入不进入生产在线接口。

浏览器适配器修正：数字席位属性选择器需带引号；减少动态断言需清掉上一局的观察样本；
冻结后仅恢复生命周期仍是隐藏页，增加原生焦点模拟并断言真正可见后继续新动作。
平局215的独立预期修正为按既有起点先派机器人108、再派本人107；产品派奖算法没有修改。
这些适配器失败与初始功能缺失的TDD red分开记录，不冒充产品缺陷或新增功能的先失败证据。

ignored 原始输出：`artifacts/hand-record-production-*.png`、
`hand-record-production-layouts.json`、`hand-continuity-browser.json`、
`chip-motion-production-browser.json`、`chip-motion-production-allin.png`、
`chip-motion-production-edges.json`、`chip-motion-production-odd-payout.json`；
旧回归另有 `mobile-browser-report.json`、`allin-browser-report.json`、
`browser-voice-report.json`、`browser-voice-restart-report.json`。

运行准备：bootstrap-go.ps1，固定SDK构建 scripts/browser-fixture 的回环服务。
各独立浏览器场景使用新 fixture 进程，POKER_CDP_URL 指向自有 Chromium CDP；
browser-voice-restart.mjs 自己拥有并重启该验收进程。未操作云环境或无关用户进程。

## 审查与提交

实现提交：`34fd7b9a8320e62e1a86211ffe040619357119e1`。
Standards与Spec两个独立子代理按固定基线审查该完整实现范围，
标准违规0、规格问题0；1个P3转发入口判断项保留，理由及两轴原文见[审查报告](review.md)。
手机跟随修正提交：`d010bcfe8fa20c57c091bceb6ed0863f3dd538f1`。
两个独立子代理对 `34fd7b9...d010bcf` 复审，标准违规／新增气味／规格问题均为0。
此后差异仅为浏览器适配器／回归检查与完成记录，正式产品代码未再变更。

## 适用范围

本地真实浏览器的设备尺寸模拟不等于实际手机性能验证；未执行公网或Render部署验收。
旧云票12不改状态。筹码仍只在内存，成功入房设100，进程重启清空。
架构报告候选仍独立等待用户选择，不在本轮功能内实施。
