# API 接口设计 —「按时吃药」

> 版本: v1.1 | 作者: 张架构 | 日期: 2026-07-07

---

## 1. 通用规范

### 1.1 Base URL

```
开发环境:  http://localhost:8080/api/v1
生产环境:  https://api.anshichiyao.com/api/v1
```

### 1.2 统一响应结构

```json
{
  "code": 0,
  "message": "success",
  "data": { }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 0 = 成功，其他 = 错误码 |
| message | string | 提示信息（开发环境返回详细错误，生产环境返回用户友好提示） |
| data | object/array/null | 响应数据 |

### 1.3 鉴权方式

```
Header: Authorization: Bearer <jwt_token>
```

- 除登录接口外，所有接口需要携带 JWT Token
- Token 过期返回 `401` + `code: 1001`
- 无权限操作返回 `403` + `code: 1002`

### 1.4 分页规范

请求：
```json
{
  "page": 1,
  "page_size": 20
}
```

响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}
```

### 1.5 错误码枚举

| 错误码 | HTTP 状态码 | 说明 |
|--------|-----------|------|
| 0 | 200 | 成功 |
| 1001 | 401 | Token 无效或已过期 |
| 1002 | 403 | 无操作权限 |
| 1003 | 400 | 参数校验失败 |
| 1004 | 404 | 资源不存在 |
| 1005 | 409 | 资源冲突（如重复加入家庭组） |
| 1006 | 429 | 请求频率超限 |
| 2001 | 500 | 服务器内部错误 |
| 2002 | 502 | 第三方服务异常（微信/短信） |
| 3001 | 400 | 邀请码无效或已过期 |
| 3002 | 403 | 非家庭组成员，无权访问 |
| 3003 | 400 | 药品已关联用药计划，无法删除 |

---

## 2. 接口清单

### 2.1 认证模块 — Auth

#### POST /api/v1/auth/login — 微信登录

```
说明: 微信 code 换取 JWT Token
权限: 无需登录
```

**Request:**
```json
{
  "code": "081xxxxx"  // 微信 wx.login() 返回的 code
}
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "access_token": "eyJhbGciOi...",
    "refresh_token": "eyJhbGciOi...",
    "expires_in": 604800,
    "user": {
      "id": 1,
      "nickname": "李薇",
      "avatar_url": "https://...",
      "is_new_user": false
    }
  }
}
```

**错误码:** 2002（微信接口调用失败）

---

#### POST /api/v1/auth/refresh — 刷新 Token

```
说明: 用 Refresh Token 换取新的 Access Token
权限: 无需登录（用 Refresh Token 鉴权）
```

**Request:**
```json
{
  "refresh_token": "eyJhbGciOi..."
}
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "access_token": "eyJhbGciOi...",
    "refresh_token": "eyJhbGciOi...",
    "expires_in": 604800
  }
}
```

---

#### POST /api/v1/auth/register — 手机号注册（Web开发阶段）

```
说明: 手机号+密码注册，开发阶段使用，小程序上线后废弃
权限: 无需登录
```

**Request:**
```json
{
  "phone": "13800138000",
  "nickname": "李薇",
  "password": "123456"
}
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "access_token": "eyJhbGciOi...",
    "refresh_token": "eyJhbGciOi...",
    "expires_in": 604800,
    "user": {
      "id": 1,
      "nickname": "李薇",
      "avatar_url": "",
      "is_new_user": true
    }
  }
}
```

**错误码:** 1005（手机号已注册）

---

#### POST /api/v1/auth/login-password — 密码登录（Web开发阶段）

```
说明: 手机号+密码登录，开发阶段使用
权限: 无需登录
```

**Request:**
```json
{
  "phone": "13800138000",
  "password": "123456"
}
```

**Response:** 同 `/auth/login`，返回 JWT Token 对 + 用户信息

**错误码:** 1001（密码错误或手机号未注册）

---

### 2.2 家庭组模块 — Family

#### POST /api/v1/families — 创建家庭组

```
说明: 创建一个新家庭组，创建者自动成为 owner
权限: 已登录
```

**Request:**
```json
{
  "name": "张秀兰的家庭"
}
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "张秀兰的家庭",
    "invite_code": "382915",
    "created_by": 1,
    "created_at": "2026-07-06T12:00:00Z"
  }
}
```

---

