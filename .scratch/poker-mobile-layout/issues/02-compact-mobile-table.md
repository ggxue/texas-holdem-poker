# 02：手机紧凑六席与常驻行动

Status: done
Type: task
Blocked by: None
Spec: [手机紧凑六席规格](../spec.md)
Implementation baseline: `9b0bba68d0fbbfeb30adbc12ce5dfe15b6b870cf`

## What to build

在真实手机牌局中采用已选A，常见竖屏首屏可看六席、公共牌、底池、本人手牌和合法操作。
极矮屏仅牌桌局部滚动，行动始终可见；玩家、结果、参考和更多功能按需展开，且不遮挡操作。
桌面保持B2，不改变游戏、连接、声音或筹码规则。

## Acceptance criteria

- [x] MAC01–03：360×640、390×720、412×780首屏核心内容可见，320×568行动常驻，玩家1/4固定席位正确。
- [x] MAC04–05：合法动作与准确金额、待确认禁用和真实响应正常；真实按钮可点击且高度至少44px。
- [x] MAC06–07：玩家、结果和参考内容可读完，六人完整结算可看，详情不遮挡操作或泄露暗牌。
- [x] MAC08–10：空席、机器人、全押、断线、重入100、声音、恢复、接管及退出沿用原规则，功能入口可访问。
- [x] MAC11–12：桌面B2保持原状，生产没有原型工具，真实HTTP/WebSocket牌局通过。
- [x] 单票公开边界red→green、完整验证、Standards/Spec分别审查及有类型前缀的本地提交完成。

## Public test boundaries

沿用已确认真实浏览器页面及HTTP/WebSocket。布局测试观察可见信息与几何、实际点击及公开反馈；
不测试私有函数、CSS实现结构或新增在线作弊入口。场景构造只复用公开离线配置。

## Validation

实施提交：`e4e437f40b0c879c1970d20e968151f91eef0725`，`fix: keep mobile poker table and actions visible`。

[验收证据](../acceptance-evidence.md)记录两项预期red、green、15个手机几何场景、真实HTTP/WebSocket
动作、原有完整牌局及声音回归、完整verify与race、两轴独立审查。全部通过，Standards/Spec均0项
已确认发现；JavaScript语法、diff空白和文档链接检查通过。限定本地验收，没有部署、远端推送或
本次手机实机验收，旧云票状态保留。

## Comments

用户授权自主分票和实施；一张完整呈现票覆盖此范围，无必要的阻塞边。
