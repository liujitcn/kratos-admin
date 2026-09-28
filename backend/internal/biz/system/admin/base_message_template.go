package biz

import (
	"context"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
)

// BaseMessageTemplateCase 管理消息 Provider 模板。
type BaseMessageTemplateCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseMessageTemplateRepository
	categoryRepo *data.BaseMessageCategoryRepository
	providerRepo *data.BaseMessageProviderRepository
}

// NewBaseMessageTemplateCase 创建消息 Provider 模板业务实例。
func NewBaseMessageTemplateCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	templateRepo *data.BaseMessageTemplateRepository,
	categoryRepo *data.BaseMessageCategoryRepository,
	providerRepo *data.BaseMessageProviderRepository,
) *BaseMessageTemplateCase {
	return &BaseMessageTemplateCase{
		BaseCase: baseCase, tx: tx, BaseMessageTemplateRepository: templateRepo,
		categoryRepo: categoryRepo, providerRepo: providerRepo,
	}
}

// PageBaseMessageTemplate 分页查询消息 Provider 模板。
func (c *BaseMessageTemplateCase) PageBaseMessageTemplate(ctx context.Context, req *adminv1.PageBaseMessageTemplateRequest) (*adminv1.PageBaseMessageTemplateResponse, error) {
	query := c.Query(ctx).BaseMessageTemplate
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.ID.Desc()))
	if req.CategoryId != nil {
		opts = append(opts, repository.Where(query.CategoryID.Eq(req.GetCategoryId())))
	}
	if req.ProviderId != nil {
		opts = append(opts, repository.Where(query.ProviderID.Eq(req.GetProviderId())))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	categoryIDs := make([]int64, 0, len(list))
	providerIDs := make([]int64, 0, len(list))
	for _, item := range list {
		categoryIDs = append(categoryIDs, item.CategoryID)
		providerIDs = append(providerIDs, item.ProviderID)
	}
	categories, err := c.categoryRepo.ListByIDs(ctx, categoryIDs)
	if err != nil {
		return nil, err
	}
	providers, err := c.providerRepo.ListByIDs(ctx, providerIDs)
	if err != nil {
		return nil, err
	}
	categoryNames := make(map[int64]string, len(categories))
	for _, item := range categories {
		categoryNames[item.ID] = item.Name
	}
	providerNames := make(map[int64]string, len(providers))
	for _, item := range providers {
		providerNames[item.ID] = item.Name
	}
	result := make([]*adminv1.BaseMessageTemplate, 0, len(list))
	for _, item := range list {
		config, decodeErr := decodeMessageJSON(item.Config)
		if decodeErr != nil {
			return nil, errorsx.Internal("解析消息模板配置失败").WithCause(decodeErr)
		}
		result = append(result, &adminv1.BaseMessageTemplate{
			Id: item.ID, CategoryId: item.CategoryID, CategoryName: categoryNames[item.CategoryID],
			ProviderId: item.ProviderID, ProviderName: providerNames[item.ProviderID], Title: item.Title,
			Content: item.Content, ContentFormat: basev1.MessageContentFormat(item.ContentFormat), Config: config,
			CreatedAt: item.CreatedAt.Format(time.DateTime), UpdatedAt: item.UpdatedAt.Format(time.DateTime),
		})
	}
	return &adminv1.PageBaseMessageTemplateResponse{BaseMessageTemplates: result, Total: int32(total)}, nil
}

// GetBaseMessageTemplate 查询消息 Provider 模板详情。
func (c *BaseMessageTemplateCase) GetBaseMessageTemplate(ctx context.Context, id int64) (*adminv1.BaseMessageTemplateForm, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	config, err := decodeMessageJSON(item.Config)
	if err != nil {
		return nil, errorsx.Internal("解析消息模板配置失败").WithCause(err)
	}
	return &adminv1.BaseMessageTemplateForm{
		Id: item.ID, CategoryId: item.CategoryID, ProviderId: item.ProviderID, Title: item.Title,
		Content: item.Content, ContentFormat: basev1.MessageContentFormat(item.ContentFormat), Config: config,
	}, nil
}