#### GET /api/v1/families — 我的家庭组列表

```
说明: 查询当前用户所属的全部家庭组
权限: 已登录
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "name": "张秀兰的家庭",
        "my_role": "owner",
        "member_count": 3,
        "elders": [
          { "user_id": 2, "nickname": "张秀兰", "avatar_url": "..." }
        ]
      }
    ]
  }
}
```

---

#### GET /api/v1/families/:id — 家庭组详情

```
说明: 查看家庭组成员列表和角色
权限: 该家庭组成员
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "张秀兰的家庭",
    "invite_code": "382915",
    "members": [
      { "user_id": 1, "nickname": "李薇", "avatar_url": "...", "role": "owner", "joined_at": "..." },
      { "user_id": 2, "nickname": "张秀兰", "avatar_url": "...", "role": "elder", "joined_at": "..." },
      { "user_id": 3, "nickname": "李强", "avatar_url": "...", "role": "member", "joined_at": "..." }
    ]
  }
}
```

---

#### POST /api/v1/families/:id/join — 加入家庭组

```
说明: 通过邀请码加入家庭组
权限: 已登录
```

**Request:**
```json
{
  "invite_code": "382915"
}
```

**Response:**
```json
{
  "code": 0,
  "message": "已加入「张秀兰的家庭」"
}
```

**错误码:** 3001（邀请码无效）、1005（已在家庭组中）

---

#### PUT /api/v1/families/:id/members/:uid/role — 设置成员角色

```
说明: 修改指定成员的角色
权限: owner 或 admin
```

**Request:**
```json
{
  "role": "elder"
}
```

**允许的角色值:** `admin` / `member` / `elder`

---

#### DELETE /api/v1/families/:id/members/:uid — 移除成员

```
说明: 将成员移出家庭组
权限: owner 或 admin（不能移除自己）
```

---

#### DELETE /api/v1/families/:id/leave — 退出家庭组

```
说明: 当前用户主动退出家庭组
权限: 该家庭成员（owner 需先转让或解散家庭组）
```

---

### 2.3 药品管理模块 — Medicine

#### POST /api/v1/families/:fid/medicines — 添加药品

```
说明: 手动录入药品信息
权限: 该家庭组成员（非 elder 角色）
```

**Request:**
```json
{
  "name": "硝苯地平缓释片",
  "specification": "30mg×7片",
  "manufacturer": "拜耳医药",
  "image_url": "",
  "notes": "饭后服用，避免与葡萄柚同食",
  "initial_stock": {
    "total_count": 28,
    "alert_threshold": 7
  }
}
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "硝苯地平缓释片",
    "specification": "30mg×7片",
    "manufacturer": "拜耳医药",
    "notes": "饭后服用，避免与葡萄柚同食",
    "stock": {
      "total_count": 28,
      "remaining_count": 28,
      "alert_threshold": 7
    },
    "created_at": "2026-07-06T12:00:00Z"
  }
}
```

---

#### GET /api/v1/families/:fid/medicines — 药品列表

```
说明: 查询家庭组内所有药品
权限: 该家庭成员
查询参数: ?keyword=硝苯地平&page=1&page_size=20
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "name": "硝苯地平缓释片",
        "specification": "30mg×7片",
        "stock_remaining": 28,
        "stock_alert_threshold": 7,
        "has_schedule": true
      }
    ],
    "total": 5
  }
}
```

---

#### GET /api/v1/families/:fid/medicines/:id — 药品详情

```
说明: 查看单个药品完整信息
权限: 该家庭成员
```

---

#### PUT /api/v1/families/:fid/medicines/:id — 编辑药品

```
说明: 修改药品信息
权限: 该家庭成员（非 elder 角色）
```

**Request:**
```json
{
  "name": "硝苯地平缓释片",
  "specification": "30mg×7片",
  "manufacturer": "拜耳医药",
  "image_url": "",
  "notes": "饭后服用"
}
```

---

#### DELETE /api/v1/families/:fid/medicines/:id — 删除药品

```
说明: 删除药品（已关联用药计划的药品无法删除）
权限: 该家庭成员（非 elder 角色）
```

**错误码:** 3003（已关联用药计划）

---

### 2.4 用药计划模块 — Schedule

#### POST /api/v1/families/:fid/schedules — 创建用药计划

