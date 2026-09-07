package kit

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/redact"
	"github.com/liujitcn/kratos-kit/sdk"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

type redactTestKey struct {
	secret string
	err    error
	calls  int
}

// Derive 返回测试密钥或注入的错误，不访问外部密钥服务。
func (k *redactTestKey) Derive(context.Context, string) ([]byte, error) {
	k.calls++
	return []byte(k.secret), k.err
}

// TestRedactConstructorsRejectInvalidDatabases 验证公开构造函数校验默认数据库。
func TestRedactConstructorsRejectInvalidDatabases(t *testing.T) {
	for _, test := range []struct {
		name      string
		databases map[string]*kitgorm.Client
	}{
		{"nil", nil},
		{"empty", map[string]*kitgorm.Client{}},
		{"missing-default", map[string]*kitgorm.Client{"other": {DB: newRedactTestDB(t)}}},
		{"nil-client", map[string]*kitgorm.Client{kitgorm.DefaultClientName: nil}},
		{"nil-db", map[string]*kitgorm.Client{kitgorm.DefaultClientName: {}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolver, err := NewRedactPolicyResolver(test.databases)
			if err == nil || resolver != nil {
				t.Fatalf("无效数据库不应构造解析器: resolver=%v err=%v", resolver, err)
			}
			var store *StorageValueStore
			store, err = NewStorageValueStore(test.databases)
			if err == nil || store != nil {
				t.Fatalf("无效数据库不应构造旁表仓储: store=%v err=%v", store, err)
			}
		})
	}
}

// TestRedactConstructorHasNoDatabaseSideEffects 验证构造及初始化前解析不查表、不注册回调。
func TestRedactConstructorHasNoDatabaseSideEffects(t *testing.T) {
	db := newRedactTestDB(t)
	queries := 0
	err := db.Callback().Query().Before("gorm:query").Register("test:count-query", func(*gorm.DB) { queries++ })
	if err != nil {
		t.Fatal(err)
	}
	databases := map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}}
	var resolver *RedactPolicyResolver
	resolver, err = NewRedactPolicyResolver(databases)
	if err != nil {
		t.Fatal(err)
	}
	var store *StorageValueStore
	store, err = NewStorageValueStore(databases)
	if err != nil {
		t.Fatal(err)
	}
	if resolver.defaultDB != db || resolver.runtime != nil || !resolver.loadedAt.IsZero() || store.repository == nil {
		t.Fatal("构造函数未保持未初始化状态或仓储未装配")
	}
	if resolver.storagePolicyRepository == nil || resolver.outputPolicyRepository == nil || resolver.ruleRepository == nil || resolver.store == nil {
		t.Fatal("解析器仓储未完整装配")
	}
	ctx := redact.WithDirection(redact.WithOperation(context.Background(), "/example/Get"), redact.DirectionResponse)
	resolver.Resolve(ctx, "example.Message.phone")
	resolver.ListStoragePolicies(ctx, "storage_callback_test")
	if queries != 0 || len(resolver.storagePolicies) != 0 || len(resolver.outputPolicies) != 0 {
		t.Fatalf("迁移前不应查询或加载策略: queries=%d", queries)
	}
	assertNoStorageCallbacks(t, db)
}

