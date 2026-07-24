package codegen

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/liujitcn/go-utils/stringcase"
)

// --- Proto 契约渲染 ---

// renderProtoFile 渲染基础 Proto 文件。
func (c *renderer) renderProtoFile(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	entity := table.EntityName
	target := ProtoTargetForTable(table)
	resourcePath := c.frontendResourcePath(table)
	methods = c.mainProtoMethods(table, methods)

	var rpcBuilder strings.Builder
	for _, method := range methods {
		rpcBuilder.WriteString(c.renderProtoRPC(table, method, resourcePath))
		rpcBuilder.WriteString("\n")
	}
	var messageBuilder strings.Builder
	messageNames := make(map[string]struct{})
	for _, method := range methods {
		for _, messageName := range c.protoMessageNamesForMethod(table, method) {
			if _, exists := messageNames[messageName]; exists {
				continue
			}
			messageNames[messageName] = struct{}{}
			messageBuilder.WriteString(c.renderProtoMessageByName(table, columns, method, messageName))
			messageBuilder.WriteString("\n")
		}
	}
	return renderTemplate("proto.tmpl", protoTemplateData{
		Entity:       entity,
		TableComment: table.TableComment,
		PackageName:  target.PackageName,
		RPCs:         rpcBuilder.String(),
		Messages:     messageBuilder.String(),
	})
}

// mainProtoMethods 返回主实体已配置为缺失时生成的方法集合。
func (c *renderer) mainProtoMethods(table *Table, methods []*Proto) []*Proto {
	list := make([]*Proto, 0, len(methods))
	for _, method := range filterProtoMethods(methods, c.defaultProtoPath(table)) {
		if method.TargetEntityName != table.EntityName {
			continue
		}
		list = append(list, method)
	}
	return sortCodeGenProtoMethods(list)
}

// renderTargetProtoFile 渲染外部目标 Proto 的最小补齐文件。
func (c *renderer) renderTargetProtoFile(table *Table, columns []*CodeGenColumn, methods []*Proto, protoPath string) string {
	methods = sortCodeGenProtoMethods(filterProtoMethods(methods, protoPath))
	if len(methods) == 0 {
		return ""
	}
	target := methods[0].TargetEntityName
	protoTarget := ProtoTargetForTable(table)
	var builder strings.Builder
	builder.WriteString("syntax = \"proto3\";\n\n")
	builder.WriteString("package " + protoTarget.PackageName + ";\n\n")
	builder.WriteString("import \"common/v1/common.proto\";\n")
	builder.WriteString("import \"buf/validate/validate.proto\";\n")
	builder.WriteString("import \"gnostic/openapi/v3/annotations.proto\";\n")
	builder.WriteString("import \"google/api/annotations.proto\";\n")
	builder.WriteString("import \"google/protobuf/empty.proto\";\n\n")
	builder.WriteString("// Admin" + methods[0].TargetBusinessName + "服务\n")
	builder.WriteString("service " + target + "Service {\n")
	for _, method := range methods {
		builder.WriteString(c.renderProtoRPC(table, method, resourcePathByEntity(target)))
	}
	builder.WriteString("}\n\n")
	builder.WriteString(c.renderGeneratedProtoMessages(table, columns, methods, protoPath))
	return builder.String()
}

// renderProtoPatch 渲染已有 Proto 文件需要追加的 RPC 与 message。
func (c *renderer) renderProtoPatch(table *Table, columns []*CodeGenColumn, methods []*Proto, protoPath string) CodeGenProtoPatch {
	filtered := filterProtoMethods(methods, protoPath)
	patch := CodeGenProtoPatch{
		ServiceNames: make([]string, 0, 1),
		RPCs:         make(map[string][]string),
		Messages:     make([]string, 0, len(filtered)),
	}
	messageNames := make(map[string]struct{})
	// 一个 Proto 文件可能包含多个 service，RPC 按目标 service 分组，message 则在文件级去重。
	for _, method := range filtered {
		exists, _ := c.protoMethodExists(method.ProtoFilePath, method.TargetEntityName, method.MethodName)
		if !exists {
			serviceName := method.TargetEntityName + "Service"
			if _, ok := patch.RPCs[serviceName]; !ok {
				patch.ServiceNames = append(patch.ServiceNames, serviceName)
			}
			patch.RPCs[serviceName] = append(patch.RPCs[serviceName], c.renderProtoRPC(table, method, resourcePathByEntity(method.TargetEntityName)))
		}
		patch.Messages = append(patch.Messages, c.renderPatchProtoMessages(table, columns, method, messageNames)...)
	}
	return patch
}

// appendProtoPatch 将 RPC 插入 service 内部，并将 message 追加到文件末尾。
func (c *renderer) appendProtoPatch(content string, patch CodeGenProtoPatch) string {
	// 任一目标 service 不存在时放弃整次补丁，避免把 RPC 插到错误位置。
	for _, serviceName := range patch.ServiceNames {
		serviceStart, serviceEnd := findProtoServiceBounds(content, serviceName)
		if serviceStart < 0 || serviceEnd < 0 {
			return content
		}
	}
	// 只有新增方法实际引用公共响应类型时才补 common.proto 导入。
	if patch.CommonImportRequired() && !strings.Contains(content, "common/v1/common.proto") {
		content = insertProtoImport(content, "import \"common/v1/common.proto\";\n")
	}
	// 动态 GET 路由必须排在 /{id} 之前，否则 HTTP 路由会把静态段误识别为 ID。
	for _, serviceName := range patch.ServiceNames {
		rpcContent := strings.Join(patch.RPCs[serviceName], "")
		_, serviceEnd := findProtoServiceBounds(content, serviceName)
		insertIndex := findProtoDynamicGETInsertIndex(content, serviceName)
		if insertIndex < 0 {
			insertIndex = serviceEnd
		}
		content = content[:insertIndex] + "\n" + strings.TrimRight(rpcContent, "\n") + "\n" + content[insertIndex:]
	}
	return appendMissingProtoMessages(content, patch.Messages)
}

