package biz

import (
	"context"
	"strings"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"golang.org/x/text/language"
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
		opts = append(opts, repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)))
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
	if _, err := language.Parse(item.LanguageCode); err != nil {
		return errorsx.InvalidArgument("语言代码必须是有效的语言代码").WithCause(err)
	}
	item.IsPrimary = false
	if err := c.Create(ctx, item); err != nil {
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("语言代码重复", "base_language", "language_code", "unique_base_language").WithCause(err)
		}
		return err
	}
	return nil
}

// UpdateBaseLanguage 更新语言。
func (c *BaseLanguageCase) UpdateBaseLanguage(ctx context.Context, req *systemadminv1.BaseLanguageForm) error {
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		current, err := c.findBaseLanguageForUpdate(ctx, req.GetId())
		if err != nil {
			return err
		}
		if _, err = language.Parse(req.GetLanguageCode()); err != nil {
			return errorsx.InvalidArgument("语言代码必须是有效的语言代码").WithCause(err)
		}
		if current.IsPrimary && req.GetStatus() == commonv1.Status_STATUS_DISABLE {
			return errorsx.ProtectedResourceConflict("主语言不能禁用", "base_language")
		}
		if current.IsPrimary && req.GetLanguageCode() != current.LanguageCode {
			return errorsx.ProtectedResourceConflict("主语言代码不允许修改", "base_language")
		}
		item := c.formMapper.ToEntity(req)
		item.ID = current.ID
		item.LanguageCode = current.LanguageCode
		item.IsPrimary = current.IsPrimary
		if err = c.UpdateByID(ctx, item); err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("语言代码重复", "base_language", "language_code", "unique_base_language").WithCause(err)
			}
			return err
		}
		return nil
	})
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
		if item.IsPrimary && req.GetStatus() == commonv1.Status_STATUS_DISABLE {
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
		if item.Status != coreconst.STATUS_STATUS_ENABLE {
			return errorsx.ProtectedResourceConflict("禁用语言不能设为主语言", "base_language")
		}
		if item.IsPrimary {
			return nil
		}
		return c.UpdateByID(ctx, &models.BaseLanguage{ID: item.ID, IsPrimary: true})
	})
}

// LocaleState 查询当前请求对应的运行时语言状态。
func (c *BaseLanguageCase) LocaleState(ctx context.Context) (*dto.LocaleState, error) {
	query := c.Query(ctx).BaseLanguage
	opts := []repository.QueryOption{
		repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	state := &dto.LocaleState{
		Current: biz.LocaleFromContext(ctx),
		Enabled: make([]string, 0, len(list)),
	}
	for _, item := range list {
		state.Enabled = append(state.Enabled, item.LanguageCode)
		if item.IsPrimary {
			state.Primary = item.LanguageCode
		}
	}
	if state.Primary == "" {
		if len(state.Enabled) > 0 {
			state.Primary = state.Enabled[0]
		}
	}
	return state, nil
}

// ResolveLocale 根据请求语言头和启用语言配置解析当前语言与主语言。
func (c *BaseLanguageCase) ResolveLocale(ctx context.Context, acceptLanguage string) (string, string, error) {
	query := c.Query(ctx).BaseLanguage
	opts := []repository.QueryOption{
		repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	}
	list, err := c.List(ctx, opts...)
	if err != nil {
		return "", "", err
	}
	if len(list) == 0 {
		return "", "", nil
	}
	primary := list[0].LanguageCode
	locales := make([]string, 0, len(list))
	for _, item := range list {
		locales = append(locales, item.LanguageCode)
		if item.IsPrimary {
			primary = item.LanguageCode
		}
	}
	acceptLanguage = strings.ReplaceAll(strings.TrimSpace(acceptLanguage), "_", "-")
	if acceptLanguage == "" {
		return primary, primary, nil
	}
	var tags []language.Tag
	var parseErr error
	tags, _, parseErr = language.ParseAcceptLanguage(acceptLanguage)
	if parseErr != nil || len(tags) == 0 {
		return primary, primary, nil
	}
	enabledTags := make([]language.Tag, 0, len(locales))
	for _, localeValue := range locales {
		var tag language.Tag
		tag, parseErr = language.Parse(localeValue)
		if parseErr != nil {
			return "", "", parseErr
		}
		enabledTags = append(enabledTags, tag)
	}
	matched, index, confidence := language.NewMatcher(enabledTags).Match(tags...)
	if confidence == language.No {
		return primary, primary, nil
	}
	_ = matched
	return locales[index], primary, nil
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
