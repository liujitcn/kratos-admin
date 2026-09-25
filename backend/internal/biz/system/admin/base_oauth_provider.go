package biz

import (
	"context"
	"encoding/json"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/oauth"
	"github.com/liujitcn/kratos-kit/oauth/provider"
	"google.golang.org/protobuf/types/known/structpb"
	"gorm.io/gen/field"
)

// BaseOauthProviderCase 管理平台级 OAuth 第三方登录方式。
type BaseOauthProviderCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseOauthProviderRepository
	baseI18nCase *BaseI18nCase
	manager      *oauth.Manager
}

// NewBaseOauthProviderCase 创建 OAuth 第三方登录方式业务实例。
func NewBaseOauthProviderCase(baseCase *biz.BaseCase, tx data.Transaction, repo *data.BaseOauthProviderRepository, baseI18nCase *BaseI18nCase, manager *oauth.Manager) *BaseOauthProviderCase {
	return &BaseOauthProviderCase{BaseCase: baseCase, tx: tx, BaseOauthProviderRepository: repo, baseI18nCase: baseI18nCase, manager: manager}
}

// RefreshBaseOauthProvider 从数据库重建运行时 OAuth Provider 快照。
func (c *BaseOauthProviderCase) RefreshBaseOauthProvider(ctx context.Context) error {
	query := c.Query(ctx).BaseOauthProvider
	list, err := c.List(ctx, repository.Where(query.Status.Eq(int32(commonv1.Status_STATUS_ENABLE))), repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc()))
	if err != nil {
		return err
	}
	configs := make(map[oauth.Type]*provider.Config, len(list))
	for _, item := range list {
		var config *provider.Config
		config, err = c.runtimeConfig(item)
		if err != nil {
			return errorsx.Internal("加载OAuth登录方式配置失败").WithCause(err)
		}
		configs[oauth.Type(item.Provider)] = config
	}
	return c.manager.Replace(configs)
}

// PageBaseOauthProvider 分页查询 OAuth 登录方式。
func (c *BaseOauthProviderCase) PageBaseOauthProvider(ctx context.Context, req *adminv1.PageBaseOauthProviderRequest) (*adminv1.PageBaseOauthProviderResponse, error) {
	query := c.Query(ctx).BaseOauthProvider
	opts := []repository.QueryOption{repository.Order(query.Sort.Asc()), repository.Order(query.ID.Asc())}
	if req.GetProvider() != "" {
		opts = append(opts, repository.Where(query.Provider.Like("%"+req.GetProvider()+"%")))
	}
	if req.GetName() != "" {
		var translatedIDs []int64
		var err error
		translatedIDs, err = c.baseI18nCase.GetTargetIdsByName(ctx, _const.I18N_TARGET_KEY_BASE_OAUTH_PROVIDER_NAME, req.GetName())
		if err != nil {
			return nil, err
		}
		condition := query.Name.Like("%" + req.GetName() + "%")
		if len(translatedIDs) > 0 {
			opts = append(opts, repository.Where(field.Or(condition, query.ID.In(translatedIDs...))))
		} else {
			opts = append(opts, repository.Where(condition))
		}
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseOauthProvider, 0, len(list))
	for _, item := range list {
		var value *adminv1.BaseOauthProvider
		value, err = c.toDTO(item)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return &adminv1.PageBaseOauthProviderResponse{BaseOauthProviders: result, Total: int32(total)}, nil
}

// GetBaseOauthProvider 查询 OAuth 登录方式详情。
func (c *BaseOauthProviderCase) GetBaseOauthProvider(ctx context.Context, idValue int64) (*adminv1.BaseOauthProviderForm, error) {
	item, err := c.FindByID(ctx, idValue)
	if err != nil {
		return nil, err
	}
	config, scopes, err := decodeOauthProviderJSON(item)
	if err != nil {
		return nil, err
	}
	nameI18ns, err := c.baseI18nCase.GetBaseI18nMapByTargetKey(ctx, _const.I18N_TARGET_KEY_BASE_OAUTH_PROVIDER_NAME, []int64{idValue})
	if err != nil {
		return nil, err
	}
	descriptionI18ns, err := c.baseI18nCase.GetBaseI18nMapByTargetKey(ctx, _const.I18N_TARGET_KEY_BASE_OAUTH_PROVIDER_DESCRIPTION, []int64{idValue})
	if err != nil {
		return nil, err
	}
	return &adminv1.BaseOauthProviderForm{Id: item.ID, Provider: item.Provider, Name: item.Name, Description: item.Description, Icon: item.Icon, ClientId: item.ClientID, SecretConfigured: item.ClientSecret != "", RedirectUri: item.RedirectURI, Scopes: scopes, Config: config, Sort: item.Sort, Status: commonv1.Status(item.Status), NameI18ns: nameI18ns[idValue], DescriptionI18ns: descriptionI18ns[idValue]}, nil
}

// CreateBaseOauthProvider 创建 OAuth 登录方式并刷新运行时快照。
func (c *BaseOauthProviderCase) CreateBaseOauthProvider(ctx context.Context, req *adminv1.BaseOauthProviderForm) error {
	if req.GetClientSecret() == "" {
		return errorsx.InvalidArgument("第三方应用密钥不能为空")
	}
	item, err := c.formEntity(req, nil)
	if err != nil {
		return err
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	item.CreatedBy, item.UpdatedBy = authInfo.UserId, authInfo.UserId
	item.CreatedAt, item.UpdatedAt = time.Now(), time.Now()
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.Create(txCtx, item); err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("Provider标识重复", "base_oauth_provider", "provider", "unique_base_oauth_provider").WithCause(err)
			}
			return err
		}
		return c.saveI18ns(txCtx, req, item)
	})
	if err != nil {
		return err
	}
	return c.RefreshBaseOauthProvider(ctx)
}

