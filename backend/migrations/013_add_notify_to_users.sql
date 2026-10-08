-- 用户提醒设置字段
ALTER TABLE `users` ADD COLUMN `notify_reminder` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '用药提醒开关' AFTER `role`;
ALTER TABLE `users` ADD COLUMN `notify_stock` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '库存提醒开关' AFTER `notify_reminder`;
