# 02: 匿名入房与持久筹码

Status: in-progress
Type: implementation
Spec: [在线德州扑克首版规格](../spec.md)
Blocked by: [01](01-restore-go-baseline.md)
Primary acceptance cases: AC07、AC25

## What to build

打开中文页面按持久 Cookie 身份进入唯一房间，看到自己、房主、真人席位和机器人，以及服务端确认的余额。首次发放 100，关闭重开查回余额，第三名新真人看到房间已满。

## Acceptance criteria

- [ ] 使用真实 PostgreSQL 共同保存身份、整数余额、房间与机器人余额；首次发放提交后才确认，同身份重访不重复发 100。
- [x] 新真人依次占真人1、真人2，最早者房主；第三名新真人拒入，同身份不重复占座或生成第二钱包。
- [ ] Cookie 是服务端签发身份凭证；余额由存档读取，客户端伪造余额或身份声明不能记账。
- [x] 不同浏览器/清Cookie为新身份，不转移或抹掉旧记录；同Cookie可查回测试预置余额37。
- [ ] 页面经 HTTP/WebSocket 展示已确认状态，电脑/手机能操作；读写失败显示恢复中，不先在内存入账。
- [ ] 从首个写命令建立稳定请求去重、存档版本校验和保存后确认规则；后续命令沿用，无临时本地JSON/SQLite权威存档。

## Public test boundaries

规格范围：D1–D2、D3 初始余额、D8 基础提交、D9；案例见[可追踪验收案例](../spec.md#可追踪验收案例)。

- 公开访问/连接及按身份视图验证首次/返回身份、座位、房主、第三人拒入。
- 真实且隔离的 PostgreSQL 验证余额连续与基础写失败，不以内存mock代替提交证据。

## Validation

Implementation baseline: 0d717c6a5b1d2c6ebd6d66b777c2891d8109c8d0。

### 本次精简后的验证

仅运行一次完整验证：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-go.ps1 -Verify
```

- 退出码 0：harness、固定 Go SDK、gofmt、vet、`go test ./...`、build 全部通过。
- 当前 `app_test.go` 仅 2 个 Test 函数：AC07 一个案例；AC25 含 `different-browser`、`cleared-cookie` 两个子案例，全部通过。根包 `[no test files]` 不计业务验收。
- AC07：真人1/2的分配、第一人为房主、第三人拒入且不替换现有席位。AC25：初始100、保留Cookie重访仍37、新浏览器/清Cookie新身份100、原身份余额仍37。
- AC25 不再重建应用，重访以关闭HTTP空闲连接、创建保留Cookie的客户端模拟；不是实际浏览器关页/手机操作的证据。
- 使用已校验的本地 PostgreSQL 18.6，测试逐例创建隔离数据库并清理。`.tools` 和日志 `artifacts/02-trimmed-verification.log` 不提交。
- 历史 RED：原初访测试返回404；原容量测试第二人返回409；随后对应单例 GREEN。这些是精简前的历史证据，不扩大当前保留测试集合。
- 未运行 race detector（未发现可用C编译器）；本轮不新增并发/故障/权限测试。浏览器控制没有可用浏览器，native pipe也不可用，电脑/手机浏览器验收未完成。预览已停止。

### Standards review

使用 `git diff --cached 0d717c6a5b1d2c6ebd6d66b777c2891d8109c8d0 --` 审查实际暂存工作，包括新文件；未用尚无提交的空 `baseline...HEAD` 替代。

发现1项P2，未完成：`app.go` 在应用锁下发送HTTP响应，普通HTTP没有写期限；慢客户端可能阻塞房间命令、连接注册与关闭。违反架构的锁下不等待网络写入原则及可取消I/O标准。按本次停止功能实施的范围保留待处理，不追加边界测试。

### Spec review

测试精简相对用户本次限定的AC07/AC25有0项新增问题；1项已记录的需求冲突：现有PostgreSQL及重访37实现与用户重申的无数据库/重连补100不一致。本票保持in-progress，提交是检查点，不代表整票或新方案完成。

本票已有入房实现，尚未完成整票验收；本次按用户要求停止功能实施，精简测试后提交检查点。
按[项目验证约定](../../../docs/agents/harness.md)记录命令、退出码、目标 RED/GREEN、
可复现浏览器/数据库证据和限制；格式、vet、测试、构建及适用并发检查缺工具或失败不得静默跳过。
完成前以实施 baseline 和本规格分别审查 Standards/Spec，记录结果；仅验收满足才标 done。

## Comments

- 用户已确认12票拆分，本票据据此发布。后续讨论与证据追加到本票，保持规则与案例引用可追踪。
- 实施前检查可用的零费用 PostgreSQL 测试环境，不假定已安装 Docker，不在生产数据写fixture。
- 本票只建立同身份唯一身份/占座基础；完整接管提示和迟到事件由07补齐，牌局重启恢复由10补齐。

### 2026-10-08：用户收窄测试，并指出需求冲突

- 本次用户要求：停止功能扩展；只保留 AC07、AC25 所需测试；精简后跑一次全部测试并提交，不继续下一票。
- 精简前共 7 个 Test 函数。4 个整项超出这两个 AC：WebSocket 通知、持久请求重试/内容冲突/旧版本、数据库写失败、伪造输入/跨站请求。另有余额 37 测试的应用重启部分和初访测试的机器人余额/版本断言超出或重复。
- 精简后仅保留 `TestAC07TwoHumansEnterAndThirdHumanIsRejected` 和 `TestAC25CookiePreservesThirtySevenChipsAndMissingCookieCreatesNewIdentity`。AC25 的新浏览器、清 Cookie 是该案例原有要求，不再增加故障、权限、重启或并发边界测试。
- 用户此次明确重申早先决定为“不做数据库、重连补回 100 筹码”。当前仓库 Q24 写同 Cookie 查回原余额，Q26 写外部 PostgreSQL，Q27 写 Neon Free；未找到所述旧决定的记录。这是记录与用户指正的冲突，不能用仓库记录否定用户决定。
- 冲突一：本票第一项及存档测试要求 PostgreSQL，与“不做数据库”冲突。AC07/AC25 本身没有指定数据库产品；AC07 不需要数据库。
- 冲突二：现有 AC25 要求同 Cookie 的余额 37 重开仍为 37，与“重连补回 100”冲突；它不是仅替换数据库技术即可解决的差异。
- 本次按明确的测试精简范围保留已有实现，不悄然改写 AC25 或删除存档实现。这份提交不代表已实现用户重申的无数据库方案。规格和票据需与该决定同步后再继续功能实施；本票不得标 done。
