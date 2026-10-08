# 小程序 UI 美化设计文档

## 元信息

- **日期**: 2026-08-05
- **状态**: 待审核
- **设计方向**: 温馨居家感 — 珊瑚暖橙主调、手绘 SVG 图标、丰富微交互动效

---

## 1. 配色方案

### 1.1 设计 Token 变更

将当前微信绿 (#07C160) 主色调替换为珊瑚/暖橙体系：

| Token | 旧值 | 新值 | 用途 |
|---|---|---|---|
| `--color-primary` | `#07C160` | `#F6745A` | 主按钮、强调色 |
| `--color-primary-dark` | `#06AD56` | `#E56448` | 按钮按下态 |
| `--color-primary-light` | `#E6F9EF` | `#FEF0EC` | 浅色背景、选中底色 |
| `--color-success` | `#07C160` | `#52C999` | 已完成状态（保留绿色语义） |
| `--color-success-light` | `#E6F9EF` | `#E8F8F1` | 成功浅底色 |
| `--color-bg` | `#F5F5F5` | `#FFF9F6` | 全局背景（暖白粉调） |
| `--color-warning` | `#FFC300` | `#FFA940` | 警告色（暖橙） |

保持不变的颜色：`--color-danger` (#FA5151)、`--color-info` (#10AEFF)、`--color-text` (#333333) 等中性色。

### 1.2 导航栏 / TabBar

- navigationBarBackgroundColor: `#F6745A`
- tabBar selectedColor: `#F6745A`
- 渐变背景场景（欢迎页、Banner）使用 `linear-gradient(135deg, #F6745A, #FF8E6B)`

---

## 2. 图标系统

### 2.1 实现方式

微信小程序 `<image>` 不支持 SVG 文件。采用 **CSS 绘制 + base64 PNG** 混合方案：

- 简单图标（check、close、arrow 等）用纯 CSS border/transform 绘制
- 复杂图标（pill、bottle、family 等）导出为 1x/2x 两档 PNG，通过 `<image>` 引用

新建 `frontend/miniprogram/components/icon/` 组件：

**组件接口**:
```js
{
  name: String,   // 图标名称
  size: String,   // 尺寸，默认 '48rpx'
}
```

图标 PNG 文件放在 `frontend/miniprogram/images/icons/` 目录下。

### 2.2 图标清单（共 18 个）

| 序号 | 名称 | 用途 |
|---|---|---|
| 1 | `pill` | 药品相关 |
| 2 | `pill-bottle` | 药品瓶（空状态主图） |
| 3 | `clock` | 提醒/时间 |
| 4 | `check-circle` | 已确认/已完成 |
| 5 | `close-circle` | 未服用/取消 |
| 6 | `skip` | 跳过 |
| 7 | `family` | 家庭组 |
| 8 | `elder` | 老人 |
| 9 | `child` | 子女 |
| 10 | `home` | 首页 Tab |
| 11 | `medicine-box` | 药品 Tab |
| 12 | `user` | 我的 Tab |
| 13 | `add` | 添加 |
| 14 | `edit` | 编辑 |
| 15 | `delete` | 删除 |
| 16 | `alert` | 告警/超时 |
| 17 | `stock-low` | 余量不足 |
| 18 | `calendar` | 日历/日期 |

### 2.3 手绘风格特征

- 线条粗细 2-3px，使用 `stroke-linecap="round"` `stroke-linejoin="round"`
- 轻微不规则边缘（非完美几何圆/直线）
- 使用 `currentColor` 继承外部颜色，保持配色灵活

---

## 3. 手绘插图系统

### 3.1 实现方式

插图也因小程序 `<image>` 不支持 SVG，改为 PNG 方案，存放在 `frontend/miniprogram/images/illustrations/`。

新建 `frontend/miniprogram/components/illustration/` 组件：

**组件接口**:
```js
{
  name: String,    // 插图名称
  width: String,   // 宽度，默认 '400rpx'
}
```

### 3.2 插图清单（共 6 张）

| 序号 | 名称 | 使用场景 |
|---|---|---|
| 1 | `empty-reminder` | 今日无用药计划 |
| 2 | `empty-medicine` | 药品列表为空 |
| 3 | `empty-family` | 无家庭组 |
| 4 | `welcome` | 登录/欢迎页 |
| 5 | `all-done` | 今日全部完成 |
| 6 | `error` | 加载失败/网络错误 |

---

## 4. 组件设计

### 4.1 medicine-card（药品卡片）— 保持核心结构，颜色跟随 Token

- 左边框色：`--color-primary`
- 剂量文字色：`--color-primary`
- 选中状态背景：`--color-primary-light`

### 4.2 confirm-dialog（确认弹窗）— 增加柔和的进入动画

- 主按钮：珊瑚背景 + 白色文字
- 增加 `cubic-bezier(0.34, 1.56, 0.64, 1)` 弹性弹出动画
- 遮罩层添加 8px 毛玻璃效果（`backdrop-filter: blur(4px)`）

### 4.3 empty-state（空状态）— 替换 emoji 为插图组件

- 图标区改为 `<illustration name="xxx" />` 引用对应手绘插图
- 标题字号加大到 `--font-size-lg`
- 描述文字颜色调整为 `--color-text-secondary`

### 4.4 loading（加载中）— 骨架屏保留

- 骨架屏渐变底色从灰色改为暖色调浅灰（`#FFF0EC` → `#FFE4DC`）
- 动画保持现有 shimmer

---

## 5. 动效体系

### 5.1 CSS 动画类库

所有动画定义在 `app.wxss` 中，组件通过 class 引用。

```css
/* 弹性弹入 — 确认弹窗、卡片出现 */
@keyframes bounceIn {
  0% { opacity:0; transform:scale(0.3); }
  50% { transform:scale(1.05); }
  70% { transform:scale(0.95); }
  100% { opacity:1; transform:scale(1); }
}

/* 勾选弹跳 — 确认吃药完成 */
@keyframes checkPop {
  0% { transform:scale(0); }
  60% { transform:scale(1.2); }
  100% { transform:scale(1); }
}

/* 柔和呼吸 — 提醒卡片等待状态 */
@keyframes breathe {
  0%, 100% { box-shadow: 0 0 0 0 rgba(246,116,90,0.4); }
  50% { box-shadow: 0 0 0 12rpx rgba(246,116,90,0); }
}

/* 药瓶摇晃 — 未服药超时提醒 */
@keyframes shake {
  0%, 100% { transform: rotate(0deg); }
  20% { transform: rotate(-8deg); }
  40% { transform: rotate(8deg); }
  60% { transform: rotate(-4deg); }
  80% { transform: rotate(4deg); }
}

/* 粒子撒花 — 全部完成时 */
@keyframes confetti-fall {
  0% { transform: translateY(-100%) rotate(0deg); opacity:1; }
  100% { transform: translateY(100vh) rotate(720deg); opacity:0; }
}

/* 进度条流动 — 现有 shimmer 保持 */
/* 页面淡入 — 现有 fadeInUp 保持，调慢 timing */
```

### 5.2 动效触发场景

| 场景 | 动画 | 触发时机 |
|---|---|---|
| 页面加载 | fadeInUp（0.4s） | 每个页面 onReady |
| 卡片出现 | bounceIn（0.5s） | 列表项依次渲染 |
| 确认吃药 | checkPop（0.4s）→ 勾选图标 | 点击"确认已服用"后 |
| 待处理提醒 | breathe（2s 循环） | 过时未操作的提醒卡片 |
| 超时告警 | shake（0.6s） | 告警卡片出现时 |
| 全部完成 | confetti（2s） | 今日所有提醒全部确认后触发撒花粒子 |
| 按钮按下 | scale(0.96)（0.15s） | 保持现有 |
| 弹窗打开 | bounceIn（0.5s） | 确认弹窗/详情弹窗 |
| 骨架屏 | shimmer（1.5s 循环） | 保持现有 |
| 进度条 | shimmer（2s 循环） | 保持现有 |

### 5.3 动效性能约束

- 只使用 `transform` 和 `opacity` 做动画，避免触发重排
- `will-change` 仅在动画期间添加，结束后移除
- 撒花粒子使用 GPU 合成层（`transform: translateZ(0)`）
- 老人端 `elder-mode` 下减少动效：保留 checkPop 和 breathe，去掉 shake 和 confetti

---

## 6. 老人 / 子女差异化

通过 `.elder-mode` class 控制，继续使用 CSS 变量方案。

### 6.1 共享变量（`app.wxss` 保持现有 `:root` 变量）

### 6.2 老人端覆盖（`.elder-mode`）

```css
.elder-mode {
  --font-size-base: 32rpx;   /* +4rpx */
  --font-size-md: 36rpx;     /* +4rpx */
  --font-size-lg: 42rpx;     /* +6rpx */
  --font-size-xl: 52rpx;     /* +12rpx */
  --spacing-md: 32rpx;       /* +8rpx */
  --radius-md: 20rpx;        /* +4rpx */
  --radius-lg: 32rpx;        /* +8rpx */
}
```

### 6.3 老人端行为差异

- 隐藏复杂操作入口（编辑、删除按钮仅在子女端显示）
- 提醒列表单列大卡片，子女端可使用双列布局
- 老人端确认按钮高度 120rpx → 132rpx

---

## 7. 页面级变更

### 7.1 首页 (index)

- 欢迎页渐变从绿色改为珊瑚暖橙 `linear-gradient(135deg, #F6745A, #FF8E6B)`
- 角色选择卡片 emoji 替换为 `<icon name="elder">` / `<icon name="child">`
- 告警卡片左边框色从 `#FF9500` 改为 `--color-primary`
- 进度条填充色改为 `--color-primary`
- 药品余量 tag 样式跟随新颜色

### 7.2 今日用药 (reminder/today)

- 时间标签背景从 `--color-primary-light`（旧绿色）替换
- 确认大按钮从绿色渐变替换为珊瑚色
- 勾选圆圈 `.checked` 颜色从绿色改为 `--color-success`
- 空状态 emoji `🎉` 替换为 `<illustration name="all-done">`

### 7.3 药品列表 (medicine/list)

- 空状态使用 `<illustration name="empty-medicine">`
- 药品卡片左边框色跟随新主题
- 标签选中态背景改为 `--color-primary-light`

### 7.4 用药计划 (medicine/schedule)

- 输入框 focus 边框改为珊瑚色 `--color-primary`
- tag 选中态跟随新主色调
- "添加时间"按钮文字改为 `--color-primary`

### 7.5 家庭组 (family/*)

- 创建/加入页面空状态插图替换
- 成员卡片角色标签跟随新配色

### 7.6 设置 (settings/*)

- 列表项右侧箭头 icon 替换为 SVG

### 7.7 TabBar 图标

- 三个 Tab（首页、药品、我的）的 iconPath/selectedIconPath 替换为自定义 SVG 图标（单色描边风）

---

## 8. 实施范围与不涉及项

### 8.1 本次实施范围

- `app.wxss` — 颜色 Token 全部替换 + 动画类库新增
- `app.json` — 导航栏颜色更新
- 4 个通用组件重写（icon、illustration 新建，medicine-card/confirm-dialog/empty-state/loading 样式更新）
- 16 个页面 wxss 改色 + wxml emoji 替换
- TabBar 图标更新
- 老人端 `.elder-mode` 变量修正

### 8.2 已识别技术约束

- 微信小程序 `<image>` 不支持 SVG / WebP，图标和插图使用 PNG 格式
- 撒花粒子过于复杂时降级为 CSS 实现的 5-8 个小圆点/星形从顶部飘落（纯 `@keyframes`，不依赖 Canvas）
- TabBar 图标需 PNG 格式，尺寸 81×81px（@3x）
- 老人端动效降级：不使用 shake/confetti，减少 GPU 负担

### 8.3 不涉及

- 后端代码不变
- API 接口不变
- 页面逻辑/JS 不变（仅换样式和图标引用）
- Web 前端 (`backend/web/static/`) 本次不改

---

## 9. 文件变更清单

| 文件 | 操作 | 说明 |
|---|---|---|
| `app.wxss` | **修改** | 全部颜色 Token 替换 + 动画类库新增 |
| `app.json` | **修改** | 导航栏/TabBar 颜色 + 图标路径 |
| `components/icon/` | **新建** | SVG 图标组件（4 文件） |
| `components/illustration/` | **新建** | 手绘插图组件（4 文件） |
| `components/medicine-card/medicine-card.wxss` | **修改** | 颜色跟随 Token |
| `components/confirm-dialog/` (wxss+wxml) | **修改** | 弹性动画 + 按钮配色 |
| `components/empty-state/` (wxss+wxml) | **修改** | 引用插图组件 |
| `components/loading/loading.wxss` | **修改** | 骨架屏暖色调 |
| `pages/index/index.wxss` | **修改** | 全部颜色/emoji 替换 |
| `pages/index/index.wxml` | **修改** | emoji → icon 组件 |
| `pages/reminder/today/today.wxss` | **修改** | 全部颜色替换 |
| `pages/reminder/today/today.wxml` | **修改** | emoji → icon 组件 |
| `pages/reminder/history/*` | **修改** | 颜色替换 |
| `pages/medicine/*/` (4 页 × wxss + wxml) | **修改** | 颜色 + emoji 替换 |
| `pages/dashboard/overview/*` | **修改** | 颜色替换 |
| `pages/family/*/` (3 页 × wxss + wxml) | **修改** | 颜色 + 插图替换 |
| `pages/settings/*/` (2 页 × wxss) | **修改** | 颜色替换 |
| `custom-tab-bar/` (wxss) | **修改** | 颜色跟随新 Token |

**预估文件数**: 约 40 个文件（2 新建组件 × 4 文件 + 约 32 个修改文件）
