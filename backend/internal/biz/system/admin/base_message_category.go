package biz

import (
	"context"
	"encoding/json"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	adminconst "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"gorm.io/gen/field"
	"gorm.io/gorm/clause"
)

// BaseMessageCategoryCase 消息分类业务实例。
type BaseMessageCategoryCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseMessageCategoryRepository
	baseMessageRepo *data.BaseMessageRepository
	deliveryRepo    *data.BaseMessageDeliveryRepository
	providerRepo    *data.BaseMessageProviderRepository
	templateRepo    *data.BaseMessageTemplateRepository
	formMapper      *mapper.CopierMapper[adminv1.BaseMessageCategoryForm, models.BaseMessageCategory]
	mapper          *mapper.CopierMapper[adminv1.BaseMessageCategory, models.BaseMessageCategory]
}

// NewBaseMessageCategoryCase 创建消息分类业务实例。
func NewBaseMessageCategoryCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseMessageCategoryRepo *data.BaseMessageCategoryRepository,
	baseMessageRepo *data.BaseMessageRepository,
	deliveryRepo *data.BaseMessageDeliveryRepository,
	providerRepo *data.BaseMessageProviderRepository,
	templateRepo *data.BaseMessageTemplateRepository,
) *BaseMessageCategoryCase {
	return &BaseMessageCategoryCase{
		BaseCase:                      baseCase,
		tx:                            tx,
		BaseMessageCategoryRepository: baseMessageCategoryRepo,
		baseMessageRepo:               baseMessageRepo,
		deliveryRepo:                  deliveryRepo,
		providerRepo:                  providerRepo,
		templateRepo:                  templateRepo,
		formMapper:                    mapper.NewCopierMapper[adminv1.BaseMessageCategoryForm, models.BaseMessageCategory](),
		mapper:                        mapper.NewCopierMapper[adminv1.BaseMessageCategory, models.BaseMessageCategory](),
	}
}

// OptionBaseMessageCategory 查询消息分类选项。
func (c *BaseMessageCategoryCase) OptionBaseMessageCategory(ctx context.Context, req *adminv1.OptionBaseMessageCategoryRequest) (*commonv1.SelectOptionResponse, error) {
	query := c.Query(ctx).BaseMessageCategory
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc()))
	var list []*models.BaseMessageCategory
	var err error
	list, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		if req.InboxEnabled != nil && item.InboxEnabled != int32(req.GetInboxEnabled()) {
			continue
		}
		options = append(options, &commonv1.SelectOptionResponse_Option{
			Label:    item.Name,
			Value:    item.ID,
			Disabled: item.Status != _const.STATUS_STATUS_ENABLE,
		})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// PageBaseMessageCategory 分页查询消息分类。
func (c *BaseMessageCategoryCase) PageBaseMessageCategory(ctx context.Context, req *adminv1.PageBaseMessageCategoryRequest) (*adminv1.PageBaseMessageCategoryResponse, error) {
	query := c.Query(ctx).BaseMessageCategory
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.Sort.Asc()), repository.Order(query.ID.Desc()))
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var list []*models.BaseMessageCategory
	var total int64
	var err error
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseMessageCategory, 0, len(list))
	for _, item := range list {
		value := c.toDTO(item)
		result = append(result, value)
	}
	return &adminv1.PageBaseMessageCategoryResponse{BaseMessageCategories: result, Total: int32(total)}, nil
}

// GetBaseMessageCategory 查询消息分类详情。
func (c *BaseMessageCategoryCase) GetBaseMessageCategory(ctx context.Context, id int64) (*adminv1.BaseMessageCategoryForm, error) {
	entity, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.toForm(entity)
}

