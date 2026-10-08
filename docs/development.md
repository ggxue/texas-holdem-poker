# 本地开发

无需数据库。Windows PowerShell 从仓库根目录运行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-go.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-race.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1 -Race
```

Go 与可选的 Windows race C 编译器安装在 .tools/，下载校验 SHA256，
不修改系统 PATH。bootstrap-race 固定 w64devkit 2.10.0，解压进程等待结束后才验证。
已有兼容编译器时可设置 CC；缺工具或任何检查失败都会返回非零。

启动最终构建：

```powershell
$env:PORT = '18080'
& artifacts/build/texas-poker.exe
```

默认端口8080，Ctrl+C停止。单例测试使用 scripts/test-go.ps1 的 -Package/-Run，
并发测试可加 -Race。房间只属于一个进程，重启清空；重新入房设100。

浏览器验收需要 Node 和 Chrome。启动独立测试浏览器及上述服务：

```powershell
Start-Process -FilePath 'C:/Program Files/Google/Chrome/Application/chrome.exe' -WindowStyle Hidden -ArgumentList '--headless=new','--remote-debugging-port=9228','--user-data-dir=D:/texas-poker/artifacts/browser-profile','--no-first-run','about:blank'
& 'C:/Program Files/nodejs/node.exe' scripts/browser-smoke.mjs http://localhost:18080
& 'C:/Program Files/nodejs/node.exe' scripts/restart-smoke.mjs
```

browser-smoke 使用独立浏览器上下文，报告及截图放在忽略的 artifacts/。
restart-smoke 在临时本地端口启动并关闭自己的两个应用进程，验证重启清空。
只关闭本次启动的浏览器和服务，不操作日常浏览器配置。
沙箱若禁止 Chrome 子进程，需获得工具执行权限才能验证，不能当作通过。
手机尺寸模拟不等于实机或云端；云验收见[部署说明](deployment.md)。
