package biz

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/oauthsecret"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	oauthcrypto "github.com/liujitcn/kratos-admin/backend/internal/server/middleware/oauth/crypto"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/id"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	gormKit "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

const (
	oauthClientCryptoSM4       = "sm4"
	oauthClientCryptoAES       = "aes"
	oauthClientCrypto3DES      = "3des"
	oauthDevelopmentPathPrefix = "/api/v1/oauth/"
)

// OauthClientCase 开放授权客户端业务实例。
type OauthClientCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.OauthClientRepository
	baseAPICase    *BaseAPICase
	baseTenantRepo *data.BaseTenantRepository
	casbinRuleCase *CasbinRuleCase
	userToken      *authData.UserToken
	protector      *oauthsecret.Protector
	credentialMu   sync.Mutex
}

// NewOauthClientCase 创建开放授权客户端业务实例。
func NewOauthClientCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	oauthClientRepo *data.OauthClientRepository,
	baseAPICase *BaseAPICase,
	baseTenantRepo *data.BaseTenantRepository,
	casbinRuleCase *CasbinRuleCase,
	userToken *authData.UserToken,
	protector *oauthsecret.Protector,
) *OauthClientCase {
	return &OauthClientCase{
		BaseCase:              baseCase,
		tx:                    tx,
		OauthClientRepository: oauthClientRepo,
		baseAPICase:           baseAPICase,
		baseTenantRepo:        baseTenantRepo,
		casbinRuleCase:        casbinRuleCase,
		userToken:             userToken,
		protector:             protector,
	}
}

// OptionOauthClientAPI 查询可授权的开发接口选项。
func (c *OauthClientCase) OptionOauthClientAPI(ctx context.Context, req *adminv1.OptionOauthClientApiRequest) (*adminv1.OptionOauthClientApiResponse, error) {
	list, err := c.baseAPICase.OptionBaseAPI(ctx, &adminv1.OptionBaseApiRequest{})
	if err != nil {
		return nil, err
	}
	baseAPIs := make([]*adminv1.BaseApi, 0, len(list.GetBaseApis()))
	for _, item := range list.GetBaseApis() {
		if isOauthDevelopmentAPI(item.GetPath()) {
			baseAPIs = append(baseAPIs, item)
		}
	}
	return &adminv1.OptionOauthClientApiResponse{BaseApis: baseAPIs}, nil
}

// PageOauthClient 分页查询开放授权客户端。
func (c *OauthClientCase) PageOauthClient(ctx context.Context, req *adminv1.PageOauthClientRequest) (*adminv1.PageOauthClientResponse, error) {
	query := c.Query(ctx).OauthClient
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Order(query.ID.Desc()))
	if req.GetClientName() != "" {
		opts = append(opts, repository.Where(query.ClientName.Like("%"+req.GetClientName()+"%")))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	if req.TenantId != nil && req.GetTenantId() > 0 {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
	var err error
	var list []*models.OauthClient
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.OauthClient, 0, len(list))
	for _, item := range list {
		items = append(items, c.toOauthClient(item))
	}
	return &adminv1.PageOauthClientResponse{OauthClients: items, Total: int32(total)}, nil
}

// GetOauthClient 查询开放授权客户端表单。
func (c *OauthClientCase) GetOauthClient(ctx context.Context, idValue int64) (*adminv1.OauthClientForm, error) {
	var item *models.OauthClient
	var err error
	item, err = c.findClientForCurrentTenant(ctx, idValue)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("开放授权客户端不存在").WithCause(err)
		}
		return nil, errorsx.Internal("查询开放授权客户端失败").WithCause(err)
	}
	return &adminv1.OauthClientForm{
		Id:          item.ID,
		TenantId:    item.TenantID,
		ClientName:  item.ClientName,
		CryptoType:  oauthClientCryptoTypeFromString(item.CryptoType),
		IpWhitelist: item.IPWhitelist,
		Api:         _string.ConvertJsonStringToStringArray(item.API),
		Status:      commonv1.Status(item.Status),
	}, nil
}

