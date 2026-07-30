package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreOpenAPI "github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen"
)

const (
	baseAPIDocJSONMediaType = "application/json"
	baseAPIDocMaxDepth      = 8
)

// BaseAPICase 接口业务实例
type BaseAPICase struct {
	*biz.BaseCase
	*data.BaseAPIRepository
	mapper          *mapper.CopierMapper[systemadminv1.BaseApi, models.BaseAPI]
	jwtCfg          *bootstrapConfigv1.Authentication_Jwt
	openAPIRegistry *coreOpenAPI.Registry
}

// NewBaseAPICase 创建接口业务实例
func NewBaseAPICase(
	baseCase *biz.BaseCase,
	baseAPIRepo *data.BaseAPIRepository,
	jwtCfg *bootstrapConfigv1.Authentication_Jwt,
	openAPIRegistry *coreOpenAPI.Registry,
) *BaseAPICase {
	baseAPIMapper := mapper.NewCopierMapper[systemadminv1.BaseApi, models.BaseAPI]()
	baseAPIMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &BaseAPICase{
		BaseCase:          baseCase,
		BaseAPIRepository: baseAPIRepo,
		mapper:            baseAPIMapper,
		jwtCfg:            jwtCfg,
		openAPIRegistry:   openAPIRegistry,
	}
}

// OptionBaseAPI 查询菜单分配接口选项列表
func (c *BaseAPICase) OptionBaseAPI(ctx context.Context, _ *systemadminv1.OptionBaseApiRequest) (*systemadminv1.OptionBaseApiResponse, error) {
	query := c.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Order(query.ServiceName.Asc(), query.Operation.Asc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	baseAPIs := make([]*systemadminv1.BaseApi, 0, len(list))
	for _, item := range list {
		// 命中免 token 或可选鉴权规则的接口，不再返回给菜单管理页面。
		if c.jwtCfg != nil {
			isNoTokenOperation := matchAuthWhiteList(c.jwtCfg.GetWhiteList(), item.Operation) ||
				matchAuthWhiteList(c.jwtCfg.GetOptionalAuth(), item.Operation)
			if isNoTokenOperation {
				continue
			}
		}
		baseAPI := c.toBaseAPIDTO(item)
		baseAPIs = append(baseAPIs, baseAPI)
	}

	return &systemadminv1.OptionBaseApiResponse{BaseApis: baseAPIs}, nil
}

// PageBaseAPI 分页查询接口列表
func (c *BaseAPICase) PageBaseAPI(ctx context.Context, req *systemadminv1.PageBaseApiRequest) (*systemadminv1.PageBaseApiResponse, error) {
	query := c.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 10)
	opts = append(opts, repository.Order(query.ID.Desc()))
	// 传入工具名时，按工具名模糊匹配。
	if req.GetToolName() != "" {
		opts = append(opts, repository.Where(query.ToolName.Like("%"+req.GetToolName()+"%")))
	}
	// 传入服务名关键字时，按服务名模糊匹配。
	if req.GetServiceName() != "" {
		opts = append(opts, repository.Where(query.ServiceName.Like("%"+req.GetServiceName()+"%")))
	}
	// 传入服务描述关键字时，按服务描述模糊匹配。
	if req.GetServiceDesc() != "" {
		opts = append(opts, repository.Where(query.ServiceDesc.Like("%"+req.GetServiceDesc()+"%")))
	}
	// 传入描述关键字时，按接口描述模糊匹配。
	if req.GetDesc() != "" {
		opts = append(opts, repository.Where(query.Desc.Like("%"+req.GetDesc()+"%")))
	}
	// 传入操作方法关键字时，按操作方法模糊匹配。
	if req.GetOperation() != "" {
		opts = append(opts, repository.Where(query.Operation.Like("%"+req.GetOperation()+"%")))
	}
	// 传入请求方式时，按请求方式精确匹配。
	if req.GetMethod() != "" {
		opts = append(opts, repository.Where(query.Method.Eq(req.GetMethod())))
	}
	// 传入请求地址关键字时，按请求地址模糊匹配。
	if req.GetPath() != "" {
		opts = append(opts, repository.Where(query.Path.Like("%"+req.GetPath()+"%")))
	}
	if req.McpStatus != nil {
		opts = append(opts, repository.Where(query.McpStatus.Eq(int32(req.GetMcpStatus()))))
	}
	if req.AgentStatus != nil {
		opts = append(opts, repository.Where(query.AgentStatus.Eq(int32(req.GetAgentStatus()))))
	}
	var list []*models.BaseAPI
	var total int64
	var err error
	if req.GetToolPrompt() != "" || req.GetOpenapiServiceCode() != "" {
		list, err = c.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		list = filterBaseAPIsByToolPrompt(list, req.GetToolPrompt())
		list = c.filterBaseAPIsByOpenAPIService(list, req.GetOpenapiServiceCode())
		total = int64(len(list))
		list = pageBaseAPIRecords(list, req.GetPageNum(), req.GetPageSize())
	} else {
		list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
		if err != nil {
			return nil, err
		}
	}

	baseAPIs := make([]*systemadminv1.BaseApi, 0, len(list))
	for _, item := range list {
		baseAPI := c.toBaseAPIDTO(item)
		baseAPIs = append(baseAPIs, baseAPI)
	}

	return &systemadminv1.PageBaseApiResponse{
		BaseApis: baseAPIs,
		Total:    int32(total),
	}, nil
}

