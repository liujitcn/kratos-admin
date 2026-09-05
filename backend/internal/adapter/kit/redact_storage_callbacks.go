package kit

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/redact"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	storagePreparedStateKey = "kratos-admin:redact/storage-prepared"
	storageDeleteStateKey   = "kratos-admin:redact/storage-delete"
)

var storageRuntimeValue atomic.Pointer[storageRuntime]

type storageRuntime struct {
	storage  *redact.RedactStorage
	resolver *RedactPolicyResolver
	store    *storageValueStore
}

type storageDigestResolver interface {
	FindRecordIDsByDigest(context.Context, redact.StorageFieldPolicy, string) ([]int64, error)
}

type preparedEntity struct {
	entity          any
	deletePolicyIDs []int64
	values          map[int64]*redact.StorageValue
}

type preparedState struct {
	entities []preparedEntity
}

type deletedState struct {
	recordIDs []int64
	policies  []redact.StorageFieldPolicy
}

func init() {
	kitgorm.RegisterCallbackQuery(rewriteStorageQuery)
	kitgorm.RegisterCallbackQueryAfter(materializeStorageResponse)
	kitgorm.RegisterCallbackCreate(prepareStorageCreate)
	kitgorm.RegisterCallbackCreateAfter(saveStorageValues)
	kitgorm.RegisterCallbackUpdateBefore("gorm:update", prepareStorageUpdate)
	kitgorm.RegisterCallbackUpdateAfter(saveStorageValues)
	kitgorm.RegisterCallbackDelete(captureStorageDelete)
	kitgorm.RegisterCallbackDeleteAfter(deleteStorageValues)
}

// NewStorageRuntime 创建并注册当前 Admin 的敏感字段存储运行时。
func NewStorageRuntime(store *storageValueStore, resolver *RedactPolicyResolver, protector *redact.StorageProtector) *redact.RedactStorage {
	storage := redact.NewRedactStorage(store, resolver, protector, gormEntityFieldAccessor{})
	storageRuntimeValue.Store(&storageRuntime{storage: storage, resolver: resolver, store: store})
	return storage
}

// NewRedactRuntime 创建 Admin 脱敏策略运行时并注册存储回调。
func NewRedactRuntime(
	storagePolicyRepository *data.BaseRedactStoragePolicyRepository,
	outputPolicyRepository *data.BaseRedactOutputPolicyRepository,
	ruleRepository *data.BaseRedactRuleRepository,
	store *storageValueStore,
	protector *redact.StorageProtector,
) (*RedactPolicyResolver, error) {
	resolver := NewRedactPolicyResolver(storagePolicyRepository, outputPolicyRepository, ruleRepository)
	err := resolver.Refresh(context.Background())
	if err != nil {
		return nil, err
	}
	NewStorageRuntime(store, resolver, protector)
	redact.SetDefaultPolicyResolver(resolver)
	return resolver, nil
}

// rewriteStorageQuery 将敏感字段明文等值条件改写为旁表摘要对应的主键条件。
func rewriteStorageQuery(db *gorm.DB) {
	runtime := storageRuntimeValue.Load()
	if runtime == nil || runtime.storage == nil || runtime.resolver == nil || db == nil || db.Statement == nil || db.Error != nil {
		return
	}
	if isRedactMetadataTable(db.Statement.Table) || db.Statement.Schema == nil || db.Statement.Schema.PrioritizedPrimaryField == nil {
		return
	}
	where, ok := db.Statement.Clauses["WHERE"]
	if !ok || where.Expression == nil {
		return
	}
	policies := runtime.resolver.ListStoragePolicies(db.Statement.Context, db.Statement.Table)
	if len(policies) == 0 {
		return
	}
	policyByColumn := make(map[string]redact.StorageFieldPolicy, len(policies))
	for _, policy := range policies {
		policyByColumn[strings.ToLower(policy.ColumnName)] = policy
	}
	primaryColumn := clause.Column{Name: db.Statement.Schema.PrioritizedPrimaryField.DBName}
	expression, changed, err := rewriteStorageExpression(db.Statement.Context, runtime.storage, where.Expression, policyByColumn, primaryColumn)
	if err != nil {
		db.AddError(err)
		return
	}
	if changed {
		where.Expression = expression
		db.Statement.Clauses["WHERE"] = where
	}
}

