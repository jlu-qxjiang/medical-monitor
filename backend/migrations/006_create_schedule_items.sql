-- 006: Create medicine_schedule_items table
-- Stores the detailed medicine list within each schedule time slot.

CREATE TABLE IF NOT EXISTS medicine_schedule_items (
    id           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '明细ID',
    schedule_id  BIGINT UNSIGNED  NOT NULL                 COMMENT '用药计划ID',
    medicine_id  BIGINT UNSIGNED  NOT NULL                 COMMENT '药品ID',
    dosage       DECIMAL(10,2)    NOT NULL                 COMMENT '每次剂量',
    unit         VARCHAR(16)      NOT NULL DEFAULT '粒'     COMMENT '剂量单位（粒/片/毫升/毫克）',
    created_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    deleted_at   DATETIME         NULL     DEFAULT NULL    COMMENT '软删除',
    PRIMARY KEY (id),
    KEY idx_schedule_id (schedule_id),
    KEY idx_medicine_id (medicine_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用药计划明细表';
