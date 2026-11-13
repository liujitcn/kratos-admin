package admin

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/message"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/model"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/structured"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
	"google.golang.org/protobuf/proto"
	gormio "gorm.io/gorm"
)

const (
	biDefaultLimit   int32 = 100
	biHardMaxLimit   int32 = 1000
	biQueryTimeout         = 15 * time.Second
	biMaxResultBytes       = 48 * 1024
	biTimeLayout           = "2006-01-02 15:04:05"
	// biDuplicateQueryWindow 是判定重复问数的回看窗口：模型在审批恢复后常会重复发起同参查询。
	biDuplicateQueryWindow = time.Minute
)

// BiQueryCase 提供 BI 数据源目录、语义查询预览和自然语言问数。
type BiQueryCase struct {
	*biz.BaseCase
	biSourceCase *BiSourceCase
	modelStore   *model.Registry
	biQueryRepo  *data.BiQueryRepository
}

// NewBiQueryCase 创建 BI 语义查询业务实例。
func NewBiQueryCase(baseCase *biz.BaseCase, biSourceCase *BiSourceCase, modelStore *model.Registry, biQueryRepo *data.BiQueryRepository) *BiQueryCase {
	return &BiQueryCase{BaseCase: baseCase, biSourceCase: biSourceCase, modelStore: modelStore, biQueryRepo: biQueryRepo}
}

// ListBiSources 查询当前访问范围内可用的数据源目录。
func (c *BiQueryCase) ListBiSources(ctx context.Context, _ *adminv1.ListBiSourcesRequest) (*adminv1.ListBiSourcesResponse, error) {
	scope, err := c.scope(ctx)
	if err != nil {
		return nil, err
	}
	var descriptors []bi.SourceDescriptor
	descriptors, err = c.biSourceCase.List(ctx, scope)
	if err != nil {
		return nil, errorsx.WrapInternal(err, "查询 BI 数据源失败")
	}
	items := make([]*adminv1.BiSourceDescriptor, 0, len(descriptors))
	for _, descriptor := range descriptors {
		items = append(items, toSourceDescriptor(descriptor))
	}
	return &adminv1.ListBiSourcesResponse{Sources: items}, nil
}

