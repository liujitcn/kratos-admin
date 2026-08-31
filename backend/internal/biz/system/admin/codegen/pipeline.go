package codegen

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/stringcase"
)

// LeftTreeConfigFromTable 从生成对象模型读取左树配置。
func LeftTreeConfigFromTable(table *Table) CodeGenLeftTreeConfig {
	if table == nil {
		return CodeGenLeftTreeConfig{}
	}
	var config CodeGenLeftTreeConfig
	if table.LeftTreeConfig != "" {
		_ = json.Unmarshal([]byte(table.LeftTreeConfig), &config)
	}
	if table.PageType == PageTypeLeftTree {
		config.Enabled = true
	}
	return config
}

// EntityOptionColumns 推导当前实体 Option 接口使用的父级、显示和值字段。
func EntityOptionColumns(table *Table, columns []*CodeGenColumn) (string, string, string) {
	parentColumn := DefaultString(table.ParentColumn, "parent_id")
	labelColumn := table.TreeLabelColumn
	valueColumn := "id"
	for _, column := range columns {
		if column.IsPrimary == 1 {
			valueColumn = column.Name
			break
		}
	}
	if labelColumn != "" {
		return parentColumn, labelColumn, valueColumn
	}
	for _, candidate := range []string{"name", "title", "label", "code"} {
		if column := FindColumnByName(columns, candidate); column != nil {
			return parentColumn, column.Name, valueColumn
		}
	}
	for _, column := range columns {
		if IsStringColumn(column) && column.Name != "created_by" && column.Name != "updated_by" && column.Name != "created_at" && column.Name != "updated_at" && column.Name != "deleted_at" {
			return parentColumn, column.Name, valueColumn
		}
	}
	return parentColumn, valueColumn, valueColumn
}

// FindColumnByName 按数据库字段名查找生成字段配置。
func FindColumnByName(columns []*CodeGenColumn, columnName string) *CodeGenColumn {
	for _, column := range columns {
		if column.Name == columnName {
			return column
		}
	}
	return nil
}

// IsStringColumn 判断字段是否可安全赋给 string。
func IsStringColumn(column *CodeGenColumn) bool {
	goType := DefaultString(column.GoType, InferGoType(column.DbType))
	return goType == "string"
}

func (c *renderer) buildExpectedProtoChecks(table *Table, columns []*CodeGenColumn) []*ProtoCheck {
	protoPath := c.defaultProtoPath(table)
	entity := table.EntityName
	leftTreeConfig := LeftTreeConfigFromTable(table)
	parentColumn, labelColumn, valueColumn := EntityOptionColumns(table, columns)
	checks := make([]*ProtoCheck, 0, 8)
	if table.PageType == PageTypeTree || table.PageType == PageTypeTreeLazy {
		checks = append(checks,
			c.newProtoCheck(table.ID, "", TriggerPageTree, APIKindTree, entity, "Tree"+entity, protoPath, parentColumn, labelColumn, valueColumn, true),
			c.newProtoCheck(table.ID, "", TriggerEntityOption, APIKindTree, entity, "Option"+entity, protoPath, parentColumn, labelColumn, valueColumn, true),
		)
	} else {
		checks = append(checks,
			c.newProtoCheck(table.ID, "", TriggerCRUD, APIKindList, entity, "Page"+entity, protoPath, "", "", "", true),
			c.newProtoCheck(table.ID, "", TriggerEntityOption, APIKindOption, entity, "Option"+entity, protoPath, "", labelColumn, valueColumn, true),
		)
	}
	checks = append(checks,
		c.newProtoCheck(table.ID, "", TriggerCRUD, APIKindCRUD, entity, "Get"+entity, protoPath, "", "", "", true),
		c.newProtoCheck(table.ID, "", TriggerCRUD, APIKindCRUD, entity, "Create"+entity, protoPath, "", "", "", true),
		c.newProtoCheck(table.ID, "", TriggerCRUD, APIKindCRUD, entity, "Update"+entity, protoPath, "", "", "", true),
		c.newProtoCheck(table.ID, "", TriggerCRUD, APIKindCRUD, entity, "Delete"+entity, protoPath, "", "", "", true),
	)
	if (table.PageType == PageTypeLeftTree || leftTreeConfig.Enabled) && leftTreeConfig.SourceType == OptionSourceTable {
		target := DefaultString(stringcase.ToPascalCase(leftTreeConfig.SourceValue), entity)
		targetProtoPath := c.defaultTargetProtoPath(table, target)
		check := c.newProtoCheck(table.ID, "", TriggerLeftTree, APIKindTree, target, "Option"+target, targetProtoPath, leftTreeConfig.ParentColumn, leftTreeConfig.LabelColumn, leftTreeConfig.ValueColumn, false)
		check.Lazy = leftTreeConfig.Lazy
		checks = append(checks, check)
	}

	for _, column := range columns {
		if column.IsStatusField == 1 && column.StatusGenerateAPI == 1 {
			checks = append(checks,
				c.newProtoCheck(
					table.ID,
					column.Name,
					TriggerFieldStatus,
					APIKindStatus,
					entity,
					"Set"+table.EntityName+stringcase.ToPascalCase(column.Name),
					protoPath,
					"",
					"",
					column.Name,
					true,
				),
			)
		}
		for _, option := range enabledCodeGenColumnOptions(column) {
			if option.SourceType != OptionSourceTable {
				continue
			}
			target := DefaultString(stringcase.ToPascalCase(option.SourceValue), entity)
			targetProtoPath := c.defaultTargetProtoPath(table, target)
			if option.Kind == APIKindTree {
				check := c.newProtoCheck(table.ID, column.Name, TriggerFieldOption, APIKindTree, target, "Option"+target, targetProtoPath, option.ParentField, option.LabelField, option.ValueField, false)
				check.Lazy = option.Lazy
				checks = append(checks, check)
				continue
			}
			checks = append(checks,
				c.newProtoCheck(table.ID, column.Name, TriggerFieldOption, APIKindOption, target, "Option"+target, targetProtoPath, "", option.LabelField, option.ValueField, false),
			)
		}
	}
	return dedupeProtoChecks(checks)
}

