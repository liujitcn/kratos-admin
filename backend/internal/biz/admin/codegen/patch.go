package codegen

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"

	"github.com/liujitcn/go-utils/stringcase"
)

const unmergeableFrontendPageMessage = "已有前端页面无法安全解析，已跳过增量合并"

// --- 已有源码的增量分析与补丁 ---

// newExternalTargetBackendPreviewFiles 创建外部选项目标的后端补齐文件。
func (c *renderer) newExternalTargetBackendPreviewFiles(table *Table, methods []*Proto) []*systemadminv1.CodeGenPreviewFile {
	targets := c.externalOptionTargets(table, methods)
	files := make([]*systemadminv1.CodeGenPreviewFile, 0, len(targets)*2)
	for _, target := range targets {
		protoTarget := ProtoTargetForTable(target.Table)
		// 外部实体只补选项查询所需的 Biz 和 Service，不生成该实体的完整 CRUD。
		bizPath := protoTarget.BackendBizFilePath(target.Table.EntityName)
		servicePath := protoTarget.BackendServiceFilePath(target.Table.EntityName)
		files = append(files,
			c.newPatchedPreviewFile(bizPath, c.renderExternalTargetBizFile(target.Table, target.Methods), func(content string) string {
				return c.appendExternalTargetBizMethods(content, target.Table, target.Methods)
			}),
			c.newPatchedPreviewFile(servicePath, c.renderExternalTargetServiceFile(target.Table, target.Methods), func(content string) string {
				return c.appendExternalTargetServiceMethods(content, target.Table, target.Methods)
			}),
		)
	}
	return files
}

// newExternalTargetFrontendPreviewFiles 创建外部选项目标的前端 API 补齐文件。
func (c *renderer) newExternalTargetFrontendPreviewFiles(table *Table, methods []*Proto) []*systemadminv1.CodeGenPreviewFile {
	targets := c.externalOptionTargets(table, methods)
	files := make([]*systemadminv1.CodeGenPreviewFile, 0, len(targets))
	for _, target := range targets {
		protoTarget := ProtoTargetForTable(target.Table)
		// 前端只需要补齐选项数据源对应的请求方法。
		path := protoTarget.FrontendAPIFilePath(target.Table.EntityName)
		files = append(files, c.newPatchedPreviewFile(path, c.renderExternalTargetFrontendAPIFile(target.Table, target.Methods), func(content string) string {
			return c.appendExternalTargetFrontendAPIMethods(content, target.Table, target.Methods)
		}))
	}
	return files
}

// newPatchedPreviewFile 创建支持替换生成方法并保留扩展方法的预览文件。
func (c *renderer) newPatchedPreviewFile(path string, createContent string, patch func(string) string) *systemadminv1.CodeGenPreviewFile {
	// 所有预览文件先经过仓库边界校验，非法路径只返回 skip 结果，不触碰磁盘。
	_, pathErr := SafeRepoFilePath(path)
	if pathErr != nil {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: createContent, Exists: false, Message: pathErr.Error()}
	}
	content, err := c.readRepoFile(path)
	if err != nil {
		// 目标不存在时按完整模板创建；已有文件才进入增量补丁逻辑。
		return c.newPreviewFile(path, createContent)
	}
	patched := patch(string(content))
	if patched == string(content) {
		// 内容相同必须标记为 skip，避免生成任务无意义地更新文件时间。
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: string(content), Exists: true, Message: "目标文件已存在，生成方法已与当前配置一致"}
	}
	return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "update", Content: patched, Exists: true, Message: "目标文件已存在，将替换生成方法并保留扩展方法"}
}

// newMergedFrontendPagePreviewFile 按功能顺序增量合并前端页面并保留已有扩展。
func (c *renderer) newMergedFrontendPagePreviewFile(path string, renderedContent string) *systemadminv1.CodeGenPreviewFile {
	_, pathErr := SafeRepoFilePath(path)
	if pathErr != nil {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: renderedContent, Exists: false, Message: pathErr.Error()}
	}
	content, err := c.readRepoFile(path)
	// 目标文件首次生成时直接使用完整模板创建。
	if err != nil {
		return c.newPreviewFile(path, renderedContent)
	}
	originalContent := string(content)
	mergedContent, ok := mergeGeneratedFrontendPage(originalContent, renderedContent)
	if !ok {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: originalContent, Exists: true, Message: unmergeableFrontendPageMessage}
	}
	if originalContent == mergedContent {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: originalContent, Exists: true, Message: "生成页面已与当前配置和扩展功能一致"}
	}
	return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "update", Content: mergedContent, Exists: true, Message: "生成页面将按功能顺序更新并保留已有扩展"}
}

// newMergedFrontendLocalePreviewFile 解析并合并已有扁平 JSON 语言包。
func (c *renderer) newMergedFrontendLocalePreviewFile(path string, prefix string, messages map[string]string) *systemadminv1.CodeGenPreviewFile {
	_, pathErr := SafeRepoFilePath(path)
	if pathErr != nil {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Exists: false, Message: pathErr.Error()}
	}
	renderedContent, err := renderFrontendLocaleMessages(messages)
	if err != nil {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Exists: false, Message: err.Error()}
	}
	content, err := c.readRepoFile(path)
	if err != nil {
		return c.newPreviewFile(path, renderedContent)
	}
	mergedContent, err := mergeFrontendLocaleMessages(string(content), prefix, messages)
	if err != nil {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: string(content), Exists: true, Message: err.Error()}
	}
	if string(content) == mergedContent {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: string(content), Exists: true, Message: "生成语言包已与当前配置一致"}
	}
	return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "update", Content: mergedContent, Exists: true, Message: "将更新当前生成对象的语言键并保留其他文案"}
}

