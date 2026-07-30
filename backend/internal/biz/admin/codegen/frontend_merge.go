package codegen

import (
	"regexp"
	"strings"
)

var (
	frontendNamedDeclarationPattern = regexp.MustCompile(`(?s)^(?:export\s+)?(?:const|let|var|type|interface|enum)\s+([A-Za-z_$][A-Za-z0-9_$]*)`)
	frontendFunctionPattern         = regexp.MustCompile(`(?s)^(?:export\s+)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`)
	frontendImportPathPattern       = regexp.MustCompile(`(?s)\bfrom\s+["']([^"']+)["']`)
	frontendImportNamedPattern      = regexp.MustCompile(`(?s)\{(.*)\}`)
	frontendTemplateAttrPattern     = regexp.MustCompile(`(?:^|\s)(ref|class|v-model|:request-api|prop|label|name)\s*=\s*["']([^"']+)["']`)
)

type frontendSourceBlock struct {
	key      string
	content  string
	kind     string
	function bool
}

type frontendImportSpec struct {
	path       string
	defaultRef string
	named      []string
	sideEffect bool
}

type frontendTextSegment struct {
	start int
	end   int
}

type frontendObjectProperty struct {
	name       string
	start      int
	end        int
	valueStart int
	valueEnd   int
}

type frontendTemplateNode struct {
	tag        string
	key        string
	start      int
	openEnd    int
	closeStart int
	end        int
	selfClose  bool
	children   []*frontendTemplateNode
}

// mergeGeneratedFrontendPage 按生成顺序更新页面功能，并将已有扩展功能稳定保留在后部。
func mergeGeneratedFrontendPage(existing string, candidate string) (string, bool) {
	existingTemplate, existingScript, existingStyles, ok := splitFrontendSFC(existing)
	if !ok {
		return existing, false
	}
	var candidateTemplate string
	var candidateScript string
	candidateTemplate, candidateScript, _, ok = splitFrontendSFC(candidate)
	if !ok {
		return existing, false
	}
	// 表单字段类型发生变化时，旧页面扩展可能继续按旧类型处理字段。
	// 此时保留扩展会把旧接口契约重新拼回生成页面，直接采用最新完整页面更安全。
	if frontendScriptSchemaChanged(existingScript, candidateScript) {
		return strings.TrimRight(candidate, " \t\r\n") + "\n", true
	}
	var mergedTemplate string
	mergedTemplate, ok = mergeFrontendTemplate(existingTemplate, candidateTemplate)
	if !ok {
		return existing, false
	}
	var mergedScript string
	mergedScript, ok = mergeFrontendScript(existingScript, candidateScript)
	if !ok {
		return existing, false
	}
	merged := replaceFrontendSFCSection(candidate, "template", mergedTemplate)
	merged = replaceFrontendSFCSection(merged, "script", mergedScript)
	if existingStyles != "" && !strings.Contains(merged, existingStyles) {
		merged = strings.TrimRight(merged, "\r\n") + "\n\n" + strings.TrimSpace(existingStyles) + "\n"
	}
	return strings.TrimRight(merged, " \t\r\n") + "\n", true
}

// frontendScriptSchemaChanged 判断已有页面扩展是否仍按旧字段类型处理生成表单字段。
func frontendScriptSchemaChanged(existing string, candidate string) bool {
	fieldKinds := frontendFormFieldKinds(candidate)
	for name, kind := range fieldKinds {
		if kind == "scalar" && frontendFieldUsesArray(existing, name) {
			return true
		}
	}
	return false
}

// frontendFormFieldKinds 提取生成表单默认值对应的字段类型。
func frontendFormFieldKinds(script string) map[string]string {
	fieldKinds := make(map[string]string)
	formIndex := strings.Index(script, "const formData")
	if formIndex < 0 {
		return fieldKinds
	}
	objectStart := strings.IndexByte(script[formIndex:], '{')
	if objectStart < 0 {
		return fieldKinds
	}
	objectStart += formIndex
	objectEnd := findFrontendMatchingDelimiter(script, objectStart, '{', '}')
	if objectEnd < 0 {
		return fieldKinds
	}
	properties, ok := frontendObjectProperties(script[objectStart : objectEnd+1])
	if !ok {
		return fieldKinds
	}
	for _, property := range properties {
		value := strings.TrimSpace(script[objectStart+property.valueStart : objectStart+property.valueEnd])
		kind := "scalar"
		if strings.HasPrefix(value, "[") {
			kind = "array"
		} else if strings.HasPrefix(value, "{") {
			kind = "object"
		}
		fieldKinds[property.name] = kind
	}
	return fieldKinds
}

// frontendFieldUsesArray 判断已有页面是否使用数组专属写法处理字段。
func frontendFieldUsesArray(content string, name string) bool {
	quotedName := regexp.QuoteMeta(name)
	patterns := []string{
		`(?m)\b` + quotedName + `\s*:\s*\[`,
		`(?m)\[\s*\.\.\.[^;\n]*\b` + quotedName + `\b`,
		`(?m)\b` + quotedName + `\b[^;\n]*(?:\.(?:filter|map|join|length)\s*\(|\?\?\s*\[)`,
		`(?m)\b` + quotedName + `\b[^;\n]*\bas\s+[A-Za-z_$][A-Za-z0-9_$]*\[\]`,
	}
	for _, pattern := range patterns {
		if regexp.MustCompile(pattern).MatchString(content) {
			return true
		}
	}
	return false
}

