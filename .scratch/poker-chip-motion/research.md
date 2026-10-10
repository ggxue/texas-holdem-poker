# 筹码飞行动画：浏览器方案调研

Date: 2026-10-10
Type: research
Status: findings

目标：现有原生 HTML/JS/SVG 牌桌的席位→底池投入、底池→赢家派奖。以下区分官方事实和本项目工程建议；没有证明某方案是业界绝对最佳。

## 主源事实

- CSS `@keyframes` 可指定各时间点的属性值；可以完成飞行、缩放和淡出。[CSS Animations 标准](https://www.w3.org/TR/css-animations-1/#keyframes)
- `Element.animate(keyframes, options)` 创建、播放并返回 `Animation`；支持运行时生成关键帧。`cancel()` 中止动画、移除效果；取消正在播放的动画会使 `finished` Promise 以 `AbortError` 拒绝，清理流程须覆盖这一分支。[Mozilla animate 文档](https://developer.mozilla.org/en-US/docs/Web/API/Element/animate)、[cancel 文档](https://developer.mozilla.org/en-US/docs/Web/API/Animation/cancel)
- Google 浏览器性能指南建议优先动画化 `transform`、`opacity`，避免触发布局／绘制的属性；并要求用性能工具确认瓶颈。API 名称本身不能保证流畅。[Google 指南](https://web.dev/articles/animations-guide)
- `getBoundingClientRect()` 给出视口坐标，滚动会改变返回位置；动态端点必须在正确布局、坐标系下测量。[Mozilla 几何文档](https://developer.mozilla.org/en-US/docs/Web/API/Element/getBoundingClientRect)
- Canvas 动画需绘制循环；官方指南建议 `requestAnimationFrame`、考虑高分屏缩放并避免昂贵绘制。WebGL 可利用硬件加速，但设备须支持相关能力。[Canvas 指南](https://developer.mozilla.org/en-US/docs/Web/API/Canvas_API/Tutorial/Optimizing_canvas)、[WebGL 文档](https://developer.mozilla.org/en-US/docs/Web/API/WebGL_API)
- Motion 官方库提供 HTML/SVG 动画、错峰和复杂时间线；这是可选能力，并非本项目必须引入的依赖。[Motion 文档](https://motion.dev/docs/animate)
- `prefers-reduced-motion: reduce` 表示用户希望移除或替换容易造成不适的动态效果。[Media Queries 标准](https://www.w3.org/TR/mediaqueries-5/#prefers-reduced-motion)

## 本项目建议（工程判断，待原型验证）

首选 **少量筹码 SVG 元素 + 原生 Web Animations API**。复用现有实体筹码风格，在独立展示层测量席位、底池端点，用 `transform/opacity` 产生弧线、旋转和错峰收拢。金额数字保持精确，飞行数量有上限并示意规模，不按每一枚筹码建立节点。

投入短促，派奖更密集、更有层次；具体时长／数量由手机原型选择。展示层不拦截操作、不决定余额；重连、后台恢复、布局变化及新局到来时，可取消残留效果并显示当前真实状态。减少动态效果时改成静态金额提示。上述均为建议，尚非产品决定。

| 方案 | 本项目取舍（判断） |
| --- | --- |
| CSS 动画 | 固定脉冲／淡出简单；动态端点和逐批取消需额外协调 |
| 原生 WAAPI | 动态关键帧及取消句柄直接适合两类筹码飞行，首选 |
| Canvas／WebGL | 大规模粒子或真实 3D 再考虑；当前需额外渲染、缩放协调 |
| 动画库 | 复杂时间线有价值；先用原生方案验证，避免提前增加依赖 |

限制：未做实际帧率、功耗、低端安卓／iPhone Safari 验收；也未调查商业扑克产品的私有实现。实现前须验证手机端数量上限、布局变化、连续事件、取消清理及减少动态效果。
