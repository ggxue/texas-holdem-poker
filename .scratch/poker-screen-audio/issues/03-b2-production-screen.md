# 03：B2真实牌桌与筹码

Status: done
Type: implementation
Blocked by: None
Spec: [生产规格](../spec.md)
Implementation baseline: `3eec8843fec6d6e69d05f07e54e7b6977408e532`

## What to build

用真实服务端视图替代原型快照，完整玩局、操作及结算，采用用户选定B2深红桌沿席位。

## Acceptance criteria

- [x] 1280×720/1920×900全部参考、六席、公共区、合法操作、连接/提交提示及六人结算一屏。
- [x] 每个占用席近桌心有余额堆；0/空席隐藏，待局/弃牌保留，断线弱化；底池独立。
- [x] 玩家称呼统一，金色本人/象牙行动者同时可辨，真实合法扣款与不足全押，隐私不变。
- [x] 满员/提前胜出/离房/座位重用/重连设100保留全押及结算快照正确呈现。
- [x] 手机纵向、参考可展开、无横向溢出；14px/12px字号下限，无原型入口。

## Public test boundaries

用户已确认真实页面与HTTP/WebSocket。沿用离线Clock/牌序/起点构造，公开操作及DOM验证。

## Validation

基线及实现后的 `scripts/verify.ps1 -Race` 均通过（harness、格式、vet、tests、build、race）。JS语法检查与 `git diff --check` 通过。

Red：旧界面在1280×720需要滚动；满员结算席位内部溢出；手机钱包筹码覆盖席位。公开DOM断言分别实际失败后修正布局，未改变游戏规则。

Green：2026-10-10真实Chrome155，通过
`node scripts/browser-smoke.mjs http://localhost:18082 --casino --layout --cards --results --pending --full-hand --tie-six`。
验证1280×720、1920×720、1920×900桌面一屏与390px手机；六人并列赢家、四轮公共牌、手牌隐私、待局、房间满员、HTTP待确认、机器人倒计时、接管设100与历史结算、退出/座位复用/房主交接。手机筹码位于所属席位下方并居中。
离线固定牌序仅在测试构造器中使用，无在线测试入口。

证据（ignored）：`artifacts/browser-tie-report.json`，`artifacts/poker-tie-full-results-desktop.png`、`artifacts/poker-tie-full-results-mobile.png`。报告13组场景，passed:true，浏览器异常0；两张最终截图人工查看完成。

Standards与Spec独立审查：各0项未解决发现；宽屏720px高度疑点已用真实浏览器排除。中文播报属于04；本票顶栏保留禁用的声音入口。实际云验收属于05。
