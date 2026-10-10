# 本局记录与筹码动效实施计划

Status: done
Type: plan
Spec: spec.md
Baseline: `7d375e357d31faf550b91ab61e393a63ebf3fcd0`

用户“一条龙用 $to-spec $to-tickets $implement 做完”授权自主确定常规粒度、依赖与沿用测试 seams。
一票一个可演示完整路径，串行 red→green，再以全范围基线完成双轴审查、全套检查和本地提交。

| 票据 | Blocked by | 交付 |
| --- | --- | --- |
| [01：完整公开过程与 C 展开](issues/01-current-hand-record.md) | None | 开局到派奖记录，传输到桌面／手机过程与结算 |
| [02：恢复、身份与回看](issues/02-record-continuity.md) | 01 | 生命周期事实、恢复与原身份快照、跟随和回看完整闭环 |
| [03：A 弧线筹码动效](issues/03-chip-motion.md) | 02 | 实际投入与派奖动效、去重、关闭／减少动态和手机遮挡处理 |

不修改旧云发布票状态，不部署；完成后给出提交及本地验收证据。

01–03均完成。实现提交 `34fd7b9`、手机跟随修正 `d010bcf`；最终回归与独立审查见
[验收证据](acceptance-evidence.md)及[审查报告](review.md)。
