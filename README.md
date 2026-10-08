# 按时吃药 —— 家庭电子药盒共享监护仪

> 面向家庭的用药监护微信小程序：子女为老人配置用药计划，系统定时生成提醒并通过微信订阅消息推送；老人超时未确认服药时，自动升级告警通知子女，形成「远程监护老人按时吃药」的完整闭环。

## 核心功能

- **家庭组模式**：子女创建家庭组，通过 6 位邀请码邀请老人加入，远程监护老人用药
- **个人模式**：个人独立管理自己的用药（与家庭组完全解耦）
- **用药计划**：支持多药品、多服药时间点、剂量、起止日期
- **定时提醒**：后端每 60 秒扫描用药计划，自动生成当日提醒，微信订阅消息推送
- **分级告警**：超时 30 分钟未确认，自动告警子女 + 微信再提醒本人
- **药品库存**：库存余量管理、低库存预警、确认服药自动扣减库存
- **微信登录**：code2Session 一键登录，新用户自动注册
- **数据安全**：手机号 AES-256-GCM 加密、bcrypt 密码哈希、JWT 双 Token 轮转、双层限流

## 技术栈

### 后端

| 技术 | 用途 |
|------|------|
| Go 1.22 + Gin | HTTP 框架 |
| GORM + MySQL 8.0 | ORM 与数据持久化 |
| Redis（可选） | Refresh Token 缓存加速 |
| robfig/cron | 定时任务（提醒生成、告警升级） |
| golang-jwt | JWT 认证（双 Token 轮转） |
| zap + lumberjack | 结构化日志 + 滚动切割 |
| viper | 配置管理（支持环境变量覆盖） |

### 前端

| 技术 | 用途 |
|------|------|
| 微信小程序原生 | 前端框架 |
| Vant Weapp | UI 组件库 |
| 原生 HTML/CSS/JS | Web 后台管理 |

### 部署

Linux + systemd + Nginx（HTTPS 反向代理）

## 系统架构

采用**三层分层架构**（Modular Monolith）：

```
HTTP 请求
  → 中间件链（Recovery → Logger → CORS → RateLimit → Auth → Family → Role）
  → Handler 层（参数校验、统一响应）
  → Service 层（业务逻辑）
  → Repository 层（数据访问，GORM）
  → MySQL
```

**定时任务**（独立于 HTTP）：

```
每 60 秒扫描：
  reminderScheduler → 生成当日提醒 → 到点推送微信订阅消息
  alertScheduler    → 检测超时 → 30 分钟告警升级
```

**外部服务**采用「接口 + 真实实现 + Mock」模式，开发环境零成本切换：

```
WechatService（微信登录、订阅消息）→ Mock / Real 实现
SMSService（短信）→ Mock 实现
```

## 目录结构

```
medical-monitor/
├── backend/                    # Go 后端
│   ├── main.go                 # 入口：组装依赖、注册路由、启动调度器
│   ├── config/                 # 配置加载（viper）
│   ├── internal/
│   │   ├── handler/            # HTTP 处理层（Controller）
│   │   ├── service/            # 业务逻辑层
│   │   ├── repository/         # 数据访问层（DAO）
│   │   ├── model/              # 数据模型（15 张表）
│   │   ├── middleware/         # 中间件（鉴权/日志/限流/panic恢复）
│   │   ├── scheduler/          # 定时任务（提醒生成/告警升级）
│   │   ├── external/           # 外部服务（微信/短信）
│   │   ├── database/           # 数据库初始化（自动建库建表）
│   │   ├── logger/             # 日志初始化（zap）
│   │   └── pkg/                # 公共工具（jwt/crypto/errcode等）
│   ├── web/static/             # Web 后台前端
│   └── cmd/                    # 命令行工具
├── frontend/miniprogram/       # 微信小程序前端
│   ├── pages/                  # 页面（首页/登录/药品/计划/提醒等）
│   ├── components/             # 自定义组件
│   ├── utils/                  # 工具（API请求/鉴权/日期）
│   └── custom-tab-bar/         # 自定义底部导航
├── docs/                       # 项目文档（PRD/架构/API设计）
├── deploy/                     # 部署脚本 + systemd 配置
└── LICENSE
```

## 快速启动

### 后端

```bash
# 1. 准备 MySQL（8.0+），创建数据库
# 2. 复制配置模板，填入数据库密码
cp backend/config.yaml.example backend/config.yaml

# 3. 启动（首次启动自动建库建表）
cd backend
go run main.go
# 访问 http://localhost:8080/health 验证
```

### 小程序

1. 微信开发者工具 → 导入项目 → 选择 `frontend/miniprogram` 目录
2. 填入你的小程序 AppID
3. 配置后端地址（`frontend/miniprogram/utils/api.js` 的 `develop` 环境）

## 核心设计亮点

1. **JWT 双 Token 轮转**：Access Token（7 天）+ Refresh Token（30 天），Refresh Token 一次一换，防重放攻击
2. **提醒去重幂等**：按「计划 + 日期 + 时间」唯一判断，高频扫描不产生重复提醒
3. **药品快照**：生成提醒时固化药品名称/剂量，历史记录不受后续修改影响
4. **软删除 + 审计日志**：全表软删除，关键操作全审计，数据可追溯
5. **双层限流**：全局限流（防分布式攻击）+ 每 IP 限流（防单 IP 恶意刷）
6. **家庭/个人解耦**：两套独立的用药数据表，互不干扰
7. **依赖倒置**：外部服务接口化，业务层不依赖具体实现，Mock/Real 零成本切换

## 数据库设计

15 张表，按业务模块划分：

- 用户认证（2）：users、refresh_tokens
- 家庭组（2）：families、family_members
- 家庭用药（6）：medicines、medicine_stocks、medicine_schedules、medicine_schedule_times、reminders、reminder_items
- 个人用药（4）：personal_medicines、personal_medicine_stocks、personal_medicine_schedules、personal_medicine_schedule_times
- 审计（1）：audit_logs

## License

[MIT License](LICENSE)
