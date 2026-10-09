# 05：整合验收与免费上线

Status: in-progress
Type: implementation
Blocked by: Render账号创建Free Web Service及提供实际URL（当前无可控制浏览器或Render API入口）
Spec: [生产规格](../spec.md)
Implementation baseline: `e1964e485a4f569c553df25ef381e656c23d8e0f`

## What to build

验证最终真实B2与声音并推送用户仓库，接入免费Render Web Service，提供真实公网游戏地址。

## Acceptance criteria

- [ ] Standards和Spec分别审查，完整verify含race、真实浏览器/模拟手机及整合证据。
- [ ] scoped commits推送 ggxue/texas-holdem-poker，保留用户内容，不强制覆盖远端。
- [ ] 单实例free、无数据库/支付资源/保活，HTTP健康及HTTPS/WSS游戏验证。
- [ ] 声音、实际局/结算/下一局/重连设100及进程重启清房间验证；真实限制/未覆盖项明确记录。
- [ ] 云依赖或权限缺失明确记录，不能以本地通过或仅推送标上线done。

## Public test boundaries

真实页面及HTTP/WSS；本地和云各自证据，实机/其他浏览器不冒充已验收。

## Validation

03/04已提交：`5f4deb7`、`e1964e4`，全部本地浏览器、HTTP/WS、实际媒体、verify -Race通过。部署同等Go build（netgo、-s -w）及 `node scripts/restart-smoke.mjs` 通过，进程重启清房间、旧Cookie新身份、首次100。整合记录见[证据](../acceptance-evidence.md)。

GitHub仓库已确认由ggxue持有且连接有push权限，仓库尚空。本机CLI缓存sweetjazz0618账号，普通push为403；限定ggxue后无可用CLI凭证。无需扩大仓库权限，改用已授权ggxue GitHub连接发布同一文件树。实际远端提交及内容核验结果发布后追加。

2026-10-10复核Render官方Free、Go、WebSocket及Blueprint/一键部署文档；保持一个Free Web Service、master、自动部署关闭、无数据库/支付资源/额外保活。README和docs/deployment.md给出可审阅的一键入口及手动字段。

Render截图显示用户已登录，但当前CUA库存apps/browsers均为空；没有可用Render API入口。需要用户账号内创建这一个Free服务并提供生成的URL后继续真实HTTPS/WSS、冷启动/重部署、地域/版本/额度验证。没有创建云资源、没有声称已上线。

本地发布准备Standards P2：缺少CDP9228启动命令，已补独立profile隐藏Chrome启动、健康探测及仅关闭本测试进程的命令。Spec无配置/虚报问题；云部分保持未完成。
