# 数据库设计 —「按时吃药」

> 版本: v1.1 | 作者: 张架构 | 日期: 2026-07-07

---

## 1. ER 关系概述

```
users  1──N  family_members  N──1  families
                 │
                 │ (通过 family_id 关联)
                 │
medicines  N──1  families
     │
     │ 1──N  medicine_schedule_items  N──1  medicine_schedules
     │                                            │
     │                                        families (N──1)
     │
     │ 1──N  reminder_items  N──1  reminders
     │                                    │
     │                              medicine_schedules (N──1)
     │                              users (elder, N──1)
     │
     │ 1──N  medicine_stocks

简化关系:

users ──┬── family_members ── families
        │
        └── reminders (elder)

families ──┬── medicines
           ├── medicine_schedules
           ├── reminders
           └── medicine_stocks

medicines ──┬── medicine_schedule_items
            ├── reminder_items
            └── medicine_stocks

medicine_schedules ──┬── medicine_schedule_items
                     └── reminders
```

---

## 2. 完整 DDL

### 2.1 通用字段规范

所有业务表包含以下通用字段：

```sql
-- 每条记录必须包含:
created_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间'
updated_at  DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间'
deleted_at  DATETIME     NULL     DEFAULT NULL                COMMENT '软删除时间 (NULL=未删除)'

-- 索引规范:
-- 1. 每个表必须有主键 id (BIGINT UNSIGNED AUTO_INCREMENT)
-- 2. deleted_at 参与唯一索引（允许同名数据在不同时间被删除后重建）
-- 3. 外键关系字段建普通索引
-- 4. 查询频率高的 WHERE 条件字段建联合索引
```

---

### 2.2 users — 用户表

```sql
CREATE TABLE users (
    id           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '用户ID',
    openid       VARCHAR(64)      NOT NULL                 COMMENT '微信OpenID（唯一标识）',
    unionid      VARCHAR(64)      NULL     DEFAULT NULL    COMMENT '微信UnionID（跨应用标识，预留）',
    nickname     VARCHAR(64)      NOT NULL DEFAULT ''      COMMENT '微信昵称',
    avatar_url   VARCHAR(512)     NOT NULL DEFAULT ''      COMMENT '微信头像URL',
    phone        VARCHAR(20)      NOT NULL DEFAULT ''      COMMENT '手机号（生产环境仅存后4位脱敏；开发环境存完整号）',
    phone_enc    VARCHAR(256)     NULL     DEFAULT NULL    COMMENT '手机号AES加密完整值（用于短信发送）',
    password_hash VARCHAR(256)    NOT NULL DEFAULT ''      COMMENT 'bcrypt密码哈希（Web开发阶段使用，小程序上线后废弃）',
    created_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '注册时间',
    updated_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    deleted_at   DATETIME         NULL     DEFAULT NULL    COMMENT '软删除',
    PRIMARY KEY (id),
    UNIQUE KEY uk_openid (openid),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';
```

---

### 2.3 families — 家庭组表

```sql
CREATE TABLE families (
    id           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '家庭组ID',
    name         VARCHAR(64)      NOT NULL                 COMMENT '家庭组名称（如「张秀兰的家庭」）',
    invite_code  VARCHAR(8)       NOT NULL                 COMMENT '6位数字邀请码',
    created_by   BIGINT UNSIGNED  NOT NULL                 COMMENT '创建者用户ID',
    created_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    deleted_at   DATETIME         NULL     DEFAULT NULL    COMMENT '软删除',
    PRIMARY KEY (id),
    UNIQUE KEY uk_invite_code (invite_code),
    KEY idx_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='家庭组表';
```

---

### 2.4 family_members — 家庭成员关联表

```sql
CREATE TABLE family_members (
    id           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '关联ID',
    family_id    BIGINT UNSIGNED  NOT NULL                 COMMENT '家庭组ID',
    user_id      BIGINT UNSIGNED  NOT NULL                 COMMENT '用户ID',
    role         VARCHAR(16)      NOT NULL DEFAULT 'member' COMMENT '角色: owner=创建者 admin=管理员 member=子女 elder=老人',
    joined_at    DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '加入时间',
    created_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    deleted_at   DATETIME         NULL     DEFAULT NULL    COMMENT '软删除（退出家庭组=软删除此记录）',
    PRIMARY KEY (id),
    UNIQUE KEY uk_family_user (family_id, user_id, deleted_at),
    KEY idx_user_id (user_id),
    KEY idx_family_role (family_id, role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='家庭成员关联表';
```