// PreviewBiQuery 校验并执行语义查询。
func (c *BiQueryCase) PreviewBiQuery(ctx context.Context, req *adminv1.PreviewBiQueryRequest) (*adminv1.BiQueryResult, error) {
	scope, err := c.scope(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetQuery() == nil {
		return nil, errorsx.InvalidArgument("BI 查询计划不能为空")
	}
	spec := fromProtoQuerySpec(req.GetQuery())
	spec, _, err = c.validate(ctx, scope, spec)
	if err != nil {
		return nil, err
	}
	var result bi.QueryResult
	result, err = c.execute(ctx, scope, spec, nil)
	if err != nil {
		return nil, err
	}
	return toProtoQueryResult(result), nil
}

// AskBiQuery 使用所选模型生成语义查询，并记录问数结果或失败原因。
func (c *BiQueryCase) AskBiQuery(ctx context.Context, req *adminv1.AskBiQueryRequest) (resp *adminv1.AskBiQueryResponse, err error) {
	question := strings.TrimSpace(req.GetQuestion())
	if question == "" {
		return nil, errorsx.InvalidArgument("BI 问题不能为空")
	}
	var scope bi.Scope
	scope, err = c.scope(ctx)
	if err != nil {
		return nil, err
	}
	recordContext := &dto.BiQueryRecordContext{Question: question, StartedAt: time.Now()}
	executed := false
	defer func() {
		if !executed {
			c.recordQuery(ctx, scope, bi.QuerySpec{}, bi.QueryResult{}, err, recordContext)
		}
	}()
	var descriptors []bi.SourceDescriptor
	descriptors, err = c.biSourceCase.List(ctx, scope)
	if err != nil {
		return nil, errorsx.WrapInternal(err, "读取 BI 数据源目录失败")
	}
	// 对话工具沿用运行时已选模型，避免工具参数改写本轮模型选择。
	client := agent.AssistantClientFromContext(ctx)
	if client == nil {
		client, err = c.modelStore.ResolveAssistantClient(req.GetProviderId(), req.GetModelName())
		if err != nil {
			return nil, errorsx.InvalidArgument("AI供应商或模型选择无效").WithCause(err)
		}
	}
	var dictionary []byte
	dictionary, err = json.Marshal(descriptors)
	if err != nil {
		return nil, errorsx.Internal("生成 BI 数据源目录失败").WithCause(err)
	}
	var response *message.AgenticMessage
	response, err = client.Generate(ctx, []*message.AgenticMessage{
		message.SystemText(`你是 BI 语义查询引擎。只能根据给定数据源目录输出单个 JSON 对象，不得输出 SQL、表名、Markdown 或解释。
JSON 格式为 {"query":{"source_code":"数据源编码","dimensions":[],"measures":[{"code":"指标编码","aggregation":"count"}],"filters":[],"time_range":null,"orders":[],"limit":100},"chart_type":"bar","title":"统计标题"}。
chart_type 只支持 bar、line、area、pie、donut、scatter、radar、gauge、funnel、treemap、sunburst、sankey、heatmap、metric。无分组的总数使用 metric；桑基、热力和旭日需要两个分类维度；散点需要两个数值字段。
所有编码必须使用目录中的 code；limit 必须为整数。每次最多 2 个分组维度、5 个指标、10 个过滤条件、2 个排序条件。
过滤格式为 {"field":"维度编码","operator":"eq","values":["0"]}，操作符只支持 eq、in、between、contains；values 必须为字符串数组，数字条件也写成字符串。
排序格式为 {"field":"已选字段编码","direction":"desc"}，direction 只支持 asc、desc。
统计总数时 dimensions 为空，使用目录中允许 count 聚合的指标。无过滤或排序需求时使用空数组；无时间范围需求时 time_range 为 null，否则使用 {"field":"时间维度编码","start":"开始时间","end":"结束时间"}。`),
		message.UserText("可用数据源目录:" + string(dictionary) + "\n用户问题:" + question),
	})
	if err != nil {
		return nil, errorsx.Internal("BI 问数生成失败").WithCause(err)
	}
	var output dto.AskOutput
	if err = structured.DecodeContent(message.Text(response), &output); err != nil {
		return nil, errorsx.InvalidArgument("BI 问数结果不是有效查询计划").WithCause(err)
	}
	// 模型判定问题无法回答时返回 error 字段，应直接透传拒绝原因而不是当作空查询计划校验。
	if output.Error != "" {
		return nil, errorsx.InvalidArgument(output.Error)
	}
	var validated bi.QuerySpec
	validated, _, err = c.validate(ctx, scope, fromProtoQuerySpec(&adminv1.BiQuerySpec{
		SourceCode: output.Query.SourceCode,
		Dimensions: output.Query.Dimensions,
		Measures:   toProtoMeasures(output.Query.Measures),
		Filters:    toProtoFilters(output.Query.Filters),
		TimeRange:  toProtoAskTimeRange(output.Query.TimeRange),
		Orders:     toProtoOrders(output.Query.Orders),
		Limit:      output.Query.Limit,
	}))
	if err != nil {
		return nil, err
	}
	var reusable *models.BiQuery
	reusable, err = c.findReusableQueryRecord(ctx, scope, validated)
	if err != nil {
		return nil, errorsx.WrapInternal(err, "检查重复问数失败")
	}
	executed = true
	var result bi.QueryResult
	if reusable != nil {
		// 审批恢复后模型常会重复发起同参问数，直接复用窗口内的成功结果，避免重复执行。
		if err = json.Unmarshal([]byte(reusable.ResultSnapshot), &result); err != nil {
			return nil, errorsx.Internal("读取重复问数结果失败").WithCause(err)
		}
		result.QueryRecordID = reusable.ID
	} else {
		result, err = c.execute(ctx, scope, validated, recordContext)
		if err != nil {
			return nil, err
		}
	}
	// 保留符合真实结果结构的图型建议，无分组时仍允许仪表图。
	fallback := "bar"
	if len(validated.Dimensions) == 0 {
		fallback = "metric"
	}
	if !slices.Contains(biCardChartTypes, output.ChartType) {
		output.ChartType = fallback
	}
	if _, err = validateCardChart(output.ChartType, "", validated, result); err != nil {
		output.ChartType = fallback
	}
	return &adminv1.AskBiQueryResponse{
		Query:     toProtoQuerySpec(validated),
		Result:    toProtoQueryResult(result),
		ChartType: output.ChartType,
		Title:     output.Title,
	}, nil
}

// findReusableQueryRecord 仅复用同一访问范围和查询计划的成功智能问数记录，排除普通预览与仪表盘执行。
func (c *BiQueryCase) findReusableQueryRecord(ctx context.Context, scope bi.Scope, spec bi.QuerySpec) (*models.BiQuery, error) {
	hash := biQueryHash(spec)
	if hash == "" {
		return nil, nil
	}
	query := c.biQueryRepo.Query(ctx).BiQuery
	since := time.Now().Add(-biDuplicateQueryWindow)
	opts := []repository.QueryOption{
		repository.Where(query.TenantID.Eq(scope.TenantID), query.UserID.Eq(scope.UserID), query.ProjectID.Eq(scope.ProjectID),
			query.QueryHash.Eq(hash), query.Status.Eq(int32(adminv1.BiQueryStatus_BI_QUERY_STATUS_SUCCESS)), query.Question.Neq(""), query.CreatedAt.Gt(since)),
		repository.Order(query.ID.Desc()),
	}
	record, err := c.biQueryRepo.Find(ctx, opts...)
	if errors.Is(err, gormio.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return record, nil
}

// execute 执行语义查询，并将成功或失败统一写入查询记录。
func (c *BiQueryCase) execute(ctx context.Context, scope bi.Scope, spec bi.QuerySpec, recordContext *dto.BiQueryRecordContext) (result bi.QueryResult, err error) {
	startedAt := time.Now()
	if recordContext == nil {
		recordContext = &dto.BiQueryRecordContext{}
	}
	if recordContext.StartedAt.IsZero() {
		recordContext.StartedAt = startedAt
	}
	defer func() {
		result.QueryRecordID = c.recordQuery(ctx, scope, spec, result, err, recordContext)
		if err == nil && result.QueryRecordID == 0 {
			err = errorsx.Internal("保存查询结果记录失败，请重试")
		}
	}()
	queryCtx, cancel := context.WithTimeout(ctx, biQueryTimeout)
	defer cancel()
	executionSpec := spec
	if spec.TimeRange != nil {
		var start, end string
		start, err = resolveRelativeTime(spec.TimeRange.Start, startedAt)
		if err != nil {
			return bi.QueryResult{}, err
		}
		end, err = resolveRelativeTime(spec.TimeRange.End, startedAt)
		if err != nil {
			return bi.QueryResult{}, err
		}
		executionSpec.Filters = append(append([]bi.FilterSpec{}, spec.Filters...), bi.FilterSpec{
			Field:    spec.TimeRange.Field,
			Operator: "between",
			Values:   []string{start, end},
		})
		executionSpec.TimeRange = nil
	}
	result, err = c.biSourceCase.Execute(queryCtx, scope, executionSpec)
	if err != nil {
		return bi.QueryResult{}, errorsx.WrapInternal(err, "执行 BI 查询失败")
	}
	result = applyResultOrders(result, spec.Orders, spec.Limit)
	if queryResultBytes(result) > biMaxResultBytes {
		return bi.QueryResult{}, errorsx.InvalidArgument("BI 查询结果超过大小限制，请缩小查询范围")
	}
	result.ElapsedMS = time.Since(startedAt).Milliseconds()
	return result, nil
}

// cardSourceRecord 仅接受所属账号成功执行的智能问数记录，并核对完整查询定义。
func (c *BiQueryCase) cardSourceRecord(ctx context.Context, scope bi.Scope, id int64, spec bi.QuerySpec) (*models.BiQuery, error) {
	query := c.biQueryRepo.Query(ctx).BiQuery
	record, err := c.biQueryRepo.Find(ctx, repository.Where(query.ID.Eq(id), query.TenantID.Eq(scope.TenantID), query.UserID.Eq(scope.UserID), query.ProjectID.Eq(scope.ProjectID), query.Status.Eq(int32(adminv1.BiQueryStatus_BI_QUERY_STATUS_SUCCESS)), query.Question.Neq("")))
	if err != nil {
		return nil, errorsx.InvalidArgument("请选择当前账号和项目下成功的智能问数记录")
	}
	var recorded bi.QuerySpec
	if err = json.Unmarshal([]byte(record.QueryJSON), &recorded); err != nil {
		return nil, errorsx.Internal("读取来源查询失败").WithCause(err)
	}
	if !proto.Equal(toProtoQuerySpec(spec), toProtoQuerySpec(recorded)) {
		return nil, errorsx.InvalidArgument("查询条件已变化，请重新问数后保存卡片")
	}
	return record, nil
}

// recordQuery 保存完整类型化结果，成功记录返回可供卡片引用的编号。
func (c *BiQueryCase) recordQuery(ctx context.Context, scope bi.Scope, spec bi.QuerySpec, result bi.QueryResult, queryErr error, recordContext *dto.BiQueryRecordContext) int64 {
	record := &models.BiQuery{
		TenantID: scope.TenantID, UserID: scope.UserID, ProjectID: scope.ProjectID,
		DashboardID: recordContext.DashboardID, DashboardVersion: recordContext.DashboardVersion, WidgetID: recordContext.WidgetID,
		SourceCode: spec.SourceCode, Question: recordContext.Question,
		Status: int32(adminv1.BiQueryStatus_BI_QUERY_STATUS_SUCCESS), RowCount: int32(len(result.Rows)),
		ElapsedMs: time.Since(recordContext.StartedAt).Milliseconds(),
	}
	if result.Truncated {
		record.Truncated = 1
	}
	var err error
	if spec.SourceCode != "" {
		record.QueryJSON = biQuerySpecJSON(spec)
		record.QueryHash = biQueryHash(spec)
	}
	if queryErr != nil {
		record.Status = int32(adminv1.BiQueryStatus_BI_QUERY_STATUS_FAILED)
		errorMessage := []rune(queryErr.Error())
		record.ErrorMsg = string(errorMessage[:min(len(errorMessage), 500)])
	} else {
		var snapshot []byte
		snapshot, err = json.Marshal(result)
		if err != nil {
			log.Error("序列化 BI 查询结果快照失败", "error", err)
			return 0
		}
		record.ResultSnapshot = string(snapshot)
	}
	// 请求取消后仍需保存失败原因，同时限制留痕写入的最长等待时间。
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), biQueryTimeout)
	defer cancel()
	if err = c.biQueryRepo.Create(recordCtx, record); err != nil {
		log.Error("写入 BI 查询记录失败", "error", err)
		return 0
	}
	return record.ID
}

