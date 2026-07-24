package codegen

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
)

// BatchGenerationInput 描述批量生成中单个表的只读生成快照。
type BatchGenerationInput struct {
	// Table 是单个表的生成配置快照。
	Table *Table
	// Columns 是表字段和生成选项快照。
	Columns []*CodeGenColumn
	// Methods 是已保存的接口生成选项快照。
	Methods []*Proto
	// TableComment 是数据库表注释。
	TableComment string
}

// BatchFileRef 标识单个表在批次文件中的来源步骤。
type BatchFileRef struct {
	// TableID 是来源表配置 ID。
	TableID int64
	// FileIndex 是来源表预览文件索引。
	FileIndex int
}

// BatchFile 保存批次合并后需要原子写入的单个文件。
type BatchFile struct {
	// Path 是仓库相对文件路径。
	Path string
	// Content 是批次合并后的最终文件内容。
	Content string
	// Action 是最终原子写入动作。
	Action string
	// Refs 是参与本文件合并的表级进度步骤。
	Refs []BatchFileRef
}

// BatchGeneration 汇总批量生成的单表预览和最终文件写入计划。
type BatchGeneration struct {
	// Generations 是每张表在合并视图中的预览结果。
	Generations []*Generation
	// Files 是需要实际写入的去重文件计划。
	Files []*BatchFile
}

// PrepareBatchGeneration 在内存中预检并合并一批生成内容，不会写入工作区。
func PrepareBatchGeneration(inputs []BatchGenerationInput) (*BatchGeneration, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("批量生成对象不能为空")
	}
	for index, input := range inputs {
		if input.Table == nil {
			return nil, fmt.Errorf("第%d个生成对象为空", index+1)
		}
	}
	orderedInputs := slices.Clone(inputs)
	slices.SortFunc(orderedInputs, func(a BatchGenerationInput, b BatchGenerationInput) int {
		if a.Table.ID < b.Table.ID {
			return -1
		}
		if a.Table.ID > b.Table.ID {
			return 1
		}
		return strings.Compare(a.Table.TableName_, b.Table.TableName_)
	})
	if err := validateBatchTableIDs(orderedInputs); err != nil {
		return nil, err
	}

	initialGenerations := make([]*Generation, 0, len(orderedInputs))
	for _, input := range orderedInputs {
		generation, err := PrepareGeneration(input.Table, input.Columns, input.Methods, nil, input.TableComment)
		if err != nil {
			return nil, err
		}
		initialGenerations = append(initialGenerations, generation)
	}
	if err := validateBatchMergeableFrontendPages(initialGenerations); err != nil {
		return nil, err
	}
	if err := validateBatchMethodConflicts(orderedInputs, initialGenerations); err != nil {
		return nil, err
	}
	if err := validateBatchFileConflicts(orderedInputs, initialGenerations); err != nil {
		return nil, err
	}

	overlay := &batchFileOverlay{contents: make(map[string]string)}
	batch := &BatchGeneration{Generations: make([]*Generation, 0, len(orderedInputs))}
	filesByPath := make(map[string]*BatchFile)
	for _, input := range orderedInputs {
		generation, err := prepareGenerationWithRenderer(input.Table, input.Columns, input.Methods, nil, &renderer{
			tableComment: input.TableComment,
			readFile:     overlay.readFile,
		})
		if err != nil {
			return nil, err
		}
		batch.Generations = append(batch.Generations, generation)
		for fileIndex, file := range generation.Files {
			if file.GetAction() != "create" && file.GetAction() != "update" {
				continue
			}
			currentContent, readErr := overlay.readFile(file.GetPath())
			// 目标内容已被本批前序表合并时，当前表只保留跳过进度，不再重复写入同一文件。
			if readErr == nil && string(currentContent) == file.GetContent() {
				file.Action = "skip"
				file.Exists = true
				file.Message = "已与本批相同文件合并，无需重复写入"
				continue
			}
			// 非“文件不存在”的读取错误会使合并基准不可信，必须终止预检。
			if readErr != nil && !os.IsNotExist(readErr) {
				return nil, readErr
			}
			batchFile := filesByPath[file.GetPath()]
			if batchFile == nil {
				exists, existsErr := overlay.fileExists(file.GetPath())
				if existsErr != nil {
					return nil, existsErr
				}
				action := "create"
				if exists {
					action = "update"
				}
				batchFile = &BatchFile{Path: file.GetPath(), Action: action}
				filesByPath[file.GetPath()] = batchFile
			}
			batchFile.Content = file.GetContent()
			batchFile.Refs = append(batchFile.Refs, BatchFileRef{TableID: generation.Table.ID, FileIndex: fileIndex})
			overlay.contents[file.GetPath()] = file.GetContent()
		}
	}
	batch.Files = make([]*BatchFile, 0, len(filesByPath))
	for _, file := range filesByPath {
		batch.Files = append(batch.Files, file)
	}
	slices.SortFunc(batch.Files, func(a *BatchFile, b *BatchFile) int {
		return strings.Compare(a.Path, b.Path)
	})
	return batch, nil
}