// newAdminRegistrationPreviewFiles 创建管理端业务模块依赖注入与传输层注册补丁。
func (c *renderer) newAdminRegistrationPreviewFiles(table *Table, methods []*Proto) []*systemadminv1.CodeGenPreviewFile {
	type registrationGroup struct {
		target   ProtoTarget
		entities []string
	}
	groups := []registrationGroup{{target: ProtoTargetForTable(table), entities: []string{table.EntityName}}}
	for _, target := range c.externalOptionTargets(table, methods) {
		protoTarget := ProtoTargetForTable(target.Table)
		groupIndex := -1
		for index := range groups {
			if groups[index].target.BackendModuleDirectory == protoTarget.BackendModuleDirectory {
				groupIndex = index
				break
			}
		}
		if groupIndex < 0 {
			groups = append(groups, registrationGroup{target: protoTarget})
			groupIndex = len(groups) - 1
		}
		groups[groupIndex].entities = append(groups[groupIndex].entities, target.Table.EntityName)
	}
	files := make([]*systemadminv1.CodeGenPreviewFile, 0, len(groups)*2)
	for _, group := range groups {
		for _, item := range []struct {
			path  string
			patch func(string, ProtoTarget, string) string
		}{
			{path: filepath.ToSlash(filepath.Join(group.target.BackendModuleDirectory, "init.go")), patch: appendAdminProviderRegistration},
			{path: group.target.ModuleRegisterPath, patch: appendProtoServiceRegistration},
		} {
			patch := item.patch
			entities := slices.Clone(group.entities)
			files = append(files, c.newExistingPatchedPreviewFile(item.path, func(content string) string {
				for _, entity := range entities {
					content = patch(content, group.target, entity)
				}
				return content
			}))
		}
	}
	return files
}

// externalOptionTargets 按外部目标实体分组选项接口。
func (c *renderer) externalOptionTargets(table *Table, methods []*Proto) []CodeGenExternalTarget {
	grouped := make(map[string][]*Proto)
	for _, method := range methods {
		if method.TargetEntityName == "" || method.TargetEntityName == table.EntityName {
			continue
		}
		if method.GenerateWhenMissing != 1 {
			continue
		}
		if !IsOptionProtoMethod(method) {
			continue
		}
		grouped[method.TargetEntityName] = append(grouped[method.TargetEntityName], method)
	}
	targets := make([]CodeGenExternalTarget, 0, len(grouped))
	for target, targetMethods := range grouped {
		targets = append(targets, CodeGenExternalTarget{
			Table:   c.externalTargetTable(table, target, targetMethods),
			Methods: targetMethods,
		})
	}
	slices.SortFunc(targets, func(a CodeGenExternalTarget, b CodeGenExternalTarget) int {
		return strings.Compare(a.Table.EntityName, b.Table.EntityName)
	})
	return targets
}

// externalTargetTable 构造外部目标实体的最小生成上下文。
func (c *renderer) externalTargetTable(table *Table, target string, methods []*Proto) *Table {
	businessName := target
	for _, method := range methods {
		if method.TargetBusinessName != "" {
			businessName = method.TargetBusinessName
			break
		}
	}
	protoPath := c.defaultTargetProtoPath(table, target)
	protoTarget, ok := ProtoTargetForProtoPath(protoPath)
	if !ok {
		protoTarget = ProtoTargetForTable(table)
	}
	return &Table{
		TableName_:     stringcase.ToSnakeCase(target),
		TableComment:   businessName,
		BusinessModule: strings.TrimSuffix(protoTarget.Directory, "/admin/v1"),
		BusinessName:   businessName,
		EntityName:     target,
		ModulePath:     resourcePathByEntity(target),
		APIPath:        protoTarget.Directory,
		ProtoFilePath:  protoPath,
		PageType:       "normal",
		GenBackend:     1,
		GenFrontend:    1,
		GenSql:         0,
		Status:         StatusDraft,
		CreatedAt:      table.CreatedAt,
		UpdatedAt:      table.UpdatedAt,
	}
}

// newExistingPatchedPreviewFile 创建只允许追加已有文件的预览补丁。
func (c *renderer) newExistingPatchedPreviewFile(path string, patch func(string) string) *systemadminv1.CodeGenPreviewFile {
	_, err := SafeRepoFilePath(path)
	if err != nil {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Message: err.Error()}
	}
	var content []byte
	content, err = c.readRepoFile(path)
	if err != nil {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Message: "目标文件不存在，无法追加服务注册"}
	}
	patched := patch(string(content))
	if patched == string(content) {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "skip", Content: string(content), Exists: true, Message: "目标文件已存在，未发现需要追加的服务注册"}
	}
	return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "update", Content: patched, Exists: true, Message: "目标文件已存在，将追加缺失服务注册"}
}

// generatedProtoMethods 合并基础能力与用户勾选的缺失接口。
func (c *renderer) generatedProtoMethods(table *Table, columns []*CodeGenColumn, methods []*Proto) []*Proto {
	checks := c.buildExpectedProtoChecks(table, columns)
	list := make([]*Proto, 0, len(checks))
	for _, check := range checks {
		saved := findSavedProtoMethod(methods, check)
		if saved != nil {
			applySavedProtoMethod(check, saved)
		}
		exists, _ := c.protoMethodExists(check.ProtoFilePath, check.TargetEntityName, check.MethodName)
		// 外部目标接口已存在时仅供当前页面引用，不再触发外部补齐文件生成。
		if exists && DefaultString(check.TargetEntityName, table.EntityName) != table.EntityName {
			check.GenerateWhenMissing = false
		}
		if exists || table.GenBackend == 1 && check.GenerateWhenMissing {
			list = append(list, c.protoCheckToModel(check))
		}
	}
	return sortCodeGenProtoMethods(list)
}

// frontendProtoMethods 合并前端页面可使用的已存在或将生成接口。
func (c *renderer) frontendProtoMethods(table *Table, columns []*CodeGenColumn, methods []*Proto) []*Proto {
	checks := c.buildExpectedProtoChecks(table, columns)
	list := make([]*Proto, 0, len(checks))
	for _, check := range checks {
		saved := findSavedProtoMethod(methods, check)
		if saved != nil {
			applySavedProtoMethod(check, saved)
		}
		exists, _ := c.protoMethodExists(check.ProtoFilePath, check.TargetEntityName, check.MethodName)
		// 外部目标接口已存在时仅供当前页面引用，不再触发外部补齐文件生成。
		if exists && DefaultString(check.TargetEntityName, table.EntityName) != table.EntityName {
			check.GenerateWhenMissing = false
		}
		if exists || table.GenBackend == 1 && check.GenerateWhenMissing {
			list = append(list, c.protoCheckToModel(check))
		}
	}
	return sortCodeGenProtoMethods(list)
}

