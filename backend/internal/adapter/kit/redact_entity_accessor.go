package kit

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/liujitcn/kratos-kit/redact"
	"gorm.io/gorm/schema"
)

var _ redact.EntityFieldAccessor = gormEntityFieldAccessor{}

// gormEntityFieldAccessor 使用 GORM 模型元数据访问业务实体字段。
type gormEntityFieldAccessor struct{}

// ValueOf 读取业务实体字段。
func (gormEntityFieldAccessor) ValueOf(ctx context.Context, entity any, fieldName string) (any, bool, error) {
	entityValue, entitySchema, err := parseEntity(ctx, entity)
	if err != nil {
		return nil, false, err
	}
	modelField := entitySchema.LookUpField(fieldName)
	if modelField == nil {
		return nil, false, fmt.Errorf("实体缺少字段 %s", fieldName)
	}
	value, zero := modelField.ValueOf(ctx, entityValue)
	return value, zero, nil
}

// Set 写入业务实体字段。
func (gormEntityFieldAccessor) Set(ctx context.Context, entity any, fieldName string, value any) error {
	entityValue, entitySchema, err := parseEntity(ctx, entity)
	if err != nil {
		return err
	}
	modelField := entitySchema.LookUpField(fieldName)
	if modelField == nil {
		return fmt.Errorf("实体缺少字段 %s", fieldName)
	}
	return modelField.Set(ctx, entityValue, value)
}

// parseEntity 解析可读写的 GORM 实体指针。
func parseEntity(ctx context.Context, entity any) (reflect.Value, *schema.Schema, error) {
	entityValue := reflect.ValueOf(entity)
	if entityValue.Kind() != reflect.Pointer || entityValue.IsNil() || entityValue.Elem().Kind() != reflect.Struct {
		return reflect.Value{}, nil, errors.New("实体必须是非空结构体指针")
	}
	entitySchema, err := schema.Parse(entity, &sync.Map{}, schema.NamingStrategy{SingularTable: true})
	if err != nil {
		return reflect.Value{}, nil, fmt.Errorf("解析实体失败: %w", err)
	}
	return entityValue, entitySchema, nil
}
