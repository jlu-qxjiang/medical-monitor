# 虚拟团队会议纪要

> **用途**: 所有 Agent 之间的沟通统一记录在此文件
> **规则**: 追加模式，每次新留言追加在最底部，不删除历史记录
> **格式**: `[日期] [发送方 → 接收方] 主题`
> **抄送**: 需要通知的其他 Agent 在结尾标注 `[抄送: xxx]`

---

## 沟通记录

### [2026-07-06] 虚拟团队正式成立

CEO 宣布团队组建完成，成员如下：
- 李产品（产品经理）— 负责产品需求、用户故事、验收标准
- 张架构（架构师）— 负责技术选型、数据库设计、API 设计
- 赵后端（后端开发）— 负责 Go API 实现、数据库、定时任务
- 钱前端（前端开发）— 负责微信小程序页面开发
- 王测试（QA）— 负责测试用例、功能测试、验收
- CEO（你）— 审批决策、冲突仲裁、资源分配

协作规则：
1. 所有 Agent 间沟通统一写入本文件
2. 所有产出物落在 `docs/`、`backend/`、`frontend/` 目录下
3. 每个阶段结束需 CEO 审批通过后才进入下一阶段
4. 分歧无法解决时，双方列理由提请 CEO 裁决

---

---

### [2026-07-06] [李产品 → CEO] PRD v1.0 完成，等待审批

CEO 你好，

三份文档已产出，路径如下：

| 文件 | 路径 | 内容概要 |
|------|------|----------|
| 产品需求文档 | `docs/PRD.md` | 市场背景、用户画像（3 人）、14 条用户故事、MoSCoW 功能分层、核心业务流程（含异常分支）、竞品分析（4 款）、7 项关键决策记录 |
| 用户故事清单 | `docs/user-stories.md` | 22 条用户故事，分老人端（6 条）、子女端（15 条）、家庭组（3 条）、非功能（4 条） |
| 验收标准 | `docs/acceptance-criteria.md` | 41 条验收标准，覆盖 8 个 Must Have 模块 |

**质量自检结果：**
- [x] 每个 Must Have 功能都有对应的用户故事
- [x] 每个用户故事都有明确的验收标准
- [x] 竞品分析覆盖了 4 款产品（超出最低要求 3 款）
- [x] MVP 边界清晰标注（Must Have = 9 项功能）
- [x] 异常流程已考虑（误操作/网络/静默期/多次漏服等 6 条异常分支）
- [x] 功能依赖关系已标注（药品 → 用药计划 → 提醒 → 确认 → 余量扣减）
- [x] 待讨论问题已列出 5 条，请 CEO 逐一确认

**需要 CEO 特别关注的事项：**

1. **Q1-Q5 待讨论问题**（[PRD.md 第 10 节](docs/PRD.md)）— 特别是短信兜底是否入 MVP、产品名称
2. **深夜静默期设计**（AC-ALERT-05）— 22:00-07:00 不发短信/语音，是否合理？
3. **确认撤销 5 分钟窗口**（AC-CONF-04）— 时间长度的设定是否合适？

审批通过后请通知我，我会将 PRD 流转给张架构进行技术方案设计。

---

### [2026-07-06] [CEO → 李产品] Q1-Q4 决策确认

CEO 答复：
1. 短信进行兜底
2. 短信 P1、语音电话 P2
3. 药品模板库先自己整理，但保留连接第三方库的扩展能力
4. Web 管理后台 = 已理解，MVP 不做
5. 产品名称待选

### [2026-07-06] [李产品 → CEO] 收到，PRD 已更新至 v1.1

已将 Q1-Q4 决议同步到 docs/PRD.md，新增决策记录 D8/D9/D10。
产品名称备选已提供：药安心 / 亲人药盒 / 吃了没 / 药点 / 药伴 / 守护药箱 / 按时吃药。
推荐「药安心」或「亲人药盒」，等 CEO 选定。

---

### [2026-07-06] [CEO → 李产品] 最终决议

1. Web 后台也做，分阶段：先做用户端功能，后做管理后台
2. 产品名称：「按时吃药」

### [2026-07-06] [李产品 → CEO] PRD v1.2 更新完成

所有 5 项决策已全部确认，PRD 无遗留待讨论问题。
产品名称已全局替换为「按时吃药」。
Web 后台纳入规划但分阶段执行，不影响当前用户端 MVP 范围。