```
说明: 为指定老人创建用药时间点及药品清单
权限: 该家庭成员（非 elder 角色）
```

**Request:**
```json
{
  "elder_user_id": 2,
  "time_of_day": "08:00",
  "items": [
    { "medicine_id": 1, "dosage": 1.00, "unit": "粒" },
    { "medicine_id": 2, "dosage": 0.50, "unit": "片" }
  ]
}
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "elder_user_id": 2,
    "elder_nickname": "张秀兰",
    "time_of_day": "08:00",
    "is_active": true,
    "items": [
      { "medicine_id": 1, "medicine_name": "硝苯地平缓释片", "dosage": 1.00, "unit": "粒" },
      { "medicine_id": 2, "medicine_name": "二甲双胍", "dosage": 0.50, "unit": "片" }
    ]
  }
}
```

---

#### GET /api/v1/families/:fid/schedules — 用药计划列表

```
说明: 查询家庭组内所有的用药计划
权限: 该家庭成员
查询参数: ?elder_user_id=2
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "elder_user_id": 2,
        "elder_nickname": "张秀兰",
        "time_of_day": "08:00",
        "is_active": true,
        "items": [
          { "medicine_id": 1, "medicine_name": "硝苯地平缓释片", "dosage": 1.00, "unit": "粒" }
        ]
      },
      {
        "id": 2,
        "elder_user_id": 2,
        "elder_nickname": "张秀兰",
        "time_of_day": "12:30",
        "is_active": true,
        "items": [
          { "medicine_id": 2, "medicine_name": "阿卡波糖", "dosage": 1.00, "unit": "粒" }
        ]
      }
    ]
  }
}
```

---

#### PUT /api/v1/families/:fid/schedules/:id — 修改用药计划

```
说明: 修改用药计划的时间或药品清单
权限: 该家庭成员（非 elder 角色）
注意: 修改后当天已过时间点不受影响，后续时间点按新计划生成提醒
```

**Request:**
```json
{
  "time_of_day": "08:30",
  "is_active": true,
  "items": [
    { "medicine_id": 1, "dosage": 1.00, "unit": "粒" }
  ]
}
```

---

#### DELETE /api/v1/families/:fid/schedules/:id — 删除用药计划

```
说明: 删除用药计划及关联的明细
权限: 该家庭成员（非 elder 角色）
注意: 不影响已生成的历史提醒记录
```

---

### 2.5 提醒确认模块 — Confirmation

#### GET /api/v1/reminders/today — 今日待确认列表

```
说明: 查询当前用户（老人）今日所有待确认的提醒
权限: 已登录
查询参数: ?family_id=1
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "date": "2026-07-06",
    "reminders": [
      {
        "id": 101,
        "remind_time": "2026-07-06T08:00:00Z",
        "time_label": "早上 08:00",
        "status": "sent",
        "items": [
          { "id": 1001, "medicine_name": "硝苯地平缓释片", "dosage": 1.00, "unit": "粒", "taken": true, "taken_at": "..." },
          { "id": 1002, "medicine_name": "二甲双胍", "dosage": 0.50, "unit": "片", "taken": true, "taken_at": "..." }
        ]
      },
      {
        "id": 102,
        "remind_time": "2026-07-06T12:30:00Z",
        "time_label": "中午 12:30",
        "status": "sent",
        "items": [
          { "id": 1003, "medicine_name": "阿卡波糖", "dosage": 1.00, "unit": "粒", "taken": null }
        ]
      }
    ]
  }
}
```

---

#### POST /api/v1/reminders/:id/confirm — 确认已服用

```
说明: 老人确认已服用本次提醒的全部药品
权限: 该提醒的 elder_user_id 本人
```

**Request:**
```json
{
  "item_ids": [1003]
}
```

**Response:**
```json
{
  "code": 0,
  "message": "已确认",
  "data": {
    "can_revoke_until": "2026-07-06T12:35:00Z"
  }
}
```

**逻辑说明:**
- 将指定 `reminder_items` 的 `taken` 设为 `true`，记录 `taken_at`
- 对应药品余量自动扣减
- 如果所有 items 都已确认，该 reminder 状态变为 `confirmed`
- 5 分钟内可撤销

---

#### POST /api/v1/reminders/:id/miss — 标记未服用

```
说明: 老人标记未服用，并选择原因
权限: 该提醒的 elder_user_id 本人
```

