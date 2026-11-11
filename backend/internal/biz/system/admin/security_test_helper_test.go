package biz

import (
	"path/filepath"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// newSecurityData 为安全业务回归建立独立 SQLite 数据库，避免连接真实业务环境。
func newSecurityData(t *testing.T, registered ...any) (*kitgorm.Client, *data.Data) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "security.db")), &gorm.Config{NamingStrategy: schema.NamingStrategy{SingularTable: true}})
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range registered {
		statement := &gorm.Statement{DB: db}
		err = statement.Parse(model)
		if err != nil {
			t.Fatal(err)
		}
		if field := statement.Schema.FieldsByName["ID"]; field != nil {
			field.DataType = schema.Int
		}
	}
	if err = db.AutoMigrate(registered...); err != nil {
		t.Fatal(err)
	}
	client := &kitgorm.Client{DB: db}
	database, err := data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: client})
	if err != nil {
		t.Fatal(err)
	}
	return client, database
}
