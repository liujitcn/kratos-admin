package biz

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/resource/i18n"
	"github.com/liujitcn/kratos-core/resource/openapi"
	"github.com/liujitcn/kratos-core/resource/openapi/dto"
	"github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

// BaseAPICase 接口业务实例
type BaseAPICase struct {
	*biz.BaseCase
	*data.BaseAPIRepository
	baseAPII18NRepo *data.BaseAPII18NRepository
	catalog         *i18n.I18n
	mapper          *mapper.CopierMapper[adminv1.BaseApi, models.BaseAPI]
	openAPI         *openapi.OpenAPI
}

// NewBaseAPICase 创建接口业务实例
func NewBaseAPICase(
	baseCase *biz.BaseCase,
	openAPI *openapi.OpenAPI,
	catalog *i18n.I18n,
	baseAPIRepo *data.BaseAPIRepository,
	baseAPII18NRepo *data.BaseAPII18NRepository,
) *BaseAPICase {
	baseAPIMapper := mapper.NewCopierMapper[adminv1.BaseApi, models.BaseAPI]()
	baseAPIMapper.AppendConverters(mapper.NewJSONTypeConverter[[]string]().NewConverterPair())
	return &BaseAPICase{
		BaseCase:          baseCase,
		BaseAPIRepository: baseAPIRepo,
		baseAPII18NRepo:   baseAPII18NRepo,
		catalog:           catalog,
		mapper:            baseAPIMapper,
		openAPI:           openAPI,
	}
}

// OptionBaseAPI 查询菜单分配接口选项列表
func (c *BaseAPICase) OptionBaseAPI(ctx context.Context, _ *adminv1.OptionBaseApiRequest) (*adminv1.OptionBaseApiResponse, error) {
	query := c.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Order(query.ServiceName.Asc(), query.Operation.Asc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var i18ns map[string]*models.BaseAPII18N
	i18ns, err = c.baseAPII18NMap(ctx, list)
	if err != nil {
		return nil, err
	}

	baseAPIs := make([]*adminv1.BaseApi, 0, len(list))
	jwtCfg := c.GetConfig().GetAuthn().GetJwt()
	for _, item := range list {
		// 命中免 token 或可选鉴权规则的接口，不再返回给菜单管理页面。
		if jwtCfg != nil {
			isNoTokenOperation := matchAuthWhiteList(jwtCfg.GetWhiteList(), item.Operation) ||
				matchAuthWhiteList(jwtCfg.GetOptionalAuth(), item.Operation)
			if isNoTokenOperation {
				continue
			}
		}
		baseAPI := c.toBaseAPIDTO(ctx, item, i18ns[item.Operation])
		baseAPIs = append(baseAPIs, baseAPI)
	}

	return &adminv1.OptionBaseApiResponse{BaseApis: baseAPIs}, nil
}

// PageBaseAPI 分页查询接口列表
func (c *BaseAPICase) PageBaseAPI(ctx context.Context, req *adminv1.PageBaseApiRequest) (*adminv1.PageBaseApiResponse, error) {
	query := c.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 10)
	opts = append(opts, repository.Order(query.ID.Desc()))
	var list []*models.BaseAPI
	var total int64
	var err error
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
		serviceDescCondition := query.ServiceDesc.Like("%" + req.GetServiceDesc() + "%")
		i18nQuery := c.baseAPII18NRepo.Query(ctx).BaseAPII18N
		var translatedOperations []string
		translatedOperations, err = c.translatedOperations(ctx, i18nQuery.ServiceDesc.Like("%"+req.GetServiceDesc()+"%"))
		if err != nil {
			return nil, err
		}
		if len(translatedOperations) > 0 {
			serviceDescCondition = field.Or(serviceDescCondition, query.Operation.In(translatedOperations...))
		}
		opts = append(opts, repository.Where(serviceDescCondition))
	}
	// 传入描述关键字时，按接口描述模糊匹配。
	if req.GetDesc() != "" {
		descCondition := query.Desc.Like("%" + req.GetDesc() + "%")
		i18nQuery := c.baseAPII18NRepo.Query(ctx).BaseAPII18N
		var translatedOperations []string
		translatedOperations, err = c.translatedOperations(ctx, i18nQuery.Desc.Like("%"+req.GetDesc()+"%"))
		if err != nil {
			return nil, err
		}
		if len(translatedOperations) > 0 {
			descCondition = field.Or(descCondition, query.Operation.In(translatedOperations...))
		}
		opts = append(opts, repository.Where(descCondition))
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
	var i18ns map[string]*models.BaseAPII18N
	if req.GetToolPrompt() != "" || req.GetOpenapiServiceCode() != "" {
		list, err = c.List(ctx, opts...)
		if err != nil {
			return nil, err
		}
		if req.GetToolPrompt() != "" {
			i18ns, err = c.baseAPII18NMap(ctx, list)
			if err != nil {
				return nil, err
			}
			list = filterBaseAPIsByToolPrompt(list, req.GetToolPrompt(), i18ns)
		}
		list = c.filterBaseAPIsByOpenAPIService(ctx, list, req.GetOpenapiServiceCode())
		total = int64(len(list))
		list = pageBaseAPIRecords(list, req.GetPageNum(), req.GetPageSize())
	} else {
		list, total, err = c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
		if err != nil {
			return nil, err
		}
	}
	if i18ns == nil {
		i18ns, err = c.baseAPII18NMap(ctx, list)
		if err != nil {
			return nil, err
		}
	}

	baseAPIs := make([]*adminv1.BaseApi, 0, len(list))
	for _, item := range list {
		baseAPI := c.toBaseAPIDTO(ctx, item, i18ns[item.Operation])
		baseAPIs = append(baseAPIs, baseAPI)
	}

	return &adminv1.PageBaseApiResponse{
		BaseApis: baseAPIs,
		Total:    int32(total),
	}, nil
}