// rewriteStorageExpression 递归改写 GORM 条件树中的敏感字段等值条件。
func rewriteStorageExpression(
	ctx context.Context,
	storage storageDigestResolver,
	expression clause.Expression,
	policyByColumn map[string]redact.StorageFieldPolicy,
	primaryColumn clause.Column,
) (clause.Expression, bool, error) {
	var err error
	switch value := expression.(type) {
	case clause.Where:
		var changed bool
		value.Exprs, changed, err = rewriteStorageExpressions(ctx, storage, value.Exprs, policyByColumn, primaryColumn)
		return value, changed, err
	case clause.AndConditions:
		var changed bool
		value.Exprs, changed, err = rewriteStorageExpressions(ctx, storage, value.Exprs, policyByColumn, primaryColumn)
		return value, changed, err
	case clause.OrConditions:
		var changed bool
		value.Exprs, changed, err = rewriteStorageExpressions(ctx, storage, value.Exprs, policyByColumn, primaryColumn)
		return value, changed, err
	case clause.NotConditions:
		var changed bool
		value.Exprs, changed, err = rewriteStorageExpressions(ctx, storage, value.Exprs, policyByColumn, primaryColumn)
		return value, changed, err
	case clause.Eq:
		policy, ok := storagePolicyForColumn(policyByColumn, value.Column)
		plainValue, valueOK := value.Value.(string)
		if !ok || !valueOK {
			return expression, false, nil
		}
		var recordIDs []int64
		recordIDs, err = storage.FindRecordIDsByDigest(ctx, policy, plainValue)
		if err != nil {
			return nil, false, err
		}
		return recordIDCondition(primaryColumn, recordIDs), true, nil
	case clause.IN:
		policy, ok := storagePolicyForColumn(policyByColumn, value.Column)
		if !ok {
			return expression, false, nil
		}
		plainValues := make([]string, 0, len(value.Values))
		for _, item := range value.Values {
			plainValue, valueOK := item.(string)
			if !valueOK {
				return expression, false, nil
			}
			plainValues = append(plainValues, plainValue)
		}
		var recordIDs []int64
		recordIDs, err = findRecordIDsByPlainValues(ctx, storage, policy, plainValues)
		if err != nil {
			return nil, false, err
		}
		return recordIDCondition(primaryColumn, recordIDs), true, nil
	default:
		return expression, false, nil
	}
}

// rewriteStorageExpressions 批量递归改写 GORM 条件列表。
func rewriteStorageExpressions(
	ctx context.Context,
	storage storageDigestResolver,
	expressions []clause.Expression,
	policyByColumn map[string]redact.StorageFieldPolicy,
	primaryColumn clause.Column,
) ([]clause.Expression, bool, error) {
	result := make([]clause.Expression, 0, len(expressions))
	changed := false
	var err error
	for _, expression := range expressions {
		var rewritten clause.Expression
		var itemChanged bool
		rewritten, itemChanged, err = rewriteStorageExpression(ctx, storage, expression, policyByColumn, primaryColumn)
		if err != nil {
			return nil, false, err
		}
		result = append(result, rewritten)
		changed = changed || itemChanged
	}
	return result, changed, nil
}

// findRecordIDsByPlainValues 批量查找多个敏感字段明文对应的主键。
func findRecordIDsByPlainValues(ctx context.Context, storage storageDigestResolver, policy redact.StorageFieldPolicy, plainValues []string) ([]int64, error) {
	seen := make(map[int64]struct{})
	result := make([]int64, 0)
	var err error
	for _, plainValue := range plainValues {
		var recordIDs []int64
		recordIDs, err = storage.FindRecordIDsByDigest(ctx, policy, plainValue)
		if err != nil {
			return nil, err
		}
		for _, recordID := range recordIDs {
			if _, ok := seen[recordID]; ok {
				continue
			}
			seen[recordID] = struct{}{}
			result = append(result, recordID)
		}
	}
	return result, nil
}