**Request:**
```json
{
  "reason": "忘记了"
}
```

**预设原因:** `忘记了` / `不舒服` / `药没了` / `其他`

---

#### POST /api/v1/reminders/:id/skip — 跳过本次用药

```
说明: 老人跳过本次用药（如医生让停药）
权限: 该提醒的 elder_user_id 本人
```

**Request:**
```json
{
  "reason": "医生让停药"
}
```

**预设原因:** `医生让停药` / `今天外出` / `其他`

---

#### POST /api/v1/reminders/:id/revoke — 撤销确认

```
说明: 撤销已确认的用药，回退余量
权限: 该提醒的 elder_user_id 本人
限制: 确认后 5 分钟内可撤销
```

**错误:** 超过 5 分钟返回错误 `「已超过撤销时间」`

---

#### GET /api/v1/reminders/history — 用药历史

```
说明: 查询指定老人某日的用药历史记录，支持日历视图
权限: 该家庭成员
查询参数: ?family_id=1&elder_user_id=2&date=2026-07-07
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "date": "2026-07-07",
    "elder": { "user_id": 2, "nickname": "张秀兰" },
    "slots": [
      {
        "reminder_id": 101,
        "time_of_day": "08:00",
        "time_label": "早上 08:00",
        "status": "confirmed",
        "status_label": "✅ 08:03 已服用",
        "items": [
          { "medicine_name": "硝苯地平缓释片", "dosage": 1.00, "unit": "粒", "taken": true, "taken_at": "2026-07-07T08:03:00Z" }
        ]
      }
    ],
    "summary": { "total": 3, "confirmed": 2, "missed": 1, "completion_rate": 0.67 }
  }
}
```

---

### 2.6 看板模块 — Dashboard

#### GET /api/v1/dashboard/today — 今日用药状态

```
说明: 子女端查看指定老人今日的用药完成情况
权限: 该家庭成员
查询参数: ?family_id=1&elder_user_id=2
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "elder": {
      "user_id": 2,
      "nickname": "张秀兰"
    },
    "date": "2026-07-06",
    "summary": {
      "total": 3,
      "confirmed": 2,
      "pending": 1,
      "missed": 0,
      "skipped": 0,
      "completion_rate": 0.67
    },
    "slots": [
      {
        "reminder_id": 101,
        "time_of_day": "08:00",
        "time_label": "早上",
        "status": "confirmed",
        "status_label": "✅ 08:03 已服用",
        "items": [
          { "medicine_name": "硝苯地平缓释片", "dosage": 1.00, "unit": "粒", "taken": true }
        ]
      },
      {
        "reminder_id": 102,
        "time_of_day": "12:30",
        "time_label": "中午",
        "status": "pending",
        "status_label": "⏳ 等待中",
        "items": [
          { "medicine_name": "阿卡波糖", "dosage": 1.00, "unit": "粒", "taken": null }
        ]
      },
      {
        "reminder_id": 103,
        "time_of_day": "18:00",
        "time_label": "晚上",
        "status": "pending",
        "status_label": "⏳ 等待中",
        "items": [
          { "medicine_name": "阿托伐他汀", "dosage": 1.00, "unit": "粒", "taken": null },
          { "medicine_name": "阿司匹林", "dosage": 1.00, "unit": "粒", "taken": null }
        ]
      }
    ]
  }
}
```

**状态颜色映射（前端使用）:**
| status | 颜色 | 图标 | 含义 |
|--------|------|------|------|
| confirmed | 绿色 | ✅ | 已确认 |
| pending | 黄色 | ⏳ | 等待中（还未到时间） |
| sent | 黄色 | ⏳ | 已推送，等待确认 |
| missed | 红色 | 🔴 | 超时未确认 |
| skipped | 蓝色 | ⏭️ | 跳过 |

---

#### GET /api/v1/dashboard/overview — 家庭用药概览

```
说明: 子女端首页，展示所有老人今日的用药状态摘要
权限: 该家庭成员
查询参数: ?family_id=1
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "family_name": "张秀兰的家庭",
    "elders": [
      {
        "user_id": 2,
        "nickname": "张秀兰",
        "avatar_url": "...",
        "overall_status": "warning",
        "summary": "2/3 已确认，中午 12:30 的药还未确认",
        "confirmed": 2,
        "total": 3,
        "next_reminder": {
          "time_of_day": "18:00",
          "time_label": "今晚 18:00",
          "medicines": "阿托伐他汀 1粒, 阿司匹林 1粒"
        }
      }
    ]
  }
}
```

