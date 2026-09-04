package biz

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/database/gorm"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	coreconst "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"
)

const (
	baseTenantAdminUserName   = "admin"
	baseTenantAdminNickName   = "管理员"
	baseTenantDefaultDeptName = "默认部门"
	baseTenantDefaultDeptPath = "/0/%d"
	baseTenantDefaultDeptSort = int32(0)
	baseTenantInitialCode     = int64(1000)
	baseTenantMaxCode         = int64(9999)
	baseTenantNumericCodeExpr = "^[0-9]+$"
)

// BaseTenantCase 租户业务实例。
type BaseTenantCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseTenantRepository
	baseDeptRepo            *data.BaseDeptRepository
	baseRoleRepo            *data.BaseRoleRepository
	baseUserRepo            *data.BaseUserRepository
	baseMessageRepo         *data.BaseMessageRepository
	baseMessageDispatchRepo *data.BaseMessageDispatchRepository
	baseMessageDeliveryRepo *data.BaseMessageDeliveryRepository
	oauthClientRepo         *data.OauthClientRepository
	baseFileRepo            *data.BaseFileRepository
	baseThirdAccountRepo    *data.BaseThirdAccountRepository
	baseUserMFARepo         *data.BaseUserMFARepository
	baseUserMFATotpRepo     *data.BaseUserMFATotpRepository
	baseUserMFARecoveryRepo *data.BaseUserMFARecoveryRepository
	baseUserMFAWebauthnRepo *data.BaseUserMFAWebauthnRepository
	userToken               *authData.UserToken
	casbinRuleRepo          *data.CasbinRuleRepository
	casbinRuleCase          *CasbinRuleCase
	formMapper              *mapper.CopierMapper[adminv1.BaseTenantForm, models.BaseTenant]
	mapper                  *mapper.CopierMapper[adminv1.BaseTenant, models.BaseTenant]
}

// NewBaseTenantCase 创建租户业务实例。
func NewBaseTenantCase(
	baseCase *biz.BaseCase,
	tx data.Transaction,
	baseTenantRepo *data.BaseTenantRepository,
	baseDeptRepo *data.BaseDeptRepository,
	baseRoleRepo *data.BaseRoleRepository,
	baseUserRepo *data.BaseUserRepository,
	baseMessageRepo *data.BaseMessageRepository,
	baseMessageDispatchRepo *data.BaseMessageDispatchRepository,
	baseMessageDeliveryRepo *data.BaseMessageDeliveryRepository,
	oauthClientRepo *data.OauthClientRepository,
	baseFileRepo *data.BaseFileRepository,
	baseThirdAccountRepo *data.BaseThirdAccountRepository,
	baseUserMFARepo *data.BaseUserMFARepository,
	baseUserMFATotpRepo *data.BaseUserMFATotpRepository,
	baseUserMFARecoveryRepo *data.BaseUserMFARecoveryRepository,
	baseUserMFAWebauthnRepo *data.BaseUserMFAWebauthnRepository,
	userToken *authData.UserToken,
	casbinRuleRepo *data.CasbinRuleRepository,
	casbinRuleCase *CasbinRuleCase,
) *BaseTenantCase {
	return &BaseTenantCase{
		BaseCase:                baseCase,
		tx:                      tx,
		BaseTenantRepository:    baseTenantRepo,
		baseDeptRepo:            baseDeptRepo,
		baseRoleRepo:            baseRoleRepo,
		baseUserRepo:            baseUserRepo,
		baseMessageRepo:         baseMessageRepo,
		baseMessageDispatchRepo: baseMessageDispatchRepo,
		baseMessageDeliveryRepo: baseMessageDeliveryRepo,
		oauthClientRepo:         oauthClientRepo,
		baseFileRepo:            baseFileRepo,
		baseThirdAccountRepo:    baseThirdAccountRepo,
		baseUserMFARepo:         baseUserMFARepo,
		baseUserMFATotpRepo:     baseUserMFATotpRepo,
		baseUserMFARecoveryRepo: baseUserMFARecoveryRepo,
		baseUserMFAWebauthnRepo: baseUserMFAWebauthnRepo,
		userToken:               userToken,
		casbinRuleRepo:          casbinRuleRepo,
		casbinRuleCase:          casbinRuleCase,
		formMapper:              mapper.NewCopierMapper[adminv1.BaseTenantForm, models.BaseTenant](),
		mapper:                  mapper.NewCopierMapper[adminv1.BaseTenant, models.BaseTenant](),
	}
}

