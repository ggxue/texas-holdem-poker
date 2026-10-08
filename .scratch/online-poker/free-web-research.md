# Render 免费 Go 网页运行研究

核查日期：2026-10-08。对应 [票 12](issues/12-free-cloud-launch.md) 和 [当前内存决定](memory-reset-decision.md)。当前版本只存进程内存，不使用数据库或磁盘存档；旧 PostgreSQL/Neon 研究不作为本次上线要求。此次仅读官方来源与本地现状，没有创建账号、服务或任何云资源，也没有完成控制台或真实云验收。

## 推荐路径及免费边界

候选为 **Hobby 工作区 + 一个 Free Go Web Service**，同一 Go 进程同时提供网页、API 和 WebSocket。Hobby 无月费，但收费计算资源与超额度使用并不自动免费，必须明确选择 Free，保持账号无支付方式、不添加付费资源。[工作区计划](https://render.com/docs/platform-features-by-plan)、[创建 Free 服务](https://render.com/docs/free)

Render 官方说明免费层适合个人项目、技术试验与预览，并不推荐用于生产应用。本项目可把它作为小规模游戏演示环境；不能承诺持续在线、明天一定部署成功或永久免费。[免费层用途](https://render.com/docs/free)

| 核实项 | 当前官方事实 | 来源 |
| --- | --- | --- |
| 原生 Go | 支持 Go；原生运行环境跟随最新 stable Go 1.x，通常发布后 24 小时内更新；不能固定具体 Go 版本，固定版本需要 Docker | [语言支持](https://render.com/docs/language-support) |
| Free 规格 | 0.1 CPU、512 MB RAM，计划 ID 为 `free` | [计算计划](https://render.com/docs/compute-plans) |
| 运行额度 | 每工作区每自然月 750 Free 实例小时，休眠不消耗；用尽后 Free 网页服务暂停至下月 | [Free 月额度](https://render.com/docs/free#monthly-usage-limits) |
| 出站流量 | Hobby 每月 5 GB；网页响应与对外 WebSocket 消息消耗额度，入站不计；无支付方式时超额度暂停工作区服务至下月 | [出站流量](https://render.com/docs/outbound-bandwidth) |
| 构建额度 | Hobby 每月 500 Starter pipeline 分钟；无支付方式或达构建支出上限时，当月停止新的构建任务 | [构建管线](https://render.com/docs/build-pipeline) |
| 常态实例数量 | Free 不能扩展超过一个实例 | [Free 限制](https://render.com/docs/free#other-limitations) |

若绑定支付方式，公网超额流量按 $0.15/GB 收费；构建额度耗尽也可能自动购买补充分钟。因此本票的无卡条件直接影响超额后是停止还是收费。构建管线的 spend limit 是构建额度限制，不能当成整个账号所有账单的通用零支出开关。[流量计费](https://render.com/docs/outbound-bandwidth)、[构建超额与 spend limit](https://render.com/docs/build-pipeline)

当前直接额度文档已确认 5 GB/500 分钟。不得沿用旧 Hobby 的 100 GB 说法；上线仍需记录实际工作区 Billing 页面额度与剩余量。

## WebSocket、域名与 HTTPS

Render Web Service 接收公网 WebSocket；公网必须使用 **WSS**，`ws://` 握手遇到 HTTPS 重定向可能失败。平台不设置固定 WebSocket 最长连接时限，但实例关闭、部署、维护和网络故障仍会断线。所有公网 HTTP/WebSocket 共用同一个服务端口。[WebSocket 文档](https://render.com/docs/websocket)

每个 Web Service 获得 `*.onrender.com` 子域名；该域名包含免费托管 TLS 证书，HTTP 自动跳转 HTTPS。平台在负载均衡器终止 TLS，再用 HTTP 转发至应用；Go 应用无需自行购买证书。浏览器对外地址为 `https://<name>.onrender.com/` 与 `wss://<name>.onrender.com/api/ws`。[Web Services](https://render.com/docs/web-services)、[托管 TLS](https://render.com/docs/tls)

本地读取发现当前 WebSocket 路径为 `/api/ws`、网页按 `location.protocol` 选择 WSS、静态资源已通过 Go embed 打包。这是本地观察，不是云运行证据。

## 休眠、冷启动与内存重置

Free 服务 15 分钟没有入站流量会休眠；入站 HTTP 请求或既有 WebSocket 消息都算流量。下一次 HTTP 请求或新 WebSocket 连接会唤醒，官方估计约一分钟，浏览器期间可能显示平台加载页面。[休眠规则](https://render.com/docs/free#spinning-down-on-idle)

运行中写入的本地文件在重部署、重启、休眠后丢失，Free 不支持持久磁盘；平台也可能随时重启 Free 服务。[临时磁盘与 Free 限制](https://render.com/docs/free)

**项目推论：** 当前只有内存，旧进程终止后房间、筹码、牌局、Cookie 对应身份记录与去重结果一起消失。平台新实例从空房间启动，成功入房从 100 筹码开始；Cookie 本身不能恢复内存。普通同进程重连的“设为 100”规则由应用保证，不能把平台唤醒当作同进程重连。约一分钟冷启动可能超过游戏 30 秒期限；本版本没有跨进程恢复这些期限的能力。

不增加外部定时访问或无人时持续连接来规避休眠。正常游戏连接的断线检测与重连应与游戏需求一致；空闲休眠验收必须让入站流量真正停止。官方 WebSocket 文档建议心跳和指数退避重连，但它们不等于状态持久化。[WebSocket 连接维护](https://render.com/docs/websocket)

## 支付方式与具体账号的限制

Render 自己的当前介绍明确说免费网页部署无需信用卡；Free/带宽/构建官方文档也描述无支付方式账户的额度耗尽行为，故存在无卡使用路径。[Render 官方免费层介绍](https://render.com/articles/platforms-with-a-real-free-tier-for-developers-in-2026)、[无支付方式的流量行为](https://render.com/docs/outbound-bandwidth)、[无支付方式的构建行为](https://render.com/docs/build-pipeline)

但 Render 官方社区中 `John_B` 的公开支持回复（2025-03-20）说明可能为进一步验证账号而要求卡信息。这是账号风控例外的官方支持信息，不能保证本用户注册/创建时绝不出现绑卡提示；本次该帖子搜索摘要可读，直接打开返回 502。[官方社区验证回复](https://community.render.com/t/the-deployement-of-a-web-service-fails/36005)

**执行边界：** 若实际账号必须添加支付方式才能继续，与票 12 的“无支付方式”条件冲突，停止该部署路径并记录阻塞；不能为了通过验证自行绑卡、升级或创建付费资源。实际控制台仍需确认 Hobby、Free、无支付方式以及无其他收费资源，本研究未完成这些检查。

## 单实例 Go 部署方法（待执行提案）

| 设置 | 此仓库建议 | 依据/验证边界 |
| --- | --- | --- |
| 服务类型 | New → Web Service，关联可授权访问的 Git 仓库与选定分支 | [官方创建步骤](https://render.com/docs/web-services) |
| Language | Go | [原生 Go 示例](https://render.com/docs/deploy-go-gin)；无需改为 Gin |
| Build command | `go version && go build -tags netgo -ldflags '-s -w' -o app .` | [官方 Go 构建示例](https://render.com/docs/deploy-go-gin)，增加版本日志与当前包目标属项目提案 |
| Start command | `./app` | [官方 Go 启动示例](https://render.com/docs/deploy-go-gin)；不能启动多个工作进程 |
| 计算计划 | Free | [Free 创建步骤](https://render.com/docs/free) |
| 监听 | `0.0.0.0:$PORT`，平台默认 PORT=10000 | [端口绑定](https://render.com/docs/web-services#port-binding)；当前 main.go 已读 PORT、监听 `:<port>` |
| 域名与端口 | 使用默认 onrender.com；HTTP 和 `/api/ws` 共用端口 | [Web Services](https://render.com/docs/web-services)、[WebSocket FAQ](https://render.com/docs/websocket#faq) |
| 自动部署 | 首次稳定后可 Off，按维护窗口手动部署 | [Auto-Deploy Off](https://render.com/docs/deploys#configuring-auto-deploys)；减少正在游戏时自动清桌，属项目提案 |
| 存储/附属资源 | 不添加数据库、Key Value、磁盘、额外独立服务 | 当前内存决定；此次不研究或创建这些资源 |

本地 `go.mod` 当前要求 Go 1.27.1。以云构建日志的实际版本和结果为准；当前语言支持文档明确原生 Go 不能固定版本，不采用旧教程中的 `GO_VERSION` 固定说法。若版本不符或构建失败，记录失败，不能声称部署通过。[当前 Go 版本策略](https://render.com/docs/language-support)

## 部署重叠：单实例不等于永远只有一个进程

当前部署文档描述：成功构建后启动新实例，接入新实例流量后，等待 60 秒再向旧进程发 SIGTERM，默认另给 30 秒关闭窗口。重启也通过创建新实例替换旧实例实现。WebSocket 文档说明实例替换会关闭连接，重连不保证回到原实例。[部署序列与重启](https://render.com/docs/deploys#zero-downtime-deploys)、[WebSocket 实例关闭](https://render.com/docs/websocket#handling-instance-shutdown)

**项目推论/待实际验证：** Free 的常态单实例限制不提供内存房间迁移保障。旧 WebSocket 与新的 HTTP/WebSocket 在替换期可能访问不同内存房间。不能把它表述为不中断本局的共享房间；首版宜把部署/重启作为清桌维护，在没有正在游戏的玩家时执行，页面说明房间会重置。若要求替换期间也绝不同时服务两个房间，普通滚动部署不满足，必须另行验证停止旧服务再启动的维护流程；本次没有代替用户确认这个更强要求，也没有试运行控制台。

## 票 12 留证要求及未完成项

后续实际执行应记录：账号是否无卡可用、Hobby/Free 配置、地域、月额度与剩余量、部署 commit 与 Go 版本、HTTPS URL 与 WSS 握手结果、电脑/手机/两浏览器牌局验收、停止流量后的休眠与唤醒耗时、进程重置后的空房间与首次 100。需要将正常断线重连和新进程重置分开验收。

当前没有可审阅的云资源、实际账号或云验收证据；研究完成不等于票 12、AC41–AC43 或整体项目完成。
