package codegen

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/go-utils/stringcase"
)

// --- 命令执行、源码排序与补充消息 ---

// RunCommand 在后端目录执行单个固定 Make 目标及其变量。
func RunCommand(ctx context.Context, backendDir string, target string, variables ...string) (string, error) {
	args := append([]string{target}, variables...)
	command := exec.CommandContext(ctx, "make", args...)
	command.Dir = backendDir
	output, err := command.CombinedOutput()
	safeOutput := TruncateText(redactCodeGenCommandOutput(string(output)), CommandOutputMaxRunes)
	if safeOutput == "" && err != nil {
		safeOutput = err.Error()
	}
	return safeOutput, err
}

// CommandFailureMessage 生成适合列表展示的命令错误摘要。
func CommandFailureMessage(target string, output string, err error) string {
	detail := strings.Join(strings.Fields(output), " ")
	if detail == "" {
		detail = err.Error()
	}
	return "代码生成失败（make " + target + "）：" + detail
}

// NormalizeQueryOperator 统一当前支持的查询方式。
func NormalizeQueryOperator(operator string) string {
	switch operator {
	case "like", "between":
		return operator
	default:
		return "eq"
	}
}

// goMethodName 将 Proto 方法名转换为与 protoc 生成接口一致的 Go 方法名。
func goMethodName(methodName string) string {
	return stringcase.ToPascalCase(methodName)
}

// ExistingProtoFilePath 查找仓库中已定义目标 RPC 的 Proto 文件。
func ExistingProtoFilePath(targetEntity string, methodName string, excludedModule string) (string, bool) {
	if targetEntity == "" || methodName == "" {
		return "", false
	}
	root := repoRoot()
	pattern := filepath.Join(root, ProtoRootPath, "*", "admin", "v1", stringcase.ToSnakeCase(targetEntity)+".proto")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return "", false
	}
	for _, skipExcluded := range []bool{true, false} {
		for _, path := range paths {
			relativeDir, relativeErr := filepath.Rel(root, filepath.Dir(path))
			if relativeErr != nil {
				continue
			}
			module := strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(relativeDir), ProtoRootPath+"/"), "/admin/v1")
			if skipExcluded && module == excludedModule {
				continue
			}
			content, readErr := os.ReadFile(path)
			if readErr == nil && protoServiceMethodExists(string(content), targetEntity+"Service", methodName) {
				relativePath, relativeErr := filepath.Rel(root, path)
				if relativeErr == nil {
					return filepath.ToSlash(relativePath), true
				}
			}
		}
	}
	return "", false
}

// FailureRemark 提取适合保存和展示的生成错误信息。
func FailureRemark(err error) string {
	var structuredError *kratosErrors.Error
	if errors.As(err, &structuredError) && structuredError.Message != "" {
		return TruncateText(structuredError.Message, RemarkMaxRunes)
	}
	return TruncateText(err.Error(), RemarkMaxRunes)
}

