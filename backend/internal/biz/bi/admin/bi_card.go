package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	dataAccess "github.com/liujitcn/kratos-admin/backend/internal/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

// biCardChartTypes 与接口校验一致，供问数建议和卡片保存共同使用。
var biCardChartTypes = []string{"bar", "line", "area", "pie", "donut", "gauge", "metric", "scatter", "radar", "funnel", "treemap", "sunburst", "sankey", "heatmap"}

// BiCardCase 管理语义数据卡片、收藏和个人工作台。
type BiCardCase struct {
	*biz.BaseCase
	cardRepo      *data.BiCardRepository
	workbenchRepo *data.BiWorkbenchRepository
	biQueryCase   *BiQueryCase
	dashboardCase *BiDashboardCase
}

// NewBiCardCase 创建数据卡片业务实例。
func NewBiCardCase(baseCase *biz.BaseCase, cardRepo *data.BiCardRepository, workbenchRepo *data.BiWorkbenchRepository, biQueryCase *BiQueryCase, dashboardCase *BiDashboardCase) *BiCardCase {
	return &BiCardCase{BaseCase: baseCase, cardRepo: cardRepo, workbenchRepo: workbenchRepo, biQueryCase: biQueryCase, dashboardCase: dashboardCase}
}

// PageBiCard 仅分页返回系统内置、自己创建或当前项目共享的卡片。
func (c *BiCardCase) PageBiCard(ctx context.Context, req *adminv1.PageBiCardRequest) (*adminv1.PageBiCardResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var config *dto.WorkbenchConfig
	config, err = c.loadWorkbench(ctx, authInfo)
	if err != nil {
		return nil, err
	}
	var counts *adminv1.BiCardCounts
	counts, err = c.countCards(ctx, authInfo, config)
	if err != nil {
		return nil, err
	}
	query := c.cardRepo.Query(ctx).BiCard
	opts := c.visibleCardOptions(ctx, authInfo)
	if req.GetCategory() == "system" {
		opts = append(opts, repository.Order(query.ID.Asc()))
	} else {
		opts = append(opts, repository.Order(query.UpdatedAt.Desc(), query.ID.Desc()))
	}
	if req.GetName() != "" {
		opts = append(opts, repository.Where(field.Or(query.Name.Like("%"+req.GetName()+"%"), query.SourceCode.Like("%"+req.GetName()+"%"), query.SourceCode.In(req.GetSearchSourceCodes()...))))
	}
	if req.GetSourceCode() != "" {
		opts = append(opts, repository.Where(query.SourceCode.Eq(req.GetSourceCode())))
	}
	switch req.GetCategory() {
	case "system":
		opts = append(opts, repository.Where(query.OwnerID.Eq(0)))
	case "mine":
		opts = append(opts, repository.Where(query.OwnerID.Eq(authInfo.UserId)))
	case "shared":
		opts = append(opts, repository.Where(query.OwnerID.Neq(0), query.OwnerID.Neq(authInfo.UserId), query.IsShared.Eq(1)))
	case "favorite":
		if config == nil || len(config.FavoriteIDs) == 0 {
			return &adminv1.PageBiCardResponse{Counts: counts}, nil
		}
		opts = append(opts, repository.Where(query.ID.In(config.FavoriteIDs...)))
	}
	var items []*models.BiCard
	var total int64
	items, total, err = c.cardRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BiCard, 0, len(items))
	for _, item := range items {
		var card *adminv1.BiCard
		card, err = toBiCard(item, authInfo.UserId)
		if err != nil {
			return nil, err
		}
		result = append(result, card)
	}
	return &adminv1.PageBiCardResponse{Cards: result, Total: int32(total), Counts: counts}, nil
}

// visibleCardOptions 收敛租户、项目与卡片可见性，供列表和工作台共同使用。
func (c *BiCardCase) visibleCardOptions(ctx context.Context, authInfo *authData.UserTokenPayload) []repository.QueryOption {
	query := c.cardRepo.Query(ctx).BiCard
	opts := []repository.QueryOption{
		repository.Where(query.TenantID.Eq(authInfo.TenantId)),
		repository.Where(field.Or(query.OwnerID.Eq(0), query.OwnerID.Eq(authInfo.UserId), query.IsShared.Eq(1))),
	}
	if !hasAllProjectsScope(authInfo) {
		opts = append(opts, repository.Where(query.ProjectID.In(allowedProjectIDs(authInfo)...)))
	}
	return opts
}