// GetBaseAPI 根据主键查询接口详情
func (c *BaseAPICase) GetBaseAPI(ctx context.Context, id int64) (*systemadminv1.BaseApi, error) {
	query := c.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(id)))

	baseAPI, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return c.toBaseAPIDTO(baseAPI), nil
}

// GetBaseAPIDoc 查询接口 OpenAPI 文档
func (c *BaseAPICase) GetBaseAPIDoc(ctx context.Context, id int64) (*systemadminv1.BaseApiDoc, error) {
	query := c.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(id)))

	baseAPI, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}

	document, found := c.openAPIRegistry.DocumentByOperation(baseAPI.Path, baseAPI.Method)
	if !found {
		return nil, errorsx.Internal("查询API文档失败").WithCause(fmt.Errorf("openapi document not found: %s %s", baseAPI.Method, baseAPI.Path))
	}
	var api *biz.OpenAPI
	api, err = biz.ParseOpenAPI(document.Data)
	if err != nil {
		return nil, err
	}
	operation := api.Operation(baseAPI.Path, baseAPI.Method)
	if operation == nil {
		return nil, errorsx.Internal("查询API文档失败").WithCause(fmt.Errorf("openapi operation not found: %s %s", baseAPI.Method, baseAPI.Path))
	}

	return &systemadminv1.BaseApiDoc{
		Id:          baseAPI.ID,
		Summary:     operation.Summary,
		Description: operation.Description,
		Parameters:  buildBaseAPIDocParameters(api, operation.Parameters),
		RequestBody: buildBaseAPIDocRequestBody(api, operation.RequestBody),
		Responses:   buildBaseAPIDocResponses(api, operation.Responses),
	}, nil
}

// OptionOpenAPIService 查询 OpenAPI 文档选项。
func (c *BaseAPICase) OptionOpenAPIService(_ context.Context, req *systemadminv1.OptionOpenApiServiceRequest) (*systemadminv1.OptionOpenApiServiceResponse, error) {
	documents := c.openAPIRegistry.Documents()
	options := make([]*systemadminv1.OpenApiServiceOption, 0, len(documents))
	for _, document := range documents {
		if req.GetServiceCode() != "" && document.Key != req.GetServiceCode() {
			continue
		}
		options = append(options, &systemadminv1.OpenApiServiceOption{
			Key:  document.Key,
			Name: document.Name,
		})
	}
	return &systemadminv1.OptionOpenApiServiceResponse{List: options}, nil
}