// FindDefault 查询默认租户。
func (c *BaseTenantCase) FindDefault(ctx context.Context) (*models.BaseTenant, error) {
	query := c.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.Code.Eq(gorm.DefaultTenantCode)))
	return c.Find(ctx, opts...)
}

// OptionBaseTenant 查询租户选项。
func (c *BaseTenantCase) OptionBaseTenant(ctx context.Context, req *adminv1.OptionBaseTenantRequest) (*commonv1.SelectOptionResponse, error) {
	query := c.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	opts = append(opts, repository.Where(query.Status.Eq(coreconst.STATUS_STATUS_ENABLE)))
	if req.GetKeyword() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetKeyword()+"%")))
	}

	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &commonv1.SelectOptionResponse_Option{
			Label: item.Name,
			Value: item.ID,
		})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

// PageBaseTenant 分页查询租户。
func (c *BaseTenantCase) PageBaseTenant(ctx context.Context, req *adminv1.PageBaseTenantRequest) (*adminv1.PageBaseTenantResponse, error) {
	query := c.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.GetCode() != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+req.GetCode()+"%")))
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

	resList := make([]*adminv1.BaseTenant, 0, len(list))
	for _, item := range list {
		baseTenant := c.mapper.ToDTO(item)
		baseTenant.IsProtected = isBaseTenantProtected(item)
		resList = append(resList, baseTenant)
	}
	return &adminv1.PageBaseTenantResponse{BaseTenants: resList, Total: int32(total)}, nil
}

// GetBaseTenant 获取租户。
func (c *BaseTenantCase) GetBaseTenant(ctx context.Context, id int64) (*adminv1.BaseTenantForm, error) {
	baseTenant, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	err = validateBaseTenantManagementTarget(baseTenant)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(baseTenant), nil
}

// CreateBaseTenant 创建租户。
func (c *BaseTenantCase) CreateBaseTenant(ctx context.Context, req *adminv1.BaseTenantForm) error {
	baseTenant := c.formMapper.ToEntity(req)
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		code, err := c.getNextBaseTenantCode(ctx)
		if err != nil {
			return err
		}

		// 租户编号只允许后端生成，避免客户端传入自定义编号。
		baseTenant.Code = code
		// 新租户 ID 与租户编号保持一致，编号从 1000 开始生成。
		var tenantID int64
		tenantID, err = strconv.ParseInt(code, 10, 64)
		if err != nil {
			return errorsx.Internal("解析租户编号失败").WithCause(err)
		}
		baseTenant.ID = tenantID
		// 未指定状态时，新租户默认启用，避免初始化完成后仍无法登录。
		if baseTenant.Status == 0 {
			baseTenant.Status = coreconst.STATUS_STATUS_ENABLE
		}
		err = c.Create(ctx, baseTenant)
		if err != nil {
			// 命中租户编号唯一索引冲突时，返回稳定的业务冲突错误。
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("租户编号重复", "base_tenant", "code", "unique_base_tenant").WithCause(err)
			}
			return err
		}
		return c.initTenantDefaults(ctx, baseTenant)
	})
}

// UpdateBaseTenant 更新租户。
func (c *BaseTenantCase) UpdateBaseTenant(ctx context.Context, req *adminv1.BaseTenantForm) error {
	oldBaseTenant, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	err = validateBaseTenantManagementTarget(oldBaseTenant)
	if err != nil {
		return err
	}

	baseTenant := c.formMapper.ToEntity(req)
	// 更新租户时沿用数据库中的原始编码，忽略客户端传入的 code。
	baseTenant.Code = oldBaseTenant.Code
	err = c.UpdateByID(ctx, baseTenant)
	if err != nil {
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("租户编号重复", "base_tenant", "code", "unique_base_tenant").WithCause(err)
		}
		return err
	}
	return nil
}

