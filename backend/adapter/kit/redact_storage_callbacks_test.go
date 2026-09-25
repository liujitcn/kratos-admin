package kit

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-kit/redact"
	mysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// TestDirectEncryptionPolicyStoresPayloadAndRestoresPlaintext 验证AES和SM4直接加密不写旁表且能够恢复原文。
func TestDirectEncryptionPolicyStoresPayloadAndRestoresPlaintext(t *testing.T) {
	cipher, err := newFieldCipher([]byte("12345678901234567890123456789012"))
	if err != nil {
		t.Fatal(err)
	}
	for _, algorithm := range []string{"AES_GCM", "SM4_GCM"} {
		t.Run(algorithm, func(t *testing.T) {
			runtime := &storageRuntime{fieldCipher: cipher}
			entity := &storageCallbackTestEntity{ID: 42, Phone: "client-secret"}
			policy := redact.StorageFieldPolicy{ID: 11, TenantID: 1, TableName: "storage_callback_test", ColumnName: "phone", Rule: redact.FieldPolicy{Mode: redact.PolicyModeApplyRule, RuleType: directEncryptionRuleType, EncryptAlgorithm: algorithm, Transform: func(value any) any { return value }}}
			prepared, testErr := runtime.prepareStorageEntity(context.Background(), []redact.StorageFieldPolicy{policy}, entity, &gorm.DB{Statement: &gorm.Statement{}}, true)
			if testErr != nil {
				t.Fatal(testErr)
			}
			if entity.Phone == "client-secret" || strings.HasPrefix(entity.Phone, "ENC[") || len(prepared.values) != 0 {
				t.Fatalf("直接加密结果无效: value=%q prepared=%d", entity.Phone, len(prepared.values))
			}
			if testErr = runtime.decryptDirectEntity(context.Background(), entity, policy); testErr != nil {
				t.Fatal(testErr)
			}
			if entity.Phone != "client-secret" {
				t.Fatalf("直接加密未恢复原文: %q", entity.Phone)
			}
		})
	}
}

type storageQueryTestResolver struct {
	recordIDs map[string][]int64
}

// FindRecordIDsByDigest 返回测试明文对应的主键集合。
func (r storageQueryTestResolver) FindRecordIDsByDigest(_ context.Context, _ redact.StorageFieldPolicy, plainValue string) ([]int64, error) {
	return append([]int64(nil), r.recordIDs[plainValue]...), nil
}

// TestStorageFieldSelected 验证更新语句只处理实际写入的敏感字段。
func TestStorageFieldSelected(t *testing.T) {
	entity := &storageCallbackTestEntity{ID: 42, Phone: "13812345678"}
	policy := redact.StorageFieldPolicy{ColumnName: "phone"}

	selected, err := storageFieldSelected(context.Background(), &gorm.DB{Statement: &gorm.Statement{}}, entity, policy)
	if err != nil {
		t.Fatal(err)
	}
	if !selected {
		t.Fatal("非零敏感字段应被识别为待更新字段")
	}

	db := &gorm.DB{Statement: &gorm.Statement{Selects: []string{"status"}}}
	selected, err = storageFieldSelected(context.Background(), db, entity, policy)
	if err != nil {
		t.Fatal(err)
	}
	if selected {
		t.Fatal("未选中的敏感字段不应被处理")
	}

	db.Statement.Selects = []string{"Phone"}
	selected, err = storageFieldSelected(context.Background(), db, &storageCallbackTestEntity{ID: 42}, policy)
	if err != nil {
		t.Fatal(err)
	}
	if !selected {
		t.Fatal("显式选中的敏感字段应被处理")
	}

	db.Statement.Selects = nil
	db.Statement.Omits = []string{"phone"}
	selected, err = storageFieldSelected(context.Background(), db, entity, policy)
	if err != nil {
		t.Fatal(err)
	}
	if selected {
		t.Fatal("被忽略的敏感字段不应被处理")
	}
}