// mergeGeneratedProtoFile 整体替换候选中的生成 RPC 和消息，并将已有扩展定义原样保留在后面。
func mergeGeneratedProtoFile(content string, candidate string) string {
	if strings.TrimSpace(candidate) == "" {
		return content
	}
	for _, importLine := range protoImportLines(candidate) {
		if !strings.Contains(content, importLine) {
			content = insertProtoImport(content, importLine+"\n")
		}
	}
	serviceNames := protoServiceNames(candidate)
	for _, serviceName := range serviceNames {
		content = mergeGeneratedProtoService(content, candidate, serviceName)
	}
	return mergeGeneratedProtoMessages(content, candidate)
}

// protoImportLines 返回 Proto 文件的全部 import 声明。
func protoImportLines(content string) []string {
	pattern := regexp.MustCompile(`(?m)^[\t ]*import[\t ]+"[^"]+";[\t ]*$`)
	matches := pattern.FindAllString(content, -1)
	lines := make([]string, 0, len(matches))
	for _, match := range matches {
		lines = append(lines, strings.TrimSpace(match))
	}
	return lines
}

// protoServiceNames 返回文件中的顶层 service 名称。
func protoServiceNames(content string) []string {
	pattern := regexp.MustCompile(`(?m)^[\t ]*service[\t ]+([A-Za-z_][A-Za-z0-9_]*)[\t ]*\{`)
	matches := pattern.FindAllStringSubmatchIndex(content, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 4 || protoBraceDepthAt(content, match[0]) != 0 {
			continue
		}
		names = append(names, content[match[2]:match[3]])
	}
	return names
}

// mergeGeneratedProtoService 替换单个 service 的生成 RPC，并保持扩展 RPC 的内容和相对顺序。
func mergeGeneratedProtoService(content string, candidate string, serviceName string) string {
	candidateBlocks, _, _, _, _, ok := protoServiceRPCBlocks(candidate, serviceName)
	if !ok || len(candidateBlocks) == 0 {
		return content
	}
	existingBlocks, bodyStart, bodyEnd, prefix, suffix, ok := protoServiceRPCBlocks(content, serviceName)
	if !ok {
		return content
	}
	generatedNames := make(map[string]struct{}, len(candidateBlocks))
	for _, block := range candidateBlocks {
		generatedNames[block.Name] = struct{}{}
	}
	blocks := make([]CodeGenProtoRPCBlock, 0, len(candidateBlocks)+len(existingBlocks))
	blocks = append(blocks, candidateBlocks...)
	for _, block := range existingBlocks {
		if _, generated := generatedNames[block.Name]; generated {
			continue
		}
		blocks = append(blocks, block)
	}
	var builder strings.Builder
	prefix = strings.TrimRight(prefix, "\r\n")
	if prefix != "" {
		builder.WriteString(prefix)
		builder.WriteString("\n\n")
	} else {
		builder.WriteString("\n")
	}
	for index, block := range blocks {
		if index > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(strings.Trim(block.Content, "\r\n"))
	}
	suffix = strings.TrimLeft(suffix, "\r\n")
	if suffix != "" {
		builder.WriteString("\n\n")
		builder.WriteString(suffix)
	}
	builder.WriteString("\n")
	return content[:bodyStart] + builder.String() + content[bodyEnd:]
}

// protoServiceRPCBlocks 返回 service 内连续 RPC 块、正文边界及 RPC 前后的扩展内容。
func protoServiceRPCBlocks(content string, serviceName string) ([]CodeGenProtoRPCBlock, int, int, string, string, bool) {
	serviceStart, serviceEnd := findProtoServiceBounds(content, serviceName)
	if serviceStart < 0 || serviceEnd < 0 {
		return nil, -1, -1, "", "", false
	}
	bodyStart := serviceStart + 1
	body := content[bodyStart:serviceEnd]
	matches := protoRPCPattern.FindAllStringSubmatchIndex(body, -1)
	blocks := make([]CodeGenProtoRPCBlock, 0, len(matches))
	firstBlockStart := -1
	lastBlockEnd := -1
	for _, match := range matches {
		if len(match) < 4 {
			return nil, -1, -1, "", "", false
		}
		openOffset := strings.Index(body[match[0]:], "{")
		if openOffset < 0 {
			return nil, -1, -1, "", "", false
		}
		blockEnd := findProtoBlockEnd(body, match[0]+openOffset)
		if blockEnd < 0 {
			return nil, -1, -1, "", "", false
		}
		blockStart := protoDeclarationCommentStart(body, match[0])
		if lastBlockEnd >= 0 && strings.TrimSpace(body[lastBlockEnd:blockStart]) != "" {
			return nil, -1, -1, "", "", false
		}
		if firstBlockStart < 0 {
			firstBlockStart = blockStart
		}
		lastBlockEnd = blockEnd + 1
		blocks = append(blocks, CodeGenProtoRPCBlock{
			Name:          body[match[2]:match[3]],
			Content:       strings.Trim(body[blockStart:lastBlockEnd], "\r\n"),
			OriginalIndex: len(blocks),
		})
	}
	if len(blocks) == 0 {
		return blocks, bodyStart, serviceEnd, body, "", true
	}
	return blocks, bodyStart, serviceEnd, body[:firstBlockStart], body[lastBlockEnd:], true
}

type protoMessageBlock struct {
	name          string
	content       string
	start         int
	end           int
	originalIndex int
}