// protoCheckToModel 将检查项转换为生成方法配置。
func (c *renderer) protoCheckToModel(check *ProtoCheck) *Proto {
	return &Proto{
		TableID:                  check.TableID,
		Name:                     check.Name,
		TriggerType:              check.TriggerType,
		APIKind:                  check.APIKind,
		TargetEntityName:         check.TargetEntityName,
		TargetBusinessName:       check.TargetBusinessName,
		MethodName:               check.MethodName,
		ProtoFilePath:            check.ProtoFilePath,
		ParentColumn:             check.ParentColumn,
		LabelColumn:              check.LabelColumn,
		ValueColumn:              check.ValueColumn,
		OptionStatusColumn:       check.OptionStatusColumn,
		OptionStatusEnabledValue: check.OptionStatusEnabledValue,
		Lazy:                     check.Lazy,
		GenerateWhenMissing:      BoolToInt32(check.GenerateWhenMissing),
	}
}

// newTargetProtoPreviewFile 创建指定 Proto 文件预览内容。
func (c *renderer) newTargetProtoPreviewFile(table *Table, columns []*CodeGenColumn, methods []*Proto, path string) *systemadminv1.CodeGenPreviewFile {
	_, err := SafeRepoFilePath(path)
	if err != nil {
		return &systemadminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: "",
			Exists:  false,
			Message: err.Error(),
		}
	}
	var content []byte
	content, err = c.readRepoFile(path)
	if err != nil {
		if path != c.defaultProtoPath(table) {
			return c.newPreviewFile(path, c.renderTargetProtoFile(table, columns, methods, path))
		}
		return c.newPreviewFile(path, c.renderProtoFile(table, columns, methods))
	}

	originalContent := string(content)
	renderedContent := c.renderTargetProtoFile(table, columns, methods, path)
	if path == c.defaultProtoPath(table) {
		renderedContent = c.renderProtoFile(table, columns, methods)
	}
	// 已有 Proto 只整体替换固定生成 RPC 与消息，自定义定义始终原样保留在后面。
	mergedContent := mergeGeneratedProtoFile(originalContent, renderedContent)
	if mergedContent != originalContent {
		return &systemadminv1.CodeGenPreviewFile{Path: path, Action: "update", Content: mergedContent, Exists: true, Message: "Proto文件将替换生成接口并保留扩展定义"}
	}
	patch := c.renderProtoPatch(table, columns, methods, path)
	normalizedContent := normalizeProtoMessageOrder(normalizeProtoRPCOrder(dedupeProtoMessageBlocks(originalContent), methods, path))
	if len(patch.ServiceNames) == 0 && len(patch.Messages) == 0 {
		if normalizedContent != originalContent {
			return &systemadminv1.CodeGenPreviewFile{
				Path:    path,
				Action:  "update",
				Content: normalizedContent,
				Exists:  true,
				Message: "Proto文件结构将按固定方法槽位整理",
			}
		}
		return &systemadminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: originalContent,
			Exists:  true,
			Message: "Proto文件已存在，未选择缺失接口",
		}
	}
	patched := normalizeProtoMessageOrder(normalizeProtoRPCOrder(dedupeProtoMessageBlocks(c.appendProtoPatch(normalizedContent, patch)), methods, path))
	if patched == originalContent {
		return &systemadminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: originalContent,
			Exists:  true,
			Message: "Proto文件已存在，未找到目标service，已跳过追加",
		}
	}
	return &systemadminv1.CodeGenPreviewFile{
		Path:    path,
		Action:  "update",
		Content: patched,
		Exists:  true,
		Message: "Proto文件已存在，将追加缺失接口",
	}
}

// newPreviewFile 创建预览文件并标记处理动作。
func (c *renderer) newPreviewFile(path string, content string) *systemadminv1.CodeGenPreviewFile {
	_, pathErr := SafeRepoFilePath(path)
	if pathErr != nil {
		return &systemadminv1.CodeGenPreviewFile{
			Path:    path,
			Action:  "skip",
			Content: content,
			Exists:  false,
			Message: pathErr.Error(),
		}
	}
	exists, err := c.repoFileExists(path)
	if err != nil {
		exists = false
	}
	action := "create"
	message := "将新增"
	if exists {
		action = "skip"
		message = "已存在，不覆盖"
	}
	return &systemadminv1.CodeGenPreviewFile{
		Path:    path,
		Action:  action,
		Content: content,
		Exists:  exists,
		Message: message,
	}
}

// goReceiverType 返回 Go 文件中指定后缀的首个方法接收者或结构体类型。
func goReceiverType(content string, suffix string) string {
	file, _, err := parseGoSource(content)
	if err != nil {
		return ""
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || len(function.Recv.List) == 0 {
			continue
		}
		receiver := goReceiverName(function.Recv.List[0].Type)
		if strings.HasSuffix(receiver, suffix) {
			return receiver
		}
	}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok || !strings.HasSuffix(typeSpec.Name.Name, suffix) {
				continue
			}
			if _, ok = typeSpec.Type.(*ast.StructType); ok {
				return typeSpec.Name.Name
			}
		}
	}
	return ""
}

// goReceiverMethodNames 返回指定接收者类型的全部方法名。
func goReceiverMethodNames(content string, receiverName string) map[string]struct{} {
	methods := make(map[string]struct{})
	file, _, err := parseGoSource(content)
	if err != nil {
		return methods
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || len(function.Recv.List) == 0 {
			continue
		}
		if goReceiverName(function.Recv.List[0].Type) == receiverName {
			methods[function.Name.Name] = struct{}{}
		}
	}
	return methods
}