// DeleteBaseTenant 删除租户。
func (c *BaseTenantCase) DeleteBaseTenant(ctx context.Context, id string) error {
	ids := _string.ConvertStringToInt64Array(id)
	query := c.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.In(ids...)))
	baseTenants, err := c.List(ctx, opts...)
	if err != nil {
		return err
	}
	// 上次删除可能已提交但权限重载失败，幂等重试仍需修复内存策略。
	if len(baseTenants) == 0 {
		return c.casbinRuleCase.RebuildPolicyRule(ctx)
	}

	tenantIDs := make([]int64, 0, len(baseTenants))
	tenantCodes := make([]string, 0, len(baseTenants))
	for _, item := range baseTenants {
		if isBaseTenantProtected(item) {
			return errorsx.ProtectedResourceConflict("操作租户失败，默认租户不能操作", "base_tenant")
		}
		tenantIDs = append(tenantIDs, item.ID)
		tenantCodes = append(tenantCodes, item.Code)
	}

	err = c.tx.Transaction(ctx, func(ctx context.Context) error {
		err = c.deleteTenantData(ctx, tenantIDs, tenantCodes)
		if err != nil {
			return err
		}
		return c.DeleteByIDs(ctx, tenantIDs)
	})
	if err != nil {
		return err
	}
	return c.casbinRuleCase.RebuildPolicyRule(ctx)
}

// SetBaseTenantStatus 设置租户状态。
func (c *BaseTenantCase) SetBaseTenantStatus(ctx context.Context, req *adminv1.SetBaseTenantStatusRequest) error {
	baseTenant, err := c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	err = validateBaseTenantManagementTarget(baseTenant)
	if err != nil {
		return err
	}
	return c.UpdateByID(ctx, &models.BaseTenant{
		ID:     req.GetId(),
		Status: req.GetStatus(),
	})
}

// getNextBaseTenantCode 获取下一个可用租户编号。
func (c *BaseTenantCase) getNextBaseTenantCode(ctx context.Context) (string, error) {
	query := c.Query(ctx).BaseTenant
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Unscoped())
	opts = append(opts, repository.Where(query.Code.Regexp(baseTenantNumericCodeExpr)))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return "", err
	}

	maxCode := baseTenantInitialCode - 1
	for _, item := range list {
		var code int64
		code, err = strconv.ParseInt(item.Code, 10, 64)
		if err != nil {
			return "", errorsx.Internal("解析租户编号失败").WithCause(err)
		}
		if code > maxCode {
			maxCode = code
		}
	}
	// 四位自定义租户编号已全部使用时，拒绝继续创建。
	if maxCode >= baseTenantMaxCode {
		return "", errorsx.StateConflict("租户编号已用完", "base_tenant", strconv.FormatInt(maxCode, 10), strconv.FormatInt(baseTenantMaxCode, 10))
	}
	return fmt.Sprintf("%04d", maxCode+1), nil
}