// GetOauthClientCredentials 查询开放授权客户端非敏感凭据元数据。
func (c *OauthClientCase) GetOauthClientCredentials(ctx context.Context, idValue int64) (*adminv1.OauthClientCredentials, error) {
	var err error
	var item *models.OauthClient
	item, err = c.findClientForCurrentTenant(ctx, idValue)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("开放授权客户端不存在").WithCause(err)
		}
		return nil, errorsx.Internal("查询开放授权客户端失败").WithCause(err)
	}
	return &adminv1.OauthClientCredentials{
		ClientId:   item.ClientID,
		CryptoType: oauthClientCryptoTypeFromString(item.CryptoType),
	}, nil
}

// RotateOauthClientCredentials 轮换客户端密钥并仅在本次响应中返回新凭据。
func (c *OauthClientCase) RotateOauthClientCredentials(ctx context.Context, idValue int64) (*adminv1.OauthClientCredentials, error) {
	c.credentialMu.Lock()
	defer c.credentialMu.Unlock()
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var item *models.OauthClient
	item, err = c.findClientForCurrentTenant(ctx, idValue)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("开放授权客户端不存在").WithCause(err)
		}
		return nil, errorsx.Internal("查询开放授权客户端失败").WithCause(err)
	}
	secret := id.NewGUIDv4NoHyphen()
	var cryptoKey string
	cryptoKey, err = oauthcrypto.GenerateKey(item.CryptoType)
	if err != nil {
		return nil, errorsx.Internal("生成客户端加密密钥失败").WithCause(err)
	}
	if c.protector == nil {
		return nil, errorsx.Internal("OAuth 凭据保护器未初始化")
	}
	var protectedSecret string
	protectedSecret, err = c.protector.Protect(secret)
	if err != nil {
		return nil, errorsx.Internal("保护客户端密钥失败").WithCause(err)
	}
	var protectedCryptoKey string
	protectedCryptoKey, err = c.protector.Protect(cryptoKey)
	if err != nil {
		return nil, errorsx.Internal("保护客户端加密密钥失败").WithCause(err)
	}
	query := c.Query(ctx).OauthClient
	updated := &models.OauthClient{ID: item.ID, ClientSecret: protectedSecret, CryptoKey: protectedCryptoKey, UpdatedBy: authInfo.UserId, UpdatedAt: time.Now()}
	if err = c.Update(ctx, updated, repository.Where(query.ID.Eq(item.ID)), repository.Select(query.ClientSecret, query.CryptoKey, query.UpdatedBy, query.UpdatedAt)); err != nil {
		return nil, errorsx.Internal("轮换客户端凭据失败").WithCause(err)
	}
	if c.userToken != nil {
		if err = c.userToken.RemoveToken(-item.ID); err != nil {
			return nil, errorsx.Internal("撤销旧客户端令牌失败").WithCause(err)
		}
	}
	return &adminv1.OauthClientCredentials{ClientId: item.ClientID, ClientSecret: secret, CryptoType: oauthClientCryptoTypeFromString(item.CryptoType), CryptoKey: cryptoKey}, nil
}

