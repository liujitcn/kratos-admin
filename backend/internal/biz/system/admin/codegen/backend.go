package codegen

import (
	"fmt"
	"strings"

	"github.com/liujitcn/go-utils/stringcase"
)

// --- Go 业务层与服务层模板渲染 ---

// renderBackendBizFile 渲染后端业务文件内容。
func (c *renderer) renderBackendBizFile(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	entity := table.EntityName
	target := ProtoTargetForTable(table)
	columns = codeGenRequestColumns(table, columns)
	methods = filterProtoMethods(methods, c.defaultProtoPath(table))
	entityVar := stringcase.ToCamelCase(entity)
	repoField := entity + "Repository"
	queryName := entity
	modelType := "models." + entity
	dtoType := target.GoAlias + "." + entity
	formType := target.GoAlias + "." + entity + "Form"
	pageMethod := "Page" + entity
	pageResponse := target.GoAlias + ".Page" + entity + "Response"
	listField := stringcase.ToPascalCase(pluralize(entity))
	treeMethod := firstTreeMethodByTrigger(methods, TriggerPageTree)
	optionMethods := methodsByKinds(methods, APIKindOption, APIKindTree)
	defaultOrderOption := renderDefaultOrderOption(columns)
	orderOptionCount := 0
	if defaultOrderOption != "" {
		orderOptionCount = 1
	}
	var methodsBuilder strings.Builder
	if treeMethod != nil {
		if table.PageType == PageTypeTreeLazy {
			methodsBuilder.WriteString(c.renderTreeLazyPageBizMethod(table, columns, treeMethod))
		} else {
			methodsBuilder.WriteString(fmt.Sprintf(`// %s 查询%s树形列表。
func (c *%sCase) %s(ctx context.Context, req *systemadminv1.%sRequest) (*systemadminv1.%sResponse, error) {
	query := c.Query(ctx).%s
	opts := make([]repository.QueryOption, 0, %d)
%s%s
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return &systemadminv1.%sResponse{%s: c.build%sTree(list, 0)}, nil
}

`, goMethodName(treeMethod.MethodName), table.BusinessName, entity, goMethodName(treeMethod.MethodName), treeMethod.MethodName, treeMethod.MethodName, queryName, countQueryColumns(columns)+orderOptionCount, defaultOrderOption, c.renderQueryOptions(columns), goMethodName(treeMethod.MethodName), listField, entity))
		}
	}
	for _, method := range optionMethods {
		if method.APIKind == APIKindOption {
			methodsBuilder.WriteString(c.renderOptionBizMethod(table, columns, method))
			continue
		}
		if method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree {
			methodsBuilder.WriteString(c.renderTreeOptionBizMethod(table, columns, method))
		}
	}
	mainMethods := fmt.Sprintf(`// %s 查询%s分页列表。
func (c *%sCase) %s(ctx context.Context, req *systemadminv1.%sRequest) (*%s, error) {
	query := c.Query(ctx).%s
	opts := make([]repository.QueryOption, 0, %d)
%s%s
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}

	resList := make([]*systemadminv1.%s, 0, len(list))
	for _, item := range list {
		res := c.mapper.ToDTO(item)
		resList = append(resList, res)
	}
	return &%s{%s: resList, Total: int32(total)}, nil
}

// Get%s 查询%s详情。
func (c *%sCase) Get%s(ctx context.Context, id int64) (*systemadminv1.%sForm, error) {
	%s, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return c.formMapper.ToDTO(%s), nil
}

// Create%s 创建%s。
func (c *%sCase) Create%s(ctx context.Context, req *systemadminv1.%sForm) error {
	%s := c.formMapper.ToEntity(req)
	return c.Create(ctx, %s)
}

// Update%s 更新%s。
func (c *%sCase) Update%s(ctx context.Context, id int64, req *systemadminv1.%sForm) error {
	%s := c.formMapper.ToEntity(req)
	%s.ID = id
	return c.UpdateByID(ctx, %s)
}

// Delete%s 删除%s。
func (c *%sCase) Delete%s(ctx context.Context, ids string) error {
	return c.DeleteByIDs(ctx, _string.ConvertStringToInt64Array(ids))
}
	`, goMethodName(pageMethod), table.BusinessName, entity, goMethodName(pageMethod), pageMethod, pageResponse, queryName, countQueryColumns(columns)+orderOptionCount, defaultOrderOption, c.renderQueryOptions(columns), entity, pageResponse, listField, goMethodName(entity), table.BusinessName, entity, goMethodName(entity), entity, entityVar, entityVar, goMethodName(entity), table.BusinessName, entity, goMethodName(entity), entity, entityVar, entityVar, goMethodName(entity), table.BusinessName, entity, goMethodName(entity), entity, entityVar, entityVar, entityVar, goMethodName(entity), table.BusinessName, entity, goMethodName(entity))
	formGetAssignments := renderBackendFormMultipleGetAssignments(columns, entityVar)
	if formGetAssignments != "" {
		mainMethods = strings.Replace(
			mainMethods,
			"\treturn c.formMapper.ToDTO("+entityVar+"), nil",
			"\tres := c.formMapper.ToDTO("+entityVar+")\n"+formGetAssignments+"\treturn res, nil",
			1,
		)
	}
	formEntityAssignments := renderBackendFormMultipleEntityAssignments(columns, entityVar)
	jsonEntityAssignments := renderBackendJSONEntityAssignments(columns, entityVar)
	entityAssignments := formEntityAssignments + jsonEntityAssignments
	if entityAssignments != "" {
		mainMethods = strings.ReplaceAll(mainMethods, "\t"+entityVar+" := c.formMapper.ToEntity(req)\n", "\t"+entityVar+" := c.formMapper.ToEntity(req)\n"+entityAssignments)
	}
	if treeMethod != nil {
		getMethodIndex := strings.Index(mainMethods, "// "+goMethodName("Get"+entity))
		if getMethodIndex >= 0 {
			mainMethods = mainMethods[getMethodIndex:]
		}
	}
	mainMethods = replaceGeneratedDeleteMethod(mainMethods, entity, c.renderDeleteBizMethod(table, columns, treeMethod != nil))
	methodsBuilder.WriteString(mainMethods)
	for _, statusMethod := range methodsByKinds(methods, APIKindStatus) {
		statusColumn := findStatusColumn(columns, statusMethod.Name)
		if statusColumn == nil {
			continue
		}
		methodsBuilder.WriteString(fmt.Sprintf(`
// %s 设置%s状态。
func (c *%sCase) %s(ctx context.Context, req *systemadminv1.%sRequest) error {
	return c.UpdateByID(ctx, &models.%s{
		ID: req.GetId(),
		%s: req.GetStatus(),
	})
}
`, goMethodName(statusMethod.MethodName), DefaultString(statusColumn.Comment, statusColumn.Name), entity, goMethodName(statusMethod.MethodName), statusMethod.MethodName, entity, modelFieldName(statusColumn.Name)))
	}
	if treeMethod != nil && table.PageType != PageTypeTreeLazy {
		parentField := DefaultString(table.ParentColumn, "parent_id")
		methodsBuilder.WriteString(fmt.Sprintf(`
// build%sTree 构建%s树。
func (c *%sCase) build%sTree(list []*models.%s, parentID int64) []*systemadminv1.%s {
	res := make([]*systemadminv1.%s, 0)
	for _, item := range list {
		if item.%s != parentID {
			continue
		}
		%s := c.mapper.ToDTO(item)
		%s.Children = c.build%sTree(list, item.ID)
		res = append(res, %s)
	}
	return res
}
`, goMethodName(entity), table.BusinessName, entity, goMethodName(entity), entity, entity, entity, modelFieldName(parentField), entityVar, entityVar, goMethodName(entity), entityVar))
	}
	content := renderTemplate("backend_biz.tmpl", backendBizTemplateData{
		Entity:       entity,
		EntityVar:    entityVar,
		BusinessName: table.BusinessName,
		APIImport:    target.GoAlias + " \"" + target.GoImportPath + "\"",
		JSONImport:   renderBackendJSONImport(columns),
		ErrorImport:  renderBackendErrorImport(treeMethod != nil),
		Repository:   repoField,
		FormType:     formType,
		ModelType:    modelType,
		DTOType:      dtoType,
		CommonImport: renderGoCommonImport(optionMethods),
		Methods:      strings.ReplaceAll(methodsBuilder.String(), "systemadminv1.", target.GoAlias+"."),
	})
	content = removeGoReceiverMethods(content, missingCoreMethodNames(table, methods))
	// 没有时间区间查询时移除模板中不再使用的时间工具依赖。
	if !strings.Contains(content, "_time.") {
		content = strings.Replace(content, "\t_time \"github.com/liujitcn/go-utils/time\"\n", "", 1)
	}
	return reorderGoReceiverMethods(content, entity+"Case")
}