// validateBatchMergeableFrontendPages 阻止无法安全解析的页面对应模块被部分覆盖。
func validateBatchMergeableFrontendPages(generations []*Generation) error {
	for _, generation := range generations {
		if generation.Table == nil || generation.Table.GenFrontend != 1 || generation.OutputPaths == nil {
			continue
		}
		pagePath := generation.OutputPaths.GetFrontendPageFilePath()
		for _, file := range generation.Files {
			if file.GetPath() != pagePath || file.GetMessage() != unmergeableFrontendPageMessage {
				continue
			}
			return fmt.Errorf("表%s的前端页面%s无法安全解析，已取消整个模块生成，避免部分覆盖", generation.Table.TableName_, pagePath)
		}
	}
	return nil
}

type batchDefinition struct {
	tableName   string
	fingerprint string
}

type batchFileOverlay struct {
	contents map[string]string
}

var protoHTTPRoutePattern = regexp.MustCompile(`(?m)^\s*(get|post|put|delete):\s*"([^"]+)"`)

// GenerationForTable 返回指定表在批次中的生成结果。
func (b *BatchGeneration) GenerationForTable(tableID int64) *Generation {
	for _, generation := range b.Generations {
		if generation.Table != nil && generation.Table.ID == tableID {
			return generation
		}
	}
	return nil
}

// validateGeneratedProtoHTTPRoutes 拒绝向已有 Proto 写入与其他 RPC 冲突的 HTTP 路由。
func (c *renderer) validateGeneratedProtoHTTPRoutes(table *Table, methods []*Proto) error {
	for _, method := range methods {
		targetEntity := DefaultString(method.TargetEntityName, table.EntityName)
		exists, _ := c.protoMethodExists(method.ProtoFilePath, targetEntity, method.MethodName)
		// 已有同名 RPC 会在完整渲染或增量补丁中复用，无需重复判定路由。
		if exists {
			continue
		}
		rpcContent := c.renderProtoRPC(table, method, resourcePathByEntity(targetEntity))
		route := protoHTTPRoutePattern.FindStringSubmatch(rpcContent)
		// 未声明 HTTP 映射的 RPC 不参与 HTTP 路由冲突检查。
		if len(route) != 3 {
			continue
		}
		content, err := c.readRepoFile(method.ProtoFilePath)
		// 文件尚不存在时由本次生成创建，不会与历史路由冲突。
		if err != nil {
			continue
		}
		for _, existingRoute := range protoHTTPRoutePattern.FindAllStringSubmatch(string(content), -1) {
			// 同一 HTTP 方法与路径只能属于一个 RPC，阻止产生不可访问的新接口。
			if len(existingRoute) == 3 && existingRoute[1] == route[1] && existingRoute[2] == route[2] {
				return fmt.Errorf("Proto文件%s中的HTTP路由%s %s已被其他RPC使用", method.ProtoFilePath, route[1], route[2])
			}
		}
	}
	return nil
}

// batchProtoMethodFingerprint 返回用于比较 Proto 方法、消息和路由的稳定内容指纹。
func (c *renderer) batchProtoMethodFingerprint(table *Table, columns []*CodeGenColumn, method *Proto) string {
	var builder strings.Builder
	targetEntity := DefaultString(method.TargetEntityName, table.EntityName)
	builder.WriteString(normalizeBatchProtoDefinition(c.renderProtoRPC(table, method, resourcePathByEntity(targetEntity))))
	// 选项接口的触发来源和外键字段只描述调用方，不会改变目标实体的接口实现。
	// 目标实体及其树形、显示和取值字段才决定可复用的接口定义。
	semanticParts := []string{
		method.APIKind,
		targetEntity,
		method.ParentColumn,
		method.LabelColumn,
		method.ValueColumn,
	}
	// 状态接口的状态字段会改变更新逻辑，必须继续参与冲突判断。
	if method.APIKind == APIKindStatus {
		semanticParts = append(semanticParts, method.Name)
	}
	builder.WriteString("\nsemantic:")
	builder.WriteString(strings.Join(semanticParts, "\x00"))
	for _, messageName := range c.protoMessageNamesForMethod(table, method) {
		builder.WriteString("\n")
		builder.WriteString(normalizeBatchProtoDefinition(c.renderProtoMessageByName(table, columns, method, messageName)))
	}
	return builder.String()
}