// TruncateText 按字符数截断命令输出和数据库备注。
func TruncateText(value string, maxRunes int) string {
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

// BuildProgressSteps 构建文件、菜单和命令的完整进度步骤。
func BuildProgressSteps(files []*systemadminv1.CodeGenPreviewFile, syncMenus bool, runCommands bool) []*systemadminv1.CodeGenTaskStep {
	stepCount := len(files)
	if syncMenus {
		stepCount++
	}
	if runCommands {
		stepCount += 6
	}
	steps := make([]*systemadminv1.CodeGenTaskStep, 0, stepCount)
	for i, file := range files {
		status := systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_PENDING
		message := "等待生成"
		if file.GetAction() != "create" && file.GetAction() != "update" {
			status = systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_SKIPPED
			message = file.GetMessage()
		}
		steps = append(steps, &systemadminv1.CodeGenTaskStep{
			Id:      FileStepID(i),
			Label:   "生成文件",
			Kind:    "file",
			Path:    file.GetPath(),
			Status:  status,
			Message: message,
		})
	}
	if syncMenus {
		steps = append(steps, &systemadminv1.CodeGenTaskStep{
			Id:      MenuStepID,
			Label:   "同步菜单权限",
			Kind:    "menu",
			Status:  systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_PENDING,
			Message: "等待同步",
		})
	}
	if runCommands {
		for _, command := range []string{"gorm-gen", "api", "openapi", "ts", "wire", "fmt"} {
			steps = append(steps, &systemadminv1.CodeGenTaskStep{
				Id:      CommandStepPrefix + command,
				Label:   "make " + command,
				Kind:    "command",
				Status:  systemadminv1.CodeGenTaskStepStatus_CODE_GEN_TASK_STEP_STATUS_PENDING,
				Message: "等待执行",
			})
		}
	}
	return steps
}

// FileStepID 返回文件步骤的稳定标识。
func FileStepID(index int) string {
	return fmt.Sprintf("file:%d", index)
}

// defaultTargetProtoPath 返回目标实体默认 Proto 路径。
func (c *renderer) defaultTargetProtoPath(table *Table, target string) string {
	if target == table.EntityName {
		return c.defaultProtoPath(table)
	}
	if path, ok := ExistingProtoFilePath(target, "Option"+target, table.BusinessModule); ok {
		return path
	}
	return ProtoFilePath(ProtoTargetForTable(table).Directory, target)
}

// renderGeneratedProtoMessages 渲染勾选补齐接口所需 message。
func (c *renderer) renderGeneratedProtoMessages(table *Table, columns []*CodeGenColumn, methods []*Proto, protoPath string) string {
	var builder strings.Builder
	for _, method := range filterProtoMethods(methods, protoPath) {
		if method.TriggerType == TriggerCRUD || method.TriggerType == TriggerPageTree {
			continue
		}
		builder.WriteString(c.renderGeneratedProtoMessage(table, columns, method))
	}
	return builder.String()
}

// renderPatchProtoMessages 渲染追加或修复 RPC 依赖的 message。
func (c *renderer) renderPatchProtoMessages(table *Table, columns []*CodeGenColumn, method *Proto, scheduled map[string]struct{}) []string {
	names := c.protoMessageNamesForMethod(table, method)
	list := make([]string, 0, len(names))
	for _, name := range names {
		if _, ok := scheduled[name]; ok {
			continue
		}
		scheduled[name] = struct{}{}
		list = append(list, c.renderProtoMessageByName(table, columns, method, name))
	}
	return list
}

// protoMessageNamesForMethod 返回方法依赖的 message 名称。
func (c *renderer) protoMessageNamesForMethod(table *Table, method *Proto) []string {
	entity := table.EntityName
	switch method.MethodName {
	case "Page" + entity:
		return []string{"Page" + entity + "Request", "Page" + entity + "Response", entity}
	case "Get" + entity:
		return []string{"Get" + entity + "Request", entity + "Form"}
	case "Create" + entity:
		return []string{"Create" + entity + "Request", entity + "Form"}
	case "Update" + entity:
		return []string{"Update" + entity + "Request", entity + "Form"}
	case "Delete" + entity:
		return []string{"Delete" + entity + "Request"}
	default:
		if method.APIKind == APIKindTree && method.TriggerType == TriggerPageTree {
			return []string{method.MethodName + "Request", method.MethodName + "Response", entity}
		}
		return []string{method.MethodName + "Request"}
	}
}

// renderProtoMessageByName 按名称渲染 message。
func (c *renderer) renderProtoMessageByName(table *Table, columns []*CodeGenColumn, method *Proto, name string) string {
	// Proto 字段顺序以代码生成字段配置的排序值为准，相同排序值保持原有顺序。
	columns = slices.Clone(columns)
	slices.SortStableFunc(columns, func(left *CodeGenColumn, right *CodeGenColumn) int {
		if left.Sort < right.Sort {
			return -1
		}
		if left.Sort > right.Sort {
			return 1
		}
		return 0
	})
	entity := table.EntityName
	pluralEntity := pluralize(entity)
	switch name {
	case "Page" + entity + "Request":
		var builder strings.Builder
		builder.WriteString(fmt.Sprintf("// %s分页查询条件\nmessage %s {\n", table.BusinessName, name))
		fieldNo := int32(1)
		for _, column := range columns {
			if !generatedRequestIncludesColumn(table, column) {
				continue
			}
			builder.WriteString(c.renderQueryProtoField(column, fieldNo))
			fieldNo++
		}
		builder.WriteString("  int64 page_num = 101 [(gnostic.openapi.v3.property) = {description: \"当前页码\"}]; // 当前页码\n\n")
		builder.WriteString("  int64 page_size = 102 [(gnostic.openapi.v3.property) = {description: \"每一页的行数\"}]; // 每一页的行数\n")
		builder.WriteString("}\n")
		return builder.String()
	case "Page" + entity + "Response":
		return fmt.Sprintf("// %s分页响应\nmessage %s {\n  repeated %s %s = 1 [(gnostic.openapi.v3.property) = {description: \"%s列表\"}]; // %s列表\n\n  int32 total = 2 [(gnostic.openapi.v3.property) = {description: \"总数\"}]; // 总数\n}\n", table.BusinessName, name, entity, stringcase.ToSnakeCase(pluralEntity), table.BusinessName, table.BusinessName)
	case "Get" + entity + "Request":
		return fmt.Sprintf("// %s详情查询条件\nmessage %s {\n  int64 id = 1 [(gnostic.openapi.v3.property) = {description: \"%sID\"}]; // %sID\n}\n", table.BusinessName, name, table.BusinessName, table.BusinessName)
	case "Create" + entity + "Request":
		return fmt.Sprintf("// %s创建条件\nmessage %s {\n  %sForm %s = 1 [(gnostic.openapi.v3.property) = {description: \"%s表单\"}, (buf.validate.field).required = true]; // %s表单\n}\n", table.BusinessName, name, entity, stringcase.ToSnakeCase(entity), table.BusinessName, table.BusinessName)
	case "Update" + entity + "Request":
		return fmt.Sprintf("// %s更新条件\nmessage %s {\n  int64 id = 1 [(gnostic.openapi.v3.property) = {description: \"%sID\"}]; // %sID\n\n  %sForm %s = 2 [(gnostic.openapi.v3.property) = {description: \"%s表单\"}, (buf.validate.field).required = true]; // %s表单\n}\n", table.BusinessName, name, table.BusinessName, table.BusinessName, entity, stringcase.ToSnakeCase(entity), table.BusinessName, table.BusinessName)
	case "Delete" + entity + "Request":
		return fmt.Sprintf("// %s删除条件\nmessage %s {\n  string ids = 1 [(gnostic.openapi.v3.property) = {description: \"%sID列表，多个用逗号分隔\"}]; // %sID列表，多个用逗号分隔\n}\n", table.BusinessName, name, table.BusinessName, table.BusinessName)
	case entity:
		return c.renderProtoMessage(table, columns, false)
	case entity + "Form":
		return c.renderProtoMessage(table, columns, true)
	default:
		if method.APIKind == APIKindTree && method.TriggerType == TriggerPageTree {
			return c.renderTreeProtoMessage(table, method, columns, name)
		}
		return c.renderGeneratedProtoMessage(table, columns, method)
	}
}

// renderGeneratedProtoMessage 渲染单个补齐接口的请求 message。
func (c *renderer) renderGeneratedProtoMessage(table *Table, columns []*CodeGenColumn, method *Proto) string {
	businessName := codeGenProtoMethodBusinessName(table, method)
	switch method.APIKind {
	case APIKindOption:
		return fmt.Sprintf("// %s下拉选择查询条件\nmessage %sRequest {}\n\n", businessName, method.MethodName)
	case APIKindTree:
		if method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree {
			parentColumn := DefaultString(method.ParentColumn, "parent_id")
			return fmt.Sprintf("// %s树形选择查询条件\nmessage %sRequest {\n  optional int64 %s = 1 [(gnostic.openapi.v3.property) = {description: \"父节点ID\"}]; // 父节点ID\n\n  optional bool lazy = 2 [(gnostic.openapi.v3.property) = {description: \"是否懒加载\"}]; // 是否懒加载\n}\n\n", businessName, method.MethodName, parentColumn)
		}
		return c.renderTreeProtoMessage(table, method, columns, method.MethodName+"Request") + "\n" + c.renderTreeProtoMessage(table, method, columns, method.MethodName+"Response")
	case APIKindStatus:
		statusColumn := findStatusColumn(columns, method.Name)
		return fmt.Sprintf("// %s状态设置条件\nmessage %sRequest {\n  int64 id = 1 [(gnostic.openapi.v3.property) = {description: \"%sID\"}]; // %sID\n\n  %s status = 2 [(gnostic.openapi.v3.property) = {description: \"状态\"}]; // 状态\n}\n\n", table.BusinessName, method.MethodName, table.BusinessName, table.BusinessName, statusProtoType(statusColumn))
	default:
		return ""
	}
}

// renderTreeProtoMessage 按消息名渲染树形列表的请求或响应，避免同一方法重复输出整组消息。
func (c *renderer) renderTreeProtoMessage(table *Table, method *Proto, columns []*CodeGenColumn, name string) string {
	switch name {
	case method.MethodName + "Request":
		var builder strings.Builder
		builder.WriteString(fmt.Sprintf("// %s树形列表查询条件\nmessage %s {\n", table.BusinessName, name))
		fieldNo := int32(1)
		seenFields := make(map[string]struct{})
		for _, column := range columns {
			if !generatedRequestIncludesColumn(table, column) {
				continue
			}
			builder.WriteString(c.renderQueryProtoField(column, fieldNo))
			seenFields[column.Name] = struct{}{}
			fieldNo++
		}
		if table.PageType == PageTypeTreeLazy {
			parentColumn := DefaultString(table.ParentColumn, "parent_id")
			if _, exists := seenFields[parentColumn]; !exists {
				builder.WriteString(fmt.Sprintf("  optional int64 %s = %d [(gnostic.openapi.v3.property) = {description: \"父节点ID\"}]; // 父节点ID\n\n", parentColumn, fieldNo))
				fieldNo++
			}
			builder.WriteString(fmt.Sprintf("  optional bool lazy = %d [(gnostic.openapi.v3.property) = {description: \"是否懒加载\"}]; // 是否懒加载\n\n", fieldNo))
		}
		builder.WriteString("}\n")
		return builder.String()
	case method.MethodName + "Response":
		return fmt.Sprintf("// %s树形列表响应\nmessage %s {\n  repeated %s %s = 1 [(gnostic.openapi.v3.property) = {description: \"%s树形列表\"}]; // %s树形列表\n}\n", table.BusinessName, name, table.EntityName, stringcase.ToSnakeCase(pluralize(table.EntityName)), table.BusinessName, table.BusinessName)
	default:
		return ""
	}
}

// renderQueryProtoField 渲染分页查询字段，区间查询使用数组承接起止值。
func (c *renderer) renderQueryProtoField(column *CodeGenColumn, fieldNo int32) string {
	if effectiveQueryOperator(column) != "between" {
		return c.renderProtoField(column, fieldNo, true, false)
	}
	protoType := DefaultString(column.ProtoType, InferProtoType(column.DbType))
	comment := DefaultString(column.Comment, column.Name)
	return fmt.Sprintf("  repeated %s %s = %d [(gnostic.openapi.v3.property) = {description: %q}]; // %s\n\n", protoType, column.Name, fieldNo, comment, comment)
}

// renderProtoField 渲染 Proto 字段。
func (c *renderer) renderProtoField(column *CodeGenColumn, fieldNo int32, optional bool, form bool) string {
	protoType := DefaultString(column.ProtoType, InferProtoType(column.DbType))
	comment := DefaultString(column.Comment, column.Name)
	prefix := ""
	if optional {
		prefix = "optional "
	}
	validation := ""
	if form && DefaultString(column.TsType, InferTSType(column.DbType)) == "string" && column.DbLength > 0 {
		expression := fmt.Sprintf("this.size() <= %d", column.DbLength)
		message := fmt.Sprintf("%s不能超过 %d 个字符", comment, column.DbLength)
		if generatedFormRequired(column) {
			expression = fmt.Sprintf("this.size() > 0 && this.size() <= %d", column.DbLength)
			message = fmt.Sprintf("%s不能为空且不超过 %d 个字符", comment, column.DbLength)
		}
		validation = fmt.Sprintf(", (buf.validate.field).cel = {id: %q message: %q expression: %q}", "system.admin.code.gen.table.field."+column.Name+".length", message, expression)
	}
	return fmt.Sprintf("  %s%s %s = %d [(gnostic.openapi.v3.property) = {description: %q}%s]; // %s\n\n", prefix, protoType, column.Name, fieldNo, comment, validation, comment)
}

// renderFormTreeMultipleProtoField 渲染多选树形字段的数组契约。
func (c *renderer) renderFormTreeMultipleProtoField(column *CodeGenColumn, fieldNo int32) string {
	comment := DefaultString(column.Comment, column.Name)
	return fmt.Sprintf("  repeated int64 %s = %d [(gnostic.openapi.v3.property) = {description: %q}]; // %s\n\n", column.Name, fieldNo, comment, comment)
}

// renderQueryOptions 渲染后端分页查询条件。
func (c *renderer) renderQueryOptions(columns []*CodeGenColumn) string {
	var builder strings.Builder
	for _, column := range columns {
		if !generatedQueryIncludesColumn(column) {
			continue
		}
		modelField := modelFieldName(column.Name)
		requestField := stringcase.ToPascalCase(column.Name)
		getter := "Get" + requestField + "()"
		switch effectiveQueryOperator(column) {
		case "like":
			builder.WriteString(fmt.Sprintf("\tif req.%s != \"\" {\n\t\topts = append(opts, repository.Where(query.%s.Like(\"%%\"+req.%s+\"%%\")))\n\t}\n", getter, modelField, getter))
		case "between":
			rangeVar := stringcase.ToCamelCase(column.Name) + "Range"
			builder.WriteString(fmt.Sprintf(`	%s := req.%s
	// 仅在传入完整时间区间时追加范围条件。
	if len(%s) == 2 {
		startTime := _time.StringTimeToTime(%s[0])
		endTime := _time.StringTimeToTime(%s[1])
		// 开始时间可解析时追加查询下界。
		if startTime != nil {
			opts = append(opts, repository.Where(query.%s.Gte(*startTime)))
		}
		// 结束时间可解析时包含所选结束日期全天。
		if endTime != nil {
			opts = append(opts, repository.Where(query.%s.Lt(endTime.AddDate(0, 0, 1))))
		}
	}
`, rangeVar, getter, rangeVar, rangeVar, rangeVar, modelField, modelField))
		default:
			builder.WriteString(fmt.Sprintf("\tif req.%s != nil {\n\t\topts = append(opts, repository.Where(query.%s.Eq(req.%s)))\n\t}\n", requestField, modelField, getter))
		}
	}
	return builder.String()
}

// renderOptionBizMethod 渲染普通下拉选择业务方法。
func (c *renderer) renderOptionBizMethod(table *Table, columns []*CodeGenColumn, method *Proto) string {
	labelField := modelFieldName(DefaultString(method.LabelColumn, "name"))
	valueExpr := fmt.Sprintf("int64(item.%s)", modelFieldName(DefaultString(method.ValueColumn, "id")))
	disabledExpr := renderOptionDisabledExpression(columns, method)
	defaultOrderOption := renderDefaultOrderOption(columns)
	orderOptionCount := 0
	queryDeclaration := ""
	if defaultOrderOption != "" {
		orderOptionCount = 1
		queryDeclaration = fmt.Sprintf("\tquery := c.Query(ctx).%s\n", table.EntityName)
	}
	return fmt.Sprintf(`// %s 查询%s下拉选择。
func (c *%sCase) %s(ctx context.Context, _ *systemadminv1.%sRequest) (*commonv1.SelectOptionResponse, error) {
%s
	opts := make([]repository.QueryOption, 0, %d)
%s
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	options := make([]*commonv1.SelectOptionResponse_Option, 0, len(list))
	for _, item := range list {
		options = append(options, &commonv1.SelectOptionResponse_Option{Label: fmt.Sprint(item.%s), Value: %s%s})
	}
	return &commonv1.SelectOptionResponse{List: options}, nil
}

`, goMethodName(method.MethodName), table.BusinessName, table.EntityName, goMethodName(method.MethodName), method.MethodName, queryDeclaration, orderOptionCount, defaultOrderOption, labelField, valueExpr, disabledExpr)
}

// renderTreeOptionBizMethod 渲染树形选择业务方法。
func (c *renderer) renderTreeOptionBizMethod(table *Table, columns []*CodeGenColumn, method *Proto) string {
	parentColumn := DefaultString(method.ParentColumn, "parent_id")
	parentField := modelFieldName(parentColumn)
	parentGetter := "Get" + stringcase.ToPascalCase(parentColumn) + "()"
	labelField := modelFieldName(DefaultString(method.LabelColumn, "name"))
	valueExpr := fmt.Sprintf("int64(item.%s)", modelFieldName(DefaultString(method.ValueColumn, "id")))
	disabledExpr := renderOptionDisabledExpression(columns, method)
	defaultOrderOption := renderDefaultOrderOption(columns)
	orderOptionCount := 0
	queryDeclaration := ""
	if defaultOrderOption != "" {
		orderOptionCount = 1
		queryDeclaration = fmt.Sprintf("\tquery := c.Query(ctx).%s\n", table.EntityName)
	}
	return fmt.Sprintf(`// %s 查询%s树形选择。
func (c *%sCase) %s(ctx context.Context, req *systemadminv1.%sRequest) (*commonv1.TreeOptionResponse, error) {
%s
	opts := make([]repository.QueryOption, 0, %d)
%s
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	lazy := req.GetLazy()
	if %t && req.Lazy == nil {
		lazy = true
	}
	return &commonv1.TreeOptionResponse{List: c.build%sOption(list, req.%s, lazy)}, nil
}

// build%sOption 构建%s树形选择。
func (c *%sCase) build%sOption(list []*models.%s, parentID int64, lazy bool) []*commonv1.TreeOptionResponse_Option {
	res := make([]*commonv1.TreeOptionResponse_Option, 0)
	for _, item := range list {
		if int64(item.%s) != parentID {
			continue
		}
		option := &commonv1.TreeOptionResponse_Option{Label: fmt.Sprint(item.%s), Value: %s%s}
		if lazy {
			option.HasChildren = c.has%sOptionChildren(list, %s)
		} else {
			option.Children = c.build%sOption(list, %s, false)
		}
		res = append(res, option)
	}
	return res
}

// has%sOptionChildren 判断树形选择节点是否存在子节点。
func (c *%sCase) has%sOptionChildren(list []*models.%s, parentID int64) bool {
	for _, item := range list {
		if int64(item.%s) == parentID {
			return true
		}
	}
	return false
}

`, goMethodName(method.MethodName), table.BusinessName, table.EntityName, goMethodName(method.MethodName), method.MethodName, queryDeclaration, orderOptionCount, defaultOrderOption, method.Lazy, goMethodName(method.MethodName), parentGetter, goMethodName(method.MethodName), table.BusinessName, table.EntityName, goMethodName(method.MethodName), table.EntityName, parentField, labelField, valueExpr, disabledExpr, goMethodName(method.MethodName), valueExpr, goMethodName(method.MethodName), valueExpr, goMethodName(method.MethodName), table.EntityName, goMethodName(method.MethodName), table.EntityName, parentField)
}

// renderServiceMethod 渲染服务层方法。
func (c *renderer) renderServiceMethod(table *Table, method *Proto, entityVar string, returnKind string, errorMessage string) string {
	goName := goMethodName(method.MethodName)
	returnType := "(*emptypb.Empty, error)"
	successReturn := "return new(emptypb.Empty), nil"
	callArgs := "ctx, req"
	switch returnKind {
	case "page":
		returnType = fmt.Sprintf("(*systemadminv1.%sResponse, error)", method.MethodName)
		successReturn = "return res, nil"
	case "get":
		returnType = fmt.Sprintf("(*systemadminv1.%sForm, error)", table.EntityName)
		successReturn = "return res, nil"
		callArgs = "ctx, req.GetId()"
	case "tree":
		returnType = fmt.Sprintf("(*systemadminv1.%sResponse, error)", method.MethodName)
		successReturn = "return res, nil"
	case "option":
		if method.APIKind == APIKindTree {
			returnType = "(*commonv1.TreeOptionResponse, error)"
		} else {
			returnType = "(*commonv1.SelectOptionResponse, error)"
		}
		successReturn = "return res, nil"
	}
	caseCall := fmt.Sprintf("s.%sCase.%s(%s)", entityVar, goName, callArgs)
	if method.MethodName == "Create"+table.EntityName {
		caseCall = fmt.Sprintf("s.%sCase.%s(ctx, req.Get%s())", entityVar, goName, table.EntityName)
	}
	if method.MethodName == "Update"+table.EntityName {
		caseCall = fmt.Sprintf("s.%sCase.%s(ctx, req.GetId(), req.Get%s())", entityVar, goName, table.EntityName)
	}
	if method.MethodName == "Delete"+table.EntityName {
		caseCall = fmt.Sprintf("s.%sCase.%s(ctx, req.GetIds())", entityVar, goName)
	}
	assign := "res, err :="
	if returnKind == "empty" {
		assign = "err :="
	}
	return fmt.Sprintf(`
// %s %s。
func (s *%sService) %s(ctx context.Context, req *systemadminv1.%sRequest) %s {
	%s %s
	if err != nil {
		log.Error(%q, "error", err)
		return nil, errorsx.WrapInternal(err, %q)
	}
	%s
}
`, goName, errorMessage, table.EntityName, goName, method.MethodName, returnType, assign, caseCall, goName, errorMessage, successReturn)
}

// renderDefaultOrderOption 根据字段快照渲染默认排序，优先创建时间，其次主键。
func renderDefaultOrderOption(columns []*CodeGenColumn) string {
	var primaryColumn *CodeGenColumn
	for _, column := range columns {
		if column.Name == "created_at" {
			return "\topts = append(opts, repository.Order(query." + modelFieldName(column.Name) + ".Desc()))\n"
		}
		if primaryColumn == nil && column.IsPrimary == 1 {
			primaryColumn = column
		}
	}
	if primaryColumn == nil {
		return ""
	}
	return "\topts = append(opts, repository.Order(query." + modelFieldName(primaryColumn.Name) + ".Desc()))\n"
}

// resourcePathByEntity 根据实体名推导管理端资源路径。
func resourcePathByEntity(entity string) string {
	return strings.ReplaceAll(stringcase.ToSnakeCase(entity), "_", "/")
}

// redactCodeGenCommandOutput 脱敏命令输出中的数据库连接和密码参数。
func redactCodeGenCommandOutput(output string) string {
	output = commandSourcePattern.ReplaceAllString(output, `${1}'***'`)
	output = commandDSNPattern.ReplaceAllString(output, "***")
	output = commandSecretPattern.ReplaceAllString(output, "${1}***")
	return strings.TrimSpace(output)
}

// reorderGoReceiverMethods 重排指定接收者的方法，并将夹在其中的包级辅助函数保留在方法组之后。
func reorderGoReceiverMethods(content string, receiverName string) string {
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return content
	}
	type sourceRange struct {
		start            int
		end              int
		declarationIndex int
	}
	blocks := make([]CodeGenSourceMethodBlock, 0)
	ranges := make([]sourceRange, 0)
	firstDeclarationIndex := -1
	lastDeclarationIndex := -1
	for declarationIndex, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || len(function.Recv.List) == 0 || goReceiverName(function.Recv.List[0].Type) != receiverName {
			continue
		}
		start := fileSet.Position(function.Pos()).Offset
		if function.Doc != nil {
			start = fileSet.Position(function.Doc.Pos()).Offset
		}
		end := fileSet.Position(function.End()).Offset
		if start < 0 || end > len(content) || start >= end {
			return content
		}
		blocks = append(blocks, CodeGenSourceMethodBlock{
			Name:          function.Name.Name,
			Content:       content[start:end],
			Start:         start,
			End:           end,
			OriginalIndex: len(blocks),
		})
		ranges = append(ranges, sourceRange{start: start, end: end, declarationIndex: declarationIndex})
		if firstDeclarationIndex < 0 {
			firstDeclarationIndex = declarationIndex
		}
		lastDeclarationIndex = declarationIndex
	}
	if len(blocks) < 2 {
		return content
	}
	// 仅跨越普通包级辅助函数；变量、类型、init 或其他接收者的方法仍是人工组织边界。
	for declarationIndex := firstDeclarationIndex + 1; declarationIndex < lastDeclarationIndex; declarationIndex++ {
		function, ok := file.Decls[declarationIndex].(*ast.FuncDecl)
		if !ok {
			return content
		}
		if function.Recv != nil && len(function.Recv.List) > 0 && goReceiverName(function.Recv.List[0].Type) == receiverName {
			continue
		}
		if function.Recv != nil || function.Name.Name == "init" {
			return content
		}
	}
	start := blocks[0].Start
	// 包级辅助函数后的方法被移走时，同时移除方法前的空白，避免末尾遗留重复空行。
	for i := 1; i < len(ranges); i++ {
		previousFunction, ok := file.Decls[ranges[i].declarationIndex-1].(*ast.FuncDecl)
		if !ok || previousFunction.Recv != nil || previousFunction.Name.Name == "init" {
			continue
		}
		for ranges[i].start > 0 {
			character := content[ranges[i].start-1]
			if character != ' ' && character != '\t' && character != '\r' && character != '\n' {
				break
			}
			ranges[i].start--
		}
	}
	// 除最后一个方法外，移除其后的空白，避免留下重复空行。
	for i := 0; i < len(ranges)-1; i++ {
		for ranges[i].end < len(content) {
			character := content[ranges[i].end]
			if character != ' ' && character != '\t' && character != '\r' && character != '\n' {
				break
			}
			ranges[i].end++
		}
	}
	// 仅移动当前接收者的方法块，包级辅助函数仍保留在原位置。
	sortCodeGenSourceMethodBlocks(blocks)
	methodContents := make([]string, 0, len(blocks))
	for _, block := range blocks {
		methodContents = append(methodContents, strings.Trim(block.Content, "\r\n"))
	}
	for i := len(ranges) - 1; i >= 0; i-- {
		rangeItem := ranges[i]
		content = content[:rangeItem.start] + content[rangeItem.end:]
	}
	separator := ""
	if start < len(content) && content[start] != '\r' && content[start] != '\n' {
		separator = "\n\n"
	}
	return content[:start] + strings.Join(methodContents, "\n\n") + separator + content[start:]
}

