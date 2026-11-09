package kit

import (
	"context"
	"errors"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	query "github.com/liujitcn/kratos-admin/backend/internal/data/gen/query"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"github.com/liujitcn/kratos-kit/redact"
	"gorm.io/gorm"
)

var _ redact.StorageValueStore = (*StorageValueStore)(nil)

// StorageValueStore 将敏感值旁表仓储适配为运行时存储接口。
type StorageValueStore struct {
	repository *data.BaseRedactStorageValueRepository
}

// NewStorageValueStore 使用默认数据库构造旁表仓储，不执行数据库查询。
func NewStorageValueStore(databases map[string]*kitgorm.Client) (*StorageValueStore, error) {
	d, err := data.NewData(databases)
	if err != nil {
		return nil, err
	}
	return &StorageValueStore{repository: data.NewBaseRedactStorageValueRepository(d)}, nil
}

// Find 查询指定入库策略和业务记录的旁表敏感值。
func (s *StorageValueStore) Find(ctx context.Context, tenantID, storagePolicyID, recordID int64) (*redact.StorageValue, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("敏感字段旁表仓储未初始化")
	}
	query := s.queryFor(ctx).BaseRedactStorageValue
	value, err := query.WithContext(ctx).Where(
		query.TenantID.Eq(tenantID),
		query.StoragePolicyID.Eq(storagePolicyID),
		query.RecordID.Eq(recordID),
	).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, redact.ErrStorageValueNotFound
	}
	if err != nil {
		return nil, err
	}
	return toStorageValue(value), nil
}

// ListByRecords 批量查询指定入库策略和业务记录的旁表敏感值。
func (s *StorageValueStore) ListByRecords(ctx context.Context, tenantID, storagePolicyID int64, recordIDs []int64) ([]*redact.StorageValue, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("敏感字段旁表仓储未初始化")
	}
	if len(recordIDs) == 0 {
		return []*redact.StorageValue{}, nil
	}
	query := s.queryFor(ctx).BaseRedactStorageValue
	values, err := query.WithContext(ctx).Where(
		query.TenantID.Eq(tenantID),
		query.StoragePolicyID.Eq(storagePolicyID),
		query.RecordID.In(recordIDs...),
	).Find()
	if err != nil {
		return nil, err
	}
	return toStorageValues(values), nil
}

// ListByDigest 按入库策略和查询摘要查找业务记录。
func (s *StorageValueStore) ListByDigest(ctx context.Context, tenantID, storagePolicyID int64, digest []byte) ([]*redact.StorageValue, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("敏感字段旁表仓储未初始化")
	}
	query := s.queryFor(ctx).BaseRedactStorageValue
	values, err := query.WithContext(ctx).Where(
		query.TenantID.Eq(tenantID),
		query.StoragePolicyID.Eq(storagePolicyID),
		query.Digest.Eq(digest),
	).Find()
	if err != nil {
		return nil, err
	}
	return toStorageValues(values), nil
}

// Save 保存或更新旁表敏感值。
func (s *StorageValueStore) Save(ctx context.Context, value *redact.StorageValue) error {
	if s == nil || s.repository == nil {
		return errors.New("敏感字段旁表仓储未初始化")
	}
	if value == nil {
		return nil
	}
	base := s.queryFor(ctx).BaseRedactStorageValue
	existing, err := base.WithContext(ctx).Where(base.TenantID.Eq(value.TenantID), base.StoragePolicyID.Eq(value.StoragePolicyID), base.RecordID.Eq(value.RecordID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return base.WithContext(ctx).Create(toStorageValueModel(value))
	}
	if err != nil {
		return err
	}
	model := toStorageValueModel(value)
	model.ID = existing.ID
	return base.WithContext(ctx).Save(model)
}

// Delete 物理删除旁表敏感值。
func (s *StorageValueStore) Delete(ctx context.Context, value *redact.StorageValue) error {
	if s == nil || s.repository == nil {
		return errors.New("敏感字段旁表仓储未初始化")
	}
	if value == nil {
		return nil
	}
	result, err := s.queryFor(ctx).BaseRedactStorageValue.WithContext(ctx).Unscoped().Delete(toStorageValueModel(value))
	if err != nil {
		return err
	}
	return result.Error
}

// SaveWithDB 使用当前 GORM 事务保存旁表敏感值。
func (s *StorageValueStore) SaveWithDB(ctx context.Context, db *gorm.DB, value *redact.StorageValue) error {
	if s == nil || s.repository == nil {
		return errors.New("敏感字段旁表仓储未初始化")
	}
	if db == nil || value == nil {
		return nil
	}
	base := queryForDB(db).BaseRedactStorageValue
	existing, err := base.WithContext(ctx).Where(
		base.TenantID.Eq(value.TenantID),
		base.StoragePolicyID.Eq(value.StoragePolicyID),
		base.RecordID.Eq(value.RecordID),
	).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return base.WithContext(ctx).Create(toStorageValueModel(value))
	}
	if err != nil {
		return err
	}
	model := toStorageValueModel(value)
	model.ID = existing.ID
	return base.WithContext(ctx).Save(model)
}