🎯 PRD 阶段完成，等待 CEO 审批通过后启动张架构（技术方案设计）。



### [2026-07-06] [CEO] PRD 审批通过

正式启动 Phase 2: 技术方案设计，由张架构接手。

---

---

### [2026-07-06] [张架构 → CEO] 技术方案设计完成，等待审批

四份技术文档已产出：

| 文件 | 路径 | 核心内容 |
|------|------|----------|
| 技术决策记录 | `docs/tech-decisions.md` | 9 项 ADR，涵盖框架选型、数据库、缓存、定时任务、鉴权、分层架构、药品扩展 |
| 数据库 Schema | `docs/database-schema.md` | 10 张表完整 DDL + 索引设计 + 迁移计划 + 安全策略 |
| API 接口设计 | `docs/api-design.md` | 9 模块 28 个接口，每个带 Request/Response/错误码/权限 |
| 系统架构文档 | `docs/architecture.md` | 架构图 + 模块划分 + 部署方案 + 关键难点 + 安全方案 + 8 个风险点 |

**技术方案要点：**
- 技术栈：Go 1.22 + Gin + GORM + MySQL 8.0 + Redis + robfig/cron
- 架构：三层架构（Handler → Service → Repository）
- 数据库：10 张表，逻辑外键，utf8mb4
- API：RESTful，JWT 鉴权，统一响应格式
- 安全：手机号 AES-256-GCM 加密，openid 不通过 API 返回
- 部署：腾讯云轻量服务器单实例，预留多实例扩展能力

**需要 CEO 确认的架构级决策：**
1. 数据库不设物理外键（便于扩展，靠代码保证一致性）— 是否接受？
2. 提醒调度用 robfig/cron 单机方案（多实例需 Redis 锁）— MVP 够用，是否 OK？
3. 药品模板库预留 `external_code` + `external_source` 扩展字段 — 满足「保留扩展」要求

审批通过后即可启动赵后端 + 钱前端并行开发。

---

### [2026-07-06] [CEO] 架构方案审批通过

批准张架构全部技术方案，要点确认：
1. 数据库不设物理外键 — 接受，代码保证一致性
2. robfig/cron 单机方案 — MVP OK，后续扩展 Redis 锁
3. 药品表 external_code/external_source 预留字段 — 满足需求

正式启动 Phase 3：赵后端 + 钱前端并行开发。

---

### [2026-07-06] [赵后端 → 张架构] API 与数据库 Schema 一致性审查结果

张架构，我已完成对 `api-design.md` 和 `database-schema.md` 的交叉审查，发现以下问题：

**1. ER 图出现 `medication_logs` 表但无对应 DDL**
- `database-schema.md` 第 1 节 ER 关系概述中标注了 `medication_logs`（关联 reminders/users/medicines），但第 2 节完整 DDL 中没有该表的定义，migration 计划（第 4 节）也未包含
- 疑为早期设计残留或被 `audit_logs` 替代，请确认是否需要在 migration 中补充还是删除 ER 图中的引用

**2. 架构文档 handler 列表遗漏 User 模块**
- API 设计有 3 个 User 接口：GET/PUT /user/profile、PUT /user/phone
- 架构文档 `backend/internal/handler/` 列表中缺少 `user.go`
- 我会在实现时补上 `user.go` handler

**3. medicines 表 `image_url` 字段在 API 中未体现**
- 数据库有 `image_url VARCHAR(512)` 字段（OCR 拍摄原图），但 POST/PUT /medicines 请求 body 中未包含
- MVP 阶段 OCR 为 P1，我会将 image_url 作为可选字段加入 Create/Update 请求

**4. Refresh Token 无长效存储**
- Refresh Token 有效期 30 天，目前仅有 Redis 黑名单机制，无持久化存储
- 如果 Redis 重启，所有 refresh token 将无法校验（但 access token 仍可正常使用）
- 建议：确认 MVP 阶段是否接受此风险。我可以不建新表，用 Redis + 数据库 fallback 策略

**5. miss 端点 reason 字段存储位置**
- POST /reminders/:id/miss 提交 `{ "reason": "..." }`，但 `reminders` 表无 reason 字段
- `reminder_items` 表有 `skip_reason` 字段
- 我会将 miss reason 写入每个 reminder_item 的 `skip_reason` 字段

以上问题 2-5 不影响开发，我会按所述方案实现。问题 1 请确认后告知。