// splitFrontendSFC 拆分 Vue 单文件组件中的模板、脚本和样式区域。
func splitFrontendSFC(content string) (string, string, string, bool) {
	template, _, _, ok := frontendSFCSection(content, "template")
	if !ok {
		return "", "", "", false
	}
	var script string
	var scriptEnd int
	script, _, scriptEnd, ok = frontendSFCSection(content, "script")
	if !ok {
		return "", "", "", false
	}
	styles := strings.TrimSpace(content[scriptEnd:])
	return template, script, styles, true
}

// frontendSFCSection 返回指定 Vue 顶层标签的内部内容和边界。
func frontendSFCSection(content string, tag string) (string, int, int, bool) {
	openStart := strings.Index(content, "<"+tag)
	if openStart < 0 {
		return "", -1, -1, false
	}
	openEnd := strings.Index(content[openStart:], ">")
	if openEnd < 0 {
		return "", -1, -1, false
	}
	openEnd += openStart + 1
	closeMarker := "</" + tag + ">"
	closeStart := -1
	if tag == "template" {
		searchEnd := len(content)
		if scriptStart := strings.Index(content[openEnd:], "<script"); scriptStart >= 0 {
			searchEnd = openEnd + scriptStart
		}
		closeStart = strings.LastIndex(content[openEnd:searchEnd], closeMarker)
	} else {
		closeStart = strings.Index(content[openEnd:], closeMarker)
	}
	if closeStart < 0 {
		return "", -1, -1, false
	}
	closeStart += openEnd
	return content[openEnd:closeStart], openEnd, closeStart + len(closeMarker), true
}

// replaceFrontendSFCSection 替换 Vue 顶层标签的内部内容。
func replaceFrontendSFCSection(content string, tag string, section string) string {
	_, start, end, ok := frontendSFCSection(content, tag)
	if !ok {
		return content
	}
	closeMarker := "</" + tag + ">"
	closeStart := end - len(closeMarker)
	return content[:start] + section + content[closeStart:]
}

// mergeFrontendScript 合并 script setup 声明，生成声明在前，已有扩展声明按原顺序追加。
func mergeFrontendScript(existing string, candidate string) (string, bool) {
	existingBlocks, ok := frontendTopLevelBlocks(existing)
	if !ok {
		return existing, false
	}
	var candidateBlocks []frontendSourceBlock
	candidateBlocks, ok = frontendTopLevelBlocks(candidate)
	if !ok {
		return existing, false
	}
	candidateByKey := make(map[string]int, len(candidateBlocks))
	for index, block := range candidateBlocks {
		if block.kind != "import" && block.key != "" {
			candidateByKey[block.key] = index
		}
	}
	for _, block := range existingBlocks {
		index, exists := candidateByKey[block.key]
		if !exists || block.kind == "import" {
			continue
		}
		candidateBlock := candidateBlocks[index]
		if merged, mergedOK := mergeFrontendDeclaration(candidateBlock.content, block.content); mergedOK {
			candidateBlocks[index].content = merged
		}
	}

	imports := mergeFrontendImports(existingBlocks, candidateBlocks)
	candidateDeclarationIndexes := make(map[string]int)
	candidateDeclarations := make([]frontendSourceBlock, 0, len(candidateBlocks))
	candidateDeclarationCount := 0
	for _, block := range candidateBlocks {
		if block.kind == "import" || block.function {
			continue
		}
		candidateDeclarationIndexes[block.key] = candidateDeclarationCount
		candidateDeclarations = append(candidateDeclarations, block)
		candidateDeclarationCount++
	}
	extraDeclarationsByIndex := make(map[int][]string)
	extraDeclarations := make([]struct {
		key            string
		content        string
		insertionIndex int
	}, 0)
	extraFunctions := make([]string, 0)
	seenExtra := make(map[string]struct{})
	for existingIndex, block := range existingBlocks {
		if block.kind == "import" {
			continue
		}
		if _, exists := candidateByKey[block.key]; exists {
			continue
		}
		dedupeKey := block.key + "\x00" + strings.TrimSpace(block.content)
		if _, exists := seenExtra[dedupeKey]; exists {
			continue
		}
		seenExtra[dedupeKey] = struct{}{}
		if block.function {
			extraFunctions = append(extraFunctions, strings.TrimSpace(block.content))
			continue
		}
		insertionIndex := candidateDeclarationCount
		for nextIndex := existingIndex + 1; nextIndex < len(existingBlocks); nextIndex++ {
			nextBlock := existingBlocks[nextIndex]
			if nextBlock.kind == "import" || nextBlock.function {
				continue
			}
			if candidateIndex, exists := candidateDeclarationIndexes[nextBlock.key]; exists {
				insertionIndex = candidateIndex
				break
			}
		}
		if insertionIndex == candidateDeclarationCount {
			for previousIndex := existingIndex - 1; previousIndex >= 0; previousIndex-- {
				previousBlock := existingBlocks[previousIndex]
				if previousBlock.kind == "import" || previousBlock.function {
					continue
				}
				if candidateIndex, exists := candidateDeclarationIndexes[previousBlock.key]; exists {
					insertionIndex = candidateIndex + 1
					break
				}
			}
		}
		extraDeclarations = append(extraDeclarations, struct {
			key            string
			content        string
			insertionIndex int
		}{key: block.key, content: strings.TrimSpace(block.content), insertionIndex: insertionIndex})
	}
	for _, extra := range extraDeclarations {
		insertionIndex := extra.insertionIndex
		if name := frontendDeclarationName(extra.key); name != "" {
			for candidateIndex, candidate := range candidateDeclarations {
				if frontendIdentifierUsed(candidate.content, name) {
					insertionIndex = min(insertionIndex, candidateIndex)
					break
				}
			}
		}
		for candidateIndex, candidate := range candidateDeclarations {
			name := frontendDeclarationName(candidate.key)
			if name != "" && frontendIdentifierUsed(extra.content, name) {
				insertionIndex = max(insertionIndex, candidateIndex+1)
			}
		}
		extraDeclarationsByIndex[insertionIndex] = append(extraDeclarationsByIndex[insertionIndex], extra.content)
	}

	declarations := make([]string, 0, len(candidateBlocks)+len(seenExtra))
	functions := make([]string, 0, len(candidateBlocks)+len(extraFunctions))
	declarationIndex := 0
	for _, block := range candidateBlocks {
		if block.kind == "import" {
			continue
		}
		if block.function {
			functions = append(functions, strings.TrimSpace(block.content))
			continue
		}
		declarations = append(declarations, extraDeclarationsByIndex[declarationIndex]...)
		declarations = append(declarations, strings.TrimSpace(block.content))
		declarationIndex++
	}
	declarations = append(declarations, extraDeclarationsByIndex[declarationIndex]...)
	functions = append(functions, extraFunctions...)
	sections := make([]string, 0, 3)
	if len(imports) > 0 {
		sections = append(sections, strings.Join(imports, "\n"))
	}
	if len(declarations) > 0 {
		sections = append(sections, strings.Join(declarations, "\n\n"))
	}
	if len(functions) > 0 {
		sections = append(sections, strings.Join(functions, "\n\n"))
	}
	return "\n" + strings.Join(sections, "\n\n") + "\n", true
}

