package admin

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	queryCompiler "github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/query"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/errorsx"
	"gorm.io/gorm/clause"
)

// GetBiCard 返回可见卡片，只有卡片所有者且有模型管理权限才得到公式和 SQL。
func (c *BiCardCase) GetBiCard(ctx context.Context, req *adminv1.GetBiCardRequest) (*adminv1.BiCard, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var item *models.BiCard
	item, err = c.findCard(ctx, authInfo, req.GetId(), false)
	if err != nil {
		return nil, err
	}
	var result *adminv1.BiCard
	result, err = toBiCard(item, authInfo.UserId)
	if err != nil || c.biQueryCase.biSourceCase.registry.Has(item.SourceCode) {
		return result, err
	}
	var source *models.BiSource
	var model dto.ConfiguredSourceSchema
	source, model, err = c.biQueryCase.biSourceCase.findEnabledSource(ctx, item.TenantID, item.SourceCode)
	if err != nil {
		return nil, err
	}
	result.SourceVersion = source.Version
	if !result.CanEdit {
		return result, nil
	}
	err = c.biQueryCase.biSourceCase.requireModelPermission(ctx)
	if errors.Code(err) == 403 {
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	result.CanEditMetrics = true
	for _, metric := range model.Measures {
		if slices.ContainsFunc(result.Query.Measures, func(selected *adminv1.BiMeasureSpec) bool { return selected.Code == metric.Code }) {
			result.MetricDefinitions = append(result.MetricDefinitions, toProtoSourceMeasure(metric))
		}
	}
	var execution *dto.CardExecution
	execution, err = decodeExecution(item.ExecutionJSON)
	if err != nil {
		return nil, err
	}
	result.Sql = execution.SQL
	return result, nil
}

// PreviewBiCardSql 将公式候选仅覆盖到内存模型，以当前访问者执行并冻结预览来源。
func (c *BiCardCase) PreviewBiCardSql(ctx context.Context, req *adminv1.PreviewBiCardSqlRequest) (*adminv1.PreviewBiCardSqlResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var projectID int64
	var card *models.BiCard
	if req.GetId() > 0 {
		card, err = c.findCard(ctx, authInfo, req.GetId(), true)
		if err != nil {
			return nil, err
		}
		if card.Version != req.GetExpectedCardVersion() || card.QueryRecordID != req.GetQueryRecordId() {
			return nil, errorsx.Conflict("卡片版本或创建来源已变化，请刷新")
		}
		projectID = card.ProjectID
	} else {
		projectID, err = projectIDForAuth(authInfo)
		if err != nil {
			return nil, err
		}
	}
	scope := bi.Scope{TenantID: authInfo.TenantId, ProjectID: projectID, UserID: authInfo.UserId, Terminal: "admin"}
	var spec bi.QuerySpec
	spec, _, err = c.biQueryCase.validate(ctx, scope, fromProtoQuerySpec(req.GetQuery()))
	if err != nil {
		return nil, err
	}
	var origin *models.BiQuery
	origin, err = c.biQueryCase.cardSourceRecord(ctx, scope, req.GetQueryRecordId(), spec)
	if err != nil {
		return nil, err
	}
	if card != nil && !biQuerySpecMatches(card.QueryJSON, spec) {
		return nil, errorsx.InvalidArgument("查询条件已变化，请重新问数后新建卡片")
	}
	sourceCase := c.biQueryCase.biSourceCase
	var execution *dto.CardExecution
	execution, err = sourceCase.compile(ctx, scope, spec)
	if err != nil {
		return nil, err
	}
	if card == nil && origin.QueryHash != execution.Hash {
		return nil, errorsx.Conflict("原问数的公共指标已更新，请重新问数")
	}
	preview := &dto.CardPreviewContext{CardID: req.GetId(), CardVersion: req.GetExpectedCardVersion(), QueryRecordID: origin.ID}
	response := &adminv1.PreviewBiCardSqlResponse{Impact: &adminv1.BiModelImpact{}}
	if sourceCase.registry.Has(spec.SourceCode) {
		if len(req.GetMeasureChanges()) > 0 {
			return nil, errorsx.PermissionDenied("代码注册数据源的公共指标由宿主代码维护")
		}
	} else {
		var source *models.BiSource
		var model dto.ConfiguredSourceSchema
		source, model, err = sourceCase.findEnabledSource(ctx, scope.TenantID, spec.SourceCode)
		if err != nil {
			return nil, err
		}
		if source.Version != req.GetExpectedSourceVersion() {
			return nil, errorsx.Conflict("数据源版本已变化，请刷新后重新预览")
		}
		preview.SourceVersion = source.Version
		response.SourceVersion = source.Version
		err = sourceCase.requireModelPermission(ctx)
		if err != nil && (errors.Code(err) != 403 || len(req.GetMeasureChanges()) > 0) {
			return nil, err
		}
		response.CanEditMetrics = err == nil
		err = nil
		var candidate dto.ConfiguredSourceSchema
		candidate, err = cardCandidateModel(model, spec, req.GetMeasureChanges(), source.Version+1)
		if err != nil {
			return nil, err
		}
		for _, metric := range candidate.Measures {
			if response.CanEditMetrics && slices.ContainsFunc(spec.Measures, func(selected bi.MeasureSpec) bool { return selected.Code == metric.Code }) {
				response.MetricDefinitions = append(response.MetricDefinitions, toProtoSourceMeasure(metric))
			}
			for _, before := range model.Measures {
				if metric.Code == before.Code && metric.Revision != before.Revision {
					preview.Changes = append(preview.Changes, metric)
					preview.Before = append(preview.Before, before)
				}
			}
		}
		if len(preview.Changes) > 0 {
			response.Impact, err = sourceCase.synchronizeModel(ctx, source, candidate, req.GetId(), false)
			if err != nil {
				return nil, err
			}
			if req.GetId() == 0 {
				response.Impact.AffectedCards++
			}
			var driver string
			driver, err = sourceCase.driverForSource(source)
			if err != nil {
				return nil, err
			}
			execution, err = queryCompiler.Compile(driver, source.MaxRows, candidate, spec)
			if err != nil {
				return nil, err
			}
		}
		if response.CanEditMetrics {
			response.Sql = execution.SQL
		}
	}
	preview.CandidateHash = cardCandidateHash(execution, preview.Changes)
	var result bi.QueryResult
	result, err = c.biQueryCase.execute(ctx, scope, spec, &dto.BiQueryRecordContext{Kind: dto.QueryKindCardSQLPreview, CardID: req.GetId(), Execution: execution, Preview: preview})
	if err != nil {
		return nil, err
	}
	if !sourceCase.registry.Has(spec.SourceCode) {
		var source *models.BiSource
		source, _, err = sourceCase.findEnabledSource(ctx, scope.TenantID, spec.SourceCode)
		if err != nil {
			return nil, err
		}
		if source.Version != preview.SourceVersion {
			return nil, errorsx.Conflict("预览期间公共模型已变化，请重新预览")
		}
	}
	response.PreviewQueryRecordId = result.QueryRecordID
	response.Result = toProtoQueryResult(result)
	return response, nil
}

// lockCardRecord 原子锁定当前账号及项目的预览或问数记录，消费状态不能由客户端声明。
func (c *BiCardCase) lockCardRecord(ctx context.Context, scope bi.Scope, id int64) (*models.BiQuery, error) {
	query := c.biQueryCase.biQueryRepo.Query(ctx).BiQuery
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id), query.TenantID.Eq(scope.TenantID), query.UserID.Eq(scope.UserID), query.ProjectID.Eq(scope.ProjectID)), repository.Clauses(clause.Locking{Strength: "UPDATE"})}
	record, err := c.biQueryCase.biQueryRepo.Find(ctx, opts...)
	if err != nil {
		return nil, errorsx.InvalidArgument("请选择当前账号和项目的成功预览")
	}
	if record.AppliedCardID > 0 {
		return nil, errorsx.Conflict("预览已保存，请重新读取卡片").WithMetadata(map[string]string{"card_id": strconv.FormatInt(record.AppliedCardID, 10), "card_version": strconv.FormatInt(record.AppliedCardVersion, 10)})
	}
	if record.Status != int32(adminv1.BiQueryStatus_BI_QUERY_STATUS_SUCCESS) {
		return nil, errorsx.InvalidArgument("预览尚未成功")
	}
	return record, nil
}