// scope 从认证信息和调用上下文构建可信的查询访问范围。
func (c *BiQueryCase) scope(ctx context.Context) (bi.Scope, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return bi.Scope{}, err
	}
	var projectID int64
	projectID, err = projectIDForAuth(authInfo)
	if err != nil {
		return bi.Scope{}, err
	}
	terminal := agent.TerminalFromContext(ctx)
	if terminal == "" {
		terminal = "admin"
	}
	return bi.Scope{TenantID: authInfo.TenantId, ProjectID: projectID, UserID: authInfo.UserId, Terminal: terminal}, nil
}

// validate 按可访问数据源的语义定义校验查询维度、指标和过滤条件。
func (c *BiQueryCase) validate(ctx context.Context, scope bi.Scope, spec bi.QuerySpec) (bi.QuerySpec, bi.SourceDescriptor, error) {
	if len(spec.Dimensions) > 2 || len(spec.Measures) == 0 || len(spec.Measures) > 5 || len(spec.Filters) > 10 || len(spec.Orders) > 2 {
		return bi.QuerySpec{}, bi.SourceDescriptor{}, errorsx.InvalidArgument("BI 查询维度、指标或过滤条件超出限制")
	}
	if spec.Limit <= 0 {
		spec.Limit = biDefaultLimit
	}
	if spec.Limit > biHardMaxLimit {
		spec.Limit = biHardMaxLimit
	}
	descriptors, err := c.biSourceCase.List(ctx, scope)
	if err != nil {
		return bi.QuerySpec{}, bi.SourceDescriptor{}, errorsx.WrapInternal(err, "查询 BI 数据源目录失败")
	}
	var descriptor bi.SourceDescriptor
	for _, item := range descriptors {
		if item.Code == spec.SourceCode {
			descriptor = item
			break
		}
	}
	if descriptor.Code == "" {
		return bi.QuerySpec{}, bi.SourceDescriptor{}, errorsx.ResourceNotFound("BI 数据源不存在")
	}
	if descriptor.MaxRows > 0 && spec.Limit > descriptor.MaxRows {
		spec.Limit = descriptor.MaxRows
	}
	dimensions := make(map[string]bi.FieldDescriptor, len(descriptor.Dimensions))
	for _, item := range descriptor.Dimensions {
		dimensions[item.Code] = item
	}
	for _, code := range spec.Dimensions {
		if _, ok := dimensions[code]; !ok {
			return bi.QuerySpec{}, bi.SourceDescriptor{}, errorsx.InvalidArgument(fmt.Sprintf("BI 维度不存在: %s", code))
		}
	}
	measures := make(map[string]bi.MeasureDescriptor, len(descriptor.Measures))
	for _, item := range descriptor.Measures {
		measures[item.Code] = item
	}
	for _, item := range spec.Measures {
		measure, ok := measures[item.Code]
		if !ok || !slices.Contains(measure.Aggregations, item.Aggregation) {
			return bi.QuerySpec{}, bi.SourceDescriptor{}, errorsx.InvalidArgument(fmt.Sprintf("BI 指标或聚合函数不可用: %s/%s", item.Code, item.Aggregation))
		}
	}
	for _, item := range spec.Filters {
		field, ok := dimensions[item.Field]
		if !ok || !field.Filterable || len(item.Values) == 0 {
			return bi.QuerySpec{}, bi.SourceDescriptor{}, errorsx.InvalidArgument(fmt.Sprintf("BI 过滤字段不可用: %s", item.Field))
		}
		if item.Operator != "eq" && item.Operator != "in" && item.Operator != "between" && item.Operator != "contains" {
			return bi.QuerySpec{}, bi.SourceDescriptor{}, errorsx.InvalidArgument("BI 过滤操作符不支持")
		}
	}
	if spec.TimeRange != nil {
		field, ok := dimensions[spec.TimeRange.Field]
		if !ok || field.Type != bi.FieldTypeDate || !field.Filterable || spec.TimeRange.Start == "" || spec.TimeRange.End == "" {
			return bi.QuerySpec{}, bi.SourceDescriptor{}, errorsx.InvalidArgument("BI 时间范围不可用")
		}
	}
	selected := make(map[string]struct{}, len(spec.Dimensions)+len(spec.Measures))
	for _, code := range spec.Dimensions {
		selected[code] = struct{}{}
	}
	for _, item := range spec.Measures {
		selected[item.Code] = struct{}{}
	}
	for _, item := range spec.Orders {
		if _, ok := selected[item.Field]; !ok || (item.Direction != "asc" && item.Direction != "desc") {
			return bi.QuerySpec{}, bi.SourceDescriptor{}, errorsx.InvalidArgument("BI 排序字段或方向不可用")
		}
	}
	return spec, descriptor, nil
}

