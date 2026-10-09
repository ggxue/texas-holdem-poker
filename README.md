# 在线德州扑克

Go + HTML/JS/CSS，单房间最多五位玩家与一个机器人。B2深红桌沿席位、独立底池及各席余额筹码、桌面一屏结算，配本地普通话女声和本人优先行动提醒。

筹码是虚拟整数，只保存在运行进程内。重新入房将本人可用筹码设为100；已下注不退款，既有弃牌/全押状态及行动期限不重置。服务重启清空房间、牌局和余额。

## 免费部署

[部署到 Render](https://render.com/deploy?repo=https%3A%2F%2Fgithub.com%2Fggxue%2Ftexas-holdem-poker)

根目录 [render.yaml](render.yaml) 预设一个Free Go Web Service、master分支、手动部署和健康检查，无数据库、付费磁盘或额外保活。打开部署入口后核对只创建这一个Free服务；需要支付方式或付费套餐时停止。部署成功后控制台会给出HTTPS游戏地址，页面自动使用WSS。

当前已完成本地验收，真实云部署和公网验收待执行。具体参数与操作见[部署说明](docs/deployment.md)。

## 本地运行

Windows PowerShell：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-go.ps1
& .tools/go1.27.1/go/bin/go.exe run .
```

打开 `http://localhost:8080`，房主点击“开始新一局”。点击“启用声音”解锁播放；音量默认60%，静音包括行动提醒。后台冻结或锁屏时无法保证提醒必达。

已安装Go1.27.1或更高兼容版本时，也可直接 `go run .`。平台运行时读取 `PORT`。

## 检查与规格

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1 -Race
```

Race检查需要C编译器。[B2和声音规格](.scratch/poker-screen-audio/spec.md)、[验收证据](.scratch/poker-screen-audio/acceptance-evidence.md)记录真实页面、HTTP/WebSocket及实际媒体测试。

语音WAV已包含在仓库并嵌入Go程序，云主机无需Windows或在线TTS。重新生成时在安装Microsoft Huihui的Windows运行 `scripts/make-voice.ps1`。
