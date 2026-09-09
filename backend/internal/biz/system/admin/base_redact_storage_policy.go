package biz

import (
	"context"
	"strings"
	"time"

	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/redact"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gorm"
)

// BaseRedactStoragePolicyCase 提供入库脱敏策略管理能力。
type BaseRedactStoragePolicyCase struct {
	*biz.BaseCase
	tx data.Transaction
	*data.BaseRedactStoragePolicyRepository
	storageValueRepo *data.BaseRedactStorageValueRepository
	ruleRepo         *data.BaseRedactRuleRepository
	resolver         *kit.RedactPolicyResolver
}

// NewBaseRedactStoragePolicyCase 创建入库脱敏策略业务实例。
func NewBaseRedactStoragePolicyCase(baseCase *biz.BaseCase, tx data.Transaction, repo *data.BaseRedactStoragePolicyRepository, storageValueRepo *data.BaseRedactStorageValueRepository, ruleRepo *data.BaseRedactRuleRepository, resolver *kit.RedactPolicyResolver) *BaseRedactStoragePolicyCase {
	return &BaseRedactStoragePolicyCase{BaseCase: baseCase, tx: tx, BaseRedactStoragePolicyRepository: repo, storageValueRepo: storageValueRepo, ruleRepo: ruleRepo, resolver: resolver}
}

