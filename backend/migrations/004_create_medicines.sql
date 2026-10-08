-- 004: Create medicines table
-- Stores medicine information per family group.

CREATE TABLE IF NOT EXISTS medicines (
    id              BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '药品ID',
    family_id       BIGINT UNSIGNED  NOT NULL                 COMMENT '所属家庭组ID',
    name            VARCHAR(128)     NOT NULL                 COMMENT '药品名称',
    specification   VARCHAR(128)     NOT NULL DEFAULT ''      COMMENT '规格',
    manufacturer    VARCHAR(128)     NOT NULL DEFAULT ''      COMMENT '生产厂家',
    image_url       VARCHAR(512)     NOT NULL DEFAULT ''      COMMENT '药品照片URL',
    notes           VARCHAR(256)     NOT NULL DEFAULT ''      COMMENT '备注',
    external_code   VARCHAR(64)      NOT NULL DEFAULT ''      COMMENT '第三方药品编码（预留）',
    external_source VARCHAR(32)      NOT NULL DEFAULT ''      COMMENT '第三方来源标识（预留）',
    created_by      BIGINT UNSIGNED  NOT NULL                 COMMENT '录入者用户ID',
    created_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at      DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    deleted_at      DATETIME         NULL     DEFAULT NULL    COMMENT '软删除',
    PRIMARY KEY (id),
    KEY idx_family_id (family_id),
    KEY idx_family_name (family_id, name),
    KEY idx_external (external_source, external_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='药品表';