// frontendDeclarationName 返回声明键中的变量名，用于检测初始化顺序依赖。
func frontendDeclarationName(key string) string {
	if !strings.HasPrefix(key, "symbol:") {
		return ""
	}
	return strings.TrimPrefix(key, "symbol:")
}

// frontendTopLevelBlocks 提取 TypeScript 顶层语句并保留相邻注释。
func frontendTopLevelBlocks(content string) ([]frontendSourceBlock, bool) {
	segments, ok := frontendStatementSegments(content)
	if !ok {
		return nil, false
	}
	blocks := make([]frontendSourceBlock, 0, len(segments))
	for index, segment := range segments {
		blockContent := strings.TrimSpace(content[segment.start:segment.end])
		if blockContent == "" {
			continue
		}
		kind, key, function := classifyFrontendSourceBlock(blockContent, index)
		blocks = append(blocks, frontendSourceBlock{key: key, content: blockContent, kind: kind, function: function})
	}
	return blocks, true
}

// frontendStatementSegments 按 TypeScript 顶层语句边界切分源码。
func frontendStatementSegments(content string) ([]frontendTextSegment, bool) {
	segments := make([]frontendTextSegment, 0)
	start := -1
	state := newFrontendScanState()
	for index := 0; index < len(content); index++ {
		if start < 0 && !isFrontendSpace(content[index]) {
			start = index
		}
		closedBrace, ok := state.consume(content, index)
		if !ok {
			return nil, false
		}
		if start < 0 || !state.topLevel() {
			continue
		}
		end := -1
		if content[index] == ';' && !state.inLiteralOrComment() {
			end = index + 1
		} else if closedBrace && frontendBlockEndsAfterBrace(content, index+1) {
			end = index + 1
		}
		if end < 0 {
			continue
		}
		segments = append(segments, frontendTextSegment{start: start, end: end})
		start = -1
	}
	if !state.valid() {
		return nil, false
	}
	if start >= 0 && strings.TrimSpace(content[start:]) != "" {
		segments = append(segments, frontendTextSegment{start: start, end: len(content)})
	}
	return segments, true
}

// classifyFrontendSourceBlock 识别 TypeScript 顶层语句的稳定功能键。
func classifyFrontendSourceBlock(content string, index int) (string, string, bool) {
	code := trimFrontendLeadingComments(content)
	if strings.HasPrefix(code, "void ") && strings.HasSuffix(code, "();") {
		return "declaration", "call:" + strings.TrimSuffix(strings.TrimPrefix(code, "void "), "();"), false
	}
	if strings.HasPrefix(code, "import ") {
		return "import", "import:" + frontendImportPath(code), false
	}
	if match := frontendFunctionPattern.FindStringSubmatch(code); len(match) == 2 {
		return "function", "symbol:" + match[1], true
	}
	if match := frontendNamedDeclarationPattern.FindStringSubmatch(code); len(match) == 2 {
		return "declaration", "symbol:" + match[1], false
	}
	if strings.HasPrefix(code, "const {") || strings.HasPrefix(code, "let {") {
		closeIndex := strings.Index(code, "}")
		if closeIndex > 0 {
			return "declaration", "symbol:" + strings.Join(frontendIdentifierList(code[strings.Index(code, "{")+1:closeIndex]), ","), false
		}
	}
	if strings.HasPrefix(code, "defineOptions(") {
		return "declaration", "call:defineOptions", false
	}
	return "raw", "raw:" + strings.TrimSpace(code) + ":" + string(rune(index)), false
}

// trimFrontendLeadingComments 去除用于归属下一语句的前置注释。
func trimFrontendLeadingComments(content string) string {
	content = strings.TrimSpace(content)
	for {
		switch {
		case strings.HasPrefix(content, "//"):
			lineEnd := strings.IndexByte(content, '\n')
			if lineEnd < 0 {
				return ""
			}
			content = strings.TrimSpace(content[lineEnd+1:])
		case strings.HasPrefix(content, "/*"):
			commentEnd := strings.Index(content, "*/")
			if commentEnd < 0 {
				return ""
			}
			content = strings.TrimSpace(content[commentEnd+2:])
		default:
			return content
		}
	}
}

