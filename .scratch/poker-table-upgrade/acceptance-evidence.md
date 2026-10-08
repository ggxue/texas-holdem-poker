# 牌桌升级验收证据（2026-10-09）

范围：[升级规格](spec.md) U1–U10、UAC01–28；适用未覆盖的首版规则与[内存决定](../online-poker/memory-reset-decision.md)。这是升级后的新证据，不以旧两真人、即时机器人或旧布局记录替代。01–06逐票已实现、验证、审查和本地提交；07补组合与归档。真实云票12独立保持blocked。

实现起始baseline：`bb30780165d285fb5f7529d6ac14f006cfc32851`。票07起始baseline：`af317c20ce24b04f8012de6d74fe403938e27c23`。审查待提交时用`git diff --cached <baseline>`覆盖工作，提交后用`git diff <baseline>...HEAD`复核范围。既有用户未提交文档保持原状。

## UAC回链

下表全部通过所列本地自动化／浏览器路径；实机和外部环境限制见末节。测试源码位于[internal/poker](../../internal/poker)，命名是可复现的`-Run`匹配项。

| UAC | 所属票 | 新证据 |
| --- | --- | --- |
| 01 | [01](issues/01-five-human-room.md) | app_test `TestUAC01FiveHumansEnterAndSixthHumanIsRejected`；capacity并发8身份争5席；浏览器五席与第六身份房满 |
| 02 | 01 | capacity `TestUAC02OneToFiveHumansReceivePrivateCardsAndPayOneAnte`：1–5人底池2–6、两张私牌、实际WS身份裁剪和摊牌 |
| 03 | 01、02、03、07 | capacity满席接管、action_start起点保持、bot_thinking原8秒；upgrade_integration思考中接管及旧页拒绝；浏览器接管设100、旧页停止重连 |
| 04 | 01、02、07 | capacity局中空席／座位重用，action_start待局不入候选，upgrade_integration思考中补位不能继承固定名单、暗牌、动作或奖项 |
| 05 | 01 | capacity两项房主先后测试；浏览器后来填真人1不越过原真人3 |
| 06 | [02](issues/02-random-start-and-odd-chips.md) | action_start `TestUAC06EachStreetUsesTheSameSelectedActionStart`，起点4后各轮4、5、bot、1、2、3 |
| 07 | 02 | `TestUAC07AllParticipantsIncludingBotCanBeSelected`，六候选、bot起点、空席与待局；离线浏览器bot起点实际等待 |
| 08 | 02 | `TestUAC08AllInAndDepartedStartRemainTheHandsStart`，起点全押／离房仍保存 |
| 09 | 02 | `TestUAC09EarlierCheckRespondsToBetInSelectedCycle`，早先过牌者补应答、不重复跟齐者 |
| 10 | 02 | `TestUAC10TakeoverKeepsStartAndNewHandDrawsAgain`，重连和新街不重抽，下一局独立选择 |
| 11 | 02 | `TestUAC11OddChipGoesToWinnerFirstInThisHandsCycle`：21池起点4，真人4得11、真人1得10 |
| 12 | 02 | `TestUAC12OddChipsSkipNonWinnerAndDepartedStart`：23池三赢家8／8／7，跳过非赢家／已离房起点 |
| 13 | [03](issues/03-bot-thinking-and-countdown.md) | `TestUAC13BotWaitsOneSecondBeforeCheck`：999ms不行动，1s后只Check一次 |
| 14 | 03 | `TestUAC14BotWaitsEightSecondsThenCallsActualBalance`：7999ms不扣款，8s Call10或Call4全押 |
| 15 | 03 | `TestUAC15And17NewCallOpportunityDrawsAgainAndHumanGetsThirtySeconds`：同轮Check1s后新Call3s，不复用首次时长 |
| 16 | 03 | `TestUAC16And18QueriesTakeoverAndOldClosePreserveThinking`：五身份查询、断线和接管保持原8s实际期限与30s显示依据 |
| 17 | 03、07 | 手动时钟新机会精确30s；3s离线浏览器观察正常30→28后动作、下一真人新30s；真实生产随机等待亦通过 |
| 18 | 03、07 | bot_thinking请求、接管、主动退出及同刻宽限；upgrade_integration五查询与到期并发，仍按身份裁剪与一次扣款 |
| 19 | 03 | `TestUAC19And20DepartureSettlesImmediatelyAndOldCallbacksCannotAffectNextHand`：最后真人退出立即结算，不等bot |
| 20 | 03 | 上项主动调用迟到旧唤醒及下一局；`TestUAC20GraceDeparturePrecedesThinkingAtSameInstant`同刻离房优先于Call |
| 21 | 03 | `TestUAC21AllInRobotAndRunoutDoNotCreateThinking`；8s短Call后立即补牌结算；原下注Runout回归 |
| 22 | [04](issues/04-responsive-six-seat-table.md) | browser `--layout`，1280桌面3／4／5上、1左2右bot下，观看者不旋转，空／满席和本人标记 |
| 23 | [05](issues/05-cards-and-hand-reference.md) | browser `--cards`，十类中英文及五示例牌、统一牌面／无数据牌背、公共五位0／3／4／5、底池在上与chips；逐个核对独立示例牌 |
| 24 | 04、05、07 | 新座位名称身份→手牌→余额→投入与状态；browser待局／空席／本人／当前／思考，公开组合测试补全全押、断线与弃牌状态输入 |
| 25 | 04、07 | 真实按钮合法Bet10／Call10、房主开局、顶部退出／重连；backend短Bet／Call金额公开契约、旧控制页禁用、房满与接管可见；--pending暂停真实HTTP验证顶部待确认及动作／退出禁用 |
| 26 | [06](issues/06-settlement-results.md) | browser `--results`：独立结果在操作下方、赢家收益、Best Five、历史余额不随接管设100改写、新局替换；提前胜出只显示本人两张允许手牌；六并列赢家各1 |
| 27 | 04、05、06、07 | Chrome155真实页面1280／390，空席／五满席四轮操作、参考展开／收起、五牌横排、结果纵向滚动、手动下一局，无横向溢出 |
| 28 | [07](issues/07-integrated-acceptance.md) | 新`TestUAC28FiveHumansShortAllInTakeoverSeatReuseAndNextHand`；容量真实HTTP／WS隐私、bot时序、旧命令及所有适用旧规则回归；全套race与两类浏览器整局 |

