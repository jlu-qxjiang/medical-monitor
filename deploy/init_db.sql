-- 数据库初始化脚本（服务器上执行：mysql -uroot -p < init_db.sql）
-- 把两处 'CHANGE_ME_DB_PASSWORD' 换成强密码，并与 .env.production 的 MEDICAL_DB_PASSWORD 保持一致

CREATE DATABASE IF NOT EXISTS medical_monitor
  DEFAULT CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

CREATE USER IF NOT EXISTS 'medical'@'localhost' IDENTIFIED BY 'CHANGE_ME_DB_PASSWORD';
GRANT ALL PRIVILEGES ON medical_monitor.* TO 'medical'@'localhost';
FLUSH PRIVILEGES;