// CreateOauthClient 创建开放授权客户端并同步授权策略。
func (c *OauthClientCase) CreateOauthClient(ctx context.Context, req *adminv1.OauthClientForm) error {
	var err error
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var apiList []*models.BaseAPI
	apiList, err = c.validateAPIs(ctx, req.GetApi())
	if err != nil {
		return err
	}
	var tenant *models.BaseTenant
	tenant, err = c.resolveClientTenant(ctx, req.GetTenantId())
	if err != nil {
		return err
	}
	clientSecret := id.NewGUIDv4NoHyphen()
	item := &models.OauthClient{
		TenantID:     tenant.ID,
		ClientID:     id.NewGUIDv4NoHyphen(),
		ClientSecret: clientSecret,
		ClientName:   req.GetClientName(),
		CryptoType:   oauthClientCryptoTypeToString(req.GetCryptoType()),
		CryptoKey:    "",
		IPWhitelist:  req.GetIpWhitelist(),
		API:          _string.ConvertStringArrayToString(req.GetApi()),
		Status:       int32(req.GetStatus()),
		CreatedBy:    authInfo.UserId,
		UpdatedBy:    authInfo.UserId,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	item.CryptoKey, err = oauthcrypto.GenerateKey(item.CryptoType)
	if err != nil {
		return errorsx.Internal("生成客户端加密密钥失败").WithCause(err)
	}
	if c.protector == nil {
		return errorsx.Internal("OAuth 凭据保护器未初始化")
	}
	item.ClientSecret, err = c.protector.Protect(clientSecret)
	if err != nil {
		return errorsx.Internal("保护客户端密钥失败").WithCause(err)
	}
	item.CryptoKey, err = c.protector.Protect(item.CryptoKey)
	if err != nil {
		return errorsx.Internal("保护客户端加密密钥失败").WithCause(err)
	}
	if item.Status == 0 {
		item.Status = coreconst.STATUS_STATUS_ENABLE
	}
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.Create(ctx, item)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("客户端标识重复", "oauth_client", "client_id", "unique_oauth_client_client_id").WithCause(err)
			}
			return err
		}
		return c.replaceClientPolicies(ctx, item, apiList)
	})
	if err != nil {
		return err
	}
	return nil
}

// UpdateOauthClient 更新开放授权客户端并同步授权策略。
func (c *OauthClientCase) UpdateOauthClient(ctx context.Context, req *adminv1.OauthClientForm) error {
	var err error
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	var current *models.OauthClient
	current, err = c.findClientForCurrentTenant(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("开放授权客户端不存在").WithCause(err)
		}
		return errorsx.Internal("查询开放授权客户端失败").WithCause(err)
	}
	var apiList []*models.BaseAPI
	apiList, err = c.validateAPIs(ctx, req.GetApi())
	if err != nil {
		return err
	}
	var tenant *models.BaseTenant
	tenant, err = c.resolveClientTenant(ctx, req.GetTenantId())
	if err != nil {
		return err
	}
	cryptoType := oauthClientCryptoTypeToString(req.GetCryptoType())
	if req.GetCryptoType() == adminv1.OauthClientCryptoType_OAUTH_CLIENT_CRYPTO_TYPE_UNSPECIFIED {
		cryptoType = current.CryptoType
	}
	item := &models.OauthClient{
		ID:          current.ID,
		TenantID:    tenant.ID,
		ClientName:  req.GetClientName(),
		CryptoType:  cryptoType,
		IPWhitelist: req.GetIpWhitelist(),
		API:         _string.ConvertStringArrayToString(req.GetApi()),
		Status:      int32(req.GetStatus()),
		UpdatedBy:   authInfo.UserId,
		UpdatedAt:   time.Now(),
	}
	if item.Status == 0 {
		item.Status = current.Status
	}
	if c.protector == nil {
		return errorsx.Internal("OAuth 凭据保护器未初始化")
	}
	var currentCryptoKey string
	currentCryptoKey, err = c.protector.Unprotect(current.CryptoKey)
	if err != nil {
		return errorsx.Internal("读取客户端加密密钥失败").WithCause(err)
	}
	if cryptoType != current.CryptoType || !oauthcrypto.KeyValid(cryptoType, currentCryptoKey) {
		var plainCryptoKey string
		plainCryptoKey, err = oauthcrypto.GenerateKey(cryptoType)
		if err != nil {
			return errorsx.Internal("生成客户端加密密钥失败").WithCause(err)
		}
		if c.protector == nil {
			return errorsx.Internal("OAuth 凭据保护器未初始化")
		}
		item.CryptoKey, err = c.protector.Protect(plainCryptoKey)
		if err != nil {
			return errorsx.Internal("保护客户端加密密钥失败").WithCause(err)
		}
	}
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.Query(ctx).OauthClient
		updateFields := []field.Expr{query.TenantID, query.ClientName, query.CryptoType, query.IPWhitelist, query.API, query.Status, query.UpdatedBy, query.UpdatedAt}
		if item.CryptoKey != "" {
			updateFields = append(updateFields, query.CryptoKey)
		}
		err = c.Update(ctx, item, repository.Where(query.ID.Eq(item.ID)), repository.Select(updateFields...))
		if err != nil {
			return err
		}
		current.ClientName = item.ClientName
		current.TenantID = item.TenantID
		current.CryptoType = item.CryptoType
		current.IPWhitelist = item.IPWhitelist
		current.API = item.API
		current.Status = item.Status
		if item.CryptoKey != "" {
			current.CryptoKey = item.CryptoKey
		}
		return c.replaceClientPolicies(ctx, current, apiList)
	})
	if err != nil {
		return err
	}
	return nil
}

