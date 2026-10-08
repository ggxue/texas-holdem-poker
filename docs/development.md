# 本地开发

无需数据库或数据库环境变量。Windows PowerShell从仓库根目录运行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-go.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1
```

启动网页：

```powershell
$env:GOCACHE = 'D:/texas-poker/.tools/gocache'
$env:GOPATH = 'D:/texas-poker/.tools/gopath'
$env:GOTOOLCHAIN = 'local'
$env:GOWORK = 'off'
& .tools/go1.27.1/go/bin/go.exe run .
```

默认http://localhost:8080；可用PORT环境变量指定端口。Ctrl+C关闭应用。
房间和筹码只在此进程内，重启清空；重新入房设100。不要将多个进程当成同一房间。
单例测试可用scripts/test-go.ps1的-Package和-Run参数；-Verify调用完整验证。
-Race需要可用C编译器，缺工具不是通过。