// TestMaterializeStorageResponseAllowsSelectWithoutProtectedFields 验证未查询受保护字段时不要求结果携带租户ID。
func TestMaterializeStorageResponseAllowsSelectWithoutProtectedFields(t *testing.T) {
	entitySchema, err := schema.Parse(&storageCallbackTestEntity{}, &sync.Map{}, schema.NamingStrategy{SingularTable: true})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &storageRuntime{resolver: &RedactPolicyResolver{
		storagePolicies: map[string][]redact.StorageFieldPolicy{
			storagePolicyKey(1, "default", "storage_callback_test"): {{ID: 1, TenantID: 1, TableName: "storage_callback_test", ColumnName: "phone"}},
		},
	}}
	db := &gorm.DB{Config: &gorm.Config{}, Statement: &gorm.Statement{
		Context: context.Background(),
		Dest:    &storageCallbackTestEntity{ID: 42, Status: 1},
		Schema:  entitySchema,
		Table:   "storage_callback_test",
		Selects: []string{"status"},
	}}

	runtime.materializeStorageResponse(db)

	if db.Error != nil {
		t.Fatalf("未查询受保护字段不应触发脱敏恢复: %v", db.Error)
	}
}

// TestMaterializeStorageResponseAllowsGlobalRecord 验证零租户全局记录不会误触发租户入库策略。
func TestMaterializeStorageResponseAllowsGlobalRecord(t *testing.T) {
	entitySchema, err := schema.Parse(&storageCallbackTestEntity{}, &sync.Map{}, schema.NamingStrategy{SingularTable: true})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &storageRuntime{resolver: &RedactPolicyResolver{
		storagePolicies: map[string][]redact.StorageFieldPolicy{
			storagePolicyKey(1, "default", "storage_callback_test"): {{ID: 1, TenantID: 1, TableName: "storage_callback_test", ColumnName: "phone"}},
		},
	}}
	db := &gorm.DB{Config: &gorm.Config{}, Statement: &gorm.Statement{
		Context: context.Background(),
		Dest:    &storageCallbackTestEntity{ID: 42, TenantID: 0, Phone: "127.0.0.1"},
		Schema:  entitySchema,
		Table:   "storage_callback_test",
	}}

	runtime.materializeStorageResponse(db)

	if db.Error != nil {
		t.Fatalf("零租户全局记录不应触发租户入库策略: %v", db.Error)
	}
}

// TestSelectedStoragePolicies 验证查询回调只恢复实际选中的敏感字段。
func TestSelectedStoragePolicies(t *testing.T) {
	policies := []redact.StorageFieldPolicy{
		{ColumnName: "phone"},
		{ColumnName: "email"},
		{ColumnName: "id_code"},
	}
	db := &gorm.DB{Statement: &gorm.Statement{Selects: []string{"id", "phone"}}}
	selected := selectedStoragePolicies(db, policies)
	if len(selected) != 1 || selected[0].ColumnName != "phone" {
		t.Fatalf("敏感字段选择结果错误: %#v", selected)
	}

	db.Statement.Selects = []string{"base_user.*"}
	selected = selectedStoragePolicies(db, policies)
	if len(selected) != len(policies) {
		t.Fatalf("整表查询应选择全部敏感字段: %#v", selected)
	}

	db.Statement.Selects = nil
	selected = selectedStoragePolicies(db, policies)
	if len(selected) != len(policies) {
		t.Fatalf("未指定查询列时应保持原有恢复行为: %#v", selected)
	}
}

// TestRewriteStorageExpression 验证敏感字段等值和集合查询会改写为主键条件。
func TestRewriteStorageExpression(t *testing.T) {
	policy := redact.StorageFieldPolicy{ID: 2001, TableName: "base_user", ColumnName: "phone"}
	resolver := storageQueryTestResolver{recordIDs: map[string][]int64{
		"13800138000": {42},
		"13900139000": {43, 44},
	}}
	expression := clause.Where{Exprs: []clause.Expression{
		clause.Eq{Column: clause.Column{Name: "phone"}, Value: "13800138000"},
		clause.IN{Column: clause.Column{Name: "phone"}, Values: []any{"13900139000"}},
	}}
	rewritten, changed, err := rewriteStorageExpression(
		context.Background(),
		resolver,
		expression,
		map[string]redact.StorageFieldPolicy{"phone": policy},
		clause.Column{Name: "id"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("敏感字段查询条件应被改写")
	}
	where, ok := rewritten.(clause.Where)
	if !ok || len(where.Exprs) != 2 {
		t.Fatalf("改写结果类型错误: %#v", rewritten)
	}
	var first clause.IN
	first, ok = where.Exprs[0].(clause.IN)
	if !ok || !reflect.DeepEqual(first.Values, []any{int64(42)}) {
		t.Fatalf("等值条件改写错误: %#v", where.Exprs[0])
	}
	var second clause.IN
	second, ok = where.Exprs[1].(clause.IN)
	if !ok || !reflect.DeepEqual(second.Values, []any{int64(43), int64(44)}) {
		t.Fatalf("集合条件改写错误: %#v", where.Exprs[1])
	}
}

// TestQueryForDBResetsMainStatement 验证旁表查询不会复用主表创建语句。
func TestQueryForDBResetsMainStatement(t *testing.T) {
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: &sql.DB{}, SkipInitializeWithVersion: true}), &gorm.Config{DisableAutomaticPing: true, NamingStrategy: schema.NamingStrategy{SingularTable: true}})
	if err != nil {
		t.Fatal(err)
	}
	db.Statement.Table = "base_user"
	storageDB := queryForDB(db).BaseRedactStorageValue.WithContext(context.Background()).UnderlyingDB() //nolint:forbidigo // 测试需要检查 gorm/gen 绑定的旁表模型。
	tableName := ""
	if storageDB.Statement != nil && storageDB.Statement.Schema != nil {
		tableName = storageDB.Statement.Schema.Table
	}
	if tableName != models.TableNameBaseRedactStorageValue {
		t.Fatalf("旁表 Statement 未重置: table=%q", tableName)
	}
}