// mergeFrontendDeclaration 合并同名配置数组中的已有扩展项。
func mergeFrontendDeclaration(candidate string, existing string) (string, bool) {
	candidateArrayStart := frontendAssignmentArrayStart(candidate)
	existingArrayStart := frontendAssignmentArrayStart(existing)
	if candidateArrayStart < 0 || existingArrayStart < 0 {
		return candidate, false
	}
	candidateArrayEnd := findFrontendMatchingDelimiter(candidate, candidateArrayStart, '[', ']')
	existingArrayEnd := findFrontendMatchingDelimiter(existing, existingArrayStart, '[', ']')
	if candidateArrayEnd < 0 || existingArrayEnd < 0 {
		return candidate, false
	}
	mergedArray, ok := mergeFrontendArray(candidate[candidateArrayStart:candidateArrayEnd+1], existing[existingArrayStart:existingArrayEnd+1])
	if !ok {
		return candidate, false
	}
	return candidate[:candidateArrayStart] + mergedArray + candidate[candidateArrayEnd+1:], true
}

// frontendAssignmentArrayStart 查找顶层声明赋值的数组字面量。
func frontendAssignmentArrayStart(content string) int {
	assignIndex := strings.Index(content, "=")
	if assignIndex < 0 {
		return -1
	}
	valueStart := frontendNextNonSpace(content, assignIndex+1)
	if valueStart >= 0 && content[valueStart] == '[' {
		return valueStart
	}
	arrowIndex := strings.Index(content[assignIndex+1:], "=>")
	if arrowIndex < 0 {
		return -1
	}
	valueStart = frontendNextNonSpace(content, assignIndex+1+arrowIndex+2)
	if valueStart >= 0 && content[valueStart] == '[' {
		return valueStart
	}
	return -1
}

// mergeFrontendArray 按对象功能键合并配置数组，候选项优先且扩展项向后追加。
func mergeFrontendArray(candidate string, existing string) (string, bool) {
	candidateSegments, ok := frontendDelimitedSegments(candidate, '[', ']')
	if !ok {
		return candidate, false
	}
	var existingSegments []frontendTextSegment
	existingSegments, ok = frontendDelimitedSegments(existing, '[', ']')
	if !ok {
		return candidate, false
	}
	candidateKeys := make(map[string]int, len(candidateSegments))
	items := make([]string, 0, len(candidateSegments)+len(existingSegments))
	changed := false
	for index, segment := range candidateSegments {
		item := strings.TrimSpace(candidate[segment.start:segment.end])
		items = append(items, item)
		candidateKeys[frontendArrayItemKey(item, index)] = index
	}
	for index, segment := range existingSegments {
		item := strings.TrimSpace(existing[segment.start:segment.end])
		key := frontendArrayItemKey(item, index)
		candidateIndex, exists := candidateKeys[key]
		if !exists {
			candidateKeys[key] = len(items)
			items = append(items, item)
			changed = true
			continue
		}
		merged, mergedOK := mergeFrontendObject(items[candidateIndex], item)
		if mergedOK && merged != items[candidateIndex] {
			items[candidateIndex] = merged
			changed = true
		}
	}
	if !changed {
		return candidate, true
	}
	indent := frontendChildIndent(candidate, 0)
	closingIndent := frontendLineIndent(candidate, len(candidate)-1)
	return "[\n" + indent + strings.Join(items, ",\n"+indent) + "\n" + closingIndent + "]", true
}

// frontendArrayItemKey 返回配置数组项的稳定业务键。
func frontendArrayItemKey(item string, index int) string {
	properties, ok := frontendObjectProperties(item)
	if ok {
		for _, name := range []string{"prop", "label", "type", "name"} {
			for _, property := range properties {
				if property.name != name {
					continue
				}
				value := strings.Trim(strings.TrimSpace(item[property.valueStart:property.valueEnd]), `"'`)
				if value != "" {
					return name + ":" + value
				}
			}
		}
	}
	return "item:" + strings.TrimSpace(item) + ":" + string(rune(index))
}

// mergeFrontendObject 合并同一配置对象的扩展属性及嵌套数组。
func mergeFrontendObject(candidate string, existing string) (string, bool) {
	candidateProperties, ok := frontendObjectProperties(candidate)
	if !ok {
		return candidate, false
	}
	var existingProperties []frontendObjectProperty
	existingProperties, ok = frontendObjectProperties(existing)
	if !ok {
		return candidate, false
	}
	candidateByName := make(map[string]frontendObjectProperty, len(candidateProperties))
	for _, property := range candidateProperties {
		candidateByName[property.name] = property
	}
	replacements := make([]frontendReplacement, 0)
	extraProperties := make([]string, 0)
	for _, property := range existingProperties {
		candidateProperty, exists := candidateByName[property.name]
		if !exists {
			// 生成字段已由候选页面接管时，跳过旧页面遗留的函数型渲染器。
			// 字段类型或接口契约发生变化后，旧闭包可能继续按旧类型处理值，导致页面运行时异常。
			if frontendFunctionValue(existing[property.valueStart:property.valueEnd]) && frontendObjectHasProperty(candidateProperties, "prop") {
				continue
			}
			extraProperties = append(extraProperties, strings.TrimSpace(existing[property.start:property.end]))
			continue
		}
		candidateValue := strings.TrimSpace(candidate[candidateProperty.valueStart:candidateProperty.valueEnd])
		existingValue := strings.TrimSpace(existing[property.valueStart:property.valueEnd])
		var merged string
		var mergedOK bool
		switch {
		case strings.HasPrefix(candidateValue, "[") && strings.HasPrefix(existingValue, "["):
			merged, mergedOK = mergeFrontendArray(candidateValue, existingValue)
		case strings.HasPrefix(candidateValue, "{") && strings.HasPrefix(existingValue, "{"):
			merged, mergedOK = mergeFrontendObject(candidateValue, existingValue)
		}
		if mergedOK && merged != candidateValue {
			replacements = append(replacements, frontendReplacement{start: candidateProperty.valueStart, end: candidateProperty.valueEnd, content: merged})
		}
	}
	for index := len(replacements) - 1; index >= 0; index-- {
		replacement := replacements[index]
		candidate = candidate[:replacement.start] + replacement.content + candidate[replacement.end:]
	}
	if len(extraProperties) == 0 {
		return candidate, true
	}
	closeIndex := strings.LastIndex(candidate, "}")
	if closeIndex < 0 {
		return candidate, false
	}
	prefix := strings.TrimRight(candidate[:closeIndex], " \t\r\n")
	if !strings.HasSuffix(prefix, "{") && !strings.HasSuffix(prefix, ",") {
		prefix += ","
	}
	indent := frontendChildIndent(candidate, 0)
	closingIndent := frontendLineIndent(candidate, closeIndex)
	return prefix + "\n" + indent + strings.Join(extraProperties, ",\n"+indent) + "\n" + closingIndent + candidate[closeIndex:], true
}