// newProtoCheck 创建 Proto 检查项。
func (c *renderer) newProtoCheck(tableID int64, columnName string, triggerType, apiKind, targetEntity, methodName, protoPath, parentColumn, labelColumn, valueColumn string, generate bool) *ProtoCheck {
	return &ProtoCheck{
		TableID:             tableID,
		Name:                columnName,
		TriggerType:         triggerType,
		APIKind:             apiKind,
		TargetEntityName:    targetEntity,
		MethodName:          methodName,
		ProtoFilePath:       protoPath,
		GenerateWhenMissing: generate,
		ParentColumn:        parentColumn,
		LabelColumn:         labelColumn,
		ValueColumn:         valueColumn,
	}
}

// resolveCodeGenOutputPaths 合并本次请求路径和默认路径，并校验启用的生成目标。
func (c *renderer) resolveCodeGenOutputPaths(table *Table, requested *adminv1.CodeGenOutputPaths) (*adminv1.CodeGenOutputPaths, error) {
	snakeEntity := stringcase.ToSnakeCase(table.EntityName)
	target, ok := ProtoTargetForBusinessModule(table.BusinessModule)
	if !ok {
		return nil, errorsx.InvalidArgument("请选择有效的Proto目录")
	}
	// 默认路径统一由实体名和资源路径推导，保证首次进入预览时即可直接生成。
	paths := &adminv1.CodeGenOutputPaths{
		ProtoFilePath:          c.defaultProtoPath(table),
		BackendBizFilePath:     target.BackendBizFilePath(snakeEntity),
		BackendServiceFilePath: target.BackendServiceFilePath(snakeEntity),
		FrontendApiFilePath:    target.FrontendAPIFilePath(snakeEntity),
		FrontendPageFilePath:   filepath.ToSlash(filepath.Join(target.FrontendPageDirectory, c.frontendResourcePath(table), "index.vue")),
	}
	// 请求只覆盖显式填写的字段，空值继续使用默认路径。
	if requested != nil {
		if requested.GetProtoFilePath() != "" {
			paths.ProtoFilePath = requested.GetProtoFilePath()
		}
		if requested.GetBackendBizFilePath() != "" {
			paths.BackendBizFilePath = requested.GetBackendBizFilePath()
		}
		if requested.GetBackendServiceFilePath() != "" {
			paths.BackendServiceFilePath = requested.GetBackendServiceFilePath()
		}
		if requested.GetFrontendApiFilePath() != "" {
			paths.FrontendApiFilePath = requested.GetFrontendApiFilePath()
		}
		if requested.GetFrontendPageFilePath() != "" {
			paths.FrontendPageFilePath = requested.GetFrontendPageFilePath()
		}
	}

	targets := []struct {
		label   string
		path    *string
		enabled bool
	}{
		{label: "Proto文件路径", path: &paths.ProtoFilePath, enabled: table.GenBackend == 1},
		{label: "后端Biz文件路径", path: &paths.BackendBizFilePath, enabled: table.GenBackend == 1},
		{label: "后端Service文件路径", path: &paths.BackendServiceFilePath, enabled: table.GenBackend == 1},
		{label: "前端API文件路径", path: &paths.FrontendApiFilePath, enabled: table.GenFrontend == 1},
		{label: "前端页面文件路径", path: &paths.FrontendPageFilePath, enabled: table.GenFrontend == 1},
	}
	seen := make(map[string]string, len(targets))
	// 只校验本轮启用的目标；未启用模块的路径允许暂时为空或保留旧配置。
	for _, target := range targets {
		*target.path = filepath.ToSlash(filepath.Clean(*target.path))
		if !target.enabled {
			continue
		}
		var pathErr error
		_, pathErr = SafeRepoFilePath(*target.path)
		if pathErr != nil {
			return nil, errorsx.InvalidArgument(target.label + "无效").WithCause(pathErr)
		}
		if previousLabel, ok := seen[*target.path]; ok {
			return nil, errorsx.InvalidArgument(fmt.Sprintf("%s不能与%s使用相同路径", target.label, previousLabel))
		}
		seen[*target.path] = target.label
	}
	// 模块边界校验用于阻止 Proto、Go 和 Vue 文件落入错误目录。
	layoutErr := validateCodeGenOutputPathLayout(target, paths)
	if layoutErr != nil {
		return nil, layoutErr
	}
	return paths, nil
}