// reorderTSClassMethods 重排指定服务类的连续方法，未知扩展方法保持在固定槽位之后。
func reorderTSClassMethods(content string, className string) string {
	blocks, start, end, ok := tsClassMethodBlocks(content, className)
	if !ok || len(blocks) < 2 {
		return content
	}
	sortCodeGenSourceMethodBlocks(blocks)
	methodContents := make([]string, 0, len(blocks))
	for _, block := range blocks {
		methodContent := strings.Trim(block.Content, "\r\n")
		// 对缺少文档的生成方法补最小注释，已有人工注释保持原样。
		if comment := codeGenTSMethodComment(block.Name); comment != "" && !strings.HasPrefix(strings.TrimSpace(methodContent), "/**") {
			methodContent = "  /** " + comment + " */\n" + methodContent
		}
		methodContents = append(methodContents, methodContent)
	}
	return content[:start] + strings.Join(methodContents, "\n\n") + content[end:]
}

// sortCodeGenSourceMethodBlocks 按固定槽位排序，未知扩展方法保持原相对顺序。
func sortCodeGenSourceMethodBlocks(blocks []CodeGenSourceMethodBlock) {
	slices.SortStableFunc(blocks, func(a CodeGenSourceMethodBlock, b CodeGenSourceMethodBlock) int {
		aPosition := codeGenSourceMethodPosition(a.Name)
		bPosition := codeGenSourceMethodPosition(b.Name)
		if aPosition != bPosition {
			return aPosition - bPosition
		}
		if aPosition < 80 && a.Name != b.Name {
			return strings.Compare(a.Name, b.Name)
		}
		return a.OriginalIndex - b.OriginalIndex
	})
}

