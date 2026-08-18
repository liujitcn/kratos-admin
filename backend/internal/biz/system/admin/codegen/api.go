package codegen

import (
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

// Generation 汇总一次预览或写入使用的完整生成结果。
type Generation struct {
	Table            *Table                              // 应用本次输出路径后的生成对象
	GeneratedMethods []*Proto                            // 本次实际参与生成的方法
	OutputPaths      *systemadminv1.CodeGenOutputPaths   // 校验后的输出路径
	Files            []*systemadminv1.CodeGenPreviewFile // 已完成增量合并的预览文件
}

// PrepareGeneration 准备一次预览或写入所需的全部纯生成内容。
func PrepareGeneration(
	table *Table,
	columns []*CodeGenColumn,
	methods []*Proto,
	requestedPaths *systemadminv1.CodeGenOutputPaths,
	tableComment string,
	localeState LocaleState,
) (*Generation, error) {
	return PrepareGenerationWithMigrationVersion(table, columns, methods, requestedPaths, tableComment, "", localeState)
}

// PrepareGenerationWithMigrationVersion 按数据库迁移版本准备代码生成内容。
func PrepareGenerationWithMigrationVersion(
	table *Table,
	columns []*CodeGenColumn,
	methods []*Proto,
	requestedPaths *systemadminv1.CodeGenOutputPaths,
	tableComment string,
	migrationVersion string,
	localeState LocaleState,
) (*Generation, error) {
	return prepareGenerationWithRenderer(table, columns, methods, requestedPaths, localeState, &renderer{
		tableComment:     tableComment,
		migrationVersion: migrationVersion,
	})
}

// prepareGenerationWithRenderer 使用指定文件视图准备单次预览或批次内合并结果。
func prepareGenerationWithRenderer(
	table *Table,
	columns []*CodeGenColumn,
	methods []*Proto,
	requestedPaths *systemadminv1.CodeGenOutputPaths,
	localeState LocaleState,
	renderer *renderer,
) (*Generation, error) {
	outputPaths, err := renderer.resolveCodeGenOutputPaths(table, requestedPaths)
	if err != nil {
		return nil, err
	}
	generationTable, generationMethods := renderer.applyCodeGenOutputPaths(table, methods, outputPaths)
	generatedMethods := renderer.generatedProtoMethods(generationTable, columns, generationMethods)
	// 写入前校验 HTTP 映射，避免生成后才暴露为路由冲突。
	if err = renderer.validateGeneratedProtoHTTPRoutes(generationTable, generatedMethods); err != nil {
		return nil, err
	}
	return &Generation{
		Table:            generationTable,
		GeneratedMethods: generatedMethods,
		OutputPaths:      outputPaths,
		Files:            renderer.buildPreviewFiles(generationTable, columns, generationMethods, outputPaths, localeState),
	}, nil
}

// applySavedProtoMethod 使用已保存的 Proto 配置覆盖模板推导值。
func applySavedProtoMethod(check *ProtoCheck, saved *Proto) {
	check.GenerateWhenMissing = saved.GenerateWhenMissing == 1
	check.TargetBusinessName = saved.TargetBusinessName
	if saved.ParentColumn != "" {
		check.ParentColumn = saved.ParentColumn
	}
	if saved.LabelColumn != "" {
		check.LabelColumn = saved.LabelColumn
	}
	if saved.ValueColumn != "" {
		check.ValueColumn = saved.ValueColumn
	}
	check.OptionStatusColumn = saved.OptionStatusColumn
	check.OptionStatusEnabledValue = saved.OptionStatusEnabledValue
	check.Lazy = saved.Lazy
	if saved.APIKind == APIKindStatus && saved.Name != "" {
		check.Name = saved.Name
	}
}
