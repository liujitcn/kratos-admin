package biz

import (
	"context"
	"fmt"
	"sort"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	admindata "github.com/liujitcn/kratos-admin/backend/internal/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	coresse "github.com/liujitcn/kratos-core/sse"

	_time "github.com/liujitcn/go-utils/time"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen/field"
)

const notificationStreamID = "base.notification"
const notificationChangedEvent = "inbox.changed"
const notificationSummaryBatchSize = 500

// NotificationCase 处理当前用户站内信收件箱业务。
type NotificationCase struct {
	*biz.BaseCase
	*data.BaseMessageDeliveryRepository
	deliveryWriter *admindata.MessageDeliveryWriter
	messageRepo    *data.BaseMessageRepository
	categoryRepo   *data.BaseMessageCategoryRepository
	sse            *coresse.SSE
}

// NewNotificationCase 创建当前用户站内信业务实例。
func NewNotificationCase(
	baseCase *biz.BaseCase,
	deliveryRepo *data.BaseMessageDeliveryRepository,
	deliveryWriter *admindata.MessageDeliveryWriter,
	messageRepo *data.BaseMessageRepository,
	categoryRepo *data.BaseMessageCategoryRepository,
	sseRuntime *coresse.SSE,
) *NotificationCase {
	return &NotificationCase{
		BaseCase:                      baseCase,
		BaseMessageDeliveryRepository: deliveryRepo,
		deliveryWriter:                deliveryWriter,
		messageRepo:                   messageRepo,
		categoryRepo:                  categoryRepo,
		sse:                           sseRuntime,
	}
}

// PageNotification 分页查询当前用户收件箱。
func (c *NotificationCase) PageNotification(ctx context.Context, req *basev1.PageNotificationRequest) (*basev1.PageNotificationResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var messageIDs []int64
	messageIDs, err = c.filterMessageIDs(ctx, req)
	if err != nil {
		return nil, err
	}
	if messageIDs != nil && len(messageIDs) == 0 {
		return &basev1.PageNotificationResponse{Notifications: []*basev1.Notification{}}, nil
	}
	query := c.Query(ctx).BaseMessageDelivery
	opts := c.inboxOptions(ctx, authInfo.UserId, req.GetView())
	if messageIDs != nil {
		opts = append(opts, repository.Where(query.MessageID.In(messageIDs...)))
	}
	pageSize := req.GetPageSize()
	if pageSize <= 0 {
		pageSize = 20
	}
	var list []*models.BaseMessageDelivery
	var total int64
	var hasMore bool
	if req.GetCursorId() > 0 {
		opts = append(opts, repository.Where(query.ID.Lt(req.GetCursorId())), repository.Limit(int(pageSize+1)))
		list, err = c.List(ctx, opts...)
		if int64(len(list)) > pageSize {
			hasMore = true
			list = list[:pageSize]
		}
	} else {
		pageNum := req.GetPageNum()
		if pageNum <= 0 {
			pageNum = 1
		}
		list, total, err = c.Page(ctx, pageNum, pageSize, opts...)
		hasMore = pageNum*pageSize < total
	}
	if err != nil {
		return nil, err
	}
	var items []*basev1.Notification
	items, err = c.buildNotifications(ctx, list, false)
	if err != nil {
		return nil, err
	}
	var nextCursorID int64
	if hasMore && len(list) > 0 {
		nextCursorID = list[len(list)-1].ID
	}
	return &basev1.PageNotificationResponse{Notifications: items, Total: int32(total), NextCursorId: nextCursorID, HasMore: hasMore}, nil
}