**overall_status 定义:**
| 值 | 含义 | 触发条件 |
|------|------|---------|
| normal | 正常 | 全部已确认或等待中（未超时） |
| warning | 警告 | 存在超时但未超过 60 分钟 |
| danger | 危险 | 存在超时超过 60 分钟 |

---

### 2.7 余量管理模块 — Stock（P1）

#### GET /api/v1/families/:fid/stocks — 药品余量列表

```
说明: 查询家庭组内所有药品的余量状态
权限: 该家庭成员
查询参数: ?status=low （可选：仅显示低余量）
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "medicine_id": 1,
        "medicine_name": "硝苯地平缓释片",
        "remaining_count": 5,
        "total_count": 28,
        "alert_threshold": 7,
        "status": "low",
        "status_label": "⚠️ 即将用完（剩余 5 粒）",
        "estimated_days_left": 2.5
      },
      {
        "medicine_id": 2,
        "medicine_name": "二甲双胍",
        "remaining_count": 0,
        "total_count": 30,
        "alert_threshold": 10,
        "status": "empty",
        "status_label": "🚫 已用完"
      }
    ]
  }
}
```

---

#### PUT /api/v1/families/:fid/stocks/:id — 手动校准余量

```
说明: 子女补充药品后手动更新余量
权限: 该家庭成员（非 elder 角色）
```

**Request:**
```json
{
  "remaining_count": 28,
  "total_count": 28
}
```

---

### 2.8 告警模块 — Alert（内部调度 + 查询）

> 告警由后端定时任务自动触发，此模块仅提供查询接口。

#### GET /api/v1/alerts — 告警记录

```
说明: 查询家庭组的告警历史
权限: 该家庭成员
查询参数: ?family_id=1&elder_user_id=2&date=2026-07-06
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "reminder_id": 102,
        "alert_level": 1,
        "alert_label": "⚠️ 30分钟未确认",
        "remind_time": "2026-07-06T12:30:00Z",
        "alert_time": "2026-07-06T13:00:00Z",
        "medicines": "阿卡波糖 1粒",
        "resolved": false,
        "resolved_at": null
      }
    ]
  }
}
```

---

### 2.9 用户设置模块 — User

#### GET /api/v1/user/profile — 获取个人信息

```
说明: 获取当前登录用户的基本信息
权限: 已登录
```

---

#### PUT /api/v1/user/profile — 更新个人信息

```
说明: 更新昵称、头像等
权限: 已登录
```

---

#### PUT /api/v1/user/phone — 绑定手机号

```
说明: 微信手机号快速验证组件获取手机号
权限: 已登录
```

**Request:**
```json
{
  "encrypted_data": "...",
  "iv": "..."
}
```

---

### 2.10 文件上传模块 — Upload

#### POST /api/v1/upload — 通用文件上传

```
说明: 上传药品照片等文件，返回文件URL
权限: 已登录
Content-Type: multipart/form-data
```

**Request:**
```
字段名: file
文件类型: image/jpeg, image/png
最大大小: 10MB
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "url": "https://cdn.example.com/uploads/abc123.jpg",
    "filename": "medicine-photo.jpg",
    "size": 204800
  }
}
```

**错误码:** 1003（文件类型不支持）、1003（文件超过大小限制）

---

## 3. 接口依赖关系

```
登录流程:
  wx.login() → POST /auth/login → JWT Token

首次设置流程:
  POST /families → POST /families/:id/join（老人侧）
  → POST /families/:fid/medicines（多次）
  → POST /families/:fid/schedules（多次）

每日核心流程:
  [定时任务] 生成 reminders → 推送微信消息
  → GET /reminders/today（老人查看）
  → POST /reminders/:id/confirm（老人确认）
  → GET /dashboard/today（子女查看）
  → [定时任务] 检测超时 → 触发告警

余量管理流程:
  POST /reminders/:id/confirm → 自动扣减 stock
  → GET /families/:fid/stocks → 查看余量
  → PUT /families/:fid/stocks/:id → 手动校准
```

---

> 📋 **API 设计完成。共 9 个模块、28 个接口。等待 CEO 审阅。**
