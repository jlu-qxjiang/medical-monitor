-- 007: Create reminders table
-- Stores daily generated reminder records for medication events.

CREATE TABLE IF NOT EXISTS reminders (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '提醒ID',
    schedule_id     BIGINT UNSIGNED  NOT NULL                 COMMENT '关联的用药计划ID',
    family_id       BIGINT UNSIGNED  NOT NULL                 COMMENT '所属家庭组ID',
    elder_user_id   BIGINT UNSIGNED  NOT NULL                 COMMENT '服药老人用户ID',
    remind_time     DATETIME         NOT NULL                 COMMENT '提醒时间',
    remind_date     DATE             NOT NULL                 COMMENT '提醒日期（冗余，方便按日查询）',
    status          VARCHAR(16)      NOT NULL DEFAULT 'pending' COMMENT '状态: pending/sent/confirmed/missed/skipped',
    wechat_msg_id   VARCHAR(128)     NOT NULL DEFAULT ''      COMMENT '微信订阅消息ID',
    sent_at         DATETIME         NULL     DEFAULT NULL    COMMENT '消息推送时间',
    confirmed_at    DATETIME         NULL     DEFAULT NULL    COMMENT '确认时间',
    confirmed_by    BIGINT UNSIGNED  NULL     DEFAULT NULL    COMMENT '确认人用户ID',
    alert_level     TINYINT          NOT NULL DEFAULT 0       COMMENT '告警级别: 0=正常 1=30min 2=60min 3=90min',
    created_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    PRIMARY KEY (id),
    KEY idx_remind_time_status (remind_time, status),
    KEY idx_elder_date (elder_user_id, remind_date),
    KEY idx_family_date (family_id, remind_date),
    KEY idx_schedule_id (schedule_id),
    KEY idx_status_alert (status, alert_level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='提醒记录表';