// CreateBaseMessageCategory 创建消息分类。
func (c *BaseMessageCategoryCase) CreateBaseMessageCategory(ctx context.Context, req *adminv1.BaseMessageCategoryForm) error {
	var err error
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	entity := c.formMapper.ToEntity(req)
	entity.ProviderID, err = encodeProviderIDs(req.GetProviderId())
	if err != nil {
		return errorsx.InvalidArgument("消息 Provider 配置无效").WithCause(err)
	}
	inboxEnabled := req.GetInboxEnabled()
	if inboxEnabled == adminv1.BaseMessageCategoryInboxEnabled_BASE_MESSAGE_CATEGORY_INBOX_ENABLED_UNSPECIFIED {
		inboxEnabled = adminv1.BaseMessageCategoryInboxEnabled_BASE_MESSAGE_CATEGORY_INBOX_ENABLED_ENABLE
	}
	if inboxEnabled != adminv1.BaseMessageCategoryInboxEnabled_BASE_MESSAGE_CATEGORY_INBOX_ENABLED_ENABLE && inboxEnabled != adminv1.BaseMessageCategoryInboxEnabled_BASE_MESSAGE_CATEGORY_INBOX_ENABLED_DISABLE {
		return errorsx.InvalidArgument("站内信状态无效")
	}
	entity.InboxEnabled = int32(inboxEnabled)
	if err = c.validateProviderIDs(ctx, req.GetProviderId()); err != nil {
		return err
	}
	if entity.DefaultPriority == 0 {
		entity.DefaultPriority = int32(basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL)
	}
	if entity.RetentionDays == 0 {
		entity.RetentionDays = defaultMessageRetentionDays
	}
	if entity.Status == 0 {
		entity.Status = _const.STATUS_STATUS_ENABLE
	}
	now := time.Now()
	entity.CreatedBy, entity.UpdatedBy, entity.CreatedAt, entity.UpdatedAt = authInfo.UserId, authInfo.UserId, now, now
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.Create(txCtx, entity)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("消息分类编码重复", "base_message_category", "code", "unique_base_message_category").WithCause(err)
			}
			return err
		}
		return c.syncTemplates(txCtx, entity.ID, req.GetProviderId(), authInfo.UserId)
	})
	if err != nil {
		return err
	}
	incrementCacheRevision(c.Cache, adminconst.NOTIFICATION_CATEGORY_CACHE_REVISION_KEY)
	return nil
}

// UpdateBaseMessageCategory 更新消息分类。
func (c *BaseMessageCategoryCase) UpdateBaseMessageCategory(ctx context.Context, req *adminv1.BaseMessageCategoryForm) error {
	var err error
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var oldEntity *models.BaseMessageCategory
	oldEntity, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if req.GetCode() != oldEntity.Code {
		return errorsx.Conflict("消息分类编码不可修改")
	}
	entity := c.formMapper.ToEntity(req)
	entity.ID = oldEntity.ID
	entity.ProviderID, err = encodeProviderIDs(req.GetProviderId())
	if err != nil {
		return errorsx.InvalidArgument("消息 Provider 配置无效").WithCause(err)
	}
	inboxEnabled := req.GetInboxEnabled()
	if inboxEnabled == adminv1.BaseMessageCategoryInboxEnabled_BASE_MESSAGE_CATEGORY_INBOX_ENABLED_UNSPECIFIED {
		inboxEnabled = adminv1.BaseMessageCategoryInboxEnabled(oldEntity.InboxEnabled)
	}
	if inboxEnabled != adminv1.BaseMessageCategoryInboxEnabled_BASE_MESSAGE_CATEGORY_INBOX_ENABLED_ENABLE && inboxEnabled != adminv1.BaseMessageCategoryInboxEnabled_BASE_MESSAGE_CATEGORY_INBOX_ENABLED_DISABLE {
		return errorsx.InvalidArgument("站内信状态无效")
	}
	entity.InboxEnabled = int32(inboxEnabled)
	routeChanged := entity.ProviderID != oldEntity.ProviderID || entity.InboxEnabled != oldEntity.InboxEnabled
	var oldProviderIDs []int64
	oldProviderIDs, err = decodeMessageProviderIDs(oldEntity.ProviderID)
	if err != nil {
		return err
	}
	selectedProviders := make(map[int64]struct{}, len(req.GetProviderId()))
	for _, providerID := range req.GetProviderId() {
		selectedProviders[providerID] = struct{}{}
	}
	removedProviderIDs := make([]int64, 0, len(oldProviderIDs))
	for _, providerID := range oldProviderIDs {
		if _, exists := selectedProviders[providerID]; !exists {
			removedProviderIDs = append(removedProviderIDs, providerID)
		}
	}
	if err = c.validateProviderIDs(ctx, req.GetProviderId()); err != nil {
		return err
	}
	if entity.Status == 0 {
		entity.Status = oldEntity.Status
	}
	entity.CreatedBy, entity.CreatedAt = oldEntity.CreatedBy, oldEntity.CreatedAt
	entity.UpdatedBy, entity.UpdatedAt = authInfo.UserId, time.Now()
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		categoryQuery := c.Query(txCtx).BaseMessageCategory
		var currentCategory *models.BaseMessageCategory
		currentCategory, err = c.Find(txCtx,
			repository.Where(categoryQuery.ID.Eq(entity.ID)),
			repository.Clauses(clause.Locking{Strength: "UPDATE"}),
		)
		if err != nil {
			return err
		}
		if currentCategory.ProviderID != oldEntity.ProviderID || currentCategory.InboxEnabled != oldEntity.InboxEnabled {
			return errorsx.Conflict("消息分类投递目标已被其他人修改，请刷新后重试")
		}
		if routeChanged {
			if err = c.validateRouteChange(txCtx, entity.ID, removedProviderIDs); err != nil {
				return err
			}
		}
		err = c.UpdateByID(txCtx, entity)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("消息分类编码重复", "base_message_category", "code", "unique_base_message_category").WithCause(err)
			}
			return err
		}
		return c.syncTemplates(txCtx, entity.ID, req.GetProviderId(), authInfo.UserId)
	})
	if err != nil {
		return err
	}
	incrementCacheRevision(c.Cache, adminconst.NOTIFICATION_CATEGORY_CACHE_REVISION_KEY)
	return nil
}