// mergeGeneratedProtoMessages 替换候选生成消息，并将已有扩展消息原样追加在后面。
func mergeGeneratedProtoMessages(content string, candidate string) string {
	candidateBlocks, _, _, ok := topLevelProtoMessageBlocks(candidate)
	if !ok || len(candidateBlocks) == 0 {
		return content
	}
	existingBlocks, firstStart, lastEnd, ok := topLevelProtoMessageBlocks(content)
	if !ok {
		return content
	}
	generatedNames := make(map[string]struct{}, len(candidateBlocks))
	for _, block := range candidateBlocks {
		generatedNames[block.name] = struct{}{}
	}
	blocks := make([]protoMessageBlock, 0, len(candidateBlocks)+len(existingBlocks))
	blocks = append(blocks, candidateBlocks...)
	for _, block := range existingBlocks {
		if _, generated := generatedNames[block.name]; generated {
			continue
		}
		blocks = append(blocks, block)
	}
	var builder strings.Builder
	for index, block := range blocks {
		if index > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(strings.Trim(block.content, "\r\n"))
	}
	if len(existingBlocks) == 0 {
		return strings.TrimRight(content, "\r\n") + "\n\n" + builder.String() + "\n"
	}
	prefix := strings.TrimRight(content[:firstStart], " \t\r\n")
	suffix := strings.TrimLeft(content[lastEnd:], " \t\r\n")
	if suffix == "" {
		return prefix + "\n\n" + builder.String() + "\n"
	}
	return prefix + "\n\n" + builder.String() + "\n\n" + suffix
}

// topLevelProtoMessageBlocks 返回连续顶层 message 块及其文件位置。
func topLevelProtoMessageBlocks(content string) ([]protoMessageBlock, int, int, bool) {
	matches := protoMessagePattern.FindAllStringSubmatchIndex(content, -1)
	blocks := make([]protoMessageBlock, 0, len(matches))
	firstStart := -1
	lastEnd := -1
	for _, match := range matches {
		if len(match) < 4 || protoBraceDepthAt(content, match[0]) != 0 {
			continue
		}
		openOffset := strings.Index(content[match[0]:match[1]], "{")
		if openOffset < 0 {
			return nil, -1, -1, false
		}
		blockEnd := findProtoBlockEnd(content, match[0]+openOffset)
		if blockEnd < 0 {
			return nil, -1, -1, false
		}
		blockStart := protoDeclarationCommentStart(content, match[0])
		if lastEnd >= 0 && strings.TrimSpace(content[lastEnd:blockStart]) != "" {
			return nil, -1, -1, false
		}
		if firstStart < 0 {
			firstStart = blockStart
		}
		lastEnd = blockEnd + 1
		blocks = append(blocks, protoMessageBlock{
			name:          content[match[2]:match[3]],
			content:       strings.Trim(content[blockStart:lastEnd], "\r\n"),
			start:         blockStart,
			end:           lastEnd,
			originalIndex: len(blocks),
		})
	}
	return blocks, firstStart, lastEnd, true
}

// GeneratedProtoMethodComment 返回生成器实际使用的 RPC 中文方法描述。
func GeneratedProtoMethodComment(businessName string, entityName string, triggerType string, apiKind string, methodName string) string {
	switch apiKind {
	case APIKindList:
		return "查询" + businessName + "分页列表"
	case APIKindTree:
		if triggerType == TriggerEntityOption || triggerType == TriggerFieldOption || triggerType == TriggerLeftTree {
			return "查询" + businessName + "树形选择"
		}
		return "查询" + businessName + "树形列表"
	case APIKindOption:
		return "查询" + businessName + "下拉选择"
	case APIKindStatus:
		return "设置状态"
	case APIKindCRUD:
		switch methodName {
		case "Get" + entityName:
			return "查询" + businessName + "详情"
		case "Create" + entityName:
			return "创建" + businessName
		case "Update" + entityName:
			return "更新" + businessName
		case "Delete" + entityName:
			return "删除" + businessName
		}
	}
	return "接口能力"
}

// renderProtoRPC 渲染单个 RPC 契约。
func (c *renderer) renderProtoRPC(table *Table, method *Proto, resourcePath string) string {
	businessName := codeGenProtoMethodBusinessName(table, method)
	switch method.APIKind {
	case APIKindList:
		return fmt.Sprintf(`  // %s
  rpc %s(%sRequest) returns (%sResponse) {
    option (google.api.http) = {
      get: "/api/v1/admin/%s"
    };
  }
`, GeneratedProtoMethodComment(businessName, table.EntityName, method.TriggerType, method.APIKind, method.MethodName), method.MethodName, method.MethodName, method.MethodName, resourcePath)
	case APIKindTree:
		if method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree {
			return fmt.Sprintf(`  // %s
  rpc %s(%sRequest) returns (.common.v1.TreeOptionResponse) {
    option (google.api.http) = {
      get: "/api/v1/admin/%s/option"
    };
  }
`, GeneratedProtoMethodComment(businessName, table.EntityName, method.TriggerType, method.APIKind, method.MethodName), method.MethodName, method.MethodName, resourcePath)
		}
		return fmt.Sprintf(`  // %s
  rpc %s(%sRequest) returns (%sResponse) {
    option (google.api.http) = {
      get: "/api/v1/admin/%s/tree"
    };
  }
`, GeneratedProtoMethodComment(businessName, table.EntityName, method.TriggerType, method.APIKind, method.MethodName), method.MethodName, method.MethodName, method.MethodName, resourcePath)
	case APIKindOption:
		return fmt.Sprintf(`  // %s
  rpc %s(%sRequest) returns (.common.v1.SelectOptionResponse) {
    option (google.api.http) = {
      get: "/api/v1/admin/%s/option"
    };
  }
`, GeneratedProtoMethodComment(businessName, table.EntityName, method.TriggerType, method.APIKind, method.MethodName), method.MethodName, method.MethodName, resourcePath)
	case APIKindStatus:
		return fmt.Sprintf(`  // %s
  rpc %s(%sRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      put: "/api/v1/admin/%s/{id}/%s"
      body: "*"
    };
  }
`, GeneratedProtoMethodComment(businessName, table.EntityName, method.TriggerType, method.APIKind, method.MethodName), method.MethodName, method.MethodName, resourcePath, statusResourcePath(table, method))
	}
	return c.renderCRUDProtoRPC(table, method, resourcePath)
}

