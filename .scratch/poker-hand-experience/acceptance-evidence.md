# 本局记录 C 与筹码动效 A：本地验收

Status: in-progress
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
  标签往返、回到最新和下一局重置。
- `browser-chip-motion.mjs`：真实底注、全押99、0余额发射，六人各得100，
  HTTP／WS去重、手机详情跳过飞行、关闭偏好刷新恢复、系统减少动态和自然清理。
- `verify.ps1 -Race`：Harness、固定SDK、gofmt、vet、全套测试、构建与race全部通过。

## TDD 与证据

第一票公开传输测试首先因记录缺失失败，呈现首先因过程标签缺失失败；
第二票首先因恢复／离房事实缺失失败；第三票首先因正式动效开关缺失失败。
这些失败均先于相应实现。后续新增边界回归不假称曾先失败。

可重复入口为仓库中的 Go 测试与上述浏览器脚本。浏览器使用固定牌序的离线 fixture、
独立身份上下文和实际 Chromium，页面请求经过真实HTTP／WS。
通过浏览器原生 animate 的透明观察适配器记录几何帧和真实精确金额，不替代动画执行。
离线控制输入不进入生产在线接口。

ignored 原始输出：`artifacts/hand-record-production-*.png`、
`hand-record-production-layouts.json`、`hand-continuity-browser.json`、
`chip-motion-production-browser.json`、`chip-motion-production-allin.png`。

## 仍在完成

后台恢复／普通投入／单赢家与存储受限浏览器组合，既有手机／全押／声音回归，
Standards与Spec独立审查及最终本地提交。

## 适用范围

本地真实浏览器的设备尺寸模拟不等于实际手机性能验证；未执行公网或Render部署验收。
旧云票12不改状态。筹码仍只在内存，成功入房设100，进程重启清空。
架构报告候选仍独立等待用户选择，不在本轮功能内实施。