// goStructDependency 查找结构体中指定包和类型后缀的依赖字段。
func goStructDependency(content string, structName string, packageName string, typeSuffix string) (string, string) {
	file, _, err := parseGoSource(content)
	if err != nil {
		return "", ""
	}
	for _, declaration := range file.Decls {
		general, isGeneral := declaration.(*ast.GenDecl)
		if !isGeneral {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, isTypeSpec := specification.(*ast.TypeSpec)
			if !isTypeSpec || typeSpec.Name.Name != structName {
				continue
			}
			structType, isStructType := typeSpec.Type.(*ast.StructType)
			if !isStructType {
				return "", ""
			}
			for _, field := range structType.Fields.List {
				selector := goSelectorType(field.Type)
				if selector == nil || selector.Sel == nil || !strings.HasSuffix(selector.Sel.Name, typeSuffix) {
					continue
				}
				identifier, isIdentifier := selector.X.(*ast.Ident)
				if !isIdentifier || identifier.Name != packageName {
					continue
				}
				fieldName := selector.Sel.Name
				if len(field.Names) > 0 {
					fieldName = field.Names[0].Name
				}
				return fieldName, selector.Sel.Name
			}
		}
	}
	return "", ""
}

// extractGoMethods 从生成候选文件中提取指定方法源码。
func extractGoMethods(content string, methodNames map[string]struct{}) []string {
	methods := make([]string, 0, len(methodNames))
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return methods
	}
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil {
			continue
		}
		if _, ok = methodNames[function.Name.Name]; !ok {
			continue
		}
		start := fileSet.Position(function.Pos()).Offset
		if function.Doc != nil {
			start = fileSet.Position(function.Doc.Pos()).Offset
		}
		end := fileSet.Position(function.End()).Offset
		if start >= 0 && end <= len(content) && start < end {
			methods = append(methods, strings.TrimSpace(content[start:end]))
		}
	}
	return methods
}

// mergeGeneratedGoReceiverMethods 按候选顺序替换生成方法，并将已有扩展方法原样保留在生成方法之后。
func mergeGeneratedGoReceiverMethods(content string, methodContent string, receiverName string, importLines ...string) string {
	generatedBlocks := generatedGoReceiverMethodBlocks(methodContent, receiverName)
	if len(generatedBlocks) == 0 {
		return content
	}
	originalContent := content
	for _, importLine := range importLines {
		content = ensureGoImport(content, importLine)
	}
	content = ensureGeneratedGoImports(content, methodContent)
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return originalContent
	}
	type sourceRange struct {
		start            int
		end              int
		declarationIndex int
	}
	existingBlocks := make([]CodeGenSourceMethodBlock, 0)
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
			return originalContent
		}
		existingBlocks = append(existingBlocks, CodeGenSourceMethodBlock{
			Name:          function.Name.Name,
			Content:       content[start:end],
			Start:         start,
			End:           end,
			OriginalIndex: len(existingBlocks),
		})
		ranges = append(ranges, sourceRange{start: start, end: end, declarationIndex: declarationIndex})
		if firstDeclarationIndex < 0 {
			firstDeclarationIndex = declarationIndex
		}
		lastDeclarationIndex = declarationIndex
	}
	generatedNames := make(map[string]struct{}, len(generatedBlocks))
	for _, block := range generatedBlocks {
		generatedNames[strings.ToLower(block.Name)] = struct{}{}
	}
	customBlocks := make([]CodeGenSourceMethodBlock, 0, len(existingBlocks))
	for _, block := range existingBlocks {
		if _, generated := generatedNames[strings.ToLower(block.Name)]; generated {
			continue
		}
		customBlocks = append(customBlocks, block)
	}
	methodContents := make([]string, 0, len(generatedBlocks)+len(customBlocks))
	for _, block := range generatedBlocks {
		methodContents = append(methodContents, strings.Trim(block.Content, "\r\n"))
	}
	for _, block := range customBlocks {
		methodContents = append(methodContents, strings.Trim(block.Content, "\r\n"))
	}
	if len(ranges) == 0 {
		patched := strings.TrimRight(content, "\r\n") + "\n\n" + strings.Join(methodContents, "\n\n") + "\n"
		if _, _, err = parseGoSource(patched); err != nil {
			return originalContent
		}
		return patched
	}
	// 只跨越普通包级辅助函数；其他声明仍作为人工组织边界，避免改变源码语义。
	for declarationIndex := firstDeclarationIndex + 1; declarationIndex < lastDeclarationIndex; declarationIndex++ {
		function, ok := file.Decls[declarationIndex].(*ast.FuncDecl)
		if !ok {
			return originalContent
		}
		if function.Recv != nil && len(function.Recv.List) > 0 && goReceiverName(function.Recv.List[0].Type) == receiverName {
			continue
		}
		if function.Recv != nil || function.Name.Name == "init" {
			return originalContent
		}
	}
	start := ranges[0].start
	for index := 1; index < len(ranges); index++ {
		previousFunction, ok := file.Decls[ranges[index].declarationIndex-1].(*ast.FuncDecl)
		if !ok || previousFunction.Recv != nil || previousFunction.Name.Name == "init" {
			continue
		}
		for ranges[index].start > 0 {
			character := content[ranges[index].start-1]
			if character != ' ' && character != '\t' && character != '\r' && character != '\n' {
				break
			}
			ranges[index].start--
		}
	}
	for index := 0; index < len(ranges)-1; index++ {
		for ranges[index].end < len(content) {
			character := content[ranges[index].end]
			if character != ' ' && character != '\t' && character != '\r' && character != '\n' {
				break
			}
			ranges[index].end++
		}
	}
	for index := len(ranges) - 1; index >= 0; index-- {
		content = content[:ranges[index].start] + content[ranges[index].end:]
	}
	separator := ""
	if start < len(content) && content[start] != '\r' && content[start] != '\n' {
		separator = "\n\n"
	}
	patched := content[:start] + strings.Join(methodContents, "\n\n") + separator + content[start:]
	if _, _, err = parseGoSource(patched); err != nil {
		return originalContent
	}
	return patched
}

