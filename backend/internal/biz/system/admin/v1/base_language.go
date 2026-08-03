package biz

import (
	"context"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	commonv1 "github.com/liujitcn/kratos-admin/backend/core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreLocale "github.com/liujitcn/kratos-admin/backend/core/pkg/locale"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm/clause"
)

// BaseLanguageCase 语言管理业务实例。
type BaseLanguageCase struct {
	*biz.BaseCase
	*data.BaseLanguageRepository
	tx         data.Transaction
	formMapper *mapper.CopierMapper[systemadminv1.BaseLanguageForm, models.BaseLanguage]
	mapper     *mapper.CopierMapper[systemadminv1.BaseLanguage, models.BaseLanguage]
}

// NewBaseLanguageCase 创建语言管理业务实例。
func NewBaseLanguageCase(baseCase *biz.BaseCase, tx data.Transaction, baseLanguageRepo *data.BaseLanguageRepository) *BaseLanguageCase {
	return &BaseLanguageCase{
		BaseCase:               baseCase,
		BaseLanguageRepository: baseLanguageRepo,
		tx:                     tx,
		formMapper:             mapper.NewCopierMapper[systemadminv1.BaseLanguageForm, models.BaseLanguage](),
		mapper:                 mapper.NewCopierMapper[systemadminv1.BaseLanguage, models.BaseLanguage](),
	}
}

// OptionBaseLanguage 查询语言选项。
func (c *BaseLanguageCase) OptionBaseLanguage(ctx context.Context, req *systemadminv1.OptionBaseLanguageRequest) (*systemadminv1.OptionBaseLanguageResponse, error) {
	query := c.Query(ctx).BaseLanguage
	opts := make([]repository.QueryOption, 0, 2)
	if req.GetEnabledOnly() {
		opts = append(opts, repository.Where(query.Status.Eq(int32(commonv1.Status_ENABLE))))
	}
	opts = append(opts, repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*systemadminv1.BaseLanguage, 0, len(list))
	for _, item := range list {
		items = append(items, c.mapper.ToDTO(item))
	}
	return &systemadminv1.OptionBaseLanguageResponse{BaseLanguages: items}, nil
}

// PageBaseLanguage 分页查询语言列表。
func (c *BaseLanguageCase) PageBaseLanguage(ctx context.Context, req *systemadminv1.PageBaseLanguageRequest) (*systemadminv1.PageBaseLanguageResponse, error) {
	query := c.Query(ctx).BaseLanguage
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc()))
	if req.GetLanguageName() != "" {
		opts = append(opts, repository.Where(query.LanguageName.Like("%"+req.GetLanguageName()+"%")))
	}
	if req.GetLanguageCode() != "" {
		opts = append(opts, repository.Where(query.LanguageCode.Like("%"+req.GetLanguageCode()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*systemadminv1.BaseLanguage, 0, len(list))
	for _, item := range list {
		items = append(items, c.mapper.ToDTO(item))
	}
	return &systemadminv1.PageBaseLanguageResponse{BaseLanguages: items, Total: int32(total)}, nil
}

// GetBaseLanguage 查询语言详情。
func (c *BaseLanguageCase) GetBaseLanguage(ctx context.Context, id int64) (*systemadminv1.BaseLanguageForm, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(item), nil
}

// CreateBaseLanguage 创建语言。
func (c *BaseLanguageCase) CreateBaseLanguage(ctx context.Context, req *systemadminv1.BaseLanguageForm) error {
	item := c.formMapper.ToEntity(req)
	if !coreLocale.IsSupported(item.LanguageCode) {
		return errorsx.InvalidArgument("语言代码必须是系统支持的语言")
	}
	item.IsPrimary = false
	return c.createLanguage(ctx, item)
}

// UpdateBaseLanguage 更新语言。
func (c *BaseLanguageCase) UpdateBaseLanguage(ctx context.Context, req *systemadminv1.BaseLanguageForm) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		current, err := c.findBaseLanguageForUpdate(ctx, req.GetId())
		if err != nil {
			return err
		}
		if !coreLocale.IsSupported(req.GetLanguageCode()) {
			return errorsx.InvalidArgument("语言代码必须是系统支持的语言")
		}
		if current.IsPrimary && req.GetStatus() == commonv1.Status_DISABLE {
			return errorsx.ProtectedResourceConflict("主语言不能禁用", "base_language")
		}
		if current.IsPrimary && req.GetLanguageCode() != current.LanguageCode {
			return errorsx.ProtectedResourceConflict("主语言代码不允许修改", "base_language")
		}
		item := c.formMapper.ToEntity(req)
		item.ID = current.ID
		item.LanguageCode = current.LanguageCode
		item.IsPrimary = current.IsPrimary
		return c.updateLanguage(ctx, item)
	})
}

