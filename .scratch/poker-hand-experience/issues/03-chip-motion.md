# 03：A 弧线抛注与整池派奖

Status: in-progress
Type: implementation
Blocked by: 02-record-continuity.md
Spec: ../spec.md
Baseline: `7d375e357d31faf550b91ab61e393a63ebf3fcd0`

## What to build

全部参与者的真实投入以实体筹码弧线错峰入池，确认派奖后整池聚拢再按实际奖项成束飞向赢家。
数字与操作即时更新，手机减量、详情不被遮挡，可关闭或使用减少动态。

## Acceptance criteria

- [ ] CMA01–CMA02：底注、下注、跟注、主动全押、单赢家及平局实际奖项准确呈现，金额即时更新。
- [ ] CMA03：HTTP／WS去重，旧快照及恢复／回看不追播；身份无法定位不落到新人。
- [ ] CMA04：连续动作不积长队、最后投入先入池，手机减量、不挡操作、详情保持、布局变化清理。
- [ ] CMA05：默认开启与偏好保存、存储受限不阻塞、关闭正常、系统减少动态静态提示。
- [ ] 原中文播报／提醒、完整结果、规则、并发和内存恢复回归；记录准确，不新增音效。

## Public test boundaries

真实 HTTP／WS确认的浏览器页面，检查可见流向、实际数字、按钮、详情、偏好与恢复。
既有离线固定牌序与时钟控制同批派奖案例，无在线管理入口。

## Validation

待 red→green、双轴审查、全套 Harness／Go／race 与浏览器证据。
