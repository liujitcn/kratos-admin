package kit

import (
	"context"
	"errors"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	query "github.com/liujitcn/kratos-admin/backend/internal/data/gen/query"
	"github.com/liujitcn/kratos-kit/redact"
	"gorm.io/gorm"
)

var _ redact.StorageValueStore = (*storageValueStore)(nil)

// storageValueStore 将敏感值旁表仓储适配为运行时存储接口。
type storageValueStore struct {
	repository *data.BaseRedactStorageValueRepository
}

// NewStorageValueStore 创建敏感值旁表存储适配器。
func NewStorageValueStore(repository *data.BaseRedactStorageValueRepository) *storageValueStore {
	return &storageValueStore{repository: repository}
}

// Find 查询指定入库策略和业务记录的旁表敏感值。
func (s *storageValueStore) Find(ctx context.Context, storagePolicyID, recordID int64) (*redact.StorageValue, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("敏感字段旁表仓储未初始化")
	}
	query := s.repository.Query(ctx).BaseRedactStorageValue
	value, err := s.repository.Find(ctx,
		repository.Where(query.StoragePolicyID.Eq(storagePolicyID)),
		repository.Where(query.RecordID.Eq(recordID)),
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, redact.ErrStorageValueNotFound
	}
	if err != nil {
		return nil, err
	}
	return toStorageValue(value), nil
}

// ListByRecords 批量查询指定入库策略和业务记录的旁表敏感值。
func (s *storageValueStore) ListByRecords(ctx context.Context, storagePolicyID int64, recordIDs []int64) ([]*redact.StorageValue, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("敏感字段旁表仓储未初始化")
	}
	if len(recordIDs) == 0 {
		return []*redact.StorageValue{}, nil
	}
	query := s.repository.Query(ctx).BaseRedactStorageValue
	values, err := s.repository.List(ctx,
		repository.Where(query.StoragePolicyID.Eq(storagePolicyID)),
		repository.Where(query.RecordID.In(recordIDs...)),
	)
	if err != nil {
		return nil, err
	}
	return toStorageValues(values), nil
}

// ListByDigest 按入库策略和查询摘要查找业务记录。
func (s *storageValueStore) ListByDigest(ctx context.Context, storagePolicyID int64, digest []byte) ([]*redact.StorageValue, error) {
	if s == nil || s.repository == nil {
		return nil, errors.New("敏感字段旁表仓储未初始化")
	}
	query := s.repository.Query(ctx).BaseRedactStorageValue
	values, err := s.repository.List(ctx,
		repository.Where(query.StoragePolicyID.Eq(storagePolicyID)),
		repository.Where(query.Digest.Eq(digest)),
	)
	if err != nil {
		return nil, err
	}
	return toStorageValues(values), nil
}

// Save 保存或更新旁表敏感值。
func (s *storageValueStore) Save(ctx context.Context, value *redact.StorageValue) error {
	if s == nil || s.repository == nil {
		return errors.New("敏感字段旁表仓储未初始化")
	}
	if value == nil {
		return nil
	}
	query := s.repository.Query(ctx).BaseRedactStorageValue
	existing, err := s.repository.Find(ctx,
		repository.Where(query.StoragePolicyID.Eq(value.StoragePolicyID)),
		repository.Where(query.RecordID.Eq(value.RecordID)),
	)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.repository.Create(ctx, toStorageValueModel(value))
	}
	if err != nil {
		return err
	}
	model := toStorageValueModel(value)
	model.ID = existing.ID
	return s.repository.UpdateByID(ctx, model)
}

// Delete 物理删除旁表敏感值。
func (s *storageValueStore) Delete(ctx context.Context, value *redact.StorageValue) error {
	if s == nil || s.repository == nil {
		return errors.New("敏感字段旁表仓储未初始化")
	}
	if value == nil {
		return nil
	}
	result, err := s.repository.Query(ctx).BaseRedactStorageValue.WithContext(ctx).Unscoped().Delete(toStorageValueModel(value))
	if err != nil {
		return err
	}
	return result.Error
}

// SaveWithDB 使用当前 GORM 事务保存旁表敏感值。
func (s *storageValueStore) SaveWithDB(ctx context.Context, db *gorm.DB, value *redact.StorageValue) error {
	if s == nil || s.repository == nil {
		return errors.New("敏感字段旁表仓储未初始化")
	}
	if db == nil || value == nil {
		return nil
	}
	base := queryForDB(db).BaseRedactStorageValue
	existing, err := base.WithContext(ctx).Where(
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
func (s *storageValueStore) DeleteWithDB(ctx context.Context, db *gorm.DB, storagePolicyID, recordID int64) error {
	if s == nil || s.repository == nil {
		return errors.New("敏感字段旁表仓储未初始化")
	}
	if db == nil || storagePolicyID <= 0 || recordID <= 0 {
		return nil
	}
	base := queryForDB(db).BaseRedactStorageValue
	value, err := base.WithContext(ctx).Where(
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
	return query.Use(storageDB)
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
		StoragePolicyID: value.StoragePolicyID,
		RecordID:        value.RecordID,
		Ciphertext:      value.Ciphertext,
		Digest:          value.Digest,
	}
}
