-- 001: Create users table
-- Stores WeChat user information and authentication data.

CREATE TABLE IF NOT EXISTS users (
    id           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '用户ID',
    openid       VARCHAR(64)      NOT NULL                 COMMENT '微信OpenID（唯一标识）',
    unionid      VARCHAR(64)      NULL     DEFAULT NULL    COMMENT '微信UnionID（跨应用标识，预留）',
    nickname     VARCHAR(64)      NOT NULL DEFAULT ''      COMMENT '微信昵称',
    avatar_url   VARCHAR(512)     NOT NULL DEFAULT ''      COMMENT '微信头像URL',
    phone        VARCHAR(20)      NOT NULL DEFAULT ''      COMMENT '手机号（脱敏存储：仅存后4位）',
    phone_enc    VARCHAR(256)     NULL     DEFAULT NULL    COMMENT '手机号AES加密完整值（用于短信发送）',
    created_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '注册时间',
    updated_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    deleted_at   DATETIME         NULL     DEFAULT NULL    COMMENT '软删除',
    PRIMARY KEY (id),
    UNIQUE KEY uk_openid (openid),
    KEY idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户表';