// ListNotificationCategories 查询当前可用的消息分类。
func (c *NotificationCase) ListNotificationCategories(ctx context.Context, _ *basev1.ListNotificationCategoriesRequest) (*basev1.ListNotificationCategoriesResponse, error) {
	_, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseMessageCategory
	opts := []repository.QueryOption{
		repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)),
		repository.Order(query.Sort.Asc()),
		repository.Order(query.ID.Asc()),
	}
	var categories []*models.BaseMessageCategory
	categories, err = c.categoryRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*basev1.NotificationCategory, 0, len(categories))
	for _, category := range categories {
		result = append(result, &basev1.NotificationCategory{
			Id: category.ID, Code: category.Code, Name: category.Name,
			Icon: category.Icon, Color: category.Color, Sort: category.Sort,
		})
	}
	return &basev1.ListNotificationCategoriesResponse{Categories: result}, nil
}

// GetNotification 查询当前用户消息详情。
func (c *NotificationCase) GetNotification(ctx context.Context, id int64) (*basev1.Notification, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var delivery *models.BaseMessageDelivery
	delivery, err = c.findOwnedDelivery(ctx, authInfo.UserId, id)
	if err != nil {
		return nil, err
	}
	var items []*basev1.Notification
	items, err = c.buildNotifications(ctx, []*models.BaseMessageDelivery{delivery}, true)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errorsx.ResourceNotFound("消息不存在")
	}
	return items[0], nil
}

// GetNotificationSummary 查询当前用户未读汇总。
func (c *NotificationCase) GetNotificationSummary(ctx context.Context) (*basev1.NotificationSummary, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseMessageDelivery
	opts := c.visibleOptions(ctx, authInfo.UserId)
	opts = append(opts, repository.Where(query.ReadAt.Eq(0)), repository.Where(query.ArchivedAt.Eq(0)))
	var count int64
	count, err = c.Count(ctx, opts...)
	if err != nil {
		return nil, err
	}
	latestOpts := c.visibleOptions(ctx, authInfo.UserId)
	latestOpts = append(latestOpts, repository.Order(query.ID.Desc()), repository.Limit(1))
	var list []*models.BaseMessageDelivery
	list, err = c.List(ctx, latestOpts...)
	if err != nil {
		return nil, err
	}
	var latestID int64
	if len(list) > 0 {
		latestID = list[0].ID
	}
	var categoryUnread []*basev1.NotificationCategoryUnread
	categoryUnread, err = c.categoryUnread(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &basev1.NotificationSummary{UnreadTotal: count, LatestDeliveryId: latestID, CategoryUnread: categoryUnread}, nil
}

// MarkNotificationRead 标记消息为已读。
func (c *NotificationCase) MarkNotificationRead(ctx context.Context, ids []int64) error {
	return c.setReadAt(ctx, ids, true)
}

// MarkNotificationUnread 标记消息为未读。
func (c *NotificationCase) MarkNotificationUnread(ctx context.Context, ids []int64) error {
	return c.setReadAt(ctx, ids, false)
}

// MarkAllNotificationRead 标记水位线之前的全部消息为已读。
func (c *NotificationCase) MarkAllNotificationRead(ctx context.Context, beforeDeliveryID int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	err = c.deliveryWriter.SetAllReadAt(ctx, authInfo.UserId, beforeDeliveryID, now)
	if err == nil {
		c.publishChanged(ctx, authInfo.TenantId, authInfo.UserId, 0, "read_all")
	}
	return err
}

// ArchiveNotification 归档消息。
func (c *NotificationCase) ArchiveNotification(ctx context.Context, id int64) error {
	return c.setArchivedAt(ctx, id, true)
}

// RestoreNotification 恢复已归档消息。
func (c *NotificationCase) RestoreNotification(ctx context.Context, id int64) error {
	return c.setArchivedAt(ctx, id, false)
}

// DeleteNotification 从个人收件箱删除消息。
func (c *NotificationCase) DeleteNotification(ctx context.Context, id int64) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var delivery *models.BaseMessageDelivery
	var category *models.BaseMessageCategory
	delivery, category, err = c.deliveryCategory(ctx, authInfo.UserId, id)
	if err != nil {
		return err
	}
	if !category.AllowDelete {
		return errorsx.PermissionDenied("该分类消息不允许删除")
	}
	err = c.DeleteByID(ctx, delivery.ID)
	if err == nil {
		c.publishChanged(ctx, authInfo.TenantId, authInfo.UserId, delivery.ID, "deleted")
	}
	return err
}