// DeleteWithDB 使用当前 GORM 事务物理删除旁表敏感值。
func (s *StorageValueStore) DeleteWithDB(ctx context.Context, db *gorm.DB, tenantID, storagePolicyID, recordID int64) error {
	if s == nil || s.repository == nil {
		return errors.New("敏感字段旁表仓储未初始化")
	}
	if db == nil || tenantID <= 0 || storagePolicyID <= 0 || recordID <= 0 {
		return nil
	}
	base := queryForDB(db).BaseRedactStorageValue
	value, err := base.WithContext(ctx).Where(
		base.TenantID.Eq(tenantID),
		base.StoragePolicyID.Eq(storagePolicyID),
		base.RecordID.Eq(recordID),
	).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = base.WithContext(ctx).Unscoped().Delete(value)
	return err
}

// queryForDB 返回绑定指定 GORM 连接的旁表查询对象。
func queryForDB(db *gorm.DB) *query.Query {
	// Model 会先实例化 NewDB 会话，避免 gorm/gen 的 UseDB 再次克隆回主表 Statement。
	storageDB := db.Session(&gorm.Session{NewDB: true}).Model(&models.BaseRedactStorageValue{})
	// 旁表按策略所属租户存储（全局策略可保护其他租户）；上层先授权主记录，再按策略、记录编号精确读取。
	return query.Use(kitgorm.SkipDataIsolation(storageDB))
}

// toStorageValues 将 Admin 模型列表转换为通用敏感值列表。
func toStorageValues(values []*models.BaseRedactStorageValue) []*redact.StorageValue {
	result := make([]*redact.StorageValue, 0, len(values))
	for _, value := range values {
		result = append(result, toStorageValue(value))
	}
	return result
}

// toStorageValue 将 Admin 模型转换为通用敏感值。
func toStorageValue(value *models.BaseRedactStorageValue) *redact.StorageValue {
	if value == nil {
		return nil
	}
	return &redact.StorageValue{
		ID:              value.ID,
		TenantID:        value.TenantID,
		StoragePolicyID: value.StoragePolicyID,
		RecordID:        value.RecordID,
		Ciphertext:      value.Ciphertext,
		Digest:          value.Digest,
	}
}

// toStorageValueModel 将通用敏感值转换为 Admin 模型。
func toStorageValueModel(value *redact.StorageValue) *models.BaseRedactStorageValue {
	if value == nil {
		return nil
	}
	return &models.BaseRedactStorageValue{
		ID:              value.ID,
		TenantID:        value.TenantID,
		StoragePolicyID: value.StoragePolicyID,
		RecordID:        value.RecordID,
		Ciphertext:      value.Ciphertext,
		Digest:          value.Digest,
	}
}

// storageTransactionKey 在回调和回填中携带真实事务连接，旁表读取不能跳出主表事务。
type storageTransactionKey struct{}

// queryFor 在事务内共用同一连接，普通查询沿用生成仓储的默认入口。
func (s *StorageValueStore) queryFor(ctx context.Context) *query.Query {
	if transaction, ok := ctx.Value(storageTransactionKey{}).(*gorm.DB); ok {
		return queryForDB(transaction)
	}
	return queryForDB(s.repository.Query(ctx).BaseRedactStorageValue.WithContext(ctx).UnderlyingDB()) //nolint:forbidigo // 旁表按策略租户读取，需保留事务并单独移除隔离回调；上层先授权主记录。
}