// fileExists 判断批次虚拟文件视图或工作区中是否存在目标文件。
func (o *batchFileOverlay) fileExists(path string) (bool, error) {
	_, err := o.readFile(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// readFile 从批次虚拟文件视图读取内容，未覆盖的路径回退到工作区。
func (o *batchFileOverlay) readFile(path string) ([]byte, error) {
	if content, exists := o.contents[path]; exists {
		return []byte(content), nil
	}
	fullPath, err := SafeRepoFilePath(path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(fullPath)
}

// validateBatchTableIDs 校验批次内不会重复引用同一生成配置。
func validateBatchTableIDs(inputs []BatchGenerationInput) error {
	seen := make(map[int64]struct{}, len(inputs))
	for _, input := range inputs {
		if input.Table.ID <= 0 {
			return fmt.Errorf("代码生成表配置ID不能为空")
		}
		if _, exists := seen[input.Table.ID]; exists {
			return fmt.Errorf("代码生成表配置ID不能重复")
		}
		seen[input.Table.ID] = struct{}{}
	}
	return nil
}

// validateBatchMethodConflicts 拒绝同一批次内定义不同签名、消息或路由的重复接口。
func validateBatchMethodConflicts(inputs []BatchGenerationInput, generations []*Generation) error {
	methods := make(map[string]batchDefinition)
	messages := make(map[string]batchDefinition)
	routes := make(map[string]batchDefinition)
	for index, generation := range generations {
		input := inputs[index]
		renderer := &renderer{tableComment: input.TableComment}
		for _, method := range generation.GeneratedMethods {
			targetEntity := DefaultString(method.TargetEntityName, generation.Table.EntityName)
			exists, _ := renderer.protoMethodExists(method.ProtoFilePath, targetEntity, method.MethodName)
			if exists {
				continue
			}
			fingerprint := renderer.batchProtoMethodFingerprint(generation.Table, input.Columns, method)
			definition := batchDefinition{tableName: generation.Table.TableName_, fingerprint: fingerprint}
			methodKey := method.ProtoFilePath + ":" + targetEntity + "Service:" + method.MethodName
			if err := compareBatchDefinition(methods, methodKey, definition, "方法"); err != nil {
				return err
			}
			for _, messageName := range renderer.protoMessageNamesForMethod(generation.Table, method) {
				messageKey := method.ProtoFilePath + ":message:" + messageName
				messageDefinition := batchDefinition{
					tableName:   generation.Table.TableName_,
					fingerprint: normalizeBatchProtoDefinition(renderer.renderProtoMessageByName(generation.Table, input.Columns, method, messageName)),
				}
				if err := compareBatchDefinition(messages, messageKey, messageDefinition, "消息"); err != nil {
					return err
				}
			}
			for _, match := range protoHTTPRoutePattern.FindAllStringSubmatch(renderer.renderProtoRPC(generation.Table, method, resourcePathByEntity(targetEntity)), -1) {
				if len(match) < 3 {
					continue
				}
				routeKey := method.ProtoFilePath + ":" + match[1] + ":" + match[2]
				if err := compareBatchDefinition(routes, routeKey, definition, "HTTP路由"); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// validateBatchFileConflicts 拒绝无法通过增量补丁合并的同文件写入。
func validateBatchFileConflicts(inputs []BatchGenerationInput, generations []*Generation) error {
	files := make(map[string]batchDefinition)
	for index, generation := range generations {
		for _, file := range generation.Files {
			if file.GetAction() != "create" && file.GetAction() != "update" || isBatchMergeableFile(file.GetPath()) {
				continue
			}
			definition := batchDefinition{tableName: inputs[index].Table.TableName_, fingerprint: file.GetContent()}
			if err := compareBatchDefinition(files, file.GetPath(), definition, "不可合并文件"); err != nil {
				return err
			}
		}
	}
	return nil
}

// compareBatchDefinition 去重完全相同的定义，并返回不同定义之间的冲突信息。
func compareBatchDefinition(definitions map[string]batchDefinition, key string, current batchDefinition, kind string) error {
	previous, exists := definitions[key]
	if !exists {
		definitions[key] = current
		return nil
	}
	if previous.fingerprint == current.fingerprint {
		return nil
	}
	return fmt.Errorf("批量生成冲突：表%s与表%s在%s中定义了不同的%s", previous.tableName, current.tableName, key, kind)
}

// normalizeBatchProtoDefinition 移除不影响接口语义的独立 Proto 注释行。
func normalizeBatchProtoDefinition(content string) string {
	lines := strings.Split(content, "\n")
	var builder strings.Builder
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		builder.WriteString(trimmed)
		builder.WriteByte('\n')
	}
	return strings.TrimSpace(builder.String())
}

// isBatchMergeableFile 判断同一文件是否支持按已有增量补丁规则合并。
func isBatchMergeableFile(path string) bool {
	if path == generatedMenuSQLPath {
		return true
	}
	if strings.HasSuffix(path, ".proto") {
		return true
	}
	return strings.HasPrefix(path, "backend/service/") && (strings.HasSuffix(path, ".go") || strings.HasSuffix(path, ".ts")) ||
		strings.HasPrefix(path, "frontend/admin/src/api/") && strings.HasSuffix(path, ".ts") ||
		strings.HasPrefix(path, "frontend/admin/src/views/") && strings.HasSuffix(path, ".vue") ||
		strings.HasPrefix(path, "backend/server/") && strings.HasSuffix(path, ".go")
}
