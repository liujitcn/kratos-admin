package admin

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	queryCompiler "github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/query"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth/data"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PreviewBiSourceModel 发现并校验候选模型，静态分析全部引用而不提前更新公共定义。
func (c *BiSourceCase) PreviewBiSourceModel(ctx context.Context, req *adminv1.SaveBiSourceModelRequest) (*adminv1.BiModelImpact, error) {
	item, model, err := c.prepareModel(ctx, req)
	if err != nil {
		return nil, err
	}
	return c.synchronizeModel(ctx, item, model, 0, false)
}

// prepareModel 在元数据事务外发现业务表结构，校验模型管理权限和预期版本。
func (c *BiSourceCase) prepareModel(ctx context.Context, req *adminv1.SaveBiSourceModelRequest) (*models.BiSource, dto.ConfiguredSourceSchema, error) {
	err := c.requireModelPermission(ctx)
	if err != nil {
		return nil, dto.ConfiguredSourceSchema{}, err
	}
	var authInfo *data.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
	if err != nil {
		return nil, dto.ConfiguredSourceSchema{}, err
	}
	var item *models.BiSource
	item, err = c.findSource(ctx, authInfo.TenantId, req.GetId())
	if err != nil {
		return nil, dto.ConfiguredSourceSchema{}, err
	}
	if item.Version != req.GetExpectedSourceVersion() {
		return nil, dto.ConfiguredSourceSchema{}, errorsx.Conflict("数据源版本已变化，请刷新后重新预览")
	}
	var database *configuredDatabase
	database, err = c.openSource(ctx, item)
	if err != nil {
		return nil, dto.ConfiguredSourceSchema{}, err
	}
	defer database.Close()
	var tables []*adminv1.BiSourceSchemaTable
	tables, err = discoverSourceTables(ctx, database, item)
	if err != nil {
		return nil, dto.ConfiguredSourceSchema{}, err
	}
	var candidate dto.ConfiguredSourceSchema
	candidate, err = configuredSourceModel(req.GetModel(), tables)
	if err != nil {
		return nil, dto.ConfiguredSourceSchema{}, err
	}
	if database.driver != "mysql" && database.driver != "postgres" && database.driver != "tidb" && slices.ContainsFunc(candidate.Dimensions, func(d dto.ConfiguredSourceDimension) bool { return d.Hierarchy != nil }) {
		return nil, dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("当前 BI 数据源不支持层级维度")
	}
	if candidate.ReportTimezone != "" {
		var restore func()
		_, restore, err = database.reportConnection(ctx, candidate.ReportTimezone)
		if err != nil {
			return nil, dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("BI 数据库不支持此报表时区").WithCause(err)
		}
		restore()
	}
	var old dto.ConfiguredSourceSchema
	if err = json.Unmarshal([]byte(item.SchemaJSON), &old); err != nil {
		return nil, dto.ConfiguredSourceSchema{}, errorsx.Internal("读取数据源模型失败").WithCause(err)
	}
	return item, reviseSourceModel(old, candidate, item.Version+1), nil
}

// lockSource 按稳定源标识取得行锁，防止卡片、模型和发布并发覆盖。
func (c *BiSourceCase) lockSource(ctx context.Context, tenantID int64, code string) (*models.BiSource, error) {
	query := c.sourceRepo.Query(ctx).BiSource
	opts := []repository.QueryOption{repository.Where(query.TenantID.Eq(tenantID), query.Code.Eq(code)), repository.Clauses(clause.Locking{Strength: "UPDATE"})}
	item, err := c.sourceRepo.Find(ctx, opts...)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ResourceNotFound("BI 数据源不存在")
	}
	return item, err
}

