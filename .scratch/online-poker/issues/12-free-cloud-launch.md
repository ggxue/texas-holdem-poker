# 12: 免费云运行与整体验收

Status: ready-for-agent
Type: implementation
Spec: [当前规格](../spec.md)
Blocked by: [05](05-balance-refill.md)、[08](08-disconnect-action-timers.md)
Primary acceptance cases: AC41–AC43

## What to build

部署支持WebSocket的免费Go网页服务。只用进程内存，不创建数据库。
[当前决定](../memory-reset-decision.md)覆盖旧Neon与跨重启存档要求。

## Acceptance criteria

- [ ] 用research查当时官方免费计划、WebSocket、休眠和额度并核对实际控制台；无支付方式/付费资源，平台免费域名及HTTPS/WSS。
- [ ] 电脑/手机及两个浏览器完成一人+机器人、两人+机器人完整牌局、结算和房主手动下一局。
- [ ] 同进程断线重新入房设100；休眠/重部署导致新进程后旧房间清空、新身份首次100，页面明确显示。
- [ ] 记录实际额度、地域、冷启动和测试结果；不通过额外保活保持常驻，不使用多个独立内存进程冒充共享房间。
- [ ] 汇总当前有效AC自动化/浏览器/云证据；AC33、35–37、39–40已撤销，不恢复数据库故障验收。
- [ ] 复核中文逐行逻辑注释、暗牌权限、进程内去重、单池/全押/退出和期限；失败或缺工具明确记录，不标整体done。

## Public test boundaries

真实云HTTP/WSS、不同浏览器与同身份页面、电脑/手机端到端操作。
复用前票公开Interface；云冷启动和新进程清空独立留证。

## Validation

Implementation baseline: 实施前记录。
尚未实施，没有云部署验收证据。

## Comments

外部前提为可访问的免费运行账号及部署权限。05和08包含其他有效票的传递依赖；
09–11为wontfix，不是上线阻塞。不得创建Neon或付费数据库。
