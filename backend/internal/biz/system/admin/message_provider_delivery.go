package biz

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/go-utils/id"
	"github.com/liujitcn/gorm-kit/repository"
	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/notify"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	messageProviderDeliveryBatchSize = 100
)

// createProviderDeliveryRecords 按分类当前绑定的 Provider 构造幂等投递记录。
func (c *BaseMessageCase) createProviderDeliveryRecords(ctx context.Context, dispatch *models.BaseMessageDispatch, message *models.BaseMessage, providerIDs []int64, users []*models.BaseUser) ([]*models.BaseMessageDelivery, error) {
	if len(providerIDs) == 0 {
		return nil, nil
	}
	var providers []*models.BaseMessageProvider
	var err error
	providers, err = c.categoryCase.providerRepo.ListByIDs(ctx, providerIDs)
	if err != nil {
		return nil, err
	}
	if len(providers) != len(providerIDs) {
		return nil, errorsx.ResourceNotFound("消息分类绑定的 Provider 不存在")
	}
	providerByID := make(map[int64]*models.BaseMessageProvider, len(providers))
	for _, provider := range providers {
		providerByID[provider.ID] = provider
	}
	templateQuery := c.categoryCase.templateRepo.Query(ctx).BaseMessageTemplate
	var templates []*models.BaseMessageTemplate
	templates, err = c.categoryCase.templateRepo.List(ctx,
		repository.Where(templateQuery.CategoryID.Eq(message.CategoryID)),
		repository.Where(templateQuery.ProviderID.In(providerIDs...)),
	)
	if err != nil {
		return nil, err
	}
	templateByProvider := make(map[int64]*models.BaseMessageTemplate, len(templates))
	for _, messageTemplate := range templates {
		templateByProvider[messageTemplate.ProviderID] = messageTemplate
	}
	now := time.Now()
	records := make([]*models.BaseMessageDelivery, 0, len(providerIDs)*len(users))
	for _, providerID := range providerIDs {
		provider := providerByID[providerID]
		if provider == nil {
			return nil, errorsx.ResourceNotFound("消息分类绑定的 Provider 不存在")
		}
		if provider.Status == int32(commonv1.Status_STATUS_ENABLE) && templateByProvider[providerID] == nil {
			return nil, errorsx.ResourceNotFound("消息分类 Provider 模板不存在")
		}
		status := _const.MessageDeliveryStatusPending
		lastError := ""
		if provider.Status != int32(commonv1.Status_STATUS_ENABLE) {
			status, lastError = _const.MessageDeliveryStatusSkipped, "Provider disabled"
		}
		if isBroadcastMessageProvider(provider) {
			if dispatch.CursorUserID == 0 {
				records = append(records, newProviderDeliveryRecord(message, provider.ID, 0, status, lastError, now))
			}
			continue
		}
		for _, user := range users {
			records = append(records, newProviderDeliveryRecord(message, provider.ID, user.ID, status, lastError, now))
		}
	}
	return records, nil
}