// frontendObjectHasProperty 判断对象是否包含指定属性。
func frontendObjectHasProperty(properties []frontendObjectProperty, name string) bool {
	for _, property := range properties {
		if property.name == name {
			return true
		}
	}
	return false
}

// frontendFunctionValue 判断前端对象属性是否为函数表达式。
func frontendFunctionValue(value string) bool {
	value = strings.TrimSpace(trimFrontendLeadingComments(value))
	return strings.HasPrefix(value, "function") || strings.HasPrefix(value, "async ") || strings.Contains(value, "=>")
}

type frontendReplacement struct {
	start   int
	end     int
	content string
}

// frontendObjectProperties 提取对象字面量的顶层属性及值边界。
func frontendObjectProperties(content string) ([]frontendObjectProperty, bool) {
	openIndex := strings.Index(content, "{")
	if openIndex < 0 {
		return nil, false
	}
	closeIndex := findFrontendMatchingDelimiter(content, openIndex, '{', '}')
	if closeIndex < 0 {
		return nil, false
	}
	segments, ok := frontendSplitSegments(content, openIndex+1, closeIndex)
	if !ok {
		return nil, false
	}
	properties := make([]frontendObjectProperty, 0, len(segments))
	for _, segment := range segments {
		colonIndex := frontendTopLevelColon(content, segment.start, segment.end)
		if colonIndex < 0 {
			continue
		}
		name := strings.Trim(strings.TrimSpace(trimFrontendLeadingComments(content[segment.start:colonIndex])), `"'`)
		if name == "" {
			continue
		}
		properties = append(properties, frontendObjectProperty{
			name:       name,
			start:      segment.start,
			end:        segment.end,
			valueStart: colonIndex + 1,
			valueEnd:   segment.end,
		})
	}
	return properties, true
}

// frontendDelimitedSegments 返回数组或对象内部逗号分隔项。
func frontendDelimitedSegments(content string, open byte, close byte) ([]frontendTextSegment, bool) {
	openIndex := strings.IndexByte(content, open)
	if openIndex < 0 {
		return nil, false
	}
	closeIndex := findFrontendMatchingDelimiter(content, openIndex, open, close)
	if closeIndex < 0 {
		return nil, false
	}
	return frontendSplitSegments(content, openIndex+1, closeIndex)
}

// frontendSplitSegments 按顶层逗号切分指定源码区间。
func frontendSplitSegments(content string, start int, end int) ([]frontendTextSegment, bool) {
	segments := make([]frontendTextSegment, 0)
	segmentStart := start
	state := newFrontendScanState()
	for index := start; index < end; index++ {
		_, ok := state.consume(content, index)
		if !ok {
			return nil, false
		}
		if content[index] == ',' && state.topLevel() && !state.inLiteralOrComment() {
			if strings.TrimSpace(content[segmentStart:index]) != "" {
				segments = append(segments, frontendTextSegment{start: segmentStart, end: index})
			}
			segmentStart = index + 1
		}
	}
	if !state.valid() {
		return nil, false
	}
	if strings.TrimSpace(content[segmentStart:end]) != "" {
		segments = append(segments, frontendTextSegment{start: segmentStart, end: end})
	}
	return segments, true
}

// frontendTopLevelColon 查找对象属性名后的顶层冒号。
func frontendTopLevelColon(content string, start int, end int) int {
	state := newFrontendScanState()
	for index := start; index < end; index++ {
		_, ok := state.consume(content, index)
		if !ok {
			return -1
		}
		if content[index] == ':' && state.topLevel() && !state.inLiteralOrComment() {
			return index
		}
	}
	return -1
}