// DeleteOauthClient 删除开放授权客户端并清理授权策略。
func (c *OauthClientCase) DeleteOauthClient(ctx context.Context, ids string) error {
	var err error
	idList := _string.ConvertStringToInt64Array(ids)
	if len(idList) == 0 {
		return nil
	}
	var list []*models.OauthClient
	list, err = c.listClientsForCurrentTenant(ctx, idList)
	if err != nil {
		return err
	}
	if len(list) != len(idList) {
		return errorsx.ResourceNotFound("开放授权客户端不存在")
	}
	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.DeleteByIDs(ctx, idList)
		if err != nil {
			return err
		}
		query := c.casbinRuleCase.Query(ctx).CasbinRule
		for _, item := range list {
			err = c.casbinRuleCase.Delete(ctx, repository.Where(query.V1.Eq(item.ClientID)))
			if err != nil {
				return err
			}
		}
		return c.casbinRuleCase.RebuildPolicyRule(ctx)
	})
	return err
}

// SetOauthClientStatus 设置开放授权客户端状态。
func (c *OauthClientCase) SetOauthClientStatus(ctx context.Context, req *adminv1.SetOauthClientStatusRequest) error {
	item, err := c.findClientForCurrentTenant(ctx, req.GetId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("开放授权客户端不存在").WithCause(err)
		}
		return errorsx.Internal("查询开放授权客户端失败").WithCause(err)
	}
	err = c.UpdateByID(ctx, &models.OauthClient{ID: item.ID, Status: int32(req.GetStatus()), UpdatedAt: time.Now()})
	if err != nil {
		return err
	}
	return c.casbinRuleCase.RebuildPolicyRule(ctx)
}

// resolveClientTenant 校验客户端绑定租户，并应用当前用户的数据权限边界。
func (c *OauthClientCase) resolveClientTenant(ctx context.Context, requestedID int64) (*models.BaseTenant, error) {
	var err error
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	tenantID := requestedID
	if authInfo.TenantCode != gormKit.DefaultTenantCode {
		if tenantID > 0 && tenantID != authInfo.TenantId {
			return nil, errorsx.PermissionDenied("不能操作其他租户的开放授权客户端")
		}
		tenantID = authInfo.TenantId
	}
	if tenantID <= 0 {
		return nil, errorsx.InvalidArgument("必须选择绑定租户")
	}
	var tenant *models.BaseTenant
	tenant, err = c.baseTenantRepo.FindByID(ctx, tenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("绑定租户不存在").WithCause(err)
		}
		return nil, errorsx.Internal("读取绑定租户失败").WithCause(err)
	}
	if tenant.Status != coreconst.STATUS_STATUS_ENABLE {
		return nil, errorsx.PermissionDenied("绑定租户已停用")
	}
	return tenant, nil
}

// findClientForCurrentTenant 按当前身份读取开放授权客户端，平台管理员可跨租户查询。
func (c *OauthClientCase) findClientForCurrentTenant(ctx context.Context, idValue int64) (*models.OauthClient, error) {
	query := c.Query(ctx).OauthClient
	return c.Find(ctx, repository.Where(query.ID.Eq(idValue)))
}

// listClientsForCurrentTenant 按当前身份读取待操作的开放授权客户端集合。
func (c *OauthClientCase) listClientsForCurrentTenant(ctx context.Context, ids []int64) ([]*models.OauthClient, error) {
	query := c.Query(ctx).OauthClient
	return c.List(ctx, repository.Where(query.ID.In(ids...)))
}

