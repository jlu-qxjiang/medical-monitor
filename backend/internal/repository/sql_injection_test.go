package repository

import (
	"context"
	"testing"

	"github.com/medical-monitor/internal/model"
)

// TestSQLInjection_MedicineSearch 验证药品搜索接口的 SQL 注入防护。
// 核心断言：注入 payload 不能绕过 name 过滤，不能泄露/删除数据。
func TestSQLInjection_MedicineSearch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMedicineRepository(db)
	ctx := context.Background()

	// 准备数据：家庭 1 下两条正常药品
	famID := uint64(1)
	db.Create(&model.Medicine{FamilyID: famID, Name: "降压药", CreatedBy: 1})
	db.Create(&model.Medicine{FamilyID: famID, Name: "降糖药", CreatedBy: 1})

	// 注入 payload 列表
	payloads := []string{
		"' OR '1'='1",
		"' OR 1=1 --",
		"' UNION SELECT id,password FROM users --",
		"'; DROP TABLE medicines; --",
		"' AND (SELECT 1 FROM (SELECT SLEEP(5))a) --",
	}

	for _, payload := range payloads {
		meds, _, err := repo.FindMedicinesByFamilyID(ctx, famID, payload, 1, 20)
		if err != nil {
			// 报错也算安全（没有泄露数据），但参数化查询理论上不该报错
			t.Logf("payload %q 触发错误（安全，但需确认非注入）: %v", payload, err)
			continue
		}
		// 关键断言：注入 payload 不应匹配到任何正常药品
		if len(meds) != 0 {
			t.Errorf("SQL注入风险！payload %q 返回了 %d 条数据，注入可能生效", payload, len(meds))
		}
	}

	// 验证 medicines 表仍然存在且数据完好
	var count int64
	db.Model(&model.Medicine{}).Count(&count)
	if count != 2 {
		t.Errorf("medicines 表数据被破坏！预期 2 条，实际 %d 条", count)
	}
}

// TestSQLInjection_PersonalMedicineSearch 个人药品搜索注入防护。
func TestSQLInjection_PersonalMedicineSearch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMedicineRepository(db)
	ctx := context.Background()

	userID := uint64(1)
	db.Create(&model.PersonalMedicine{UserID: userID, Name: "维生素C"})

	payloads := []string{
		"' OR '1'='1",
		"' OR 1=1 --",
		"'; DROP TABLE personal_medicines; --",
	}

	for _, payload := range payloads {
		meds, err := repo.FindPersonalMedicines(ctx, userID, payload)
		if err != nil {
			t.Logf("payload %q 触发错误: %v", payload, err)
			continue
		}
		if len(meds) != 0 {
			t.Errorf("SQL注入风险！payload %q 返回了 %d 条数据", payload, len(meds))
		}
	}

	var count int64
	db.Model(&model.PersonalMedicine{}).Count(&count)
	if count != 1 {
		t.Errorf("personal_medicines 表数据被破坏！预期 1 条，实际 %d 条", count)
	}
}

// TestSQLInjection_KeywordLiteralMatch 验证正常关键词仍能精确匹配。
// 防止"防注入"过度，导致正常搜索也失效。
func TestSQLInjection_KeywordLiteralMatch(t *testing.T) {
	db := setupTestDB(t)
	repo := NewMedicineRepository(db)
	ctx := context.Background()

	famID := uint64(1)
	db.Create(&model.Medicine{FamilyID: famID, Name: "阿司匹林", CreatedBy: 1})

	// 正常关键词应命中
	meds, _, err := repo.FindMedicinesByFamilyID(ctx, famID, "阿司匹林", 1, 20)
	if err != nil {
		t.Fatal("正常搜索失败:", err)
	}
	if len(meds) != 1 {
		t.Errorf("正常关键词未命中，预期 1 条，实际 %d 条", len(meds))
	}
}