// generatedGoReceiverMethodBlocks 解析待写入的生成方法片段。
func generatedGoReceiverMethodBlocks(methodContent string, receiverName string) []CodeGenSourceMethodBlock {
	prefix := "package generated\n\n"
	content := prefix + methodContent
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return nil
	}
	blocks := make([]CodeGenSourceMethodBlock, 0)
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil || len(function.Recv.List) == 0 || goReceiverName(function.Recv.List[0].Type) != receiverName {
			continue
		}
		start := fileSet.Position(function.Pos()).Offset
		if function.Doc != nil {
			start = fileSet.Position(function.Doc.Pos()).Offset
		}
		end := fileSet.Position(function.End()).Offset
		if start < len(prefix) || end > len(content) || start >= end {
			return nil
		}
		blocks = append(blocks, CodeGenSourceMethodBlock{
			Name:          function.Name.Name,
			Content:       strings.TrimSpace(content[start:end]),
			OriginalIndex: len(blocks),
		})
	}
	return blocks
}

// removeGoReceiverMethods 从 Go 源码中删除指定接收者方法。
func removeGoReceiverMethods(content string, methodNames map[string]struct{}) string {
	file, fileSet, err := parseGoSource(content)
	if err != nil || len(methodNames) == 0 {
		return content
	}
	type sourceRange struct {
		start int
		end   int
	}
	var ranges []sourceRange
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if !ok || function.Recv == nil {
			continue
		}
		if _, ok = methodNames[function.Name.Name]; !ok {
			continue
		}
		start := fileSet.Position(function.Pos()).Offset
		if function.Doc != nil {
			start = fileSet.Position(function.Doc.Pos()).Offset
		}
		ranges = append(ranges, sourceRange{start: start, end: fileSet.Position(function.End()).Offset})
	}
	for i := len(ranges) - 1; i >= 0; i-- {
		item := ranges[i]
		if item.start >= 0 && item.end <= len(content) && item.start < item.end {
			content = content[:item.start] + content[item.end:]
		}
	}
	return content
}

// appendAdminProviderRegistration 向管理端 ProviderSet 追加业务与服务构造函数。
func appendAdminProviderRegistration(content string, _ ProtoTarget, entity string) string {
	additions := make([]string, 0, 2)
	if !goIdentifierExists(content, "New"+entity+"Case") {
		additions = append(additions, "biz.New"+entity+"Case")
	}
	if !goIdentifierExists(content, "New"+entity+"Service") {
		additions = append(additions, "New"+entity+"Service")
	}
	for _, addition := range additions {
		content = insertGoProviderSetItem(content, addition)
	}
	return content
}

// appendProtoServiceRegistration 向目标业务模块追加服务字段和传输层注册。
func appendProtoServiceRegistration(content string, target ProtoTarget, entity string) string {
	grpcReceiver := goFuncReceiverName(content, "RegisterGRPC")
	httpReceiver := goFuncReceiverName(content, "RegisterHTTP")
	mcpReceiver := goFuncReceiverName(content, "RegisterMCP")
	if grpcReceiver == "" || httpReceiver == "" || mcpReceiver == "" {
		return content
	}
	fieldName := goStructSelectorFieldName(content, "Services", target.ServiceImportAlias, entity+"Service")
	if fieldName == "" {
		fieldName = entity
		content = insertGoStructSelectorField(content, "Services", target.ServiceImportAlias, entity+"Service", "\t"+fieldName+" *"+target.ServiceImportAlias+"."+entity+"Service")
	}

	grpcRegisterName := "Register" + entity + "ServiceServer"
	if !goCallNameExists(content, target.GoAlias, grpcRegisterName) {
		content = insertGoPackageCall(content, "RegisterGRPC", target.GoAlias, grpcRegisterName, "\t"+target.GoAlias+"."+grpcRegisterName+"(srv, "+grpcReceiver+"."+fieldName+")")
	}
	httpRegisterName := "Register" + entity + "ServiceHTTPServer"
	if !goCallNameExists(content, target.GoAlias, httpRegisterName) {
		content = insertGoPackageCall(content, "RegisterHTTP", target.GoAlias, httpRegisterName, "\t"+target.GoAlias+"."+httpRegisterName+"(srv, "+httpReceiver+"."+fieldName+")")
	}
	mcpRegisterName := "Register" + entity + "ServiceMCPTools"
	if !goCallNameExists(content, target.GoAlias, mcpRegisterName) {
		content = insertGoPackageCall(content, "RegisterMCP", target.GoAlias, mcpRegisterName, "\t"+target.GoAlias+"."+mcpRegisterName+"(mcpSrv, "+mcpReceiver+"."+fieldName+")")
	}
	return content
}

// insertGoProviderSetItem 在相同 ProviderSet 分组内按名称插入依赖提供者。
func insertGoProviderSetItem(content string, providerName string) string {
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return content
	}
	var providerSet *ast.CallExpr
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range general.Specs {
			valueSpec, ok := specification.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || valueSpec.Names[0].Name != "ProviderSet" || len(valueSpec.Values) != 1 {
				continue
			}
			call, ok := valueSpec.Values[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "NewSet" {
				continue
			}
			identifier, ok := selector.X.(*ast.Ident)
			if ok && identifier.Name == "wire" {
				providerSet = call
			}
		}
	}
	if providerSet == nil {
		return content
	}
	isBizProvider := strings.HasPrefix(providerName, "biz.")
	providerKey := goProviderEntitySortKey(providerName)
	offset := fileSet.Position(providerSet.Rparen).Offset
	for _, argument := range providerSet.Args {
		name := goQualifiedExpressionName(argument)
		if name == "" || strings.HasPrefix(name, "biz.") != isBizProvider {
			continue
		}
		if strings.Compare(goProviderEntitySortKey(name), providerKey) > 0 {
			offset = fileSet.Position(argument.Pos()).Offset
			break
		}
		offset = fileSet.Position(argument.End()).Offset
	}
	return validGoPatch(content, insertGoLines(content, offset, "\t"+providerName+","))
}

// insertGoStructSelectorField 在同包类型字段中按类型名插入结构体字段。
func insertGoStructSelectorField(content string, structName string, packageName string, typeName string, line string) string {
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return content
	}
	structType := findGoStructType(file, structName)
	if structType == nil {
		return content
	}
	offset := fileSet.Position(structType.Fields.Closing).Offset
	for _, field := range structType.Fields.List {
		selector := goSelectorType(field.Type)
		if selector == nil || selector.Sel == nil {
			continue
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Name != packageName {
			continue
		}
		if strings.Compare(goServiceEntitySortKey(selector.Sel.Name), goServiceEntitySortKey(typeName)) > 0 {
			offset = fileSet.Position(field.Pos()).Offset
			break
		}
		offset = fileSet.Position(field.End()).Offset
	}
	return validGoPatch(content, insertGoLines(content, offset, line))
}

