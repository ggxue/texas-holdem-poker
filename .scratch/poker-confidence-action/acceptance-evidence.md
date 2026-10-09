# 主动全押与黑桃图标：本地验收

Date: 2026-10-10（Asia/Singapore）
Spec: [确认规格](spec.md)
Tickets: [02](issues/02-active-allin.md)、[03](issues/03-spade-favicon.md)、[04](issues/04-verification-and-update.md)
Baseline: `0d71575da9400ff48688d7152198d321da36019d`
Feature commits: `15b683f`（主动全押）、`253caec`（favicon）；最终检查修订SHA及双轴审查范围在票04补录。

## 规则与公开检查

正式实现沿用用户已审原型：本人全部余款主动全押；累计本轮投入提高目标并重开应答；短额不降低目标，已全押不再操作。单池、小额最强者可拿全池，无边池/退款。机器人仍只过牌或跟注，真人超时不自动全押。重连设100但全押/投入/期限不撤销。

公开HTTP/WS回归在`internal/poker/allin_test.go`，静态资源在`favicon_test.go`：

| 规格案例 | 独立手算或观察 |
| --- | --- |
| AAC01–02 | 首注99使池102/目标99；先投10再对99补89，全局池300自动结算 |
| AAC03–04 | 5/20/20池48全部给小码最强者；20→50→80加机器人80和四底注池234，机器人余20 |
| AAC05–08 | 对99仅50/30可跟或全押实扣余款，目标不降低；余300跟99留201、全押300或弃牌；最后能行动者余99欠10不得多投，余5可全押 |
| AAC09–10 | 已投10重连100再全押成为本轮110；已全押重连设100仍不能再投，旧控制页拒绝 |
| AAC11–14 | 同标识并发仅扣一次、同内容重试同响应；换内容/过时拒绝，无新播报；重新应答30秒，机器人等1秒跟一次，真人超时弃牌；溢出及机会抽样失败回滚，无事件；私牌按身份裁剪 |
| 大额精度 | 余款9007199254740993及欠额以字符串精确传递，金额事件精确；int64溢出拒绝 |
| 早先跟注者 | 先跟10后目标80需补70、实际余40；实际总池263，小码可赢全池 |
| AAC18–19 | favicon GET/HEAD 200/SVG、HEAD无body、未知路径404/POST405；六人池111分19/19/19/18/18/18 |

TDD的有效red为缺少首注allin、缺少唯一短额响应allin、真实页缺少全押按钮、favicon404；实施后对应green。测试准备中的旧DTO解码错误和跨街目标断言已修正，不冒充行为red。

## 仓库检查

`bootstrap-go.ps1`成功，固定SDK为`go1.27.1 windows/amd64`；`verify.ps1 -Race`最终通过Harness、格式、vet、全部Go测试、构建和race，使用已有固定gcc。JS语法及diff检查通过。

第一次整套verify/race失败于旧UAC09完整合法列表仍要求仅call/fold；按已确认主动全押规则更新为call/fold/allin，仍保留原先过牌者再次回应、起点与底池断言。第二次整套检查通过，无race告警。日志在ignored `artifacts/allin-verify.txt`。

## 真实浏览器

回环固定牌序fixture只用于离线验收，不进入正式可执行文件、不增加线上控制。使用独立隐藏Chrome及隔离profile，原9228占用进程保留，QA采用9237；浏览器脚本通过可选`POKER_CDP_URL`连接。

- 新`browser-allin.mjs`：五个独立真人身份+机器人；真实点击全押99、拦截提交确认禁用、HTTP/WS权威池105及私牌隐私；实际AudioBufferSource媒体播放“玩家一全押九十九筹码。”，不重复全押；六人池600平分100及下一局。
- 正常全押、跟注、弃牌使三名获奖者101、三名弃牌者99；随后下一局目标100，手机真玩家余98看到“跟注 98”“全押 98”，辅助提示“需跟100／本次跟注98后全押”。跟注后全押状态、目标不下降、最终各分100均通过，未注入余额。
- 桌面1280×720六人结算整屏、390px模拟手机无横向溢出，已人工查看PNG。手机是桌面Chrome设备尺寸模拟，不等同真机/锁屏/后台保证。
- favicon真实页面声明与加载成功，16/32px截图`artifacts/favicon-16-32.png`人工查看，与用户截图棕色圆章/浅金黑桃一致。
- 最终`browser-allin.mjs --icon`通过全部新流程及favicon资源；`browser-voice.mjs --casino`通过82份WAV和14组实际媒体/恢复/提醒回归；`browser-smoke.mjs --casino --layout --cards --results --pending --full-hand --tie-six`通过13组既有对局、隐私、接管、容量及布局回归。三套报告无JS错误，Chrome/155.0.8059.40。

主要截图在ignored `artifacts/allin-production-before.png`、`allin-production-call.png`、`allin-production-six-results.png`、`allin-production-short-mobile.png`；公开媒体/流程报告`artifacts/allin-browser-report.json`。

## 复现命令

从仓库根目录运行。先运行`powershell -NoProfile -ExecutionPolicy Bypass -File scripts/bootstrap-go.ps1`，再执行`powershell -NoProfile -ExecutionPolicy Bypass -File scripts/verify.ps1 -Race`。缺race编译器时先按仓库harness文档安装，不静默跳过。

启动专用隐藏Chrome，不使用日常配置；若9237被占用，选其他空闲端口并相应调整变量：

```powershell
$taskProfile = Join-Path $PWD 'artifacts/allin-production-qa-chrome'
$taskChrome = Start-Process -FilePath 'C:/Program Files/Google/Chrome/Application/chrome.exe' -ArgumentList '--headless=new','--disable-gpu','--no-first-run','--disable-extensions','--remote-debugging-port=9237',"--user-data-dir=$taskProfile",'about:blank' -WindowStyle Hidden -PassThru
Invoke-RestMethod http://127.0.0.1:9237/json/version
$env:POKER_CDP_URL = 'http://127.0.0.1:9237'
& .tools/go1.27.1/go/bin/go.exe build -o artifacts/build/browser-fixture-allin.exe ./scripts/browser-fixture
```

在另一个终端运行`& artifacts/build/browser-fixture-allin.exe`，绑定127.0.0.1:18082。依次运行下列三套验收，每套之间Ctrl+C关闭并重新启动fixture，以获得全新房间及身份密钥：

```powershell
node scripts/browser-voice.mjs http://localhost:18082 --casino
node scripts/browser-smoke.mjs http://localhost:18082 --casino --layout --cards --results --pending --full-hand --tie-six
node scripts/browser-allin.mjs http://localhost:18082 --icon
```

结束后Ctrl+C关闭fixture，仅关闭上面创建并记录的测试Chrome进程`Stop-Process -Id $taskChrome.Id`，不按名称停止所有Chrome。本轮已清理自己的fixture与专用测试浏览器，保留原占用9228的进程。

## 独立审查

Standards与Spec两个独立agent审查`0d71575...15b683f`均零发现，涵盖金额整数、目标/应答/期限状态转换、事务及并发。
追加`0d71575...253caec`：favicon两轴零发现；Standards发现实现票02误用决策票状态resolved，已改done。最终完整修订复查在票04记录。无未处理玩法或并发问题。

## 发布边界

用户审过的prototype继续留在独立分支和ignored worktree，未合入正式资源。只完成本地实施、提交与交接；没有推送或Render操作、没有猜测URL。用户已有Render试玩不代替公网完整验收，旧云票状态保持。实际更新流程见[部署文档](../../docs/deployment.md#更新已有render服务本轮交接)。进程替换清空房间和筹码，新入房100。
