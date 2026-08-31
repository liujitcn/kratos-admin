package data

import (
	"reflect"
	"testing"

	generateddata "github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"gorm.io/gorm/schema"
)

// TestModelsAvoidDatabaseDefaults 验证自动迁移模型的默认值全部由程序显式填写。
func TestModelsAvoidDatabaseDefaults(t *testing.T) {
	for _, model := range generateddata.Models() {
		modelType := reflect.TypeOf(model)
		if modelType.Kind() == reflect.Ptr {
			modelType = modelType.Elem()
		}
		for i := 0; i < modelType.NumField(); i++ {
			field := modelType.Field(i)
			settings := schema.ParseTagSetting(field.Tag.Get("gorm"), ";")
			if defaultValue, ok := settings["DEFAULT"]; ok {
				t.Errorf("%s.%s 不应使用数据库默认值 %q", modelType.Name(), field.Name, defaultValue)
			}
		}
	}
}