// PageBaseRedactStoragePolicy 分页查询入库脱敏策略。
func (c *BaseRedactStoragePolicyCase) PageBaseRedactStoragePolicy(ctx context.Context, req *adminv1.PageBaseRedactStoragePolicyRequest) (*adminv1.PageBaseRedactStoragePolicyResponse, error) {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).BaseRedactStoragePolicy
	opts := make([]repository.QueryOption, 0, 6)
	opts = append(opts, repository.Order(query.ID.Asc()))
	if req.GetTableName() != "" {
		opts = append(opts, repository.Where(query.TableName_.Like("%"+req.GetTableName()+"%")))
	}
	if req.GetColumnName() != "" {
		opts = append(opts, repository.Where(query.ColumnName.Like("%"+req.GetColumnName()+"%")))
	}
	if req.GetSourceName() != "" {
		opts = append(opts, repository.Where(query.SourceName.Eq(req.GetSourceName())))
	}
	if req.RuleId != nil {
		opts = append(opts, repository.Where(query.RuleID.Eq(req.GetRuleId())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	var list []*models.BaseRedactStoragePolicy
	var total int64
	list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BaseRedactStoragePolicy, 0, len(list))
	for _, item := range list {
		var dto *adminv1.BaseRedactStoragePolicy
		dto, err = c.toBaseRedactStoragePolicy(ctx, item)
		if err != nil {
			return nil, err
		}
		result = append(result, dto)
	}
	return &adminv1.PageBaseRedactStoragePolicyResponse{BaseRedactStoragePolicies: result, Total: int32(total)}, nil
}

// GetBaseRedactStoragePolicy 查询入库脱敏策略详情。
func (c *BaseRedactStoragePolicyCase) GetBaseRedactStoragePolicy(ctx context.Context, req *adminv1.GetBaseRedactStoragePolicyRequest) (*adminv1.BaseRedactStoragePolicyForm, error) {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	var item *models.BaseRedactStoragePolicy
	item, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &adminv1.BaseRedactStoragePolicyForm{Id: item.ID, TableName: item.TableName_, ColumnName: item.ColumnName, RuleId: item.RuleID, RuleParams: item.RuleParams, SourceName: item.SourceName, Status: commonv1.Status(item.Status), Remark: item.Remark}, nil
}

// CreateBaseRedactStoragePolicy 批量创建入库脱敏策略。
func (c *BaseRedactStoragePolicyCase) CreateBaseRedactStoragePolicy(ctx context.Context, inputs []*adminv1.BaseRedactStoragePolicyForm) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return errorsx.InvalidArgument("入库脱敏策略列表不能为空")
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	items := make([]*models.BaseRedactStoragePolicy, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			return errorsx.InvalidArgument("入库脱敏策略不能为空")
		}
		var rule *models.BaseRedactRule
		rule, err = c.validateStorageForm(ctx, input)
		if err != nil {
			return err
		}
		item := &models.BaseRedactStoragePolicy{SourceName: input.GetSourceName(), TableName_: input.GetTableName(), ColumnName: input.GetColumnName(), RuleID: rule.ID, RuleParams: input.GetRuleParams(), Status: int32(input.GetStatus()), Remark: input.GetRemark(), CreatedBy: authInfo.UserId, UpdatedBy: authInfo.UserId, CreatedAt: now, UpdatedAt: now}
		if item.Status == 0 {
			item.Status = _const.STATUS_STATUS_ENABLE
		}
		items = append(items, item)
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.BatchCreate(txCtx, items)
		if err != nil {
			if errorsx.IsDuplicateKey(err) {
				return errorsx.UniqueConflict("同一数据表字段的入库脱敏策略已存在", "base_redact_storage_policy", "table_name,column_name", "unique_base_redact_storage_policy").WithCause(err)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return redact.RefreshRedactRuntime(ctx, c.resolver)
}

// UpdateBaseRedactStoragePolicy 批量更新入库脱敏策略。
func (c *BaseRedactStoragePolicyCase) UpdateBaseRedactStoragePolicy(ctx context.Context, inputs []*adminv1.BaseRedactStoragePolicyForm) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return errorsx.InvalidArgument("入库脱敏策略列表不能为空")
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	items := make([]*models.BaseRedactStoragePolicy, 0, len(inputs))
	for _, input := range inputs {
		if input == nil {
			return errorsx.InvalidArgument("入库脱敏策略不能为空")
		}
		var oldItem *models.BaseRedactStoragePolicy
		oldItem, err = c.FindByID(ctx, input.GetId())
		if err != nil {
			return err
		}
		if input.GetSourceName() != oldItem.SourceName || input.GetTableName() != oldItem.TableName_ || input.GetColumnName() != oldItem.ColumnName {
			return errorsx.ProtectedResourceConflict("入库脱敏策略创建后不允许修改数据源、数据表或字段", "base_redact_storage_policy")
		}
		var rule *models.BaseRedactRule
		rule, err = c.validateStorageForm(ctx, input)
		if err != nil {
			return err
		}
		item := &models.BaseRedactStoragePolicy{ID: oldItem.ID, SourceName: input.GetSourceName(), TableName_: input.GetTableName(), ColumnName: input.GetColumnName(), RuleID: rule.ID, RuleParams: input.GetRuleParams(), Status: int32(input.GetStatus()), Remark: input.GetRemark(), CreatedBy: oldItem.CreatedBy, UpdatedBy: authInfo.UserId, CreatedAt: oldItem.CreatedAt, UpdatedAt: now}
		if item.Status == 0 {
			item.Status = oldItem.Status
		}
		if item.Status == _const.STATUS_STATUS_DISABLE {
			err = c.ensureNoStoredValues(ctx, []int64{item.ID})
			if err != nil {
				return err
			}
		}
		items = append(items, item)
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		for _, item := range items {
			err = c.UpdateByID(txCtx, item)
			if err != nil {
				if errorsx.IsDuplicateKey(err) {
					return errorsx.UniqueConflict("同一数据表字段的入库脱敏策略已存在", "base_redact_storage_policy", "table_name,column_name", "unique_base_redact_storage_policy").WithCause(err)
				}
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return redact.RefreshRedactRuntime(ctx, c.resolver)
}

// DeleteBaseRedactStoragePolicy 删除入库脱敏策略。
func (c *BaseRedactStoragePolicyCase) DeleteBaseRedactStoragePolicy(ctx context.Context, value string) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	ids := _string.ConvertStringToInt64Array(value)
	if len(ids) == 0 {
		return errorsx.InvalidArgument("入库策略ID不能为空")
	}
	var items []*models.BaseRedactStoragePolicy
	items, err = c.ListByIDs(ctx, ids)
	if err != nil {
		return err
	}
	if len(items) != len(ids) {
		return errorsx.ResourceNotFound("入库脱敏策略不存在")
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		err = c.ensureNoStoredValues(txCtx, ids)
		if err != nil {
			return err
		}
		return c.DeleteByIDs(txCtx, ids)
	})
	if err != nil {
		return err
	}
	return redact.RefreshRedactRuntime(ctx, c.resolver)
}

// SetBaseRedactStoragePolicyStatus 设置入库脱敏策略状态。
func (c *BaseRedactStoragePolicyCase) SetBaseRedactStoragePolicyStatus(ctx context.Context, req *adminv1.SetBaseRedactStoragePolicyStatusRequest) error {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return err
	}
	if req.GetStatus() != commonv1.Status_STATUS_ENABLE && req.GetStatus() != commonv1.Status_STATUS_DISABLE {
		return errorsx.InvalidArgument("入库脱敏策略状态无效")
	}
	var item *models.BaseRedactStoragePolicy
	item, err = c.FindByID(ctx, req.GetId())
	if err != nil {
		return err
	}
	if item.Status == int32(req.GetStatus()) {
		return nil
	}
	if req.GetStatus() == commonv1.Status_STATUS_DISABLE {
		err = c.ensureNoStoredValues(ctx, []int64{item.ID})
		if err != nil {
			return err
		}
	}
	err = c.UpdateByID(ctx, &models.BaseRedactStoragePolicy{ID: item.ID, Status: int32(req.GetStatus())})
	if err != nil {
		return err
	}
	return redact.RefreshRedactRuntime(ctx, c.resolver)
}

// ensureNoStoredValues 检查入库策略是否已经保存敏感字段原文。
func (c *BaseRedactStoragePolicyCase) ensureNoStoredValues(ctx context.Context, ids []int64) error {
	query := c.storageValueRepo.Query(ctx).BaseRedactStorageValue
	var count int64
	var err error
	count, err = c.storageValueRepo.Count(ctx, repository.Where(query.StoragePolicyID.In(ids...)))
	if err != nil {
		return err
	}
	if count > 0 {
		return errorsx.ProtectedResourceConflict("已有数据使用该入库脱敏策略，不能停用或删除", "base_redact_storage_policy")
	}
	return nil
}

// validateStorageForm 校验入库策略的目标字段和规则参数。
func (c *BaseRedactStoragePolicyCase) validateStorageForm(ctx context.Context, input *adminv1.BaseRedactStoragePolicyForm) (*models.BaseRedactRule, error) {
	client, err := GormClientBySourceName(c.BaseCase, input.GetSourceName())
	if err != nil {
		return nil, errorsx.InvalidArgument("请选择已初始化的数据源").WithCause(err)
	}
	if !client.Migrator().HasTable(input.GetTableName()) {
		return nil, errorsx.ResourceNotFound("数据库表不存在")
	}
	if !client.Migrator().HasColumn(input.GetTableName(), input.GetColumnName()) {
		return nil, errorsx.ResourceNotFound("数据库字段不存在")
	}
	var columnTypes []gorm.ColumnType
	columnTypes, err = client.Migrator().ColumnTypes(input.GetTableName())
	if err != nil {
		return nil, errorsx.Internal("查询数据库字段约束失败").WithCause(err)
	}
	for _, columnType := range columnTypes {
		if !strings.EqualFold(columnType.Name(), input.GetColumnName()) {
			continue
		}
		databaseType := strings.ToLower(columnType.DatabaseTypeName())
		switch databaseType {
		case "char", "varchar", "text", "tinytext", "mediumtext", "longtext", "enum", "set", "json":
		default:
			return nil, errorsx.InvalidArgument("只有字符串字段支持入库脱敏")
		}
		unique, ok := columnType.Unique()
		if ok && unique {
			return nil, errorsx.InvalidArgument("唯一索引字段不支持入库脱敏")
		}
		break
	}
	var rule *models.BaseRedactRule
	rule, err = c.ruleRepo.FindByID(ctx, input.GetRuleId())
	if err != nil {
		return nil, errorsx.ResourceNotFound("脱敏规则不存在").WithCause(err)
	}
	if rule.Status != _const.STATUS_STATUS_ENABLE {
		return nil, errorsx.InvalidArgument("脱敏规则已停用")
	}
	_, err = redact.ValidateRuleTemplate(rule.Code, rule.RuleType, input.GetRuleParams())
	if err != nil {
		return nil, errorsx.InvalidArgument("入库脱敏规则参数无效").WithCause(err)
	}
	status := input.GetStatus()
	if status != commonv1.Status_STATUS_UNSPECIFIED && status != commonv1.Status_STATUS_ENABLE && status != commonv1.Status_STATUS_DISABLE {
		return nil, errorsx.InvalidArgument("入库脱敏策略状态无效")
	}
	return rule, nil
}

// toBaseRedactStoragePolicy 转换入库脱敏策略列表项。
func (c *BaseRedactStoragePolicyCase) toBaseRedactStoragePolicy(ctx context.Context, item *models.BaseRedactStoragePolicy) (*adminv1.BaseRedactStoragePolicy, error) {
	rule, err := c.ruleRepo.FindByID(ctx, item.RuleID)
	if err != nil {
		return nil, err
	}
	return &adminv1.BaseRedactStoragePolicy{Id: item.ID, SourceName: item.SourceName, TableName: item.TableName_, ColumnName: item.ColumnName, RuleId: item.RuleID, RuleCode: rule.Code, RuleName: rule.Name, RuleType: rule.RuleType, RuleParams: item.RuleParams, Status: commonv1.Status(item.Status), Remark: item.Remark, CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05")}, nil
}