// toSourceDescriptor 将领域层数据源字段描述（维度、度量、行数上限）转换为接口响应结构。
func toSourceDescriptor(value bi.SourceDescriptor) *adminv1.BiSourceDescriptor {
	dimensions := make([]*adminv1.BiFieldDescriptor, 0, len(value.Dimensions))
	for _, item := range value.Dimensions {
		dimensions = append(dimensions, &adminv1.BiFieldDescriptor{Code: item.Code, Name: item.Name, Type: string(item.Type), Filterable: item.Filterable})
	}
	measures := make([]*adminv1.BiMeasureDescriptor, 0, len(value.Measures))
	for _, item := range value.Measures {
		measures = append(measures, &adminv1.BiMeasureDescriptor{Code: item.Code, Name: item.Name, Type: string(item.Type), Aggregations: item.Aggregations})
	}
	return &adminv1.BiSourceDescriptor{Code: value.Code, Name: value.Name, Description: value.Description, Version: value.Version, Dimensions: dimensions, Measures: measures, MaxRows: value.MaxRows}
}

// fromProtoQuerySpec 将接口请求中的查询规格转换为领域查询对象，供执行器校验和下推执行。
func fromProtoQuerySpec(value *adminv1.BiQuerySpec) bi.QuerySpec {
	measures := make([]bi.MeasureSpec, 0, len(value.GetMeasures()))
	for _, item := range value.GetMeasures() {
		measures = append(measures, bi.MeasureSpec{Code: item.GetCode(), Aggregation: item.GetAggregation()})
	}
	filters := make([]bi.FilterSpec, 0, len(value.GetFilters()))
	for _, item := range value.GetFilters() {
		filters = append(filters, bi.FilterSpec{Field: item.GetField(), Operator: item.GetOperator(), Values: item.GetValues()})
	}
	orders := make([]bi.OrderSpec, 0, len(value.GetOrders()))
	for _, item := range value.GetOrders() {
		orders = append(orders, bi.OrderSpec{Field: item.GetField(), Direction: item.GetDirection()})
	}
	var timeRange *bi.TimeRangeSpec
	if item := value.GetTimeRange(); item != nil {
		timeRange = &bi.TimeRangeSpec{Field: item.GetField(), Start: item.GetStart(), End: item.GetEnd()}
	}
	return bi.QuerySpec{SourceCode: value.GetSourceCode(), Dimensions: value.GetDimensions(), Measures: measures, Filters: filters, TimeRange: timeRange, Orders: orders, Limit: value.GetLimit()}
}