// GeneratedHTTPRoute 返回生成接口对应的 HTTP 方法和路径。
func GeneratedHTTPRoute(table *Table, method *Proto) (string, string, bool) {
	if table == nil || method == nil {
		return "", "", false
	}
	targetEntity := DefaultString(method.TargetEntityName, table.EntityName)
	resourcePath := resourcePathByEntity(targetEntity)
	switch method.APIKind {
	case APIKindList:
		return "GET", "/api/v1/admin/" + resourcePath, true
	case APIKindOption, APIKindTree:
		if method.APIKind == APIKindTree && method.TriggerType == TriggerPageTree {
			return "GET", "/api/v1/admin/" + resourcePath + "/tree", true
		}
		return "GET", "/api/v1/admin/" + resourcePath + "/option", true
	case APIKindStatus:
		return "PUT", "/api/v1/admin/" + resourcePath + "/{id}/" + statusResourcePath(table, method), true
	case APIKindCRUD:
		switch method.MethodName {
		case "Get" + targetEntity:
			return "GET", "/api/v1/admin/" + resourcePath + "/{id}", true
		case "Create" + targetEntity:
			return "POST", "/api/v1/admin/" + resourcePath, true
		case "Update" + targetEntity:
			return "PUT", "/api/v1/admin/" + resourcePath + "/{id}", true
		case "Delete" + targetEntity:
			return "DELETE", "/api/v1/admin/" + resourcePath + "/{ids}", true
		}
	}
	return "", "", false
}

// renderCRUDProtoRPC 渲染标准 CRUD RPC。
func (c *renderer) renderCRUDProtoRPC(table *Table, method *Proto, resourcePath string) string {
	entity := table.EntityName
	businessName := codeGenProtoMethodBusinessName(table, method)
	switch method.MethodName {
	case "Get" + entity:
		return fmt.Sprintf(`  // %s
  rpc Get%s(Get%sRequest) returns (%sForm) {
    option (google.api.http) = {
      get: "/api/v1/admin/%s/{id}"
    };
  }
`, GeneratedProtoMethodComment(businessName, entity, method.TriggerType, method.APIKind, method.MethodName), entity, entity, entity, resourcePath)
	case "Create" + entity:
		return fmt.Sprintf(`  // %s
  rpc Create%s(Create%sRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      post: "/api/v1/admin/%s"
      body: "%s"
    };
  }
`, GeneratedProtoMethodComment(businessName, entity, method.TriggerType, method.APIKind, method.MethodName), entity, entity, resourcePath, stringcase.ToSnakeCase(entity))
	case "Update" + entity:
		return fmt.Sprintf(`  // %s
  rpc Update%s(Update%sRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      put: "/api/v1/admin/%s/{id}"
      body: "%s"
    };
  }
`, GeneratedProtoMethodComment(businessName, entity, method.TriggerType, method.APIKind, method.MethodName), entity, entity, resourcePath, stringcase.ToSnakeCase(entity))
	case "Delete" + entity:
		return fmt.Sprintf(`  // %s
  rpc Delete%s(Delete%sRequest) returns (google.protobuf.Empty) {
    option (google.api.http) = {
      delete: "/api/v1/admin/%s/{ids}"
    };
  }
`, GeneratedProtoMethodComment(businessName, entity, method.TriggerType, method.APIKind, method.MethodName), entity, entity, resourcePath)
	default:
		return ""
	}
}

// renderProtoMessage 渲染 Proto 数据消息。
func (c *renderer) renderProtoMessage(table *Table, columns []*CodeGenColumn, form bool) string {
	entity := table.EntityName
	name := entity
	title := table.BusinessName
	if form {
		name += "Form"
		title += "表单"
	}
	var builder strings.Builder
	builder.WriteString("// " + title + "\n")
	builder.WriteString("message " + name + " {\n")
	fieldNo := int32(1)
	treeParentColumn := ""
	treeLabelColumn := ""
	if !form && isTreePageType(table.PageType) {
		treeParentColumn = DefaultString(table.ParentColumn, "parent_id")
		treeLabelColumn = table.TreeLabelColumn
		if treeLabelColumn == "" {
			_, treeLabelColumn, _ = EntityOptionColumns(table, columns)
		}
	}
	for _, column := range columns {
		isTreeField := column.Name == treeParentColumn || column.Name == treeLabelColumn
		if !form && (column.Name == "deleted_at" || !isTreeField && !generatedListIncludesColumn(column) && column.IsPrimary != 1 && column.Name != "created_at" && column.Name != "updated_at") {
			continue
		}
		if form && !generatedFormIncludesColumn(column) && column.IsPrimary != 1 {
			continue
		}
		if form && isFormTreeMultiple(column) {
			builder.WriteString(c.renderFormTreeMultipleProtoField(column, fieldNo))
		} else {
			builder.WriteString(c.renderProtoField(column, fieldNo, form && column.IsPrimary != 1 && !generatedFormRequired(column), form))
		}
		fieldNo++
	}
	if !form && isTreePageType(table.PageType) {
		if table.PageType == PageTypeTreeLazy {
			builder.WriteString("\n  bool has_children = 299 [(gnostic.openapi.v3.property) = {description: \"是否存在子节点\"}]; // 是否存在子节点\n")
		} else {
			builder.WriteString(fmt.Sprintf("\n  repeated %s children = 300 [(gnostic.openapi.v3.property) = {description: \"子节点树\"}]; // 子节点树\n", entity))
		}
	}
	builder.WriteString("}\n")
	return builder.String()
}

// isTreePageType 判断页面类型是否使用树形实体结构。
func isTreePageType(pageType string) bool {
	return pageType == PageTypeTree || pageType == PageTypeTreeLazy
}

// appendMissingProtoMessages 补齐缺失 message 和字段，并按 RPC 顺序整理消息位置。
func appendMissingProtoMessages(content string, messages []string) string {
	missingMessages := make([]string, 0, len(messages))
	for _, message := range messages {
		messageName := protoMessageName(message)
		if messageName == "" {
			continue
		}
		messageStart, messageEnd := findProtoMessageBounds(content, messageName)
		if messageStart < 0 || messageEnd < 0 {
			missingMessages = append(missingMessages, message)
			continue
		}
		content = appendMissingProtoMessageFields(content, messageStart, messageEnd, message)
	}
	if len(missingMessages) == 0 {
		return normalizeProtoMessageOrder(content)
	}
	content = strings.TrimRight(content, "\n") + "\n\n" + strings.TrimSpace(strings.Join(missingMessages, "\n")) + "\n"
	return normalizeProtoMessageOrder(content)
}

