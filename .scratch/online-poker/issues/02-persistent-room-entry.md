# 02: 匿名入房与持久筹码

Status: ready-for-agent
Type: implementation
Spec: [在线德州扑克首版规格](../spec.md)
Blocked by: [01](01-restore-go-baseline.md)
Primary acceptance cases: AC07、AC25

## What to build

打开中文页面按持久 Cookie 身份进入唯一房间，看到自己、房主、真人席位和机器人，以及服务端确认的余额。首次发放 100，关闭重开查回余额，第三名新真人看到房间已满。

## Acceptance criteria

- [ ] 使用真实 PostgreSQL 共同保存身份、整数余额、房间与机器人余额；首次发放提交后才确认，同身份重访不重复发 100。
- [ ] 新真人依次占真人1、真人2，最早者房主；第三名新真人拒入，同身份不重复占座或生成第二钱包。
- [ ] Cookie 是服务端签发身份凭证；余额由存档读取，客户端伪造余额或身份声明不能记账。
- [ ] 不同浏览器/清Cookie为新身份，不转移或抹掉旧记录；同Cookie可查回测试预置余额37。
- [ ] 页面经 HTTP/WebSocket 展示已确认状态，电脑/手机能操作；读写失败显示恢复中，不先在内存入账。
- [ ] 从首个写命令建立稳定请求去重、存档版本校验和保存后确认规则；后续命令沿用，无临时本地JSON/SQLite权威存档。

## Public test boundaries

规格范围：D1–D2、D3 初始余额、D8 基础提交、D9；案例见[可追踪验收案例](../spec.md#可追踪验收案例)。

- 公开访问/连接及按身份视图验证首次/返回身份、座位、房主、第三人拒入。
- 真实且隔离的 PostgreSQL 验证余额连续与基础写失败，不以内存mock代替提交证据。

## Validation

Implementation baseline: 待实施开始、任何代码编辑前记录 SHA。

尚未实施，本次发布不构成游戏验收通过。
按[项目验证约定](../../../docs/agents/harness.md)记录命令、退出码、目标 RED/GREEN、
可复现浏览器/数据库证据和限制；格式、vet、测试、构建及适用并发检查缺工具或失败不得静默跳过。
完成前以实施 baseline 和本规格分别审查 Standards/Spec，记录结果；仅验收满足才标 done。

## Comments

- 用户已确认12票拆分，本票据据此发布。后续讨论与证据追加到本票，保持规则与案例引用可追踪。
- 实施前检查可用的零费用 PostgreSQL 测试环境，不假定已安装 Docker，不在生产数据写fixture。
- 本票只建立同身份唯一身份/占座基础；完整接管提示和迟到事件由07补齐，牌局重启恢复由10补齐。