// validateAPIs 校验客户端选择的 API 均为当前系统已注册接口。
func (c *OauthClientCase) validateAPIs(ctx context.Context, operations []string) ([]*models.BaseAPI, error) {
	seen := make(map[string]struct{}, len(operations))
	for _, operation := range operations {
		if operation == "" {
			return nil, errorsx.InvalidArgument("授权 API operation 不能为空")
		}
		if _, ok := seen[operation]; ok {
			return nil, errorsx.InvalidArgument("授权 API 不能重复")
		}
		seen[operation] = struct{}{}
	}
	query := c.baseAPICase.Query(ctx).BaseAPI
	list, err := c.baseAPICase.List(ctx, repository.Where(query.Operation.In(operations...)))
	if err != nil {
		return nil, err
	}
	if len(list) != len(operations) {
		return nil, errorsx.InvalidArgument("授权 API operation 中存在不存在的接口")
	}
	for _, item := range list {
		if !isOauthDevelopmentAPI(item.Path) {
			return nil, errorsx.InvalidArgument("只能授权开发授权接口")
		}
	}
	return list, nil
}

// replaceClientPolicies 重建单个客户端的 Casbin API 策略。
func (c *OauthClientCase) replaceClientPolicies(ctx context.Context, item *models.OauthClient, apiList []*models.BaseAPI) error {
	var err error
	var tenant *models.BaseTenant
	tenant, err = c.baseTenantRepo.FindByID(ctx, item.TenantID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errorsx.ResourceNotFound("客户端绑定租户不存在").WithCause(err)
		}
		return errorsx.Internal("读取客户端绑定租户失败").WithCause(err)
	}
	query := c.casbinRuleCase.Query(ctx).CasbinRule
	err = c.casbinRuleCase.Delete(ctx, repository.Where(query.V1.Eq(item.ClientID)))
	if err != nil {
		return err
	}
	rules := make([]*models.CasbinRule, 0, len(apiList))
	for _, api := range apiList {
		rules = append(rules, &models.CasbinRule{
			Ptype: "p",
			V0:    _const.OAuthClientTenantCode(tenant.Code, item.ClientID),
			V1:    item.ClientID,
			V2:    api.Operation,
			V3:    api.Method,
			V4:    "*",
		})
	}
	if len(rules) > 0 {
		err = c.casbinRuleCase.BatchCreate(ctx, rules)
		if err != nil {
			return err
		}
	}
	return c.casbinRuleCase.RebuildPolicyRule(ctx)
}

// toOauthClient 转换客户端列表响应。
func (c *OauthClientCase) toOauthClient(item *models.OauthClient) *adminv1.OauthClient {
	return &adminv1.OauthClient{
		Id:          item.ID,
		TenantId:    item.TenantID,
		ClientId:    item.ClientID,
		ClientName:  item.ClientName,
		CryptoType:  oauthClientCryptoTypeFromString(item.CryptoType),
		IpWhitelist: item.IPWhitelist,
		Api:         _string.ConvertJsonStringToStringArray(item.API),
		Status:      commonv1.Status(item.Status),
	}
}

// oauthClientCryptoTypeToString 将协议枚举转换为持久化值。
func oauthClientCryptoTypeToString(value adminv1.OauthClientCryptoType) string {
	switch value {
	case adminv1.OauthClientCryptoType_OAUTH_CLIENT_CRYPTO_TYPE_AES:
		return oauthClientCryptoAES
	default:
		return oauthClientCryptoSM4
	}
}

// oauthClientCryptoTypeFromString 将持久化值转换为协议枚举。
func oauthClientCryptoTypeFromString(value string) adminv1.OauthClientCryptoType {
	switch strings.ToLower(value) {
	case oauthClientCryptoAES:
		return adminv1.OauthClientCryptoType_OAUTH_CLIENT_CRYPTO_TYPE_AES
	default:
		return adminv1.OauthClientCryptoType_OAUTH_CLIENT_CRYPTO_TYPE_SM4
	}
}

// isOauthDevelopmentAPI 判断接口是否属于外部开发授权路由。
func isOauthDevelopmentAPI(path string) bool {
	return strings.HasPrefix(path, oauthDevelopmentPathPrefix)
}