// UpdateBaseAPI 更新接口 MCP、Agent 与工具提示词配置。
func (c *BaseAPICase) UpdateBaseAPI(ctx context.Context, req *systemadminv1.UpdateBaseApiRequest) error {
	query := c.Query(ctx).BaseAPI
	conditions := make([]gen.Condition, 0, 2)
	baseAPI, err := c.Find(ctx, repository.Where(query.ID.Eq(req.GetId())))
	if err != nil {
		return err
	}
	// 同名工具可能来自历史重复 API 记录，运行时配置需要同步到同一个工具名称。
	if baseAPI.ToolName != "" {
		conditions = append(conditions, query.ToolName.Eq(baseAPI.ToolName))
	} else {
		conditions = append(conditions, query.ID.Eq(req.GetId()))
	}
	toolPrompts := normalizeToolPrompts(req.GetToolPrompts())
	_, err = query.WithContext(ctx).
		Where(conditions...).
		UpdateSimple(
			query.ToolPrompts.Value(encodeToolPrompts(toolPrompts)),
			query.McpStatus.Value(int32(req.GetMcpStatus())),
			query.AgentStatus.Value(int32(req.GetAgentStatus())),
		)
	return err
}

// SetBaseAPIAgentStatus 设置接口 Agent 工具状态
func (c *BaseAPICase) SetBaseAPIAgentStatus(ctx context.Context, req *systemadminv1.SetBaseApiAgentStatusRequest) error {
	query := c.Query(ctx).BaseAPI
	conditions := make([]gen.Condition, 0, 2)
	baseAPI, err := c.Find(ctx, repository.Where(query.ID.Eq(req.GetId())))
	if err != nil {
		return err
	}
	// 同名工具可能来自历史重复 API 记录，状态需要同步到同一个 Agent Tool 名称。
	if baseAPI.ToolName != "" {
		conditions = append(conditions, query.ToolName.Eq(baseAPI.ToolName))
	} else {
		conditions = append(conditions, query.ID.Eq(req.GetId()))
	}
	_, err = query.WithContext(ctx).
		Where(conditions...).
		UpdateSimple(query.AgentStatus.Value(int32(req.GetAgentStatus())))
	return err
}

// SetBaseAPIMcpStatus 设置接口 MCP 工具状态
func (c *BaseAPICase) SetBaseAPIMcpStatus(ctx context.Context, req *systemadminv1.SetBaseApiMcpStatusRequest) error {
	query := c.Query(ctx).BaseAPI
	conditions := make([]gen.Condition, 0, 1)
	conditions = append(conditions, query.ID.Eq(req.GetId()))
	_, err := query.WithContext(ctx).
		Where(conditions...).
		UpdateSimple(query.McpStatus.Value(int32(req.GetMcpStatus())))
	return err
}

// toBaseAPIDTO 转换接口数据并补充所属 OpenAPI 文档信息。
func (c *BaseAPICase) toBaseAPIDTO(item *models.BaseAPI) *systemadminv1.BaseApi {
	baseAPI := c.mapper.ToDTO(item)
	document, found := c.openAPIRegistry.DocumentByOperation(item.Path, item.Method)
	if found {
		baseAPI.OpenapiServiceCode = document.Key
		baseAPI.OpenapiServiceName = document.Name
	}
	return baseAPI
}

// filterBaseAPIsByOpenAPIService 按 OpenAPI 文档 key 过滤接口列表。
func (c *BaseAPICase) filterBaseAPIsByOpenAPIService(list []*models.BaseAPI, serviceCode string) []*models.BaseAPI {
	if serviceCode == "" {
		return list
	}
	values := make([]*models.BaseAPI, 0, len(list))
	for _, item := range list {
		document, found := c.openAPIRegistry.DocumentByOperation(item.Path, item.Method)
		if found && document.Key == serviceCode {
			values = append(values, item)
		}
	}
	return values
}

