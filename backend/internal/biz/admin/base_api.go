package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	commonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/common/v1"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	coreOpenAPI "github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	"github.com/liujitcn/kratos-admin/backend/internal/biz"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	kitutils "github.com/liujitcn/kratos-kit/utils"
	"gopkg.in/yaml.v3"
	"gorm.io/gen"
)

const (
	baseAPIDocJSONMediaType = "application/json"
	baseAPIDocMaxDepth      = 8
)

// OpenAPI 描述 OpenAPI 文档结构。
type OpenAPI struct {
	Paths      map[string]PathItem `yaml:"paths"`
	Tags       []TagsItem          `yaml:"tags"`
	Components Components          `yaml:"components"`
}

// PathItem 描述单个路径的请求方法。
type PathItem struct {
	Get    *Operation `yaml:"get,omitempty"`
	Post   *Operation `yaml:"post,omitempty"`
	Put    *Operation `yaml:"put,omitempty"`
	Delete *Operation `yaml:"delete,omitempty"`
}

// PathOperation 描述单个路径下的 HTTP 操作。
type PathOperation struct {
	Method    string
	Operation *Operation
}

// TagsItem 描述 OpenAPI 标签信息。
type TagsItem struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// Operation 描述单个接口操作项。
type Operation struct {
	Tags        []string            `yaml:"tags"`
	Summary     string              `yaml:"summary"`
	Description string              `yaml:"description"`
	OperationID string              `yaml:"operationId"`
	Parameters  []Parameter         `yaml:"parameters"`
	RequestBody *RequestBody        `yaml:"requestBody"`
	Responses   map[string]Response `yaml:"responses"`
}

// Components 描述 OpenAPI 组件定义。
type Components struct {
	Schemas map[string]Schema `yaml:"schemas"`
}

