-- 003: Create family_members table
-- Association table linking users to families with role-based access control.

CREATE TABLE IF NOT EXISTS family_members (
    id           BIGINT UNSIGNED  NOT NULL AUTO_INCREMENT  COMMENT '关联ID',
    family_id    BIGINT UNSIGNED  NOT NULL                 COMMENT '家庭组ID',
    user_id      BIGINT UNSIGNED  NOT NULL                 COMMENT '用户ID',
    role         VARCHAR(16)      NOT NULL DEFAULT 'member' COMMENT '角色: owner/admin/member/elder',
    joined_at    DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '加入时间',
    created_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP  COMMENT '创建时间',
    updated_at   DATETIME         NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP  COMMENT '更新时间',
    deleted_at   DATETIME         NULL     DEFAULT NULL    COMMENT '软删除（退出家庭组=软删除此记录）',
    PRIMARY KEY (id),
    UNIQUE KEY uk_family_user (family_id, user_id, deleted_at),
    KEY idx_user_id (user_id),
    KEY idx_family_role (family_id, role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='家庭成员关联表';