---

### 2.5 medicines — 药品表

```sql
CREATE TABLE medicines (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '药品ID',
    family_id       BIGINT UNSIGNED  NOT NULL                 COMMENT '所属家庭组ID',
    name            VARCHAR(128)     NOT NULL                 COMMENT '药品名称（如「硝苯地平缓释片」）',
    specification   VARCHAR(128)     NOT NULL DEFAULT ''      COMMENT '规格（如「30mg×7片」）',
    manufacturer    VARCHAR(128)     NOT NULL DEFAULT ''      COMMENT '生产厂家',
    image_url       VARCHAR(512)     NOT NULL DEFAULT ''      COMMENT '药品照片URL（OCR 拍摄的原图）',
    notes           VARCHAR(256)     NOT NULL DEFAULT ''      COMMENT '备注（如「饭后服用」）',
    -- 第三方药品库扩展字段（MVP 阶段暂不使用，预留对接能力）
    external_code   VARCHAR(64)      NOT NULL DEFAULT ''      COMMENT '第三方药品编码（预留，如 ypk 药品ID）',
    external_source VARCHAR(32)      NOT NULL DEFAULT ''      COMMENT '第三方来源标识（预留，如 ypk/39health）',
    created_by      BIGINT UNSIGNED  NOT NULL                 COMMENT '录入者用户ID',
    created_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    deleted_at      DATETIME         NULL     DEFAULT NULL    COMMENT '软删除',
    PRIMARY KEY (id),
    KEY idx_family_id (family_id),
    KEY idx_family_name (family_id, name),
    KEY idx_external (external_source, external_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='药品表';
```

---

### 2.6 medicine_schedules — 用药计划（时间槽）

```sql
CREATE TABLE medicine_schedules (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '计划ID',
    family_id       BIGINT UNSIGNED  NOT NULL                 COMMENT '所属家庭组ID',
    elder_user_id   BIGINT UNSIGNED  NOT NULL                 COMMENT '服药老人用户ID',
    time_of_day     TIME             NOT NULL                 COMMENT '每日用药时间（如 08:00:00）',
    is_active       TINYINT(1)       NOT NULL DEFAULT 1       COMMENT '是否启用（1=启用 0=暂停）',
    created_by      BIGINT UNSIGNED  NOT NULL                 COMMENT '创建者用户ID',
    created_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    deleted_at      DATETIME         NULL     DEFAULT NULL    COMMENT '软删除',
    PRIMARY KEY (id),
    KEY idx_family_elder (family_id, elder_user_id),
    KEY idx_time_active (time_of_day, is_active),
    KEY idx_elder_active (elder_user_id, is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用药计划表（时间槽）';
```

---

### 2.7 medicine_schedule_items — 用药计划明细

```sql
CREATE TABLE medicine_schedule_items (
    id           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '明细ID',
    schedule_id  BIGINT UNSIGNED  NOT NULL                 COMMENT '用药计划ID',
    medicine_id  BIGINT UNSIGNED  NOT NULL                 COMMENT '药品ID',
    dosage       DECIMAL(10,2)    NOT NULL                 COMMENT '每次剂量（如 1.00 表示 1 粒，0.50 表示半片）',
    unit         VARCHAR(16)      NOT NULL DEFAULT '粒'     COMMENT '剂量单位（粒/片/毫升/毫克）',
    created_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    deleted_at   DATETIME         NULL     DEFAULT NULL    COMMENT '软删除',
    PRIMARY KEY (id),
    KEY idx_schedule_id (schedule_id),
    KEY idx_medicine_id (medicine_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用药计划明细表';
```

---

### 2.8 reminders — 提醒记录表（每日生成）