// processPendingProviderDeliveries 认领并处理到期的 Provider 投递记录。
func (c *BaseMessageCase) processPendingProviderDeliveries(ctx context.Context, messageID int64) (int, error) {
	now := time.Now()
	query := c.deliveryRepo.Query(ctx).BaseMessageDelivery
	retryReady := field.Or(query.NextRetryAt.Eq(0), query.NextRetryAt.Lte(now.UnixMilli()))
	claimable := field.Or(
		field.And(query.Status.Eq(_const.MessageDeliveryStatusPending), retryReady),
		field.And(query.Status.Eq(_const.MessageDeliveryStatusRunning), query.LockedUntil.Lte(now.UnixMilli())),
	)
	opts := []repository.QueryOption{
		repository.Where(query.DeliveryType.Eq(_const.MessageDeliveryTypeProvider)),
		repository.Where(claimable),
		repository.Order(query.ID.Asc()),
		repository.Limit(messageProviderDeliveryBatchSize),
	}
	if messageID > 0 {
		opts = append(opts, repository.Where(query.MessageID.Eq(messageID)))
	}
	records, err := c.deliveryRepo.List(ctx, opts...)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, record := range records {
		if record.Status == _const.MessageDeliveryStatusRunning && record.AttemptCount >= messageDispatchMaxAttempts {
			if err = c.finishProviderDelivery(ctx, record, _const.MessageDeliveryStatusFailed, "Provider send interrupted", "", 0); err != nil {
				return processed, err
			}
			processed++
			continue
		}
		var claimed bool
		claimed, err = c.claimProviderDelivery(ctx, record, time.Now())
		if err != nil {
			return processed, err
		}
		if !claimed {
			continue
		}
		var receipt *notify.Receipt
		var skipped bool
		receipt, skipped, err = c.sendProviderDelivery(ctx, record)
		status, lastError, nextRetryAt := _const.MessageDeliveryStatusSucceeded, "", int64(0)
		if skipped {
			status, lastError = _const.MessageDeliveryStatusSkipped, "Recipient unavailable"
		} else if err != nil {
			log.Error(fmt.Sprintf("provider delivery failed: message=%d provider=%d user=%d: %v", record.MessageID, record.ProviderID, record.UserID, err))
			lastError = "Provider send failed"
			if record.AttemptCount < messageDispatchMaxAttempts {
				status = _const.MessageDeliveryStatusPending
				nextRetryAt = time.Now().Add(messageDispatchRetryDelay(record.AttemptCount)).UnixMilli()
			} else {
				status = _const.MessageDeliveryStatusFailed
			}
		}
		receiptID := ""
		if receipt != nil {
			receiptID = receipt.MessageID
		}
		if err = c.finishProviderDelivery(ctx, record, status, lastError, receiptID, nextRetryAt); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

// sendProviderDelivery 按投递记录读取 Provider 当前配置和分类模板并执行发送。
func (c *BaseMessageCase) sendProviderDelivery(ctx context.Context, delivery *models.BaseMessageDelivery) (*notify.Receipt, bool, error) {
	message, err := c.FindByID(ctx, delivery.MessageID)
	if err != nil {
		return nil, false, err
	}
	if message.Status == int32(basev1.MessageStatus_MESSAGE_STATUS_REVOKED) {
		return nil, true, nil
	}
	providerQuery := c.categoryCase.providerRepo.Query(ctx).BaseMessageProvider
	provider, err := c.categoryCase.providerRepo.Find(ctx, repository.Where(providerQuery.ID.Eq(delivery.ProviderID)))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, true, nil
	}
	if err != nil {
		return nil, false, err
	}
	if provider.Status != int32(commonv1.Status_STATUS_ENABLE) {
		return nil, true, nil
	}
	templateQuery := c.categoryCase.templateRepo.Query(ctx).BaseMessageTemplate
	messageTemplate, err := c.categoryCase.templateRepo.Find(ctx,
		repository.Where(templateQuery.CategoryID.Eq(message.CategoryID)),
		repository.Where(templateQuery.ProviderID.Eq(delivery.ProviderID)),
	)
	if err != nil {
		return nil, false, err
	}
	var user *models.BaseUser
	if delivery.UserID > 0 {
		userQuery := c.baseUserRepo.Query(ctx).BaseUser
		user, err = c.baseUserRepo.Find(ctx,
			repository.Where(userQuery.ID.Eq(delivery.UserID)),
			repository.Where(userQuery.TenantID.Eq(delivery.TenantID)),
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, true, nil
		}
		if err != nil {
			return nil, false, err
		}
	}
	var recipient notify.Recipient
	var available bool
	recipient, available, err = c.messageProviderRecipient(ctx, provider, user)
	if err != nil {
		return nil, false, err
	}
	if !available {
		return nil, true, nil
	}
	var payload notify.Message
	payload, err = buildMessageProviderPayload(message, messageTemplate, provider, user, recipient)
	if err != nil {
		return nil, false, err
	}
	var receipt *notify.Receipt
	receipt, err = c.providerManager.Send(ctx, provider, payload)
	return receipt, false, err
}

// claimProviderDelivery 使用租约原子认领单条 Provider 投递。
func (c *BaseMessageCase) claimProviderDelivery(ctx context.Context, delivery *models.BaseMessageDelivery, now time.Time) (bool, error) {
	query := c.deliveryRepo.Query(ctx).BaseMessageDelivery
	retryReady := field.Or(query.NextRetryAt.Eq(0), query.NextRetryAt.Lte(now.UnixMilli()))
	claimable := field.Or(
		field.And(query.Status.Eq(_const.MessageDeliveryStatusPending), retryReady),
		field.And(query.Status.Eq(_const.MessageDeliveryStatusRunning), query.LockedUntil.Lte(now.UnixMilli())),
	)
	token := id.NewGUIDv4NoHyphen()
	result, err := query.WithContext(ctx).
		Where(
			query.TenantID.Eq(delivery.TenantID),
			query.MessageID.Eq(delivery.MessageID),
			query.DeliveryType.Eq(_const.MessageDeliveryTypeProvider),
			query.ProviderID.Eq(delivery.ProviderID),
			query.UserID.Eq(delivery.UserID),
			claimable,
			query.AttemptCount.Lt(messageDispatchMaxAttempts),
		).
		UpdateSimple(
			query.Status.Value(_const.MessageDeliveryStatusRunning),
			query.AttemptCount.Add(1),
			query.LockToken.Value(token),
			query.LockedUntil.Value(now.Add(messageDispatchLease).UnixMilli()),
			query.UpdatedAt.Value(now),
		)
	if err != nil {
		return false, err
	}
	if result.RowsAffected == 0 {
		return false, nil
	}
	delivery.Status = _const.MessageDeliveryStatusRunning
	delivery.AttemptCount++
	delivery.LockToken = token
	delivery.LockedUntil = now.Add(messageDispatchLease).UnixMilli()
	return true, nil
}

// finishProviderDelivery 保存当前租约的 Provider 结果并同步已发布消息统计。
func (c *BaseMessageCase) finishProviderDelivery(ctx context.Context, delivery *models.BaseMessageDelivery, status int32, lastError, receiptID string, nextRetryAt int64) error {
	return c.tx.Transaction(ctx, func(txCtx context.Context) error {
		messageQuery := c.Query(txCtx).BaseMessage
		message, err := c.Find(txCtx,
			repository.Where(messageQuery.ID.Eq(delivery.MessageID)),
			repository.Clauses(clause.Locking{Strength: "UPDATE"}),
		)
		if err != nil {
			return err
		}
		query := c.deliveryRepo.Query(txCtx).BaseMessageDelivery
		var result gen.ResultInfo
		result, err = query.WithContext(txCtx).
			Where(
				query.TenantID.Eq(delivery.TenantID),
				query.MessageID.Eq(delivery.MessageID),
				query.DeliveryType.Eq(_const.MessageDeliveryTypeProvider),
				query.ProviderID.Eq(delivery.ProviderID),
				query.UserID.Eq(delivery.UserID),
				query.Status.Eq(_const.MessageDeliveryStatusRunning),
				query.LockToken.Eq(delivery.LockToken),
			).
			UpdateSimple(
				query.Status.Value(status),
				query.NextRetryAt.Value(nextRetryAt),
				query.ReceiptID.Value(receiptID),
				query.LastError.Value(lastError),
				query.LockToken.Value(""),
				query.LockedUntil.Value(0),
				query.UpdatedAt.Value(time.Now()),
			)
		if err != nil || result.RowsAffected == 0 || message.Status != int32(basev1.MessageStatus_MESSAGE_STATUS_PUBLISHED) {
			return err
		}
		if status == _const.MessageDeliveryStatusSucceeded {
			_, err = messageQuery.WithContext(txCtx).Where(messageQuery.ID.Eq(message.ID)).UpdateSimple(
				messageQuery.DeliveredTotal.Add(1), messageQuery.UpdatedAt.Value(time.Now()),
			)
		} else if status == _const.MessageDeliveryStatusFailed {
			_, err = messageQuery.WithContext(txCtx).Where(messageQuery.ID.Eq(message.ID)).UpdateSimple(
				messageQuery.FailedTotal.Add(1), messageQuery.UpdatedAt.Value(time.Now()),
			)
		}
		return err
	})
}

// messageProviderRecipient 解析当前 Provider 所需的收件人标识。
func (c *BaseMessageCase) messageProviderRecipient(ctx context.Context, provider *models.BaseMessageProvider, user *models.BaseUser) (notify.Recipient, bool, error) {
	var err error
	if isBroadcastMessageProvider(provider) {
		return notify.Recipient{}, true, nil
	}
	if user == nil {
		return notify.Recipient{}, false, nil
	}
	switch strings.ToLower(provider.Provider) {
	case "email":
		return notify.Recipient{Email: user.Email}, user.Email != "", nil
	case "sms":
		return notify.Recipient{Phone: user.Phone}, user.Phone != "", nil
	case "wechat", "dingtalk", "feishu", "wechatwork":
		accountQuery := c.thirdAccountRepo.Query(ctx).BaseThirdAccount
		var account *models.BaseThirdAccount
		account, err = c.thirdAccountRepo.Find(ctx,
			repository.Where(accountQuery.TenantID.Eq(user.TenantID)),
			repository.Where(accountQuery.UserID.Eq(user.ID)),
			repository.Where(accountQuery.Provider.Eq(strings.ToLower(provider.Provider))),
		)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return notify.Recipient{}, false, nil
		}
		if err != nil {
			return notify.Recipient{}, false, err
		}
		recipient := notify.Recipient{}
		switch strings.ToLower(provider.Provider) {
		case "wechat":
			recipient.OpenID = account.Identifier
		case "dingtalk":
			if account.UnionID == "" {
				return notify.Recipient{}, false, nil
			}
			recipient.UnionID = account.UnionID
		case "wechatwork":
			recipient.UserID = account.Identifier
		case "feishu":
			config, configErr := decodeMessageJSON(provider.Config)
			if configErr != nil {
				return notify.Recipient{}, false, configErr
			}
			receiveIDType, _ := config.AsMap()["receive_id_type"].(string)
			switch receiveIDType {
			case "union_id":
				if account.UnionID == "" {
					return notify.Recipient{}, false, nil
				}
				recipient.UnionID = account.UnionID
			case "user_id", "chat_id":
				return notify.Recipient{}, false, nil
			case "email":
				recipient.Email = user.Email
				if user.Email == "" {
					return notify.Recipient{}, false, nil
				}
			default:
				recipient.OpenID = account.Identifier
			}
		}
		return recipient, true, nil
	default:
		return notify.Recipient{}, false, fmt.Errorf("unsupported message Provider %q", provider.Provider)
	}
}

