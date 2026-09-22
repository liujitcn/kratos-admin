package biz

import (
	"context"
	"database/sql"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestUpdatePrimaryLanguagePreservesProtectedFields 验证主语言可更新展示信息且不会修改代码和启用状态。
func TestUpdatePrimaryLanguagePreservesProtectedFields(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var connection *sql.DB
	connection, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err = connection.Close(); err != nil {
			t.Error(err)
		}
	})
	if err = db.AutoMigrate(&models.BaseLanguage{}); err != nil {
		t.Fatal(err)
	}
	primary := &models.BaseLanguage{ID: 10, LanguageCode: "zh-CN", LanguageName: "中文（简体）", NativeName: "简体中文", Sort: 10, IsPrimary: true, Status: int32(commonv1.Status_STATUS_ENABLE)}
	if err = db.Create(primary).Error; err != nil {
		t.Fatal(err)
	}
	var store *data.Data
	store, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	languageCase := NewBaseLanguageCase(&biz.BaseCase{}, data.NewTransaction(store), data.NewBaseLanguageRepository(store))
	req := &adminv1.BaseLanguageForm{Id: primary.ID, LanguageCode: "en-US", LanguageName: "简体中文", NativeName: "简体中文", Sort: 20, Status: commonv1.Status_STATUS_DISABLE}
	if err = languageCase.UpdateBaseLanguage(context.Background(), req); err != nil {
		t.Fatalf("更新主语言展示信息失败: %v", err)
	}
	var updated models.BaseLanguage
	if err = db.First(&updated, primary.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.LanguageName != req.LanguageName || updated.Sort != req.Sort {
		t.Fatalf("主语言展示信息未更新: %+v", updated)
	}
	if updated.LanguageCode != primary.LanguageCode {
		t.Fatalf("主语言代码被修改: %s", updated.LanguageCode)
	}
	if !updated.IsPrimary || updated.Status != int32(commonv1.Status_STATUS_ENABLE) {
		t.Fatalf("主语言保护字段被修改: %+v", updated)
	}
}
