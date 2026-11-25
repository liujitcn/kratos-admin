package biz

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/redact"
	"github.com/liujitcn/kratos-admin/backend/internal/data/dbmeta"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"

	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/database/gorm"
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
	if req.TenantId != nil {
		opts = append(opts, repository.Where(query.TenantID.Eq(req.GetTenantId())))
	}
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
	list, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var comments map[string]string
	comments, err = c.listTableComments(ctx, list)
	if err != nil {
		return nil, err
	}
	groups := make([]*adminv1.BaseRedactStoragePolicyGroup, 0, len(list))
	groupIndexes := make(map[string]int, len(list))
	for _, item := range list {
		key := fmt.Sprintf("%d\x00%s\x00%s", item.TenantID, item.SourceName, item.TableName_)
		index, ok := groupIndexes[key]
		if !ok {
			index = len(groups)
			groupIndexes[key] = index
			groups = append(groups, &adminv1.BaseRedactStoragePolicyGroup{Id: item.ID, TenantId: item.TenantID, SourceName: item.SourceName, TableName: item.TableName_, TableComment: comments[item.SourceName+"\x00"+item.TableName_]})
		}
		var dto *adminv1.BaseRedactStoragePolicy
		dto, err = c.toBaseRedactStoragePolicy(ctx, item, comments)
		if err != nil {
			return nil, err
		}
		groups[index].Policies = append(groups[index].Policies, dto)
	}
	start := (req.GetPageNum() - 1) * req.GetPageSize()
	if start > int64(len(groups)) {
		start = int64(len(groups))
	}
	end := min(start+req.GetPageSize(), int64(len(groups)))
	return &adminv1.PageBaseRedactStoragePolicyResponse{BaseRedactStoragePolicies: groups[start:end], Total: int32(len(groups))}, nil
}