[抄送: 李产品]

---

### [2026-07-06] [钱前端 → 张架构] API 设计前端评审反馈

张架构你好，我从纯前端视角审查了 `docs/api-design.md`，以下是反馈：

**一、缺失接口（需要补充）**

1. **GET /api/v1/reminders/history** — 缺少用药历史查询接口。PRD S9 和 US-CHILD-010 都需要日历视图查看历史服药记录，但 API 中没有历史查询端点。建议增加：
   - 请求参数: `?family_id=1&elder_user_id=2&date=2026-07-06&page=1&page_size=20`
   - 响应: 按日期分组的已确认/未服用/跳过记录列表

2. **文件上传接口** — 药品图片上传涉及 `wx.uploadFile`，需要专用上传端点。建议 `POST /api/v1/upload` 通用文件上传，返回 URL。

**二、接口字段补充建议**

3. **登录接口建议增加家庭组标识** — `POST /auth/login` 响应中无家庭组信息。首页需要判断用户是否有家庭组来决定跳转逻辑，建议登录响应中增加 `has_family: bool` 字段，避免额外请求导致的页面闪烁。

4. **GET /families 建议预留分页** — P2 阶段支持多家庭组，建议接口预留分页参数保持一致性。

5. **药品列表缺少 manufacturer 字段** — `GET /medicines` 列表响应缺少 `manufacturer`（厂家），列表页建议展示此信息。

**三、错误码补充建议**

6. **缺少撤销超时错误码** — `POST /reminders/:id/revoke` 超过5分钟建议增加专用错误码（如 3004），以便前端展示针对性提示。
7. **缺少提醒过期错误码** — 确认已过期提醒时应返回明确错误码。

**四、已确认合理的设计**

8. ✅ 分页格式 (page/page_size + list/total) 合理
9. ✅ 统一响应结构 { code, message, data } 便于前端拦截处理
10. ✅ `/dashboard/overview` 返回 `overall_status` 前端可直接映射红黄绿灯
11. ✅ `/reminders/:id/confirm` 支持 `item_ids[]` 部分确认，满足逐药确认需求

以上，缺失的接口（历史记录、文件上传）请会前补充，其他为建议项可后续迭代。我这边同步开始搭建小程序骨架。

---

### [2026-07-06] [钱前端] 前端骨架与页面开发完成，等待联调

各位，小程序前端开发工作全部完成，以下是交付概要：

**Step 1-2: 设计文档读取 + API 评审**
- 已完成全部设计文档阅读
- 已向张架构提交 API 评审反馈（见上文）

**Step 3: 小程序骨架**
- 项目配置 5 份文件（app.json/app.js/app.wxss/project.config.json/sitemap.json）
- 工具封装 3 份（utils/api.js — 统一请求封装含 token 自动刷新；utils/auth.js — 登录鉴权；utils/date.js — 日期格式化）
- 公共组件 4 个（medicine-card、confirm-dialog、loading、empty-state）
- 自定义 TabBar（根据用户角色动态显示「首页/用药/我的」或「首页/药品/我的」）

**Step 4: P0 核心页面（7 个）**
| 页面 | 路径 | 功能 |
|------|------|------|
| 登录页 | pages/login/ | 微信一键登录，家庭组检测，角色分流跳转 |
| 首页 | pages/index/ | 角色分流：老人→今日用药列表，子女→家庭看板卡片 |
| 今日用药 | pages/reminder/today/ | 大按钮确认/跳过/未服用，confirm-dialog 二次确认，5分钟撤销 |
| 家庭看板 | pages/dashboard/overview/ | 时间槽+状态指示灯，完成率统计 |
| 家庭组列表 | pages/family/list/ | 切换家庭组，成员展示 |
| 创建家庭组 | pages/family/create/ | 表单+6位邀请码生成+复制分享 |
| 加入家庭组 | pages/family/join/ | 邀请码输入+扫码加入 |

**Step 5: P1 页面（7 个）**
| 页面 | 路径 | 功能 |
|------|------|------|
| 药品列表 | pages/medicine/list/ | 搜索+分页+悬浮添加按钮 |
| 添加药品 | pages/medicine/add/ | 表单（名称/规格/厂家/备注/照片/库存）+编辑模式复用 |
| 药品详情 | pages/medicine/detail/ | 完整信息展示+编辑/删除（含关联校验提示） |
| 用药计划 | pages/medicine/schedule/ | 时间槽管理+药品剂量设置+增删改+弹窗表单 |
| 用药历史 | pages/reminder/history/ | 日历选择器+按日查看历史记录 |
| 个人资料 | pages/settings/profile/ | 头像昵称+功能菜单+退出登录 |
| 通知设置 | pages/settings/notification/ | 订阅消息授权+4类通知开关 |

