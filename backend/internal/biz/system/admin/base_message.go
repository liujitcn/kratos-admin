package biz

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/log"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	admindata "github.com/liujitcn/kratos-admin/backend/internal/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/notification"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-core/queue"
	"github.com/liujitcn/kratos-core/sse"

	"github.com/liujitcn/go-utils/id"
	_string "github.com/liujitcn/go-utils/string"
	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/database/gorm"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/microcosm-cc/bluemonday"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm/clause"
)

const messageDispatchStream = "base.message.dispatch"
const messageNotificationStream = "base.notification"
const messageNotificationEvent = "inbox.changed"
const messageDispatchBatchSize = 500
const messageDispatchLease = 5 * time.Minute
const messageDeliveryCleanupBatchSize = 1000
const messageDispatchMaxAttempts = 5
const messageDispatchRetryBase = time.Minute
const messageDispatchRetryMaxDelay = 15 * time.Minute
const defaultMessageRetentionDays = 180
const maxMessageActionParamsBytes = 16 << 10
const maxMessageActionDepth = 8
const maxMessageActionKeyRunes = 64
const maxMessageActionStringRunes = 512
const defaultMessageActionParams = "{}"

var messageRichTextPolicy = bluemonday.UGCPolicy()
var messageMarkdownPolicy = bluemonday.StrictPolicy()

// BaseMessageCase 站内信管理和异步投递业务实例。
type BaseMessageCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseMessageRepository
	dispatchRepo   *data.BaseMessageDispatchRepository
	deliveryRepo   *data.BaseMessageDeliveryRepository
	deliveryWriter *admindata.MessageDeliveryWriter
	categoryCase   *BaseMessageCategoryCase
	baseUserRepo   *data.BaseUserRepository
	baseRoleRepo   *data.BaseRoleRepository
	baseDeptRepo   *data.BaseDeptRepository
	basePostRepo   *data.BasePostRepository
	baseMenuRepo   *data.BaseMenuRepository
	sse            *sse.SSE
}

var _ notification.Publisher = (*BaseMessageCase)(nil)

// NewBaseMessageCase 创建站内信业务实例。
func NewBaseMessageCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseMessageRepo *data.BaseMessageRepository,
	dispatchRepo *data.BaseMessageDispatchRepository,
	deliveryRepo *data.BaseMessageDeliveryRepository,
	deliveryWriter *admindata.MessageDeliveryWriter,
	categoryCase *BaseMessageCategoryCase,
	baseUserRepo *data.BaseUserRepository,
	baseRoleRepo *data.BaseRoleRepository,
	baseDeptRepo *data.BaseDeptRepository,
	basePostRepo *data.BasePostRepository,
	baseMenuRepo *data.BaseMenuRepository,
	sseRuntime *sse.SSE,
) *BaseMessageCase {
	caseValue := &BaseMessageCase{
		BaseCase:              baseCase,
		tx:                    tx,
		BaseMessageRepository: baseMessageRepo,
		dispatchRepo:          dispatchRepo,
		deliveryRepo:          deliveryRepo,
		deliveryWriter:        deliveryWriter,
		categoryCase:          categoryCase,
		baseUserRepo:          baseUserRepo,
		baseRoleRepo:          baseRoleRepo,
		baseDeptRepo:          baseDeptRepo,
		basePostRepo:          basePostRepo,
		baseMenuRepo:          baseMenuRepo,
		sse:                   sseRuntime,
	}
	notification.SetDefaultPublisher(caseValue)
	return caseValue
}

// resolveWriteTenant 解析消息写入租户。
func (c *BaseMessageCase) resolveWriteTenant(ctx context.Context, tenantID int64) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	if tenantID == 0 {
		return authInfo.TenantId, nil
	}
	if authInfo.TenantCode != gorm.DefaultTenantCode && tenantID != authInfo.TenantId {
		return 0, errorsx.PermissionDenied("不能操作其他租户的消息")
	}
	return tenantID, nil
}

// ensureTenantAccess 校验消息租户访问范围。
func (c *BaseMessageCase) ensureTenantAccess(ctx context.Context, tenantID int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if authInfo.TenantCode != gorm.DefaultTenantCode && authInfo.TenantId != tenantID {
		return errorsx.PermissionDenied("不能操作其他租户的消息")
	}
	return nil
}