## 组合与原规则回归

组合测试固定起点真人4、3s机器人。五真人六底注后，真人4／5各付10，bot等待期间真人3退出，新身份填其座位；真人4接管设100但已投11及原bot期限不变。五身份查询与到期竞争后bot仅Call10，真人1余额4仅Call4全押。真人1随后重连设100，仍全押、原投入5及下一真人期限不变；真人2Call10。手算单池`6 + 4×10 + 4 = 50`，AA的真人1胜出得50、结算余额150，离房真人3投入1不退且无奖；补位者无本局私牌／动作／奖项。下一局六底注，真人1余额149、新成员99，名单使用新身份。

原AC01–06的十类牌型、点数优先、花色完整比较、A低顺子与最佳五张由cards测试保持。AC08–24的开局、阶段、四动作、固定与短额下注、单池、投入不退、全押争全池、仅零余额补给由game／betting／membership／refill测试保持。AC25–32、34、38的内存身份、重连设100、接管、幂等、旧局／旧机会／伪造钱包拒绝、HTTP／WS隐私、真人期限、断线宽限与同刻离房由app／takeover／socket_game／timers测试保持。旧AC07容量和立即bot／固定起点／旧零头排序已被升级对应UAC覆盖，不以旧语义计通过；已撤销持久化AC不恢复。

新时序案例用原始`gameCommand`回执及手动时钟。旧人类动作场景显式使用`settledCommand`推进公开观察到的bot机会，并保留原始命令函数；生产与离线等待都没有0秒变体。生产等概率分别由crypto/rand的无偏区间抽样保证，经双轴审查；没有用偶然统计通过代替策略审阅。

## 浏览器步骤与复现

Chrome155独立配置与浏览器上下文；生产服务`localhost:18081`，离线固定牌序服务仅`127.0.0.1:18082`。离线程序复用公开Options，不被生产main引用，也不添加HTTP／WS选牌、起点、时长或改筹码入口。