// setReadAt 批量设置当前用户消息已读状态。
func (c *NotificationCase) setReadAt(ctx context.Context, ids []int64, read bool) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if err = c.ensureOwnedDeliveries(ctx, authInfo.UserId, ids); err != nil {
		return err
	}
	var readAt *time.Time
	if read {
		now := time.Now()
		readAt = &now
	}
	err = c.deliveryWriter.SetReadAt(ctx, authInfo.UserId, ids, readAt)
	if err == nil {
		c.publishChanged(ctx, authInfo.TenantId, authInfo.UserId, 0, "read")
	}
	return err
}

// setArchivedAt 设置当前用户消息归档状态。
func (c *NotificationCase) setArchivedAt(ctx context.Context, id int64, archived bool) error {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var delivery *models.BaseMessageDelivery
	var category *models.BaseMessageCategory
	delivery, category, err = c.deliveryCategory(ctx, authInfo.UserId, id)
	if err != nil {
		return err
	}
	if !category.AllowArchive {
		return errorsx.PermissionDenied("该分类消息不允许归档")
	}
	var archivedAt *time.Time
	if archived {
		now := time.Now()
		archivedAt = &now
	}
	err = c.deliveryWriter.SetArchivedAt(ctx, authInfo.UserId, delivery.ID, archivedAt)
	if err == nil {
		c.publishChanged(ctx, authInfo.TenantId, authInfo.UserId, delivery.ID, "archived")
	}
	return err
}

// inboxOptions 构造收件箱视图查询条件。
func (c *NotificationCase) inboxOptions(ctx context.Context, userID int64, view basev1.NotificationView) []repository.QueryOption {
	query := c.Query(ctx).BaseMessageDelivery
	opts := c.visibleOptions(ctx, userID)
	switch view {
	case basev1.NotificationView_NOTIFICATION_VIEW_UNREAD:
		opts = append(opts, repository.Where(query.ArchivedAt.Eq(0)), repository.Where(query.ReadAt.Eq(0)))
	case basev1.NotificationView_NOTIFICATION_VIEW_ARCHIVED:
		opts = append(opts, repository.Where(query.ArchivedAt.Neq(0)))
	default:
		opts = append(opts, repository.Where(query.ArchivedAt.Eq(0)))
	}
	return append(opts, repository.Order(query.ID.Desc()))
}

// visibleOptions 构造当前用户可见投递查询条件。
func (c *NotificationCase) visibleOptions(ctx context.Context, userID int64) []repository.QueryOption {
	query := c.Query(ctx).BaseMessageDelivery
	expiresCondition := field.Or(query.ExpiresAt.Eq(0), query.ExpiresAt.Gt(time.Now().UnixMilli()))
	return []repository.QueryOption{
		repository.Where(query.UserID.Eq(userID)),
		repository.Where(query.RevokedAt.Eq(0)),
		repository.Where(expiresCondition),
	}
}

// filterMessageIDs 查询分类或优先级过滤对应的消息ID。
func (c *NotificationCase) filterMessageIDs(ctx context.Context, req *basev1.PageNotificationRequest) ([]int64, error) {
	if req.CategoryId == nil && req.Priority == nil {
		return nil, nil
	}
	query := c.messageRepo.Query(ctx).BaseMessage
	opts := make([]repository.QueryOption, 0, 2)
	if req.CategoryId != nil {
		opts = append(opts, repository.Where(query.CategoryID.Eq(req.GetCategoryId())))
	}
	if req.Priority != nil {
		opts = append(opts, repository.Where(query.Priority.Eq(int32(req.GetPriority()))))
	}
	list, err := c.messageRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(list))
	for _, item := range list {
		ids = append(ids, item.ID)
	}
	return ids, nil
}