// PageBaseMessage 分页查询消息。
func (c *BaseMessageCase) PageBaseMessage(ctx context.Context, req *adminv1.PageBaseMessageRequest) (*adminv1.PageBaseMessageResponse, error) {
	query := c.Query(ctx).BaseMessage
	opts := make([]repository.QueryOption, 0, 7)
	opts = append(opts, repository.Order(query.ID.Desc()))
	if req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	if req.CategoryId != nil {
		opts = append(opts, repository.Where(query.CategoryID.Eq(req.GetCategoryId())))
	}
	if req.GetTitle() != "" {
		opts = append(opts, repository.Where(query.Title.Like("%"+req.GetTitle()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var list []*models.BaseMessage
	var total int64
	var err error
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	categoryIDs := make([]int64, 0, len(list))
	for _, item := range list {
		categoryIDs = append(categoryIDs, item.CategoryID)
	}
	var categories []*models.BaseMessageCategory
	categories, err = c.categoryCase.ListByIDs(ctx, categoryIDs)
	if err != nil {
		return nil, err
	}
	categoryMap := make(map[int64]string, len(categories))
	for _, item := range categories {
		categoryMap[item.ID] = item.Name
	}
	result := make([]*adminv1.BaseMessage, 0, len(list))
	for _, item := range list {
		result = append(result, c.toBaseMessage(item, categoryMap[item.CategoryID]))
	}
	return &adminv1.PageBaseMessageResponse{BaseMessages: result, Total: int32(total)}, nil
}

// GetBaseMessage 查询消息详情。
func (c *BaseMessageCase) GetBaseMessage(ctx context.Context, id int64) (*adminv1.BaseMessageDetail, error) {
	entity, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err = c.ensureTenantAccess(ctx, entity.TenantID); err != nil {
		return nil, err
	}
	var category *models.BaseMessageCategory
	category, err = c.categoryCase.FindByID(ctx, entity.CategoryID)
	if err != nil {
		return nil, err
	}
	dispatchQuery := c.dispatchRepo.Query(ctx).BaseMessageDispatch
	var dispatches []*models.BaseMessageDispatch
	dispatches, err = c.dispatchRepo.List(ctx, repository.Where(dispatchQuery.MessageID.Eq(entity.ID)), repository.Order(dispatchQuery.ID.Asc()))
	if err != nil {
		return nil, err
	}
	form := c.toBaseMessageForm(entity, dispatches)
	dispatchValues := make([]*adminv1.BaseMessageDispatch, 0, len(dispatches))
	for _, item := range dispatches {
		dispatchValues = append(dispatchValues, c.toBaseMessageDispatch(item))
	}
	return &adminv1.BaseMessageDetail{
		BaseMessage: c.toBaseMessage(entity, category.Name),
		Form:        form,
		Dispatches:  dispatchValues,
	}, nil
}

// CreateBaseMessage 创建消息草稿。
func (c *BaseMessageCase) CreateBaseMessage(ctx context.Context, req *adminv1.BaseMessageForm) (int64, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return 0, err
	}
	var tenantID int64
	tenantID, err = c.resolveWriteTenant(ctx, req.GetTenantId())
	if err != nil {
		return 0, err
	}
	if err = c.validateMessageForm(ctx, tenantID, req); err != nil {
		return 0, err
	}
	var entity *models.BaseMessage
	entity, err = c.newBaseMessage(ctx, req, tenantID, authInfo.UserId, authInfo.UserName)
	if err != nil {
		return 0, err
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.Create(txCtx, entity)
		if err != nil {
			return err
		}
		return c.replaceDispatches(txCtx, entity, req.GetAudiences(), basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_WAITING)
	})
	if err != nil {
		return 0, err
	}
	return entity.ID, nil
}

// Publish 发布业务站内信并返回消息ID，数据库提交成功后由恢复任务保证最终投递。
func (c *BaseMessageCase) Publish(ctx context.Context, request notification.Message) (int64, error) {
	var err error
	if request.TenantID <= 0 {
		return 0, errorsx.InvalidArgument("消息租户不能为空")
	}
	request.ActionParams = normalizeMessageActionParams(request.ActionParams)
	if err = validatePublishedMessage(request); err != nil {
		return 0, err
	}
	var category *models.BaseMessageCategory
	categoryQuery := c.categoryCase.Query(ctx).BaseMessageCategory
	category, err = c.categoryCase.Find(ctx,
		repository.Where(categoryQuery.Code.Eq(request.CategoryCode)),
		repository.Where(categoryQuery.Status.Eq(coreconst.STATUS_STATUS_ENABLE)),
	)
	if err != nil {
		return 0, errorsx.ResourceNotFound("消息分类不存在").WithCause(err)
	}
	if request.Source == "" {
		request.Source = "business"
	}
	if request.ContentFormat == basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_UNSPECIFIED {
		request.ContentFormat = basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT
	}
	if request.Priority == basev1.MessagePriority_MESSAGE_PRIORITY_UNSPECIFIED {
		request.Priority = basev1.MessagePriority(category.DefaultPriority)
		if request.Priority == basev1.MessagePriority_MESSAGE_PRIORITY_UNSPECIFIED {
			request.Priority = basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL
		}
	}
	if request.SenderName == "" {
		request.SenderName = "系统"
	}
	if request.IdempotencyKey == "" {
		request.IdempotencyKey = id.NewGUIDv4NoHyphen()
	}
	audiences := make([]*adminv1.BaseMessageAudienceForm, 0, len(request.Audiences))
	for _, audience := range request.Audiences {
		audiences = append(audiences, &adminv1.BaseMessageAudienceForm{Type: audience.Type, Id: audience.ID, IncludeChildren: audience.IncludeChildren})
	}
	form := &adminv1.BaseMessageForm{
		TenantId: request.TenantID, CategoryId: category.ID, Title: request.Title, Content: request.Content,
		ContentFormat: request.ContentFormat, Priority: request.Priority, ActionType: request.ActionType,
		ActionTarget: request.ActionTarget, ActionParams: request.ActionParams, Audiences: audiences,
	}
	if err = c.validateMessageForm(ctx, request.TenantID, form); err != nil {
		return 0, err
	}
	var content string
	content, err = sanitizeMessageContent(request.Content, request.ContentFormat)
	if err != nil {
		return 0, err
	}
	request.Content = content
	messageQuery := c.Query(ctx).BaseMessage
	inTransaction := c.BaseMessageRepository != nil && c.Query(ctx) != c.Query(context.Background())
	var existing []*models.BaseMessage
	existing, err = c.List(ctx,
		repository.Where(messageQuery.Source.Eq(request.Source)),
		repository.Where(messageQuery.IdempotencyKey.Eq(request.IdempotencyKey)),
	)
	if err != nil {
		return 0, err
	}
	if len(existing) > 0 && request.ExpiresAt == 0 {
		request.ExpiresAt = existing[0].ExpiresAt
	}
	if len(existing) == 0 && request.ExpiresAt == 0 {
		request.ExpiresAt = time.Now().AddDate(0, 0, int(messageRetentionDays(category))).UnixMilli()
	}
	var payload []byte
	payload, err = json.Marshal(request)
	if err != nil {
		return 0, err
	}
	hash := sha256.Sum256(payload)
	payloadHash := hex.EncodeToString(hash[:])
	if len(existing) > 0 {
		if existing[0].PayloadHash != payloadHash {
			return 0, errorsx.Conflict("幂等键对应的消息内容不一致")
		}
		if !inTransaction {
			err = c.enqueuePendingDispatches(ctx, existing[0].ID, existing[0].TenantID)
		}
		if err != nil && !inTransaction {
			log.Error("唤醒站内信投递任务失败", "message_id", existing[0].ID, "error", err)
		}
		return existing[0].ID, nil
	}
	now := time.Now()
	entity := &models.BaseMessage{
		TenantID: request.TenantID, CategoryID: category.ID, SourceType: int32(basev1.MessageSourceType_MESSAGE_SOURCE_TYPE_BUSINESS), Source: request.Source,
		IdempotencyKey: request.IdempotencyKey, SenderType: int32(basev1.MessageSenderType_MESSAGE_SENDER_TYPE_SYSTEM), SenderName: request.SenderName,
		Priority: int32(request.Priority), Title: request.Title, Content: content, ContentFormat: int32(request.ContentFormat),
		ActionType: int32(request.ActionType), ActionTarget: request.ActionTarget, ActionParams: request.ActionParams,
		Status: int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHING), ExpiresAt: request.ExpiresAt,
		Version: 1, CreatedAt: now, UpdatedAt: now,
	}
	entity.PayloadHash = payloadHash
	persist := func(txCtx context.Context) error {
		if err = c.Create(txCtx, entity); err != nil {
			return err
		}
		return c.replaceDispatches(txCtx, entity, audiences, basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING)
	}
	if inTransaction {
		err = persist(ctx)
	} else {
		err = c.tx.Transaction(ctx, persist)
	}
	if err != nil {
		if errorsx.IsDuplicateKey(err) {
			var concurrentExisting []*models.BaseMessage
			concurrentExisting, err = c.List(ctx,
				repository.Where(messageQuery.Source.Eq(request.Source)),
				repository.Where(messageQuery.IdempotencyKey.Eq(request.IdempotencyKey)),
			)
			if err != nil {
				return 0, err
			}
			if len(concurrentExisting) > 0 && concurrentExisting[0].PayloadHash == payloadHash {
				if !inTransaction {
					err = c.enqueuePendingDispatches(ctx, concurrentExisting[0].ID, concurrentExisting[0].TenantID)
				}
				if err != nil && !inTransaction {
					log.Error("唤醒站内信投递任务失败", "message_id", concurrentExisting[0].ID, "error", err)
				}
				return concurrentExisting[0].ID, nil
			}
		}
		return 0, err
	}
	if !inTransaction {
		err = c.enqueuePendingDispatches(ctx, entity.ID, entity.TenantID)
	}
	if err != nil && !inTransaction {
		log.Error("唤醒站内信投递任务失败", "message_id", entity.ID, "error", err)
	}
	return entity.ID, nil
}

// UpdateBaseMessage 更新消息草稿。
func (c *BaseMessageCase) UpdateBaseMessage(ctx context.Context, req *adminv1.BaseMessageForm) error {
	if req.GetVersion() <= 0 {
		return errorsx.InvalidArgument("消息版本无效")
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var oldEntity *models.BaseMessage
	oldEntity, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if err = c.ensureTenantAccess(ctx, oldEntity.TenantID); err != nil {
		return err
	}
	if oldEntity.Status != int32(basev1.MessageStatus_MESSAGE_STATUS_DRAFT) {
		return errorsx.Conflict("只有草稿消息可以编辑")
	}
	if err = c.validateMessageForm(ctx, oldEntity.TenantID, req); err != nil {
		return err
	}
	var entity *models.BaseMessage
	entity, err = c.newBaseMessage(ctx, req, oldEntity.TenantID, oldEntity.SenderID, oldEntity.SenderName)
	if err != nil {
		return err
	}
	entity.ID = oldEntity.ID
	entity.Source = oldEntity.Source
	entity.SourceType = oldEntity.SourceType
	entity.IdempotencyKey = oldEntity.IdempotencyKey
	entity.CreatedBy = oldEntity.CreatedBy
	entity.CreatedAt = oldEntity.CreatedAt
	entity.UpdatedBy = authInfo.UserId
	entity.Version = oldEntity.Version + 1
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		query := c.Query(txCtx).BaseMessage
		var result gen.ResultInfo
		result, err = query.WithContext(txCtx).
			Where(query.ID.Eq(oldEntity.ID), query.Version.Eq(req.GetVersion())).
			UpdateSimple(
				query.CategoryID.Value(entity.CategoryID),
				query.Title.Value(entity.Title),
				query.Content.Value(entity.Content),
				query.ContentFormat.Value(entity.ContentFormat),
				query.Priority.Value(entity.Priority),
				query.ActionType.Value(entity.ActionType),
				query.ActionTarget.Value(entity.ActionTarget),
				query.ActionParams.Value(entity.ActionParams),
				query.ScheduledAt.Value(entity.ScheduledAt),
				query.ExpiresAt.Value(entity.ExpiresAt),
				query.Version.Value(entity.Version),
				query.UpdatedBy.Value(entity.UpdatedBy),
				query.UpdatedAt.Value(entity.UpdatedAt),
			)
		if err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return errorsx.Conflict("消息已被其他人修改，请刷新后重试")
		}
		return c.replaceDispatches(txCtx, entity, req.GetAudiences(), basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_WAITING)
	})
}