// validateRouteChange 防止在消息派发或 Provider 重试期间改变分类路由。
func (c *BaseMessageCategoryCase) validateRouteChange(ctx context.Context, categoryID int64, removedProviderIDs []int64) error {
	messageQuery := c.baseMessageRepo.Query(ctx).BaseMessage
	publishingCount, err := c.baseMessageRepo.Count(ctx,
		repository.Where(messageQuery.CategoryID.Eq(categoryID)),
		repository.Where(messageQuery.Status.Eq(int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHING))),
	)
	if err != nil {
		return err
	}
	if publishingCount > 0 {
		return errorsx.Conflict("消息正在派发或等待重试，不能修改分类投递目标")
	}
	if len(removedProviderIDs) == 0 {
		return nil
	}
	opts := []repository.QueryOption{
		repository.Where(messageQuery.CategoryID.Eq(categoryID)),
		repository.Where(field.Or(
			messageQuery.Status.Eq(int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHED)),
			messageQuery.Status.Eq(int32(basev1.MessageStatus_MESSAGE_STATUS_REVOKED)),
		)),
	}
	for pageNum := int64(1); ; pageNum++ {
		var messages []*models.BaseMessage
		var total int64
		messages, total, err = c.baseMessageRepo.Page(ctx, pageNum, 500, opts...)
		if err != nil {
			return err
		}
		if len(messages) == 0 {
			return nil
		}
		messageIDs := make([]int64, 0, len(messages))
		for _, message := range messages {
			messageIDs = append(messageIDs, message.ID)
		}
		deliveryQuery := c.deliveryRepo.Query(ctx).BaseMessageDelivery
		pendingOpts := []repository.QueryOption{
			repository.Where(deliveryQuery.MessageID.In(messageIDs...)),
			repository.Where(deliveryQuery.DeliveryType.Eq(adminconst.MessageDeliveryTypeProvider)),
			repository.Where(deliveryQuery.ProviderID.In(removedProviderIDs...)),
			repository.Where(field.Or(
				deliveryQuery.Status.Eq(adminconst.MessageDeliveryStatusPending),
				deliveryQuery.Status.Eq(adminconst.MessageDeliveryStatusRunning),
			)),
		}
		var pendingCount int64
		pendingCount, err = c.deliveryRepo.Count(ctx, pendingOpts...)
		if err != nil {
			return err
		}
		if pendingCount > 0 {
			return errorsx.Conflict("分类存在尚未完成的 Provider 投递，不能移除该 Provider")
		}
		if pageNum*500 >= total {
			return nil
		}
	}
}

// DeleteBaseMessageCategory 删除未被消息引用的分类。
func (c *BaseMessageCategoryCase) DeleteBaseMessageCategory(ctx context.Context, id string) error {
	var err error
	ids := _string.ConvertStringToInt64Array(id)
	var list []*models.BaseMessageCategory
	list, err = c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(list) != len(ids) {
		return errorsx.ResourceNotFound("删除消息分类失败，分类不存在")
	}
	query := c.baseMessageRepo.Query(ctx).BaseMessage
	var count int64
	count, err = c.baseMessageRepo.Count(ctx, repository.Where(query.CategoryID.In(ids...)))
	if err != nil {
		return err
	}
	if count > 0 {
		return errorsx.HasChildrenConflict("删除消息分类失败，仍有消息使用该分类", "base_message_category", "base_message")
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.DeleteByIDs(txCtx, ids)
		if err != nil {
			return err
		}
		templateQuery := c.templateRepo.Query(txCtx).BaseMessageTemplate
		return c.templateRepo.Delete(txCtx, repository.Where(templateQuery.CategoryID.In(ids...)))
	})
	if err != nil {
		return err
	}
	incrementCacheRevision(c.Cache, adminconst.NOTIFICATION_CATEGORY_CACHE_REVISION_KEY)
	return nil
}