// renderTreeLazyPageBizMethod 渲染懒加载树形列表业务方法及子节点标记查询。
func (c *renderer) renderTreeLazyPageBizMethod(table *Table, columns []*CodeGenColumn, method *Proto) string {
	entity := table.EntityName
	parentColumn := DefaultString(table.ParentColumn, "parent_id")
	parentField := modelFieldName(parentColumn)
	parentGetter := "Get" + stringcase.ToPascalCase(parentColumn) + "()"
	listField := stringcase.ToPascalCase(pluralize(entity))
	defaultOrderOption := renderDefaultOrderOption(columns)
	orderOptionCount := 0
	if defaultOrderOption != "" {
		orderOptionCount = 1
	}
	return fmt.Sprintf(`// %s 查询%s懒加载树形列表。
func (c *%sCase) %s(ctx context.Context, req *systemadminv1.%sRequest) (*systemadminv1.%sResponse, error) {
	query := c.Query(ctx).%s
	opts := make([]repository.QueryOption, 0, %d)
	opts = append(opts, repository.Where(query.%s.Eq(req.%s)))
%s
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var hasChildren map[int64]struct{}
	hasChildren, err = c.list%sParentIDsWithChildren(ctx, list)
	if err != nil {
		return nil, err
	}
	resList := make([]*systemadminv1.%s, 0, len(list))
	for _, item := range list {
		res := c.mapper.ToDTO(item)
		_, res.HasChildren = hasChildren[item.ID]
		resList = append(resList, res)
	}
	return &systemadminv1.%sResponse{%s: resList}, nil
}

// list%sParentIDsWithChildren 查询当前层节点中存在子节点的编号。
func (c *%sCase) list%sParentIDsWithChildren(ctx context.Context, list []*models.%s) (map[int64]struct{}, error) {
	parentIDs := make([]int64, 0, len(list))
	for _, item := range list {
		parentIDs = append(parentIDs, int64(item.ID))
	}
	hasChildren := make(map[int64]struct{}, len(parentIDs))
	if len(parentIDs) == 0 {
		return hasChildren, nil
	}
	query := c.Query(ctx).%s
	childList, err := c.List(ctx, repository.Where(query.%s.In(parentIDs...)))
	if err != nil {
		return nil, err
	}
	for _, item := range childList {
		hasChildren[int64(item.%s)] = struct{}{}
	}
	return hasChildren, nil
}

`, goMethodName(method.MethodName), table.BusinessName, entity, goMethodName(method.MethodName), method.MethodName, method.MethodName, entity, orderOptionCount+1, parentField, parentGetter, defaultOrderOption, goMethodName(entity), entity, method.MethodName, listField, goMethodName(entity), entity, goMethodName(entity), entity, entity, parentField, parentField)
}