// toProtoQuerySpec 将领域查询对象转换回接口结构，用于查询记录详情回显原始查询条件。
func toProtoQuerySpec(value bi.QuerySpec) *adminv1.BiQuerySpec {
	return &adminv1.BiQuerySpec{SourceCode: value.SourceCode, Dimensions: value.Dimensions, Measures: toProtoMeasuresFromSpec(value.Measures), Filters: toProtoFiltersFromSpec(value.Filters), TimeRange: toProtoTimeRange(value.TimeRange), Orders: toProtoOrdersFromSpec(value.Orders), Limit: value.Limit}
}

// toProtoMeasures 将 AI 问数解析出的度量列表转换为接口查询规格。
func toProtoMeasures(value []dto.AskMeasure) []*adminv1.BiMeasureSpec {
	items := make([]*adminv1.BiMeasureSpec, 0, len(value))
	for _, item := range value {
		items = append(items, &adminv1.BiMeasureSpec{Code: item.Code, Aggregation: item.Aggregation})
	}
	return items
}

// toProtoFilters 将 AI 问数解析出的筛选条件转换为接口查询规格。
func toProtoFilters(value []dto.AskFilter) []*adminv1.BiFilterSpec {
	items := make([]*adminv1.BiFilterSpec, 0, len(value))
	for _, item := range value {
		items = append(items, &adminv1.BiFilterSpec{Field: item.Field, Operator: item.Operator, Values: item.Values})
	}
	return items
}