// insertGoFuncSelectorParameter 在同包类型参数中按类型名插入函数参数。
func insertGoFuncSelectorParameter(content string, functionName string, packageName string, typeName string, line string) string {
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return content
	}
	function := findGoFuncDecl(file, functionName)
	if function == nil || function.Type.Params == nil {
		return content
	}
	offset := fileSet.Position(function.Type.Params.Closing).Offset
	for _, field := range function.Type.Params.List {
		selector := goSelectorType(field.Type)
		if selector == nil || selector.Sel == nil {
			continue
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Name != packageName {
			continue
		}
		if strings.Compare(goServiceEntitySortKey(selector.Sel.Name), goServiceEntitySortKey(typeName)) > 0 {
			offset = fileSet.Position(field.Pos()).Offset
			break
		}
		offset = fileSet.Position(field.End()).Offset
	}
	return validGoPatch(content, insertGoLines(content, offset, line))
}

// insertGoCompositeLiteralField 在管理端字段分组内按字段名插入结构体字面量成员。
func insertGoCompositeLiteralField(content string, functionName string, typeName string, fieldName string, line string) string {
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return content
	}
	function := findGoFuncDecl(file, functionName)
	if function == nil {
		return content
	}
	var literal *ast.CompositeLit
	ast.Inspect(function.Body, func(node ast.Node) bool {
		candidate, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		identifier, ok := candidate.Type.(*ast.Ident)
		if ok && identifier.Name == typeName {
			literal = candidate
			return false
		}
		return true
	})
	if literal == nil {
		return content
	}
	offset := fileSet.Position(literal.Rbrace).Offset
	for _, element := range literal.Elts {
		keyValue, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		identifier, ok := keyValue.Key.(*ast.Ident)
		if !ok || !strings.HasPrefix(identifier.Name, "admin") {
			continue
		}
		if strings.Compare(goServiceFieldEntitySortKey(identifier.Name), goServiceFieldEntitySortKey(fieldName)) > 0 {
			offset = fileSet.Position(element.Pos()).Offset
			break
		}
		offset = fileSet.Position(element.End()).Offset
	}
	return validGoPatch(content, insertGoLines(content, offset, line))
}

// insertGoPackageCall 在函数内相同包的调用分组中按函数名插入调用。
func insertGoPackageCall(content string, functionName string, packageName string, callName string, line string) string {
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return content
	}
	function := findGoFuncDecl(file, functionName)
	if function == nil {
		return content
	}
	offset := fileSet.Position(function.Body.Rbrace).Offset
	for _, statement := range function.Body.List {
		expression, ok := statement.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := expression.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Name != packageName {
			continue
		}
		if strings.Compare(goRegistrationEntitySortKey(selector.Sel.Name), goRegistrationEntitySortKey(callName)) > 0 {
			offset = fileSet.Position(statement.Pos()).Offset
			break
		}
		offset = fileSet.Position(statement.End()).Offset
	}
	return validGoPatch(content, insertGoLines(content, offset, line))
}

// goQualifiedExpressionName 返回简单标识符或包选择表达式的完整名称。
func goQualifiedExpressionName(expression ast.Expr) string {
	switch value := expression.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.SelectorExpr:
		identifier, ok := value.X.(*ast.Ident)
		if ok {
			return identifier.Name + "." + value.Sel.Name
		}
	}
	return ""
}

// goProviderEntitySortKey 返回 ProviderSet 条目对应的实体排序键。
func goProviderEntitySortKey(providerName string) string {
	providerName = strings.TrimPrefix(providerName, "biz.")
	providerName = strings.TrimPrefix(providerName, "New")
	providerName = strings.TrimSuffix(providerName, "Case")
	providerName = strings.TrimSuffix(providerName, "Service")
	return stringcase.ToSnakeCase(providerName)
}

// goServiceEntitySortKey 返回服务类型对应的实体排序键。
func goServiceEntitySortKey(typeName string) string {
	return stringcase.ToSnakeCase(strings.TrimSuffix(typeName, "Service"))
}

// goServiceFieldEntitySortKey 返回管理端服务字段对应的实体排序键。
func goServiceFieldEntitySortKey(fieldName string) string {
	return stringcase.ToSnakeCase(strings.TrimPrefix(fieldName, "admin"))
}

// goRegistrationEntitySortKey 返回服务注册调用对应的实体排序键。
func goRegistrationEntitySortKey(callName string) string {
	callName = strings.TrimPrefix(callName, "Register")
	for _, suffix := range []string{"ServiceHTTPServer", "ServiceMCPTools", "ServiceServer"} {
		callName = strings.TrimSuffix(callName, suffix)
	}
	return stringcase.ToSnakeCase(callName)
}

// goIdentifierExists 判断 Go 源码是否已包含指定标识符，兼容缩写大小写差异。
func goIdentifierExists(content string, name string) bool {
	file, _, err := parseGoSource(content)
	if err != nil {
		return false
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		identifier, ok := node.(*ast.Ident)
		if ok && strings.EqualFold(identifier.Name, name) {
			found = true
			return false
		}
		return !found
	})
	return found
}

// goCallNameExists 判断 Go 源码是否已调用指定包中的指定函数。
func goCallNameExists(content string, packageName string, name string) bool {
	file, _, err := parseGoSource(content)
	if err != nil {
		return false
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch function := call.Fun.(type) {
		case *ast.Ident:
			return true
		case *ast.SelectorExpr:
			identifier, ok := function.X.(*ast.Ident)
			// 无法定位到目标包的调用不应影响当前注册函数的判定。
			if !ok || identifier.Name != packageName {
				return true
			}
			// 包名和函数名均相同才视为已完成注册。
			if function.Sel.Name == name {
				found = true
				return false
			}
		}
		return !found
	})
	return found
}