// countCards 按同一可见范围计算分类数量，私有和跨项目卡片不计入结果。
func (c *BiCardCase) countCards(ctx context.Context, authInfo *authData.UserTokenPayload, config *dto.WorkbenchConfig) (*adminv1.BiCardCounts, error) {
	query := c.cardRepo.Query(ctx).BiCard
	opts := c.visibleCardOptions(ctx, authInfo)
	counts := &adminv1.BiCardCounts{}
	var err error
	counts.All, err = c.cardRepo.Count(ctx, opts...)
	if err != nil {
		return nil, err
	}
	counts.System, err = c.cardRepo.Count(ctx, append(slices.Clone(opts), repository.Where(query.OwnerID.Eq(0)))...)
	if err != nil {
		return nil, err
	}
	counts.Mine, err = c.cardRepo.Count(ctx, append(slices.Clone(opts), repository.Where(query.OwnerID.Eq(authInfo.UserId)))...)
	if err != nil {
		return nil, err
	}
	counts.Shared, err = c.cardRepo.Count(ctx, append(slices.Clone(opts), repository.Where(query.OwnerID.Neq(0), query.OwnerID.Neq(authInfo.UserId), query.IsShared.Eq(1)))...)
	if err != nil {
		return nil, err
	}
	if config != nil && len(config.FavoriteIDs) > 0 {
		counts.Favorite, err = c.cardRepo.Count(ctx, append(slices.Clone(opts), repository.Where(query.ID.In(config.FavoriteIDs...)))...)
		if err != nil {
			return nil, err
		}
	}
	return counts, nil
}

// loadWorkbench 读取当前用户配置，缺少记录时返回未配置状态。
func (c *BiCardCase) loadWorkbench(ctx context.Context, authInfo *authData.UserTokenPayload) (*dto.WorkbenchConfig, error) {
	query := c.workbenchRepo.Query(ctx).BiWorkbench
	opts := []repository.QueryOption{repository.Where(query.TenantID.Eq(authInfo.TenantId), query.UserID.Eq(authInfo.UserId))}
	item, err := c.workbenchRepo.Find(ctx, opts...)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var config dto.WorkbenchConfig
	if err = json.Unmarshal([]byte(item.ConfigJSON), &config); err != nil {
		return nil, errorsx.Internal("读取个人工作台失败").WithCause(err)
	}
	return &config, nil
}