// buildMessageProviderPayload 根据分类模板构造Provider发送内容。
func buildMessageProviderPayload(message *models.BaseMessage, messageTemplate *models.BaseMessageTemplate, provider *models.BaseMessageProvider, user *models.BaseUser, recipient notify.Recipient) (notify.Message, error) {
	config, err := decodeMessageJSON(messageTemplate.Config)
	if err != nil {
		return notify.Message{}, fmt.Errorf("decode message template config: %w", err)
	}
	values := map[string]string{
		"Title": message.Title, "Content": message.Content, "SenderName": message.SenderName,
	}
	if user != nil {
		values["UserName"], values["NickName"], values["UserCode"] = user.UserName, user.NickName, user.UserCode
		values["Email"], values["Phone"] = user.Email, user.Phone
	}
	title := messageTemplate.Title
	if title == "" {
		title = message.Title
	}
	content := messageTemplate.Content
	if content == "" {
		content = message.Content
	}
	title, err = renderMessageTemplateText(title, values)
	if err != nil {
		return notify.Message{}, err
	}
	content, err = renderMessageTemplateText(content, values)
	if err != nil {
		return notify.Message{}, err
	}
	contentFormat := basev1.MessageContentFormat(messageTemplate.ContentFormat)
	if contentFormat == basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_UNSPECIFIED {
		contentFormat = basev1.MessageContentFormat(message.ContentFormat)
	}
	switch contentFormat {
	case basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_SAFE_MARKDOWN:
		content = messageMarkdownPolicy.Sanitize(content)
	case basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_RICH_TEXT:
		content = messageRichTextPolicy.Sanitize(content)
	}
	contentType := notify.PlainText
	switch contentFormat {
	case basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_SAFE_MARKDOWN:
		contentType = notify.Markdown
	case basev1.MessageContentFormat_MESSAGE_CONTENT_FORMAT_RICH_TEXT:
		contentType = notify.HTML
	}
	result := notify.Message{
		Title: title, Content: content, ContentType: contentType,
		Metadata: map[string]string{"message_id": fmt.Sprint(message.ID), "category_id": fmt.Sprint(message.CategoryID), "provider_id": fmt.Sprint(provider.ID)},
	}
	if recipient.Email != "" || recipient.Phone != "" || recipient.OpenID != "" || recipient.UnionID != "" || recipient.UserID != "" || recipient.ChatID != "" {
		result.Recipients = []notify.Recipient{recipient}
	}
	settings := config.AsMap()
	code, _ := settings["template_code"].(string)
	params := make(map[string]string)
	if rawParams, ok := settings["params"].(map[string]any); ok {
		for key, value := range rawParams {
			params[key], err = renderMessageTemplateText(fmt.Sprint(value), values)
			if err != nil {
				return notify.Message{}, err
			}
		}
	}
	var orderedParams []string
	if rawParams, ok := settings["ordered_params"].([]any); ok {
		orderedParams = make([]string, 0, len(rawParams))
		for _, value := range rawParams {
			var rendered string
			rendered, err = renderMessageTemplateText(fmt.Sprint(value), values)
			if err != nil {
				return notify.Message{}, err
			}
			orderedParams = append(orderedParams, rendered)
		}
	}
	urlValue, _ := settings["url"].(string)
	if urlValue != "" {
		urlValue, err = renderMessageTemplateText(urlValue, values)
		if err != nil {
			return notify.Message{}, err
		}
	}
	if code != "" || len(params) > 0 || len(orderedParams) > 0 || urlValue != "" {
		result.Template = &notify.Template{Code: code, Params: params, OrderedParams: orderedParams, URL: urlValue}
	}
	return result, nil
}