// initTenantDefaults 初始化租户默认组织、角色和管理员账号。
func (c *BaseTenantCase) initTenantDefaults(ctx context.Context, baseTenant *models.BaseTenant) error {
	baseDept := &models.BaseDept{
		TenantID: baseTenant.ID,
		ParentID: 0,
		Name:     baseTenantDefaultDeptName,
		Sort:     baseTenantDefaultDeptSort,
		Status:   coreconst.STATUS_STATUS_ENABLE,
		Remark:   "租户默认部门",
	}
	err := c.baseDeptRepo.Create(ctx, baseDept)
	if err != nil {
		return errorsx.Internal("初始化租户默认部门失败").WithCause(err)
	}

	baseDept.Path = fmt.Sprintf(baseTenantDefaultDeptPath, baseDept.ID)
	err = c.baseDeptRepo.UpdateByID(ctx, baseDept)
	if err != nil {
		return errorsx.Internal("初始化租户默认部门失败").WithCause(err)
	}

	roleQuery := c.baseRoleRepo.Query(ctx).BaseRole
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(roleQuery.Code.Eq(coreconst.BASE_ROLE_CODE_TENANT)))
	var defaultRole *models.BaseRole
	defaultRole, err = c.baseRoleRepo.Find(ctx, opts...)
	if err != nil {
		return errorsx.Internal("初始化租户管理员角色失败").WithCause(err)
	}

	baseRole := &models.BaseRole{
		TenantID:  baseTenant.ID,
		Name:      defaultRole.Name,
		Code:      defaultRole.Code,
		DataScope: defaultRole.DataScope,
		Menus:     defaultRole.Menus,
		Status:    defaultRole.Status,
		Remark:    defaultRole.Remark,
	}
	err = c.baseRoleRepo.Create(ctx, baseRole)
	if err != nil {
		// 命中角色编码唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("同一租户的角色编码重复", "base_role", "", "unique_base_role").WithCause(err)
		}
		return errorsx.Internal("初始化租户管理员角色失败").WithCause(err)
	}

	// 生成不可预测的随机口令并禁用账号，由平台管理员重置后再启用。
	randomPassword := make([]byte, 32)
	if _, err = rand.Read(randomPassword); err != nil {
		return errorsx.Internal("初始化租户管理员账号失败").WithCause(err)
	}
	var password string
	password, err = crypto.Encrypt(base64.RawURLEncoding.EncodeToString(randomPassword))
	if err != nil {
		return errorsx.Internal("初始化租户管理员账号失败").WithCause(err)
	}

	baseUser := &models.BaseUser{
		TenantID:           baseTenant.ID,
		UserName:           baseTenantAdminUserName,
		UserCode:           baseTenantAdminUserName,
		NickName:           baseTenantAdminNickName,
		RoleID:             baseRole.ID,
		DeptID:             baseDept.ID,
		Phone:              baseTenant.ContactPhone,
		Password:           password,
		Gender:             _const.BASE_USER_GENDER_SECRET,
		Status:             coreconst.STATUS_STATUS_DISABLE,
		Remark:             "租户默认管理员，须由平台管理员设置密码后启用",
		PasswordChangedAt:  time.Now(),
		PasswordHistory:    "[]",
		MustChangePassword: _const.BASE_USER_PASSWORD_CHANGE_STATUS_REQUIRED,
	}
	err = c.baseUserRepo.Create(ctx, baseUser)
	if err != nil {
		// 命中用户账号或用户编号唯一索引冲突时，返回稳定的业务冲突错误。
		if errorsx.IsDuplicateKey(err) {
			return errorsx.UniqueConflict("同一租户的用户账号或用户编号重复", "base_user", "", "unique_base_user").WithCause(err)
		}
		return errorsx.Internal("初始化租户管理员账号失败").WithCause(err)
	}
	err = c.casbinRuleCase.RebuildCasbinRuleByRole(ctx, baseRole)
	if err != nil {
		return errorsx.Internal("初始化租户管理员角色权限失败").WithCause(err)
	}
	return nil
}