// toProtoAskTimeRange 将 AI 问数解析出的时间范围转换为接口查询规格，未解析到时间条件时返回空。
func toProtoAskTimeRange(value *dto.AskTimeRange) *adminv1.BiTimeRangeSpec {
	if value == nil {
		return nil
	}
	return &adminv1.BiTimeRangeSpec{Field: value.Field, Start: value.Start, End: value.End}
}

// toProtoOrders 将 AI 问数解析出的排序要求转换为接口查询规格。
func toProtoOrders(value []dto.AskOrder) []*adminv1.BiOrderSpec {
	items := make([]*adminv1.BiOrderSpec, 0, len(value))
	for _, item := range value {
		items = append(items, &adminv1.BiOrderSpec{Field: item.Field, Direction: item.Direction})
	}
	return items
}

// toProtoMeasuresFromSpec 将领域度量规格转换为接口结构，用于查询记录回显。
func toProtoMeasuresFromSpec(value []bi.MeasureSpec) []*adminv1.BiMeasureSpec {
	items := make([]*adminv1.BiMeasureSpec, 0, len(value))
	for _, item := range value {
		items = append(items, &adminv1.BiMeasureSpec{Code: item.Code, Aggregation: item.Aggregation})
	}
	return items
}

// toProtoFiltersFromSpec 将领域筛选规格转换为接口结构，用于查询记录回显。
func toProtoFiltersFromSpec(value []bi.FilterSpec) []*adminv1.BiFilterSpec {
	items := make([]*adminv1.BiFilterSpec, 0, len(value))
	for _, item := range value {
		items = append(items, &adminv1.BiFilterSpec{Field: item.Field, Operator: item.Operator, Values: item.Values})
	}
	return items
}