// GetBaseAPI 根据主键查询接口详情
func (c *BaseAPICase) GetBaseAPI(ctx context.Context, id int64) (*adminv1.BaseApi, error) {
	query := c.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Where(query.ID.Eq(id)))

	baseAPI, err := c.Find(ctx, opts...)
	if err != nil {
		return nil, err
	}

	var i18ns map[string]*models.BaseAPII18N
	i18ns, err = c.baseAPII18NMap(ctx, []*models.BaseAPI{baseAPI})
	if err != nil {
		return nil, err
	}
	return c.toBaseAPIDTO(ctx, baseAPI, i18ns[baseAPI.Operation]), nil
}

// GetBaseAPIDoc 查询接口 OpenAPI 文档
func (c *BaseAPICase) GetBaseAPIDoc(ctx context.Context, id int64) (*adminv1.BaseApiDoc, error) {
	query := c.Query(ctx).BaseAPI
	baseAPI, err := c.Find(ctx, repository.Where(query.ID.Eq(id)))
	if err != nil {
		return nil, err
	}
	coreDocument, err := c.openAPI.GetOperation(ctx, baseAPI.Path, baseAPI.Method)
	if err != nil {
		return nil, err
	}
	return mapBaseAPIDoc(baseAPI.ID, coreDocument), nil
}

// OptionOpenAPIService 查询 OpenAPI 文档选项。
func (c *BaseAPICase) OptionOpenAPIService(ctx context.Context, req *adminv1.OptionOpenApiServiceRequest) (*adminv1.OptionOpenApiServiceResponse, error) {
	services, err := c.openAPI.Services(ctx, req.GetServiceCode())
	if err != nil {
		return nil, err
	}
	options := make([]*adminv1.OpenApiServiceOption, 0, len(services))
	for _, service := range services {
		operations := make([]*adminv1.OpenApiServiceOperation, 0, len(service.Operations))
		for _, operation := range service.Operations {
			operations = append(operations, &adminv1.OpenApiServiceOperation{
				Path:   operation.Path,
				Method: operation.Method,
			})
		}
		options = append(options, &adminv1.OpenApiServiceOption{
			Key:        service.Key,
			Name:       c.localizeOpenAPIServiceName(ctx, service.Name),
			Operations: operations,
		})
	}
	return &adminv1.OptionOpenApiServiceResponse{List: options}, nil
}

