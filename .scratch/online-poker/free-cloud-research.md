# 免费云部署与 chips 存档研究

- 查阅日期：2026-10-08（Asia/Singapore）；目标上线日期：2026-10-09。
- 状态：research 完成；推荐为提案，用户尚未选择供应商或接受其限制。
- 范围：Go + HTML/JS/CSS；单房间，最多两个真人和一个机器人；WebSocket；已提交的 chips 存档必须跨应用重启保留。未注册账号、创建资源或部署。
- 方法：仅采用官方文档、定价与更新公告。Neon 页面返回 Markdown，浏览工具解析失败，改用只读 `curl.exe -L` 获取当前官方页面；没有依赖旧额度的记忆或第三方教程。

## 推荐提案

**优先 Render Hobby workspace + Free Web Service + Neon Free PostgreSQL。** Go 服务同时提供静态页面和 WebSocket，chips 放在外部 PostgreSQL；不在容器本地保存权威余额。Render 原生支持 Go，也支持 Docker；Neon 支持标准 PostgreSQL 客户端。选择此组合是为了减少明天部署的运维工作量，属于项目判断，并非平台上线保证。[Render 原生运行环境](https://render.com/docs/native-runtimes)、[Neon 定价](https://neon.com/pricing)

该组合是有额度的免费计划，不是无条件持续可用：闲置会冷启动、超额度会停用，平台维护或重新部署会断开连接。用户是否接受这些限制仍需单独确认。[Render Free 限制](https://render.com/docs/free)、[Neon 计划](https://neon.com/docs/introduction/plans)

## 主方案的已核实事实

### Render：运行 Go 与 WebSocket

| 项目 | 官方当前规则 |
| --- | --- |
| 计算资源 | `free`：0.1 CPU、512 MB RAM；只允许单实例。[Compute Plans](https://render.com/docs/compute-plans)、[Scaling](https://render.com/docs/scaling) |
| 月度运行额度 | 每 workspace 750 Free instance-hours；休眠不计时，耗尽后免费服务暂停到下月。[Free](https://render.com/docs/free) |
| 出站流量 | Hobby 每月共享 5 GB，WebSocket 发往公网的消息计入。[Bandwidth](https://render.com/docs/outbound-bandwidth)、[WebSockets](https://render.com/docs/websocket) |
| 构建额度 | Hobby 每月共享 500 Starter pipeline-minutes。[Build Pipeline](https://render.com/docs/build-pipeline) |
| 休眠与唤醒 | 15 分钟没有入站 HTTP 请求或 WebSocket 消息就休眠；新 HTTP 请求或 WebSocket 连接可唤醒，通常约一分钟。[Free](https://render.com/docs/free) |
| 本地磁盘 | 重启、重部署、休眠时本地改动丢失；Free 不支持持久盘，因此本地 SQLite/JSON 不能承担 chips 存档。[Free](https://render.com/docs/free) |
| 长连接 | 支持入站 WebSocket，没有平台固定最大连接时长；实例退出、维护、部署或网络问题仍会断开，官方建议 ping/pong 与重连。[WebSockets](https://render.com/docs/websocket) |
| 地域 | 可选 Oregon、Ohio、Virginia、Frankfurt、Singapore；现有服务不能直接改地域。[Regions](https://render.com/docs/regions) |

**零费用操作条件：** 官方说明免费 web service 不必绑定信用卡。保持 Hobby workspace、服务 `free`，不添加付费服务或支付方式：流量超额停服务，构建超额停止新构建。若绑定支付方式，免费实例也可能产生出站/构建超额费用；构建 spend limit 不能被当作所有收费项目的总上限。[Render 官方免费平台说明](https://render.com/articles/platforms-with-a-real-free-tier-for-developers-in-2026)、[FAQ](https://render.com/docs/faq)、[Bandwidth](https://render.com/docs/outbound-bandwidth)、[Build Pipeline](https://render.com/docs/build-pipeline)

外部数据库访问仍属于公网流量。Render 未公布“异常大量服务主动请求”的数字阈值，免费服务可能因此被暂停；本游戏低流量是否触发不能仅靠文档保证。[Free](https://render.com/docs/free)

### Neon：外部持久 PostgreSQL

| 项目 | 官方当前规则 |
| --- | --- |
| 免费性质 | Free 为持续免费计划，非试用；不要求信用卡。[Pricing](https://neon.com/pricing) |
| 存储 | 每项目 1 GB PostgreSQL，账户所有 Free 项目合计上限 20 GB。[Plans](https://neon.com/docs/introduction/plans#storage) |
| 计算 | 每项目每月 100 CU-hours；0.25 CU 运行 400 小时即耗尽这份额度，不能理解为无条件 24/7；Free 最大 2 CU，约 8 GB RAM。[Plans](https://neon.com/docs/introduction/plans#compute) |
| 出站 | 每项目每月 5 GB 公网传输，与项目其他产品共享。[Plans](https://neon.com/docs/introduction/plans#public-network-transfer) |
| 闲置 | 5 分钟闲置自动停止计算，Free 不能关闭此设置；再次访问可唤醒，冷启动增加延迟。[Plans](https://neon.com/docs/introduction/plans#scale-to-zero) |
| 额度耗尽 | 计算或出站耗尽后暂停计算到下一计费周期；超存储限制则增加存储的操作失败。官方明确这些限制不会删除数据。[Plans FAQ](https://neon.com/docs/introduction/plans#what-happens-if-i-exceed-my-free-plan-limits) |

**重启保留的依据：** Neon 将数据库持久状态与计算节点分离，停掉计算仍保留存储；因此 Render 应用重启不应清空已经提交到 Neon 的 chips。这是基于架构的项目推断，应用仍须正确提交事务并在重新启动时读取数据库，不能把仅在内存中改过的数字叫作“已存档”。[Neon 存储架构](https://neon.com/blog/wal-s3-lakebase-storage-for-the-era-of-agents)

**长期闲置：** 官方分支归档规则是分支创建超过 14 天且过去 24 小时未访问时可归档，访问时自动解除归档；这是移到归档存储，不是删除。没有从已查文档找到与 Render 免费 PostgreSQL 类似的固定 30 天到期删除规则；这也不构成账号永不被停用、数据永不丢失的承诺。[Branch Archiving](https://neon.com/docs/guides/branch-archiving)

**备份边界：** Free 有最多 6 小时的恢复历史，且受变更量限制；短期恢复窗口不能代替独立长期备份。若后续要承诺误删恢复或长期数据保障，需要另行确认导出、保留与恢复要求。[Plans](https://neon.com/docs/introduction/plans#history-window)

**零费用操作条件：** 只选择 Free，不升级 Launch/Scale，不激活付费产品。Free 达到额度暂停或拒绝写入，保持 Free 不会按付费计划自动补购资源。默认最小计算、缩短空闲数据库连接与避免定时查询可帮助节省额度，具体配置在技术决策和实测后确定。[Pricing](https://neon.com/pricing)、[Plans](https://neon.com/docs/introduction/plans)

## 其他候选与排除原因

| 候选组合 | 适用性与限制 |
| --- | --- |
| Render Free + Supabase Free PostgreSQL | 现实备选：数据库 500 MB、共享 CPU/500 MB RAM、5 GB egress、最多两个活跃 Free 项目；一周闲置会暂停，需在控制台恢复。暂停后 90 天内可恢复原项目，超过窗口仍可下载暂停前逻辑备份再迁移。Free 不含常规自动备份，超额度可能限制服务。相比 Neon，多了长期闲置后的人工恢复工作；暂不首推。[Pricing](https://supabase.com/pricing)、[Pause policy](https://supabase.com/changelog/27497-paused-free-plan-projects-are-restorable-for-90-days)、[Billing FAQ](https://supabase.com/docs/guides/platform/billing-faq) |
| OCI Always Free VM + 同机数据库/SQLite | 条件备选：自管 Linux VM 可运行 Go（推断）；Always Free 是无时间到期额度，限 home region，当前 A1 为合计 2 OCPU/12 GB、块盘合计 200 GB及 5 份卷备份。资源缺货会无法创建；低利用率 VM 可被回收，账号闲置 30 天也可能停用。需要信用卡/符合条件借记卡验证，自行配置 HTTPS、网络和进程恢复，明天第一次注册不宜依赖此路线。[Always Free](https://docs.oracle.com/en-us/iaas/Content/FreeTier/freetier_topic-Always_Free_Resources.htm)、[Free Tier FAQ](https://www.oracle.com/cloud/free/faq/) |

Supabase 若用 PostgreSQL 原生连接，Free 直连是 IPv6；免费 shared pooler 支持 IPv4，可避免购买 IPv4 add-on。Go 后端的 WebSocket 仍由 Render 承担，不必再引入 Supabase Realtime。[Database Connections](https://supabase.com/docs/guides/database/connecting-to-postgres)

明确排除：Render 免费 PostgreSQL 是 30 天到期，之后 14 天宽限期后删除，不满足长期免费存档；免费 Key Value 重启即丢数据。[Free](https://render.com/docs/free)

Koyeb 虽有 Free web instance，但当前官方 FAQ 说明注册默认 Pro 会即时扣按比例订阅费，需要信用卡，验证偶尔可达三个工作日；没有在本次查阅中确认可靠的零扣费注册流程，因此不推荐给“明天、全部免费”的新账号路线。[Koyeb Pricing FAQ](https://www.koyeb.com/docs/faqs/pricing)

## 文档冲突、未知和后续必须确认

- Neon 2026-10-02 官方公告已从 0.5 GB 提高到 1 GB；当前 Plans/Pricing 也写 1 GB。旧 FAQ、旧博客或搜索缓存里的 0.5 GB 不作为当前额度。[变更公告](https://neon.com/blog/neon-free-plan-1-gb-per-project)、[Plans](https://neon.com/docs/introduction/plans)
- Render 在 2026-02-24 修改了 Free 休眠判定，现在入站 WebSocket 消息也延后休眠；只保持一个无消息的连接不能据此保证不休眠。[变更公告](https://render.com/changelog/free-web-services-now-remain-active-while-receiving-websocket-messages)
- Render 原生 Go 现在自动跟随稳定版本，官方 language-support 说明精确固定版本要用 Docker；某些旧教程仍提 `GO_VERSION`，不作为固定 SDK 的依据。[Language Support](https://render.com/docs/language-support)、[2026-04-16 更新](https://render.com/changelog/automatic-go-version-updates)
- 注册与区域实际可用性、用户所在地区可访问性、部署容量、最终控制台额度都尚未实测。优先选择相近的应用/数据库地域，不能只凭用户时区推定访问地点或延迟。尚未创建任何账号。
- 跨应用重启保留已提交 chips，不等于服务崩溃时自动恢复正在进行的牌局。未结算下注如何持久化、如何避免重复结算、失败时是否拒绝继续行动、重连是否回座，仍要逐项决策。
- 明天上线的必要验收应包括：两浏览器实际 WebSocket 对战；应用重启/重部署后原身份余额相同；数据库不可用时没有假成功存档或重复发筹码；新用户冷启动提示合理。上述是建议验收项，具体规则仍由后续 spec 确认。