// storagePolicyForColumn 按 GORM 条件列名匹配入库策略。
func storagePolicyForColumn(policyByColumn map[string]redact.StorageFieldPolicy, column any) (redact.StorageFieldPolicy, bool) {
	name := clauseColumnName(column)
	policy, ok := policyByColumn[strings.ToLower(name)]
	return policy, ok
}

// recordIDCondition 构造摘要查询结果对应的主键条件。
func recordIDCondition(primaryColumn clause.Column, recordIDs []int64) clause.Expression {
	values := make([]any, 0, len(recordIDs))
	for _, recordID := range recordIDs {
		values = append(values, recordID)
	}
	return clause.IN{Column: primaryColumn, Values: values}
}

// prepareStorageCreate 在 GORM 创建前将敏感字段转换为存储值。
func prepareStorageCreate(db *gorm.DB) {
	prepareStorageEntities(db, true)
}

// prepareStorageUpdate 在 GORM 更新前将敏感字段转换为存储值。
func prepareStorageUpdate(db *gorm.DB) {
	prepareStorageEntities(db, false)
}

// prepareStorageEntities 按当前物理表处理待持久化实体。
func prepareStorageEntities(db *gorm.DB, creating bool) {
	runtime := storageRuntimeValue.Load()
	if runtime == nil || runtime.storage == nil || runtime.resolver == nil || db == nil || db.Statement == nil || db.Error != nil {
		return
	}
	policies := runtime.resolver.ListStoragePolicies(db.Statement.Context, db.Statement.Table)
	if len(policies) == 0 {
		return
	}
	entities := destinationEntities(db.Statement.Dest)
	if len(entities) == 0 {
		if destinationContainsProtectedColumn(db.Statement.Dest, policies) {
			db.AddError(fmt.Errorf("受保护表 %s 的敏感字段更新必须使用实体模型", db.Statement.Table))
		}
		return
	}
	state := &preparedState{entities: make([]preparedEntity, 0, len(entities))}
	var err error
	for _, entity := range entities {
		var prepared preparedEntity
		prepared, err = prepareStorageEntity(db.Statement.Context, runtime, policies, entity, db, creating)
		if err != nil {
			db.AddError(err)
			return
		}
		if len(prepared.values) == 0 && len(prepared.deletePolicyIDs) == 0 {
			continue
		}
		state.entities = append(state.entities, prepared)
	}
	if len(state.entities) > 0 {
		db.InstanceSet(storagePreparedStateKey, state)
	}
}

// prepareStorageEntity 处理单个实体并记录待保存或删除的旁表值。
func prepareStorageEntity(ctx context.Context, runtime *storageRuntime, policies []redact.StorageFieldPolicy, entity any, db *gorm.DB, creating bool) (preparedEntity, error) {
	selectedPolicies := make([]redact.StorageFieldPolicy, 0, len(policies))
	deletePolicyIDs := make([]int64, 0)
	accessor := gormEntityFieldAccessor{}
	var err error
	for _, policy := range policies {
		var selected bool
		selected, err = storageFieldSelected(ctx, db, entity, policy)
		if err != nil {
			return preparedEntity{}, err
		}
		if !selected || !storageFieldSupportsAtRestProtection(db, policy) {
			continue
		}
		var value any
		var zero bool
		value, zero, err = accessor.ValueOf(ctx, entity, policy.ColumnName)
		if err != nil {
			return preparedEntity{}, err
		}
		if zero || value == nil || value == "" {
			if !creating {
				deletePolicyIDs = append(deletePolicyIDs, policy.ID)
			}
			continue
		}
		if _, ok := value.(string); !ok {
			return preparedEntity{}, fmt.Errorf("实体 %s 字段 %s 不是字符串", policy.TableName, policy.ColumnName)
		}
		selectedPolicies = append(selectedPolicies, policy)
	}
	var values map[int64]*redact.StorageValue
	values, err = runtime.storage.PrepareEntityWithPolicies(ctx, entity, selectedPolicies)
	if err != nil {
		return preparedEntity{}, err
	}
	return preparedEntity{entity: entity, deletePolicyIDs: deletePolicyIDs, values: values}, nil
}