**设计规范执行情况：**
- [x] 老人端大字体（>=32rpx body, >=36rpx button）
- [x] 可点击区域 >= 88rpx
- [x] 关键操作二次确认（confirm-dialog）
- [x] 确认后5分钟可撤销
- [x] 子女端红黄绿灯状态指标
- [x] 加载态（骨架屏+spinner）、空态（引导文案）、错误态（重试按钮）全覆盖
- [x] 设计 Token 全部应用为 CSS 变量
- [x] 所有 API 调用通过 utils/api.js 封装，不直接使用 wx.request
- [x] 全局登录态管理、Token 自动刷新

**待联调事项（给赵后端）：**
1. 用药历史接口 `GET /reminders/history` 尚未定义（目前用 `/reminders/today` 暂代）
2. 文件上传接口 `POST /upload` 尚未定义（药品照片上传待对接）
3. 加入家庭组接口 `POST /families/:id/join` 需要确认 family_id 获取方式
4. 自定义 TabBar 图标是占位文件，需要设计提供正式图标

**待测试事项（给王测试）：**
- 全部 14 个页面的 WXML+WXSS+JS 已就绪，可开始 UI 审查
- 重点测试：登录流程、角色分流、今日用药确认/撤销、家庭组创建/加入

[抄送: 赵后端, 王测试, 张架构]

---

---

### [2026-07-06] [赵后端 → 钱前端/王测试] [抄送 张架构] 后端全部模块开发完成

钱前端、王测试、张架构，各模块代码已全部实现，可以启动联调和测试。

**实现范围（全部 8 个模块 + 定时任务 + 外部服务）：**

| 模块 | 文件 | 接口数 |
|------|------|--------|
| Auth（认证） | handler/auth.go | 5 (login/refresh/profile/update/phone) |
| Family（家庭组） | handler/family.go | 7 (CRUD/join/role/remove/leave) |
| Medicine（药品） | handler/medicine.go | 5 (CRUD + list) |
| Schedule（用药计划） | handler/schedule.go | 4 (CRUD) |
| Reminder（提醒确认） | handler/reminder.go | 5 (today/confirm/miss/skip/revoke) |
| Dashboard（看板） | handler/dashboard.go | 2 (today/overview) |
| Stock（余量） | handler/stock.go | 2 (list/update) |
| Alert（告警） | handler/alert.go | 1 (list) |

**定时任务：**
- `scheduler/reminder_scheduler.go` — 每60秒扫描用药计划 → 生成提醒 → 推送微信
- `scheduler/alert_scheduler.go` — 超时检测 + 30/60/90分钟三级告警升级

**外部服务（Mock 实现）：**
- `external/wechat.go` — code2session + 订阅消息
- `external/sms.go` — 短信发送接口
- `external/voice.go` — 语音电话接口

**基础设施完整清单：**
- 10 个 GORM Model / 7 个 Repository / 5 个 Service / 8 个 Handler
- 6 个中间件（Auth/Family/Role/CORS/Logger/RateLimit/Recovery）
- 统一错误码(12个) / 统一响应格式 / JWT / AES-Crypto / 工具函数
- 10 个 SQL 迁移文件 / viper 配置管理 / main.go 完整路由注册

**待回答钱前端的问题：**
1. 用药历史接口 `GET /reminders/history` — 暂未在 api-design.md 中定义，需张架构补充设计后我再实现
2. 文件上传接口 `POST /upload` — 同上，需补充 API 设计
3. `POST /families/:id/join` — family_id 从邀请码解析获得（invite_code 是唯一的，请求中携带invite_code即可）

**构建说明：**
```
cd backend
go mod tidy
# Set env: MEDICAL_JWT_SECRET, MEDICAL_ENCRYPTION_KEY
go run main.go
```

[抄送: 张架构]

### [2026-07-07] [CEO → 张架构, 赵后端] 数据库 Schema 与文档一致性问题 + API 缺失补充

张架构、赵后端，

昨天的开发审查暴露了几个跨阶段的细节问题，需要你们协同修复：

---

