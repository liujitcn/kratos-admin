package biz

import (
	"context"
	"encoding/json"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/notification"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"google.golang.org/protobuf/types/known/structpb"
)

// BaseMessageProviderCase 管理消息发送 Provider。
type BaseMessageProviderCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseMessageProviderRepository
	categoryRepo *data.BaseMessageCategoryRepository
	templateRepo *data.BaseMessageTemplateRepository
}

// NewBaseMessageProviderCase 创建消息发送 Provider 业务实例。
func NewBaseMessageProviderCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	providerRepo *data.BaseMessageProviderRepository,
	categoryRepo *data.BaseMessageCategoryRepository,
	templateRepo *data.BaseMessageTemplateRepository,
) *BaseMessageProviderCase {
	return &BaseMessageProviderCase{
		BaseCase: baseCase, tx: tx, BaseMessageProviderRepository: providerRepo,
		categoryRepo: categoryRepo, templateRepo: templateRepo,
	}
}

// OptionBaseMessageProvider 查询消息发送 Provider 选项。
func (c *BaseMessageProviderCase) OptionBaseMessageProvider(ctx context.Context, _ *adminv1.OptionBaseMessageProviderRequest) (*commonv1.SelectOptionResponse, error) {
	query := c.Query(ctx).BaseMessageProvider
	list, err := c.List(ctx, repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc()))
	if err != nil {
		return nil, err
	}
	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		// Provider 停用只影响运行时发送，不阻止分类提前绑定待配置的 Provider。
		options = append(options, &commonv1.SelectOptionResponse_Option{Label: item.Name, Value: item.ID})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// PageBaseMessageProvider 分页查询消息发送 Provider。
func (c *BaseMessageProviderCase) PageBaseMessageProvider(ctx context.Context, req *adminv1.PageBaseMessageProviderRequest) (*adminv1.PageBaseMessageProviderResponse, error) {
	query := c.Query(ctx).BaseMessageProvider
	opts := make([]repository.QueryOption, 0, 5)
	opts = append(opts, repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc()))
	if req.GetProvider() != "" {
		opts = append(opts, repository.Where(query.Provider.Like("%"+req.GetProvider()+"%")))
	}
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseMessageProvider, 0, len(list))
	for _, item := range list {
		value, decodeErr := c.toDTO(item)
		if decodeErr != nil {
			return nil, decodeErr
		}
		result = append(result, value)
	}
	return &adminv1.PageBaseMessageProviderResponse{BaseMessageProviders: result, Total: int32(total)}, nil
}

// GetBaseMessageProvider 查询消息发送 Provider 详情。
func (c *BaseMessageProviderCase) GetBaseMessageProvider(ctx context.Context, id int64) (*adminv1.BaseMessageProviderForm, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	config, err := decodeMessageJSON(item.Config)
	if err != nil {
		return nil, errorsx.Internal("解析消息 Provider 配置失败").WithCause(err)
	}
	return &adminv1.BaseMessageProviderForm{
		Id: item.ID, Provider: item.Provider, Name: item.Name, Description: item.Description, Icon: item.Icon,
		ClientId: item.ClientID, SecretConfigured: item.ClientSecret != "", Config: config,
		Sort: item.Sort, Status: commonv1.Status(item.Status),
	}, nil
}

// CreateBaseMessageProvider 创建消息发送 Provider。
func (c *BaseMessageProviderCase) CreateBaseMessageProvider(ctx context.Context, req *adminv1.BaseMessageProviderForm) error {
	item, err := c.formEntity(req, nil)
	if err != nil {
		return err
	}
	if err = c.validateEnabledProvider(item); err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	item.CreatedBy, item.UpdatedBy, item.CreatedAt, item.UpdatedAt = authInfo.UserId, authInfo.UserId, now, now
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.Create(txCtx, item); err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("消息 Provider 标识重复", "base_message_provider", "provider", "unique_base_message_provider").WithCause(err)
			}
			return err
		}
		return nil
	})
	return err
}

// UpdateBaseMessageProvider 更新消息发送 Provider。
func (c *BaseMessageProviderCase) UpdateBaseMessageProvider(ctx context.Context, req *adminv1.BaseMessageProviderForm) error {
	oldItem, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if req.GetProvider() != oldItem.Provider {
		return errorsx.Conflict("Provider 标识不可修改")
	}
	item, err := c.formEntity(req, oldItem)
	if err != nil {
		return err
	}
	if err = c.validateEnabledProvider(item); err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	item.ID, item.CreatedBy, item.CreatedAt = oldItem.ID, oldItem.CreatedBy, oldItem.CreatedAt
	item.UpdatedBy, item.UpdatedAt = authInfo.UserId, time.Now()
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.UpdateByID(txCtx, item); err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("消息 Provider 标识重复", "base_message_provider", "provider", "unique_base_message_provider").WithCause(err)
			}
			return err
		}
		return nil
	})
	return err
}