// DeleteBaseMessage 删除消息草稿。
func (c *BaseMessageCase) DeleteBaseMessage(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	list, err := c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(list) != len(ids) {
		return errorsx.ResourceNotFound("删除消息失败，消息不存在")
	}
	for _, item := range list {
		if err = c.ensureTenantAccess(ctx, item.TenantID); err != nil {
			return err
		}
		if item.Status != int32(basev1.MessageStatus_MESSAGE_STATUS_DRAFT) {
			return errorsx.Conflict("只有草稿消息可以删除")
		}
	}
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		dispatchQuery := c.dispatchRepo.Query(txCtx).BaseMessageDispatch
		err = c.dispatchRepo.Delete(txCtx, repository.Where(dispatchQuery.MessageID.In(ids...)))
		if err != nil {
			return err
		}
		return c.DeleteByIDs(txCtx, ids)
	})
}

// PublishBaseMessage 发布或安排定时发布消息。
func (c *BaseMessageCase) PublishBaseMessage(ctx context.Context, id int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var entity *models.BaseMessage
	entity, err = c.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err = c.ensureTenantAccess(ctx, entity.TenantID); err != nil {
		return err
	}
	if entity.Status != int32(basev1.MessageStatus_MESSAGE_STATUS_DRAFT) {
		return errorsx.Conflict("只有草稿消息可以发布")
	}
	dispatchQuery := c.dispatchRepo.Query(ctx).BaseMessageDispatch
	var dispatches []*models.BaseMessageDispatch
	dispatches, err = c.dispatchRepo.List(ctx, repository.Where(dispatchQuery.MessageID.Eq(entity.ID)))
	if err != nil {
		return err
	}
	if len(dispatches) == 0 {
		return errorsx.InvalidArgument("消息受众不能为空")
	}
	var category *models.BaseMessageCategory
	category, err = c.categoryCase.FindByID(ctx, entity.CategoryID)
	if err != nil {
		return errorsx.ResourceNotFound("消息分类不存在").WithCause(err)
	}
	if category.Status != coreconst.STATUS_STATUS_ENABLE {
		return errorsx.InvalidArgument("消息分类不可用")
	}
	now := time.Now()
	if entity.ExpiresAt > 0 && entity.ExpiresAt <= now.UnixMilli() {
		return errorsx.InvalidArgument("消息过期时间必须晚于当前时间")
	}
	if entity.ExpiresAt == 0 {
		retentionStart := now
		if entity.ScheduledAt > now.UnixMilli() {
			retentionStart = time.UnixMilli(entity.ScheduledAt)
		}
		entity.ExpiresAt = retentionStart.AddDate(0, 0, int(messageRetentionDays(category))).UnixMilli()
	}
	status := basev1.MessageStatus_MESSAGE_STATUS_PUBLISHING
	dispatchStatus := basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING
	if entity.ScheduledAt > now.UnixMilli() {
		status = basev1.MessageStatus_MESSAGE_STATUS_SCHEDULED
		dispatchStatus = basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_WAITING
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		messageQuery := c.Query(txCtx).BaseMessage
		var result gen.ResultInfo
		result, err = messageQuery.WithContext(txCtx).
			Where(messageQuery.ID.Eq(entity.ID), messageQuery.Version.Eq(entity.Version)).
			UpdateSimple(
				messageQuery.Status.Value(int32(status)),
				messageQuery.ExpiresAt.Value(entity.ExpiresAt),
				messageQuery.Version.Value(entity.Version+1),
				messageQuery.UpdatedBy.Value(authInfo.UserId),
				messageQuery.UpdatedAt.Value(now),
			)
		if err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return errorsx.Conflict("消息状态已变化，请刷新后重试")
		}
		entity.Status = int32(status)
		entity.Version++
		entity.UpdatedAt = now
		entity.UpdatedBy = authInfo.UserId
		dispatchQuery := c.dispatchRepo.Query(txCtx).BaseMessageDispatch
		for _, dispatch := range dispatches {
			oldVersion := dispatch.Version
			dispatch.Status = int32(dispatchStatus)
			dispatch.QueuedAt = now.UnixMilli()
			dispatch.UpdatedAt = now
			dispatch.Version++
			result, err = dispatchQuery.WithContext(txCtx).
				Where(dispatchQuery.ID.Eq(dispatch.ID), dispatchQuery.Version.Eq(oldVersion)).
				UpdateSimple(
					dispatchQuery.Status.Value(dispatch.Status),
					dispatchQuery.QueuedAt.Value(dispatch.QueuedAt),
					dispatchQuery.Version.Value(dispatch.Version),
					dispatchQuery.UpdatedAt.Value(dispatch.UpdatedAt),
				)
			if err != nil {
				return err
			}
			if result.RowsAffected == 0 {
				return errorsx.Conflict("消息投递状态已变化，请刷新后重试")
			}
		}
		return nil
	})
	if err != nil || status == basev1.MessageStatus_MESSAGE_STATUS_SCHEDULED {
		return err
	}
	for _, dispatch := range dispatches {
		if err = c.EnqueueDispatch(ctx, dispatch.ID, dispatch.TenantID); err != nil {
			log.Error("唤醒站内信投递任务失败", "dispatch_id", dispatch.ID, "error", err)
		}
	}
	return nil
}

// CancelBaseMessageSchedule 取消消息定时发布并恢复为草稿。
func (c *BaseMessageCase) CancelBaseMessageSchedule(ctx context.Context, id int64) error {
	entity, err := c.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err = c.ensureTenantAccess(ctx, entity.TenantID); err != nil {
		return err
	}
	if entity.Status != int32(basev1.MessageStatus_MESSAGE_STATUS_SCHEDULED) {
		return errorsx.Conflict("消息不在定时发布状态")
	}
	query := c.Query(ctx).BaseMessage
	var result gen.ResultInfo
	result, err = query.WithContext(ctx).
		Where(query.ID.Eq(entity.ID), query.Version.Eq(entity.Version), query.Status.Eq(entity.Status)).
		UpdateSimple(
			query.Status.Value(int32(basev1.MessageStatus_MESSAGE_STATUS_DRAFT)),
			query.Version.Value(entity.Version+1),
			query.UpdatedAt.Value(time.Now()),
		)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return errorsx.Conflict("消息状态已变化，请刷新后重试")
	}
	return nil
}