// applyCodeGenOutputPaths 创建仅供本次预览或生成使用的配置副本。
func (c *renderer) applyCodeGenOutputPaths(table *Table, methods []*Proto, paths *adminv1.CodeGenOutputPaths) (*Table, []*Proto) {
	generationTable := *table
	generationTable.ProtoFilePath = paths.GetProtoFilePath()
	generationMethods := make([]*Proto, 0, len(methods))
	for _, method := range methods {
		generationMethod := *method
		// 当前实体的方法跟随本次临时 Proto 路径，外部选项实体仍使用其自身路径。
		if DefaultString(generationMethod.TargetEntityName, table.EntityName) == table.EntityName {
			generationMethod.ProtoFilePath = paths.GetProtoFilePath()
		}
		generationMethods = append(generationMethods, &generationMethod)
	}
	return &generationTable, generationMethods
}

// buildPreviewFiles 按后端、前端顺序构建本轮全部预览文件。
func (c *renderer) buildPreviewFiles(table *Table, columns []*CodeGenColumn, methods []*Proto, paths *adminv1.CodeGenOutputPaths, localeState LocaleState) []*adminv1.CodeGenPreviewFile {
	generatedMethods := c.generatedProtoMethods(table, columns, methods)
	frontendMethods := c.frontendProtoMethods(table, columns, methods)
	files := make([]*adminv1.CodeGenPreviewFile, 0, 9)
	if table.GenBackend == 1 {
		// 主 Proto 先生成，随后按路径去重补齐外部选项目标的 Proto 文件。
		files = append(files, c.newTargetProtoPreviewFile(table, columns, generatedMethods, c.defaultProtoPath(table)))
		seenProtoPaths := map[string]struct{}{c.defaultProtoPath(table): {}}
		for _, method := range generatedMethods {
			if method.GenerateWhenMissing != 1 || method.ProtoFilePath == "" {
				continue
			}
			if _, ok := seenProtoPaths[method.ProtoFilePath]; ok {
				continue
			}
			seenProtoPaths[method.ProtoFilePath] = struct{}{}
			files = append(files, c.newTargetProtoPreviewFile(table, columns, generatedMethods, method.ProtoFilePath))
		}
		// 主实体固定生成方法按最新配置整体替换，已有扩展方法原样后移保留。
		files = append(files,
			c.newPatchedPreviewFile(paths.GetBackendBizFilePath(), c.renderBackendBizFile(table, columns, generatedMethods), func(content string) string {
				return c.appendMainBizMethods(content, table, columns, generatedMethods)
			}),
			c.newPatchedPreviewFile(paths.GetBackendServiceFilePath(), c.renderBackendServiceFile(table, columns, generatedMethods), func(content string) string {
				return c.appendMainServiceMethods(content, table, columns, generatedMethods)
			}),
		)
		files = append(files, c.newExternalTargetBackendPreviewFiles(table, generatedMethods)...)
		files = append(files, c.newAdminRegistrationPreviewFiles(table, generatedMethods)...)
	}
	if table.GenFrontend == 1 {
		// 主实体前端 API 替换固定生成方法，页面则按稳定功能键增量合并。
		files = append(files, c.newPatchedPreviewFile(paths.GetFrontendApiFilePath(), c.renderFrontendAPIFile(table, columns, frontendMethods), func(content string) string {
			return c.appendMainFrontendAPIMethods(content, table, columns, frontendMethods)
		}))
		pagePath := paths.GetFrontendPageFilePath()
		if frontendPageMethodsComplete(table, frontendMethods) {
			files = append(files, c.newMergedFrontendPagePreviewFile(pagePath, c.renderFrontendPageFile(table, columns, frontendMethods, paths)))
		} else {
			pageFile := c.newPreviewFile(pagePath, "")
			pageFile.Action = "skip"
			pageFile.Message = Message(c.localeState, "preview.page_requirements_missing", nil)
			files = append(files, pageFile)
		}
		files = append(files, c.newExternalTargetFrontendPreviewFiles(table, frontendMethods)...)
		files = append(files, c.newFrontendLocalePreviewFiles(table, columns, localeState)...)
	}
	if ShouldSyncMenus(table, generatedMethods) {
		sqlFile := c.newGeneratedMenuSQLPreviewFile(table, RenderGeneratedMenuSQL(
			table,
			columns,
			generatedMethods,
			FrontendPageComponentPath(paths.GetFrontendPageFilePath()),
			table.TableComment,
			localeState,
		))
		files = append(files, sqlFile)
	}
	return files
}