// buildNotifications 批量组装收件箱响应。
func (c *NotificationCase) buildNotifications(ctx context.Context, deliveries []*models.BaseMessageDelivery, includeContent bool) ([]*basev1.Notification, error) {
	messageIDs := make([]int64, 0, len(deliveries))
	for _, item := range deliveries {
		messageIDs = append(messageIDs, item.MessageID)
	}
	messages, err := c.messageRepo.ListByIDs(ctx, messageIDs)
	if err != nil {
		return nil, err
	}
	messageMap := make(map[int64]*models.BaseMessage, len(messages))
	categoryIDs := make([]int64, 0, len(messages))
	for _, item := range messages {
		messageMap[item.ID] = item
		categoryIDs = append(categoryIDs, item.CategoryID)
	}
	var categories []*models.BaseMessageCategory
	categories, err = c.categoryRepo.ListByIDs(ctx, categoryIDs)
	if err != nil {
		return nil, err
	}
	categoryMap := make(map[int64]*models.BaseMessageCategory, len(categories))
	for _, item := range categories {
		categoryMap[item.ID] = item
	}
	result := make([]*basev1.Notification, 0, len(deliveries))
	for _, delivery := range deliveries {
		message := messageMap[delivery.MessageID]
		if message == nil {
			continue
		}
		if message.Status == int32(basev1.MessageStatus_MESSAGE_STATUS_REVOKED) {
			continue
		}
		category := categoryMap[message.CategoryID]
		if category == nil {
			continue
		}
		content := ""
		if includeContent {
			content = message.Content
		}
		result = append(result, &basev1.Notification{
			Id: delivery.ID, MessageId: message.ID, CategoryId: category.ID,
			CategoryName: category.Name, CategoryIcon: category.Icon, CategoryColor: category.Color,
			Priority: basev1.MessagePriority(message.Priority), Title: message.Title, Content: content,
			ContentFormat: basev1.MessageContentFormat(message.ContentFormat), ActionType: basev1.MessageActionType(message.ActionType),
			ActionTarget: message.ActionTarget, ActionParams: message.ActionParams, SenderName: message.SenderName,
			ReceivedAt: formatNotificationTime(delivery.ReceivedAt), ReadAt: formatNotificationMillis(delivery.ReadAt),
			ArchivedAt: formatNotificationMillis(delivery.ArchivedAt), ExpiresAt: formatNotificationMillis(delivery.ExpiresAt),
			AllowArchive: category.AllowArchive, AllowDelete: category.AllowDelete,
		})
	}
	return result, nil
}

// findOwnedDelivery 查询当前用户拥有且可见的投递记录。
func (c *NotificationCase) findOwnedDelivery(ctx context.Context, userID, id int64) (*models.BaseMessageDelivery, error) {
	query := c.Query(ctx).BaseMessageDelivery
	opts := c.visibleOptions(ctx, userID)
	opts = append(opts, repository.Where(query.ID.Eq(id)))
	return c.Find(ctx, opts...)
}

// ensureOwnedDeliveries 校验批量投递记录均属于当前用户。
func (c *NotificationCase) ensureOwnedDeliveries(ctx context.Context, userID int64, ids []int64) error {
	query := c.Query(ctx).BaseMessageDelivery
	count, err := c.Count(ctx,
		repository.Where(query.UserID.Eq(userID)),
		repository.Where(query.ID.In(ids...)),
		repository.Where(query.RevokedAt.Eq(0)),
		repository.Where(field.Or(query.ExpiresAt.Eq(0), query.ExpiresAt.Gt(time.Now().UnixMilli()))),
	)
	if err != nil {
		return err
	}
	if count != int64(len(ids)) {
		return errorsx.ResourceNotFound("消息不存在")
	}
	return nil
}