// listTableComments 批量查询本页策略涉及的数据表注释，返回数据源和表名组合键到注释的映射。
func (c *BaseRedactStoragePolicyCase) listTableComments(ctx context.Context, rows []*models.BaseRedactStoragePolicy) (map[string]string, error) {
	tablesBySource := make(map[string][]string)
	seen := make(map[string]struct{})
	for _, row := range rows {
		key := row.SourceName + "\x00" + row.TableName_
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tablesBySource[row.SourceName] = append(tablesBySource[row.SourceName], row.TableName_)
	}
	comments := make(map[string]string)
	var err error
	for sourceName, tableNames := range tablesBySource {
		var client *gorm.Client
		client, err = GormClientBySourceName(c.BaseCase, sourceName)
		if err != nil {
			// 数据源已下线的策略回退展示裸表名，不阻塞整个列表。
			continue
		}
		var metas []dbmeta.Table
		metas, err = listDatabaseTableMetadata(ctx, client, tableNames)
		if err != nil {
			return nil, errorsx.Internal("查询数据表注释失败").WithCause(err)
		}
		for _, meta := range metas {
			comments[sourceName+"\x00"+meta.TableName] = meta.TableComment
		}
	}
	return comments, nil
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
	return &adminv1.BaseRedactStoragePolicyForm{Id: item.ID, TenantId: item.TenantID, TableName: item.TableName_, ColumnName: item.ColumnName, RuleId: item.RuleID, RuleParams: item.RuleParams, SourceName: item.SourceName, Status: commonv1.Status(item.Status), Remark: item.Remark}, nil
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
		item := &models.BaseRedactStoragePolicy{TenantID: input.GetTenantId(), SourceName: input.GetSourceName(), TableName_: input.GetTableName(), ColumnName: input.GetColumnName(), RuleID: rule.ID, RuleParams: input.GetRuleParams(), Status: int32(input.GetStatus()), Remark: input.GetRemark(), CreatedBy: authInfo.UserId, UpdatedBy: authInfo.UserId, CreatedAt: now}
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

// storageEncryptionAlgorithm 解析 ENCRYPT 规则选择的字段加密算法。
func storageEncryptionAlgorithm(raw string) (string, error) {
	var values struct {
		Encrypt struct {
			Algorithm string `json:"algorithm"`
		} `json:"encrypt"`
	}
	err := json.Unmarshal([]byte(raw), &values)
	if err != nil {
		return "", err
	}
	if values.Encrypt.Algorithm != "AES_GCM" && values.Encrypt.Algorithm != "SM4_GCM" {
		return "", errors.New("不支持的字段加密算法")
	}
	return values.Encrypt.Algorithm, nil
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
		if input.GetTenantId() != oldItem.TenantID || input.GetSourceName() != oldItem.SourceName || input.GetTableName() != oldItem.TableName_ || input.GetColumnName() != oldItem.ColumnName {
			return errorsx.ProtectedResourceConflict("入库脱敏策略创建后不允许修改租户、数据源、数据表或字段", "base_redact_storage_policy")
		}
		var rule *models.BaseRedactRule
		rule, err = c.validateStorageForm(ctx, input)
		if err != nil {
			return err
		}
		var oldRule *models.BaseRedactRule
		oldRule, err = c.ruleRepo.FindByID(ctx, oldItem.RuleID)
		if err != nil {
			return err
		}
		if strings.EqualFold(oldRule.RuleType, "ENCRYPT") {
			var oldAlgorithm string
			oldAlgorithm, err = storageEncryptionAlgorithm(oldItem.RuleParams)
			if err != nil {
				return errorsx.Internal("解析原字段加密算法失败").WithCause(err)
			}
			var newAlgorithm string
			newAlgorithm, err = storageEncryptionAlgorithm(input.GetRuleParams())
			if err != nil {
				return errorsx.InvalidArgument("字段加密算法无效").WithCause(err)
			}
			if !strings.EqualFold(rule.RuleType, "ENCRYPT") || oldAlgorithm != newAlgorithm {
				return errorsx.ProtectedResourceConflict("已有密文的字段加密算法不允许直接修改", "base_redact_storage_policy")
			}
		}
		item := &models.BaseRedactStoragePolicy{ID: oldItem.ID, TenantID: oldItem.TenantID, SourceName: input.GetSourceName(), TableName_: input.GetTableName(), ColumnName: input.GetColumnName(), RuleID: rule.ID, RuleParams: input.GetRuleParams(), Status: int32(input.GetStatus()), Remark: input.GetRemark(), CreatedBy: oldItem.CreatedBy, UpdatedBy: authInfo.UserId, CreatedAt: oldItem.CreatedAt}
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
	if req.GetStatus() == commonv1.Status_STATUS_ENABLE {
		_, err = c.validateStorageForm(ctx, &adminv1.BaseRedactStoragePolicyForm{
			Id:         item.ID,
			TenantId:   item.TenantID,
			SourceName: item.SourceName,
			TableName:  item.TableName_,
			ColumnName: item.ColumnName,
			RuleId:     item.RuleID,
			RuleParams: item.RuleParams,
			Status:     req.GetStatus(),
			Remark:     item.Remark,
		})
		if err != nil {
			return err
		}
	}
	if req.GetStatus() == commonv1.Status_STATUS_DISABLE {
		err = c.ensureNoStoredValues(ctx, []int64{item.ID})
		if err != nil {
			return err
		}
	}
	item.Status = int32(req.GetStatus())
	err = c.UpdateByID(ctx, item)
	if err != nil {
		return err
	}
	return redact.RefreshRedactRuntime(ctx, c.resolver)
}

// ListBaseRedactStorageTable 查询包含租户ID字段的数据表列表。
func (c *BaseRedactStoragePolicyCase) ListBaseRedactStorageTable(ctx context.Context, req *adminv1.ListBaseRedactStorageTableRequest) (*adminv1.ListBaseRedactStorageTableResponse, error) {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	client, err := GormClientBySourceName(c.BaseCase, req.GetSourceName())
	if err != nil {
		return nil, errorsx.InvalidArgument("请选择已初始化的数据源").WithCause(err)
	}
	var rows []dbmeta.Table
	rows, err = dbmeta.ListTables(ctx, client.DB, client.Driver(), dbmeta.TableScope{RequireColumn: "tenant_id"})
	if err != nil {
		return nil, errorsx.Internal("查询包含租户ID的数据表失败").WithCause(err)
	}
	tables := make([]*adminv1.BaseRedactStorageTable, 0, len(rows))
	for _, row := range rows {
		comment := row.TableComment
		if comment == "" {
			comment = row.TableName
		}
		tables = append(tables, &adminv1.BaseRedactStorageTable{Name: row.TableName, Comment: comment})
	}
	return &adminv1.ListBaseRedactStorageTableResponse{Tables: tables}, nil
}

// ListBaseRedactStorageColumn 查询可入库脱敏且不属于主键、唯一索引的字符串字段。
func (c *BaseRedactStoragePolicyCase) ListBaseRedactStorageColumn(ctx context.Context, req *adminv1.ListBaseRedactStorageColumnRequest) (*adminv1.ListBaseRedactStorageColumnResponse, error) {
	err := redact.EnsureRedactPlatformOperator(ctx, c.BaseCase)
	if err != nil {
		return nil, err
	}
	client, err := GormClientBySourceName(c.BaseCase, req.GetSourceName())
	if err != nil {
		return nil, errorsx.InvalidArgument("请选择已初始化的数据源").WithCause(err)
	}
	if !client.Migrator().HasTable(req.GetTableName()) {
		return nil, errorsx.ResourceNotFound("数据库表不存在")
	}
	if !client.Migrator().HasColumn(req.GetTableName(), "tenant_id") {
		return nil, errorsx.InvalidArgument("只能选择包含租户ID的数据表")
	}
	var metadata []dbmeta.StorageColumn
	metadata, err = dbmeta.ListStorageColumns(ctx, client.DB, client.Driver(), req.GetTableName())
	if err != nil {
		return nil, errorsx.Internal("查询数据库字段失败").WithCause(err)
	}
	columns := make([]*adminv1.BaseRedactStorageColumn, 0, len(metadata))
	for _, column := range metadata {
		if column.IsKey || !redact.IsRedactStringDatabaseType(column.DataType) {
			continue
		}
		columns = append(columns, &adminv1.BaseRedactStorageColumn{
			Name:       column.Name,
			Comment:    cmp.Or(column.Comment, column.Name),
			DbType:     column.DataType,
			ColumnType: column.ColumnType,
		})
	}
	return &adminv1.ListBaseRedactStorageColumnResponse{Columns: columns}, nil
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

// validateStorageForm 校验入库策略字段不属于主键或唯一索引，并检查规则参数。
func (c *BaseRedactStoragePolicyCase) validateStorageForm(ctx context.Context, input *adminv1.BaseRedactStoragePolicyForm) (*models.BaseRedactRule, error) {
	if input.GetTenantId() <= 0 {
		return nil, errorsx.InvalidArgument("请选择租户")
	}
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
	var rule *models.BaseRedactRule
	rule, err = c.ruleRepo.FindByID(ctx, input.GetRuleId())
	if err != nil {
		return nil, errorsx.ResourceNotFound("脱敏规则不存在").WithCause(err)
	}
	if rule.Status != _const.STATUS_STATUS_ENABLE {
		return nil, errorsx.InvalidArgument("脱敏规则已停用")
	}
	if !client.Migrator().HasColumn(input.GetTableName(), "tenant_id") && !strings.EqualFold(rule.RuleType, "ENCRYPT") {
		return nil, errorsx.InvalidArgument("只能选择包含租户ID的数据表")
	}
	var metadata []dbmeta.StorageColumn
	metadata, err = dbmeta.ListStorageColumns(ctx, client.DB, client.Driver(), input.GetTableName())
	if err != nil {
		return nil, errorsx.Internal("查询数据库字段约束失败").WithCause(err)
	}
	index := slices.IndexFunc(metadata, func(column dbmeta.StorageColumn) bool {
		return strings.EqualFold(column.Name, input.GetColumnName())
	})
	if index < 0 {
		return nil, errorsx.InvalidArgument("数据库字段不存在")
	}
	column := metadata[index]
	if !redact.IsRedactStringDatabaseType(column.DataType) {
		return nil, errorsx.InvalidArgument("只有字符串字段支持入库脱敏")
	}
	if column.IsKey {
		return nil, errorsx.InvalidArgument("唯一索引字段不支持入库脱敏")
	}
	err = kit.ValidateRedactRule(rule.Code, rule.RuleType, input.GetRuleParams())
	if err != nil {
		return nil, errorsx.InvalidArgument("入库脱敏规则参数无效").WithCause(err)
	}
	status := input.GetStatus()
	if status != commonv1.Status_STATUS_UNSPECIFIED && status != commonv1.Status_STATUS_ENABLE && status != commonv1.Status_STATUS_DISABLE {
		return nil, errorsx.InvalidArgument("入库脱敏策略状态无效")
	}
	return rule, nil
}

// toBaseRedactStoragePolicy 转换入库脱敏策略列表项并填充数据表注释。
func (c *BaseRedactStoragePolicyCase) toBaseRedactStoragePolicy(ctx context.Context, item *models.BaseRedactStoragePolicy, comments map[string]string) (*adminv1.BaseRedactStoragePolicy, error) {
	rule, err := c.ruleRepo.FindByID(ctx, item.RuleID)
	if err != nil {
		return nil, err
	}
	return &adminv1.BaseRedactStoragePolicy{Id: item.ID, TenantId: item.TenantID, SourceName: item.SourceName, TableName: item.TableName_, TableComment: comments[item.SourceName+"\x00"+item.TableName_], ColumnName: item.ColumnName, RuleId: item.RuleID, RuleCode: rule.Code, RuleName: rule.Name, RuleType: rule.RuleType, RuleParams: item.RuleParams, Status: commonv1.Status(item.Status), Remark: item.Remark, CreatedAt: item.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: item.UpdatedAt.Format("2006-01-02 15:04:05")}, nil
}