// SaveBiCard 校验语义查询并保存卡片，所有者和项目均由服务端确定。
func (c *BiCardCase) SaveBiCard(ctx context.Context, req *adminv1.SaveBiCardRequest) (*adminv1.BiCard, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	form := req.GetCard()
	now := time.Now()
	item := &models.BiCard{TenantID: authInfo.TenantId, OwnerID: authInfo.UserId, CreatedBy: authInfo.UserId, CreatedAt: now}
	if req.GetId() > 0 {
		item, err = c.findCard(ctx, authInfo, req.GetId(), true)
		if err != nil {
			return nil, err
		}
	} else {
		item.ProjectID, err = projectIDForAuth(authInfo)
		if err != nil {
			return nil, err
		}
	}
	if err = validateBiCardForm(form); err != nil {
		return nil, err
	}
	scope := bi.Scope{TenantID: authInfo.TenantId, ProjectID: item.ProjectID, UserID: authInfo.UserId, Terminal: "admin"}
	var spec bi.QuerySpec
	spec, _, err = c.biQueryCase.validate(ctx, scope, fromProtoQuerySpec(form.GetQuery()))
	if err != nil {
		return nil, err
	}
	var record *models.BiQuery
	record, err = c.biQueryCase.cardSourceRecord(ctx, scope, form.GetQueryRecordId(), spec)
	if err != nil {
		return nil, err
	}
	var result bi.QueryResult
	if err = json.Unmarshal([]byte(record.ResultSnapshot), &result); err != nil {
		return nil, errorsx.Internal("读取来源查询结果失败").WithCause(err)
	}
	var chartJSON string
	chartJSON, err = validateCardChart(form.GetChartType(), form.GetChartJson(), spec, result)
	if err != nil {
		return nil, err
	}
	item.QueryRecordID = record.ID
	item.ChartJSON = chartJSON
	var queryJSON []byte
	queryJSON, err = json.Marshal(spec)
	if err != nil {
		return nil, errorsx.Internal("保存卡片查询配置失败").WithCause(err)
	}
	item.Name = form.GetName()
	item.SourceCode = spec.SourceCode
	item.QueryJSON = string(queryJSON)
	item.ChartType = form.GetChartType()
	item.RefreshSeconds = form.GetRefreshSeconds()
	item.IsShared = 0
	if form.GetIsShared() {
		item.IsShared = 1
	}
	item.UpdatedBy = authInfo.UserId
	item.UpdatedAt = now
	if item.ID == 0 {
		err = c.cardRepo.Create(ctx, item)
	} else {
		query := c.cardRepo.Query(ctx).BiCard
		opts := []repository.QueryOption{
			repository.Where(query.ID.Eq(item.ID), query.TenantID.Eq(authInfo.TenantId), query.OwnerID.Eq(authInfo.UserId)),
			repository.Select(query.Name, query.SourceCode, query.QueryJSON, query.QueryRecordID, query.ChartJSON, query.ChartType, query.RefreshSeconds, query.IsShared, query.UpdatedBy, query.UpdatedAt),
		}
		err = c.cardRepo.Update(ctx, item, opts...)
	}
	if err != nil {
		return nil, err
	}
	return toBiCard(item, authInfo.UserId)
}

// findCard 在取数、编辑和删除前检查卡片所属租户、项目与所有者。
func (c *BiCardCase) findCard(ctx context.Context, authInfo *authData.UserTokenPayload, id int64, editing bool) (*models.BiCard, error) {
	query := c.cardRepo.Query(ctx).BiCard
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id), query.TenantID.Eq(authInfo.TenantId))}
	item, err := c.cardRepo.Find(ctx, opts...)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ResourceNotFound("数据卡片不存在")
	}
	if err != nil {
		return nil, err
	}
	if !canAccessBiCard(authInfo, item, editing) {
		return nil, errorsx.PermissionDenied("无权访问数据卡片")
	}
	return item, nil
}

// DeleteBiCard 删除自己创建的卡片，不影响其他卡片与用户工作台配置。
func (c *BiCardCase) DeleteBiCard(ctx context.Context, req *adminv1.DeleteBiCardRequest) (*adminv1.DeleteBiCardResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var item *models.BiCard
	item, err = c.findCard(ctx, authInfo, req.GetId(), true)
	if err != nil {
		return nil, err
	}
	query := c.cardRepo.Query(ctx).BiCard
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(item.ID), query.TenantID.Eq(authInfo.TenantId), query.OwnerID.Eq(authInfo.UserId))}
	if err = c.cardRepo.Delete(ctx, opts...); err != nil {
		return nil, err
	}
	return &adminv1.DeleteBiCardResponse{}, nil
}

// GetBiCardData 按访问者的可信身份执行保存的语义查询，不复用分享者的数据结果。
func (c *BiCardCase) GetBiCardData(ctx context.Context, req *adminv1.GetBiCardDataRequest) (*adminv1.BiQueryResult, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var item *models.BiCard
	item, err = c.findCard(ctx, authInfo, req.GetId(), false)
	if err != nil {
		return nil, err
	}
	if !slices.Contains([]int32{0, 7, 30}, req.GetRangeDays()) {
		return nil, errorsx.InvalidArgument("仅支持卡片配置、近7日或近30日")
	}
	var spec bi.QuerySpec
	if err = json.Unmarshal([]byte(item.QueryJSON), &spec); err != nil {
		return nil, errorsx.Internal("读取卡片查询配置失败").WithCause(err)
	}
	scope := bi.Scope{TenantID: authInfo.TenantId, ProjectID: item.ProjectID, UserID: authInfo.UserId, Terminal: "admin"}
	var descriptor bi.SourceDescriptor
	spec, descriptor, err = c.biQueryCase.validate(ctx, scope, spec)
	if err != nil {
		return nil, err
	}
	spec = applyBiCardPeriod(spec, descriptor, req.GetRangeDays(), time.Now())
	var result bi.QueryResult
	result, err = c.biQueryCase.execute(ctx, scope, spec, nil)
	if err != nil {
		return nil, err
	}
	return toProtoQueryResult(result), nil
}