// UpdateBaseOauthProvider 更新 OAuth 登录方式并刷新运行时快照。
func (c *BaseOauthProviderCase) UpdateBaseOauthProvider(ctx context.Context, req *adminv1.BaseOauthProviderForm) error {
	oldItem, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if req.GetProvider() != oldItem.Provider {
		return errorsx.Conflict("Provider标识不可修改")
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
		if err = c.UpdateByID(txCtx, item); err != nil {
			return err
		}
		return c.saveI18ns(txCtx, req, item)
	})
	if err != nil {
		return err
	}
	return c.RefreshBaseOauthProvider(ctx)
}

// DeleteBaseOauthProvider 删除 OAuth 登录方式及其翻译并刷新运行时快照。
func (c *BaseOauthProviderCase) DeleteBaseOauthProvider(ctx context.Context, ids string) error {
	idValues := _string.ConvertStringToInt64Array(ids)
	list, err := c.ListByIDs(ctx, idValues)
	if err != nil {
		return err
	}
	if len(list) != len(idValues) {
		return errorsx.ResourceNotFound("OAuth登录方式不存在")
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.DeleteByIDs(txCtx, idValues); err != nil {
			return err
		}
		if err = c.baseI18nCase.DeleteBaseI18n(txCtx, _const.I18N_TARGET_KEY_BASE_OAUTH_PROVIDER_NAME, idValues); err != nil {
			return err
		}
		return c.baseI18nCase.DeleteBaseI18n(txCtx, _const.I18N_TARGET_KEY_BASE_OAUTH_PROVIDER_DESCRIPTION, idValues)
	})
	if err != nil {
		return err
	}
	return c.RefreshBaseOauthProvider(ctx)
}

// SetBaseOauthProviderStatus 设置 OAuth 登录方式状态并刷新运行时快照。
func (c *BaseOauthProviderCase) SetBaseOauthProviderStatus(ctx context.Context, req *adminv1.SetBaseOauthProviderStatusRequest) error {
	if req.GetStatus() != commonv1.Status_STATUS_ENABLE && req.GetStatus() != commonv1.Status_STATUS_DISABLE {
		return errorsx.InvalidArgument("OAuth登录方式状态无效")
	}
	item, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if item.Status == int32(req.GetStatus()) {
		return nil
	}
	query := c.Query(ctx).BaseOauthProvider
	if _, err = query.WithContext(ctx).Where(query.ID.Eq(item.ID)).UpdateSimple(query.Status.Value(int32(req.GetStatus())), query.UpdatedAt.Value(time.Now())); err != nil {
		return err
	}
	return c.RefreshBaseOauthProvider(ctx)
}