// SetBaseMessageCategoryStatus 设置消息分类状态。
func (c *BaseMessageCategoryCase) SetBaseMessageCategoryStatus(ctx context.Context, req *adminv1.SetBaseMessageCategoryStatusRequest) error {
	var err error
	var entity *models.BaseMessageCategory
	entity, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if int32(req.GetStatus()) != _const.STATUS_STATUS_ENABLE && int32(req.GetStatus()) != _const.STATUS_STATUS_DISABLE {
		return errorsx.InvalidArgument("消息分类状态无效")
	}
	if entity.Status == int32(req.GetStatus()) {
		return nil
	}
	err = c.UpdateByID(ctx, &models.BaseMessageCategory{ID: entity.ID, Status: int32(req.GetStatus())})
	if err != nil {
		return err
	}
	incrementCacheRevision(c.Cache, adminconst.NOTIFICATION_CATEGORY_CACHE_REVISION_KEY)
	return nil
}

// ListInboxCategoryIDs 查询启用站内信的分类 ID。
func (c *BaseMessageCategoryCase) ListInboxCategoryIDs(ctx context.Context) ([]int64, error) {
	query := c.Query(ctx).BaseMessageCategory
	list, err := c.List(ctx, repository.Where(query.InboxEnabled.Eq(int32(adminv1.BaseMessageCategoryInboxEnabled_BASE_MESSAGE_CATEGORY_INBOX_ENABLED_ENABLE))))
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ID)
	}
	return ids, nil
}

// validateProviderIDs 校验消息分类绑定的 Provider 是否存在且不重复。
func (c *BaseMessageCategoryCase) validateProviderIDs(ctx context.Context, ids []int64) error {
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return errorsx.InvalidArgument("消息 Provider ID 无效")
		}
		if _, exists := seen[id]; exists {
			return errorsx.InvalidArgument("消息 Provider 不能重复")
		}
		seen[id] = struct{}{}
	}
	if len(ids) == 0 {
		return nil
	}
	providers, err := c.providerRepo.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(providers) != len(ids) {
		return errorsx.ResourceNotFound("消息 Provider 不存在")
	}
	return nil
}

// syncTemplates 按分类当前绑定的 Provider 自动补齐并清理模板记录。
func (c *BaseMessageCategoryCase) syncTemplates(ctx context.Context, categoryID int64, providerIDs []int64, userID int64) error {
	query := c.templateRepo.Query(ctx).BaseMessageTemplate
	existing, err := c.templateRepo.List(ctx, repository.Where(query.CategoryID.Eq(categoryID)))
	if err != nil {
		return err
	}
	selected := make(map[int64]struct{}, len(providerIDs))
	for _, providerID := range providerIDs {
		selected[providerID] = struct{}{}
	}
	now := time.Now()
	for _, item := range existing {
		if _, ok := selected[item.ProviderID]; ok {
			delete(selected, item.ProviderID)
			continue
		}
		if err = c.templateRepo.DeleteByIDs(ctx, []int64{item.ID}); err != nil {
			return err
		}
	}
	for providerID := range selected {
		if err = c.templateRepo.Create(ctx, &models.BaseMessageTemplate{
			CategoryID: categoryID, ProviderID: providerID, ContentFormat: int32(basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT),
			Config: "{}", CreatedBy: userID, UpdatedBy: userID, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return err
		}
	}
	return nil
}

// toDTO 将消息分类实体转换为列表响应。
func (c *BaseMessageCategoryCase) toDTO(entity *models.BaseMessageCategory) *adminv1.BaseMessageCategory {
	result := c.mapper.ToDTO(entity)
	providerIDs, err := decodeMessageProviderIDs(entity.ProviderID)
	if err == nil {
		result.ProviderId = providerIDs
	}
	result.InboxEnabled = adminv1.BaseMessageCategoryInboxEnabled(entity.InboxEnabled)
	return result
}

// toForm 将消息分类实体转换为编辑表单。
func (c *BaseMessageCategoryCase) toForm(entity *models.BaseMessageCategory) (*adminv1.BaseMessageCategoryForm, error) {
	result := c.formMapper.ToDTO(entity)
	providerIDs, err := decodeMessageProviderIDs(entity.ProviderID)
	if err != nil {
		return nil, err
	}
	result.ProviderId, result.InboxEnabled = providerIDs, adminv1.BaseMessageCategoryInboxEnabled(entity.InboxEnabled)
	return result, nil
}

// encodeProviderIDs 编码消息分类绑定的 Provider ID JSON数组。
func encodeProviderIDs(ids []int64) (string, error) {
	if ids == nil {
		ids = []int64{}
	}
	value, err := json.Marshal(ids)
	if err != nil {
		return "", err
	}
	return string(value), nil
}
