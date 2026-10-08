-- 008: Create reminder_items table
-- Stores snapshots of medicine information at the time of reminder generation.
-- These snapshots do not change when the original medicine or schedule is modified.

CREATE TABLE IF NOT EXISTS reminder_items (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '明细ID',
    reminder_id     BIGINT UNSIGNED  NOT NULL                 COMMENT '提醒ID',
    medicine_id     BIGINT UNSIGNED  NOT NULL                 COMMENT '药品ID',
    medicine_name   VARCHAR(128)     NOT NULL                 COMMENT '药品名称快照',
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