// GetBiWorkbench 返回当前用户收藏的仪表盘入口与卡片，未配置时保持为空。
func (c *BiCardCase) GetBiWorkbench(ctx context.Context, _ *adminv1.GetBiWorkbenchRequest) (*adminv1.BiWorkbench, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var config *dto.WorkbenchConfig
	config, err = c.loadWorkbench(ctx, authInfo)
	if err != nil {
		return nil, err
	}
	if config == nil {
		return &adminv1.BiWorkbench{}, nil
	}
	return c.buildWorkbench(ctx, authInfo, *config, false)
}

// buildWorkbench 过滤仪表盘入口和卡片收藏，保存时拒绝无权限的引用。
func (c *BiCardCase) buildWorkbench(ctx context.Context, authInfo *authData.UserTokenPayload, config dto.WorkbenchConfig, strict bool) (*adminv1.BiWorkbench, error) {
	result := &adminv1.BiWorkbench{}
	var err error
	for _, id := range config.DashboardIDs {
		var item *models.BiDashboard
		item, err = c.dashboardCase.findAccessibleDashboard(ctx, authInfo, id, biDashboardPermissionView)
		if err != nil && kratosErrors.Code(err) != 403 && kratosErrors.Code(err) != 404 {
			return nil, err
		}
		if err != nil || item.PublishedVersion == 0 {
			if strict {
				return nil, errorsx.PermissionDenied("工作台包含不可访问或未发布的仪表盘")
			}
			continue
		}
		var version *models.BiDashboardVersion
		version, err = c.dashboardCase.findVersion(ctx, item.ID, item.PublishedVersion)
		if err != nil {
			return nil, err
		}
		result.DashboardIds = append(result.DashboardIds, id)
		result.Dashboards = append(result.Dashboards, toBiDashboard(item, version.SchemaJSON))
	}
	if len(config.FavoriteIDs) > 0 {
		query := c.cardRepo.Query(ctx).BiCard
		opts := c.visibleCardOptions(ctx, authInfo)
		opts = append(opts, repository.Where(query.ID.In(config.FavoriteIDs...)))
		var items []*models.BiCard
		items, err = c.cardRepo.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		visible := make(map[int64]bool, len(items))
		for _, item := range items {
			visible[item.ID] = true
		}
		for _, id := range config.FavoriteIDs {
			if visible[id] {
				result.FavoriteIds = append(result.FavoriteIds, id)
			} else if strict {
				return nil, errorsx.PermissionDenied("收藏包含无权访问的卡片")
			}
		}
	}
	return result, nil
}

// SaveBiWorkbench 原子保存当前用户的仪表盘入口顺序与卡片收藏。
func (c *BiCardCase) SaveBiWorkbench(ctx context.Context, req *adminv1.SaveBiWorkbenchRequest) (*adminv1.BiWorkbench, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	config := dto.WorkbenchConfig{DashboardIDs: req.GetDashboardIds(), FavoriteIDs: req.GetFavoriteIds()}
	if err = validateWorkbenchConfig(config); err != nil {
		return nil, err
	}
	var result *adminv1.BiWorkbench
	result, err = c.buildWorkbench(ctx, authInfo, config, true)
	if err != nil {
		return nil, err
	}
	var configJSON []byte
	configJSON, err = json.Marshal(config)
	if err != nil {
		return nil, errorsx.Internal("保存个人工作台失败").WithCause(err)
	}
	now := time.Now()
	item := &models.BiWorkbench{TenantID: authInfo.TenantId, UserID: authInfo.UserId, ConfigJSON: string(configJSON), CreatedAt: now, UpdatedAt: now}
	if err = dataAccess.SaveBiWorkbench(ctx, c.workbenchRepo, item); err != nil {
		return nil, err
	}
	return result, nil
}