从仓库根目录先按[开发说明](../../docs/development.md)准备Go、race、Node和独立调试Chrome9228。启动生产构建（另一个终端保持服务）：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1 -Race
$env:PORT = '18081'
& artifacts/build/texas-poker.exe
```

运行浏览器；离线程序使用同一手工环境设置后的本地SDK，也可按开发说明运行已配置的Go：

```powershell
& 'C:/Program Files/nodejs/node.exe' scripts/browser-smoke.mjs http://localhost:18081 --layout --cards --results --full-hand --pending
$env:GOCACHE = Join-Path $PWD '.tools/gocache'
$env:GOPATH = Join-Path $PWD '.tools/gopath'
$env:GOTOOLCHAIN = 'local'
& .tools/go1.27.1/go/bin/go.exe run ./scripts/browser-fixture
```

另一个终端：

```powershell
& 'C:/Program Files/nodejs/node.exe' scripts/browser-smoke.mjs http://127.0.0.1:18082 --layout --cards --results --full-hand --tie-six --pending
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-go.ps1 -Package ./internal/poker -Run '^TestUAC28'
& 'C:/Program Files/nodejs/node.exe' scripts/restart-smoke.mjs
```

浏览器步骤：一真人＋bot四轮；第二身份局中等待；下一局两真人Bet／Call并摊牌；五席占满、第六拒绝；同Cookie新页接管且旧页失权；五真人＋bot完整四轮及六人结果；手动下一局五真人弃牌提前胜出；退出、补低号座位、按入房先后移交房主。手机展开十类参考再收起，两个宽度均检查区域关系及无横向溢出。离线额外确认30→28正常递减和共享同花大顺六底注各得1；--pending仅暂停并恢复真实HTTP，验证待确认文字与动作／退出禁用，没有伪造响应。

报告为忽略的`artifacts/browser-report.json`和`browser-tie-report.json`；均退出0、passed=true、errors=[]。截图已目视复核：生产`poker-desktop.png`／`poker-mobile.png`、`poker-full-results-desktop.png`／`mobile.png`、capacity与reference；离线使用独立`poker-tie-`前缀，包括六并列结果与`poker-tie-thinking-elapsed.png`的28秒显示，避免相互覆盖。可重跑生成，不把旧图当新验收。仅清理本次自有进程与上下文，未操作日常浏览器。

## 检查、失败和限制

- 各票RED／GREEN、baseline、双轴审查和本地提交见01–06票据。07新增牌局组合测试针对已交付实现首跑GREEN；最终补查UAC25时，暂停真实HTTP请求发现有合法按钮的提交状态只禁用控件、缺待确认文字，按目标行为RED后补顶部独立提示，再GREEN整局。没有改变后端规则；不伪造已实现组合测试的RED。
- `verify.ps1 -Race`包含普通全套Go测试与构建再运行race，harness、格式、vet、普通测试、构建及race均通过；fixture程序亦编译。缺工具没有略过。票06与07的新fixture／组合源码变更后有对应重新构建与检查。
- 已修复并留证的检查失败：01旧两席fixture空位索引；03旧验收仍假设bot即时；04旧布局位置；05旧页面缺参考／五牌位、真实消息生成时间需100ms容差；06旧页面无独立结果区，平局初版误假设历史余额至少100。具体原因与复验见对应票。未为这些测试偏差改变已约定规则。
- 原生PowerShell Start-Process遇当前执行环境的PATH／Path重复键，使用本任务ignored Node启动脚本规范环境键并记录PID；Chrome沙箱启动无法就绪后，以已获工具权限启动专用实例，随后全部浏览器路径实际执行通过。未把失败启动计通过。
- code-review最终双轴审查与提交提示修复复审均无未关闭发现：Standards 0、Spec 0；覆盖票07差异并复核自实施baseline以来的完整升级实现。初次最终审查未发现提交文字遗漏，随后主动暂停请求补查复现并修正，两轴对追加差异复审通过。
- 手机390为模拟，真实手机和其他浏览器仍待验证；真实云HTTPS／WSS、休眠与重部署未验证，票12保持blocked。本批未创建云资源、未部署。
- 只声明实际执行的路径及race检查通过，不宣称穷尽所有并发交错；内存重启清空与重连设100继续适用，不增加恢复保证。