// findFrontendMatchingDelimiter 查找忽略字符串和注释后的配对分隔符。
func findFrontendMatchingDelimiter(content string, openIndex int, open byte, close byte) int {
	depth := 0
	state := newFrontendScanState()
	for index := openIndex; index < len(content); index++ {
		_, ok := state.consume(content, index)
		if !ok {
			return -1
		}
		if state.inLiteralOrComment() {
			continue
		}
		switch content[index] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

// mergeFrontendImports 合并候选页与旧页使用的导入标识符。
func mergeFrontendImports(existingBlocks []frontendSourceBlock, candidateBlocks []frontendSourceBlock) []string {
	imports := make([]string, 0)
	importedNames := make(map[string]struct{})
	importedPaths := make(map[string]struct{})
	for _, block := range candidateBlocks {
		if block.kind != "import" {
			continue
		}
		imports = append(imports, strings.TrimSpace(block.content))
		spec := parseFrontendImport(block.content)
		for _, name := range append([]string{spec.defaultRef}, spec.named...) {
			if name != "" {
				importedNames[frontendImportLocalName(name)] = struct{}{}
			}
		}
		importedPaths[spec.path] = struct{}{}
	}
	for _, block := range existingBlocks {
		if block.kind != "import" {
			continue
		}
		spec := parseFrontendImport(block.content)
		if spec.sideEffect {
			if _, exists := importedPaths[spec.path]; !exists {
				imports = append(imports, strings.TrimSpace(block.content))
				importedPaths[spec.path] = struct{}{}
			}
			continue
		}
		missing := frontendImportSpec{path: spec.path}
		if spec.defaultRef != "" {
			name := frontendImportLocalName(spec.defaultRef)
			if _, exists := importedNames[name]; !exists {
				missing.defaultRef = spec.defaultRef
				importedNames[name] = struct{}{}
			}
		}
		for _, named := range spec.named {
			name := frontendImportLocalName(named)
			if _, exists := importedNames[name]; exists {
				continue
			}
			missing.named = append(missing.named, named)
			importedNames[name] = struct{}{}
		}
		if rendered := renderFrontendImport(missing); rendered != "" {
			imports = append(imports, rendered)
			importedPaths[spec.path] = struct{}{}
		}
	}
	return imports
}

// pruneUnusedFrontendTypeImports 移除合并后不再被源码引用的类型导入。
func pruneUnusedFrontendTypeImports(content string) string {
	blocks, ok := frontendTopLevelBlocks(content)
	if !ok {
		return content
	}

	usage := content
	for _, block := range blocks {
		if block.kind == "import" {
			usage = strings.Replace(usage, block.content, "", 1)
		}
	}

	var builder strings.Builder
	lastIndex := 0
	for _, block := range blocks {
		if block.kind != "import" {
			continue
		}
		blockStart := strings.Index(content[lastIndex:], block.content)
		if blockStart < 0 {
			continue
		}
		blockStart += lastIndex
		builder.WriteString(content[lastIndex:blockStart])
		lastIndex = blockStart
		blockEnd := lastIndex + len(block.content)
		builder.WriteString(pruneFrontendImportBlock(block.content, usage))
		lastIndex = blockEnd
	}
	builder.WriteString(content[lastIndex:])
	return builder.String()
}

// pruneFrontendImportBlock 清理单条导入语句中未使用的类型标识符。
func pruneFrontendImportBlock(block string, usage string) string {
	spec := parseFrontendImport(block)
	if spec.sideEffect || len(spec.named) == 0 {
		return block
	}
	kept := make([]string, 0, len(spec.named))
	for _, named := range spec.named {
		name := frontendImportLocalName(named)
		if strings.HasPrefix(strings.TrimSpace(named), "type ") && !frontendIdentifierUsed(usage, name) {
			continue
		}
		kept = append(kept, named)
	}
	if len(kept) == 0 && spec.defaultRef == "" {
		return ""
	}
	spec.named = kept
	rendered := renderFrontendImport(spec)
	trimmed := strings.TrimSpace(trimFrontendLeadingComments(block))
	if trimmed == "" || rendered == "" {
		return rendered
	}
	prefixIndex := strings.Index(block, trimmed)
	if prefixIndex < 0 {
		return rendered
	}
	return block[:prefixIndex] + rendered
}

// frontendIdentifierUsed 判断标识符是否仍在导入语句之外被引用。
func frontendIdentifierUsed(content string, name string) bool {
	if name == "" {
		return false
	}
	return regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(content)
}

// parseFrontendImport 解析常用 default、named 和 type import 形式。
func parseFrontendImport(content string) frontendImportSpec {
	code := strings.TrimSpace(trimFrontendLeadingComments(content))
	path := frontendImportPath(code)
	spec := frontendImportSpec{path: path}
	if path == "" {
		return spec
	}
	if !strings.Contains(code, " from ") {
		spec.sideEffect = true
		return spec
	}
	beforeFrom := strings.TrimSpace(code[len("import "):strings.Index(code, " from ")])
	typeOnly := strings.HasPrefix(beforeFrom, "type ")
	if match := frontendImportNamedPattern.FindStringSubmatch(beforeFrom); len(match) == 2 {
		for _, item := range strings.Split(match[1], ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				if typeOnly && !strings.HasPrefix(item, "type ") {
					item = "type " + item
				}
				spec.named = append(spec.named, item)
			}
		}
		beforeFrom = strings.TrimSpace(beforeFrom[:strings.Index(beforeFrom, "{")])
		beforeFrom = strings.TrimSuffix(beforeFrom, ",")
	}
	beforeFrom = strings.TrimSpace(strings.TrimPrefix(beforeFrom, "type "))
	if beforeFrom != "" {
		spec.defaultRef = beforeFrom
	}
	return spec
}

// renderFrontendImport 渲染仅包含缺失标识符的导入语句。
func renderFrontendImport(spec frontendImportSpec) string {
	if spec.path == "" || spec.defaultRef == "" && len(spec.named) == 0 {
		return ""
	}
	parts := make([]string, 0, 2)
	if spec.defaultRef != "" {
		parts = append(parts, spec.defaultRef)
	}
	if len(spec.named) > 0 {
		parts = append(parts, "{ "+strings.Join(spec.named, ", ")+" }")
	}
	return "import " + strings.Join(parts, ", ") + " from \"" + spec.path + "\";"
}

// frontendImportPath 返回导入语句的模块路径。
func frontendImportPath(content string) string {
	if match := frontendImportPathPattern.FindStringSubmatch(content); len(match) == 2 {
		return match[1]
	}
	if strings.HasPrefix(strings.TrimSpace(content), "import \"") {
		return strings.Trim(strings.TrimSuffix(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(content), "import ")), ";"), `"'`)
	}
	return ""
}

