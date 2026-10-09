# 03：黑桃浏览器图标

Status: done
Type: implementation
Blocked by: None
Spec: [规格](../spec.md)
Implementation baseline: `15b683f`

## What to build

浏览器标签复用用户截图的金棕圆底、浅金黑桃，使用本地自包含矢量资源。黑桃指花色；实际截图的形状为浅金色，按截图和页面现有品牌颜色重建。

## Acceptance criteria

- [x] AAC18：页面声明和真实资源匹配；GET/HEAD、SVG类型及静态路径/方法限制有效。
- [x] 真实浏览器及16/32px视觉检查清晰；无需外部图片服务。
- [x] 保留既有牌桌与业务行为；记录验证和双轴审查。

## Public test boundaries

真实HTTP静态资源及浏览器页面，无测试私有embed结构。

## Validation

2026-10-10：公开HTTP测试red为`GET /favicon.svg`404；增加固定白名单资源与页面声明后green。

- `go test ./internal/poker -run TestAAC18 -count=1`通过：GET/HEAD 200与SVG MIME、HEAD无body、POST405及未知路径404、页面icon声明。
- 真实Chrome页面声明及图像加载成功；16px/32px渲染截图`artifacts/favicon-16-32.png`已人工查看，棕圆章、浅金黑桃及轮廓清晰。
- 复核用户原始截图后纠正规格草稿的“深色黑桃”描述：截图实际为浅金色黑桃，与现有页面圆章颜色一致；未改牌桌品牌或业务行为。
- 双轴审查与最终真实全押浏览器`--icon`整合核验在票04记录；公网尚未验收。
- 追加独立Standards/Spec审查覆盖`0d71575...253caec`。favicon两轴零发现；Standards发现票02状态名称错误（已改done），不涉及图标或玩法代码。