// filterBaseAPIsByToolPrompt 按工具提示词内容过滤接口列表。
func filterBaseAPIsByToolPrompt(list []*models.BaseAPI, keyword string) []*models.BaseAPI {
	if keyword == "" {
		return list
	}
	values := make([]*models.BaseAPI, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		prompts := parseToolPrompts(item.ToolPrompts)
		for _, prompt := range prompts {
			if !strings.Contains(prompt, keyword) {
				continue
			}
			values = append(values, item)
			break
		}
	}
	return values
}

// pageBaseAPIRecords 对 Go 侧过滤后的接口列表进行分页。
func pageBaseAPIRecords(list []*models.BaseAPI, pageNum, pageSize int64) []*models.BaseAPI {
	if pageSize <= 0 {
		return list
	}
	if pageNum <= 0 {
		pageNum = 1
	}
	start := (pageNum - 1) * pageSize
	if start >= int64(len(list)) {
		return []*models.BaseAPI{}
	}
	end := start + pageSize
	if end > int64(len(list)) {
		end = int64(len(list))
	}
	return list[start:end]
}

// normalizeToolPrompts 清理空工具提示词，保留非空提示词的原始内容。
func normalizeToolPrompts(prompts []string) []string {
	values := make([]string, 0, len(prompts))
	for _, item := range prompts {
		if item == "" {
			continue
		}
		values = append(values, item)
	}
	return values
}

// encodeToolPrompts 将工具提示词编码为数据库 JSON 字段。
func encodeToolPrompts(prompts []string) string {
	raw, err := json.Marshal(prompts)
	if err != nil {
		return "[]"
	}
	return string(raw)
}

// parseToolPrompts 解析数据库中的工具提示词 JSON。
func parseToolPrompts(value string) []string {
	if value == "" {
		return nil
	}
	var prompts []string
	err := json.Unmarshal([]byte(value), &prompts)
	if err != nil {
		return nil
	}
	return prompts
}

// buildBaseAPIDocParameters 构建请求参数文档。
func buildBaseAPIDocParameters(api *biz.OpenAPI, parameters []biz.Parameter) []*systemadminv1.BaseApiDocSchema {
	items := make([]*systemadminv1.BaseApiDocSchema, 0, len(parameters))
	for _, parameter := range parameters {
		item := buildBaseAPIDocSchema(api, parameter.Name, parameter.Name, parameter.In, parameter.Required, parameter.Schema, 0)
		if parameter.Description != "" {
			item.Description = parameter.Description
		}
		items = append(items, item)
	}
	return items
}

// buildBaseAPIDocSchema 展开 OpenAPI Schema 为前端可直接渲染的字段树。
func buildBaseAPIDocSchema(api *biz.OpenAPI, name, path, in string, required bool, schema biz.Schema, depth int) *systemadminv1.BaseApiDocSchema {
	schema, refName := dereferenceBaseAPIDocSchema(api, schema)
	item := &systemadminv1.BaseApiDocSchema{
		Name:        name,
		Path:        path,
		In:          in,
		Type:        schema.Type,
		Format:      schema.Format,
		Required:    required,
		Description: schema.Description,
		Ref:         refName,
		Enum:        schema.Enum,
	}
	if item.Type == "" {
		item.Type = inferBaseAPIDocSchemaType(schema)
	}
	if depth >= baseAPIDocMaxDepth {
		return item
	}
	if schema.Items != nil {
		child := buildBaseAPIDocSchema(api, name+"[]", path+"[]", in, false, *schema.Items, depth+1)
		item.Children = []*systemadminv1.BaseApiDocSchema{child}
	}
	if len(schema.Properties) > 0 {
		requiredFields := make(map[string]bool, len(schema.Required))
		for _, field := range schema.Required {
			requiredFields[field] = true
		}
		item.Children = make([]*systemadminv1.BaseApiDocSchema, 0, len(schema.Properties))
		fieldNames := make([]string, 0, len(schema.Properties))
		for fieldName := range schema.Properties {
			fieldNames = append(fieldNames, fieldName)
		}
		sort.Strings(fieldNames)
		for _, fieldName := range fieldNames {
			fieldSchema := schema.Properties[fieldName]
			fieldPath := fieldName
			if path != "" {
				fieldPath = path + "." + fieldName
			}
			item.Children = append(item.Children, buildBaseAPIDocSchema(api, fieldName, fieldPath, in, requiredFields[fieldName], fieldSchema, depth+1))
		}
	}
	return item
}