// isBroadcastMessageProvider 判断 Provider 是否不需要逐用户收件人。
func isBroadcastMessageProvider(provider *models.BaseMessageProvider) bool {
	if strings.EqualFold(provider.Provider, "webhook") {
		return true
	}
	config, err := decodeMessageJSON(provider.Config)
	if err != nil {
		return false
	}
	implementation, _ := config.AsMap()["implementation"].(string)
	return strings.EqualFold(implementation, "group")
}

// newProviderDeliveryRecord 构造一条带有初始状态的Provider投递记录。
func newProviderDeliveryRecord(message *models.BaseMessage, providerID, userID int64, status int32, lastError string, now time.Time) *models.BaseMessageDelivery {
	return &models.BaseMessageDelivery{
		TenantID: message.TenantID, MessageID: message.ID,
		DeliveryType: _const.MessageDeliveryTypeProvider, ProviderID: providerID, UserID: userID,
		Status: status, LastError: lastError, ReceivedAt: now, CreatedAt: now, UpdatedAt: now,
	}
}

// renderMessageTemplateText 将消息模板文本按收件人上下文渲染。
func renderMessageTemplateText(raw string, values map[string]string) (string, error) {
	parsed, err := template.New("message-provider").Option("missingkey=zero").Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse message template: %w", err)
	}
	var output bytes.Buffer
	if err = parsed.Execute(&output, values); err != nil {
		return "", fmt.Errorf("render message template: %w", err)
	}
	return output.String(), nil
}