// storageFieldSupportsAtRestProtection 判断字段是否适合保存脱敏值。
func storageFieldSupportsAtRestProtection(db *gorm.DB, policy redact.StorageFieldPolicy) bool {
	if db == nil || db.Statement == nil || db.Statement.Schema == nil {
		return true
	}
	field := db.Statement.Schema.LookUpField(policy.ColumnName)
	if field == nil {
		return true
	}
	// 唯一索引字段改写后可能发生碰撞，必须保留原值。
	if field.Unique || field.UniqueIndex != "" {
		return false
	}
	for _, index := range db.Statement.Schema.ParseIndexes() {
		if index.Class != "UNIQUE" {
			continue
		}
		for _, indexField := range index.Fields {
			if indexField.DBName == field.DBName {
				return false
			}
		}
	}
	return true
}

// storageFieldSelected 判断更新语句是否会写入指定敏感字段。
func storageFieldSelected(ctx context.Context, db *gorm.DB, entity any, policy redact.StorageFieldPolicy) (bool, error) {
	if db == nil || db.Statement == nil {
		return false, errors.New("当前 GORM 语句不能为空")
	}
	for _, name := range db.Statement.Omits {
		if sameStorageFieldName(name, policy.ColumnName) {
			return false, nil
		}
	}
	if len(db.Statement.Selects) > 0 {
		for _, name := range db.Statement.Selects {
			if name == "*" || sameStorageFieldName(name, policy.ColumnName) {
				return true, nil
			}
		}
		return false, nil
	}
	value, zero, err := (gormEntityFieldAccessor{}).ValueOf(ctx, entity, policy.ColumnName)
	if err != nil {
		return false, err
	}
	return !zero && value != nil, nil
}

// sameStorageFieldName 比较 GORM 字段名和数据库字段名。
func sameStorageFieldName(left, right string) bool {
	left = strings.Trim(left, "`")
	if index := strings.LastIndex(left, "."); index >= 0 {
		left = left[index+1:]
	}
	return strings.EqualFold(left, right)
}

// saveStorageValues 在主表写入完成后保存旁表敏感值。
func saveStorageValues(db *gorm.DB) {
	runtime := storageRuntimeValue.Load()
	if runtime == nil || runtime.store == nil || db == nil || db.Statement == nil || db.Error != nil {
		return
	}
	value, ok := db.InstanceGet(storagePreparedStateKey)
	if !ok {
		return
	}
	state, ok := value.(*preparedState)
	if !ok {
		db.AddError(errors.New("敏感字段存储状态类型无效"))
		return
	}
	var err error
	for _, prepared := range state.entities {
		var recordID int64
		recordID, err = entityPrimaryID(db.Statement.Context, db, prepared.entity)
		if err != nil {
			db.AddError(err)
			return
		}
		if recordID <= 0 {
			db.AddError(fmt.Errorf("表 %s 的业务主键不能为空", db.Statement.Table))
			return
		}
		for _, item := range prepared.values {
			item.RecordID = recordID
			err = runtime.store.SaveWithDB(db.Statement.Context, db, item)
			if err != nil {
				db.AddError(err)
				return
			}
		}
		for _, storagePolicyID := range prepared.deletePolicyIDs {
			err = runtime.store.DeleteWithDB(db.Statement.Context, db, storagePolicyID, recordID)
			if err != nil {
				db.AddError(err)
				return
			}
		}
	}
}

