# B2与中文播报验收

日期：2026-10-10。规格：[spec.md](spec.md)。票03/04完成；票05真实云部分待执行。

## 本地已通过

| 范围 | 证据 |
| --- | --- |
| B2正式界面 | 1280×720、1920×720、1920×900无需页面或面板滚动；390px手机纵向布局；六席、十类牌型参考、合法操作、独立底池及六人结算 |
| 筹码与状态 | 占用席实体感余额堆、空位无堆、归零隐藏、钱包/投入/底池分开；桌面朝桌心及手机所属玩家下方居中；本人金框与行动者象牙标记共存 |
| 真实游戏 | HTTP/WS过牌/下注/跟注/弃牌、四轮公共牌、机器人三秒离线fixture与真实30秒显示、待局/满员、提交待确认、接管设100及历史结算、提前胜出隐私、座位复用与房主交接 |
| 服务端播报事实 | 实际字符串金额、全押、手动/超时/机器人原因、每个快速runout阶段、全部赢家、房间变化；事务回滚/重复/拒绝无新事件，查询和初始WS无历史 |
| 真实媒体 | 81段惠惠普通话女声WAV可读取；可信点击后WebAudio实际running且duration非零；14组启用/静音/音量/提醒/恢复/网络失败/接管交错案例 |
| 重启 | 实际Go程序重启清空活动局、房主和座位；旧Cookie变新身份，首次100。浏览器保持页面跨fixture进程重启后，新的开局音能再次播出 |
| 代码检查 | verify.ps1 -Race：harness、格式、vet、tests、build、race全部通过；JS语法及git diff --check通过 |
| 独立审查 | Standards发现恢复竞态P2，真实HTTP/WS回归后关闭；后续接管与新进程游标修复均复核。Standards/Spec无未解决问题 |

## 可复现命令

Windows本地Chrome使用独立测试配置，通过CDP9228运行（不使用用户浏览器配置）。先构建并运行回环fixture：

先在仓库根目录启动独立的隐藏测试Chrome（安装在其他路径时只调整FilePath）：

```powershell
$taskProfile = Join-Path $PWD 'artifacts/production-qa-chrome'
$taskChrome = Start-Process -FilePath 'C:/Program Files/Google/Chrome/Application/chrome.exe' -ArgumentList '--headless=new','--disable-gpu','--no-first-run','--disable-extensions','--remote-debugging-port=9228',"--user-data-dir=$taskProfile",'about:blank' -WindowStyle Hidden -PassThru
Invoke-RestMethod http://127.0.0.1:9228/json/version
```

CDP9228应只属于这个测试浏览器；如果端口已被占用，先确认来源，不连接日常浏览器。

```powershell
& .tools/go1.27.1/go/bin/go.exe build -o artifacts/build/browser-fixture.exe ./scripts/browser-fixture
& artifacts/build/browser-fixture.exe
```

在另一个终端执行：

```powershell
node scripts/browser-smoke.mjs http://localhost:18082 --casino --layout --cards --results --pending --full-hand --tie-six
node scripts/browser-voice.mjs http://localhost:18082 --casino
```

两组完整验收各使用新fixture进程，避免前一组留存的断线宽限席位影响下一组。关闭fixture后执行：

```powershell
node scripts/browser-voice-restart.mjs artifacts/build/browser-fixture.exe
& .tools/go1.27.1/go/bin/go.exe build -tags netgo -ldflags '-s -w' -o artifacts/build/texas-poker.exe .
node scripts/restart-smoke.mjs
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1 -Race
```

手动启动的fixture用该终端Ctrl+C关闭；重启验收脚本自己清理所启动的独立子进程。
全部浏览器验收后只关闭上面记录的测试进程：`Stop-Process -Id $taskChrome.Id`。不按名称关闭所有Chrome。

固定牌序、起点及三秒思考只来自已有离线构造器，不属于正式程序，也没有HTTP在线作弊入口。
目标red包括旧页面滚动、席位溢出、手机钱包偏移、缺失事件、恢复重复提醒、重复接管截断首播、新进程事件误去重，修复后公开界面green。

机器证据位于ignored artifacts：browser-tie-report.json（13组）、browser-voice-report.json（14组）、
browser-voice-restart-report.json、restart-report.json；桌面和手机最终六人结算截图已人工查看。

## 尚未验证

Render控制台资源创建、实际免费额度/地域/Go版本、真实URL与HTTPS/WSS、平台冷启动和重部署，
以及实机手机/其他浏览器和用户对声线是否顺耳的试听结论。浏览器模拟与实际媒体启动不冒充这些结果。
后台冻结或锁屏无法保证提醒必达；语音从不暂停服务器计时。
