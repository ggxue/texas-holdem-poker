# B2 正式接入与上线研究

Status: complete-research
Type: research
核查日期：2026-10-09。研究仅读取代码、部署配置及官方文档；没有访问凭据值、创建云资源或部署。本文件中的“当前代码”指本轮正式改动开始前的主分支，不代表后续实现已经满足建议。

## 已验证：上线入口与免费运行

- 当前 `git remote -v` 输出为空。项目已记录缺少可访问仓库 URL 和 Render 账号权限；票 12 仍为 blocked，而非已上线。[部署说明](../../docs/deployment.md)、[票 12](../online-poker/issues/12-free-cloud-launch.md)
- 仅检查 `RENDER_API_KEY`、`GH_TOKEN`、`GITHUB_TOKEN` 环境变量是否存在，三者均为 false；没有读取或输出凭据值。这不能证明所有浏览器登录或其他账号入口都不可用，也不能替代实际账号访问确认。
- 根配置明确 Go Web Service、`plan: free`、一个实例、健康检查和手动部署，没有数据库；应用内存状态并未实现跨进程恢复。[配置](../../render.yaml)、[内存决定](../online-poker/memory-reset-decision.md)
- 本次官方复核仍确认 Free：15 分钟无入站流量休眠，下一次请求/新 WebSocket 唤醒约一分钟；每工作区每月 750 小时；无支付方式时带宽耗尽会暂停服务，构建额度耗尽会停止新构建。Free 不支持超过一个实例或持久磁盘，平台可重启服务。[Render Free](https://render.com/docs/free)
- Render 公网支持 WebSocket，应使用 WSS；没有固定最长连接时限，但部署/实例替换会关闭连接，重连不保证原实例，公开 HTTP/WS 共用一个端口。[Render WebSocket](https://render.com/docs/websocket)
- 原生 Go 自动跟随最新 stable，不能固定具体 Go 版本；本仓库要求 Go 1.27.1，构建日志需检查实际版本和编译成功。[Render 语言支持](https://render.com/docs/language-support)、[go.mod](../../go.mod)

**推荐执行路径：** 完成 B2 真实 API/WS 接入、检查与本地验收，然后通过已有免费 Render 配置部署批准的提交。继续保持无卡、单实例、无数据库、无额外保活约束。需要可授权仓库及免费账号入口才能把本地发布候选变成实际 HTTPS/WSS 服务；不能把 localhost 原型链接当作云上线证据。[既有票 12 边界](../online-poker/issues/12-free-cloud-launch.md)

**待验证：** 具体账号能否无卡创建服务、实际地域与当月剩余额度、浏览器登录是否可供操作、云 SDK 版本、真实 HTTPS/WSS、冷启动和实机/第二浏览器结果。历史 5 GB/500 构建分钟记录见[先前研究](../online-poker/free-web-research.md)，本轮没有读取实际 Billing 页面，不把该记录冒充该账号当前额度。Render 官方仍将 Free 定位为试验/兴趣项目而非有可用性保证的生产服务；本项目的“上线”只能按已确认的免费、内存版本验收。[Render Free](https://render.com/docs/free)

## 已验证：现有协议不能无歧义推导全部播报

当前公开边界 `/api/state`、`/api/command`、`/api/ws` 返回身份裁剪的快照。`view` 有版本、本人、房主、座位、连接期限、serverTime；`handView` 有局/机会 ID、阶段、行动者、公共牌、底池、实际可下注/跟注金额、期限、botThinking；参赛者有投入、全押、弃牌、奖项和结算余额。[state.go](../../internal/poker/state.go)、[game.go](../../internal/poker/game.go)、[socket.go](../../internal/poker/socket.go)

| 需求 | 当前可知 | 缺口 |
| --- | --- | --- |
| 当前自己的机会及一次提醒 | hand.id、hand.turn、actor、deadline、serverTime | 可去重和计算实际剩余时间；无需新增游戏期限 |
| 新入房、移出席位、房主变更 | seats、host 的相邻快照变化 | 可知最终状态，不能辨认主动离开与宽限到期原因 |
| 断线、恢复 | disconnectedUntil、connectingUntil 的变化 | 可观察状态；首次握手/重新入房/接管的恢复原因并非明确事件 |
| 下注/跟注及金额 | 投入、余额、target 变化 | 前后快照跨阶段重置 street；没有本次动作字段，不应猜测 |
| 过牌、弃牌、超时 | turn、folded、actor 变化 | 手动过牌与超时自动过牌可产生相同快照；弃牌原因也有歧义 |
| 发牌/翻牌/转牌/河牌/摊牌/结算 | 最终 stage、board、won | 全押推进可一次补足 5 张并 finished；没有各中间阶段事件 |
| 机器人思考 | botThinking、turn | 每个新机器人机会可正确标记，但实际动作仍缺事件 |

这些缺口来自代码：`advance` 可能在同次成功事务内跳过多个阶段；`tickLocked` 可处理多个到期变更，仅最后广播。HTTP 与 WS 同时送达同一结果时，相邻快照推导还容易重复。慢客户端发送通道只有 16 条，满后断开，不是持久可靠消息日志。[推进](../../internal/poker/game.go)、[期限批处理](../../internal/poker/deadlines.go)、[命令去重与事务](../../internal/poker/commands.go)、[WS 队列与广播](../../internal/poker/socket.go)

**推荐协议补充（尚非实现事实）：** 在现有权威快照增加本次确认的公开播报事件批次。每条有唯一事件标识、确认版本、局/机会标识、时间、事件种类、公开玩家/座位、实际金额、allIn 和必要原因；成功动作、自动超时、断线/恢复、离房/房主交接、阶段和奖项都在产生处记录。事件应随同事务回滚，失败动作和重复 requestID 不产生新事件；不携带暗牌、私有牌力、随机未来动作时刻。无需创建数据库或历史恢复队列。[现有事务边界与权限](../../internal/poker/commands.go)、[公开视图裁剪](../../internal/poker/game.go)

前端先按事件标识去重，再按局/机会与年龄丢弃过期项。首次页面/重新连接建立的快照只设游标，不补播此前事件；静音、返回前台也清空旧队列。提醒以当前确认机会和原期限为准，可中断并清空普通队列，绝不等待声音结束再推进计时。恢复入房设 100 时不得据余额误判为全押取消。[已确认声音规则](requirements-discussion.md)、[全押保留规则](../online-poker/memory-reset-decision.md)

## 已验证：本地女声资产如何扩展

- 现有原型在构建前用 SAPI 的 Microsoft Huihui 生成 WAV；脚本包含固定玩家二、固定 6/10/66 的样句。它是 UI 演示资产，不足以覆盖真实五名玩家与所有金额。[原型生成脚本](../../artifacts/poker-screen-audio-prototype/scripts/make-prototype-voice.ps1)
- SAPI 支持将 SpFileStream 设为语音 AudioOutputStream 并写文件；不需要云运行机器安装 Windows/SAPI，只需把生成后的静态文件随 Go 产物携带。[Microsoft SpFileStream](https://learn.microsoft.com/en-us/previous-versions/windows/desktop/ms722562(v=vs.85))
- 本项目没有任意用户昵称字段：成员是随机匿名 ID，界面按固定席位叫“玩家 1–5”及“机器人”。所以当前版本应预制这六种称呼，不能新引入昵称注册/发音问题。[身份](../../internal/poker/identity.go)、[座位与当前 UI](../../internal/poker/state.go)、[app.js](../../internal/poker/web/app.js)
- 金额为 int64 且跨局累积；不能把所有动作录成十筹码。语音需要使用服务端确认的实际扣款及奖项，结算播每个正奖项，零奖项保持屏幕可核对。[筹码与扣款](../../internal/poker/game.go)
- 正式静态路由目前只有根页面和 app.js/cards.js/style.css 白名单；虽然嵌入目录可包含新文件，语音路径仍须显式开放，否则 404。当前 CSP 只允许同源资源，适合本地资产，无需放宽至外部语音服务。[page.go](../../internal/poker/page.go)

**推荐资产方案：** 固定女声在开发机一次生成“玩家一～五/机器人”、动作/阶段/连接短语、数字与单位组件；客户端用组件组合已确认金额，普通中文数词需处理零、十、百、千、万、亿的规则和边界。精确十进制逐位数字是覆盖任意金额的较简单方案，但自然度需试听；不要遗漏数额或读错成固定十。独立的“轮到你出牌”和十秒提醒整句预录，可立即插播。现有范围无须实时昵称 TTS、云调用或付费额度。

**推荐播放器方案：** 同源 WAV 解码缓存到 Web Audio，顺序播放组件，GainNode 控音量，AudioBufferSourceNode.stop() 实现插播取消；取消须同时失效异步资源加载回调，避免已清空旧队列又开始播放。用户交互中恢复 AudioContext，失败保留显式状态与重试入口；不要依赖用户机器恰好有中文女声。Web Audio 的解码、源节点、停止与音量控制有标准接口；其存在不代表拼接后的中文自然度已验收。[W3C Web Audio Recommendation](https://www.w3.org/TR/webaudio-1.0/)

Web Speech 可枚举当前 user agent 的声音且列表可能变化，规范不保证跨设备统一的 Huihui 女声，因此不能用 speechSynthesis 替换已约定的固定声线而声称一致。[Web Speech 规范](https://webaudio.github.io/web-speech-api/)

## 浏览器播放与后台限制

Chrome 有声自动播放受用户交互/站点媒体参与度策略影响；应在“启用声音”点击中解锁播放，捕获 play()/resume() 的失败，不能把 UI 偏好“已启用”当作播放成功。持久偏好恢复后仍可能需要当前页面交互。[Chrome autoplay](https://developer.chrome.com/blog/autoplay/)

隐藏页可能冻结或丢弃；冻结时 JS 计时及 fetch 回调暂停，丢弃后脚本不能运行。因此后台音效只能尽力，声音失败不能阻塞游戏。恢复应先读取新快照、丢旧语音，再只提醒仍然有效的本人机会与真实剩余秒数。[Chrome Page Lifecycle](https://developer.chrome.com/docs/web-platform/page-lifecycle-api)、[已确认范围](requirements-discussion.md)

**验收建议：** 用真实公开 HTTP/WS 验证成功/重复/拒绝命令、机器人、超时、一次跨多阶段、弃牌隐私、房主交接和重连；客户端验证同结果双通道去重、首次不追播、插播取消异步加载、10 秒一次、实际金额组合、静音/前台恢复、播放失败可继续操作；最低桌面满员结算检查所有内容与字体，手机无横向溢出。云账号缺口独立记录，不拖延本地实现，也不把本地浏览器结果标成云验收完成。