type protoMessageField struct {
	name        string
	number      int
	numberStart int
	numberEnd   int
	content     string
	start       int
	end         int
}

// normalizeProtoMessageOrder 按文件内 RPC 顺序整理顶层请求和响应消息，其他消息保持相对顺序。
func normalizeProtoMessageOrder(content string) string {
	messageOrder := protoRPCMessageOrder(content)
	if len(messageOrder) == 0 {
		return content
	}
	type messageBlock struct {
		name          string
		content       string
		originalIndex int
	}
	matches := protoMessagePattern.FindAllStringSubmatchIndex(content, -1)
	blocks := make([]messageBlock, 0, len(matches))
	firstBlockStart := -1
	lastBlockEnd := -1
	for _, match := range matches {
		if len(match) < 4 || protoBraceDepthAt(content, match[0]) != 0 {
			continue
		}
		openOffset := strings.Index(content[match[0]:match[1]], "{")
		if openOffset < 0 {
			return content
		}
		blockEnd := findProtoBlockEnd(content, match[0]+openOffset)
		if blockEnd < 0 {
			return content
		}
		blockStart := protoDeclarationCommentStart(content, match[0])
		if lastBlockEnd >= 0 && strings.TrimSpace(content[lastBlockEnd:blockStart]) != "" {
			return content
		}
		if firstBlockStart < 0 {
			firstBlockStart = blockStart
		}
		lastBlockEnd = blockEnd + 1
		blocks = append(blocks, messageBlock{
			name:          content[match[2]:match[3]],
			content:       strings.Trim(content[blockStart:lastBlockEnd], "\r\n"),
			originalIndex: len(blocks),
		})
	}
	if len(blocks) < 2 {
		return content
	}
	// RPC 对应的请求与响应优先按接口位置排列，实体、表单及人工消息保留在其后。
	slices.SortStableFunc(blocks, func(a messageBlock, b messageBlock) int {
		aOrder, aKnown := messageOrder[a.name]
		bOrder, bKnown := messageOrder[b.name]
		if aKnown && bKnown && aOrder != bOrder {
			return aOrder - bOrder
		}
		if aKnown != bKnown {
			if aKnown {
				return -1
			}
			return 1
		}
		return a.originalIndex - b.originalIndex
	})
	prefix := strings.TrimRight(content[:firstBlockStart], " \t\r\n")
	suffix := strings.TrimSpace(content[lastBlockEnd:])
	var builder strings.Builder
	if prefix != "" {
		builder.WriteString(prefix)
		builder.WriteString("\n\n")
	}
	for index, block := range blocks {
		if index > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(block.content)
	}
	if suffix != "" {
		builder.WriteString("\n\n")
		builder.WriteString(suffix)
	}
	builder.WriteString("\n")
	return builder.String()
}

// protoRPCMessageOrder 返回由 RPC 方法名推导出的请求和响应消息顺序。
func protoRPCMessageOrder(content string) map[string]int {
	matches := protoRPCPattern.FindAllStringSubmatchIndex(content, -1)
	order := make(map[string]int, len(matches)*2)
	for index, match := range matches {
		if len(match) < 4 {
			continue
		}
		methodName := content[match[2]:match[3]]
		requestName := methodName + "Request"
		responseName := methodName + "Response"
		if _, exists := order[requestName]; !exists {
			order[requestName] = index * 2
		}
		if _, exists := order[responseName]; !exists {
			order[responseName] = index*2 + 1
		}
	}
	return order
}

// appendMissingProtoMessageFields 按候选 message 的字段顺序补齐和整理已有字段。
func appendMissingProtoMessageFields(content string, messageStart, messageEnd int, candidate string) string {
	candidateStart, candidateEnd := findProtoMessageBounds(candidate, protoMessageName(candidate))
	if candidateStart < 0 || candidateEnd < 0 {
		return content
	}
	existingBody := content[messageStart+1 : messageEnd]
	existingFields := protoMessageFields(existingBody)
	candidateFields := protoMessageFields(candidate[candidateStart+1 : candidateEnd])
	// 仅重建完整字段区域，避免覆盖含有 oneof、reserved 等人工结构的 message。
	if len(candidateFields) == 0 || !isProtoMessageFieldOnly(existingBody, existingFields) || !isProtoMessageFieldOnly(candidate[candidateStart+1:candidateEnd], candidateFields) {
		return content
	}
	fields := mergeProtoMessageFields(existingFields, candidateFields)
	if sameProtoMessageFieldOrder(existingFields, fields) {
		return content
	}
	body := renderProtoMessageFields(fields)
	return content[:messageStart+1] + body + content[messageEnd:]
}

// protoMessageFields 返回 message 中可由生成器维护的字段声明和原始位置。
func protoMessageFields(content string) []protoMessageField {
	matches := protoMessageFieldPattern.FindAllStringSubmatchIndex(content, -1)
	fields := make([]protoMessageField, 0, len(matches))
	for _, match := range matches {
		if len(match) < 6 {
			continue
		}
		number, err := strconv.Atoi(content[match[4]:match[5]])
		if err != nil {
			continue
		}
		fieldStart := match[0]
		fieldStart = protoDeclarationCommentStart(content, fieldStart)
		fields = append(fields, protoMessageField{
			name:        content[match[2]:match[3]],
			number:      number,
			numberStart: match[4] - fieldStart,
			numberEnd:   match[5] - fieldStart,
			content:     strings.TrimRight(content[fieldStart:match[1]], "\r\n"),
			start:       fieldStart,
			end:         match[1],
		})
	}
	return fields
}

// isProtoMessageFieldOnly 判断 message 内容是否只包含字段声明和字段注释。
func isProtoMessageFieldOnly(content string, fields []protoMessageField) bool {
	cursor := 0
	for _, field := range fields {
		// 字段之外存在其他结构时保留人工 message，避免代码生成覆盖其语义。
		if strings.TrimSpace(content[cursor:field.start]) != "" {
			return false
		}
		cursor = field.end
	}
	return strings.TrimSpace(content[cursor:]) == ""
}