// RevokeBaseMessage 撤回已发布或发布中的消息。
func (c *BaseMessageCase) RevokeBaseMessage(ctx context.Context, id int64) error {
	entity, err := c.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err = c.ensureTenantAccess(ctx, entity.TenantID); err != nil {
		return err
	}
	if entity.Status != int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHING) && entity.Status != int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHED) {
		return errorsx.Conflict("只有发布中的消息或已发布消息可以撤回")
	}
	now := time.Now()
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		messageQuery := c.Query(txCtx).BaseMessage
		var result gen.ResultInfo
		result, err = messageQuery.WithContext(txCtx).
			Where(messageQuery.ID.Eq(entity.ID), messageQuery.Version.Eq(entity.Version)).
			UpdateSimple(
				messageQuery.Status.Value(int32(basev1.MessageStatus_MESSAGE_STATUS_REVOKED)),
				messageQuery.RevokedAt.Value(now.UnixMilli()),
				messageQuery.Version.Value(entity.Version+1),
				messageQuery.UpdatedAt.Value(now),
			)
		if err != nil {
			return err
		}
		if result.RowsAffected == 0 {
			return errorsx.Conflict("消息状态已变化，请刷新后重试")
		}
		entity.Status = int32(basev1.MessageStatus_MESSAGE_STATUS_REVOKED)
		entity.RevokedAt = now.UnixMilli()
		entity.Version++
		entity.UpdatedAt = now
		err = c.deliveryWriter.RevokeMessage(txCtx, entity.ID, now)
		if err != nil {
			return err
		}
		dispatchQuery := c.dispatchRepo.Query(txCtx).BaseMessageDispatch
		_, err = dispatchQuery.WithContext(txCtx).
			Where(dispatchQuery.MessageID.Eq(entity.ID)).
			UpdateSimple(
				dispatchQuery.Status.Value(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_CANCELLED)),
				dispatchQuery.LockToken.Value(""),
				dispatchQuery.LockedUntil.Value(0),
				dispatchQuery.Version.Add(1),
				dispatchQuery.UpdatedAt.Value(now),
			)
		return err
	})
}

// RetryBaseMessageDispatch 重试失败的消息投递任务。
func (c *BaseMessageCase) RetryBaseMessageDispatch(ctx context.Context, id int64) error {
	dispatch, err := c.dispatchRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err = c.ensureTenantAccess(ctx, dispatch.TenantID); err != nil {
		return err
	}
	var message *models.BaseMessage
	message, err = c.FindByID(ctx, dispatch.MessageID)
	if err != nil {
		return err
	}
	if message.Status == int32(basev1.MessageStatus_MESSAGE_STATUS_REVOKED) {
		return errorsx.Conflict("已撤回消息不能重试投递")
	}
	if dispatch.Status != int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_FAILED) {
		return errorsx.Conflict("只有失败的投递任务可以重试")
	}
	now := time.Now()
	query := c.dispatchRepo.Query(ctx).BaseMessageDispatch
	var result gen.ResultInfo
	result, err = query.WithContext(ctx).
		Where(query.ID.Eq(dispatch.ID), query.Status.Eq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_FAILED)), query.Version.Eq(dispatch.Version)).
		UpdateSimple(
			query.Status.Value(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING)),
			query.AttemptCount.Value(0),
			query.NextRetryAt.Value(0),
			query.LastError.Value(""),
			query.LockToken.Value(""),
			query.LockedUntil.Value(0),
			query.QueuedAt.Value(now.UnixMilli()),
			query.Version.Value(dispatch.Version+1),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return err
	}
	if result.RowsAffected == 0 {
		return errorsx.Conflict("投递状态已变化，请刷新后重试")
	}
	dispatch.Version++
	return c.EnqueueDispatch(ctx, dispatch.ID, dispatch.TenantID)
}

// EnqueueDispatch 将投递任务定位信息写入 Redis Streams。
func (c *BaseMessageCase) EnqueueDispatch(ctx context.Context, dispatchID, tenantID int64) error {
	dispatch, err := c.dispatchRepo.FindByID(ctx, dispatchID)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(&dto.MessageDispatchTask{DispatchID: dispatchID, TenantID: tenantID, ExpectedVersion: dispatch.Version})
	if err != nil {
		return err
	}
	if c.Queue == nil {
		return errorsx.Internal("消息队列未初始化")
	}
	return c.Queue.Append(messageDispatchStream, queueData.Message{Values: map[string]interface{}{"data": string(payload)}})
}

// enqueuePendingDispatches 唤醒指定消息尚未完成的投递任务。
func (c *BaseMessageCase) enqueuePendingDispatches(ctx context.Context, messageID, tenantID int64) error {
	query := c.dispatchRepo.Query(ctx).BaseMessageDispatch
	dispatches, err := c.dispatchRepo.List(ctx,
		repository.Where(query.MessageID.Eq(messageID)),
		repository.Where(query.Status.Eq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING))),
		repository.Order(query.ID.Asc()),
	)
	if err != nil {
		return err
	}
	for _, dispatch := range dispatches {
		if err = c.EnqueueDispatch(ctx, dispatch.ID, tenantID); err != nil {
			return err
		}
	}
	return nil
}

// HandleDispatchMessage 处理 Redis Streams 消息投递任务。
func (c *BaseMessageCase) HandleDispatchMessage(message queueData.Message) error {
	task, err := queue.Decode[dto.MessageDispatchTask](message)
	if err != nil {
		return err
	}
	return c.ProcessDispatch(context.Background(), task)
}

// ProcessDispatch 执行一批受众展开并按需续投下一批。
func (c *BaseMessageCase) ProcessDispatch(ctx context.Context, task *dto.MessageDispatchTask) error {
	dispatch, err := c.dispatchRepo.FindByID(ctx, task.DispatchID)
	if err != nil {
		return err
	}
	if dispatch.TenantID != task.TenantID || dispatch.Status == int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_SUCCEEDED) || dispatch.Status == int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_CANCELLED) {
		return nil
	}
	if task.ExpectedVersion > 0 && dispatch.Version != task.ExpectedVersion {
		return nil
	}
	var message *models.BaseMessage
	message, err = c.FindByID(ctx, dispatch.MessageID)
	if err != nil {
		return err
	}
	if message.Status == int32(basev1.MessageStatus_MESSAGE_STATUS_REVOKED) {
		return nil
	}
	now := time.Now()
	var claimed bool
	claimed, err = c.claimDispatch(ctx, dispatch, now)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	var users []*models.BaseUser
	users, err = c.listDispatchUsers(ctx, dispatch)
	if err != nil {
		return c.failDispatch(ctx, dispatch, err, 0)
	}
	var insertedTotal int64
	var completed bool
	var updated bool
	var revoked bool
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		// 撤回事务也先锁定消息行，确保撤回与收件写入按同一顺序串行化。
		messageQuery := c.Query(txCtx).BaseMessage
		message, err = c.Find(txCtx,
			repository.Where(messageQuery.ID.Eq(dispatch.MessageID)),
			repository.Clauses(clause.Locking{Strength: "UPDATE"}),
		)
		if err != nil {
			return err
		}
		if message.Status == int32(basev1.MessageStatus_MESSAGE_STATUS_REVOKED) {
			revoked = true
			return nil
		}
		deliveries := make([]*models.BaseMessageDelivery, 0, len(users))
		for _, user := range users {
			deliveries = append(deliveries, &models.BaseMessageDelivery{
				TenantID: dispatch.TenantID, MessageID: dispatch.MessageID, UserID: user.ID,
				ReceivedAt: now, ExpiresAt: message.ExpiresAt, CreatedAt: now, UpdatedAt: now,
			})
		}
		insertedTotal, err = c.deliveryWriter.CreateIgnore(txCtx, deliveries, int(dispatch.BatchSize))
		if err != nil {
			return err
		}
		if len(users) > 0 {
			dispatch.CursorUserID = users[len(users)-1].ID
		}
		dispatch.MatchedTotal += int64(len(users))
		dispatch.InsertedTotal += insertedTotal
		dispatch.UpdatedAt = time.Now()
		completed = len(users) < int(dispatch.BatchSize)
		if completed {
			dispatch.Status = int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_SUCCEEDED)
			dispatch.CompletedAt = dispatch.UpdatedAt.UnixMilli()
		} else {
			dispatch.Status = int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING)
		}
		dispatchQuery := c.dispatchRepo.Query(txCtx).BaseMessageDispatch
		updated, err = c.updateOwnedDispatch(txCtx, dispatch,
			dispatchQuery.CursorUserID.Value(dispatch.CursorUserID),
			dispatchQuery.MatchedTotal.Value(dispatch.MatchedTotal),
			dispatchQuery.InsertedTotal.Value(dispatch.InsertedTotal),
			dispatchQuery.Status.Value(dispatch.Status),
			dispatchQuery.CompletedAt.Value(dispatch.CompletedAt),
			dispatchQuery.NextRetryAt.Value(0),
			dispatchQuery.LastError.Value(""),
			dispatchQuery.LockToken.Value(""),
			dispatchQuery.LockedUntil.Value(0),
			dispatchQuery.QueuedAt.Value(dispatch.UpdatedAt.UnixMilli()),
			dispatchQuery.UpdatedAt.Value(dispatch.UpdatedAt),
		)
		return err
	})
	if err != nil {
		return c.failDispatch(ctx, dispatch, err, int64(len(users)))
	}
	if revoked || !updated {
		return nil
	}
	for _, user := range users {
		c.publishNotificationChanged(ctx, dispatch.TenantID, user.ID, message.ID)
	}
	if !completed {
		return c.EnqueueDispatch(ctx, dispatch.ID, dispatch.TenantID)
	}
	return c.finishMessageIfCompleted(ctx, message)
}