// dereferenceBaseAPIDocSchema 解析本地组件引用。
func dereferenceBaseAPIDocSchema(api *biz.OpenAPI, schema biz.Schema) (biz.Schema, string) {
	refName := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
	if refName == "" || api == nil {
		return schema, refName
	}
	refSchema, ok := api.Components.Schemas[refName]
	if !ok {
		return schema, refName
	}
	if refSchema.Description == "" {
		refSchema.Description = schema.Description
	}
	return refSchema, refName
}

// inferBaseAPIDocSchemaType 推断缺省 Schema 类型。
func inferBaseAPIDocSchemaType(schema biz.Schema) string {
	if len(schema.Properties) > 0 {
		return "object"
	}
	if schema.Items != nil {
		return "array"
	}
	if schema.Ref != "" {
		return "object"
	}
	return "string"
}

// buildBaseAPIDocRequestBody 构建请求体文档。
func buildBaseAPIDocRequestBody(api *biz.OpenAPI, requestBody *biz.RequestBody) *systemadminv1.BaseApiDocSchema {
	if requestBody == nil {
		return nil
	}
	schema := selectBaseAPIDocContentSchema(requestBody.Content)
	if schema == nil {
		return nil
	}
	item := buildBaseAPIDocSchema(api, "body", "body", "body", requestBody.Required, *schema, 0)
	if requestBody.Description != "" {
		item.Description = requestBody.Description
	}
	return item
}

// selectBaseAPIDocContentSchema 选择可展示的 JSON Schema。
func selectBaseAPIDocContentSchema(content map[string]biz.MediaType) *biz.Schema {
	if len(content) == 0 {
		return nil
	}
	if media, ok := content[baseAPIDocJSONMediaType]; ok {
		return &media.Schema
	}
	for _, media := range content {
		return &media.Schema
	}
	return nil
}

// buildBaseAPIDocResponses 构建响应文档。
func buildBaseAPIDocResponses(api *biz.OpenAPI, responses map[string]biz.Response) []*systemadminv1.BaseApiDocResponse {
	items := make([]*systemadminv1.BaseApiDocResponse, 0, len(responses))
	statuses := make([]string, 0, len(responses))
	for status := range responses {
		statuses = append(statuses, status)
	}
	sort.Strings(statuses)
	for _, status := range statuses {
		response := responses[status]
		schema := selectBaseAPIDocContentSchema(response.Content)
		item := &systemadminv1.BaseApiDocResponse{
			Status:      status,
			Description: response.Description,
		}
		if schema != nil {
			item.Body = buildBaseAPIDocSchema(api, "body", "body", "body", false, *schema, 0)
		}
		items = append(items, item)
	}
	return items
}

// matchAuthWhiteList 按认证白名单规则匹配当前接口操作名。
func matchAuthWhiteList(whiteList *bootstrapConfigv1.Authentication_Jwt_WhiteList, operation string) bool {
	// 白名单配置为空时，当前规则无需参与匹配。
	if whiteList == nil {
		return false
	}
	for _, prefix := range whiteList.GetPrefix() {
		// 前缀规则命中时，直接判定为免 token 接口。
		if strings.HasPrefix(operation, prefix) {
			return true
		}
	}
	for _, regexValue := range whiteList.GetRegex() {
		regex, err := regexp.Compile(regexValue)
		if err != nil {
			continue
		}
		// 正则完整命中当前操作名时，按白名单处理。
		if regex.FindString(operation) == operation {
			return true
		}
	}
	for _, path := range whiteList.GetPath() {
		// Path 精确匹配命中时，按白名单处理。
		if path == operation {
			return true
		}
	}
	for _, item := range whiteList.GetMatch() {
		// Match 精确匹配命中时，按白名单处理。
		if item == operation {
			return true
		}
	}
	return false
}