// deliveryCategory 查询当前用户投递记录及其分类策略。
func (c *NotificationCase) deliveryCategory(ctx context.Context, userID, id int64) (*models.BaseMessageDelivery, *models.BaseMessageCategory, error) {
	delivery, err := c.findOwnedDelivery(ctx, userID, id)
	if err != nil {
		return nil, nil, err
	}
	var message *models.BaseMessage
	message, err = c.messageRepo.FindByID(ctx, delivery.MessageID)
	if err != nil {
		return nil, nil, err
	}
	var category *models.BaseMessageCategory
	category, err = c.categoryRepo.FindByID(ctx, message.CategoryID)
	if err != nil {
		return nil, nil, err
	}
	return delivery, category, nil
}

// categoryUnread 分批汇总当前用户未读投递对应的消息分类数量。
func (c *NotificationCase) categoryUnread(ctx context.Context, baseOpts []repository.QueryOption) ([]*basev1.NotificationCategoryUnread, error) {
	query := c.Query(ctx).BaseMessageDelivery
	counts := make(map[int64]int64)
	cursorID := int64(0)
	var err error
	for {
		opts := append([]repository.QueryOption{}, baseOpts...)
		if cursorID > 0 {
			opts = append(opts, repository.Where(query.ID.Lt(cursorID)))
		}
		opts = append(opts, repository.Order(query.ID.Desc()), repository.Limit(notificationSummaryBatchSize))
		var deliveries []*models.BaseMessageDelivery
		deliveries, err = c.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		if len(deliveries) == 0 {
			break
		}
		messageIDs := make([]int64, 0, len(deliveries))
		for _, delivery := range deliveries {
			messageIDs = append(messageIDs, delivery.MessageID)
		}
		messageQuery := c.messageRepo.Query(ctx).BaseMessage
		messageOpts := []repository.QueryOption{repository.Where(messageQuery.ID.In(messageIDs...))}
		var messages []*models.BaseMessage
		messages, err = c.messageRepo.List(ctx, messageOpts...)
		if err != nil {
			return nil, err
		}
		categoryByMessage := make(map[int64]int64, len(messages))
		for _, message := range messages {
			if message.Status == int32(basev1.MessageStatus_MESSAGE_STATUS_REVOKED) {
				continue
			}
			categoryByMessage[message.ID] = message.CategoryID
		}
		for _, delivery := range deliveries {
			if categoryID, ok := categoryByMessage[delivery.MessageID]; ok {
				counts[categoryID]++
			}
		}
		if len(deliveries) < notificationSummaryBatchSize {
			break
		}
		cursorID = deliveries[len(deliveries)-1].ID
	}
	categoryIDs := make([]int64, 0, len(counts))
	for categoryID := range counts {
		categoryIDs = append(categoryIDs, categoryID)
	}
	sort.Slice(categoryIDs, func(i, j int) bool { return categoryIDs[i] < categoryIDs[j] })
	result := make([]*basev1.NotificationCategoryUnread, 0, len(categoryIDs))
	for _, categoryID := range categoryIDs {
		result = append(result, &basev1.NotificationCategoryUnread{CategoryId: categoryID, UnreadTotal: counts[categoryID]})
	}
	return result, nil
}

// publishChanged 尽力发布用户收件箱变化事件。
func (c *NotificationCase) publishChanged(ctx context.Context, tenantID, userID, deliveryID int64, changeType string) {
	if c.sse == nil {
		return
	}
	streamID := fmt.Sprintf("%s:%d:%d", notificationStreamID, tenantID, userID)
	c.sse.PublishJSON(ctx, streamID, notificationChangedEvent, map[string]any{"change_type": changeType, "delivery_id": deliveryID})
}

// formatNotificationTime 格式化可空数据库时间。
func formatNotificationTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return _time.TimeToTimeString(value)
}

// formatNotificationMillis 格式化毫秒时间戳。
func formatNotificationMillis(value int64) string {
	if value <= 0 {
		return ""
	}
	return _time.TimeToTimeString(time.UnixMilli(value))
}