// TestRedactInitializeRetriesAndIsIdempotent 验证查询和密钥失败可重试，成功后并发重复初始化无副作用。
func TestRedactInitializeRetriesAndIsIdempotent(t *testing.T) {
	previousKey := sdk.Runtime.GetKey()
	t.Cleanup(func() { sdk.Runtime.SetKey(previousKey) })
	keyFailure := errors.New("测试密钥失败")
	key := &redactTestKey{secret: "test-storage-key", err: keyFailure}
	sdk.Runtime.SetKey(key)
	db := newRedactTestDB(t)
	queryFailure := errors.New("测试查询失败")
	failQuery := true
	queries := 0
	err := db.Callback().Query().Before("gorm:query").Register("test:query-failure", func(tx *gorm.DB) {
		queries++
		if failQuery {
			tx.AddError(queryFailure)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	var resolver *RedactPolicyResolver
	resolver, err = NewRedactPolicyResolver(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	err = resolver.Initialize(context.Background())
	if !errors.Is(err, queryFailure) || resolver.runtime != nil || key.calls != 0 {
		t.Fatalf("查询失败不应初始化运行时或密钥: %v", err)
	}
	assertNoStorageCallbacks(t, db)
	failQuery = false
	err = resolver.Initialize(context.Background())
	if !errors.Is(err, keyFailure) || resolver.runtime != nil {
		t.Fatalf("密钥失败不应绑定运行时: %v", err)
	}
	assertNoStorageCallbacks(t, db)
	key.err = nil
	err = resolver.Initialize(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resolver.runtime == nil {
		t.Fatal("初始化成功后必须持有实例运行时")
	}
	queryCount := queries
	keyCalls := key.calls
	initialized := resolver.runtime
	results := make(chan error, 8)
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			results <- resolver.Initialize(context.Background())
		}()
	}
	group.Wait()
	close(results)
	for err = range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if queries != queryCount || key.calls != keyCalls || resolver.runtime != initialized {
		t.Fatal("重复初始化不应查询、派生密钥或替换运行时")
	}
	var other *RedactPolicyResolver
	other, err = NewRedactPolicyResolver(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db.Session(&gorm.Session{NewDB: true})}})
	if err != nil {
		t.Fatal(err)
	}
	err = other.Initialize(context.Background())
	if err == nil || !strings.Contains(err.Error(), "已注册脱敏回调") || other.runtime != nil || resolver.runtime != initialized {
		t.Fatalf("同库不同解析器不得覆盖已有回调: %v", err)
	}
}

// TestStorageCallbacksIsolateDatabases 验证两库使用各自策略和密钥，派生会话不丢失回调。
func TestStorageCallbacksIsolateDatabases(t *testing.T) {
	previousKey := sdk.Runtime.GetKey()
	t.Cleanup(func() { sdk.Runtime.SetKey(previousKey) })
	var resolvers []*RedactPolicyResolver
	var protectors []*redact.StorageProtector
	var err error
	for _, secret := range []string{"first-instance-key", "second-instance-key"} {
		sdk.Runtime.SetKey(&redactTestKey{secret: secret})
		var resolver *RedactPolicyResolver
		resolver, err = NewRedactPolicyResolver(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: newRedactTestDB(t)}})
		if err != nil {
			t.Fatal(err)
		}
		err = resolver.Initialize(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		var protector *redact.StorageProtector
		protector, err = redact.NewStorageProtector(base64.RawStdEncoding.EncodeToString([]byte(secret)))
		if err != nil {
			t.Fatal(err)
		}
		resolvers = append(resolvers, resolver)
		protectors = append(protectors, protector)
	}
	for index, resolver := range resolvers {
		mask := strings.Repeat("*", index+1)
		policy := redact.StorageFieldPolicy{
			ID: 1, TableName: "storage_callback_test", ColumnName: "phone",
			Rule: redact.FieldPolicy{Mode: redact.PolicyModeApplyRule, Transform: func(any) any { return mask }},
		}
		resolver.storagePolicies[storagePolicyKey(kitgorm.DefaultClientName, policy.TableName)] = []redact.StorageFieldPolicy{policy}
		resolver.loadedAt = time.Now()
		for _, db := range []*gorm.DB{resolver.defaultDB, resolver.defaultDB.WithContext(context.Background()), resolver.defaultDB.Session(&gorm.Session{}), resolver.defaultDB.Session(&gorm.Session{NewDB: true})} {
			entity := &storageCallbackTestEntity{ID: 42, Phone: "13800138000"}
			result := db.Create(entity)
			if result.Error != nil {
				t.Fatal(result.Error)
			}
			if entity.Phone != mask {
				t.Fatalf("实例 %d 的策略被覆盖: %q", index, entity.Phone)
			}
			value, ok := result.InstanceGet(storagePreparedStateKey)
			if !ok {
				t.Fatal("派生会话没有保存入库状态")
			}
			state := value.(*preparedState)
			ciphertext := string(state.entities[0].values[1].Ciphertext)
			var plain string
			plain, err = protectors[index].Decrypt(ciphertext, "storage-policy\x001")
			if err != nil || plain != "13800138000" {
				t.Fatalf("实例 %d 的密钥或原文错误: %q %v", index, plain, err)
			}
			_, err = protectors[1-index].Decrypt(ciphertext, "storage-policy\x001")
			if err == nil {
				t.Fatal("另一个实例的密钥不应能解密当前实例数据")
			}
		}
	}
	unbound := newRedactTestDB(t)
	assertNoStorageCallbacks(t, unbound)
}