**一、张架构需确认/修复的文档问题**

1. **ER 图出现 `medication_logs` 表但无对应 DDL**
   - `database-schema.md` 第 1 节 ER 关系概述中标注了 `medication_logs`，关联 reminders/users/medicines
   - 但第 2 节 DDL 中无此表，migration 中也无对应 SQL
   - 赵后端判断可能是被 `audit_logs` 替代后的残留引用
   - **行动**：张架构确认后，要么从 ER 图中删除 `medication_logs`，要么补充 DDL + migration

2. **架构文档 handler 列表遗漏 `user.go`**
   - API 设计有 3 个 User 接口（GET/PUT /user/profile、PUT /user/phone）
   - `architecture.md` 的 handler 列表中缺少 `user.go`
   - **行动**：补充上去

3. **medicines 表 `image_url` 字段在 API 设计中未体现**
   - 数据库有 `image_url VARCHAR(512)` 字段，但 POST/PUT /medicines 请求 body 中未包含
   - **行动**：在 Create/Update 药品接口的 Request body 中增加 `image_url` 可选字段

4. **钱前端要求的两个缺失接口**
   - `GET /reminders/history` — 用药历史查询（日历视图）
   - `POST /upload` — 文件上传（药品照片）
   - **行动**：张架构在 `api-design.md` 中补充这两个接口的设计

---

**二、赵后端需同步的修改**

1. 待张架构补充 API 设计后，实现 `GET /reminders/history` 和 `POST /upload` 两个接口
2. `ER 设计残留 medication_logs` 的问题解决后，检查代码中是否有残留引用
3. `image_url` 字段在 API 设计补充后，加入 Create/Update 逻辑

---

**三、角色分工**

- **张架构**：先改文档，改完后 @赵后端 通知
- **赵后端**：文档更新后再改代码，不要自己拍脑袋改设计
- 所有修改保持在各自的产出目录（docs/ + backend/），不要跨边界

[抄送: 钱前端, 王测试]

---
### [2026-07-07] [CEO → 全体] 开发阶段策略说明

澄清一个执行层面的问题，避免误解：

**最终产品形态仍然是微信小程序，不变。** PRD 和架构文档的方向正确。

但**开发测试阶段**，我目前更熟悉网页开发，所以：
- 先用 `backend/web/static/` 下的网页界面做后端功能验证
- 网页测试通过后，钱前端的小程序代码再对接
- 小程序开发本身不受影响，钱前端已有的页面代码保留

对各位的影响：
- **赵后端**：后端 API 同时服务 Web 测试页面和将来小程序，接口设计保持通用即可
- **钱前端**：不受影响，小程序代码继续保留，等 Web 端验证完核心链路再联调
- **其他 Agent**：无影响

这不是架构变更，只是开发顺序调整。

---
### [2026-07-07] [CEO → 李产品, 张架构] docs 文档全面审查与修改计划

李产品、张架构，

Web 开发阶段已启动（`backend/web/static/`），后端新增了 `POST /register` + `POST /login-password` 两个开发期接口。
docs 目录下的 7 份文档需要逐份审查，确认是否需要修改。

---

**一、必须有改动的（3 份）**

| 文件 | 问题 | 修改内容 | 负责人 |
|------|------|----------|--------|
| `database-schema.md` | ① ER 图有 `medication_logs` 残留 ② `users` 表实际新增了 `password_hash` 字段 | ① 删除 ER 图中 `medication_logs` 引用 ② DDL 中补充 `password_hash VARCHAR(256)` | 张架构 |
| `api-design.md` | ① 缺 `GET /reminders/history` ② 缺 `POST /upload` ③ `image_url` 字段未体现在请求体 ④ 缺 `POST /auth/register` 和 `POST /auth/login-password` 两个开发期接口 | 补充 4 项缺失的接口设计 | 张架构 |
| `architecture.md` | ① handler 列表漏了 `user.go` ② 模块划分中未体现 Web 静态文件目录 `web/static/` ③ 开发阶段外部服务使用 Mock | ① 补上 user.go ② 架构图中标注 Web 前端（开发期）和微信小程序（生产期）两条前端线 ③ 外部服务说明 Mock/生产切换 | 张架构 |

**二、建议小幅更新的（2 份）**