// renderDeleteBizMethod 渲染删除业务方法，树形页面额外阻止删除含有子节点的记录。
func (c *renderer) renderDeleteBizMethod(table *Table, columns []*CodeGenColumn, tree bool) string {
	entity := table.EntityName
	if !tree {
		return fmt.Sprintf(`// Delete%s 删除%s。
func (c *%sCase) Delete%s(ctx context.Context, ids string) error {
	return c.DeleteByIDs(ctx, _string.ConvertStringToInt64Array(ids))
}
`, entity, table.BusinessName, entity, entity)
	}
	parentColumn := DefaultString(table.ParentColumn, "parent_id")
	parentType := "int64"
	if parent := FindColumnByName(columns, parentColumn); parent != nil {
		parentType = DefaultString(parent.GoType, InferGoType(parent.DbType))
	}
	parentID := "parentID"
	if parentType == "int32" {
		parentID = "int32(parentID)"
	}
	return fmt.Sprintf(`// Delete%s 删除%s。
func (c *%sCase) Delete%s(ctx context.Context, ids string) error {
	idList := _string.ConvertStringToInt64Array(ids)
	if len(idList) == 0 {
		return nil
	}
	query := c.Query(ctx).%s
	for _, parentID := range idList {
		count, err := c.Count(ctx, repository.Where(query.%s.Eq(%s)))
		if err != nil {
			return err
		}
		if count > 0 {
			return errorsx.HasChildrenConflict("删除%s失败，下面有%s", "%s", "%s")
		}
	}
	return c.DeleteByIDs(ctx, idList)
}
`, entity, table.BusinessName, entity, entity, entity, modelFieldName(parentColumn), parentID, table.BusinessName, table.BusinessName, table.TableName_, table.TableName_)
}