// claimDispatch 原子抢占可执行的消息投递任务并建立执行租约。
func (c *BaseMessageCase) claimDispatch(ctx context.Context, dispatch *models.BaseMessageDispatch, now time.Time) (bool, error) {
	query := c.dispatchRepo.Query(ctx).BaseMessageDispatch
	retryReady := field.Or(query.NextRetryAt.Eq(0), query.NextRetryAt.Lte(now.UnixMilli()))
	claimable := field.Or(
		field.And(
			query.Status.Eq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING)),
			retryReady,
		),
		field.And(
			query.Status.Eq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_RUNNING)),
			query.LockedUntil.Lte(now.UnixMilli()),
		),
	)
	lockToken := id.NewGUIDv4NoHyphen()
	lockedUntil := now.Add(messageDispatchLease).UnixMilli()
	result, err := query.WithContext(ctx).
		Where(query.ID.Eq(dispatch.ID), query.Version.Eq(dispatch.Version), claimable).
		UpdateSimple(
			query.Status.Value(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_RUNNING)),
			query.StartedAt.Value(now.UnixMilli()),
			query.AttemptCount.Value(dispatch.AttemptCount+1),
			query.NextRetryAt.Value(0),
			query.LastError.Value(""),
			query.LockToken.Value(lockToken),
			query.LockedUntil.Value(lockedUntil),
			query.Version.Value(dispatch.Version+1),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return false, err
	}
	if result.RowsAffected == 0 {
		return false, nil
	}
	dispatch.Status = int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_RUNNING)
	dispatch.StartedAt = now.UnixMilli()
	dispatch.AttemptCount++
	dispatch.LockToken = lockToken
	dispatch.LockedUntil = lockedUntil
	dispatch.Version++
	dispatch.UpdatedAt = now
	return true, nil
}

// updateOwnedDispatch 仅允许持有当前租约的消费者更新投递任务。
func (c *BaseMessageCase) updateOwnedDispatch(ctx context.Context, dispatch *models.BaseMessageDispatch, values ...field.AssignExpr) (bool, error) {
	if dispatch.LockToken == "" {
		return false, nil
	}
	query := c.dispatchRepo.Query(ctx).BaseMessageDispatch
	result, err := query.WithContext(ctx).
		Where(
			query.ID.Eq(dispatch.ID),
			query.Status.Eq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_RUNNING)),
			query.LockToken.Eq(dispatch.LockToken),
			query.Version.Eq(dispatch.Version),
		).
		UpdateSimple(append(values, query.Version.Value(dispatch.Version+1))...)
	if err != nil {
		return false, err
	}
	if result.RowsAffected > 0 {
		dispatch.Version++
	}
	return result.RowsAffected > 0, nil
}

// RecoverPendingDispatches 恢复到期定时消息和遗漏的待投递任务。
func (c *BaseMessageCase) RecoverPendingDispatches(ctx context.Context) (int, error) {
	now := time.Now()
	messageQuery := c.Query(ctx).BaseMessage
	messages, err := c.List(ctx,
		repository.Where(messageQuery.Status.Eq(int32(basev1.MessageStatus_MESSAGE_STATUS_SCHEDULED))),
		repository.Where(messageQuery.ScheduledAt.Gt(0)),
		repository.Where(messageQuery.ScheduledAt.Lte(now.UnixMilli())),
	)
	if err != nil {
		return 0, err
	}
	for _, message := range messages {
		var result gen.ResultInfo
		result, err = messageQuery.WithContext(ctx).
			Where(
				messageQuery.ID.Eq(message.ID),
				messageQuery.Status.Eq(int32(basev1.MessageStatus_MESSAGE_STATUS_SCHEDULED)),
				messageQuery.Version.Eq(message.Version),
				messageQuery.ScheduledAt.Lte(now.UnixMilli()),
			).
			UpdateSimple(
				messageQuery.Status.Value(int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHING)),
				messageQuery.Version.Value(message.Version+1),
				messageQuery.UpdatedAt.Value(now),
			)
		if err != nil {
			return 0, err
		}
		if result.RowsAffected == 0 {
			continue
		}
		message.Version++
	}
	dispatchQuery := c.dispatchRepo.Query(ctx).BaseMessageDispatch
	retryReady := field.Or(dispatchQuery.NextRetryAt.Eq(0), dispatchQuery.NextRetryAt.Lte(now.UnixMilli()))
	var dispatches []*models.BaseMessageDispatch
	dispatches, err = c.dispatchRepo.List(ctx,
		repository.Where(field.Or(
			dispatchQuery.Status.Eq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_WAITING)),
			field.And(
				dispatchQuery.Status.Eq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING)),
				retryReady,
			),
			field.And(
				dispatchQuery.Status.Eq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_RUNNING)),
				dispatchQuery.LockedUntil.Lte(now.UnixMilli()),
			),
		)),
	)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, dispatch := range dispatches {
		var message *models.BaseMessage
		message, err = c.FindByID(ctx, dispatch.MessageID)
		if err != nil || message.Status != int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHING) {
			continue
		}
		var result gen.ResultInfo
		result, err = dispatchQuery.WithContext(ctx).
			Where(
				dispatchQuery.ID.Eq(dispatch.ID),
				dispatchQuery.Version.Eq(dispatch.Version),
				field.Or(
					dispatchQuery.Status.Eq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_WAITING)),
					field.And(
						dispatchQuery.Status.Eq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING)),
						retryReady,
					),
					field.And(
						dispatchQuery.Status.Eq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_RUNNING)),
						dispatchQuery.LockedUntil.Lte(now.UnixMilli()),
					),
				),
			).
			UpdateSimple(
				dispatchQuery.Status.Value(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING)),
				dispatchQuery.LockToken.Value(""),
				dispatchQuery.LockedUntil.Value(0),
				dispatchQuery.QueuedAt.Value(now.UnixMilli()),
				dispatchQuery.Version.Value(dispatch.Version+1),
				dispatchQuery.UpdatedAt.Value(now),
			)
		if err != nil {
			return count, err
		}
		if result.RowsAffected == 0 {
			continue
		}
		dispatch.Version++
		if err = c.EnqueueDispatch(ctx, dispatch.ID, dispatch.TenantID); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// CleanupExpiredDeliveries 分批软删除已过期的用户收件记录。
func (c *BaseMessageCase) CleanupExpiredDeliveries(ctx context.Context) (int, error) {
	now := time.Now().UnixMilli()
	query := c.deliveryRepo.Query(ctx).BaseMessageDelivery
	list, err := c.deliveryRepo.List(ctx,
		repository.Where(query.ExpiresAt.Gt(0)),
		repository.Where(query.ExpiresAt.Lte(now)),
		repository.Order(query.ID.Asc()),
		repository.Limit(messageDeliveryCleanupBatchSize),
	)
	if err != nil {
		return 0, err
	}
	if len(list) == 0 {
		return 0, nil
	}
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ID)
	}
	if err = c.deliveryRepo.DeleteByIDs(ctx, ids); err != nil {
		return 0, err
	}
	return len(ids), nil
}