// findBaseLanguageForUpdate 查询并锁定待修改的语言记录。
func (c *BaseLanguageCase) findBaseLanguageForUpdate(ctx context.Context, id int64) (*models.BaseLanguage, error) {
	query := c.Query(ctx).BaseLanguage
	opts := []repository.QueryOption{
		repository.Where(query.ID.Eq(id)),
		repository.Clauses(clause.Locking{Strength: "UPDATE"}),
	}
	return c.Find(ctx, opts...)
}

// DeleteBaseLanguage 删除语言。
func (c *BaseLanguageCase) DeleteBaseLanguage(ctx context.Context, ids string) error {
	idList := _string.ConvertStringToInt64Array(ids)
	if len(idList) == 0 {
		return nil
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		for _, id := range idList {
			item, err := c.findBaseLanguageForUpdate(ctx, id)
			if err != nil {
				return err
			}
			if item.IsPrimary {
				return errorsx.ProtectedResourceConflict("主语言不允许删除", "base_language")
			}
		}
		return c.DeleteByIDs(ctx, idList)
	})
}

// SetBaseLanguageStatus 设置语言启用状态。
func (c *BaseLanguageCase) SetBaseLanguageStatus(ctx context.Context, req *systemadminv1.SetBaseLanguageStatusRequest) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		item, err := c.findBaseLanguageForUpdate(ctx, req.GetId())
		if err != nil {
			return err
		}
		if item.IsPrimary && req.GetStatus() == commonv1.Status_DISABLE {
			return errorsx.ProtectedResourceConflict("主语言不允许禁用", "base_language")
		}
		return c.UpdateByID(ctx, &models.BaseLanguage{ID: item.ID, Status: int32(req.GetStatus())})
	})
}

// SetBaseLanguagePrimary 设置主语言并清除其他主语言标记。
func (c *BaseLanguageCase) SetBaseLanguagePrimary(ctx context.Context, req *systemadminv1.SetBaseLanguagePrimaryRequest) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		var err error
		err = c.clearPrimaryLanguage(ctx, req.GetId())
		if err != nil {
			return err
		}
		var item *models.BaseLanguage
		item, err = c.findBaseLanguageForUpdate(ctx, req.GetId())
		if err != nil {
			return err
		}
		if item.Status != int32(commonv1.Status_ENABLE) {
			return errorsx.ProtectedResourceConflict("禁用语言不能设为主语言", "base_language")
		}
		if item.IsPrimary {
			return nil
		}
		return c.UpdateByID(ctx, &models.BaseLanguage{ID: item.ID, IsPrimary: true})
	})
}

// createLanguage 创建语言并将数据库唯一冲突转换为稳定业务错误。
func (c *BaseLanguageCase) createLanguage(ctx context.Context, item *models.BaseLanguage) error {
	if err := c.Create(ctx, item); err != nil {
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("语言代码重复", "base_language", "language_code", "unique_base_language").WithCause(err)
		}
		return err
	}
	return nil
}

// updateLanguage 更新语言并将数据库唯一冲突转换为稳定业务错误。
func (c *BaseLanguageCase) updateLanguage(ctx context.Context, item *models.BaseLanguage) error {
	if err := c.UpdateByID(ctx, item); err != nil {
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("语言代码重复", "base_language", "language_code", "unique_base_language").WithCause(err)
		}
		return err
	}
	return nil
}

// clearPrimaryLanguage 在事务中清除其他语言的主语言标记。
func (c *BaseLanguageCase) clearPrimaryLanguage(ctx context.Context, keepID int64) error {
	opts := []repository.QueryOption{repository.Clauses(clause.Locking{Strength: "UPDATE"})}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return err
	}
	for _, item := range list {
		if item.ID == keepID || !item.IsPrimary {
			continue
		}
		item.IsPrimary = false
		if err = c.UpdateByID(ctx, item); err != nil {
			return err
		}
	}
	return nil
}
