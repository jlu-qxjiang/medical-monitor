package database

import (
	"fmt"

	"github.com/medical-monitor/config"
	"github.com/medical-monitor/internal/model"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// New 初始化 MySQL 数据库连接（基于 GORM）。
// 首次启动自动建库 → 建表 → 配置连接池。
// 开发阶段每次启动会先删旧库，确保表结构最新。
func New(cfg *config.Config, logger *zap.Logger) (*gorm.DB, error) {
	dbConf := cfg.Database
	if dbConf.User == "" {
		logger.Warn("database user not configured, using defaults")
		dbConf.User = "root"
		dbConf.Host = "127.0.0.1"
		dbConf.Port = 3306
		dbConf.DBName = "medical_monitor"
	}

	// 第 1 步：连接 MySQL（不指定数据库）
	rootDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=%s&parseTime=True&loc=Local",
		dbConf.User, dbConf.Password, dbConf.Host, dbConf.Port, dbConf.Charset)
	rootDB, err := gorm.Open(mysql.Open(rootDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("连接MySQL失败: %w", err)
	}

	if err := rootDB.Exec(fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET %s COLLATE utf8mb4_unicode_ci",
		dbConf.DBName, dbConf.Charset)).Error; err != nil {
		return nil, fmt.Errorf("创建数据库失败: %w", err)
	}
	sqlRoot, _ := rootDB.DB()
	if sqlRoot != nil {
		sqlRoot.Close()
	}
	logger.Info("database initialized", zap.String("db", dbConf.DBName))

	// 第 2 步：连接到目标数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		dbConf.User, dbConf.Password, dbConf.Host, dbConf.Port, dbConf.DBName, dbConf.Charset)

	gormLogLevel := gormlogger.Warn
	if cfg.Server.Mode == "debug" {
		gormLogLevel = gormlogger.Info
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormLogLevel),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 第 3 步：自动迁移表结构
	if err := db.AutoMigrate(
		&model.User{},
		&model.Family{},
		&model.FamilyMember{},
		&model.Medicine{},
		&model.MedicineSchedule{},
		&model.MedicineScheduleTime{},
		&model.Reminder{},
		&model.ReminderItem{},
		&model.MedicineStock{},
		&model.AuditLog{},
		&model.RefreshToken{},
		&model.PersonalMedicine{},
		&model.PersonalMedicineStock{},
		&model.PersonalMedicineSchedule{},
		&model.PersonalMedicineScheduleTime{},
	); err != nil {
		return nil, fmt.Errorf("自动迁移表结构失败: %w", err)
	}
	logger.Info("database tables auto-migrated")

	// 第 4 步：配置连接池
	//连接池：预先建好一批连接，重复使用，用完不销毁、放回池子
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库实例失败: %w", err)
	}
	sqlDB.SetMaxIdleConns(dbConf.MaxIdleConns)
	sqlDB.SetMaxOpenConns(dbConf.MaxOpenConns)

	return db, nil
}