// CreateBaseMessageTemplate 创建消息 Provider 模板。
func (c *BaseMessageTemplateCase) CreateBaseMessageTemplate(ctx context.Context, req *adminv1.BaseMessageTemplateForm) error {
	if err := c.validateBinding(ctx, req.GetCategoryId(), req.GetProviderId()); err != nil {
		return err
	}
	item, err := c.formEntity(req, nil)
	if err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	item.CreatedBy, item.UpdatedBy, item.CreatedAt, item.UpdatedAt = authInfo.UserId, authInfo.UserId, now, now
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.Create(txCtx, item); err != nil && errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("消息分类与 Provider 模板已存在", "base_message_template", "category_id,provider_id", "unique_base_message_template").WithCause(err)
		}
		return err
	})
	return err
}

// UpdateBaseMessageTemplate 更新消息 Provider 模板。
func (c *BaseMessageTemplateCase) UpdateBaseMessageTemplate(ctx context.Context, req *adminv1.BaseMessageTemplateForm) error {
	if err := c.validateBinding(ctx, req.GetCategoryId(), req.GetProviderId()); err != nil {
		return err
	}
	oldItem, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	item, err := c.formEntity(req, oldItem)
	if err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	item.ID, item.CreatedBy, item.CreatedAt = oldItem.ID, oldItem.CreatedBy, oldItem.CreatedAt
	item.UpdatedBy, item.UpdatedAt = authInfo.UserId, time.Now()
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.UpdateByID(txCtx, item); err != nil && errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("消息分类与 Provider 模板已存在", "base_message_template", "category_id,provider_id", "unique_base_message_template").WithCause(err)
		}
		return err
	})
	return err
}

// DeleteBaseMessageTemplate 删除消息 Provider 模板。
func (c *BaseMessageTemplateCase) DeleteBaseMessageTemplate(ctx context.Context, ids string) error {
	idValues := _string.ConvertStringToInt64Array(ids)
	if len(idValues) == 0 {
		return errorsx.InvalidArgument("消息模板 ID不能为空")
	}
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		return c.DeleteByIDs(txCtx, idValues)
	})
}

// validateBinding 校验模板的分类与 Provider 绑定关系。
func (c *BaseMessageTemplateCase) validateBinding(ctx context.Context, categoryID, providerID int64) error {
	category, err := c.categoryRepo.FindByID(ctx, categoryID)
	if err != nil {
		return errorsx.ResourceNotFound("消息分类不存在").WithCause(err)
	}
	providerIDs, err := decodeMessageProviderIDs(category.ProviderID)
	if err != nil {
		return err
	}
	if !hasAnyInt64(providerIDs, []int64{providerID}) {
		return errorsx.InvalidArgument("消息 Provider 未绑定到该消息分类")
	}
	_, err = c.providerRepo.FindByID(ctx, providerID)
	if err != nil {
		return errorsx.ResourceNotFound("消息 Provider 不存在").WithCause(err)
	}
	return nil
}

// formEntity 将消息模板表单转换为持久化实体。
func (c *BaseMessageTemplateCase) formEntity(req *adminv1.BaseMessageTemplateForm, _ *models.BaseMessageTemplate) (*models.BaseMessageTemplate, error) {
	config, err := encodeMessageJSON(req.GetConfig())
	if err != nil {
		return nil, errorsx.InvalidArgument("消息模板扩展配置无效").WithCause(err)
	}
	contentFormat := req.GetContentFormat()
	if contentFormat == basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_UNSPECIFIED {
		contentFormat = basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT
	}
	return &models.BaseMessageTemplate{
		CategoryID: req.GetCategoryId(), ProviderID: req.GetProviderId(), Title: req.GetTitle(), Content: req.GetContent(),
		ContentFormat: int32(contentFormat), Config: config,
	}, nil
}