// codeGenTSMethodComment 返回固定槽位方法缺失注释时使用的简短说明。
func codeGenTSMethodComment(methodName string) string {
	switch codeGenSourceMethodPosition(methodName) {
	case 10:
		return "查询选择项"
	case 20:
		return "查询列表"
	case 30:
		return "查询详情"
	case 40:
		return "创建"
	case 50:
		return "更新"
	case 60:
		return "删除"
	case 70:
		return "设置状态"
	default:
		return ""
	}
}

// codeGenSourceMethodPosition 按方法名返回生成文件中的固定槽位。
func codeGenSourceMethodPosition(methodName string) int {
	switch {
	case strings.HasPrefix(methodName, "Option"):
		return 10
	case strings.HasPrefix(methodName, "Page"), strings.HasPrefix(methodName, "Tree"):
		return 20
	case strings.HasPrefix(methodName, "List"):
		return 21
	case strings.HasPrefix(methodName, "Get"):
		return 30
	case strings.HasPrefix(methodName, "Create"):
		return 40
	case strings.HasPrefix(methodName, "Update"):
		return 50
	case strings.HasPrefix(methodName, "Delete"):
		return 60
	case strings.HasPrefix(methodName, "Set"):
		return 70
	default:
		return 80
	}
}

// filterProtoMethods 按 Proto 文件筛选需要生成的方法。
func filterProtoMethods(methods []*Proto, protoPath string) []*Proto {
	list := make([]*Proto, 0, len(methods))
	for _, method := range methods {
		if method.GenerateWhenMissing == 1 && method.ProtoFilePath == protoPath {
			list = append(list, method)
		}
	}
	return sortCodeGenProtoMethods(list)
}

