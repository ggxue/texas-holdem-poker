# 12: 免费云运行与整体验收

Status: blocked
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

Implementation baseline: 2536ef6be944db7c8531f53885f7a3c6efe663af。
已完成显式免费单实例render.yaml、健康检查、运行文档、可复现浏览器和进程重启脚本，
以及Windows race工具链。当前官方研究见[免费云研究](../free-web-research.md)。
本地证据见[AC映射](../acceptance-evidence.md)：真实Chrome和模拟手机通过；新进程清空通过。
实际云配置、HTTPS/WSS、休眠、实机和其他浏览器待执行，不标done。
外部阻塞：项目无Git remote，无可用Render账号访问；已向用户询问仓库URL及免费账号入口。
不得创建付费资源、数据库或额外保活。

最终 verify.ps1 -Race 全部通过、退出0。双轴审查范围为baseline至本次暂存改动：
Standards发现1项重启脚本握手拒绝挂起，修复并复查后0；Spec为0。
正常脚本退出0，真实403拒绝实验退出1且清理子进程；最终Chrome复跑退出0。
本地工作已完成，blocked仅指缺仓库/免费账号访问及后续真实云验收，不能标整体完成。

## Comments

外部前提为可访问的免费运行账号及部署权限。05和08包含其他有效票的传递依赖；
09–11为wontfix，不是上线阻塞。不得创建Neon或付费数据库。