// mergeProtoMessageFields 按候选字段顺序合并已有字段，并将人工扩展字段保留在末尾。
func mergeProtoMessageFields(existingFields, candidateFields []protoMessageField) []protoMessageField {
	existingByName := make(map[string]protoMessageField, len(existingFields))
	for _, field := range existingFields {
		if _, exists := existingByName[field.name]; !exists {
			existingByName[field.name] = field
		}
	}
	fields := make([]protoMessageField, 0, len(existingFields)+len(candidateFields))
	generatedNames := make(map[string]struct{}, len(candidateFields))
	usedNumbers := make(map[int]struct{}, len(existingFields)+len(candidateFields))
	maxNumber := 0
	for _, candidateField := range candidateFields {
		generatedNames[candidateField.name] = struct{}{}
		field, exists := existingByName[candidateField.name]
		if !exists {
			field = candidateField
		}
		field = withProtoMessageFieldNumber(field, candidateField.number)
		fields = append(fields, field)
		usedNumbers[field.number] = struct{}{}
		if field.number > maxNumber {
			maxNumber = field.number
		}
	}
	for _, field := range existingFields {
		if _, generated := generatedNames[field.name]; generated {
			continue
		}
		// 人工字段排在生成字段之后；编号冲突或倒序时顺延，保持 Proto 定义有效。
		if _, used := usedNumbers[field.number]; used || field.number <= maxNumber {
			field = withProtoMessageFieldNumber(field, nextProtoMessageFieldNumber(usedNumbers, maxNumber+1))
		}
		fields = append(fields, field)
		usedNumbers[field.number] = struct{}{}
		if field.number > maxNumber {
			maxNumber = field.number
		}
	}
	return fields
}

// withProtoMessageFieldNumber 更新字段编号并保留字段其他声明内容。
func withProtoMessageFieldNumber(field protoMessageField, number int) protoMessageField {
	if field.number == number {
		return field
	}
	field.content = field.content[:field.numberStart] + strconv.Itoa(number) + field.content[field.numberEnd:]
	field.number = number
	field.numberEnd = field.numberStart + len(strconv.Itoa(number))
	return field
}

// nextProtoMessageFieldNumber 返回从指定编号起未被占用的字段编号。
func nextProtoMessageFieldNumber(usedNumbers map[int]struct{}, number int) int {
	for {
		if _, used := usedNumbers[number]; !used {
			return number
		}
		number++
	}
}

// sameProtoMessageFieldOrder 判断已有字段是否已按目标顺序和编号排列。
func sameProtoMessageFieldOrder(existingFields, fields []protoMessageField) bool {
	if len(existingFields) != len(fields) {
		return false
	}
	for index, field := range fields {
		if existingFields[index].name != field.name || existingFields[index].number != field.number {
			return false
		}
	}
	return true
}

// renderProtoMessageFields 渲染字段区域，统一保留字段之间的空行。
func renderProtoMessageFields(fields []protoMessageField) string {
	if len(fields) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("\n")
	for index, field := range fields {
		if index > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(field.content)
	}
	builder.WriteString("\n")
	return builder.String()
}

