-- 011_create_refresh_tokens.sql
-- Refresh Token 持久化存储，解决 Redis 重启后 Token 无法校验的问题

CREATE TABLE refresh_tokens (
    id            BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '记录ID',
    user_id       BIGINT UNSIGNED  NOT NULL                 COMMENT '用户ID',
    token_hash    VARCHAR(128)     NOT NULL                 COMMENT 'Refresh Token 哈希值（SHA256）',
    expires_at    DATETIME         NOT NULL                 COMMENT '过期时间',
    revoked_at    DATETIME         NULL     DEFAULT NULL    COMMENT '撤销时间（登出时设置）',
    created_at    DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_token_hash (token_hash),
    KEY idx_user_id (user_id),
    KEY idx_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Refresh Token 持久化表';
