-- 009: Create medicine_stocks table
-- Tracks remaining quantities of medicines per family.

CREATE TABLE IF NOT EXISTS medicine_stocks (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '余量ID',
    medicine_id     BIGINT UNSIGNED  NOT NULL                 COMMENT '药品ID',
    family_id       BIGINT UNSIGNED  NOT NULL                 COMMENT '所属家庭组ID',
    total_count     INT              NOT NULL DEFAULT 0       COMMENT '总数量（粒/片）',
    remaining_count INT              NOT NULL DEFAULT 0       COMMENT '剩余数量',
    alert_threshold INT              NOT NULL DEFAULT 10      COMMENT '预警阈值',
    last_refill_at  DATETIME         NULL     DEFAULT NULL    COMMENT '上次补充时间',
    created_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_medicine_id (medicine_id),
    KEY idx_family_id (family_id),
    KEY idx_remaining (remaining_count)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='药品余量表';