// validateMessageForm 校验消息分类、动作和受众范围。
func (c *BaseMessageCase) validateMessageForm(ctx context.Context, tenantID int64, req *adminv1.BaseMessageForm) error {
	if len(req.GetAudiences()) == 0 {
		return errorsx.InvalidArgument("消息受众不能为空")
	}
	category, err := c.categoryCase.FindByID(ctx, req.GetCategoryId())
	if err != nil {
		return errorsx.ResourceNotFound("消息分类不存在").WithCause(err)
	}
	if category.Status != coreconst.STATUS_STATUS_ENABLE {
		return errorsx.InvalidArgument("消息分类不可用")
	}
	switch req.GetContentFormat() {
	case basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_UNSPECIFIED,
		basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT,
		basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_SAFE_MARKDOWN,
		basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_RICH_TEXT:
	default:
		return errorsx.InvalidArgument("消息正文格式无效")
	}
	switch req.GetPriority() {
	case basev1.MessagePriority_MESSAGE_PRIORITY_UNSPECIFIED,
		basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL,
		basev1.MessagePriority_MESSAGE_PRIORITY_IMPORTANT,
		basev1.MessagePriority_MESSAGE_PRIORITY_URGENT:
	default:
		return errorsx.InvalidArgument("消息优先级无效")
	}
	switch req.GetActionType() {
	case basev1.MessageActionType_MESSAGE_ACTION_TYPE_UNSPECIFIED,
		basev1.MessageActionType_MESSAGE_ACTION_TYPE_VIEW_KEY:
	default:
		return errorsx.InvalidArgument("消息动作类型无效")
	}
	if req.GetActionType() == basev1.MessageActionType_MESSAGE_ACTION_TYPE_VIEW_KEY && req.GetActionTarget() == "" {
		return errorsx.InvalidArgument("消息动作目标不能为空")
	}
	if utf8.RuneCountInString(req.GetActionTarget()) > 255 {
		return errorsx.InvalidArgument("消息动作目标不能超过255个字符")
	}
	if err = c.validateActionTarget(ctx, req.GetActionType(), req.GetActionTarget()); err != nil {
		return err
	}
	if err = validateMessageActionParams(req.GetActionParams()); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(req.GetAudiences()))
	for _, audience := range req.GetAudiences() {
		key := messageAudienceKey(audience)
		if _, exists := seen[key]; exists {
			return errorsx.InvalidArgument("消息受众不能重复")
		}
		seen[key] = struct{}{}
		if err = c.validateAudience(ctx, tenantID, audience); err != nil {
			return err
		}
	}
	return nil
}

// messageAudienceKey 返回受众去重键，仅部门受众区分是否包含子部门。
func messageAudienceKey(audience *adminv1.BaseMessageAudienceForm) string {
	includeChildren := audience.GetType() == basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_DEPT && audience.GetIncludeChildren()
	return fmt.Sprintf("%d:%d:%t", audience.GetType(), audience.GetId(), includeChildren)
}

// validateActionTarget 校验消息动作目标是否为已启用移动端菜单注册的稳定视图键。
func (c *BaseMessageCase) validateActionTarget(ctx context.Context, actionType basev1.MessageActionType, target string) error {
	if actionType != basev1.MessageActionType_MESSAGE_ACTION_TYPE_VIEW_KEY {
		return nil
	}
	if c.baseMenuRepo == nil {
		return errorsx.Internal("消息动作目标校验未初始化")
	}
	query := c.baseMenuRepo.Query(ctx).BaseMenu
	opts := []repository.QueryOption{
		repository.Where(query.ID.Gt(_const.BASE_MENU_APP_ROOT_ID)),
		repository.Where(query.Type.Eq(_const.BASE_MENU_TYPE_MENU)),
		repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)),
	}
	menus, err := c.baseMenuRepo.List(ctx, opts...)
	if err != nil {
		return errorsx.Internal("查询消息动作目标失败").WithCause(err)
	}
	for _, menu := range menus {
		var meta struct {
			App struct {
				ViewKey string `json:"view_key"`
			} `json:"app"`
		}
		if err = json.Unmarshal([]byte(menu.Meta), &meta); err != nil {
			continue
		}
		if meta.App.ViewKey == target {
			return nil
		}
	}
	return errorsx.InvalidArgument("消息动作目标无效")
}

// validateAudience 校验单项受众存在且属于消息租户。
func (c *BaseMessageCase) validateAudience(ctx context.Context, tenantID int64, audience *adminv1.BaseMessageAudienceForm) error {
	var err error
	switch audience.GetType() {
	case basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_TENANT:
		if audience.GetId() != 0 {
			return errorsx.InvalidArgument("租户全员受众ID必须为0")
		}
		return nil
	case basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_USER:
		var entity *models.BaseUser
		entity, err = c.baseUserRepo.FindByID(ctx, audience.GetId())
		if err != nil || entity.TenantID != tenantID {
			return errorsx.InvalidArgument("消息用户受众无效").WithCause(err)
		}
	case basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_ROLE:
		var entity *models.BaseRole
		entity, err = c.baseRoleRepo.FindByID(ctx, audience.GetId())
		if err != nil || entity.TenantID != tenantID {
			return errorsx.InvalidArgument("消息角色受众无效").WithCause(err)
		}
	case basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_DEPT:
		var entity *models.BaseDept
		entity, err = c.baseDeptRepo.FindByID(ctx, audience.GetId())
		if err != nil || entity.TenantID != tenantID {
			return errorsx.InvalidArgument("消息部门受众无效").WithCause(err)
		}
	case basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_POST:
		var entity *models.BasePost
		entity, err = c.basePostRepo.FindByID(ctx, audience.GetId())
		if err != nil || entity.TenantID != tenantID {
			return errorsx.InvalidArgument("消息岗位受众无效").WithCause(err)
		}
	default:
		return errorsx.InvalidArgument("消息受众类型无效")
	}
	return nil
}

// newBaseMessage 根据管理表单构造消息实体。
func (c *BaseMessageCase) newBaseMessage(ctx context.Context, req *adminv1.BaseMessageForm, tenantID, senderID int64, senderName string) (*models.BaseMessage, error) {
	contentFormat := req.GetContentFormat()
	if contentFormat == basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_UNSPECIFIED {
		contentFormat = basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT
	}
	category, err := c.categoryCase.FindByID(ctx, req.GetCategoryId())
	if err != nil {
		return nil, errorsx.ResourceNotFound("消息分类不存在").WithCause(err)
	}
	priority := req.GetPriority()
	if priority == basev1.MessagePriority_MESSAGE_PRIORITY_UNSPECIFIED {
		priority = basev1.MessagePriority(category.DefaultPriority)
		if priority == basev1.MessagePriority_MESSAGE_PRIORITY_UNSPECIFIED {
			priority = basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL
		}
	}
	var content string
	content, err = sanitizeMessageContent(req.GetContent(), contentFormat)
	if err != nil {
		return nil, err
	}
	var payload []byte
	payload, err = json.Marshal(req)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(payload)
	now := time.Now()
	actionParams := normalizeMessageActionParams(req.GetActionParams())
	return &models.BaseMessage{
		TenantID: tenantID, CategoryID: req.GetCategoryId(), SourceType: int32(basev1.MessageSourceType_MESSAGE_SOURCE_TYPE_ADMIN), Source: "admin",
		IdempotencyKey: id.NewGUIDv4NoHyphen(), PayloadHash: hex.EncodeToString(hash[:]), SenderType: int32(basev1.MessageSenderType_MESSAGE_SENDER_TYPE_USER),
		SenderID: senderID, SenderName: senderName, Priority: int32(priority), Title: req.GetTitle(), Content: content,
		ContentFormat: int32(contentFormat), ActionType: int32(req.GetActionType()), ActionTarget: req.GetActionTarget(), ActionParams: actionParams,
		Status: int32(basev1.MessageStatus_MESSAGE_STATUS_DRAFT), ScheduledAt: parseMessageTime(req.GetScheduledAt()), ExpiresAt: parseMessageTime(req.GetExpiresAt()),
		Version: 1, CreatedBy: senderID, UpdatedBy: senderID, CreatedAt: now, UpdatedAt: now,
	}, nil
}