// toProtoTimeRange 将领域时间范围规格转换为接口结构，无时间条件时返回空。
func toProtoTimeRange(value *bi.TimeRangeSpec) *adminv1.BiTimeRangeSpec {
	if value == nil {
		return nil
	}
	return &adminv1.BiTimeRangeSpec{Field: value.Field, Start: value.Start, End: value.End}
}

// toProtoOrdersFromSpec 将领域排序规格转换为接口结构，用于查询记录回显。
func toProtoOrdersFromSpec(value []bi.OrderSpec) []*adminv1.BiOrderSpec {
	items := make([]*adminv1.BiOrderSpec, 0, len(value))
	for _, item := range value {
		items = append(items, &adminv1.BiOrderSpec{Field: item.Field, Direction: item.Direction})
	}
	return items
}

// toProtoQueryResult 将类型化结果和来源记录编号转换为接口响应。
func toProtoQueryResult(value bi.QueryResult) *adminv1.BiQueryResult {
	columns := make([]*adminv1.BiResultColumn, 0, len(value.Columns))
	for _, item := range value.Columns {
		columns = append(columns, &adminv1.BiResultColumn{Code: item.Code, Name: item.Name, Type: string(item.Type)})
	}
	rows := make([]*adminv1.BiResultRow, 0, len(value.Rows))
	for _, item := range value.Rows {
		rows = append(rows, &adminv1.BiResultRow{Values: item})
	}
	return &adminv1.BiQueryResult{Columns: columns, Rows: rows, Truncated: value.Truncated, ElapsedMs: value.ElapsedMS, QueryRecordId: value.QueryRecordID}
}

// projectIDForAuth 在认证范围明确只有一个项目时传递项目范围；无法表达有限多项目范围时拒绝执行。
func projectIDForAuth(authInfo *authData.UserTokenPayload) (int64, error) {
	if authInfo.TenantCode == gorm.DefaultTenantCode {
		return 0, nil
	}
	if len(authInfo.TenantProjects) == 0 {
		return 0, errorsx.PermissionDenied("当前用户没有可用项目范围")
	}
	var projectID int64
	found := false
	for _, scope := range authInfo.TenantProjects {
		if scope.TenantId != authInfo.TenantId || len(scope.ProjectId) != 1 {
			continue
		}
		found = true
		if scope.ProjectId[0] == 0 {
			return 0, nil
		}
		if projectID != 0 {
			return 0, errorsx.PermissionDenied("当前用户拥有多个项目范围，请选择单个项目后查询 BI")
		}
		projectID = scope.ProjectId[0]
	}
	if !found {
		return 0, errorsx.PermissionDenied("当前用户没有可用项目范围")
	}
	return projectID, nil
}

// queryResultBytes 计算结果体积，避免 Adapter 返回过大的内存结果。
func queryResultBytes(result bi.QueryResult) int {
	size := 0
	for _, column := range result.Columns {
		size += len(column.Code) + len(column.Name) + len(column.Type)
	}
	for _, row := range result.Rows {
		for _, value := range row {
			size += len(value)
		}
	}
	return size
}

