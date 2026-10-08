-- 002: Create families table
-- Stores family groups that connect users together for medication management.

CREATE TABLE IF NOT EXISTS families (
    id           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '家庭组ID',
    name         VARCHAR(64)      NOT NULL                 COMMENT '家庭组名称',
    invite_code  VARCHAR(8)       NOT NULL                 COMMENT '6位数字邀请码',
    created_by   BIGINT UNSIGNED  NOT NULL                 COMMENT '创建者用户ID',
    created_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    deleted_at   DATETIME         NULL     DEFAULT NULL    COMMENT '软删除',
    PRIMARY KEY (id),
    UNIQUE KEY uk_invite_code (invite_code),
    KEY idx_created_by (created_by)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='家庭组表';
