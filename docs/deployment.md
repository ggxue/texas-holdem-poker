# 免费部署

源码仓库为 [ggxue/texas-holdem-poker](https://github.com/ggxue/texas-holdem-poker)，发布分支为master。
B2界面和中文播报已完成本地验收；用户已在既有Render服务试玩，当前还没有提供可独立核验的公网游戏地址。旧公网完整验收状态仍以[发布票05](../.scratch/poker-screen-audio/issues/05-release.md)和真实云票12为准，不将试玩反馈当完整验收。主动全押与黑桃图标的本地证据见[本轮验收](../.scratch/poker-confidence-action/acceptance-evidence.md)。

## 更新已有Render服务（本轮交接）

本轮已在本地master提交主动全押、精确动作金额、中文全押播报和截图风格黑桃favicon。无需新建服务或修改现有构建/启动设置。

1. 在`D:\texas-poker`打开PowerShell，执行`git -c credential.username=ggxue push origin master`，使用此前已认证的仓库账号，确保包含本轮最终提交。
2. 打开Render现有游戏Web Service，核对连接上述仓库和`master`分支。
3. 在服务的Deploys页面点`Manual Deploy → Deploy latest commit`。项目配置自动部署为Off；实际控制台设置以账号为准。此选项构建所连分支的最新提交，见[Render官方部署说明](https://render.com/docs/deploys#manual-deploys)（2026-10-10核验）。
4. 等待新部署成为Live，核对Deploys里提交SHA和`git rev-parse HEAD`相同。打开自己实际游戏链接并`Ctrl+F5`，检查标签黑桃和本人行动时的全押金额。
5. 用两个独立身份试一局：首注全押、对方跟注/弃牌、自动补牌、单池结算和下一局；启用声音听“玩家N全押X筹码”。检查`/healthz`及WSS连接。完整公网验收仍按下文逐项记录。

部署会替换进程，房间、未结束牌局和内存筹码清空。等本局结束且没有人在玩时再更新，之后新入房为100。本轮没有推送或操作Render，也没有推测游戏URL。

## 推荐入口

打开[部署到Render](https://render.com/deploy?repo=https%3A%2F%2Fgithub.com%2Fggxue%2Ftexas-holdem-poker)。
登录后由根目录render.yaml创建Blueprint，检查资源列表只有一个 `texas-poker` Free Web Service，再开始部署。
若仓库为私有，仅授权Render访问这个仓库。需要支付方式或付费计划时停止，不创建数据库或付费资源。
官方提供这种[部署入口](https://render.com/docs/deploy-to-render)，项目已关闭自动部署。

## 手动创建Web Service

在控制台选择 New → Web Service，连接上述仓库；如果此前在Postgres页面，返回Web Service。
按下表设置，先核对Free再创建。

| 字段 | 值 |
| --- | --- |
| Name | texas-poker |
| Language / Runtime | Go |
| Branch | master |
| Root Directory | 留空（仓库根目录） |
| Build Command | `go version && go build -tags netgo -ldflags '-s -w' -o app .` |
| Start Command | `./app` |
| Instance Type | Free；单实例 |
| Health Check Path | `/healthz` |
| Auto-Deploy | Off / 手动 |
| Environment | `CGO_ENABLED=0`、`GOTOOLCHAIN=local` |

应用自动读取Render的PORT并提供网页及WebSocket。原生Go自动跟随最新stable，不能固定Go版本；
检查构建日志输出的版本满足go.mod，并确认编译成功。参见[官方Go部署](https://render.com/docs/deploy-go-gin)与[语言支持](https://render.com/docs/language-support)。

## 部署后的验收

服务状态为Live后，复制实际 `https://…onrender.com` 地址。核对 `/healthz` 返回ok；
使用两个独立浏览器身份完成开局、动作、结算及下一局，确认实时连接为WSS和手牌隐私。
在页面点击启用声音，试听声线并验证轮到本人、十秒提醒、静音和恢复。
同进程重连须设100且保留原期限；实际重部署后须清空房间、新入房100。
记录服务URL、地域、提交SHA、Go版本、实际免费额度、结果及失败，不能用本地证据代替云验收。

## 免费运行限制

2026-10-10复核：[Render Free](https://render.com/docs/free)仍规定15分钟无入站流量后休眠，
下次请求或新WebSocket连接唤醒约一分钟；每工作区每月750免费实例小时。
无支付方式时，出站额度耗尽会暂停服务，构建额度耗尽会停止新构建。
具体带宽与构建分钟看该账号Billing页面，本轮没有验证账号额度。

Free仅单实例且平台可能重启服务。项目无数据库、无额外保活；休眠导致进程结束、重启或重部署
均清空房间、牌局和筹码，不能找回旧余额。页面有连接/唤醒提示。
实际公网WS使用WSS，部署和维护可能断开连接，详见[官方WebSocket说明](https://render.com/docs/websocket)。
维护部署先确认没有进行中的牌局；自动部署已关闭。旧新进程在替换时可能短暂并存，
两者不是共享房间，不承诺无停机或迁移。