| 文件 | 修改内容 | 负责人 |
|------|----------|--------|
| `tech-decisions.md` | 新增 ADR-010：Web 开发阶段认证方案（手机号+bcrypt → 开发期 / 微信 code → 生产期） | 张架构 |
| `PRD.md` | 版本记录中补充 Web 开发阶段的说明，MVP 范围不变 | 李产品 |

**三、无需修改的（2 份）**

| 文件 | 原因 |
|------|------|
| `user-stories.md` | 用户故事描述的是最终产品的使用场景（微信小程序），与开发阶段技术方案无关 |
| `acceptance-criteria.md` | 验收标准对应最终产品功能，不因开发方式改变 |

---

**执行顺序：**

1. **张架构**：先改 `database-schema.md` → `api-design.md` → `architecture.md` → `tech-decisions.md`
2. **李产品**：最后更新 `PRD.md` 版本记录
3. 全部改完后通知 CEO 审批

[抄送: 赵后端, 钱前端, 王测试]

---
### [2026-07-07] [张架构 → CEO, 赵后端] 4 份技术文档修改完成，待审批

四份文档已按 CEO 要求修改完成：

| 文件 | 版本 | 修改内容 |
|------|------|----------|
| `database-schema.md` | v1.0 → v1.1 | users 表 DDL 新增 `password_hash VARCHAR(256)` 字段；medication_logs 残留确认不存在（已清理） |
| `api-design.md` | v1.0 → v1.1 | 新增 `POST /auth/register`、`POST /auth/login-password`、`GET /reminders/history`、`POST /upload` 四个接口；Create/Update 药品请求体补充 `image_url` 字段 |
| `architecture.md` | v1.0 → v1.1 | handler 列表补全 `user.go` 和 `alert.go`；客户端层标注 Web前端(开发期) + 微信小程序(生产期) 双线；新增第5节开发阶段说明（Web前端 + Mock服务） |
| `tech-decisions.md` | v1.0 → v1.1 | 新增 ADR-010：Web 开发阶段双模式认证方案（bcrypt密码 ← 开发期 / 微信code ← 生产期） |

**赵后继可同步的代码修改：**
1. `GET /reminders/history` — 查询指定老人某日历史记录接口实现
2. `POST /upload` — 文件上传接口实现（multipart/form-data）
3. `image_url` — 确认药品 Create/Update 已透传该字段

**李产品待更新：** `PRD.md` 版本记录补充 Web 开发阶段说明

审批通过后赵后端可开始实现新增接口。

[抄送: 李产品, 赵后端, 钱前端, 王测试]

---
### [2026-07-15] [CEO → 王测试] 正式启动测试

王测试，

前后端代码均已完成，核心闭环可运行。现正式通知测试启动。

---

**一、测试范围**

| 端 | 测试方式 | 状态 |
|------|---------|------|
| Web 端 | 浏览器打开 `http://localhost:8080`，三个标签页分别登录老人/子女/管理员 | 已就绪 |
| 小程序 | 微信开发者工具打开 `frontend/miniprogram/`，模拟器测试 | 已就绪 |
| 后端 API | curl / Postman 逐接口验证 | 已就绪 |

**二、核心测试用例**

1. **注册登录**：三个角色分别注册（老人13800000001/子女13800000002/管理员13800000003），验证 role 字段是否正确写入
2. **家庭组闭环**：子女创建家庭组→获得邀请码→老人输入加入→老人看不到药品Tab
3. **药品管理**：子女添加家庭药品+个人药品，老人端确认药品Tab不显示
4. **用药计划**：子女创建家庭计划（选老人+药品+时间）→老人端10s内看到今日服药
5. **确认服药**：老人点已服用→库存扣减→子女看板颜色变化
6. **超时告警**：老人不点确认→10分钟后子女首页出现告警条
7. **个人计划**：子女给自己创建个人计划→首页"我的用药"显示
8. **余量告警**：库存低于阈值→子女首页标红
9. **Token刷新**：等待 Access Token 过期（或手动删 sessionStorage）→验证自动刷新
10. **异常场景**：错误密码/过期邀请码/空表单提交/删除关联药品

**三、测试前准备**

```bash
# 1. 启动后端
cd backend && go run main.go

# 2. Web 测试：浏览器打开 http://localhost:8080
# 3. 小程序测试：微信开发者工具导入 frontend/miniprogram/
#    勾选「不校验合法域名」
```

**四、测试记录**

每条用例结果写在下方（通过/不通过/描述现象）。

---
<!-- 新留言请追加在此行之上 -->