// applyResultOrders 对结果排序并统一裁剪，即使未指定排序也保持查询行数上限。
// shortcut: 外部 Adapter 可能先限制再排序，需在 Adapter 内下推排序才能保证严格 Top-N。
func applyResultOrders(result bi.QueryResult, orders []bi.OrderSpec, limit int32) bi.QueryResult {
	if len(result.Rows) == 0 {
		return result
	}
	columnIndexes := make(map[string]int, len(result.Columns))
	for index, column := range result.Columns {
		columnIndexes[column.Code] = index
	}
	orderable := true
	for _, order := range orders {
		if _, ok := columnIndexes[order.Field]; !ok {
			orderable = false
			break
		}
	}
	if orderable && len(orders) > 0 {
		sort.SliceStable(result.Rows, func(left, right int) bool {
			for _, order := range orders {
				index := columnIndexes[order.Field]
				comparison := compareResultValues(result.Rows[left][index], result.Rows[right][index])
				if comparison == 0 {
					continue
				}
				if order.Direction == "desc" {
					return comparison > 0
				}
				return comparison < 0
			}
			return false
		})
	}
	if limit > 0 && int32(len(result.Rows)) > limit {
		result.Rows = result.Rows[:limit]
		result.Truncated = true
	}
	return result
}

// compareResultValues 比较结果单元格值：两侧都能解析为数字时按数值比较，否则退回字符串比较，
// 保证数值列排序符合数值语义而非字典序。
func compareResultValues(left, right string) int {
	leftNumber, leftErr := strconv.ParseFloat(left, 64)
	rightNumber, rightErr := strconv.ParseFloat(right, 64)
	if leftErr == nil && rightErr == nil {
		switch {
		case leftNumber < rightNumber:
			return -1
		case leftNumber > rightNumber:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(left, right)
}

// resolveRelativeTime 将问数时间范围中的 now 相对表达式折叠为数据库可比较的绝对时间，
// 支持 now、now-Nd/Nh/Nw/Nm；非相对表达式原样返回，形如 now-abc 的非法表达式显式报错，
// 避免相对时间原样进入 SQL 比较后静默匹配不到任何数据。
// biQuerySpecJSON 序列化查询计划，作为记录留痕与哈希计算的统一输入。
func biQuerySpecJSON(spec bi.QuerySpec) string {
	payload, err := json.Marshal(spec)
	if err != nil {
		log.Error("序列化 BI 查询计划失败", "error", err)
		return ""
	}
	return string(payload)
}

// biQueryHash 计算查询计划的稳定哈希，用于识别重复问数。
func biQueryHash(spec bi.QuerySpec) string {
	payload := biQuerySpecJSON(spec)
	if payload == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("%x", digest[:])
}

// resolveRelativeTime 将问数时间范围中的 now 相对表达式折叠为数据库可比较的绝对时间，
// 支持 now、now-Nd/Nh/Nw/Nm；非相对表达式原样返回，形如 now-abc 的非法表达式显式报错，
// 避免相对时间原样进入 SQL 比较后静默匹配不到任何数据。
func resolveRelativeTime(value string, now time.Time) (string, error) {
	text := strings.TrimSpace(value)
	if !strings.HasPrefix(strings.ToLower(text), "now") {
		return text, nil
	}
	offset := strings.ToLower(strings.TrimSpace(text[3:]))
	if offset == "" {
		return now.Format(biTimeLayout), nil
	}
	if len(offset) < 3 || offset[0] != '-' {
		return "", errorsx.InvalidArgument("BI 时间范围表达式无效: " + value)
	}
	amount, err := strconv.Atoi(offset[1 : len(offset)-1])
	if err != nil {
		return "", errorsx.InvalidArgument("BI 时间范围表达式无效: " + value)
	}
	switch offset[len(offset)-1] {
	case 'd':
		return now.AddDate(0, 0, -amount).Format(biTimeLayout), nil
	case 'w':
		return now.AddDate(0, 0, -7*amount).Format(biTimeLayout), nil
	case 'm':
		return now.AddDate(0, -amount, 0).Format(biTimeLayout), nil
	case 'h':
		return now.Add(-time.Duration(amount) * time.Hour).Format(biTimeLayout), nil
	default:
		return "", errorsx.InvalidArgument("BI 时间范围表达式无效: " + value)
	}
}