```sql
CREATE TABLE reminders (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '提醒ID',
    schedule_id     BIGINT UNSIGNED  NOT NULL                 COMMENT '关联的用药计划ID',
    family_id       BIGINT UNSIGNED  NOT NULL                 COMMENT '所属家庭组ID',
    elder_user_id   BIGINT UNSIGNED  NOT NULL                 COMMENT '服药老人用户ID',
    remind_time     DATETIME         NOT NULL                 COMMENT '提醒时间（如 2026-07-07 08:00:00）',
    remind_date     DATE             NOT NULL                 COMMENT '提醒日期（冗余，方便按日查询）',
    status          VARCHAR(16)      NOT NULL DEFAULT 'pending' COMMENT '状态: pending=待提醒 sent=已推送 confirmed=已服用 missed=未服用 skipped=跳过',
    wechat_msg_id   VARCHAR(128)     NOT NULL DEFAULT ''      COMMENT '微信订阅消息ID（用于追踪送达）',
    sent_at         DATETIME         NULL     DEFAULT NULL    COMMENT '消息推送时间',
    confirmed_at    DATETIME         NULL     DEFAULT NULL    COMMENT '确认时间',
    confirmed_by    BIGINT UNSIGNED  NULL     DEFAULT NULL    COMMENT '确认人用户ID（老人本人或代确认的子女）',
    alert_level     TINYINT          NOT NULL DEFAULT 0       COMMENT '告警级别: 0=正常 1=30min超时 2=60min超时 3=90min高危',
    created_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    PRIMARY KEY (id),
    KEY idx_remind_time_status (remind_time, status),
    KEY idx_elder_date (elder_user_id, remind_date),
    KEY idx_family_date (family_id, remind_date),
    KEY idx_schedule_id (schedule_id),
    KEY idx_status_alert (status, alert_level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='提醒记录表';
```

---

### 2.9 reminder_items — 提醒明细（每次提醒包含的药品清单快照）

```sql
CREATE TABLE reminder_items (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '明细ID',
    reminder_id     BIGINT UNSIGNED  NOT NULL                 COMMENT '提醒ID',
    medicine_id     BIGINT UNSIGNED  NOT NULL                 COMMENT '药品ID',
    medicine_name   VARCHAR(128)     NOT NULL                 COMMENT '药品名称快照（生成提醒时的值，不随后续编辑变化）',
    dosage          DECIMAL(10,2)    NOT NULL                 COMMENT '剂量快照',
    unit            VARCHAR(16)      NOT NULL DEFAULT '粒'     COMMENT '单位快照',
    taken           TINYINT(1)       NULL     DEFAULT NULL    COMMENT '是否已服用（NULL=未操作 1=已服用 0=未服用/跳过）',
    taken_at        DATETIME         NULL     DEFAULT NULL    COMMENT '服用确认时间',
    skip_reason     VARCHAR(64)      NOT NULL DEFAULT ''      COMMENT '跳过/未服用原因',
    created_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    PRIMARY KEY (id),
    KEY idx_reminder_id (reminder_id),
    KEY idx_medicine_id (medicine_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='提醒明细表（药品快照）';
```

---

### 2.10 medicine_stocks — 药品余量表

```sql
CREATE TABLE medicine_stocks (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '余量ID',
    medicine_id     BIGINT UNSIGNED  NOT NULL                 COMMENT '药品ID',
    family_id       BIGINT UNSIGNED  NOT NULL                 COMMENT '所属家庭组ID',
    total_count     INT              NOT NULL DEFAULT 0       COMMENT '总数量（粒/片）',
    remaining_count INT              NOT NULL DEFAULT 0       COMMENT '剩余数量',
    alert_threshold INT              NOT NULL DEFAULT 10      COMMENT '预警阈值（剩余低于此值时触发预警）',
    last_refill_at  DATETIME         NULL     DEFAULT NULL    COMMENT '上次补充时间',
    created_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_medicine_id (medicine_id),
    KEY idx_family_id (family_id),
    KEY idx_remaining (remaining_count)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='药品余量表';
```

---

### 2.11 audit_logs — 操作审计日志