// appendMainBizMethods 根据最新配置替换业务文件的固定生成方法并保留扩展方法。
func (c *renderer) appendMainBizMethods(content string, table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	// 候选文件提供当前配置下生成方法的完整实现，已有文件只保留非生成扩展方法。
	candidate := c.renderBackendBizFile(table, columns, methods)
	generatedReceiver := table.EntityName + "Case"
	existingReceiver := goReceiverType(content, "Case")
	if existingReceiver == "" {
		return content
	}
	// Mapper 已统一处理时间字段，清理旧模板生成的重复赋值。
	content = redundantTimePattern.ReplaceAllString(content, "")
	if !strings.Contains(content, "_time.") {
		content = strings.Replace(content, "\t_time \"github.com/liujitcn/go-utils/time\"\n", "", 1)
	}
	// 已有业务文件可能采用不同实体名，先从接收者依赖中识别真实 Repository 类型。
	_, repositoryType := goStructDependency(content, existingReceiver, "data", "Repository")
	repositoryType = strings.TrimSuffix(repositoryType, "Repository")
	if repositoryType == "" {
		return content
	}

	methodNames := goReceiverMethodNames(candidate, generatedReceiver)
	methodContent := strings.Join(extractGoMethods(candidate, methodNames), "\n\n")
	if methodContent == "" {
		return content
	}
	// 仅改写生成方法片段，避免替换已有扩展代码中的同名标识符。
	entityVar := stringcase.ToCamelCase(table.EntityName)
	methodContent = strings.ReplaceAll(methodContent, "*"+generatedReceiver, "*"+existingReceiver)
	methodContent = strings.ReplaceAll(methodContent, "models."+table.EntityName, "models."+repositoryType)
	methodContent = strings.ReplaceAll(methodContent, "c.Query(ctx)."+table.EntityName, "c.Query(ctx)."+repositoryType)
	apiAlias := ProtoTargetForTable(table).GoAlias
	methodContent = strings.ReplaceAll(methodContent, "c.formMapper.ToEntity(req)", "mapper.NewCopierMapper["+apiAlias+"."+table.EntityName+"Form, models."+repositoryType+"]().ToEntity(req)")
	methodContent = strings.ReplaceAll(methodContent, "c.formMapper.ToDTO("+entityVar+")", "mapper.NewCopierMapper["+apiAlias+"."+table.EntityName+"Form, models."+repositoryType+"]().ToDTO("+entityVar+")")
	methodContent = strings.ReplaceAll(methodContent, "c.mapper.ToDTO(item)", "mapper.NewCopierMapper["+apiAlias+"."+table.EntityName+", models."+repositoryType+"]().ToDTO(item)")
	target := ProtoTargetForTable(table)
	content = mergeGeneratedGoReceiverMethods(content, methodContent, existingReceiver, target.GoAlias+" \""+target.GoImportPath+"\"")
	if !strings.Contains(content, "_time.") {
		content = strings.Replace(content, "\t_time \"github.com/liujitcn/go-utils/time\"\n", "", 1)
	}
	return content
}

