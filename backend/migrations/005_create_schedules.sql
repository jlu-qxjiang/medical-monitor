-- 005: Create medicine_schedules table
-- Stores medication schedules (time slots) for elders.

CREATE TABLE IF NOT EXISTS medicine_schedules (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '计划ID',
    family_id       BIGINT UNSIGNED  NOT NULL                 COMMENT '所属家庭组ID',
    elder_user_id   BIGINT UNSIGNED  NOT NULL                 COMMENT '服药老人用户ID',
    time_of_day     TIME             NOT NULL                 COMMENT '每日用药时间',
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