// replaceGeneratedDeleteMethod 替换标准业务方法中的删除方法。
func replaceGeneratedDeleteMethod(content string, entity string, deleteMethod string) string {
	start := strings.Index(content, "// Delete"+entity+" ")
	if start < 0 {
		return content
	}
	closingBrace := strings.Index(content[start:], "\n}")
	if closingBrace < 0 {
		return content
	}
	end := start + closingBrace + len("\n}")
	if end < len(content) && content[end] == '\n' {
		end++
	}
	return content[:start] + deleteMethod + content[end:]
}

// renderBackendErrorImport 返回树删除校验所需的错误包导入。
func renderBackendErrorImport(tree bool) string {
	if !tree {
		return ""
	}
	return `"github.com/liujitcn/kratos-core/errorsx"`
}

// renderBackendJSONImport 判断生成业务文件是否需要 JSON 标准库导入。
func renderBackendJSONImport(columns []*CodeGenColumn) string {
	for _, column := range columns {
		if isGeneratedJSONScalar(column) {
			return "\t\"encoding/json\""
		}
	}
	return ""
}

// renderBackendJSONEntityAssignments 渲染 JSON 字段写入前的标准化逻辑。
func renderBackendJSONEntityAssignments(columns []*CodeGenColumn, entityVar string) string {
	var builder strings.Builder
	for _, column := range columns {
		if !isGeneratedJSONScalar(column) {
			continue
		}
		field := modelFieldName(column.Name)
		getter := "req.Get" + stringcase.ToPascalCase(column.Name) + "()"
		builder.WriteString(fmt.Sprintf("\tif %s.%s == \"\" {\n\t\t%s.%s = \"[]\"\n\t} else if !json.Valid([]byte(%s.%s)) {\n\t\traw, err := json.Marshal([]string{%s})\n\t\tif err != nil {\n\t\t\treturn err\n\t\t}\n\t\t%s.%s = string(raw)\n\t}\n", entityVar, field, entityVar, field, entityVar, field, getter, entityVar, field))
	}
	return builder.String()
}

// isGeneratedJSONScalar 判断是否需要将普通字符串适配为 JSON 字段值。
func isGeneratedJSONScalar(column *CodeGenColumn) bool {
	if column == nil || isFormTreeMultiple(column) || !strings.EqualFold(column.DbType, "json") {
		return false
	}
	return DefaultString(column.GoType, InferGoType(column.DbType)) == "string" && DefaultString(column.ProtoType, InferProtoType(column.DbType)) == "string"
}