// deleteTenantData 清理租户下全部用户、角色、部门和权限规则。
func (c *BaseTenantCase) deleteTenantData(ctx context.Context, tenantIDs []int64, tenantCodes []string) error {
	casbinRuleQuery := c.casbinRuleRepo.Query(ctx).CasbinRule
	casbinRuleOpts := make([]repository.QueryOption, 0, 1)
	casbinRuleOpts = append(casbinRuleOpts, repository.Where(casbinRuleQuery.V0.In(tenantCodes...)))
	err := c.casbinRuleRepo.Delete(ctx, casbinRuleOpts...)
	if err != nil {
		return err
	}

	userQuery := c.baseUserRepo.Query(ctx).BaseUser
	userOpts := make([]repository.QueryOption, 0, 1)
	userOpts = append(userOpts, repository.Where(userQuery.TenantID.In(tenantIDs...)))
	var users []*models.BaseUser
	users, err = c.baseUserRepo.List(ctx, userOpts...)
	if err != nil {
		return err
	}
	userIDs := make([]int64, 0, len(users))
	for _, item := range users {
		userIDs = append(userIDs, item.ID)
	}
	if c.userToken != nil {
		for _, userID := range userIDs {
			if err = c.userToken.RemoveToken(userID); err != nil {
				return errorsx.Internal("撤销租户用户登录令牌失败").WithCause(err)
			}
		}
	}
	if len(userIDs) > 0 {
		thirdAccountQuery := c.baseThirdAccountRepo.Query(ctx).BaseThirdAccount
		if err = c.baseThirdAccountRepo.Delete(ctx, repository.Where(thirdAccountQuery.UserID.In(userIDs...))); err != nil {
			return err
		}
		mfaQuery := c.baseUserMFARepo.Query(ctx).BaseUserMFA
		var mfas []*models.BaseUserMFA
		mfas, err = c.baseUserMFARepo.List(ctx, repository.Where(mfaQuery.UserID.In(userIDs...)))
		if err != nil {
			return err
		}
		mfaIDs := make([]int64, 0, len(mfas))
		for _, item := range mfas {
			mfaIDs = append(mfaIDs, item.ID)
		}
		if len(mfaIDs) > 0 {
			recoveryQuery := c.baseUserMFARecoveryRepo.Query(ctx).BaseUserMFARecovery
			if err = c.baseUserMFARecoveryRepo.Delete(ctx, repository.Where(recoveryQuery.MFAID.In(mfaIDs...))); err != nil {
				return err
			}
			totpQuery := c.baseUserMFATotpRepo.Query(ctx).BaseUserMFATotp
			if err = c.baseUserMFATotpRepo.Delete(ctx, repository.Where(totpQuery.MFAID.In(mfaIDs...))); err != nil {
				return err
			}
			webauthnQuery := c.baseUserMFAWebauthnRepo.Query(ctx).BaseUserMFAWebauthn
			if err = c.baseUserMFAWebauthnRepo.Delete(ctx, repository.Where(webauthnQuery.MFAID.In(mfaIDs...))); err != nil {
				return err
			}
			if err = c.baseUserMFARepo.DeleteByIDs(ctx, mfaIDs); err != nil {
				return err
			}
		}
	}
	err = c.baseUserRepo.DeleteByIDs(ctx, userIDs)
	if err != nil {
		return err
	}
	oauthQuery := c.oauthClientRepo.Query(ctx).OauthClient
	var oauthClients []*models.OauthClient
	oauthClients, err = c.oauthClientRepo.List(ctx, repository.Where(oauthQuery.TenantID.In(tenantIDs...)))
	if err != nil {
		return err
	}
	if c.userToken != nil {
		for _, client := range oauthClients {
			if err = c.userToken.RemoveToken(-client.ID); err != nil {
				return errorsx.Internal("撤销租户开放授权令牌失败").WithCause(err)
			}
		}
	}
	fileQuery := c.baseFileRepo.Query(ctx).BaseFile
	if err = c.baseFileRepo.Delete(ctx, repository.Where(fileQuery.TenantID.In(tenantIDs...))); err != nil {
		return err
	}
	if err = c.oauthClientRepo.Delete(ctx, repository.Where(oauthQuery.TenantID.In(tenantIDs...))); err != nil {
		return err
	}

	deliveryQuery := c.baseMessageDeliveryRepo.Query(ctx).BaseMessageDelivery
	err = c.baseMessageDeliveryRepo.Delete(ctx, repository.Where(deliveryQuery.TenantID.In(tenantIDs...)))
	if err != nil {
		return err
	}
	dispatchQuery := c.baseMessageDispatchRepo.Query(ctx).BaseMessageDispatch
	err = c.baseMessageDispatchRepo.Delete(ctx, repository.Where(dispatchQuery.TenantID.In(tenantIDs...)))
	if err != nil {
		return err
	}
	messageQuery := c.baseMessageRepo.Query(ctx).BaseMessage
	err = c.baseMessageRepo.Delete(ctx, repository.Where(messageQuery.TenantID.In(tenantIDs...)))
	if err != nil {
		return err
	}
	roleQuery := c.baseRoleRepo.Query(ctx).BaseRole
	roleOpts := make([]repository.QueryOption, 0, 1)
	roleOpts = append(roleOpts, repository.Where(roleQuery.TenantID.In(tenantIDs...)))
	err = c.baseRoleRepo.Delete(ctx, roleOpts...)
	if err != nil {
		return err
	}

	deptQuery := c.baseDeptRepo.Query(ctx).BaseDept
	deptOpts := make([]repository.QueryOption, 0, 1)
	deptOpts = append(deptOpts, repository.Where(deptQuery.TenantID.In(tenantIDs...)))
	err = c.baseDeptRepo.Delete(ctx, deptOpts...)
	if err != nil {
		return err
	}
	return nil
}

// validateBaseTenantManagementTarget 校验目标租户是否允许通过租户管理接口操作。
func validateBaseTenantManagementTarget(baseTenant *models.BaseTenant) error {
	if isBaseTenantProtected(baseTenant) {
		return errorsx.ProtectedResourceConflict("操作租户失败，默认租户不能操作", "base_tenant")
	}
	return nil
}

// isBaseTenantProtected 判断租户是否禁止通过租户管理操作。
func isBaseTenantProtected(baseTenant *models.BaseTenant) bool {
	return baseTenant.Code == gorm.DefaultTenantCode
}