// captureStorageDelete 在删除前捕获受保护实体的业务主键。
func captureStorageDelete(db *gorm.DB) {
	runtime := storageRuntimeValue.Load()
	if runtime == nil || runtime.resolver == nil || db == nil || db.Statement == nil || db.Error != nil {
		return
	}
	policies := runtime.resolver.ListStoragePolicies(db.Statement.Context, db.Statement.Table)
	if len(policies) == 0 {
		return
	}
	recordIDs := make([]int64, 0)
	entities := destinationEntities(db.Statement.Dest)
	var err error
	for _, entity := range entities {
		var recordID int64
		recordID, err = entityPrimaryID(db.Statement.Context, db, entity)
		if err != nil {
			db.AddError(err)
			return
		}
		if recordID > 0 {
			recordIDs = append(recordIDs, recordID)
		}
	}
	if len(recordIDs) == 0 && db.Statement.Schema != nil && db.Statement.Schema.PrioritizedPrimaryField != nil {
		recordIDs = recordIDsFromWhere(db.Statement.Clauses["WHERE"].Expression, db.Statement.Schema.PrioritizedPrimaryField.DBName)
	}
	if len(recordIDs) == 0 {
		db.AddError(fmt.Errorf("受保护表 %s 的删除必须包含主键条件", db.Statement.Table))
		return
	}
	db.InstanceSet(storageDeleteStateKey, &deletedState{recordIDs: recordIDs, policies: policies})
}

// deleteStorageValues 在主表删除完成后物理删除旁表敏感值。
func deleteStorageValues(db *gorm.DB) {
	runtime := storageRuntimeValue.Load()
	if runtime == nil || runtime.store == nil || db == nil || db.Statement == nil || db.Error != nil {
		return
	}
	value, ok := db.InstanceGet(storageDeleteStateKey)
	if !ok {
		return
	}
	state, ok := value.(*deletedState)
	if !ok {
		db.AddError(errors.New("敏感字段删除状态类型无效"))
		return
	}
	var err error
	for _, recordID := range state.recordIDs {
		for _, policy := range state.policies {
			err = runtime.store.DeleteWithDB(db.Statement.Context, db, policy.ID, recordID)
			if err != nil {
				db.AddError(err)
				return
			}
		}
	}
}

// materializeStorageResponse 在业务查询完成后恢复主表敏感字段原文。
func materializeStorageResponse(db *gorm.DB) {
	runtime := storageRuntimeValue.Load()
	if runtime == nil || runtime.storage == nil || runtime.resolver == nil || db == nil || db.Statement == nil || db.Error != nil {
		return
	}
	if isRedactMetadataTable(db.Statement.Table) {
		return
	}
	policies := runtime.resolver.ListStoragePolicies(db.Statement.Context, db.Statement.Table)
	if len(policies) == 0 {
		return
	}
	entities := destinationEntities(db.Statement.Dest)
	responseEntities := make([]redact.ResponseEntity, 0, len(entities))
	var err error
	for _, entity := range entities {
		var recordID int64
		recordID, err = entityPrimaryID(db.Statement.Context, db, entity)
		if err != nil {
			db.AddError(err)
			return
		}
		if recordID > 0 {
			responseEntities = append(responseEntities, redact.ResponseEntity{RecordID: recordID, Entity: entity})
		}
	}
	err = runtime.storage.RestoreEntities(db.Statement.Context, policies, responseEntities)
	if err != nil {
		db.AddError(err)
	}
}

// isRedactMetadataTable 判断查询是否来自脱敏配置和旁表元数据。
func isRedactMetadataTable(tableName string) bool {
	switch tableName {
	case "base_redact_rule", "base_redact_storage_policy", "base_redact_output_policy", "base_redact_storage_value":
		return true
	default:
		return false
	}
}