// protoMessageName 返回 message 定义中的名称。
func protoMessageName(content string) string {
	match := protoMessagePattern.FindStringSubmatch(content)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

// findProtoMessageBounds 返回指定 message 的大括号起止位置。
func findProtoMessageBounds(content string, messageName string) (int, int) {
	if messageName == "" {
		return -1, -1
	}
	pattern := regexp.MustCompile(`(?m)^[\t ]*message[\t ]+` + regexp.QuoteMeta(messageName) + `[\t ]*\{`)
	location := pattern.FindStringIndex(content)
	if location == nil {
		return -1, -1
	}
	openIndex := strings.LastIndex(content[location[0]:location[1]], "{") + location[0]
	closeIndex := findProtoBlockEnd(content, openIndex)
	if closeIndex < 0 {
		return -1, -1
	}
	return openIndex, closeIndex
}

// normalizeProtoRPCOrder 按服务分别整理已有 RPC，未识别的方法保持原有相对顺序。
func normalizeProtoRPCOrder(content string, methods []*Proto, protoPath string) string {
	serviceMethods := make(map[string][]*Proto)
	// 先按 service 拆分，避免不同实体的同名方法相互影响排序。
	for _, method := range methods {
		if method.ProtoFilePath != protoPath {
			continue
		}
		serviceName := method.TargetEntityName + "Service"
		serviceMethods[serviceName] = append(serviceMethods[serviceName], method)
	}
	serviceNames := make([]string, 0, len(serviceMethods))
	for serviceName := range serviceMethods {
		serviceNames = append(serviceNames, serviceName)
	}
	slices.Sort(serviceNames)
	for _, serviceName := range serviceNames {
		content = normalizeProtoServiceRPCOrder(content, serviceName, serviceMethods[serviceName])
	}
	return content
}

// normalizeProtoServiceRPCOrder 按固定槽位重排单个 service 的 RPC 块。
func normalizeProtoServiceRPCOrder(content string, serviceName string, methods []*Proto) string {
	serviceStart, serviceEnd := findProtoServiceBounds(content, serviceName)
	if serviceStart < 0 || serviceEnd < 0 {
		return content
	}
	bodyStart := serviceStart + 1
	body := content[bodyStart:serviceEnd]
	matches := protoRPCPattern.FindAllStringSubmatchIndex(body, -1)
	if len(matches) < 2 {
		return content
	}
	blocks := make([]CodeGenProtoRPCBlock, 0, len(matches))
	methodNames := make(map[string]struct{}, len(matches))
	firstBlockStart := -1
	lastBlockEnd := -1
	// 每个块包含紧邻方法的注释，遇到无法识别的非空间隔时保持原文不动。
	for _, match := range matches {
		if len(match) < 4 {
			return content
		}
		openOffset := strings.Index(body[match[0]:], "{")
		if openOffset < 0 {
			return content
		}
		blockEnd := findProtoBlockEnd(body, match[0]+openOffset)
		if blockEnd < 0 {
			return content
		}
		blockStart := protoDeclarationCommentStart(body, match[0])
		if lastBlockEnd >= 0 && strings.TrimSpace(body[lastBlockEnd:blockStart]) != "" {
			return content
		}
		if firstBlockStart < 0 {
			firstBlockStart = blockStart
		}
		lastBlockEnd = blockEnd + 1
		name := body[match[2]:match[3]]
		// 同名 RPC 不能在同一 service 中重复声明，保留首次出现的稳定定义。
		if _, exists := methodNames[name]; exists {
			continue
		}
		methodNames[name] = struct{}{}
		blocks = append(blocks, CodeGenProtoRPCBlock{
			Name:          name,
			Content:       strings.Trim(body[blockStart:lastBlockEnd], "\r\n"),
			OriginalIndex: len(blocks),
		})
	}
	// 已知生成方法进入固定槽位，人工扩展方法保持原有相对顺序并排在其后。
	methodOrder := make(map[string]int, len(methods))
	for i, method := range sortCodeGenProtoMethods(methods) {
		if _, ok := methodOrder[method.MethodName]; !ok {
			methodOrder[method.MethodName] = i
		}
	}
	slices.SortStableFunc(blocks, func(a CodeGenProtoRPCBlock, b CodeGenProtoRPCBlock) int {
		aOrder, aKnown := methodOrder[a.Name]
		bOrder, bKnown := methodOrder[b.Name]
		if aKnown && bKnown && aOrder != bOrder {
			return aOrder - bOrder
		}
		if aKnown != bKnown {
			if aKnown {
				return -1
			}
			return 1
		}
		return a.OriginalIndex - b.OriginalIndex
	})
	prefix := strings.TrimRight(body[:firstBlockStart], " \t\r\n")
	suffix := strings.TrimSpace(body[lastBlockEnd:])
	// 只重建 service 内连续 RPC 区域，service 外部内容和无法识别的尾部内容保持不变。
	var builder strings.Builder
	builder.WriteString(prefix)
	if prefix == "" {
		builder.WriteString("\n")
	} else {
		builder.WriteString("\n\n")
	}
	for i, block := range blocks {
		if i > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(block.Content)
	}
	if suffix != "" {
		builder.WriteString("\n\n")
		builder.WriteString(suffix)
	}
	builder.WriteString("\n")
	return content[:bodyStart] + builder.String() + content[serviceEnd:]
}

// dedupeProtoMessageBlocks 删除重复的顶层 message 声明并保留首次出现的定义。
func dedupeProtoMessageBlocks(content string) string {
	matches := protoMessagePattern.FindAllStringSubmatchIndex(content, -1)
	if len(matches) < 2 {
		return content
	}
	seen := make(map[string]struct{}, len(matches))
	var builder strings.Builder
	cursor := 0
	// 仅处理顶层 message，嵌套 message 即使同名也属于不同作用域。
	for _, match := range matches {
		if len(match) < 4 || protoBraceDepthAt(content, match[0]) != 0 {
			continue
		}
		name := content[match[2]:match[3]]
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			continue
		}
		openOffset := strings.Index(content[match[0]:match[1]], "{")
		if openOffset < 0 {
			continue
		}
		blockEnd := findProtoBlockEnd(content, match[0]+openOffset)
		if blockEnd < 0 {
			continue
		}
		blockStart := protoDeclarationCommentStart(content, match[0])
		builder.WriteString(content[cursor:blockStart])
		cursor = blockEnd + 1
		for cursor < len(content) && content[cursor] == '\n' {
			cursor++
		}
	}
	if cursor == 0 {
		return content
	}
	builder.WriteString(content[cursor:])
	return builder.String()
}

// protoBraceDepthAt 返回指定位置之前的 Proto 大括号嵌套深度。
func protoBraceDepthAt(content string, position int) int {
	depth := 0
	for i := 0; i < position; i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
		}
	}
	return depth
}

// protoDeclarationCommentStart 返回声明及其紧邻注释块的起始位置。
func protoDeclarationCommentStart(content string, declarationStart int) int {
	lineStart := strings.LastIndex(content[:declarationStart], "\n") + 1
	for lineStart > 0 {
		previousLineEnd := lineStart - 1
		previousLineStart := strings.LastIndex(content[:previousLineEnd], "\n") + 1
		if !strings.HasPrefix(strings.TrimSpace(content[previousLineStart:previousLineEnd]), "//") {
			break
		}
		lineStart = previousLineStart
	}
	return lineStart
}

// findProtoDynamicGETInsertIndex 查找 service 中首个动态 GET RPC 的插入位置。
func findProtoDynamicGETInsertIndex(content string, serviceName string) int {
	serviceStart, serviceEnd := findProtoServiceBounds(content, serviceName)
	if serviceStart < 0 || serviceEnd < 0 {
		return -1
	}
	bodyStart := serviceStart + 1
	body := content[bodyStart:serviceEnd]
	dynamicGETPattern := regexp.MustCompile(`(?m)^[\t ]*get:[\t ]*"[^"]*\{[^"]*"[\t ]*$`)
	dynamicGET := dynamicGETPattern.FindStringIndex(body)
	if dynamicGET == nil {
		return -1
	}
	prefix := body[:dynamicGET[0]]
	rpcIndex := strings.LastIndex(prefix, "\n  rpc ")
	if rpcIndex < 0 {
		return -1
	}
	rpcIndex++
	commentIndex := strings.LastIndex(prefix[:rpcIndex], "\n  // ")
	if commentIndex >= 0 {
		return bodyStart + commentIndex + 1
	}
	return bodyStart + rpcIndex
}

// insertProtoImport 将 import 插入 package 声明之后。
func insertProtoImport(content string, importLine string) string {
	packageIndex := strings.Index(content, "package ")
	if packageIndex < 0 {
		return importLine + content
	}
	lineEnd := strings.Index(content[packageIndex:], "\n")
	if lineEnd < 0 {
		return content + "\n\n" + importLine
	}
	insertIndex := packageIndex + lineEnd + 1
	return content[:insertIndex] + "\n" + importLine + content[insertIndex:]
}