// TestStorageCallbackRegistrationRollsBack 验证排序冲突会撤销所有本次回调，修复后可重试。
func TestStorageCallbackRegistrationRollsBack(t *testing.T) {
	db := newRedactTestDB(t)
	err := db.Callback().Create().After("gorm:before_create").Before("kratos-admin:redact/create").Register("test:conflict", func(*gorm.DB) {})
	if err != nil {
		t.Fatal(err)
	}
	runtime := &storageRuntime{}
	err = runtime.registerCallbacks(db)
	if err == nil || !strings.Contains(err.Error(), "注册脱敏回调 kratos-admin:redact/create 失败") || runtime.db != nil {
		t.Fatalf("应报告具体失败回调且不绑定数据库: %v", err)
	}
	assertNoStorageCallbacks(t, db)
	if db.Callback().Create().Get("test:conflict") == nil {
		t.Fatal("撤销不应删除其他注册者的回调")
	}
	err = db.Callback().Create().Remove("test:conflict")
	if err != nil {
		t.Fatal(err)
	}
	err = runtime.registerCallbacks(db)
	if err != nil {
		t.Fatal(err)
	}
	err = runtime.registerCallbacks(db.Session(&gorm.Session{}))
	if err != nil {
		t.Fatalf("同一运行时和数据库重复绑定应无副作用: %v", err)
	}
	err = runtime.registerCallbacks(newRedactTestDB(t))
	if err == nil {
		t.Fatal("已绑定运行时不应复用到另一数据库")
	}
}

// TestStorageCallbackErrorsPrecedeCommit 验证创建、更新和删除的旁表错误在提交节点前可见。
func TestStorageCallbackErrorsPrecedeCommit(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			db := newRedactTestDB(t)
			resolver, err := NewRedactPolicyResolver(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
			if err != nil {
				t.Fatal(err)
			}
			runtime := newStorageRuntime(resolver.store, resolver, nil)
			err = runtime.registerCallbacks(db)
			if err != nil {
				t.Fatal(err)
			}
			processor := db.Callback().Create()
			if operation == "update" {
				processor = db.Callback().Update()
			}
			if operation == "delete" {
				processor = db.Callback().Delete()
			}
			stateKey := storagePreparedStateKey
			if operation == "delete" {
				stateKey = storageDeleteStateKey
			}
			// 注入错误状态，让真正的旁表回调通过 AddError 阻止提交。
			err = processor.Before("kratos-admin:redact/"+operation+"-after").Register("test:invalid-storage-state", func(tx *gorm.DB) {
				tx.InstanceSet(stateKey, "invalid-state")
			})
			if err != nil {
				t.Fatal(err)
			}
			commitCalled := false
			err = processor.Register("gorm:commit_or_rollback_transaction", func(tx *gorm.DB) {
				commitCalled = true
				if tx.Error == nil || !strings.Contains(tx.Error.Error(), "状态类型无效") {
					t.Errorf("到达提交节点前必须已记录旁表错误: %v", tx.Error)
				}
			})
			if err != nil {
				t.Fatal(err)
			}
			entity := &storageCallbackTestEntity{ID: 42, Phone: "13800138000"}
			var result *gorm.DB
			switch operation {
			case "create":
				result = db.Create(entity)
			case "update":
				result = db.Model(entity).Updates(entity)
			case "delete":
				result = db.Delete(entity)
			}
			if !commitCalled || result.Error == nil {
				t.Fatalf("旁表错误必须在提交前传播: commit=%v err=%v", commitCalled, result.Error)
			}
		})
	}
}

// newRedactTestDB 创建不连接数据库、不执行 SQL 的 GORM 测试客户端。
func newRedactTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: &sql.DB{}, SkipInitializeWithVersion: true}), &gorm.Config{
		DisableAutomaticPing:   true,
		DryRun:                 true,
		SkipDefaultTransaction: true,
		NamingStrategy:         schema.NamingStrategy{SingularTable: true},
		Logger:                 logger.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

// assertNoStorageCallbacks 校验数据库没有残留任何脱敏回调。
func assertNoStorageCallbacks(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, group := range []struct {
		operation string
		get       func(string) func(*gorm.DB)
	}{
		{"query", db.Callback().Query().Get},
		{"create", db.Callback().Create().Get},
		{"update", db.Callback().Update().Get},
		{"delete", db.Callback().Delete().Get},
	} {
		for _, suffix := range []string{"", "-after"} {
			name := "kratos-admin:redact/" + group.operation + suffix
			if group.get(name) != nil {
				t.Fatalf("数据库不应残留回调 %s", name)
			}
		}
	}
}