// localizeOpenAPIServiceName 返回请求语言对应的 OpenAPI 服务名称。
func (c *BaseAPICase) localizeOpenAPIServiceName(ctx context.Context, name string) string {
	messageKey, found := c.catalog.KeyForSource(name)
	if !found {
		return name
	}
	return c.catalog.Localize(biz.LocaleFromContext(ctx), "", messageKey, nil, name)
}

// UpdateBaseAPI 更新接口 MCP、Agent 与工具提示词配置。
func (c *BaseAPICase) UpdateBaseAPI(ctx context.Context, req *adminv1.UpdateBaseApiRequest) error {
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
func (c *BaseAPICase) SetBaseAPIAgentStatus(ctx context.Context, req *adminv1.SetBaseApiAgentStatusRequest) error {
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
func (c *BaseAPICase) SetBaseAPIMcpStatus(ctx context.Context, req *adminv1.SetBaseApiMcpStatusRequest) error {
	query := c.Query(ctx).BaseAPI
	conditions := make([]gen.Condition, 0, 1)
	conditions = append(conditions, query.ID.Eq(req.GetId()))
	_, err := query.WithContext(ctx).
		Where(conditions...).
		UpdateSimple(query.McpStatus.Value(int32(req.GetMcpStatus())))
	return err
}

// toBaseAPIDTO 转换接口数据并补充所属 OpenAPI 文档信息。
func (c *BaseAPICase) toBaseAPIDTO(ctx context.Context, item *models.BaseAPI, i18n *models.BaseAPII18N) *adminv1.BaseApi {
	baseAPI := c.mapper.ToDTO(item)
	if i18n != nil {
		if i18n.ToolPrompts != "" {
			baseAPI.ToolPrompts = parseToolPrompts(i18n.ToolPrompts)
		}
		if i18n.ServiceDesc != "" {
			baseAPI.ServiceDesc = i18n.ServiceDesc
		}
		if i18n.Desc != "" {
			baseAPI.Desc = i18n.Desc
		}
	}
	document, found := c.openAPI.Service(ctx, item.Path, item.Method)
	if found {
		baseAPI.OpenapiServiceCode = document.Key
		baseAPI.OpenapiServiceName = c.localizeOpenAPIServiceName(ctx, document.Name)
	}
	return baseAPI
}

// filterBaseAPIsByOpenAPIService 按 OpenAPI 文档 key 过滤接口列表。
func (c *BaseAPICase) filterBaseAPIsByOpenAPIService(ctx context.Context, list []*models.BaseAPI, serviceCode string) []*models.BaseAPI {
	if serviceCode == "" {
		return list
	}
	values := make([]*models.BaseAPI, 0, len(list))
	for _, item := range list {
		document, found := c.openAPI.Service(ctx, item.Path, item.Method)
		if found && document.Key == serviceCode {
			values = append(values, item)
		}
	}
	return values
}

// filterBaseAPIsByToolPrompt 按工具提示词内容过滤接口列表。
func filterBaseAPIsByToolPrompt(list []*models.BaseAPI, keyword string, i18ns map[string]*models.BaseAPII18N) []*models.BaseAPI {
	if keyword == "" {
		return list
	}
	values := make([]*models.BaseAPI, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		matched := false
		for _, prompt := range parseToolPrompts(item.ToolPrompts) {
			if strings.Contains(prompt, keyword) {
				matched = true
				break
			}
		}
		if !matched {
			i18n := i18ns[item.Operation]
			if i18n != nil {
				for _, prompt := range parseToolPrompts(i18n.ToolPrompts) {
					if strings.Contains(prompt, keyword) {
						matched = true
						break
					}
				}
			}
		}
		if matched {
			values = append(values, item)
		}
	}
	return values
}

// baseAPII18NMap 查询当前语言对应的 API 翻译信息。
func (c *BaseAPICase) baseAPII18NMap(ctx context.Context, list []*models.BaseAPI) (map[string]*models.BaseAPII18N, error) {
	i18ns := make(map[string]*models.BaseAPII18N)
	operations := make([]string, 0, len(list))
	seen := make(map[string]struct{}, len(list))
	for _, item := range list {
		if item.Operation == "" {
			continue
		}
		if _, ok := seen[item.Operation]; ok {
			continue
		}
		seen[item.Operation] = struct{}{}
		operations = append(operations, item.Operation)
	}
	locale := biz.LocaleFromContext(ctx)
	if locale == "" || len(operations) == 0 {
		return i18ns, nil
	}
	query := c.baseAPII18NRepo.Query(ctx).BaseAPII18N
	rows, err := c.baseAPII18NRepo.List(ctx,
		repository.Where(query.Operation.In(operations...)),
		repository.Where(query.Locale.Eq(locale)),
	)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		i18ns[row.Operation] = row
	}
	return i18ns, nil
}

// translatedOperations 查询当前语言中匹配字段关键字的 API 操作名。
func (c *BaseAPICase) translatedOperations(ctx context.Context, condition gen.Condition) ([]string, error) {
	locale := biz.LocaleFromContext(ctx)
	if locale == "" {
		return nil, nil
	}
	query := c.baseAPII18NRepo.Query(ctx).BaseAPII18N
	rows, err := c.baseAPII18NRepo.List(ctx,
		repository.Where(query.Locale.Eq(locale)),
		repository.Where(condition),
	)
	if err != nil {
		return nil, err
	}
	operations := make([]string, 0, len(rows))
	for _, row := range rows {
		operations = append(operations, row.Operation)
	}
	return operations, nil
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
		if item != "" {
			values = append(values, item)
		}
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

// mapBaseAPIDoc 将 Core OpenAPI 文档转换为 Admin 接口响应。
func mapBaseAPIDoc(id int64, document *dto.OpenAPIOperationDocument) *adminv1.BaseApiDoc {
	if document == nil {
		return nil
	}
	parameters := make([]*adminv1.BaseApiDocSchema, 0, len(document.Parameters))
	for _, parameter := range document.Parameters {
		parameters = append(parameters, mapBaseAPIDocSchema(parameter))
	}
	responses := make([]*adminv1.BaseApiDocResponse, 0, len(document.Responses))
	for _, response := range document.Responses {
		responses = append(responses, &adminv1.BaseApiDocResponse{
			Status:      response.Status,
			Description: response.Description,
			Body:        mapBaseAPIDocSchema(response.Body),
		})
	}
	return &adminv1.BaseApiDoc{
		Id:          id,
		Summary:     document.Summary,
		Description: document.Description,
		Parameters:  parameters,
		RequestBody: mapBaseAPIDocSchema(document.RequestBody),
		Responses:   responses,
	}
}

// mapBaseAPIDocSchema 将 Core OpenAPI 字段结构转换为 Admin 接口字段结构。
func mapBaseAPIDocSchema(schema *dto.OpenAPISchema) *adminv1.BaseApiDocSchema {
	if schema == nil {
		return nil
	}
	children := make([]*adminv1.BaseApiDocSchema, 0, len(schema.Children))
	for _, child := range schema.Children {
		children = append(children, mapBaseAPIDocSchema(child))
	}
	return &adminv1.BaseApiDocSchema{
		Name:        schema.Name,
		Path:        schema.Path,
		In:          schema.In,
		Type:        schema.Type,
		Format:      schema.Format,
		Required:    schema.Required,
		Description: schema.Description,
		Ref:         schema.Ref,
		Enum:        append([]string(nil), schema.Enum...),
		Children:    children,
	}
}

// matchAuthWhiteList 按认证白名单规则匹配当前接口操作名。
func matchAuthWhiteList(whiteList *configv1.Authentication_Jwt_WhiteList, operation string) bool {
	if whiteList == nil {
		return false
	}
	for _, prefix := range whiteList.GetPrefix() {
		if strings.HasPrefix(operation, prefix) {
			return true
		}
	}
	var err error
	for _, regexValue := range whiteList.GetRegex() {
		var regex *regexp.Regexp
		regex, err = regexp.Compile(regexValue)
		if err != nil {
			continue
		}
		if regex.FindString(operation) == operation {
			return true
		}
	}
	for _, path := range whiteList.GetPath() {
		if path == operation {
			return true
		}
	}
	for _, item := range whiteList.GetMatch() {
		if item == operation {
			return true
		}
	}
	return false
}