// frontendImportLocalName 返回 import 标识符在当前模块中的名称。
func frontendImportLocalName(spec string) string {
	spec = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(spec), "type "))
	parts := strings.Fields(spec)
	if len(parts) >= 3 && parts[len(parts)-2] == "as" {
		return parts[len(parts)-1]
	}
	return spec
}

// frontendIdentifierList 返回解构声明中的稳定标识符列表。
func frontendIdentifierList(content string) []string {
	list := make([]string, 0)
	for _, item := range strings.Split(content, ",") {
		item = strings.TrimSpace(item)
		if colonIndex := strings.Index(item, ":"); colonIndex >= 0 {
			item = strings.TrimSpace(item[colonIndex+1:])
		}
		if item != "" {
			list = append(list, item)
		}
	}
	return list
}

// mergeFrontendTemplate 递归合并模板节点，候选节点优先，旧页独有节点向后追加。
func mergeFrontendTemplate(existing string, candidate string) (string, bool) {
	existingRoot, ok := parseFrontendTemplate(existing)
	if !ok {
		return existing, false
	}
	var candidateRoot *frontendTemplateNode
	candidateRoot, ok = parseFrontendTemplate(candidate)
	if !ok || candidateRoot.tag != existingRoot.tag {
		return existing, false
	}
	merged := mergeFrontendTemplateNode(candidate, candidateRoot, existing, existingRoot)
	prefix := candidate[:candidateRoot.start]
	suffix := candidate[candidateRoot.end:]
	return prefix + merged + suffix, true
}

// parseFrontendTemplate 解析模板的首个根节点和直接子节点。
func parseFrontendTemplate(content string) (*frontendTemplateNode, bool) {
	nodes := make([]*frontendTemplateNode, 0)
	stack := make([]*frontendTemplateNode, 0)
	for index := 0; index < len(content); {
		openIndex := strings.IndexByte(content[index:], '<')
		if openIndex < 0 {
			break
		}
		openIndex += index
		if strings.HasPrefix(content[openIndex:], "<!--") {
			commentEnd := strings.Index(content[openIndex+4:], "-->")
			if commentEnd < 0 {
				return nil, false
			}
			index = openIndex + 4 + commentEnd + 3
			continue
		}
		closeIndex := frontendTagEnd(content, openIndex)
		if closeIndex < 0 {
			return nil, false
		}
		tagContent := strings.TrimSpace(content[openIndex+1 : closeIndex])
		if tagContent == "" || strings.HasPrefix(tagContent, "!") || strings.HasPrefix(tagContent, "?") {
			index = closeIndex + 1
			continue
		}
		if strings.HasPrefix(tagContent, "/") {
			tag := frontendTagName(strings.TrimSpace(tagContent[1:]))
			if len(stack) == 0 || stack[len(stack)-1].tag != tag {
				return nil, false
			}
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			node.closeStart = openIndex
			node.end = closeIndex + 1
			index = closeIndex + 1
			continue
		}
		selfClose := strings.HasSuffix(tagContent, "/")
		tag := frontendTagName(tagContent)
		if tag == "" {
			return nil, false
		}
		node := &frontendTemplateNode{
			tag:       tag,
			key:       frontendTemplateNodeKey(tag, tagContent),
			start:     openIndex,
			openEnd:   closeIndex + 1,
			selfClose: selfClose || frontendVoidTag(tag),
		}
		if len(stack) > 0 {
			stack[len(stack)-1].children = append(stack[len(stack)-1].children, node)
		} else {
			nodes = append(nodes, node)
		}
		if node.selfClose {
			node.closeStart = closeIndex
			node.end = closeIndex + 1
		} else {
			stack = append(stack, node)
		}
		index = closeIndex + 1
	}
	if len(stack) != 0 || len(nodes) != 1 {
		return nil, false
	}
	return nodes[0], true
}

// mergeFrontendTemplateNode 合并同一模板节点的子节点。
func mergeFrontendTemplateNode(candidateContent string, candidate *frontendTemplateNode, existingContent string, existing *frontendTemplateNode) string {
	if candidate.selfClose || existing.selfClose {
		return candidateContent[candidate.start:candidate.end]
	}
	existingByKey := make(map[string][]*frontendTemplateNode)
	for _, child := range existing.children {
		existingByKey[child.key] = append(existingByKey[child.key], child)
	}
	used := make(map[*frontendTemplateNode]struct{})
	var builder strings.Builder
	builder.WriteString(candidateContent[candidate.start:candidate.openEnd])
	cursor := candidate.openEnd
	for _, child := range candidate.children {
		builder.WriteString(candidateContent[cursor:child.start])
		matches := existingByKey[child.key]
		var existingChild *frontendTemplateNode
		for _, match := range matches {
			if _, exists := used[match]; !exists {
				existingChild = match
				break
			}
		}
		if existingChild != nil && existingChild.tag == child.tag {
			used[existingChild] = struct{}{}
			builder.WriteString(mergeFrontendTemplateNode(candidateContent, child, existingContent, existingChild))
		} else {
			builder.WriteString(candidateContent[child.start:child.end])
		}
		cursor = child.end
	}
	beforeClose := candidateContent[cursor:candidate.closeStart]
	for _, child := range existing.children {
		if _, exists := used[child]; exists {
			continue
		}
		if strings.TrimSpace(beforeClose) == "" && !strings.HasSuffix(builder.String(), "\n") {
			builder.WriteByte('\n')
		}
		builder.WriteString(existingContent[child.start:child.end])
		if !strings.HasSuffix(existingContent[child.start:child.end], "\n") {
			builder.WriteByte('\n')
		}
	}
	builder.WriteString(beforeClose)
	builder.WriteString(candidateContent[candidate.closeStart:candidate.end])
	return builder.String()
}