// sanitizeMessageContent 按正文格式清洗富文本或 Markdown 原生 HTML，阻止可执行标记入库。
func sanitizeMessageContent(content string, format basev1.MessageContentFormat) (string, error) {
	switch format {
	case basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_SAFE_MARKDOWN:
		content = messageMarkdownPolicy.Sanitize(content)
	case basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_RICH_TEXT:
		content = messageRichTextPolicy.Sanitize(content)
	default:
		return content, nil
	}
	if content == "" {
		return "", errorsx.InvalidArgument("消息正文清洗后不能为空")
	}
	return content, nil
}

// normalizeMessageActionParams 将空动作参数转换为可持久化的 JSON 对象。
func normalizeMessageActionParams(raw string) string {
	if raw == "" {
		return defaultMessageActionParams
	}
	return raw
}

// validatePublishedMessage 校验绕过 Proto 的业务发布请求边界。
func validatePublishedMessage(request notification.Message) error {
	if request.CategoryCode == "" {
		return errorsx.InvalidArgument("消息分类编码不能为空")
	}
	if utf8.RuneCountInString(request.Title) == 0 || utf8.RuneCountInString(request.Title) > 200 {
		return errorsx.InvalidArgument("消息标题不能为空且不能超过200个字符")
	}
	if utf8.RuneCountInString(request.Content) == 0 || utf8.RuneCountInString(request.Content) > 20000 {
		return errorsx.InvalidArgument("消息正文不能为空且不能超过20000个字符")
	}
	if utf8.RuneCountInString(request.Source) > 64 || utf8.RuneCountInString(request.IdempotencyKey) > 128 || utf8.RuneCountInString(request.SenderName) > 50 {
		return errorsx.InvalidArgument("消息来源、幂等键或发送者名称长度无效")
	}
	if utf8.RuneCountInString(request.ActionTarget) > 255 {
		return errorsx.InvalidArgument("消息动作目标不能超过255个字符")
	}
	switch request.ContentFormat {
	case basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_UNSPECIFIED,
		basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_PLAIN_TEXT,
		basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_SAFE_MARKDOWN,
		basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_RICH_TEXT:
	default:
		return errorsx.InvalidArgument("消息正文格式无效")
	}
	switch request.Priority {
	case basev1.MessagePriority_MESSAGE_PRIORITY_UNSPECIFIED,
		basev1.MessagePriority_MESSAGE_PRIORITY_NORMAL,
		basev1.MessagePriority_MESSAGE_PRIORITY_IMPORTANT,
		basev1.MessagePriority_MESSAGE_PRIORITY_URGENT:
	default:
		return errorsx.InvalidArgument("消息优先级无效")
	}
	switch request.ActionType {
	case basev1.MessageActionType_MESSAGE_ACTION_TYPE_UNSPECIFIED,
		basev1.MessageActionType_MESSAGE_ACTION_TYPE_VIEW_KEY:
	default:
		return errorsx.InvalidArgument("消息动作类型无效")
	}
	var err error
	err = validateMessageActionParams(request.ActionParams)
	if err != nil {
		return err
	}
	if request.ExpiresAt > 0 && request.ExpiresAt <= time.Now().UnixMilli() {
		return errorsx.InvalidArgument("消息过期时间必须晚于当前时间")
	}
	return nil
}

// validateMessageActionParams 校验动作参数 JSON 的大小、深度和字符串边界。
func validateMessageActionParams(raw string) error {
	if raw == "" {
		return nil
	}
	if len([]byte(raw)) > maxMessageActionParamsBytes {
		return errorsx.InvalidArgument("消息动作参数不能超过16KB")
	}
	var value interface{}
	var err error
	err = json.Unmarshal([]byte(raw), &value)
	if err != nil {
		return errorsx.InvalidArgument("消息动作参数必须是有效JSON")
	}
	return validateMessageActionValue(value, 0)
}

// validateMessageActionValue 递归校验动作参数对象的结构边界。
func validateMessageActionValue(value interface{}, depth int) error {
	var err error
	switch item := value.(type) {
	case map[string]interface{}:
		if depth >= maxMessageActionDepth {
			return errorsx.InvalidArgument("消息动作参数嵌套层级过深")
		}
		for key, child := range item {
			if utf8.RuneCountInString(key) > maxMessageActionKeyRunes {
				return errorsx.InvalidArgument("消息动作参数键名过长")
			}
			err = validateMessageActionValue(child, depth+1)
			if err != nil {
				return err
			}
		}
	case []interface{}:
		if depth >= maxMessageActionDepth {
			return errorsx.InvalidArgument("消息动作参数嵌套层级过深")
		}
		for _, child := range item {
			err = validateMessageActionValue(child, depth+1)
			if err != nil {
				return err
			}
		}
	case string:
		if utf8.RuneCountInString(item) > maxMessageActionStringRunes {
			return errorsx.InvalidArgument("消息动作参数值过长")
		}
	}
	return nil
}

// replaceDispatches 替换草稿消息的受众投递任务。
func (c *BaseMessageCase) replaceDispatches(ctx context.Context, message *models.BaseMessage, audiences []*adminv1.BaseMessageAudienceForm, status basev1.MessageDispatchStatus) error {
	query := c.dispatchRepo.Query(ctx).BaseMessageDispatch
	err := c.dispatchRepo.Delete(ctx, repository.Where(query.MessageID.Eq(message.ID)))
	if err != nil {
		return err
	}
	now := time.Now()
	list := make([]*models.BaseMessageDispatch, 0, len(audiences))
	for _, audience := range audiences {
		queuedAt := int64(0)
		if status == basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING {
			queuedAt = now.UnixMilli()
		}
		list = append(list, &models.BaseMessageDispatch{
			TenantID: message.TenantID, MessageID: message.ID, AudienceType: int32(audience.GetType()), AudienceID: audience.GetId(),
			IncludeChildren: audience.GetIncludeChildren(), Status: int32(status), BatchSize: messageDispatchBatchSize,
			Version: 1, QueuedAt: queuedAt, CreatedBy: message.UpdatedBy, UpdatedBy: message.UpdatedBy, CreatedAt: now, UpdatedAt: now,
		})
	}
	return c.dispatchRepo.BatchCreate(ctx, list)
}

// toBaseMessageForm 转换消息详情表单。
func (c *BaseMessageCase) toBaseMessageForm(entity *models.BaseMessage, dispatches []*models.BaseMessageDispatch) *adminv1.BaseMessageForm {
	form := &adminv1.BaseMessageForm{
		Id: entity.ID, TenantId: entity.TenantID, CategoryId: entity.CategoryID, Title: entity.Title, Content: entity.Content,
		ContentFormat: basev1.MessageContentFormat(entity.ContentFormat), Priority: basev1.MessagePriority(entity.Priority),
		ActionType: basev1.MessageActionType(entity.ActionType), ActionTarget: entity.ActionTarget, ActionParams: entity.ActionParams,
		ScheduledAt: formatMessageMillis(entity.ScheduledAt), ExpiresAt: formatMessageMillis(entity.ExpiresAt), Version: entity.Version,
	}
	for _, item := range dispatches {
		form.Audiences = append(form.Audiences, &adminv1.BaseMessageAudienceForm{Type: basev1.MessageAudienceType(item.AudienceType), Id: item.AudienceID, IncludeChildren: item.IncludeChildren})
	}
	return form
}

// toBaseMessage 转换消息列表项。
func (c *BaseMessageCase) toBaseMessage(entity *models.BaseMessage, categoryName string) *adminv1.BaseMessage {
	return &adminv1.BaseMessage{
		Id: entity.ID, TenantId: entity.TenantID, CategoryId: entity.CategoryID, CategoryName: categoryName,
		Title: entity.Title, Priority: basev1.MessagePriority(entity.Priority), Status: basev1.MessageStatus(entity.Status),
		SenderName: entity.SenderName, RecipientTotal: entity.RecipientTotal, DeliveredTotal: entity.DeliveredTotal, FailedTotal: entity.FailedTotal,
		ScheduledAt: formatMessageMillis(entity.ScheduledAt), PublishedAt: formatMessageMillis(entity.PublishedAt),
		CreatedAt: _time.TimeToTimeString(entity.CreatedAt), UpdatedAt: _time.TimeToTimeString(entity.UpdatedAt),
	}
}

