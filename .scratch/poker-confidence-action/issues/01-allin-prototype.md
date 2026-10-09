# 01：主动全押玩法原型审阅

Status: resolved
Type: prototype
Mode: HITL
Blocked by: None
Spec: 尚未发布；规则依据[讨论Q3–Q7](../requirements-discussion.md)

## Question

主动全押、重新应答、小额争全池及唯一可行动者边界，实际点击后的行为是否符合用户预期？按钮金额、跟注出现时机及结果是否看得懂？

## Authorized scope

用户确认Q3–Q7，并要求今天审阅prototype。仅创建独立throwaway演示，不修改正式游戏或部署。

## Primary source

Branch: `prototype/poker-allin`
Baseline: `0d71575da9400ff48688d7152198d321da36019d`
File: `internal/poker/web/prototype-allin.html`
Local worktree: `artifacts/poker-allin-prototype`
Commit: `69a96da`（初版`62622bd`）
Review copy: `artifacts/prototype-allin.html`（单个HTML，双击即可运行）

## Validation

2026-10-10：在独立隐藏Chrome中实际打开file页面，点击8个案例全部引导步骤；另做自由弃牌、模拟超时和390px页面操作。无浏览器JS异常。

手算与观察：满员全押池600；已有下注后的全押与差额89形成池300；20→50→80及机器人跟80形成池234，机器人余20；小额全押5获池48；唯一可行动者尝试押99被拒，池23/余额99不变，再跟10得池33；短额跟5保持目标10，最终池28；重连设100保留全押，获48后余额148；六人平局池111分19/19/19/18/18/18。

1280×720页面和面板不滚动、六人结算及播报文案完整可见；390×844模拟手机纵向布局，无横向溢出。已人工查看初始桌面、六人平局、手机截图；首轮发现结算与播报面板重叠，已调整区域并重新截图核对。证据在ignored `artifacts/allin-qa/observations.json`、`layout-observations.json`及PNG。

原型按skill不增加业务测试；验证是浏览器演示核对。牌力/赢家预设，行动起点固定玩家1；没有真实随机牌局、服务端原子提交/并发、HTTP/WS、30秒墙钟、真实声音或云部署。不能以原型结果代替正式验收。当前显示同金额的跟注与全押按钮，是否合并等待用户审阅。

`git diff --check`与`verify.ps1 -Stage Harness`通过。HTML仅提交到独立原型分支，不合并master、不推送、不部署。

按钮/提示修订`69a96da`：动作按钮仍仅专业名称和实际金额；补充过牌/弃牌悬停提示、短额跟注后的全押状态辅助说明。审阅副本已同步；JS语法检查、diff与Harness通过。该次仅修改提示文本，未改变状态模型。

## Answer

用户已确认“这套全押逻辑是对的”，并在解释过牌/弃牌后明确“prototype html我审算通过了”，授权自动规划、分票和正式实施。保留原型中的合法同金额跟注/全押两个动作；按钮仅动作名和实际金额，无“（不足全押）”。原型一手源保持独立分支；正式规格见上级spec.md，不将HTML壳合入master。