// sortCodeGenProtoMethods 按服务、固定槽位和方法名稳定排序。
func sortCodeGenProtoMethods(methods []*Proto) []*Proto {
	list := slices.Clone(methods)
	slices.SortStableFunc(list, func(a *Proto, b *Proto) int {
		if a.TargetEntityName != b.TargetEntityName {
			return strings.Compare(a.TargetEntityName, b.TargetEntityName)
		}
		aPosition := codeGenProtoMethodPosition(a.APIKind, a.TriggerType, a.MethodName)
		bPosition := codeGenProtoMethodPosition(b.APIKind, b.TriggerType, b.MethodName)
		if aPosition != bPosition {
			return aPosition - bPosition
		}
		return strings.Compare(a.MethodName, b.MethodName)
	})
	return list
}

// codeGenProtoMethodPosition 返回生成方法对应的固定槽位。
func codeGenProtoMethodPosition(apiKind string, triggerType string, methodName string) int {
	if apiKind == APIKindOption || apiKind == APIKindTree && triggerType != TriggerPageTree {
		return 10
	}
	if apiKind == APIKindList || triggerType == TriggerPageTree {
		return 20
	}
	if apiKind == APIKindCRUD && triggerType == TriggerCRUD {
		switch {
		case strings.HasPrefix(methodName, "Get"):
			return 30
		case strings.HasPrefix(methodName, "Create"):
			return 40
		case strings.HasPrefix(methodName, "Update"):
			return 50
		case strings.HasPrefix(methodName, "Delete"):
			return 60
		}
	}
	if apiKind == APIKindStatus {
		return 70
	}
	return 80
}

