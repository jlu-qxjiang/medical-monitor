package main

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	db, _ := gorm.Open(mysql.Open("root:123456@tcp(127.0.0.1:3306)/medical_monitor?charset=utf8mb4&parseTime=True"), &gorm.Config{})

	tables := []string{"users", "families", "family_members", "medicines", "medicine_schedules", "medicine_schedule_items", "reminders", "reminder_items", "medicine_stocks", "audit_logs", "refresh_tokens"}
	for _, t := range tables {
		var count int64
		db.Raw("SELECT COUNT(*) FROM " + t).Scan(&count)
		fmt.Printf("%-30s %d 条\n", t, count)
	}
}