// renderBackendServiceFile 渲染后端服务文件内容。
func (c *renderer) renderBackendServiceFile(table *Table, methods []*Proto) string {
	entity := table.EntityName
	target := ProtoTargetForTable(table)
	methods = filterProtoMethods(methods, c.defaultProtoPath(table))
	entityVar := stringcase.ToCamelCase(entity)
	var methodsBuilder strings.Builder
	if treeMethod := firstTreeMethodByTrigger(methods, TriggerPageTree); treeMethod != nil {
		methodsBuilder.WriteString(c.renderServiceMethod(table, treeMethod, entityVar, "tree", "查询"+table.BusinessName+"树形列表失败"))
	} else {
		methodsBuilder.WriteString(c.renderServiceMethod(table, &Proto{APIKind: APIKindList, MethodName: "Page" + entity}, entityVar, "page", "查询"+table.BusinessName+"分页列表失败"))
	}
	for _, method := range methodsByKinds(methods, APIKindOption, APIKindTree) {
		if method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree || method.APIKind == APIKindOption {
			methodsBuilder.WriteString(c.renderServiceMethod(table, method, entityVar, "option", "查询选项失败"))
		}
	}
	methodsBuilder.WriteString(c.renderServiceMethod(table, &Proto{APIKind: APIKindCRUD, MethodName: "Get" + entity}, entityVar, "get", "查询"+table.BusinessName+"失败"))
	methodsBuilder.WriteString(c.renderServiceMethod(table, &Proto{APIKind: APIKindCRUD, MethodName: "Create" + entity}, entityVar, "empty", "创建"+table.BusinessName+"失败"))
	methodsBuilder.WriteString(c.renderServiceMethod(table, &Proto{APIKind: APIKindCRUD, MethodName: "Update" + entity}, entityVar, "empty", "更新"+table.BusinessName+"失败"))
	methodsBuilder.WriteString(c.renderServiceMethod(table, &Proto{APIKind: APIKindCRUD, MethodName: "Delete" + entity}, entityVar, "empty", "删除"+table.BusinessName+"失败"))
	for _, statusMethod := range methodsByKinds(methods, APIKindStatus) {
		methodsBuilder.WriteString(c.renderServiceMethod(table, statusMethod, entityVar, "empty", "设置状态失败"))
	}
	content := renderTemplate("backend_service.tmpl", backendServiceTemplateData{
		Entity:       entity,
		EntityVar:    entityVar,
		BusinessName: table.BusinessName,
		APIAlias:     target.GoAlias,
		APIImport:    target.GoAlias + " \"" + target.GoImportPath + "\"",
		BizImport:    "\t\"" + target.BackendBizImportPath() + "\"",
		CommonImport: renderGoCommonImport(methodsByKinds(methods, APIKindOption, APIKindTree)),
		Methods:      strings.ReplaceAll(methodsBuilder.String(), "systemadminv1.", target.GoAlias+"."),
	})
	return reorderGoReceiverMethods(removeGoReceiverMethods(content, missingCoreMethodNames(table, methods)), entity+"Service")
}

// appendExternalTargetBizMethods 仅补齐已有业务文件缺失的外部目标选项方法。
func (c *renderer) appendExternalTargetBizMethods(content string, table *Table, methods []*Proto) string {
	generatedReceiver := table.EntityName + "Case"
	if goReceiverType(content, "Case") != generatedReceiver {
		return content
	}
	existingMethodNames := goReceiverMethodNames(content, generatedReceiver)
	missingMethods := make([]*Proto, 0, len(methods))
	for _, method := range methods {
		if _, exists := existingMethodNames[method.MethodName]; !exists {
			missingMethods = append(missingMethods, method)
		}
	}
	if len(missingMethods) == 0 {
		return content
	}
	candidate := c.renderExternalTargetBizFile(table, missingMethods)
	methodNames := goReceiverMethodNames(candidate, generatedReceiver)
	methodContent := strings.Join(extractGoMethods(candidate, methodNames), "\n\n")
	if methodContent == "" {
		return content
	}
	target := ProtoTargetForTable(table)
	return mergeGeneratedGoReceiverMethods(content, methodContent, generatedReceiver, target.GoAlias+" \""+target.GoImportPath+"\"")
}

// renderExternalTargetBizFile 渲染外部目标实体的最小业务文件。
func (c *renderer) renderExternalTargetBizFile(table *Table, methods []*Proto) string {
	entity := table.EntityName
	entityVar := stringcase.ToCamelCase(entity)
	repoField := entity + "Repository"
	target := ProtoTargetForTable(table)
	return renderTemplate("backend_external_biz.tmpl", backendBizTemplateData{
		Entity:       entity,
		EntityVar:    entityVar,
		BusinessName: table.BusinessName,
		APIImport:    target.GoAlias + " \"" + target.GoImportPath + "\"",
		Repository:   repoField,
		Methods:      strings.ReplaceAll(c.renderExternalTargetBizMethods(table, methods), "systemadminv1.", target.GoAlias+"."),
	})
}