// firstMethodByKind 查找指定类型和触发来源的首个方法。
func firstMethodByKind(methods []*Proto, apiKind string, triggerType string) *Proto {
	for _, method := range methods {
		if method.APIKind != apiKind {
			continue
		}
		if triggerType != "" && method.TriggerType != triggerType {
			continue
		}
		return method
	}
	return nil
}

// methodsByKinds 按接口类型筛选方法列表。
func methodsByKinds(methods []*Proto, kinds ...string) []*Proto {
	list := make([]*Proto, 0, len(methods))
	for _, method := range methods {
		for _, kind := range kinds {
			if method.APIKind == kind {
				list = append(list, method)
				break
			}
		}
	}
	return list
}

// missingCoreMethodNames 返回未启用的列表与 CRUD 核心方法名。
func missingCoreMethodNames(table *Table, methods []*Proto) map[string]struct{} {
	methodNames := protoMethodNameSet(methods)
	entity := table.EntityName
	candidates := []string{
		"Page" + entity,
		"Tree" + entity,
		"Get" + entity,
		"Create" + entity,
		"Update" + entity,
		"Delete" + entity,
	}
	missing := make(map[string]struct{}, len(candidates))
	for _, methodName := range candidates {
		if _, ok := methodNames[methodName]; !ok {
			missing[methodName] = struct{}{}
			missing[goMethodName(methodName)] = struct{}{}
		}
	}
	return missing
}

