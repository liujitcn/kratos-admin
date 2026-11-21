package data

import (
	"path/filepath"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-kit/cache"
	"github.com/liujitcn/kratos-kit/cache/memory"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// TestLoadCachesAPIRecordsAndInvalidates 验证重复API保留、暖缓存不查库、版本切换重新读取。
func TestLoadCachesAPIRecordsAndInvalidates(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "api.db")), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}})
	if err != nil {
		t.Fatal(err)
	}
	statement := &gorm.Statement{DB: db}
	if err = statement.Parse(new(models.BaseAPI)); err != nil {
		t.Fatal(err)
	}
	statement.Schema.FieldsByName["ID"].DataType = schema.Int
	if err = db.AutoMigrate(new(models.BaseAPI)); err != nil {
		t.Fatal(err)
	}
	rows := []*models.BaseAPI{{ID: 1, ToolName: "read", AgentStatus: 1, ToolPrompts: "[]"}, {ID: 2, ToolName: "read", AgentStatus: 1, ToolPrompts: "[]"}, {ID: 3, ToolName: "removed", DeletedAt: 1, ToolPrompts: "[]"}}
	if err = db.Create(rows).Error; err != nil {
		t.Fatal(err)
	}
	var provider *data.Data
	provider, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	repo := data.NewBaseAPIRepository(provider)
	var store cache.Cache
	var cleanup func()
	store, cleanup, err = memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	queries := 0
	if err = db.Callback().Query().Before("gorm:query").Register("count_api", func(tx *gorm.DB) {
		if tx.Statement.Table == "base_api" {
			queries++
		}
	}); err != nil {
		t.Fatal(err)
	}
	var first map[string][]*models.BaseAPI
	first, err = LoadBaseAPICache(t.Context(), repo, store)
	if err != nil {
		t.Fatal(err)
	}
	if len(first["read"]) != 2 || len(first["removed"]) != 0 {
		t.Fatalf("快照未保留重复记录或包含软删除: %+v", first)
	}
	first["read"][0].AgentStatus = 99
	var second map[string][]*models.BaseAPI
	second, err = LoadBaseAPICache(t.Context(), repo, store)
	if err != nil {
		t.Fatal(err)
	}
	if queries != 1 || second["read"][0].AgentStatus != 1 {
		t.Fatalf("暖缓存查库或共享可变指针: queries=%d rows=%+v", queries, second)
	}
	query := provider.Query(t.Context()).BaseAPI
	if _, err = query.WithContext(t.Context()).Where(query.ID.Eq(1)).Update(query.AgentStatus, 0); err != nil {
		t.Fatal(err)
	}
	if err = InvalidateBaseAPICache(store); err != nil {
		t.Fatal(err)
	}
	var third map[string][]*models.BaseAPI
	third, err = LoadBaseAPICache(t.Context(), repo, store)
	if err != nil {
		t.Fatal(err)
	}
	if queries != 2 || third["read"][0].AgentStatus != 0 {
		t.Fatalf("版本更新未重新读取: queries=%d rows=%+v", queries, third)
	}
	// 损坏或不可用的缓存必须回源，不能把缺失配置视为授权成功。
	if err = store.Set(baseAPICacheRevisionKey, "invalid", 0); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadBaseAPICache(t.Context(), repo, store); err != nil {
		t.Fatal(err)
	}
	if _, err = LoadBaseAPICache(t.Context(), repo, nil); err != nil {
		t.Fatal(err)
	}
	if queries != 4 {
		t.Fatalf("缓存不可用未回源: %d", queries)
	}
}
