# Windows 本地 PostgreSQL 测试实例研究

核查日期：2026-10-08。范围仅为实现任务 02 的真实 PostgreSQL 集成测试环境；不改变已选云平台。本次仅读取官方资料和下载响应头，没有下载、解压或运行二进制，没有注册 Windows 服务或修改系统配置。

## 结论与官方依据

建议使用 **EDB 提供的 PostgreSQL 18.6、包装修订 5、Windows x64 二进制 ZIP**，解压到项目 `.tools`，以当前用户执行 `initdb` 与 `pg_ctl start`。PostgreSQL 官方明确链接到无需安装器的 ZIP；EDB 说明其中是安装器所安装文件的归档。PostgreSQL 本体许可允许免费使用。此路径无需 Docker，也无需执行 `pg_ctl register` 注册服务；普通后台启动和服务注册是不同命令。[PostgreSQL Windows 下载](https://www.postgresql.org/download/windows/)、[EDB 二进制归档](https://www.enterprisedb.com/download-postgresql-binaries)、[PostgreSQL 许可](https://www.postgresql.org/about/licence/)、[pg_ctl](https://www.postgresql.org/docs/18/app-pg-ctl.html)

“当前用户在可写项目目录内运行即可”的判断是上述归档、启动命令与文件权限要求的项目应用，尚未在本机实际运行验证；不承诺这台机器缺少运行库或存在执行限制时仍可免管理员权限完成准备。官方 Windows 表列出 PG18 的安装器测试平台为 Windows Server 2025/2022，称可比较的桌面版通常也能运行，未给所有桌面版兼容保证。[Windows 平台表](https://www.postgresql.org/download/windows/)、[initdb 所有者要求](https://www.postgresql.org/docs/18/app-initdb.html)

## 固定下载及完整性验证

官方 EDB 页面 PG18.6 的 Windows x86-64 按钮为 [fileid=1260609](https://sbp.enterprisedb.com/getfile.jsp?fileid=1260609)。本次只进行 `HEAD` 跟随重定向，得到以下归档 URL、HTTP 200、`application/zip`、384620317 字节（约 367 MiB）；响应头最后修改时间为 2026-10-01。包装修订会变化，应固定文件名和摘要，不能只固定“18.6”。

- 归档：[postgresql-18.6-5-windows-x64-binaries.zip](https://get.enterprisedb.com/postgresql/postgresql-18.6-5-windows-x64-binaries.zip)
- SHA-256：`e2246ba91d22345bc3d017586c09ede52d9df180b1eeb480f050445f1cad84e2`
- 摘要来源：EDB 官方安装器仓库中，贡献者 `mayankjaiswal-dev` 针对该精确文件给出的回复。[EDB issue #733 的摘要回复](https://github.com/EnterpriseDB/edb-installers/issues/733#issuecomment-6012030716)
- 本次通过 [GitHub 官方 API 的公开回复](https://api.github.com/repos/EnterpriseDB/edb-installers/issues/733/comments) 读到回复及贡献者标记；网页文本提取可能不显示 Activity 中的回复。

校验是下载后、解压与首次执行前的步骤；以下为待执行示例。摘要不一致必须停止，不能改成以本地算出的值作为预期摘要。

```powershell
$pgTestZip = 'D:\texas-poker\.tools\postgresql-18.6-5-windows-x64-binaries.zip'
$pgTestExpectedSHA256 = 'e2246ba91d22345bc3d017586c09ede52d9df180b1eeb480f050445f1cad84e2'
$pgTestActualSHA256 = (Get-FileHash -LiteralPath $pgTestZip -Algorithm SHA256).Hash
if ($pgTestActualSHA256 -ne $pgTestExpectedSHA256) { throw 'PostgreSQL ZIP SHA-256 mismatch' }
```

限制：这是供应商官方仓库回复中的摘要，并非独立签名的发行清单；归档与摘要都依赖 HTTPS 和相应账户可信度。S3 multipart ETag（本次带 `-46` 后缀）不能充当 SHA-256。没有实际检查该 ZIP 中每个 EXE 的 Authenticode 签名；不能把问题提问者声称“未签名”当成我们已验证的事实。

## 运行库与空间前置条件

EDB 当前安装器参数明确包括默认安装 **Microsoft Visual C++ runtime libraries** 的选项；ZIP 不会替用户运行安装器。本次所查当前 EDB 文档没有给出这个精确 ZIP 的编译工具版本或完整 DLL 清单，因此尚不能确认本机现有运行库是否足够、ZIP 是否捆绑所有依赖。[EDB install_runtimes 参数](https://www.enterprisedb.com/docs/supported-open-source/postgresql/installing/command_line_parameters/)

如父任务首次执行 `postgres.exe --version` 或 `initdb.exe --version` 出现缺少 MSVC DLL 错误，应记录原始错误并先解决运行库，不能从任意 DLL 下载站补文件。Microsoft 文档说明运行库架构必须匹配程序，版本不能旧于编译工具；其当前 v14 x64 官方下载为 [vc_redist.x64.exe](https://aka.ms/vc14/vc_redist.x64.exe)。这只是需要时的官方候选，并不构成本次已证实该归档必须安装某个精确 runtime 版本；本次不安装运行库。[Microsoft 最新受支持 Visual C++ Redistributable](https://learn.microsoft.com/en-us/cpp/windows/latest-supported-vc-redist)

需要 Windows x64、可写解压/数据/日志目录和空闲 TCP 端口。除 ZIP 本身外还需解压和数据库空间；EDB 默认安装最低要求页面列出 2 GB RAM、512 MB 硬盘并说明数据/其他组件另需空间，不能把 512 MB 当作本归档加数据库所需总空间。[EDB requirements](https://www.enterprisedb.com/docs/supported-open-source/postgresql/installing/requirements/)

## 最小操作步骤（待父任务执行）

1. 下载上述固定 ZIP 到 `.tools`，核对 SHA-256，然后解压到 `.tools\postgresql-18.6-5`；保留整个归档所含核心目录及 DLL，先确认实际 `bin\initdb.exe` 路径。下例假设归档顶层为 `pgsql`，目录结构须解压后复核。
2. 使用当前用户和绝对路径，不设置全局 PATH，不创建 OS 用户或服务。先执行版本命令检查工具可加载。创建全新的数据目录，一次 `initdb`；已有数据目录不重复初始化、不自动删除。
3. 创建仅回环监听的后台服务器、专用测试数据库，完成测试后正常停止。保留数据目录可用于服务重启/恢复验证；测试数据生命周期由测试任务安排。

参数依据：`initdb` 的 `-D/-U/-E/--locale/-A/-W`；显式 UTF8 避免 C locale 的默认 SQL_ASCII；`-W` 交互输入密码，自动化可改用 `--pwfile`（文件首行密码、不可提交仓库或输出）。[initdb](https://www.postgresql.org/docs/18/app-initdb.html)

```powershell
$pgTestBin = 'D:\texas-poker\.tools\postgresql-18.6-5\pgsql\bin'
$pgTestData = 'D:\texas-poker\.tools\pg-test-data'
$pgTestLog = 'D:\texas-poker\artifacts\postgres-test.log'
& "$pgTestBin\postgres.exe" --version
& "$pgTestBin\initdb.exe" --version
& "$pgTestBin\initdb.exe" -D $pgTestData -U poker_test -E UTF8 --locale=C -A scram-sha-256 -W
```

启动前确保 `artifacts` 已存在。`pg_ctl start` 在后台启动，`-l` 写日志，`-w/-t` 等待启动；`postgres -h/-p` 指定回环地址和备用端口。这里不使用会关闭 fsync 的 `-F`，以免削弱存档测试。[pg_ctl](https://www.postgresql.org/docs/18/app-pg-ctl.html)、[postgres](https://www.postgresql.org/docs/18/app-postgres.html)

```powershell
& "$pgTestBin\pg_ctl.exe" start -D $pgTestData -l $pgTestLog -o '-h 127.0.0.1 -p 55432' -w -t 30
& "$pgTestBin\pg_ctl.exe" status -D $pgTestData
& "$pgTestBin\psql.exe" -h 127.0.0.1 -p 55432 -U poker_test -d postgres -W -v ON_ERROR_STOP=1 -c 'CREATE DATABASE poker_test;'
& "$pgTestBin\pg_ctl.exe" stop -D $pgTestData -m fast -w -t 30
```

`psql -W` 提示输入密码，`ON_ERROR_STOP` 避免 SQL 失败仍继续脚本；示例只在首次准备时创建数据库，重复执行 CREATE DATABASE 会明确失败。[psql](https://www.postgresql.org/docs/18/app-psql.html)

测试连接目标：`host=127.0.0.1 port=55432 user=poker_test dbname=poker_test`，密码由测试环境提供；本地回环可用 `sslmode=disable`，此设置不可直接复制到 Neon 生产连接。

## 失败条件与验证边界

- 下载失败、SHA-256 不匹配、平台/运行库或 Windows 应用控制阻止执行：停止环境准备，记录具体原因；不降低校验要求、不禁用系统保护。
- `initdb` 失败：检查数据目录是否全新且当前用户可写、运行库及完整解压目录；保留错误，不能将失败当成集成测试通过。
- 启动端口占用：换一个空闲回环端口并同步 DSN；`pg_ctl -w` 超时后进程可能仍在后台启动，应先查 status 和日志，不能立即再启动第二个实例。[pg_ctl 等待语义](https://www.postgresql.org/docs/18/app-pg-ctl.html)
- 关机或测试后未正常停止时，下次可能需要 PostgreSQL 自身恢复；不要删除 `postmaster.pid` 来掩盖正在运行的实例。`-m fast` 正常关闭会回滚活动事务。[pg_ctl](https://www.postgresql.org/docs/18/app-pg-ctl.html)
- 本地 PG18.6 能验证真实事务、锁、唯一约束和重启读档；不验证 Neon 网络中断、额度、冷启动或具体生产版本差异。生产 Neon 的实际 PostgreSQL 主版本仍须对照部署环境，版本不同应保留兼容性验证记录。

完成环境准备的证据应包括：固定 ZIP 和摘要、工具版本、initdb 成功、回环连接/SQL 成功、集成测试结果、停止命令成功。该文档仅研究路径，尚不包含这些执行结果。
