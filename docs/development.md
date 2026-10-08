# 本地开发

从项目根目录的 Windows PowerShell 执行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-go.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-go.ps1 -Verify
```

数据库准备脚本下载固定、校验过的 EDB PostgreSQL ZIP 到忽略的 `.tools`，
以当前用户初始化 UTF8、SCRAM 密码认证的数据目录，并监听 `127.0.0.1:55432`。
不会注册 Windows 服务。首次执行需要下载约 367 MiB 的归档，另需解压和数据空间。
下载与摘要来源见[本地数据库研究](../.scratch/online-poker/local-postgres-research.md)。

准备脚本在**当前 PowerShell 进程**设置 `TEST_DATABASE_URL`；`test-go.ps1` 在同一进程内
准备数据库并验证（已有显式测试连接时直接使用）。数据库缺失、密码不匹配、
启动失败或缺少环境变量均明确失败，不跳过集成测试。

测试为每个案例创建随机命名的独立数据库，结束后关闭应用、连接池并删除该测试库。
`TEST_DATABASE_URL` 必须指向专用测试 PostgreSQL；测试账号需要创建数据库的权限。
不要设置为生产连接。脚本生成的随机密码保存在忽略的 `.tools/pg-test-password.txt`，
不写入 Git，不在日志输出；不要分享该文件或完整连接串。

测试完成后正常关闭本地实例：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/start-test-postgres.ps1 -Stop
```

该命令保留数据目录，下次启动不重新初始化。
`verify.ps1 -Race` 还需要可用的 C 编译器；普通 Go 测试不代替 race detector。

单个 TDD 案例可运行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/test-go.ps1 -Package ./internal/poker -Run TestFirstVisitJoinsWithPersistentIdentityAndOneHundredChips
```
