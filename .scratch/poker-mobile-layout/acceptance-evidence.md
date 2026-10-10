# 手机紧凑六席：本地实施验收

Date: 2026-10-10
Status: local acceptance passed
Spec: [手机规格](spec.md)
Ticket: [02：手机紧凑六席与常驻行动](issues/02-compact-mobile-table.md)
Implementation baseline: `9b0bba68d0fbbfeb30adbc12ce5dfe15b6b870cf`
Implementation commit: `e4e437f40b0c879c1970d20e968151f91eef0725`

## 实施结果

真实页面采用A的手机紧凑六席，行动区常驻底部；玩家、结果、更多功能使用按需详情。
详情移动原有声音/重连/退出控件与结算节点，保留监听及实时更新，不生成第二份控制状态。
玩家详情只复制已经按身份裁剪的席位内容。跨手机/桌面断点后恢复原有面板位置。

新增手机呈现模块只负责详情与布局测量；后端仅增加固定静态模块资源入口，
没有改动游戏规则、App状态提交、HTTP/WebSocket契约、声音模块或计时器。
原型工具没有进入正式资源；架构候选未实施。所有提交均使用类型前缀并经hook校验。

## Red → green

1. 先在未修改的真实页面创建五位独立玩家，开始六人牌局并在玩家1行动时检查360×640。
   浏览器测试按预期失败：`current action or buttons outside viewport`；操作区顶部约1284px，
   超出640px视口。保存原页面截图与几何报告，确认失败来自目标行为。
2. 接入紧凑六席及常驻行动区，第一项360×640公开浏览器检查通过。
3. 增加“更多功能可访问”的下一项测试，按预期失败：`mobile more controls entry missing`。
4. 接入详情与原控件归位逻辑，检查通过；扩展真实提交、结算、重入、声音和断线场景。

这轮实际应用了tdd的公开边界red→green和codebase-design的小接口原则，
没有给私有布局函数增加单测或在线余额/牌序控制入口。

## 可复现检查