// Parameter 描述 OpenAPI 请求参数。
type Parameter struct {
	Name        string `yaml:"name"`
	In          string `yaml:"in"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Schema      Schema `yaml:"schema"`
}

// RequestBody 描述 OpenAPI 请求体。
type RequestBody struct {
	Description string               `yaml:"description"`
	Required    bool                 `yaml:"required"`
	Content     map[string]MediaType `yaml:"content"`
}

// Response 描述 OpenAPI 响应。
type Response struct {
	Description string               `yaml:"description"`
	Content     map[string]MediaType `yaml:"content"`
}

// MediaType 描述 OpenAPI 媒体类型。
type MediaType struct {
	Schema Schema `yaml:"schema"`
}

// Schema 描述 OpenAPI Schema。
type Schema struct {
	Ref         string            `yaml:"$ref"`
	Type        string            `yaml:"type"`
	Format      string            `yaml:"format"`
	Description string            `yaml:"description"`
	Enum        []string          `yaml:"enum"`
	Required    []string          `yaml:"required"`
	Properties  map[string]Schema `yaml:"properties"`
	Items       *Schema           `yaml:"items"`
}

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

// InitializeOpenAPI 根据全部具名 OpenAPI 文档同步接口元数据。
func (c *BaseAPICase) InitializeOpenAPI(ctx context.Context, documents []coreOpenAPI.Document) error {
	baseAPIList := make([]*models.BaseAPI, 0)
	var err error
	for _, document := range documents {
		var documentBaseAPIs []*models.BaseAPI
		documentBaseAPIs, err = c.openAPIDataToBaseAPI(document.Data)
		if err != nil {
			return fmt.Errorf("解析 OpenAPI 文档 %q: %w", document.Key, err)
		}
		baseAPIList = append(baseAPIList, documentBaseAPIs...)
	}

	// 所有模块文档收集完成后一次重建接口数据，避免扩展模块接口缺失或重复同步。
	err = c.batchCreateBaseAPI(ctx, baseAPIList)
	if err != nil {
		return err
	}
	return nil
}

// openAPIDataToBaseAPI 将 OpenAPI 文档转换为待持久化的接口模型。
//
// 此方法只负责内存转换，不访问数据库。转换分为两阶段：先按“终端 + 服务名”
// 推断完整 protobuf 包名，再生成每个 HTTP operation，避免不同终端的同名服务互相覆盖。
func (c *BaseAPICase) openAPIDataToBaseAPI(openAPIData []byte) ([]*models.BaseAPI, error) {
	api, err := ParseOpenAPI(openAPIData)
	if err != nil {
		return nil, err
	}

	// OpenAPI operationId 不含 protobuf 包名，必须先从请求和响应 schema 建立服务包名索引。
	servicePackages := buildServicePackageMap(api.Paths)
	// tag 只提供展示描述，索引键仍使用完整服务名以消除同名 tag 的歧义。
	tagsMap := buildTagsMap(api.Tags, servicePackages)

	baseAPIList := make([]*models.BaseAPI, 0)
	for path, item := range api.Paths {
		for _, operation := range pathOperations(item) {
			var baseAPI *models.BaseAPI
			baseAPI, err = parseOperation(path, operation.Method, operation.Operation, tagsMap, servicePackages)
			if err != nil {
				return nil, err
			}
			// 当前路径存在对应 HTTP 操作时，写入接口权限与 MCP 元数据。
			if baseAPI != nil {
				baseAPIList = append(baseAPIList, baseAPI)
			}
		}
	}

	// 数据库存储和后续 Casbin 重建都以 operation 为标识，排序保证每次生成顺序稳定。
	sort.Slice(baseAPIList, func(i, j int) bool {
		return baseAPIList[i].Operation < baseAPIList[j].Operation
	})
	return baseAPIList, nil
}

// batchCreateBaseAPI 按当前 OpenAPI 完整重建接口数据。
//
// 启动链路会在本方法之后重建 Casbin 内存策略，因此这里以 OpenAPI 为唯一事实来源，
// 先清空旧接口记录并重置自增 ID，再批量写入本次转换结果。
func (c *BaseAPICase) batchCreateBaseAPI(ctx context.Context, apis []*models.BaseAPI) error {
	query := c.Query(ctx).BaseAPI
	// 接口定义完全以当前 OpenAPI 为准，清空表并重置自增 ID 后重新生成。
	if err := query.WithContext(ctx).UnderlyingDB().Exec("TRUNCATE TABLE `base_api`").Error; err != nil { //nolint:forbidigo // TRUNCATE 重置自增 ID，gorm/gen 无法表达
		return err
	}
	return c.BatchCreate(ctx, apis)
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
	var api *OpenAPI
	api, err = ParseOpenAPI(document.Data)
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

// Operation 按精确的 HTTP path 和 method 查询 OpenAPI 操作定义。
//
// 未找到路径、文档为空或 method 不在当前权限同步支持的 GET、POST、PUT、DELETE 范围时返回 nil。
func (api *OpenAPI) Operation(path, method string) *Operation {
	if api == nil {
		return nil
	}
	item, ok := api.Paths[path]
	if !ok {
		return nil
	}
	// HTTP 方法决定同一路径下需要返回的操作定义。
	switch method {
	case "GET":
		return item.Get
	case "POST":
		return item.Post
	case "PUT":
		return item.Put
	case "DELETE":
		return item.Delete
	default:
		return nil
	}
}

// ParseOpenAPI 解析 OpenAPI YAML 文档。
//
// 该方法只做 YAML 解码并完整保留 paths、tags 与 schemas，业务层可同时复用同一份文档
// 生成权限数据和展示接口文档。
func ParseOpenAPI(openAPIData []byte) (*OpenAPI, error) {
	var api OpenAPI
	err := yaml.Unmarshal(openAPIData, &api)
	if err != nil {
		return nil, err
	}
	return &api, nil
}

// buildServicePackageMap 从全部 OpenAPI operation 的 schema 引用推断服务所属 protobuf 包。
//
// 返回键为 servicePackageKey(path, operationId服务名)，值为唯一候选包。一个服务的多个
// operation 可以共同提供 schema 线索；只要候选包不唯一就不返回该服务，宁可跳过错误记录，
// 也不能把权限 operation 绑定到另一个模块。
func buildServicePackageMap(paths map[string]PathItem) map[string]string {
	candidatesByService := make(map[string]map[string]struct{})
	for path, item := range paths {
		for _, pathOperation := range pathOperations(item) {
			// 当前 HTTP 方法未声明 operation 时，没有可供推断的 operationId 或 schema。
			if pathOperation.Operation == nil {
				continue
			}
			serviceTag, _ := parseOperationID(pathOperation.Operation.OperationID)
			serviceKey := servicePackageKey(path, serviceTag)
			// operationId 或主版本路径无效时，不参与服务包名推断。
			if serviceKey == "" {
				continue
			}
			// 同一个服务的每个 HTTP operation 都为所属包提供候选，集合自动消除重复引用。
			for packageName := range operationProtoPackages(path, pathOperation.Operation) {
				if candidatesByService[serviceKey] == nil {
					candidatesByService[serviceKey] = make(map[string]struct{})
				}
				candidatesByService[serviceKey][packageName] = struct{}{}
			}
		}
	}

	servicePackages := make(map[string]string, len(candidatesByService))
	for serviceKey, candidates := range candidatesByService {
		// 同一终端服务只允许归属一个包，避免共享 DTO 导致权限 operation 绑定到错误模块。
		if len(candidates) != 1 {
			continue
		}
		for packageName := range candidates {
			servicePackages[serviceKey] = packageName
		}
	}
	return servicePackages
}

// pathOperations 返回参与接口权限同步的 HTTP 操作。
//
// 当前生成链路只把 GET、POST、PUT、DELETE 写入 base_api；未列出的 HEAD、OPTIONS、PATCH
// 不会生成权限记录，新增受支持方法时必须同时扩展此处和 OpenAPI.Operation。
func pathOperations(item PathItem) []PathOperation {
	return []PathOperation{
		{Method: "GET", Operation: item.Get},
		{Method: "POST", Operation: item.Post},
		{Method: "PUT", Operation: item.Put},
		{Method: "DELETE", Operation: item.Delete},
	}
}

// parseOperationID 从 OpenAPI operationId 拆出 RPC 服务名与方法名。
//
// protoc-gen-openapi 按 Service_Method 生成 operationId；两段分别用于构造 service_name
// 与 Kratos 运行时 operation。格式不合法的 operation 不写入权限表，避免生成无法匹配的策略。
func parseOperationID(operationID string) (string, string) {
	parts := strings.SplitN(operationID, "_", 2)
	// operationId 必须满足 Service_Method 格式才能还原 Kratos 运行时 operation。
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", ""
	}
	return parts[0], parts[1]
}

// servicePackageKey 生成“终端 + 服务名”唯一索引键。
//
// 同一个 Service 可以同时出现在不同终端，因此不能仅用服务名定位 protobuf 包。
func servicePackageKey(path, serviceTag string) string {
	terminal := openAPITerminal(path)
	// 非主版本路径或缺少服务名时，没有可用于包名推断的稳定索引。
	if terminal == "" || serviceTag == "" {
		return ""
	}
	return terminal + "\x00" + serviceTag
}

// openAPITerminal 获取 /api/v1/{terminal}/... 路径中的终端段。
//
// 仅主版本路径参与权限同步；additional_bindings 产生的兼容旧路径不应重复生成同一个 RPC 权限。
func openAPITerminal(path string) string {
	paths := strings.Split(strings.Trim(path, "/"), "/")
	// 路径必须同时包含 api、版本和终端三个固定段。
	if len(paths) < 3 || paths[0] != "api" || paths[1] != "v1" {
		return ""
	}
	return paths[2]
}

// operationProtoPackages 收集一个 HTTP operation 中属于当前终端的 protobuf 包。
//
// 请求参数、请求体和响应可能引用不同的 schema；统一收集后由 buildServicePackageMap
// 判断服务是否只有一个候选包，防止共享 DTO 或错误引用污染权限 operation。
func operationProtoPackages(path string, operation *Operation) map[string]struct{} {
	packages := make(map[string]struct{})
	// 缺少 operation 时没有 schema 可以用于包名推断。
	if operation == nil {
		return packages
	}
	for _, parameter := range operation.Parameters {
		collectSchemaProtoPackages(path, parameter.Schema, packages)
	}
	// 请求体的每种媒体类型都可能定义独立 schema，需要全部检查。
	if operation.RequestBody != nil {
		for _, mediaType := range operation.RequestBody.Content {
			collectSchemaProtoPackages(path, mediaType.Schema, packages)
		}
	}
	// 响应按状态码和媒体类型分组，任一分支都可提供所属 protobuf 包线索。
	for _, response := range operation.Responses {
		for _, mediaType := range response.Content {
			collectSchemaProtoPackages(path, mediaType.Schema, packages)
		}
	}
	return packages
}

// collectSchemaProtoPackages 递归收集 schema 及其内嵌字段中属于当前终端的 protobuf 包。
//
// 共享 schema（如 common.v1）会出现在多种业务 operation 中，只有包名终端与 HTTP 路径终端
// 一致时才成为候选，避免将共享类型误认为服务所属包。
func collectSchemaProtoPackages(path string, schema Schema, packages map[string]struct{}) {
	packageName := protoPackageFromSchemaRef(schema.Ref)
	// schema 引用的包与当前 HTTP 终端一致时，才能作为服务所属包名候选。
	if packageName != "" && protoPackageTerminal(packageName) == openAPITerminal(path) {
		packages[packageName] = struct{}{}
	}
	for _, property := range schema.Properties {
		collectSchemaProtoPackages(path, property, packages)
	}
	// 数组元素也可能直接引用另一个 protobuf message，不能只检查数组外层 schema。
	if schema.Items != nil {
		collectSchemaProtoPackages(path, *schema.Items, packages)
	}
}

// protoPackageFromSchemaRef 从 OpenAPI schema 引用提取完整 protobuf 包名。
//
// 在 fq_schema_naming=true 时，引用包含完整 protobuf 包名和消息名。
// 项目约定 protobuf 包以版本段结束，因此第一个 vN 分段及其前缀就是包名，后续部分是 message 名。
func protoPackageFromSchemaRef(ref string) string {
	const schemaRefPrefix = "#/components/schemas/"
	schemaName := strings.TrimPrefix(ref, schemaRefPrefix)
	// 非组件 schema 引用不能可靠还原 protobuf 包名。
	if schemaName == ref {
		return ""
	}

	parts := strings.Split(schemaName, ".")
	for index, part := range parts {
		// 版本段后必须仍有 message 名，避免把不完整的 schema 名误判为包名。
		if isProtoPackageVersion(part) && index > 0 && index < len(parts)-1 {
			return strings.Join(parts[:index+1], ".")
		}
	}
	return ""
}

// isProtoPackageVersion 判断包名分段是否为以数字开头的版本段。
//
// 除 v1 外，也兼容 v2、v1beta1 等版本命名；只要 v 后首字符为数字，即可作为包名边界。
func isProtoPackageVersion(value string) bool {
	return len(value) > 1 && value[0] == 'v' && value[1] >= '0' && value[1] <= '9'
}

// protoPackageTerminal 返回 protobuf 包版本段前的终端名称。
//
// 例如 system.admin.v1 返回 admin，base.v1 返回 base；该规则不依赖具体模块列表。
func protoPackageTerminal(packageName string) string {
	parts := strings.Split(packageName, ".")
	// 包名必须以版本段结尾，版本段前一段即为 HTTP 终端。
	if len(parts) < 2 || !isProtoPackageVersion(parts[len(parts)-1]) {
		return ""
	}
	return parts[len(parts)-2]
}

// buildTagsMap 构建“完整服务名 -> 服务描述”的索引。
//
// OpenAPI tag 名可能在 admin 与 app 重复，因此先由 tag 描述确定终端，再结合服务包索引生成完整键。
func buildTagsMap(tags []TagsItem, servicePackages map[string]string) map[string]string {
	tagsMap := make(map[string]string, len(tags))
	for _, item := range tags {
		terminal := terminalByTagDescription(item.Description)
		packageName := servicePackages[terminal+"\x00"+item.Name]
		// 标签描述可确定终端和服务所属包时，才写入完整服务名索引。
		if packageName != "" {
			tagsMap[fmt.Sprintf("%s.%s", packageName, item.Name)] = item.Description
		}
	}
	return tagsMap
}

// terminalByTagDescription 根据生成器写入的 tag 描述前缀识别 HTTP 终端。
//
// 该值仅用于查找展示描述，不参与 service_name 或 operation 的生成。
func terminalByTagDescription(description string) string {
	// Admin 标签描述对应管理后台服务。
	if strings.HasPrefix(description, "Admin") {
		return "admin"
	}
	// App 标签描述对应应用端服务。
	if strings.HasPrefix(description, "App") {
		return "app"
	}
	// Base 标签描述对应公共基础服务。
	if strings.HasPrefix(description, "Base") {
		return "base"
	}
	return ""
}

// parseOperation 将一个 HTTP operation 转换为 base_api 记录。
//
// service_name 固定为 {protobuf包}.{operationId服务名}，operation 固定为
// /{protobuf包}.{operationId服务名}/{operationId方法名}，两者与 Kratos 运行时鉴权值保持一致。
func parseOperation(path, method string, operationDefinition *Operation, tagsMap map[string]string, servicePackages map[string]string) (*models.BaseAPI, error) {
	// 操作项为空时，当前请求方法无需生成接口权限数据。
	if operationDefinition == nil {
		return nil, nil
	}

	serviceTag, methodName := parseOperationID(operationDefinition.OperationID)
	// operationId 缺少服务或方法时无法还原运行时 operation，直接跳过异常数据。
	if serviceTag == "" || methodName == "" {
		return nil, nil
	}
	packageName := servicePackages[servicePackageKey(path, serviceTag)]
	// 未能从当前终端和服务名确定唯一 protobuf 包时，跳过异常数据，避免写入错误权限。
	if packageName == "" {
		return nil, nil
	}

	serviceName := fmt.Sprintf("%s.%s", packageName, serviceTag)
	operation := fmt.Sprintf("/%s/%s", serviceName, methodName)
	serviceDesc := tagsMap[serviceName]
	// tag 只作为服务描述回退来源，不能替换 serviceName，否则会和 operation 的服务段不一致。
	if len(operationDefinition.Tags) > 0 {
		tagServiceName := fmt.Sprintf("%s.%s", packageName, operationDefinition.Tags[0])
		if value, ok := tagsMap[tagServiceName]; ok {
			serviceDesc = value
		}
	}

	baseAPI := &models.BaseAPI{
		ToolName:    kitutils.ToolNameFromRPCPath(operation),
		ServiceName: serviceName,
		ServiceDesc: serviceDesc,
		Desc:        operationDescription(operationDefinition),
		Operation:   operation,
		Method:      method,
		Path:        path,
		McpStatus:   int32(commonv1.Status_ENABLE),
		AgentStatus: int32(commonv1.Status_ENABLE),
	}
	baseAPI.ToolPrompts = encodeToolPrompts(defaultToolPrompts(baseAPI.ToolName, baseAPI.ServiceName, baseAPI.ServiceDesc, baseAPI.Desc, baseAPI.Operation, baseAPI.Method, baseAPI.Path))
	return baseAPI, nil
}

// operationDescription 获取接口面向管理端和工具目录的说明文本。
// description 用于完整说明，缺失时才使用简短的 summary。
func operationDescription(operation *Operation) string {
	if operation.Description != "" {
		return operation.Description
	}
	return operation.Summary
}

// defaultToolPrompts 根据接口元数据生成 MCP 与 Agent 的默认检索提示词。
//
// 每条提示词只表达一个稳定维度：服务/接口业务语义、终端语义、工具名、RPC operation 或 HTTP
// 路由。最后统一去重，既便于自然语言命中，也让管理端能看出一条接口对应的实际调用标识。
func defaultToolPrompts(toolName, serviceName, serviceDesc, desc, operation, method, path string) []string {
	values := make([]string, 0, 6)
	// 服务描述与接口描述同时存在时，保留组合提示，增强完整语义命中。
	if serviceDesc != "" && desc != "" {
		values = append(values, serviceDesc+"："+desc)
	}
	if desc != "" {
		values = append(values, desc)
	}
	if serviceDesc != "" && desc == "" {
		values = append(values, serviceDesc)
	}
	terminal := terminalToolPrompt(serviceName)
	if terminal != "" && desc != "" {
		values = append(values, terminal+"："+desc)
	}
	if toolName != "" {
		values = append(values, "工具名："+toolName)
	}
	if operation != "" {
		values = append(values, "RPC："+operation)
	}
	if method != "" || path != "" {
		values = append(values, strings.TrimSpace("HTTP："+method+" "+path))
	}
	return uniqueToolPrompts(values)
}

// terminalToolPrompt 根据完整服务名生成面向用户的终端提示词。
//
// 服务名中的版本段前一段就是终端，例如 system.admin.v1.AuthService 的 admin，
// 这个解析与包名推断相同，不依赖具体模块名称。
func terminalToolPrompt(serviceName string) string {
	parts := strings.Split(serviceName, ".")
	for index, part := range parts {
		// 服务名中的版本段前一段为终端，兼容新增模块和版本。
		if !isProtoPackageVersion(part) || index == 0 || index == len(parts)-1 {
			continue
		}
		// 根据 HTTP 终端生成面向用户的提示词。
		switch parts[index-1] {
		case "admin":
			return "管理后台"
		case "app":
			return "应用移动端"
		case "base":
			return "公共基础"
		}
	}
	return ""
}

// uniqueToolPrompts 按首次出现顺序去除空提示词和重复提示词。
//
// 保持顺序可使默认提示词在每次 OpenAPI 同步后稳定，避免无意义的数据库内容变化。
func uniqueToolPrompts(prompts []string) []string {
	values := make([]string, 0, len(prompts))
	seen := make(map[string]struct{}, len(prompts))
	for _, item := range prompts {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		values = append(values, item)
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
func buildBaseAPIDocParameters(api *OpenAPI, parameters []Parameter) []*systemadminv1.BaseApiDocSchema {
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
func buildBaseAPIDocSchema(api *OpenAPI, name, path, in string, required bool, schema Schema, depth int) *systemadminv1.BaseApiDocSchema {
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
func dereferenceBaseAPIDocSchema(api *OpenAPI, schema Schema) (Schema, string) {
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
func inferBaseAPIDocSchemaType(schema Schema) string {
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
func buildBaseAPIDocRequestBody(api *OpenAPI, requestBody *RequestBody) *systemadminv1.BaseApiDocSchema {
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
func selectBaseAPIDocContentSchema(content map[string]MediaType) *Schema {
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
func buildBaseAPIDocResponses(api *OpenAPI, responses map[string]Response) []*systemadminv1.BaseApiDocResponse {
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