// toBaseMessageDispatch 转换消息投递任务。
func (c *BaseMessageCase) toBaseMessageDispatch(entity *models.BaseMessageDispatch) *adminv1.BaseMessageDispatch {
	return &adminv1.BaseMessageDispatch{
		Id: entity.ID, AudienceType: basev1.MessageAudienceType(entity.AudienceType), AudienceId: entity.AudienceID,
		IncludeChildren: entity.IncludeChildren, Status: basev1.MessageDispatchStatus(entity.Status), MatchedTotal: entity.MatchedTotal,
		InsertedTotal: entity.InsertedTotal, AttemptCount: entity.AttemptCount, LastError: entity.LastError,
	}
}

// listDispatchUsers 查询当前投递任务的下一批用户。
func (c *BaseMessageCase) listDispatchUsers(ctx context.Context, dispatch *models.BaseMessageDispatch) ([]*models.BaseUser, error) {
	query := c.baseUserRepo.Query(ctx).BaseUser
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts,
		repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)),
		repository.Where(query.ID.Gt(dispatch.CursorUserID)),
		repository.Order(query.ID.Asc()),
		repository.Limit(int(dispatch.BatchSize)),
	)
	switch basev1.MessageAudienceType(dispatch.AudienceType) {
	case basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_USER:
		opts = append(opts, repository.Where(query.ID.Eq(dispatch.AudienceID)))
	case basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_ROLE:
		opts = append(opts, repository.Where(query.RoleID.Eq(dispatch.AudienceID)))
	case basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_DEPT:
		deptIDs := []int64{dispatch.AudienceID}
		if dispatch.IncludeChildren {
			dept, err := c.baseDeptRepo.FindByID(ctx, dispatch.AudienceID)
			if err != nil {
				return nil, err
			}
			if dept.TenantID != dispatch.TenantID {
				return nil, errorsx.InvalidArgument("消息部门受众无效")
			}
			deptQuery := c.baseDeptRepo.Query(ctx).BaseDept
			var depts []*models.BaseDept
			depts, err = c.baseDeptRepo.List(ctx,
				repository.Where(field.Or(deptQuery.Path.Eq(dept.Path), deptQuery.Path.Like(dept.Path+"/%"))),
			)
			if err != nil {
				return nil, err
			}
			deptIDs = make([]int64, 0, len(depts))
			for _, item := range depts {
				deptIDs = append(deptIDs, item.ID)
			}
		}
		opts = append(opts, repository.Where(query.DeptID.In(deptIDs...)))
	case basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_POST:
		opts = append(opts, repository.Where(query.PostID.Eq(dispatch.AudienceID)))
	case basev1.MessageAudienceType_MESSAGE_AUDIENCE_TYPE_TENANT:
	default:
		return nil, errorsx.InvalidArgument("消息受众类型无效")
	}
	return c.baseUserRepo.List(ctx, opts...)
}

// failDispatch 记录投递失败状态和脱敏错误。
func (c *BaseMessageCase) failDispatch(ctx context.Context, dispatch *models.BaseMessageDispatch, cause error, failedCount int64) error {
	lastError := cause.Error()
	if len(lastError) > 500 {
		lastError = lastError[:500]
	}
	now := time.Now()
	status := int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_FAILED)
	nextRetryAt := int64(0)
	if dispatch.AttemptCount < messageDispatchMaxAttempts {
		status = int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_PENDING)
		nextRetryAt = now.Add(messageDispatchRetryDelay(dispatch.AttemptCount)).UnixMilli()
	}
	query := c.dispatchRepo.Query(ctx).BaseMessageDispatch
	var err error
	var updated bool
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		updated, err = c.updateOwnedDispatch(txCtx, dispatch,
			query.Status.Value(status),
			query.LastError.Value(lastError),
			query.NextRetryAt.Value(nextRetryAt),
			query.LockToken.Value(""),
			query.LockedUntil.Value(0),
			query.UpdatedAt.Value(now),
		)
		if err != nil || !updated || status != int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_FAILED) || failedCount <= 0 {
			return err
		}
		messageQuery := c.Query(txCtx).BaseMessage
		_, err = messageQuery.WithContext(txCtx).
			Where(messageQuery.ID.Eq(dispatch.MessageID)).
			UpdateSimple(messageQuery.FailedTotal.Add(failedCount), messageQuery.UpdatedAt.Value(now))
		return err
	})
	if err != nil {
		return err
	}
	return cause
}

// messageDispatchRetryDelay 计算投递失败后的指数退避时间，并限制最大等待时长。
func messageDispatchRetryDelay(attempt int32) time.Duration {
	if attempt <= 1 {
		return messageDispatchRetryBase
	}
	delay := messageDispatchRetryBase
	for index := int32(1); index < attempt; index++ {
		delay *= 2
		if delay >= messageDispatchRetryMaxDelay {
			return messageDispatchRetryMaxDelay
		}
	}
	return delay
}

// finishMessageIfCompleted 在全部投递任务完成后更新消息完成状态。
func (c *BaseMessageCase) finishMessageIfCompleted(ctx context.Context, message *models.BaseMessage) error {
	var err error
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		dispatchQuery := c.dispatchRepo.Query(txCtx).BaseMessageDispatch
		var remaining int64
		remaining, err = c.dispatchRepo.Count(txCtx,
			repository.Where(dispatchQuery.MessageID.Eq(message.ID)),
			repository.Where(dispatchQuery.Status.Neq(int32(basev1.MessageDispatchStatus_MESSAGE_DISPATCH_STATUS_SUCCEEDED))),
		)
		if err != nil || remaining > 0 {
			return err
		}
		deliveryQuery := c.deliveryRepo.Query(txCtx).BaseMessageDelivery
		var delivered int64
		delivered, err = c.deliveryRepo.Count(txCtx, repository.Where(deliveryQuery.MessageID.Eq(message.ID)))
		if err != nil {
			return err
		}
		messageQuery := c.Query(txCtx).BaseMessage
		var current *models.BaseMessage
		current, err = c.Find(txCtx,
			repository.Where(messageQuery.ID.Eq(message.ID)),
			repository.Clauses(clause.Locking{Strength: "UPDATE"}),
		)
		if err != nil {
			return err
		}
		if current.Status != int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHING) {
			return nil
		}
		now := time.Now()
		current.Status = int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHED)
		current.PublishedAt = now.UnixMilli()
		current.RecipientTotal = delivered
		current.DeliveredTotal = delivered
		current.Version++
		current.UpdatedAt = now
		return c.UpdateByID(txCtx, current)
	})
	return err
}

// publishNotificationChanged 尽力发布用户收件箱变化事件。
func (c *BaseMessageCase) publishNotificationChanged(ctx context.Context, tenantID, userID, messageID int64) {
	if c.sse == nil {
		return
	}
	streamID := fmt.Sprintf("%s:%d:%d", messageNotificationStream, tenantID, userID)
	c.sse.PublishJSON(ctx, streamID, messageNotificationEvent, map[string]any{"change_type": "delivered", "message_id": messageID})
}

// parseMessageTime 解析表单时间并返回毫秒时间戳。
func parseMessageTime(value string) int64 {
	parsed := _time.StringTimeToTime(value)
	if parsed == nil {
		return 0
	}
	return parsed.UnixMilli()
}

// formatMessageMillis 格式化毫秒时间戳。
func formatMessageMillis(value int64) string {
	if value <= 0 {
		return ""
	}
	return _time.TimeToTimeString(time.UnixMilli(value))
}

// messageRetentionDays 返回分类保留期，零值采用系统默认保留期。
func messageRetentionDays(category *models.BaseMessageCategory) int32 {
	if category == nil || category.RetentionDays <= 0 {
		return defaultMessageRetentionDays
	}
	return category.RetentionDays
}
