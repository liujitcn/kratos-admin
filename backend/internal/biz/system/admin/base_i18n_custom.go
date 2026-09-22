package biz

import (
	"context"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// BaseI18nCustomCase 管理前端国际化自定义翻译信息。
type BaseI18nCustomCase struct {
	*biz.BaseCase
	*data.BaseI18NCustomRepository
	tx               data.Transaction
	baseLanguageCase *BaseLanguageCase
	formMapper       *mapper.CopierMapper[adminv1.BaseI18nCustomForm, models.BaseI18NCustom]
	mapper           *mapper.CopierMapper[adminv1.BaseI18nCustom, models.BaseI18NCustom]
}

// NewBaseI18nCustomCase 创建国际化自定义翻译业务实例。
func NewBaseI18nCustomCase(baseCase *biz.BaseCase, tx data.Transaction, repo *data.BaseI18NCustomRepository, baseLanguageCase *BaseLanguageCase) *BaseI18nCustomCase {
	return &BaseI18nCustomCase{
		BaseCase:                 baseCase,
		BaseI18NCustomRepository: repo,
		tx:                       tx,
		baseLanguageCase:         baseLanguageCase,
		formMapper:               mapper.NewCopierMapper[adminv1.BaseI18nCustomForm, models.BaseI18NCustom](),
		mapper:                   mapper.NewCopierMapper[adminv1.BaseI18nCustom, models.BaseI18NCustom](),
	}
}

// PageBaseI18nCustom 分页查询国际化自定义翻译信息。
func (c *BaseI18nCustomCase) PageBaseI18nCustom(ctx context.Context, req *adminv1.PageBaseI18nCustomRequest) (*adminv1.PageBaseI18nCustomResponse, error) {
	tenantID, err := c.queryTenantID(ctx, req.GetTenantId())
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseI18NCustom
	opts := make([]repository.QueryOption, 0, 7)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()), repository.Order(query.ID.Desc()))
	if tenantID > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(tenantID)))
	}
	if req.Site != nil {
		opts = append(opts, repository.Where(query.Site.Eq(int32(req.GetSite()))))
	}
	if req.GetKey() != "" {
		opts = append(opts, repository.Where(query.Key.Like("%"+req.GetKey()+"%")))
	}
	if req.GetLocale() != "" {
		opts = append(opts, repository.Where(query.Locale.Eq(req.GetLocale())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var list []*models.BaseI18NCustom
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BaseI18nCustom, 0, len(list))
	for _, item := range list {
		items = append(items, c.mapper.ToDTO(item))
	}
	return &adminv1.PageBaseI18nCustomResponse{Items: items, Total: int32(total)}, nil
}

// GetBaseI18nCustom 查询国际化自定义翻译详情。
func (c *BaseI18nCustomCase) GetBaseI18nCustom(ctx context.Context, id int64) (*adminv1.BaseI18nCustomForm, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = c.validateTenantAccess(ctx, item.TenantID); err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(item), nil
}

// CreateBaseI18nCustom 创建国际化自定义翻译。
func (c *BaseI18nCustomCase) CreateBaseI18nCustom(ctx context.Context, req *adminv1.BaseI18nCustomForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if err = c.validateTenantAccess(ctx, req.GetTenantId()); err != nil {
		return err
	}
	if err = c.validateLocale(ctx, req.GetLocale()); err != nil {
		return err
	}
	item := c.formMapper.ToEntity(req)
	if item.Status == 0 {
		item.Status = _const.STATUS_STATUS_ENABLE
	}
	now := time.Now()
	item.CreatedBy = authInfo.UserId
	item.UpdatedBy = authInfo.UserId
	item.CreatedAt = now
	item.UpdatedAt = now
	err = c.Create(ctx, item)
	if err != nil {
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("同一位置、语言和语言键的自定义翻译重复", "base_i18n_custom", "", "unique_base_i18n_custom").WithCause(err)
		}
		return err
	}
	return nil
}

// UpdateBaseI18nCustom 更新国际化自定义翻译。
func (c *BaseI18nCustomCase) UpdateBaseI18nCustom(ctx context.Context, req *adminv1.BaseI18nCustomForm) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	current, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if err = c.validateTenantAccess(ctx, current.TenantID); err != nil {
		return err
	}
	if err = c.validateLocale(ctx, req.GetLocale()); err != nil {
		return err
	}
	item := c.formMapper.ToEntity(req)
	item.ID = current.ID
	item.TenantID = current.TenantID
	item.Key = current.Key
	item.Locale = current.Locale
	if item.Status == 0 {
		item.Status = current.Status
	}
	item.Site = current.Site
	item.UpdatedBy = authInfo.UserId
	item.UpdatedAt = time.Now()
	err = c.UpdateByID(ctx, item)
	if err != nil {
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("同一位置、语言和语言键的自定义翻译重复", "base_i18n_custom", "", "unique_base_i18n_custom").WithCause(err)
		}
		return err
	}
	return nil
}

// DeleteBaseI18nCustom 删除国际化自定义翻译。
func (c *BaseI18nCustomCase) DeleteBaseI18nCustom(ctx context.Context, ids string) error {
	idList := _string.ConvertStringToInt64Array(ids)
	if len(idList) == 0 {
		return nil
	}
	query := c.Query(ctx).BaseI18NCustom
	items, err := c.List(ctx, repository.Where(query.ID.In(idList...)))
	if err != nil {
		return err
	}
	for _, item := range items {
		if err = c.validateTenantAccess(ctx, item.TenantID); err != nil {
			return err
		}
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		return c.DeleteByIDs(ctx, idList)
	})
}

// SetBaseI18nCustomStatus 设置国际化自定义翻译状态。
func (c *BaseI18nCustomCase) SetBaseI18nCustomStatus(ctx context.Context, req *adminv1.SetBaseI18nCustomStatusRequest) error {
	if int32(req.GetStatus()) != _const.STATUS_STATUS_ENABLE && int32(req.GetStatus()) != _const.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("国际化自定义翻译状态无效")
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var item *models.BaseI18NCustom
	item, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if err = c.validateTenantAccess(ctx, item.TenantID); err != nil {
		return err
	}
	item.Status = int32(req.GetStatus())
	item.UpdatedBy = authInfo.UserId
	item.UpdatedAt = time.Now()
	return c.UpdateByID(ctx, item)
}

// queryTenantID 返回当前身份允许查询的租户编号，零表示平台管理员查询全部租户。
func (c *BaseI18nCustomCase) queryTenantID(ctx context.Context, requestedTenantID int64) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	if authInfo.TenantCode != gorm.DefaultTenantCode {
		return authInfo.TenantId, nil
	}
	return requestedTenantID, nil
}

// validateTenantAccess 校验当前身份能否管理目标租户的自定义翻译。
func (c *BaseI18nCustomCase) validateTenantAccess(ctx context.Context, tenantID int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if authInfo.TenantCode != gorm.DefaultTenantCode && tenantID != authInfo.TenantId {
		return errorsx.PermissionDenied("无权管理其他租户的自定义翻译")
	}
	return nil
}

// validateLocale 校验自定义翻译语言是否为当前启用语言。
func (c *BaseI18nCustomCase) validateLocale(ctx context.Context, locale string) error {
	state, err := c.baseLanguageCase.LocaleState(ctx)
	if err != nil {
		return err
	}
	for _, enabledLocale := range state.Enabled {
		if enabledLocale == locale {
			return nil
		}
	}
	return errorsx.InvalidArgument("只能配置已启用的语言区域")
}