// goStructSelectorFieldName 返回结构体中指定包类型对应的字段名。
func goStructSelectorFieldName(content string, structName string, packageName string, typeName string) string {
	file, _, err := parseGoSource(content)
	if err != nil {
		return ""
	}
	structType := findGoStructType(file, structName)
	if structType == nil {
		return ""
	}
	for _, field := range structType.Fields.List {
		selector := goSelectorType(field.Type)
		if selector == nil || !strings.EqualFold(selector.Sel.Name, typeName) {
			continue
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Name != packageName || len(field.Names) == 0 {
			continue
		}
		return field.Names[0].Name
	}
	return ""
}

// findGoStructType 查找指定 Go 结构体。
func findGoStructType(file *ast.File, name string) *ast.StructType {
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, specification := range general.Specs {
			typeSpec, ok := specification.(*ast.TypeSpec)
			if !ok || typeSpec.Name.Name != name {
				continue
			}
			structType, _ := typeSpec.Type.(*ast.StructType)
			return structType
		}
	}
	return nil
}

// goFuncSelectorParamName 返回函数参数中指定包类型对应的参数名。
func goFuncSelectorParamName(content string, functionName string, packageName string, typeName string) string {
	file, _, err := parseGoSource(content)
	if err != nil {
		return ""
	}
	function := findGoFuncDecl(file, functionName)
	if function == nil || function.Type.Params == nil {
		return ""
	}
	for _, field := range function.Type.Params.List {
		selector := goSelectorType(field.Type)
		if selector == nil || !strings.EqualFold(selector.Sel.Name, typeName) {
			continue
		}
		identifier, ok := selector.X.(*ast.Ident)
		if !ok || identifier.Name != packageName || len(field.Names) == 0 {
			continue
		}
		return field.Names[0].Name
	}
	return ""
}

// goCompositeKeyExists 判断结构体字面量是否已包含指定字段。
func goCompositeKeyExists(literal *ast.CompositeLit, fieldName string) bool {
	for _, element := range literal.Elts {
		keyValue, isKeyValue := element.(*ast.KeyValueExpr)
		if !isKeyValue {
			continue
		}
		identifier, isIdentifier := keyValue.Key.(*ast.Ident)
		if isIdentifier && strings.EqualFold(identifier.Name, fieldName) {
			return true
		}
	}
	return false
}

// insertBeforeGoFuncReturn 在函数最终返回前插入代码。
func insertBeforeGoFuncReturn(content string, functionName string, line string) string {
	file, fileSet, err := parseGoSource(content)
	if err != nil {
		return content
	}
	function := findGoFuncDecl(file, functionName)
	if function == nil {
		return content
	}
	for index := len(function.Body.List) - 1; index >= 0; index-- {
		statement, ok := function.Body.List[index].(*ast.ReturnStmt)
		if !ok {
			continue
		}
		patched := insertGoLines(content, fileSet.Position(statement.Pos()).Offset, line)
		return validGoPatch(content, patched)
	}
	return content
}

// findGoFuncDecl 查找指定 Go 函数。
func findGoFuncDecl(file *ast.File, name string) *ast.FuncDecl {
	for _, declaration := range file.Decls {
		function, ok := declaration.(*ast.FuncDecl)
		if ok && function.Name.Name == name {
			return function
		}
	}
	return nil
}

// goFuncReceiverName 返回指定方法的 receiver 标识符名称。
func goFuncReceiverName(content string, functionName string) string {
	file, _, err := parseGoSource(content)
	if err != nil {
		return ""
	}
	function := findGoFuncDecl(file, functionName)
	if function == nil || function.Recv == nil || len(function.Recv.List) != 1 {
		return ""
	}
	receiver := function.Recv.List[0]
	if len(receiver.Names) != 1 {
		return ""
	}
	return receiver.Names[0].Name
}

// insertGoLines 在源码偏移位置插入完整代码行。
func insertGoLines(content string, offset int, lines string) string {
	if offset < 0 || offset > len(content) || lines == "" {
		return content
	}
	prefix := ""
	if offset > 0 && content[offset-1] != '\n' {
		prefix = "\n"
	}
	return content[:offset] + prefix + lines + "\n" + content[offset:]
}

// validGoPatch 仅返回语法有效的 Go 源码补丁。
func validGoPatch(original string, patched string) string {
	if _, _, err := parseGoSource(patched); err != nil {
		return original
	}
	return patched
}

// goReceiverName 返回方法接收者类型名。
func goReceiverName(expression ast.Expr) string {
	if pointer, ok := expression.(*ast.StarExpr); ok {
		expression = pointer.X
	}
	identifier, _ := expression.(*ast.Ident)
	if identifier == nil {
		return ""
	}
	return identifier.Name
}

// goSelectorType 返回字段类型中的包选择器。
func goSelectorType(expression ast.Expr) *ast.SelectorExpr {
	if pointer, ok := expression.(*ast.StarExpr); ok {
		expression = pointer.X
	}
	selector, _ := expression.(*ast.SelectorExpr)
	return selector
}

// parseGoSource 解析 Go 源文件并返回位置映射。
func parseGoSource(content string) (*ast.File, *token.FileSet, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "generated.go", content, parser.ParseComments)
	return file, fileSet, err
}

// ensureGeneratedGoImports 按追加的方法源码补齐必要导入。
func ensureGeneratedGoImports(content string, methodContent string) string {
	imports := []struct {
		marker     string
		importLine string
		importPath string
	}{
		{marker: "context.", importLine: `"context"`, importPath: "context"},
		{marker: "json.", importLine: `"encoding/json"`, importPath: "encoding/json"},
		{marker: "fmt.", importLine: `"fmt"`, importPath: "fmt"},
		{marker: "commonv1.", importLine: `commonv1 "github.com/liujitcn/kratos-admin/backend/core/api/gen/go/common/v1"`, importPath: "github.com/liujitcn/kratos-admin/backend/core/api/gen/go/common/v1"},
		{marker: "errorsx.", importLine: `"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"`, importPath: "github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"},
		{marker: "models.", importLine: `"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"`, importPath: "github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"},
		{marker: "mapper.", importLine: `"github.com/liujitcn/go-utils/mapper"`, importPath: "github.com/liujitcn/go-utils/mapper"},
		{marker: "_string.", importLine: `_string "github.com/liujitcn/go-utils/string"`, importPath: "github.com/liujitcn/go-utils/string"},
		{marker: "_time.", importLine: `_time "github.com/liujitcn/go-utils/time"`, importPath: "github.com/liujitcn/go-utils/time"},
		{marker: "repository.", importLine: `"github.com/liujitcn/gorm-kit/repository"`, importPath: "github.com/liujitcn/gorm-kit/repository"},
		{marker: "log.", importLine: `"github.com/go-kratos/kratos/v3/log"`, importPath: "github.com/go-kratos/kratos/v3/log"},
		{marker: "emptypb.", importLine: `"google.golang.org/protobuf/types/known/emptypb"`, importPath: "google.golang.org/protobuf/types/known/emptypb"},
	}
	for _, item := range imports {
		if strings.Contains(methodContent, item.marker) && !strings.Contains(content, `"`+item.importPath+`"`) {
			content = ensureGoImport(content, item.importLine)
		}
	}
	return content
}

