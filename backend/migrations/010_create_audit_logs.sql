-- 010: Create audit_logs table
-- Stores operational audit trails for compliance and debugging.

CREATE TABLE IF NOT EXISTS audit_logs (
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