// frontendPageMethodsComplete 判断生成前端页面所需的查询与 CRUD 接口是否完整。
func frontendPageMethodsComplete(table *Table, methods []*Proto) bool {
	methodNames := protoMethodNameSet(methods)
	listMethod := "Page" + table.EntityName
	if isTreePageType(table.PageType) {
		listMethod = "Tree" + table.EntityName
	}
	for _, methodName := range []string{
		listMethod,
		"Get" + table.EntityName,
		"Create" + table.EntityName,
		"Update" + table.EntityName,
		"Delete" + table.EntityName,
	} {
		if _, ok := methodNames[methodName]; !ok {
			return false
		}
	}
	return true
}

// protoMethodNameSet 构建 Proto 方法名集合。
func protoMethodNameSet(methods []*Proto) map[string]struct{} {
	names := make(map[string]struct{}, len(methods))
	for _, method := range methods {
		names[method.MethodName] = struct{}{}
	}
	return names
}

// currentEntityOptionMethod 查找当前实体需要生成的选择接口方法。
func currentEntityOptionMethod(table *Table, methods []*Proto) *Proto {
	for _, method := range methods {
		if method.TargetEntityName == table.EntityName && method.APIKind == APIKindOption {
			return method
		}
		if method.TargetEntityName == table.EntityName && method.APIKind == APIKindTree && (method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree) {
			return method
		}
	}
	return nil
}