// removeTSClassMethods 从 TypeScript 服务类中删除指定方法。
func removeTSClassMethods(content string, methodNames map[string]struct{}) string {
	for methodName := range methodNames {
		methodContent := extractTSClassMethod(content, methodName)
		if methodContent != "" {
			content = strings.Replace(content, methodContent, "", 1)
		}
	}
	return content
}

// extractTSClassMethod 从 TypeScript 服务类中提取指定方法。
func extractTSClassMethod(content string, methodName string) string {
	methodStart := strings.Index(content, "\n  "+methodName+"(")
	if methodStart < 0 {
		return ""
	}
	methodStart++
	blockStart := methodStart
	if docStart := strings.LastIndex(content[:methodStart], "\n  /**"); docStart >= 0 {
		docStart++
		docEnd := strings.Index(content[docStart:methodStart], "*/")
		if docEnd >= 0 && strings.TrimSpace(content[docStart+docEnd+2:methodStart]) == "" {
			blockStart = docStart
		}
	}
	openIndex := strings.Index(content[methodStart:], "{")
	if openIndex < 0 {
		return ""
	}
	openIndex += methodStart
	depth := 0
	for index := openIndex; index < len(content); index++ {
		switch content[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimRight(content[blockStart:index+1], "\n")
			}
		}
	}
	return ""
}

// mergeGeneratedTSClassMethods 按候选顺序替换生成方法，并将已有扩展方法原样保留在后面。
func mergeGeneratedTSClassMethods(content string, candidate string, className string) string {
	generatedBlocks, _, _, ok := tsClassMethodBlocks(candidate, className)
	if !ok || len(generatedBlocks) == 0 {
		return content
	}
	existingBlocks, firstStart, lastEnd, ok := tsClassMethodBlocks(content, className)
	if !ok {
		return content
	}
	generatedNames := make(map[string]struct{}, len(generatedBlocks))
	for _, block := range generatedBlocks {
		generatedNames[strings.ToLower(block.Name)] = struct{}{}
	}
	blocks := make([]CodeGenSourceMethodBlock, 0, len(generatedBlocks)+len(existingBlocks))
	blocks = append(blocks, generatedBlocks...)
	for _, block := range existingBlocks {
		if _, generated := generatedNames[strings.ToLower(block.Name)]; generated {
			continue
		}
		blocks = append(blocks, block)
	}
	methodContents := make([]string, 0, len(blocks))
	for _, block := range blocks {
		methodContents = append(methodContents, strings.Trim(block.Content, "\r\n"))
	}
	joinedMethods := strings.Join(methodContents, "\n\n")
	if len(existingBlocks) == 0 {
		classEnd := findTSClassEndIndex(content, className)
		if classEnd < 0 {
			return content
		}
		return pruneUnusedFrontendTypeImports(content[:classEnd] + "\n" + joinedMethods + "\n" + content[classEnd:])
	}
	return pruneUnusedFrontendTypeImports(content[:firstStart] + joinedMethods + content[lastEnd:])
}

// tsClassMethodBlocks 返回类中连续的方法块及其源码位置。
func tsClassMethodBlocks(content string, className string) ([]CodeGenSourceMethodBlock, int, int, bool) {
	classStart := strings.Index(content, "class "+className)
	classEnd := findTSClassEndIndex(content, className)
	if classStart < 0 || classEnd < 0 {
		return nil, -1, -1, false
	}
	classContent := content[classStart : classEnd+1]
	matches := tsClassMethodPattern.FindAllStringSubmatch(classContent, -1)
	blocks := make([]CodeGenSourceMethodBlock, 0, len(matches))
	searchOffset := 0
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		methodName := match[1]
		if _, exists := seen[methodName]; exists {
			continue
		}
		seen[methodName] = struct{}{}
		methodContent := extractTSClassMethod(classContent, methodName)
		if methodContent == "" {
			return nil, -1, -1, false
		}
		startOffset := strings.Index(classContent[searchOffset:], methodContent)
		if startOffset < 0 {
			return nil, -1, -1, false
		}
		startOffset += searchOffset
		endOffset := startOffset + len(methodContent)
		blocks = append(blocks, CodeGenSourceMethodBlock{
			Name:          methodName,
			Content:       methodContent,
			Start:         classStart + startOffset,
			End:           classStart + endOffset,
			OriginalIndex: len(blocks),
		})
		searchOffset = endOffset
	}
	for index := 1; index < len(blocks); index++ {
		if strings.TrimSpace(content[blocks[index-1].End:blocks[index].Start]) != "" {
			return nil, -1, -1, false
		}
	}
	if len(blocks) == 0 {
		return blocks, classEnd, classEnd, true
	}
	return blocks, blocks[0].Start, blocks[len(blocks)-1].End, true
}

// findTSClassEndIndex 查找 TypeScript 服务类的结束大括号。
func findTSClassEndIndex(content string, className string) int {
	classIndex := strings.Index(content, "class "+className)
	if classIndex < 0 {
		return -1
	}
	openIndex := strings.Index(content[classIndex:], "{")
	if openIndex < 0 {
		return -1
	}
	openIndex += classIndex
	depth := 0
	for index := openIndex; index < len(content); index++ {
		switch content[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}