// DeleteBaseMessageProvider 删除未被消息分类或模板使用的 Provider。
func (c *BaseMessageProviderCase) DeleteBaseMessageProvider(ctx context.Context, ids string) error {
	idValues := _string.ConvertStringToInt64Array(ids)
	if len(idValues) == 0 {
		return errorsx.InvalidArgument("消息 Provider ID不能为空")
	}
	list, err := c.ListByIDs(ctx, idValues)
	if err != nil {
		return err
	}
	if len(list) != len(idValues) {
		return errorsx.ResourceNotFound("消息 Provider 不存在")
	}
	templateQuery := c.templateRepo.Query(ctx).BaseMessageTemplate
	templateCount, err := c.templateRepo.Count(ctx, repository.Where(templateQuery.ProviderID.In(idValues...)))
	if err != nil {
		return err
	}
	if templateCount > 0 {
		return errorsx.Conflict("消息 Provider 已被消息模板使用")
	}
	categories, err := c.categoryRepo.List(ctx)
	if err != nil {
		return err
	}
	for _, category := range categories {
		providerIDs, decodeErr := decodeMessageProviderIDs(category.ProviderID)
		if decodeErr != nil {
			return decodeErr
		}
		if hasAnyInt64(providerIDs, idValues) {
			return errorsx.Conflict("消息 Provider 已被消息分类绑定")
		}
	}
	return c.tx.Transaction(ctx, func(txCtx context.Context) error { return c.DeleteByIDs(txCtx, idValues) })
}

// SetBaseMessageProviderStatus 设置消息发送 Provider 状态。
func (c *BaseMessageProviderCase) SetBaseMessageProviderStatus(ctx context.Context, req *adminv1.SetBaseMessageProviderStatusRequest) error {
	if req.GetStatus() != commonv1.Status_STATUS_ENABLE && req.GetStatus() != commonv1.Status_STATUS_DISABLE {
		return errorsx.InvalidArgument("消息 Provider 状态无效")
	}
	item, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if item.Status == int32(req.GetStatus()) {
		return nil
	}
	item.Status = int32(req.GetStatus())
	if err = c.validateEnabledProvider(item); err != nil {
		return err
	}
	query := c.Query(ctx).BaseMessageProvider
	_, err = query.WithContext(ctx).Where(query.ID.Eq(item.ID)).UpdateSimple(query.Status.Value(int32(req.GetStatus())), query.UpdatedAt.Value(time.Now()))
	return err
}

// formEntity 将消息 Provider 表单转换为持久化实体。
func (c *BaseMessageProviderCase) formEntity(req *adminv1.BaseMessageProviderForm, oldItem *models.BaseMessageProvider) (*models.BaseMessageProvider, error) {
	config, err := encodeMessageJSON(req.GetConfig())
	if err != nil {
		return nil, errorsx.InvalidArgument("消息 Provider 扩展配置无效").WithCause(err)
	}
	var secret string
	if req.GetClientSecret() != nil {
		secret = req.GetClientSecret().GetText()
		if len(secret) > maxSecretLength {
			return nil, errorsx.InvalidArgument("通用凭证密钥不能超过1024个字符")
		}
	}
	if secret == "" && oldItem != nil {
		secret = oldItem.ClientSecret
	}
	return &models.BaseMessageProvider{
		Provider: req.GetProvider(), Name: req.GetName(), Description: req.GetDescription(), Icon: req.GetIcon(),
		ClientID: req.GetClientId(), ClientSecret: secret, Config: config, Sort: req.GetSort(), Status: int32(req.GetStatus()),
	}, nil
}

// validateEnabledProvider 校验启用状态下的发送器配置。
func (c *BaseMessageProviderCase) validateEnabledProvider(item *models.BaseMessageProvider) error {
	if item.Status != int32(commonv1.Status_STATUS_ENABLE) && item.Status != int32(commonv1.Status_STATUS_DISABLE) {
		return errorsx.InvalidArgument("消息 Provider 状态无效")
	}
	if item.Status != int32(commonv1.Status_STATUS_ENABLE) {
		return nil
	}
	err := notification.ValidateProviderConfig(item, c.Cache)
	if err != nil {
		return errorsx.InvalidArgument("消息 Provider 配置无效").WithCause(err)
	}
	return nil
}

// toDTO 将消息 Provider 实体转换为列表响应。
func (c *BaseMessageProviderCase) toDTO(item *models.BaseMessageProvider) (*adminv1.BaseMessageProvider, error) {
	config, err := decodeMessageJSON(item.Config)
	if err != nil {
		return nil, errorsx.Internal("解析消息 Provider 配置失败").WithCause(err)
	}
	return &adminv1.BaseMessageProvider{
		Id: item.ID, Provider: item.Provider, Name: item.Name, Description: item.Description, Icon: item.Icon,
		ClientId: item.ClientID, Config: config, Sort: item.Sort, Status: commonv1.Status(item.Status),
		SecretConfigured: item.ClientSecret != "", CreatedAt: item.CreatedAt.Format(time.DateTime), UpdatedAt: item.UpdatedAt.Format(time.DateTime),
	}, nil
}

// decodeMessageJSON 将数据库 JSON 对象转换为 Proto Struct。
func decodeMessageJSON(raw string) (*structpb.Struct, error) {
	values := make(map[string]any)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &values); err != nil {
			return nil, err
		}
	}
	return structpb.NewStruct(values)
}

// encodeMessageJSON 将 Proto Struct 转换为数据库 JSON 对象。
func encodeMessageJSON(value *structpb.Struct) (string, error) {
	if value == nil {
		return "{}", nil
	}
	encoded, err := json.Marshal(value.AsMap())
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// decodeMessageProviderIDs 解析消息分类绑定的 Provider ID JSON数组。
func decodeMessageProviderIDs(raw string) ([]int64, error) {
	if raw == "" {
		return []int64{}, nil
	}
	var ids []int64
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, errorsx.Internal("解析消息 Provider ID失败").WithCause(err)
	}
	return ids, nil
}

// hasAnyInt64 判断两个 ID 列表是否存在交集。
func hasAnyInt64(values, targets []int64) bool {
	set := make(map[int64]struct{}, len(targets))
	for _, value := range targets {
		set[value] = struct{}{}
	}
	for _, value := range values {
		if _, ok := set[value]; ok {
			return true
		}
	}
	return false
}