// renderExternalTargetBizMethods 渲染外部目标实体选项业务方法。
func (c *renderer) renderExternalTargetBizMethods(table *Table, methods []*Proto) string {
	var builder strings.Builder
	for _, method := range methods {
		if method.APIKind == APIKindOption {
			builder.WriteString(c.renderOptionBizMethod(table, nil, method))
			continue
		}
		if method.APIKind == APIKindTree {
			builder.WriteString(c.renderTreeOptionBizMethod(table, nil, method))
		}
	}
	return builder.String()
}

// appendExternalTargetServiceMethods 仅补齐已有服务文件缺失的外部目标选项方法。
func (c *renderer) appendExternalTargetServiceMethods(content string, table *Table, methods []*Proto) string {
	generatedReceiver := table.EntityName + "Service"
	if goReceiverType(content, "Service") != generatedReceiver {
		return content
	}
	existingMethodNames := goReceiverMethodNames(content, generatedReceiver)
	missingMethods := make([]*Proto, 0, len(methods))
	for _, method := range methods {
		if _, exists := existingMethodNames[method.MethodName]; !exists {
			missingMethods = append(missingMethods, method)
		}
	}
	if len(missingMethods) == 0 {
		return content
	}
	candidate := c.renderExternalTargetServiceFile(table, missingMethods)
	methodNames := goReceiverMethodNames(candidate, generatedReceiver)
	methodContent := strings.Join(extractGoMethods(candidate, methodNames), "\n\n")
	if methodContent == "" {
		return content
	}
	target := ProtoTargetForTable(table)
	return mergeGeneratedGoReceiverMethods(content, methodContent, generatedReceiver, target.GoAlias+" \""+target.GoImportPath+"\"")
}

// renderExternalTargetServiceFile 渲染外部目标实体的最小服务文件。
func (c *renderer) renderExternalTargetServiceFile(table *Table, methods []*Proto) string {
	entity := table.EntityName
	entityVar := stringcase.ToCamelCase(entity)
	target := ProtoTargetForTable(table)
	var methodsBuilder strings.Builder
	for _, method := range methods {
		methodsBuilder.WriteString(c.renderServiceMethod(table, method, entityVar, "option", "查询选项失败"))
	}
	return renderTemplate("backend_external_service.tmpl", backendServiceTemplateData{
		Entity:       entity,
		EntityVar:    entityVar,
		BusinessName: table.BusinessName,
		APIAlias:     target.GoAlias,
		APIImport:    target.GoAlias + " \"" + target.GoImportPath + "\"",
		BizImport:    "\t\"" + target.BackendBizImportPath() + "\"",
		Methods:      strings.ReplaceAll(methodsBuilder.String(), "systemadminv1.", target.GoAlias+"."),
	})
}

// renderBackendFormMultipleGetAssignments 渲染 JSON 字段回填到多选表单数组的语句。
func renderBackendFormMultipleGetAssignments(columns []*CodeGenColumn, entityVar string) string {
	var builder strings.Builder
	for _, column := range columns {
		if !isFormTreeMultiple(column) {
			continue
		}
		field := modelFieldName(column.Name)
		builder.WriteString(fmt.Sprintf("\tres.%s = _string.ConvertJsonStringToInt64Array(%s.%s)\n", field, entityVar, field))
	}
	return builder.String()
}

// renderBackendFormMultipleEntityAssignments 渲染多选表单数组写回 JSON 字段的语句。
func renderBackendFormMultipleEntityAssignments(columns []*CodeGenColumn, entityVar string) string {
	var builder strings.Builder
	for _, column := range columns {
		if !isFormTreeMultiple(column) {
			continue
		}
		field := modelFieldName(column.Name)
		builder.WriteString(fmt.Sprintf("\t%s.%s = _string.ConvertInt64ArrayToString(req.Get%s())\n", entityVar, field, field))
	}
	return builder.String()
}

// renderGoCommonImport 按需渲染 commonv1 导入。
func renderGoCommonImport(methods []*Proto) string {
	for _, method := range methods {
		if method.APIKind == APIKindOption {
			return "\tcommonv1 \"github.com/liujitcn/kratos-core/api/gen/go/common/v1\""
		}
		if method.APIKind == APIKindTree && (method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree) {
			return "\tcommonv1 \"github.com/liujitcn/kratos-core/api/gen/go/common/v1\""
		}
	}
	return ""
}
