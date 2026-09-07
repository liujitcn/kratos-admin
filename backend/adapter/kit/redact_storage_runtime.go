package kit

import (
	"errors"
	"fmt"

	"github.com/liujitcn/kratos-kit/redact"
	"gorm.io/gorm"
)

type storageRuntime struct {
	db       *gorm.DB
	storage  *redact.RedactStorage
	resolver *RedactPolicyResolver
	store    *StorageValueStore
}

// newStorageRuntime 构造仅属于当前解析器的敏感字段存储运行时。
func newStorageRuntime(store *StorageValueStore, resolver *RedactPolicyResolver, protector *redact.StorageProtector) *storageRuntime {
	return &storageRuntime{
		storage:  redact.NewRedactStorage(store, resolver, protector, gormEntityFieldAccessor{}),
		resolver: resolver,
		store:    store,
	}
}

// registerCallbacks 在启动阶段绑定当前数据库回调，拒绝覆盖并撤销失败的注册。
func (r *storageRuntime) registerCallbacks(db *gorm.DB) error {
	if r.db != nil {
		if r.db.Callback() != db.Callback() {
			return errors.New("脱敏存储运行时已绑定其他数据库")
		}
		return nil
	}
	callbacks := []struct {
		name     string
		handler  func(*gorm.DB)
		get      func(string) func(*gorm.DB)
		register func(string, func(*gorm.DB)) error
		remove   func(string) error
	}{
		{"kratos-admin:redact/query", r.rewriteStorageQuery, db.Callback().Query().Get, db.Callback().Query().Before("gorm:query").Register, db.Callback().Query().Remove},
		{"kratos-admin:redact/query-after", r.materializeStorageResponse, db.Callback().Query().Get, db.Callback().Query().After("gorm:after_query").Register, db.Callback().Query().Remove},
		{"kratos-admin:redact/create", r.prepareStorageCreate, db.Callback().Create().Get, db.Callback().Create().Before("gorm:before_create").Register, db.Callback().Create().Remove},
		{"kratos-admin:redact/create-after", r.saveStorageValues, db.Callback().Create().Get, db.Callback().Create().After("gorm:after_create").Before("gorm:commit_or_rollback_transaction").Register, db.Callback().Create().Remove},
		{"kratos-admin:redact/update", r.prepareStorageUpdate, db.Callback().Update().Get, db.Callback().Update().Before("gorm:update").Register, db.Callback().Update().Remove},
		{"kratos-admin:redact/update-after", r.saveStorageValues, db.Callback().Update().Get, db.Callback().Update().After("gorm:after_update").Before("gorm:commit_or_rollback_transaction").Register, db.Callback().Update().Remove},
		{"kratos-admin:redact/delete", r.captureStorageDelete, db.Callback().Delete().Get, db.Callback().Delete().Before("gorm:delete").Register, db.Callback().Delete().Remove},
		{"kratos-admin:redact/delete-after", r.deleteStorageValues, db.Callback().Delete().Get, db.Callback().Delete().After("gorm:after_delete").Before("gorm:commit_or_rollback_transaction").Register, db.Callback().Delete().Remove},
	}
	for _, callback := range callbacks {
		if callback.get(callback.name) != nil {
			return fmt.Errorf("数据库已注册脱敏回调 %s，不能绑定其他解析器", callback.name)
		}
	}
	var err error
	for index, callback := range callbacks {
		err = callback.register(callback.name, callback.handler)
		if err != nil {
			err = fmt.Errorf("注册脱敏回调 %s 失败: %w", callback.name, err)
			// GORM 先追加回调再编译，失败项也必须一并撤销。
			for rollbackIndex := index; rollbackIndex >= 0; rollbackIndex-- {
				registered := callbacks[rollbackIndex]
				rollbackErr := registered.remove(registered.name)
				if rollbackErr != nil {
					err = errors.Join(err, fmt.Errorf("撤销脱敏回调 %s 失败: %w", registered.name, rollbackErr))
				}
			}
			return err
		}
	}
	r.db = db
	return nil
}