从仓库根目录运行。固定SDK、Go缓存及完整校验沿用仓库脚本：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-go.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-go.ps1 -Package ./internal/poker -Run 'TestAAC|TestAC25|TestUAC01'
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1 -Race
```

结果：SDK已存在且版本正确；定向测试通过；Harness校验16个固定技能及资源通过；
Go格式、vet、全套测试、生产构建与race全通过。JavaScript语法检查和git diff空白检查通过。

浏览器为独立Chrome155配置，CDP监听本机9440；服务为既有离线牌序构造的真实Go
浏览器验收程序，绑定127.0.0.1:18082。它与生产页面使用相同静态资源和真实HTTP/WebSocket，
固定牌序与机器人三秒等待只经既有公开离线Options构造，不提供在线测试命令。

先用固定SDK构建并在独立终端运行该验收程序，另启动带独立user-data-dir和
remote-debugging-port=9440的Chrome。各套浏览器测试前重启该程序以获得全新房间。

```powershell
# 沿用SDK与缓存环境，构建已有离线验收程序；在另一终端运行生成的exe。
& .tools/go1.27.1/go/bin/go.exe build -o artifacts/build/mobile-fixture-next.exe ./scripts/browser-fixture
$env:POKER_CDP_URL = 'http://127.0.0.1:9440'
node scripts/browser-mobile.mjs http://localhost:18082
# 重启离线验收程序后：
node scripts/browser-allin.mjs http://localhost:18082 --icon
# 重启离线验收程序后：
node scripts/browser-smoke.mjs http://localhost:18082 --casino --layout --cards --results --pending --full-hand --tie-six
# 重启离线验收程序后：
node scripts/browser-voice.mjs http://localhost:18082 --casino
```

## 浏览器结果与案例追踪

| 案例 | 实际验证 | 结果 |
| --- | --- | --- |
| MAC01–02 | 360×640、390×720、412×780满员本人行动；320×568矮屏 | 常见尺寸六席/公共牌/底池/手牌/操作首屏可见；矮屏牌桌局部滚动，行动可见；无外层页面滚动或横向溢出 |
| MAC03 | 玩家1和玩家4视角 | 席位空间关系固定，本人标记准确 |
| MAC04–05 | 真实按钮命中测试，暂停实际HTTP全押请求再确认 | 99金额准确；待确认提示可见且按钮禁用；开放详情时操作仍能点击；六人池600每人得100 |
| MAC06 | 玩家、更多、参考开关；真实状态/期限前后查询 | 六席详情及10类参考完整；无游戏版本变化或期限重置；关闭恢复原视图 |
| MAC07 | 六人真实结算与公开最佳五张 | 六人结果与30张最佳五张可查看；详情滚动，操作区不被遮挡 |
| MAC08 | 空席、机器人思考、实际断线；原有完整牌局回归 | 状态、筹码、席位身份正确；完整四轮牌局正常，早胜不强制亮牌 |
| MAC09 | 用真实下注/弃牌形成差额，跟注98后刷新 | 原目标100及短额全押保持；重入钱包100，仍无新全押资格 |
| MAC10 | 手机真实音频播放、音量偏好35记忆、控制页接管、从更多入口实际退出 | 本地WAV确实播放；原控件及偏好保持；旧控制页不能操作；退出后可重新入房 |
| MAC11 | 手机→桌面→手机、1280×720与更宽桌面；原有B2完整回归 | 原参考、控件和结果归位，六人结算/筹码/底池无滚动或遮挡；桌面布局保持 |
| MAC12 | 实际页面、真实HTTP/WebSocket流程及静态图标 | 无原型工具；真实对局通过；本地黑桃图标正常 |

新增手机测试记录15个几何场景，全部行动区及按钮可见；320×568行动中的牌桌需要局部滚动，
符合已确认取舍。原有全押套件、完整牌局/隐私/计时/卡片套件和真实媒体套件均通过，
浏览器未捕获JS异常。媒体套件覆盖默认音量、静音、优先打断、10秒提醒、HTTP/WS恢复竞态、
媒体失败后继续游戏、当前提醒恢复及重复接管故障不截断播放。

报告和截图置于ignored artifacts，可由上述命令重新生成：
`mobile-browser-report.json`、`mobile-browser-red.json`、`mobile-details-red.json`、
`mobile-before-360x640.png`、`mobile-own-360x640.png`、`mobile-more-360x640.png`、
`mobile-results-390x720.png`、`mobile-desktop-1280x720.png`及其余场景图。
已实际查看修改前、核心牌桌、更多、六人结果的PNG，结合既有桌面回归核对。

## Standards

独立审查基线9b0bba6至e4e437f：**0项发现**，未发现违反记录标准或需修复的具体代码气味。

手机模块集中详情、原位恢复和尺寸测量，调用方仅创建模块并refresh；玩家详情读取已裁剪
席位，没有增加合法动作或客户端金额计算。原有监听、金额和控制权限保留。
详情是非模态区域，固定操作可继续使用；关闭/Escape恢复触发按钮焦点。
后端状态提交、传输、计时、存储和并发模型未变。

## Spec

独立审查同一基线与手机规格：**0项确证发现**，未发现缺失要求、范围扩张或实现错误。

原控件/结果节点保留事件与实时更新；玩家详情不新增暗牌来源；开关详情不发命令或重置期限；
700px断点归位，既有牌型参考监听恢复桌面展开。改动限定手机呈现，没有夹带架构重构、
原型工具或部署。审查曾提出参考折叠疑点，核对既有断点监听后撤回。

审查与测试覆盖本轮真实金额场景；几何验证不等于对所有数字长度及任意屏幕的阅读保证。

两轴总计：Standards 0、Spec 0；两轴均无已确认的最严重问题。

## 环境问题与限制

- Windows PowerShell的Start-Process遇到PATH/Path重复键，改用本轮拥有的Node子进程及规范化环境。
  沙箱下Chrome子进程未能正常启动，随后在自动审批允许的环境执行独立本机浏览器验收。
- 首次服务重启时旧验收进程仍持有端口，检查实际监听PID及资源内容后停止本轮启动的旧进程，
  确认新构建服务生效后继续验收。没有停止用户原有服务。
- 新增媒体检查的一次Page.captureScreenshot超时被记录为失败；重新启动验收房间并重跑
  完整手机套件通过，随后媒体回归也通过。没有把失败调用当成成功。
- 以上是本地真实浏览器和手机视口模拟，不是本次改动后的手机实机或完整公网验收。
  没有部署或推送远端，没有修改旧云端验收票状态。架构报告阶段完成，候选设计仍待选择。
