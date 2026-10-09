# 05：整合验收与免费上线

Status: ready-for-agent
Type: implementation
Blocked by: [03](03-b2-production-screen.md)、[04](04-announcements-and-voice.md)
Spec: [生产规格](../spec.md)

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

origin已设置，ls-remote成功，远端空；Render截图已登录，尚未创建Web Service。