// countQueryColumns 统计查询字段数量。
func countQueryColumns(columns []*CodeGenColumn) int {
	count := 0
	for _, column := range columns {
		if generatedQueryIncludesColumn(column) {
			if effectiveQueryOperator(column) == "between" {
				count += 2
				continue
			}
			count++
		}
	}
	return count
}

// effectiveQueryOperator 返回字段实际可用的查询方式，非时间字段不接受区间查询。
func effectiveQueryOperator(column *CodeGenColumn) string {
	operator := NormalizeQueryOperator(column.QueryOperator)
	if operator == "between" && !isDateTimeDBType(DefaultString(column.ColumnType, column.DbType)) {
		return "eq"
	}
	return operator
}

// renderOptionDisabledExpression 根据字段开关选项渲染公共 Option 的禁用条件。
func renderOptionDisabledExpression(columns []*CodeGenColumn, method *Proto) string {
	column, option := findOptionStatusConfig(columns)
	if column != nil {
		return fmt.Sprintf(", Disabled: fmt.Sprint(item.%s) != %q", modelFieldName(column.Name), option.ActiveValue)
	}
	if method == nil || method.OptionStatusColumn == "" || method.OptionStatusEnabledValue == "" {
		return ""
	}
	return fmt.Sprintf(", Disabled: fmt.Sprint(item.%s) != %q", modelFieldName(method.OptionStatusColumn), method.OptionStatusEnabledValue)
}

// OptionStatusMetadata 返回选项接口可使用的状态字段和启用值。
func OptionStatusMetadata(columns []*CodeGenColumn) (string, string, bool) {
	column, option := findOptionStatusConfig(columns)
	if column == nil {
		return "", "", false
	}
	return column.Name, option.ActiveValue, true
}

// findOptionStatusConfig 查找可用于控制 Option 禁用状态的字段开关配置。
func findOptionStatusConfig(columns []*CodeGenColumn) (*CodeGenColumn, CodeGenColumnOptionConfig) {
	for _, column := range columns {
		for _, option := range []CodeGenColumnOptionConfig{column.ListOption, column.FormOption} {
			if option.Kind == "switch" && option.SourceType == OptionSourceDict && option.ActiveValue != "" {
				return column, option
			}
		}
	}
	return nil, CodeGenColumnOptionConfig{}
}

// modelFieldName 将数据库字段名转为生成模型字段名。
func modelFieldName(columnName string) string {
	if columnName == "id" {
		return "ID"
	}
	if strings.HasSuffix(columnName, "_id") {
		return stringcase.ToGoPascalCase(strings.TrimSuffix(columnName, "_id")) + "ID"
	}
	return stringcase.ToGoPascalCase(columnName)
}
