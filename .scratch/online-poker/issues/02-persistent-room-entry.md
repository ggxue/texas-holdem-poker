# 02: 匿名入房与内存筹码

Status: done
Type: implementation
Spec: [在线德州扑克首版规格](../spec.md)
Blocked by: [01](01-restore-go-baseline.md)
Primary acceptance cases: AC07、AC25（内存修订）

## What to build

打开页面进入唯一房间；首次100，重连本人筹码设100。身份、钱包、房间和请求结果只在单个进程内。
不依赖数据库、磁盘、数据库环境变量或本地数据库测试准备。

## Acceptance criteria

- [x] AC07：第一/第二真人占真人1/2，第一人为房主；第三人拒入，不抢占现有席位。
- [x] AC25：首次100；同Cookie在同进程内重入沿用身份/席位/房主，将本人余额37设为100。
- [x] AC25：换浏览器/清Cookie按新身份100，不转移或重置仍在内存的其他身份余额。
- [x] 删除PostgreSQL运行与测试依赖；普通启动和完整验证均无需数据库配置。

## Public test boundaries

沿用已确认的HTTP身份与房间视图，仅保留AC07、修订AC25两个Test函数。
余额37可由非公开内存fixture预置，断言通过HTTP观察；没有线上修改筹码入口。
本票不扩充故障、权限、去重、并发或单独WebSocket边界测试。

## Validation

Implementation baseline: ab5afcec917b6757488ddb1ee9beff4b5c48d69a。

当前TDD证据：
- 无数据库构造/AC07 RED：旧New访问nil PostgreSQL连接池而panic（数据库仍强制）；移除依赖、改New后该单例GREEN。
- 修订AC25/same-cookie RED：入房仍保留37，目标为100；改为成功入房设100后AC25三子案例GREEN。
- 未实施牌局，不等于游戏可玩。

最终只运行一次完整验证（清空DATABASE_URL及TEST_DATABASE_URL）：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1
```

退出码0：Harness、Go SDK、gofmt、vet、全部测试和build通过。当前只有2个Test函数：AC07和新版AC25（same-cookie、different-browser、cleared-cookie三个子案例）。日志在忽略的artifacts/02-memory-verification.log，不提交。

启动证据：没有数据库环境变量时构建并运行内存版本，默认配置仅临时PORT=18080；GET页面和/api/state均200。预览验证后停止，没有启动PostgreSQL。go.mod/go.sum只保留WebSocket第三方依赖；数据库准备脚本已删除，测试脚本不再读取数据库配置。

Standards审查：0项新问题；上一轮P2的锁下HTTP写响应已修复，响应解锁后发送并设置期限。
Spec审查：代码/两AC测试无错误；发现2组旧文档覆盖漏项（D9/Out of Scope和03/05/08），已修正，包括05最后的刷新补给条款。
审查命令为git diff --cached ab5afcec917b6757488ddb1ee9beff4b5c48d69a --，覆盖实际暂存修改及新文件。

限制：未运行race detector，当前未发现可用C编译器；没有新增并发/权限/故障测试。没有可用浏览器控制环境，本轮只声明HTTP自动化与启动冒烟检查，不声明电脑/手机可玩验收。重启清空是当前规格及实现生命周期，不添加原已撤销的恢复测试。

## Comments

- 用户明确决定见[内存与重连覆盖](../memory-reset-decision.md)，取代Q26/Q27与旧AC25。旧“原身份保留37”不再是当前验收。
- 补回100是设为100，不是增加100；GET不补给，同一入房请求重试不再次执行。
- 普通关页/刷新/断线后成功重新入房触发；应用重启清空内存，从新房间开始。关闭HTTP空闲连接并保留Cookie可在当前公开测试中模拟重新访问。
- 旧检查点ab5afce：当时PostgreSQL实现及保留37通过，用户要求删减为2测试；与用户决定冲突已记录。本轮消除该冲突，不将旧测试成功当新规则验收。
- 旧Standards发现“HTTP响应在房间锁内、无写期限”；本轮解锁后写响应且设置期限，一并修复。
- 完整接管、离房、计时和牌局是后续票据；本轮不越票实施。
