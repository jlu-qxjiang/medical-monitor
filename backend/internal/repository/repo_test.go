package repository

import (
	"fmt"

	"github.com/medical-monitor/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const testDSN = "root:123456@tcp(127.0.0.1:3306)/?charset=utf8mb4&parseTime=True&loc=Local"
const testDBName = "medical_monitor_test"

// setupTestDB 创建独立测试数据库，每次全新迁移。
func setupTestDB(t interface{ Fatal(args ...interface{}) }) *gorm.DB {
	rootDB, err := gorm.Open(mysql.Open(testDSN), &gorm.Config{})
	if err != nil {
		t.Fatal("连接 MySQL 失败:", err)
	}
	rootDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", testDBName))
	rootDB.Exec(fmt.Sprintf("CREATE DATABASE `%s` DEFAULT CHARSET utf8mb4", testDBName))

	dsn := fmt.Sprintf("root:123456@tcp(127.0.0.1:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local", testDBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal("failed to open test db:", err)
	}
	// 建表
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
		&model.RefreshToken{},
		&model.PersonalMedicine{},
		&model.PersonalMedicineStock{},
		&model.PersonalMedicineSchedule{},
		&model.PersonalMedicineScheduleTime{},
	); err != nil {
		t.Fatal("failed to migrate:", err)
	}
	return db
}