// synchronizeModel 用同一编译路径分析或事务内同步同源全部卡片，只修改受信派生字段。
func (c *BiSourceCase) synchronizeModel(ctx context.Context, source *models.BiSource, candidate dto.ConfiguredSourceSchema, editingCardID int64, write bool) (*adminv1.BiModelImpact, error) {
	var old dto.ConfiguredSourceSchema
	err := json.Unmarshal([]byte(source.SchemaJSON), &old)
	if err != nil {
		return nil, errorsx.Internal("读取公共指标失败").WithCause(err)
	}
	impact := modelImpact(old, candidate, source.Version+1)
	var driver string
	driver, err = c.driverForSource(source)
	if err != nil {
		return nil, err
	}
	query := c.cardRepo.Query(ctx).BiCard
	opts := []repository.QueryOption{repository.Where(query.TenantID.Eq(source.TenantID), query.SourceCode.Eq(source.Code)), repository.Order(query.ID.Asc())}
	if write {
		opts = append(opts, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
	}
	var cards []*models.BiCard
	cards, err = c.cardRepo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	updates := make([]*models.BiCard, 0, len(cards))
	for _, card := range cards {
		var spec bi.QuerySpec
		if err = json.Unmarshal([]byte(card.QueryJSON), &spec); err != nil {
			return nil, errorsx.Conflict("关联卡片查询配置无效，不能保存公共模型")
		}
		var execution *dto.CardExecution
		execution, err = queryCompiler.Compile(driver, source.MaxRows, candidate, spec)
		if err != nil {
			return nil, errorsx.Conflict("公共模型无法满足关联卡片，请先解除引用").WithCause(err)
		}
		if _, err = validateCardChart(card.ChartType, card.ChartJSON, spec, bi.QueryResult{Columns: execution.Columns}); err != nil {
			return nil, errorsx.Conflict("公共模型与关联卡片图表字段不兼容").WithCause(err)
		}
		var current dto.CardExecution
		if err = json.Unmarshal([]byte(card.ExecutionJSON), &current); err != nil {
			return nil, errorsx.Conflict("关联卡片执行定义无效")
		}
		if current.Hash == execution.Hash {
			continue
		}
		impact.AffectedCards++
		if card.ID == editingCardID {
			continue
		}
		var payload []byte
		payload, err = json.Marshal(execution)
		if err != nil {
			return nil, err
		}
		card.ExecutionJSON = string(payload)
		card.Version++
		card.UpdatedAt = time.Now()
		updates = append(updates, card)
	}
	impact.AffectedDashboards, err = c.affectedPublishedDashboards(ctx, source, candidate)
	if err != nil {
		return nil, err
	}
	if write {
		for _, card := range updates {
			err = c.cardRepo.Update(ctx, card, repository.Where(query.ID.Eq(card.ID), query.TenantID.Eq(source.TenantID)), repository.Select(query.ExecutionJSON, query.Version, query.UpdatedAt))
			if err != nil {
				return nil, err
			}
		}
	}
	return impact, nil
}

// affectedPublishedDashboards 检查当前发布指针的指标引用，只返回总数，不暴露私人卡片内容。
func (c *BiSourceCase) affectedPublishedDashboards(ctx context.Context, source *models.BiSource, candidate dto.ConfiguredSourceSchema) (int32, error) {
	query := c.dashboardRepo.Query(ctx).BiDashboard
	opts := []repository.QueryOption{repository.Where(query.TenantID.Eq(source.TenantID), query.PublishedVersion.Gt(0)), repository.Order(query.ID.Asc())}
	items, err := c.dashboardRepo.List(ctx, opts...)
	if err != nil {
		return 0, err
	}
	var count int32
	for _, item := range items {
		versionQuery := c.versionRepo.Query(ctx).BiDashboardVersion
		var version *models.BiDashboardVersion
		version, err = c.versionRepo.Find(ctx, repository.Where(versionQuery.TenantID.Eq(source.TenantID), versionQuery.DashboardID.Eq(item.ID), versionQuery.Version.Eq(item.PublishedVersion)))
		if err != nil {
			return 0, err
		}
		var layout dto.DashboardSchema
		if err = json.Unmarshal([]byte(version.SchemaJSON), &layout); err != nil {
			return 0, errorsx.Conflict("当前发布快照无效，不能保存公共模型")
		}
		changed := false
		for _, placed := range layout.Cards {
			if placed.Query == nil || placed.Query.SourceCode != source.Code {
				continue
			}
			for _, selected := range placed.Query.Measures {
				if !slices.ContainsFunc(candidate.Measures, func(metric dto.ConfiguredSourceMeasure) bool { return metric.Code == selected.Code }) {
					return 0, errorsx.Conflict("公共指标仍被当前发布版本引用，请先解除引用")
				}
			}
			if !executionMatchesModel(placed.Execution, candidate) {
				changed = true
			}
		}
		if changed {
			count++
		}
	}
	return count, nil
}

// requireUnusedSource 防止删除或更换连接使现存卡片和当前发布快照失去依赖。
func (c *BiSourceCase) requireUnusedSource(ctx context.Context, source *models.BiSource) error {
	query := c.cardRepo.Query(ctx).BiCard
	count, err := c.cardRepo.Count(ctx, repository.Where(query.TenantID.Eq(source.TenantID), query.SourceCode.Eq(source.Code)))
	if err != nil {
		return err
	}
	if count > 0 {
		return errorsx.Conflict("数据源仍被卡片引用，请先解除引用")
	}
	_, err = c.affectedPublishedDashboards(ctx, source, dto.ConfiguredSourceSchema{})
	return err
}

// reviseSourceModel 使指标身份沿用单调源版本，画布坐标不影响查询结构修订。
func reviseSourceModel(old, candidate dto.ConfiguredSourceSchema, version int32) dto.ConfiguredSourceSchema {
	candidate.Measures = slices.Clone(candidate.Measures)
	for index, metric := range candidate.Measures {
		metric.Revision = int64(version)
		for _, previous := range old.Measures {
			if previous.Code == metric.Code && previous.Name == metric.Name && previous.Type == metric.Type && previous.Expression == metric.Expression && previous.Unit == metric.Unit {
				metric.Revision = previous.Revision
				break
			}
		}
		candidate.Measures[index] = metric
	}
	candidate.ModelRevision = old.ModelRevision
	if sourceStructureHash(old) != sourceStructureHash(candidate) {
		candidate.ModelRevision++
	}
	return candidate
}

// sourceStructureHash 排除指标、修订和画布位置，只比较影响查询结构与元数据的内容。
func sourceStructureHash(model dto.ConfiguredSourceSchema) string {
	model.ModelRevision = 0
	model.Measures = nil
	model.Tables = slices.Clone(model.Tables)
	for index := range model.Tables {
		model.Tables[index].X, model.Tables[index].Y = 0, 0
	}
	payload, _ := json.Marshal(model) // 结构只包含 JSON 原生值，不存在无法序列化的字段。
	return hashSchema(string(payload))
}

// modelImpact 计算公共定义差异，影响资源数量由仓储分析补充。
func modelImpact(old, candidate dto.ConfiguredSourceSchema, version int32) *adminv1.BiModelImpact {
	impact := &adminv1.BiModelImpact{SourceVersion: version}
	for _, metric := range candidate.Measures {
		var previous *adminv1.BiSourceMeasure
		for _, before := range old.Measures {
			if before.Code == metric.Code {
				previous = toProtoSourceMeasure(before)
				break
			}
		}
		if previous == nil || previous.Revision != metric.Revision {
			impact.Changes = append(impact.Changes, &adminv1.BiMetricDifference{Before: previous, After: toProtoSourceMeasure(metric)})
		}
	}
	for _, metric := range old.Measures {
		if !slices.ContainsFunc(candidate.Measures, func(item dto.ConfiguredSourceMeasure) bool { return item.Code == metric.Code }) {
			impact.Changes = append(impact.Changes, &adminv1.BiMetricDifference{Before: toProtoSourceMeasure(metric)})
		}
	}
	return impact
}

// executionMatchesModel 比较实际查询结构及所选指标身份，不用连接密码版本判断过期。
func executionMatchesModel(execution *dto.CardExecution, model dto.ConfiguredSourceSchema) bool {
	if execution == nil || execution.ModelRevision != model.ModelRevision {
		return false
	}
	for _, selected := range execution.Metrics {
		if !slices.ContainsFunc(model.Measures, func(metric dto.ConfiguredSourceMeasure) bool {
			return metric.Code == selected.Code && metric.Revision == selected.Revision
		}) {
			return false
		}
	}
	return true
}

// toProtoSourceMeasure 转换有权管理源模型的指标定义。
func toProtoSourceMeasure(metric dto.ConfiguredSourceMeasure) *adminv1.BiSourceMeasure {
	return &adminv1.BiSourceMeasure{Code: metric.Code, Name: metric.Name, Type: string(metric.Type), Expression: metric.Expression, Unit: metric.Unit, Revision: metric.Revision}
}