// frontendTemplateNodeKey 返回模板节点用于顺序合并的稳定键。
func frontendTemplateNodeKey(tag string, tagContent string) string {
	for _, match := range frontendTemplateAttrPattern.FindAllStringSubmatch(tagContent, -1) {
		if len(match) == 3 {
			return tag + ":" + match[1] + ":" + match[2]
		}
	}
	return tag
}

// frontendTagEnd 查找忽略属性字符串后的标签结束位置。
func frontendTagEnd(content string, start int) int {
	quote := byte(0)
	escaped := false
	for index := start + 1; index < len(content); index++ {
		char := content[index]
		if quote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == quote {
				quote = 0
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote = char
			continue
		}
		if char == '>' {
			return index
		}
	}
	return -1
}

// frontendTagName 返回标签文本中的标签名。
func frontendTagName(content string) string {
	content = strings.TrimSpace(strings.TrimSuffix(content, "/"))
	for index, char := range content {
		if char == ' ' || char == '\t' || char == '\r' || char == '\n' {
			return content[:index]
		}
	}
	return content
}

// frontendVoidTag 判断无需结束标签的原生元素。
func frontendVoidTag(tag string) bool {
	switch strings.ToLower(tag) {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

// frontendChildIndent 返回容器下一层的缩进。
func frontendChildIndent(content string, index int) string {
	return frontendLineIndent(content, index) + "  "
}

// frontendLineIndent 返回指定位置所在行的前导空白。
func frontendLineIndent(content string, index int) string {
	if index < 0 {
		return ""
	}
	if index > len(content) {
		index = len(content)
	}
	lineStart := strings.LastIndex(content[:index], "\n") + 1
	line := content[lineStart:index]
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// frontendNextNonSpace 返回指定位置之后的首个非空白字节位置。
func frontendNextNonSpace(content string, index int) int {
	for index < len(content) {
		if !isFrontendSpace(content[index]) {
			return index
		}
		index++
	}
	return -1
}

// frontendBlockEndsAfterBrace 判断顶层右花括号是否结束当前语句。
func frontendBlockEndsAfterBrace(content string, index int) bool {
	for index < len(content) && (content[index] == ' ' || content[index] == '\t' || content[index] == '\r') {
		index++
	}
	if index >= len(content) || content[index] == '\n' {
		return true
	}
	return content[index] == ';'
}

// isFrontendSpace 判断字节是否为空白。
func isFrontendSpace(char byte) bool {
	return char == ' ' || char == '\t' || char == '\r' || char == '\n'
}

type frontendScanState struct {
	braces       int
	brackets     int
	parentheses  int
	quote        byte
	escaped      bool
	lineComment  bool
	blockComment bool
}

// newFrontendScanState 创建 TypeScript 文本扫描状态。
func newFrontendScanState() *frontendScanState {
	return &frontendScanState{}
}

// consume 消费当前位置字符并更新嵌套、字符串和注释状态。
func (s *frontendScanState) consume(content string, index int) (bool, bool) {
	char := content[index]
	next := byte(0)
	if index+1 < len(content) {
		next = content[index+1]
	}
	if s.lineComment {
		if char == '\n' {
			s.lineComment = false
		}
		return false, true
	}
	if s.blockComment {
		if char == '*' && next == '/' {
			s.blockComment = false
		}
		return false, true
	}
	if s.quote != 0 {
		if s.escaped {
			s.escaped = false
			return false, true
		}
		if char == '\\' {
			s.escaped = true
			return false, true
		}
		if char == s.quote {
			s.quote = 0
		}
		return false, true
	}
	if char == '/' && next == '/' {
		s.lineComment = true
		return false, true
	}
	if char == '/' && next == '*' {
		s.blockComment = true
		return false, true
	}
	if char == '\'' || char == '"' || char == '`' {
		s.quote = char
		return false, true
	}
	closedBrace := false
	switch char {
	case '{':
		s.braces++
	case '}':
		s.braces--
		closedBrace = true
	case '[':
		s.brackets++
	case ']':
		s.brackets--
	case '(':
		s.parentheses++
	case ')':
		s.parentheses--
	}
	return closedBrace, s.braces >= 0 && s.brackets >= 0 && s.parentheses >= 0
}

// topLevel 判断当前扫描位置是否位于 TypeScript 顶层。
func (s *frontendScanState) topLevel() bool {
	return s.braces == 0 && s.brackets == 0 && s.parentheses == 0 && !s.inLiteralOrComment()
}

// inLiteralOrComment 判断当前是否位于字符串或注释中。
func (s *frontendScanState) inLiteralOrComment() bool {
	return s.quote != 0 || s.lineComment || s.blockComment
}

// valid 判断扫描结束后的语法状态是否闭合。
func (s *frontendScanState) valid() bool {
	return s.braces == 0 && s.brackets == 0 && s.parentheses == 0 && s.quote == 0 && !s.blockComment
}