// destinationEntities 提取 GORM 当前语句中的可写实体指针。
func destinationEntities(destination any) []any {
	if destination == nil {
		return nil
	}
	value := reflect.ValueOf(destination)
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		if value.IsNil() {
			return nil
		}
		if value.Kind() == reflect.Pointer && value.Elem().Kind() == reflect.Struct {
			return []any{destination}
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return nil
	}
	entities := make([]any, 0, value.Len())
	for index := 0; index < value.Len(); index++ {
		item := value.Index(index)
		if item.Kind() == reflect.Pointer {
			if !item.IsNil() {
				entities = append(entities, item.Interface())
			}
			continue
		}
		if item.CanAddr() {
			entities = append(entities, item.Addr().Interface())
		}
	}
	return entities
}

// destinationContainsProtectedColumn 判断 map 更新是否实际修改了受保护列。
func destinationContainsProtectedColumn(destination any, policies []redact.StorageFieldPolicy) bool {
	value := reflect.ValueOf(destination)
	for value.IsValid() && (value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) {
		if value.IsNil() {
			return false
		}
		value = value.Elem()
	}
	if !value.IsValid() || value.Kind() != reflect.Map || value.Type().Key().Kind() != reflect.String {
		return false
	}
	for _, policy := range policies {
		for _, name := range []string{policy.ColumnName, strings.ToLower(policy.ColumnName)} {
			key := reflect.ValueOf(name).Convert(value.Type().Key())
			if value.MapIndex(key).IsValid() {
				return true
			}
		}
	}
	return false
}

// entityPrimaryID 从 GORM 模型元数据读取 int64 业务主键。
func entityPrimaryID(ctx context.Context, db *gorm.DB, entity any) (int64, error) {
	if entity == nil {
		return 0, nil
	}
	entityValue := reflect.ValueOf(entity)
	entitySchema := db.Statement.Schema
	var err error
	if entitySchema == nil {
		_, entitySchema, err = parseEntity(ctx, entity)
		if err != nil {
			return 0, err
		}
	}
	if entitySchema.PrioritizedPrimaryField == nil {
		return 0, fmt.Errorf("表 %s 未定义单一主键", db.Statement.Table)
	}
	value, zero := entitySchema.PrioritizedPrimaryField.ValueOf(ctx, entityValue)
	if zero || value == nil {
		return 0, nil
	}
	return integerValue(value, entitySchema.PrioritizedPrimaryField.DBName)
}

// integerValue 将整数类型字段转换为 int64。
func integerValue(value any, fieldName string) (int64, error) {
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return reflected.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(reflected.Uint()), nil
	default:
		return 0, fmt.Errorf("实体字段 %s 不是整数类型", fieldName)
	}
}

// recordIDsFromWhere 从 GORM 主键条件中提取待删除记录编号。
func recordIDsFromWhere(expression clause.Expression, fieldName string) []int64 {
	result := make([]int64, 0)
	var collect func(clause.Expression)
	var recordID int64
	var err error
	collect = func(item clause.Expression) {
		switch value := item.(type) {
		case clause.Where:
			for _, child := range value.Exprs {
				collect(child)
			}
		case clause.AndConditions:
			for _, child := range value.Exprs {
				collect(child)
			}
		case clause.OrConditions:
			for _, child := range value.Exprs {
				collect(child)
			}
		case clause.Eq:
			if clauseColumnName(value.Column) == fieldName {
				recordID, err = integerValue(value.Value, fieldName)
				if err == nil && recordID > 0 {
					result = append(result, recordID)
				}
			}
		case clause.IN:
			if clauseColumnName(value.Column) != fieldName {
				return
			}
			for _, child := range value.Values {
				recordID, err = integerValue(child, fieldName)
				if err == nil && recordID > 0 {
					result = append(result, recordID)
				}
			}
		}
	}
	if expression != nil {
		collect(expression)
	}
	return result
}

// clauseColumnName 返回 GORM 条件的数据库列名称。
func clauseColumnName(column any) string {
	switch value := column.(type) {
	case clause.Column:
		return value.Name
	case string:
		return value
	default:
		return ""
	}
}