// appendMainServiceMethods 根据最新配置替换已有服务文件的固定生成方法。
func (c *renderer) appendMainServiceMethods(content string, table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	// 服务层从完整候选文件中提取固定生成方法，已有同名实现必须整体替换。
	candidate := c.renderBackendServiceFile(table, columns, methods)
	generatedReceiver := table.EntityName + "Service"
	existingReceiver := goReceiverType(content, "Service")
	if existingReceiver == "" {
		return content
	}
	methodNames := goReceiverMethodNames(candidate, generatedReceiver)
	methodContent := strings.Join(extractGoMethods(candidate, methodNames), "\n\n")
	if methodContent == "" {
		return content
	}
	// 根据已有 Service 的依赖字段改写候选接收者，兼容人工命名的 Case 字段。
	existingCaseField, _ := goStructDependency(content, existingReceiver, "biz", "Case")
	generatedCaseField := stringcase.ToCamelCase(table.EntityName) + "Case"
	if existingCaseField == "" {
		return content
	}
	methodContent = strings.ReplaceAll(methodContent, "*"+generatedReceiver, "*"+existingReceiver)
	methodContent = strings.ReplaceAll(methodContent, "s."+generatedCaseField, "s."+existingCaseField)
	target := ProtoTargetForTable(table)
	return mergeGeneratedGoReceiverMethods(content, methodContent, existingReceiver, target.GoAlias+" \""+target.GoImportPath+"\"")
}

// appendMainFrontendAPIMethods 根据最新配置替换前端服务类的固定生成方法。
func (c *renderer) appendMainFrontendAPIMethods(content string, table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	candidate := c.renderFrontendAPIFile(table, columns, methods)
	className := table.EntityName + "ServiceImpl"
	if findTSClassEndIndex(content, className) < 0 {
		return content
	}
	methodBlocks, _, _, ok := tsClassMethodBlocks(candidate, className)
	if !ok || len(methodBlocks) == 0 {
		return content
	}
	typeNames := make([]string, 0, len(methodBlocks)*2)
	methodContents := make([]string, 0, len(methodBlocks))
	// 方法和请求响应类型同步收集，保证替换实现后 import 仍然完整。
	for _, block := range methodBlocks {
		methodContents = append(methodContents, block.Content)
		typeNames = append(typeNames, block.Name+"Request")
		methodContent := block.Content
		if strings.Contains(methodContent, table.EntityName+"Form") {
			typeNames = append(typeNames, table.EntityName+"Form")
		}
		if strings.Contains(methodContent, block.Name+"Response") {
			typeNames = append(typeNames, block.Name+"Response")
		}
	}

	// 先补类型导入，因为新增 import 会改变类和方法在源码中的偏移量。
	rpcPath := frontendRPCImportPath(c.defaultProtoPath(table))
	content = ensureTSNamedTypeNames(content, rpcPath, typeNames)
	joinedMethods := strings.Join(methodContents, "\n\n")
	target := ProtoTargetForTable(table)
	if strings.Contains(joinedMethods, "Empty") {
		content = ensureTSNamedTypeNames(content, target.FrontendPackageName+"/rpc/google/protobuf/empty", []string{"Empty"})
	}
	commonTypes := make([]string, 0, 2)
	if strings.Contains(joinedMethods, "SelectOptionResponse") {
		commonTypes = append(commonTypes, "SelectOptionResponse")
	}
	if strings.Contains(joinedMethods, "TreeOptionResponse") {
		commonTypes = append(commonTypes, "TreeOptionResponse")
	}
	content = ensureTSNamedTypeNames(content, target.FrontendPackageName+"/rpc/common/v1/common", commonTypes)
	return mergeGeneratedTSClassMethods(content, candidate, className)
}

// enabledCodeGenColumnOptions 返回当前字段各作用域实际启用的选项配置。
func enabledCodeGenColumnOptions(column *CodeGenColumn) []CodeGenColumnOptionConfig {
	options := make([]CodeGenColumnOptionConfig, 0, 3)
	if column.IsQuery == 1 && column.QueryOption.Kind != "" {
		options = append(options, column.QueryOption)
	}
	if column.IsList == 1 && column.ListOption.Kind != "" {
		options = append(options, column.ListOption)
	}
	if column.IsForm == 1 && column.FormOption.Kind != "" {
		options = append(options, column.FormOption)
	}
	return options
}
