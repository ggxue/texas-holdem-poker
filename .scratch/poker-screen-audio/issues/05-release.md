# 05：整合验收与免费上线

Status: blocked
Type: implementation
Blocked by: Render账号创建Free Web Service/实际URL（当前无可控制浏览器或Render API入口）；GitHub认证及上传已完成。
Spec: [生产规格](../spec.md)
Implementation baseline: `e1964e485a4f569c553df25ef381e656c23d8e0f`

## What to build

验证最终真实B2与声音并推送用户仓库，接入免费Render Web Service，提供真实公网游戏地址。

## Acceptance criteria

- [x] Standards和Spec分别审查，完整verify含race、真实浏览器/模拟手机及整合证据。
- [x] scoped commits推送 ggxue/texas-holdem-poker，保留用户内容，不强制覆盖远端。
- [ ] 单实例free、无数据库/支付资源/保活，HTTP健康及HTTPS/WSS游戏验证。
- [ ] 声音、实际局/结算/下一局/重连设100及进程重启清房间验证；真实限制/未覆盖项明确记录。
- [x] 云依赖或权限缺失明确记录，不能以本地通过或仅推送标上线done。

## Public test boundaries

真实页面及HTTP/WSS；本地和云各自证据，实机/其他浏览器不冒充已验收。

## Validation

03/04已提交：`5f4deb7`、`e1964e4`，全部本地浏览器、HTTP/WS、实际媒体、verify -Race通过。部署同等Go build（netgo、-s -w）及 `node scripts/restart-smoke.mjs` 通过，进程重启清房间、旧Cookie新身份、首次100。整合记录见[证据](../acceptance-evidence.md)。

2026-10-10用户完成官方Git Credential Manager设备认证后，`git -c credential.username=ggxue push -u origin master`成功，将已有scoped commits上传到用户仓库；无force。`git ls-remote --symref origin HEAD refs/heads/master`确认默认分支为master，HEAD及master均为`939cec8aa88e62e7c2573a32eb3c72c497036bfb`，与本地上传提交一致。早先旧账号CLI、GitHub连接写入及SSH失败已解决；未删除原账号凭证，不记录临时验证码或凭据。

2026-10-10复核Render官方Free、Go、WebSocket及Blueprint/一键部署文档；保持一个Free Web Service、master、自动部署关闭、无数据库/支付资源/额外保活。README和docs/deployment.md给出可审阅的一键入口及手动字段。

Render截图显示用户已登录，但当前CUA库存apps/browsers均为空；没有可用Render API入口。需要用户账号内创建这一个Free服务并提供生成的URL后继续真实HTTPS/WSS、冷启动/重部署、地域/版本/额度验证。没有创建云资源、没有声称已上线。

本地发布准备Standards P2：缺少CDP9228启动命令，已补独立profile隐藏Chrome启动、健康探测及仅关闭本测试进程的命令。Spec无配置/虚报问题；云部分保持未完成。

发布准备本地提交：`2bda9fb8895052d3a348d9b12a5d40498d1838bb`。2026-10-10已向用户提供根目录render.yaml的一键部署入口；等待其账号内创建唯一Free Web Service并回传实际URL，再继续公网验收。