```sql
CREATE TABLE audit_logs (
    id           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '日志ID',
    user_id      BIGINT UNSIGNED  NULL                     COMMENT '操作用户ID（NULL=系统操作）',
    family_id    BIGINT UNSIGNED  NULL                     COMMENT '关联家庭组ID',
    action       VARCHAR(64)      NOT NULL                 COMMENT '操作类型（如 medicine.create / reminder.confirm / family.join）',
    target_type  VARCHAR(32)      NOT NULL                 COMMENT '操作目标类型（medicine / reminder / family / user）',
    target_id    BIGINT UNSIGNED  NULL                     COMMENT '操作目标ID',
    detail       JSON             NULL                     COMMENT '操作详情（JSON格式，敏感字段已脱敏）',
    ip           VARCHAR(64)      NOT NULL DEFAULT ''      COMMENT '操作IP',
    created_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '操作时间',
    PRIMARY KEY (id),
    KEY idx_user_action (user_id, action, created_at),
    KEY idx_target (target_type, target_id),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='操作审计日志表';
```

---

## 3. 索引设计说明

| 表 | 索引 | 类型 | 用途 |
|------|------|------|------|
| users | uk_openid | UNIQUE | 微信登录时根据 openid 查找用户 |
| families | uk_invite_code | UNIQUE | 邀请码加入时快速查找家庭组 |
| family_members | uk_family_user | UNIQUE | 防止用户重复加入同一家庭组 |
| family_members | idx_user_id | INDEX | 查询「我属于哪些家庭组」 |
| family_members | idx_family_role | INDEX | 查询「家庭组中所有老人」 |
| medicines | idx_family_name | INDEX | 按家庭组+药品名搜索 |
| medicine_schedules | idx_elder_active | INDEX | 定时任务扫描：找出所有启用的计划 |
| reminders | idx_remind_time_status | INDEX | 定时任务核心查询：找到当前需要推送的提醒 |
| reminders | idx_elder_date | INDEX | 子女看板：查询某老人今日用药状态 |
| reminder_items | idx_reminder_id | INDEX | 查询某次提醒包含的药品清单 |
| medicine_stocks | uk_medicine_id | UNIQUE | 一种药品只有一条余量记录 |

---

## 4. 数据迁移计划

```
Phase 1 (MVP):
  migrations/
  ├── 001_create_users.sql          -- 用户表
  ├── 002_create_families.sql       -- 家庭组表
  ├── 003_create_family_members.sql -- 成员关联表
  ├── 004_create_medicines.sql      -- 药品表
  ├── 005_create_schedules.sql      -- 用药计划表
  ├── 006_create_schedule_items.sql -- 用药计划明细表
  ├── 007_create_reminders.sql      -- 提醒记录表
  ├── 008_create_reminder_items.sql -- 提醒明细表
  ├── 009_create_stocks.sql         -- 余量表
  └── 010_create_audit_logs.sql     -- 审计日志表

Phase 2 (P1 迭代):
  ├── 011_create_medicine_templates.sql -- 药品模板库表（自建100种常见药）
  └── 012_seed_medicine_templates.sql   -- 预置药品数据

Phase 3 (P2 迭代):
  ├── 013_create_health_records.sql     -- 健康数据表
  └── 014_create_revisit_reminders.sql  -- 复诊提醒表
```

---

## 5. 数据安全设计

| 字段 | 策略 | 实现方式 |
|------|------|----------|
| phone（手机号） | 脱敏展示 + 加密存储 | 展示：`138****1234`；存储：`phone` 字段仅存后 4 位，`phone_enc` 字段存 AES-256-GCM 密文 |
| openid | 禁止通过 API 返回 | 仅后端内部使用，不在任何接口响应中出现 |
| 身份证号 | MVP 不收集 | 如需实名认证（语音电话业务要求），后续单独建表加密存储 |
| 用药数据 | 家庭组隔离 | 所有查询带 `family_id` 条件，Repository 层统一注入 |
| 操作日志 | JSON 脱敏 | `audit_logs.detail` 中敏感字段自动替换为 `***` |

---

> 📋 **数据库 Schema 设计完成。共 10 张表，覆盖 MVP 全部功能。张架构等待 CEO 审阅。**
