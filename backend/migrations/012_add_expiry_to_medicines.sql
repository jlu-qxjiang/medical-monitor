-- 为药品表添加有效期字段
ALTER TABLE `medicines` ADD COLUMN `expiry` VARCHAR(10) NOT NULL DEFAULT '' COMMENT '有效期（YYYY-MM-DD）' AFTER `notes`;
ALTER TABLE `personal_medicines` ADD COLUMN `expiry` VARCHAR(10) NOT NULL DEFAULT '' COMMENT '有效期（YYYY-MM-DD）' AFTER `notes`;
