# 04：确认事件与中文播报

Status: done
Type: implementation
Blocked by: None (03 done)
Spec: [生产规格](../spec.md)
Implementation baseline: `5f4deb77a16964053e1b8a7b559db9e56d14f79d`

## What to build

真实确认动作/原因/阶段及房间事件带动本地普通话女声；优先提醒本人，计时照常。

## Acceptance criteria

- [x] 动作含公开行动者/实际金额/全押/超时；开局至全部快速阶段/摊牌/奖项，房间及机器人事件完整。
- [x] 成功事务才发布，失败回滚/重复请求无新事件；不携私牌/牌序/未来思考期限，不回放历史。
- [x] 本地音频覆盖所有席位及真实金额，启用/静音/60%音量及浏览器偏好，播放失败可恢复且不挡游戏。
- [x] 每机会一次和十秒一次，提醒抢播清队列；机会失效/重连/静音恢复/前台回来不播过期，仅有效当前机会。
- [x] 本人故障/接管去重，只本人页；后台尽力且不承诺必达。

## Public test boundaries

用户已确认真实页面、HTTP/WebSocket；离线Clock/牌序控制阶段与自动动作，实际媒体验证播放。

## Validation

公开协议：`announcements`只在已提交HTTP命令结果和在线WS更新出现。每条为进程内稳定`version:index`编号、确认时间、局/机会标识、kind、seat（0–4玩家、5机器人、-1阶段）、action/reason、十进制字符串amount、allIn。查询及新连接无历史；同请求返回相同编号，客户端去重。新进程签发新身份后清客户端事件游标。

Red→green：公开HTTP/WS依次验证实际下注、超时过牌/弃牌、全押连续runout全部阶段及三名并列赢家、机器人思考、断线回来/房主交接。旧协议缺失事件时实际失败。补裸WS恢复的return事件同样先失败，再与握手原子提交；回滚/重复/拒绝/初始快照无新播报，短余额下注2且全押。

真实Chrome155：`node scripts/browser-voice.mjs http://localhost:18082 --casino`。
可信点击后实际WAV解码与WebAudio启动（running、非零duration）、81个音频文件完整；偏好60%/35%保存、静音、恢复实际剩余、不足10秒不重复十秒提示、每机会一次、优先插播取消、HTTP/WS同一前台机会去重、同浏览器新页不追历史、真实媒体请求失败仍可行动且可恢复、HTTP/WS重复接管故障首播不被截断。恢复竞态及接管截断均有公开界面的目标red后green。

真实进程重启：`node scripts/browser-voice-restart.mjs artifacts/build/browser-fixture-v11.exe`。
旧Cookie得到新身份、房间清空、100余额，新进程开局音可再次播放；旧实现事件游标冲突实际red，身份变更清游标后green。测试只操作本地独立fixture进程，使用已有离线构造器，无线上作弊入口。

整合界面：`node scripts/browser-smoke.mjs http://localhost:18082 --casino --layout --cards --results --pending --full-hand --tie-six`，13组通过，无浏览器异常。最终桌面/手机六人结算截图已查看。

`scripts/verify.ps1 -Race` 全部通过（harness、格式、vet、tests、build、race）；JS语法及diff检查通过。
Standards发现前台恢复P2竞态，已修复并以真实HTTP/WS媒体回归关闭；Spec无未解决发现。后续小增量独立复核，无未解决问题。

ignored证据：`artifacts/browser-voice-report.json`（14组及实际媒体播放/停止记录）、`artifacts/browser-voice-restart-report.json`、`artifacts/browser-tie-report.json`。
范围限制：真实媒体启动证据不是用户“顺耳”的试听结论；手机为390px浏览器模拟，未冒充实机/其他浏览器/云验证。后台冻结或锁屏不承诺声音必达。实际云发布及URL属于05。