// formEntity 将管理表单转换为持久化实体，字段加密由存储回调完成。
func (c *BaseOauthProviderCase) formEntity(req *adminv1.BaseOauthProviderForm, oldItem *models.BaseOauthProvider) (*models.BaseOauthProvider, error) {
	scopes, err := json.Marshal(req.GetScopes())
	if err != nil {
		return nil, errorsx.InvalidArgument("OAuth Scope配置无效").WithCause(err)
	}
	config := []byte("{}")
	if req.GetConfig() != nil {
		config, err = json.Marshal(req.GetConfig().AsMap())
		if err != nil {
			return nil, errorsx.InvalidArgument("Provider个性化配置无效").WithCause(err)
		}
	}
	secret := req.GetClientSecret()
	if secret == "" && oldItem != nil {
		secret = oldItem.ClientSecret
	}
	return &models.BaseOauthProvider{Provider: req.GetProvider(), Name: req.GetName(), Description: req.GetDescription(), Icon: req.GetIcon(), ClientID: req.GetClientId(), ClientSecret: secret, RedirectURI: req.GetRedirectUri(), Scopes: string(scopes), Config: string(config), Sort: req.GetSort(), Status: int32(req.GetStatus())}, nil
}

// runtimeConfig 将查询回调已解密的记录转换为 OAuth SDK 参数。
func (c *BaseOauthProviderCase) runtimeConfig(item *models.BaseOauthProvider) (*provider.Config, error) {
	config, scopes, err := decodeOauthProviderJSON(item)
	if err != nil {
		return nil, err
	}
	return &provider.Config{ClientID: item.ClientID, ClientSecret: item.ClientSecret, RedirectURI: item.RedirectURI, Scopes: scopes, Parameters: provider.Parameters(config.AsMap())}, nil
}

// toDTO 将 OAuth 登录方式实体转换为列表响应。
func (c *BaseOauthProviderCase) toDTO(item *models.BaseOauthProvider) (*adminv1.BaseOauthProvider, error) {
	config, scopes, err := decodeOauthProviderJSON(item)
	if err != nil {
		return nil, err
	}
	return &adminv1.BaseOauthProvider{Id: item.ID, Provider: item.Provider, Name: item.Name, Description: item.Description, Icon: item.Icon, ClientId: item.ClientID, RedirectUri: item.RedirectURI, Scopes: scopes, Config: config, Sort: item.Sort, Status: commonv1.Status(item.Status), SecretConfigured: item.ClientSecret != "", CreatedAt: item.CreatedAt.Format(time.DateTime), UpdatedAt: item.UpdatedAt.Format(time.DateTime)}, nil
}

// saveI18ns 保存 OAuth 登录方式名称和提示语翻译。
func (c *BaseOauthProviderCase) saveI18ns(ctx context.Context, req *adminv1.BaseOauthProviderForm, item *models.BaseOauthProvider) error {
	var err error
	err = c.baseI18nCase.SaveBaseI18n(ctx, _const.I18N_TARGET_KEY_BASE_OAUTH_PROVIDER_NAME, item.ID, item.Name, req.GetNameI18ns(), nil)
	if err != nil {
		return err
	}
	return c.baseI18nCase.SaveBaseI18n(ctx, _const.I18N_TARGET_KEY_BASE_OAUTH_PROVIDER_DESCRIPTION, item.ID, item.Description, req.GetDescriptionI18ns(), nil)
}

// decodeOauthProviderJSON 解析 OAuth Scope 数组和个性化配置对象。
func decodeOauthProviderJSON(item *models.BaseOauthProvider) (*structpb.Struct, []string, error) {
	var err error
	var scopes []string
	err = json.Unmarshal([]byte(item.Scopes), &scopes)
	if err != nil {
		return nil, nil, errorsx.Internal("解析OAuth Scope失败").WithCause(err)
	}
	values := make(map[string]any)
	err = json.Unmarshal([]byte(item.Config), &values)
	if err != nil {
		return nil, nil, errorsx.Internal("解析Provider个性化配置失败").WithCause(err)
	}
	var config *structpb.Struct
	config, err = structpb.NewStruct(values)
	if err != nil {
		return nil, nil, errorsx.Internal("转换Provider个性化配置失败").WithCause(err)
	}
	return config, scopes, nil
}