// applyBiCardPeriod 只覆盖卡片已指定的日期轴或时间范围，不改变非日期卡片和其他业务过滤。
func applyBiCardPeriod(spec bi.QuerySpec, descriptor bi.SourceDescriptor, days int32, now time.Time) bi.QuerySpec {
	if days == 0 {
		return spec
	}
	dateField := ""
	for _, dimension := range descriptor.Dimensions {
		if dimension.Type != bi.FieldTypeDate || !dimension.Filterable {
			continue
		}
		if spec.TimeRange != nil && spec.TimeRange.Field == dimension.Code {
			dateField = dimension.Code
			break
		}
		if spec.TimeRange == nil && slices.Contains(spec.Dimensions, dimension.Code) && dateField == "" {
			dateField = dimension.Code
		}
	}
	if dateField == "" {
		return spec
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	spec.TimeRange = &bi.TimeRangeSpec{Field: dateField, Start: today.AddDate(0, 0, 1-int(days)).Format("2006-01-02 15:04:05"), End: today.AddDate(0, 0, 1).Add(-time.Millisecond).Format("2006-01-02 15:04:05.999")}
	// 周期筛选替换同一日期上的旧条件，其他条件和可信 Scope 保持原值。
	spec.Filters = slices.DeleteFunc(slices.Clone(spec.Filters), func(filter bi.FilterSpec) bool { return filter.Field == dateField })
	return spec
}

// canAccessBiCard 校验共享只扩大卡片可见性，不扩大租户、项目或编辑权限。
func canAccessBiCard(authInfo *authData.UserTokenPayload, item *models.BiCard, editing bool) bool {
	if item.TenantID != authInfo.TenantId || !projectAllowed(authInfo, item.ProjectID) {
		return false
	}
	if editing {
		return item.OwnerID != 0 && item.OwnerID == authInfo.UserId
	}
	return item.OwnerID == 0 || item.OwnerID == authInfo.UserId || item.IsShared == 1
}

// validateBiCardForm 为进程内调用同步执行卡片配置边界校验。
func validateBiCardForm(form *adminv1.BiCardForm) error {
	if strings.TrimSpace(form.GetName()) == "" || len([]rune(form.GetName())) > 128 || form.GetQuery() == nil || form.GetQueryRecordId() <= 0 || len(form.GetChartJson()) > 16384 {
		return errorsx.InvalidArgument("卡片名称和查询配置无效")
	}
	if !slices.Contains(biCardChartTypes, form.GetChartType()) {
		return errorsx.InvalidArgument("不支持的卡片图表类型")
	}
	if form.GetChartType() == "metric" && len(form.GetQuery().GetDimensions()) > 0 {
		return errorsx.InvalidArgument("指标卡不能选择分组维度")
	}

	if seconds := form.GetRefreshSeconds(); seconds != 0 && (seconds < 30 || seconds > 86400) {
		return errorsx.InvalidArgument("刷新间隔应为0或30至86400秒")
	}
	return nil
}

// validateWorkbenchConfig 校验卡片数量、正数编号和唯一性，保留用户排列顺序。
func validateWorkbenchConfig(config dto.WorkbenchConfig) error {
	for _, ids := range [][]int64{config.DashboardIDs, config.FavoriteIDs} {
		if len(ids) > 50 {
			return errorsx.InvalidArgument("工作台和收藏分别最多50项")
		}
		seen := make(map[int64]struct{}, len(ids))
		for _, id := range ids {
			if _, exists := seen[id]; id <= 0 || exists {
				return errorsx.InvalidArgument("引用ID须为正数且不重复")
			}
			seen[id] = struct{}{}
		}
	}
	return nil
}

// toBiCard 装配卡片配置与当前用户的分类，不包含业务结果快照。
func toBiCard(item *models.BiCard, userID int64) (*adminv1.BiCard, error) {
	var spec bi.QuerySpec
	err := json.Unmarshal([]byte(item.QueryJSON), &spec)
	if err != nil {
		return nil, errorsx.Internal("读取卡片查询配置失败").WithCause(err)
	}
	category := "shared"
	if item.OwnerID == 0 {
		category = "system"
	} else if item.OwnerID == userID {
		category = "mine"
	}
	return &adminv1.BiCard{Id: item.ID, TenantId: item.TenantID, ProjectId: item.ProjectID, OwnerId: item.OwnerID, Name: item.Name, SourceCode: item.SourceCode, Query: toProtoQuerySpec(spec), ChartType: item.ChartType, ChartJson: item.ChartJSON, QueryRecordId: item.QueryRecordID, RefreshSeconds: item.RefreshSeconds, IsShared: item.IsShared == 1, UpdatedAt: item.UpdatedAt.Format(time.RFC3339Nano), Category: category, CanEdit: item.OwnerID != 0 && item.OwnerID == userID}, nil
}

// validateCardChart 校验图表字段只引用真实结果，并根据图型检查数据结构。
func validateCardChart(chartType, raw string, spec bi.QuerySpec, result bi.QueryResult) (string, error) {
	config := dto.CardChartConfig{Style: dto.CardChartStyle{ShowLegend: true}}
	var err error
	if raw != "" {
		if !json.Valid([]byte(raw)) {
			return "", errorsx.InvalidArgument("图表配置格式无效")
		}
		decoder := json.NewDecoder(bytes.NewBufferString(raw))
		decoder.DisallowUnknownFields()
		if err = decoder.Decode(&config); err != nil {
			return "", errorsx.InvalidArgument("图表配置格式无效").WithCause(err)
		}
	}
	if chartType == "scatter" && (config.Encoding.Dimension == "" || config.Encoding.Measure == "") {
		var numbers []string
		for _, column := range result.Columns {
			if column.Type == bi.FieldTypeNumber {
				numbers = append(numbers, column.Code)
			}
		}
		if len(numbers) < 2 {
			return "", errorsx.InvalidArgument("散点图需要两个不同的数值字段")
		}
		if config.Encoding.Dimension == "" {
			config.Encoding.Dimension = numbers[0]
		}
		if config.Encoding.Measure == "" {
			config.Encoding.Measure = numbers[1]
		}
	}
	if config.Encoding.Measure == "" && len(spec.Measures) > 0 {
		config.Encoding.Measure = spec.Measures[0].Code
	}
	if config.Encoding.Dimension == "" && len(spec.Dimensions) > 0 {
		config.Encoding.Dimension = spec.Dimensions[0]
	}
	if config.Encoding.Series == "" && len(spec.Dimensions) > 1 && chartType != "scatter" {
		config.Encoding.Series = spec.Dimensions[1]
	}
	fields := make(map[string]bi.FieldType, len(result.Columns))
	for _, column := range result.Columns {
		fields[column.Code] = column.Type
	}
	encoding := config.Encoding
	for _, name := range append([]string{encoding.Dimension, encoding.Measure, encoding.Series}, encoding.Measures...) {
		if name != "" {
			if _, ok := fields[name]; !ok {
				return "", errorsx.InvalidArgument("图表引用了查询结果不存在的字段")
			}
		}
	}
	if fields[encoding.Measure] != bi.FieldTypeNumber {
		return "", errorsx.InvalidArgument("图表需要数值字段")
	}
	for _, name := range encoding.Measures {
		if fields[name] != bi.FieldTypeNumber {
			return "", errorsx.InvalidArgument("多指标图表需要数值字段")
		}
	}
	switch chartType {
	case "scatter":
		if fields[encoding.Dimension] != bi.FieldTypeNumber || encoding.Dimension == encoding.Measure {
			return "", errorsx.InvalidArgument("散点图需要两个不同的数值字段")
		}
	case "sankey", "heatmap", "sunburst":
		if encoding.Dimension == "" || encoding.Series == "" || encoding.Dimension == encoding.Series {
			return "", errorsx.InvalidArgument("当前图表需要两个分类字段和一个数值字段")
		}
	case "metric", "gauge":
		if len(spec.Dimensions) > 0 {
			return "", errorsx.InvalidArgument("指标卡和仪表图需要不分组的查询结果")
		}
	default:
		if encoding.Dimension == "" {
			return "", errorsx.InvalidArgument("当前图表需要分类或日期字段")
		}
	}
	var payload []byte
	payload, err = json.Marshal(config)
	if err != nil {
		return "", errorsx.Internal("保存图表配置失败").WithCause(err)
	}
	return string(payload), nil
}