// TestIsRedactMetadataTable 验证脱敏元数据表不会触发响应物化。
func TestIsRedactMetadataTable(t *testing.T) {
	for _, tableName := range []string{"base_redact_rule", "base_redact_storage_policy", "base_redact_output_policy", "base_redact_storage_value"} {
		if !isRedactMetadataTable(tableName) {
			t.Fatalf("表 %s 应被识别为脱敏元数据表", tableName)
		}
	}
	if isRedactMetadataTable("base_user") {
		t.Fatal("业务表不应被识别为脱敏元数据表")
	}
}

// TestStorageFieldSupportsAtRestProtection 验证唯一索引字段不会被改写为碰撞值。
func TestStorageFieldSupportsAtRestProtection(t *testing.T) {
	entitySchema, err := schema.Parse(&storageCallbackTestEntity{}, &sync.Map{}, schema.NamingStrategy{SingularTable: true})
	if err != nil {
		t.Fatal(err)
	}
	db := &gorm.DB{Statement: &gorm.Statement{Schema: entitySchema}}
	if storageFieldSupportsAtRestProtection(db, redact.StorageFieldPolicy{ColumnName: "user_code"}) {
		t.Fatal("唯一索引字段不应启用入库掩码")
	}
	if !storageFieldSupportsAtRestProtection(db, redact.StorageFieldPolicy{ColumnName: "remark"}) {
		t.Fatal("普通字段应允许启用入库掩码")
	}
}

// TestEntityPrimaryID 验证回调会从 GORM 模型元数据读取 int64 主键。
func TestEntityPrimaryID(t *testing.T) {
	entitySchema, err := schema.Parse(&storageCallbackTestEntity{}, &sync.Map{}, schema.NamingStrategy{SingularTable: true})
	if err != nil {
		t.Fatal(err)
	}
	db := &gorm.DB{Statement: &gorm.Statement{Schema: entitySchema, Table: "storage_callback_test"}}
	var recordID int64
	recordID, err = entityPrimaryID(context.Background(), db, &storageCallbackTestEntity{ID: 42})
	if err != nil {
		t.Fatal(err)
	}
	if recordID != 42 {
		t.Fatalf("主键读取错误: %d", recordID)
	}
}

// TestRecordIDsFromWhere 验证删除回调可以从主键等值和集合条件提取记录ID。
func TestRecordIDsFromWhere(t *testing.T) {
	expression := clause.Where{Exprs: []clause.Expression{
		clause.Eq{Column: clause.Column{Name: "id"}, Value: int64(42)},
		clause.IN{Column: clause.Column{Name: "id"}, Values: []any{int64(43), int64(44)}},
	}}
	recordIDs := recordIDsFromWhere(expression, "id")
	if len(recordIDs) != 3 || recordIDs[0] != 42 || recordIDs[1] != 43 || recordIDs[2] != 44 {
		t.Fatalf("主键条件提取错误: %v", recordIDs)
	}
}

// storageCallbackTestEntity 提供更新字段选择测试实体。
type storageCallbackTestEntity struct {
	ID       int64
	TenantID int64  `gorm:"column:tenant_id"`
	UserCode string `gorm:"column:user_code;uniqueIndex:unique_storage_callback_user_code"`
	Phone    string
	Remark   string
	Status   int32
}

// TableName 返回更新字段选择测试实体的物理表名。
func (storageCallbackTestEntity) TableName() string {
	return "storage_callback_test"
}