// ensureGoImport 确保 Go 文件包含指定 import。
func ensureGoImport(content string, importLine string) string {
	if strings.Contains(content, importLine) {
		return content
	}
	importBlock := "import (\n"
	if strings.Contains(content, importBlock) {
		return strings.Replace(content, importBlock, importBlock+"\t"+importLine+"\n", 1)
	}
	importIndex := strings.Index(content, "import ")
	if importIndex < 0 {
		return content
	}
	lineEnd := strings.Index(content[importIndex:], "\n")
	if lineEnd < 0 {
		return content
	}
	existingImport := strings.TrimSpace(content[importIndex : importIndex+lineEnd])
	return content[:importIndex] + "import (\n\t" + strings.TrimPrefix(existingImport, "import ") + "\n\t" + importLine + "\n)" + content[importIndex+lineEnd:]
}

// ensureTSNamedTypeNames 确保 TS 文件从指定模块导入类型集合。
func ensureTSNamedTypeNames(content string, importPath string, typeNames []string) string {
	pattern := regexp.MustCompile(`(?ms)import\s+(?:type\s+)?\{([^}]*)\}\s+from\s+"` + regexp.QuoteMeta(importPath) + `";`)
	matches := pattern.FindAllStringSubmatchIndex(content, -1)
	names := make([]string, 0, len(typeNames))
	seen := make(map[string]struct{}, len(typeNames))
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}
		for _, item := range strings.Split(content[match[2]:match[3]], ",") {
			name := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(item), "type "))
			if name == "" {
				continue
			}
			key := strings.Fields(name)[0]
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			names = append(names, name)
		}
	}
	for _, typeName := range typeNames {
		if typeName == "" {
			continue
		}
		if _, ok := seen[typeName]; ok {
			continue
		}
		seen[typeName] = struct{}{}
		names = append(names, typeName)
	}
	if len(names) == 0 {
		return content
	}
	slices.Sort(names)
	imports := make([]string, 0, len(names))
	for _, name := range names {
		imports = append(imports, "  type "+name)
	}
	declaration := "import {\n" + strings.Join(imports, ",\n") + "\n} from \"" + importPath + "\";"
	if len(matches) == 0 {
		return declaration + "\n" + content
	}
	var builder strings.Builder
	lastIndex := 0
	for index, match := range matches {
		builder.WriteString(content[lastIndex:match[0]])
		if index == 0 {
			builder.WriteString(declaration)
		}
		lastIndex = match[1]
	}
	builder.WriteString(content[lastIndex:])
	return builder.String()
}

// ensureTSCommonOptionImport 确保 TS 文件只导入方法实际使用的选项响应类型。
func ensureTSCommonOptionImport(content string, methods []*Proto) string {
	typeNames := tsCommonOptionResponseTypes(methods)
	if len(typeNames) == 0 {
		return content
	}
	if strings.Contains(content, "@/rpc/common/v1/common") {
		for _, typeName := range typeNames {
			if !strings.Contains(content, typeName) {
				fromLine := "} from \"@/rpc/common/v1/common\";"
				index := strings.Index(content, fromLine)
				if index >= 0 {
					content = content[:index] + ", " + typeName + " " + content[index:]
				}
			}
		}
		return content
	}
	importLine := "import type { " + strings.Join(typeNames, ", ") + " } from \"@/rpc/common/v1/common\";"
	insertIndex := strings.Index(content, "\n\nconst ")
	if insertIndex < 0 {
		return importLine + "\n" + content
	}
	return content[:insertIndex] + "\n" + importLine + content[insertIndex:]
}

// tsCommonOptionResponseTypes 返回选项方法实际使用的公共响应类型。
func tsCommonOptionResponseTypes(methods []*Proto) []string {
	typeNames := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	for _, method := range methods {
		typeName := "SelectOptionResponse"
		if method.APIKind == APIKindTree {
			typeName = "TreeOptionResponse"
		}
		if _, ok := seen[typeName]; ok {
			continue
		}
		seen[typeName] = struct{}{}
		typeNames = append(typeNames, typeName)
	}
	return typeNames
}

// protoServiceMethodExists 判断目标 Proto service 内是否已经存在指定 RPC。
func protoServiceMethodExists(content string, serviceName string, methodName string) bool {
	serviceStart, serviceEnd := findProtoServiceBounds(content, serviceName)
	if serviceStart < 0 || serviceEnd < 0 {
		return false
	}
	matches := protoRPCPattern.FindAllStringSubmatch(content[serviceStart:serviceEnd], -1)
	for _, match := range matches {
		if len(match) > 1 && match[1] == methodName {
			return true
		}
	}
	return false
}

// findProtoServiceBounds 返回指定 Proto service 的大括号起止位置。
func findProtoServiceBounds(content string, serviceName string) (int, int) {
	if serviceName == "" {
		return -1, -1
	}
	pattern := regexp.MustCompile(`(?m)^[\t ]*service[\t ]+` + regexp.QuoteMeta(serviceName) + `[\t ]*\{`)
	location := pattern.FindStringIndex(content)
	if location == nil {
		return -1, -1
	}
	openIndex := strings.LastIndex(content[location[0]:location[1]], "{") + location[0]
	closeIndex := findProtoBlockEnd(content, openIndex)
	if closeIndex < 0 {
		return -1, -1
	}
	return openIndex, closeIndex
}

// findProtoBlockEnd 返回指定左大括号对应的右大括号位置。
func findProtoBlockEnd(content string, openIndex int) int {
	depth := 0
	for i := openIndex; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// codeGenProtoMethodBusinessName 返回 Proto 方法描述使用的业务名称。
func codeGenProtoMethodBusinessName(table *Table, method *Proto) string {
	if method.TargetEntityName == "" || method.TargetEntityName == table.EntityName {
		return table.BusinessName
	}
	return DefaultString(method.TargetBusinessName, method.TargetEntityName)
}
