# 01：完整公开过程与 C 阶段展开

Status: done
Type: implementation
Blocked by: None
Spec: ../spec.md
Baseline: `7d375e357d31faf550b91ab61e393a63ebf3fcd0`

## What to build

玩家从桌面右栏或手机“本局”看到当前局开局、底注、发牌、各阶段、全部行动与派奖的完整过程。
采用 C 阶段展开、过程／结算标签，精确快照与公共牌；不改变游戏规则与声音。

## Acceptance criteria

- [x] HRC01–HRC05：公开过程完整、顺序／标识／时间稳定，实际金额精确、拒绝／重试／回滚无新增记录。
- [x] 实时 HTTP／WS 输出和查询均含当前公开过程；无私人手牌或未来公共牌泄漏。
- [x] C 章节、当前阶段默认展开、默认过程和完整结算在桌面／手机可使用。
- [x] 新过程内部滚动；满员桌面结算一屏六人，手机常驻操作与紧凑六席保持。

## Public test boundaries

既有 HTTP／WebSocket 游戏视图与真实浏览器页面。固定输入仅离线构造，预期来自规格手算。

## Validation

公开 HTTP 测试首次因记录为空失败，实现后通过；浏览器首次因过程标签缺失失败，接入后通过。完整六人结算检查发现3px、随后1px溢出，调整冗余标题及行间距后五尺寸重跑通过。

通过：test-go.ps1 -Package ./internal/poker -Run ^TestHandRecord；browser-hand-record.mjs。截图与几何：artifacts/hand-record-production-*.png 和 hand-record-production-layouts.json（ignored）。完整回归、race与双轴审查在03票末尾记录。