// cardCandidateModel 校验仅编辑选中指标的候选，规范化后计算单调修订。
func cardCandidateModel(model dto.ConfiguredSourceSchema, spec bi.QuerySpec, changes []*adminv1.BiMetricChange, version int32) (dto.ConfiguredSourceSchema, error) {
	candidate := model
	candidate.Measures = slices.Clone(model.Measures)
	seen := make(map[string]bool, len(changes))
	fields := model.Tables[findModelTable(model.Tables, model.BaseTable)].Fields
	for _, change := range changes {
		if seen[change.GetMetricCode()] || !slices.ContainsFunc(spec.Measures, func(item bi.MeasureSpec) bool { return item.Code == change.GetMetricCode() }) {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("只能修改当前卡片已选择的公共指标且不能重复")
		}
		seen[change.GetMetricCode()] = true
		index := slices.IndexFunc(candidate.Measures, func(metric dto.ConfiguredSourceMeasure) bool { return metric.Code == change.GetMetricCode() })
		if index < 0 || candidate.Measures[index].Revision != change.GetExpectedRevision() {
			return dto.ConfiguredSourceSchema{}, errorsx.Conflict("公共指标版本已变化，请刷新后重新预览")
		}
		if change.GetName() == "" || strings.TrimSpace(change.GetName()) != change.GetName() || len([]rune(change.GetName())) > 128 || len([]rune(change.GetUnit())) > 32 {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("公共指标名称或单位无效")
		}
		expression, err := queryCompiler.Parse(change.GetExpression(), fields)
		if err != nil {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("公共指标公式无效").WithCause(err)
		}
		candidate.Measures[index].Expression = expression.Canonical()
		candidate.Measures[index].Name = change.GetName()
		candidate.Measures[index].Unit = change.GetUnit()
	}
	names := make(map[string]bool, len(candidate.Measures))
	for _, metric := range candidate.Measures {
		if names[metric.Name] {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("同一数据源的公共指标名称不能重复")
		}
		names[metric.Name] = true
	}
	return reviseSourceModel(model, candidate, version), nil
}

// cardCandidateHash 绑定规范化公式候选及完整编译结果，不绑定展示样式。
func cardCandidateHash(execution *dto.CardExecution, changes []dto.ConfiguredSourceMeasure) string {
	payload, _ := json.Marshal(changes) // 指标定义只含 JSON 原生值。
	return hashSchema(execution.Hash + string(payload))
}
