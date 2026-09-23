package biz

import (
	"context"
	"database/sql"
	"testing"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	corebiz "github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/cache/memory"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestSaveBaseI18nReusesCallerTransaction 验证主记录与翻译保存复用调用方事务，避免嵌套事务等待连接。
func TestSaveBaseI18nReusesCallerTransaction(t *testing.T) {
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
	if err = db.AutoMigrate(&models.BaseLanguage{}, &models.BaseI18N{}, &models.BaseDict{}); err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&models.BaseLanguage{LanguageCode: "zh-CN", LanguageName: "中文", NativeName: "中文", IsPrimary: true, Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	dict := &models.BaseDict{ID: 100, Code: "test", Name: "旧名称", Status: 1}
	if err = db.Create(dict).Error; err != nil {
		t.Fatal(err)
	}

	var store *data.Data
	store, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	tx := data.NewTransaction(store)
	baseCase := &corebiz.BaseCase{}
	languageCase := NewBaseLanguageCase(baseCase, tx, data.NewBaseLanguageRepository(store))
	i18nCase := NewBaseI18nCase(baseCase, data.NewBaseI18NRepository(store), languageCase)
	dictCase := NewBaseDictCase(baseCase, tx, data.NewBaseDictRepository(store), nil, i18nCase)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = dictCase.UpdateBaseDict(ctx, &adminv1.BaseDictForm{Id: dict.ID, Code: dict.Code, Name: "新名称", Status: commonv1.Status_STATUS_ENABLE})
	if err != nil {
		t.Fatalf("单连接事务内保存字典及翻译失败: %v", err)
	}
	var updated models.BaseDict
	if err = db.First(&updated, dict.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Name != "新名称" {
		t.Fatalf("字典名称未更新: %s", updated.Name)
	}
}

// TestCreateBaseConfigSavesI18n 验证新增配置可在单一事务内取得自增 ID 并保存对应译文。
func TestCreateBaseConfigSavesI18n(t *testing.T) {
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
	if err = db.AutoMigrate(&models.BaseLanguage{}, &models.BaseI18N{}, &models.BaseConfig{}); err != nil {
		t.Fatal(err)
	}
	if err = db.Callback().Create().Before("gorm:create").Register("test:assign-base-config-id", func(tx *gorm.DB) {
		config, ok := tx.Statement.Dest.(*models.BaseConfig)
		if ok && config.ID == 0 {
			config.ID = 100
		}
	}); err != nil {
		t.Fatal(err)
	}
	languages := []*models.BaseLanguage{
		{ID: 1, LanguageCode: "zh-CN", LanguageName: "中文", NativeName: "中文", Sort: 1, IsPrimary: true, Status: 1},
		{ID: 2, LanguageCode: "en-US", LanguageName: "英文", NativeName: "English", Sort: 2, Status: 1},
	}
	if err = db.Create(languages).Error; err != nil {
		t.Fatal(err)
	}
	cacheStore, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)

	var store *data.Data
	store, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	tx := data.NewTransaction(store)
	baseCase := &corebiz.BaseCase{Cache: cacheStore}
	languageCase := NewBaseLanguageCase(baseCase, tx, data.NewBaseLanguageRepository(store))
	i18nCase := NewBaseI18nCase(baseCase, data.NewBaseI18NRepository(store), languageCase)
	configCase := NewBaseConfigCase(baseCase, tx, data.NewBaseConfigRepository(store), i18nCase)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = configCase.CreateBaseConfig(ctx, &adminv1.BaseConfigForm{
		Site:   2,
		Name:   "站点名称",
		Type:   adminv1.BaseConfigType_BASE_CONFIG_TYPE_TEXT,
		Key:    "siteNameTest",
		Value:  "测试站点",
		Status: commonv1.Status_STATUS_ENABLE,
		NameI18ns: []*adminv1.BaseI18n{{
			TargetType: adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_CONFIG_NAME,
			Locale:     "en-US",
			Name:       "Site name",
		}},
		ValueI18ns: []*adminv1.BaseI18n{{
			TargetType: adminv1.I18nTargetType_I18N_TARGET_TYPE_BASE_CONFIG_VALUE,
			Locale:     "en-US",
			Name:       "Test site",
		}},
	})
	if err != nil {
		t.Fatalf("新增系统配置失败: %v", err)
	}
	var config models.BaseConfig
	if err = db.Where("key = ?", "siteNameTest").First(&config).Error; err != nil {
		t.Fatal(err)
	}
	if config.ID != 100 {
		t.Fatalf("新增系统配置 ID 错误: %d", config.ID)
	}
	var count int64
	if err = db.Model(&models.BaseI18N{}).Where("target_id = ?", config.ID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("新增系统配置译文数量错误: %d", count)
	}
}
