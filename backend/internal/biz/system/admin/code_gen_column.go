package biz

import (
	"context"
	"math"
	"regexp"
	"slices"
	"strings"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

var codeGenDatabaseTableNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// CodeGenColumnCase 管理代码生成字段元数据与生成配置。
type CodeGenColumnCase struct {
	*data.CodeGenColumnRepository
	dbClient         *databaseGorm.Client
	tx               data.Transaction
	baseDictRepo     *data.BaseDictRepository
	codeGenTableRepo *data.CodeGenTableRepository
	mapper           *mapper.CopierMapper[systemadminv1.CodeGenColumn, models.CodeGenColumn]
}

// codeGenColumnDefaultSource 描述字段默认选项的数据来源。
type codeGenColumnDefaultSource struct {
	dictionaryCode string
	tableName      string
}

// NewCodeGenColumnCase 创建代码生成字段业务实例。
func NewCodeGenColumnCase(
	codeGenColumnRepo *data.CodeGenColumnRepository,
	dbClient *databaseGorm.Client,
	tx data.Transaction,
	baseDictRepo *data.BaseDictRepository,
	codeGenTableRepo *data.CodeGenTableRepository,
) *CodeGenColumnCase {
	columnMapper := mapper.NewCopierMapper[systemadminv1.CodeGenColumn, models.CodeGenColumn]()
	columnMapper.AppendConverters(mapper.NewJSONTypeConverter[*systemadminv1.CodeGenColumnQueryConfig]().NewConverterPair())
	columnMapper.AppendConverters(mapper.NewJSONTypeConverter[*systemadminv1.CodeGenColumnListConfig]().NewConverterPair())
	columnMapper.AppendConverters(mapper.NewJSONTypeConverter[*systemadminv1.CodeGenColumnFormConfig]().NewConverterPair())
	columnMapper.AppendConverters(mapper.NewJSONTypeConverter[map[string]*systemadminv1.CodeGenLocaleConfig]().NewConverterPair())
	return &CodeGenColumnCase{
		CodeGenColumnRepository: codeGenColumnRepo,
		dbClient:                dbClient,
		tx:                      tx,
		baseDictRepo:            baseDictRepo,
		codeGenTableRepo:        codeGenTableRepo,
		mapper:                  columnMapper,
	}
}

// ListCodeGenColumn 查询允许用户维护的字段配置。
func (c *CodeGenColumnCase) ListCodeGenColumn(ctx context.Context, tableID int64) (*systemadminv1.ListCodeGenColumnResponse, error) {
	columns, err := c.listCodeGenColumns(ctx, tableID)
	if err != nil {
		return nil, err
	}
	return &systemadminv1.ListCodeGenColumnResponse{
		CodeGenColumns: filterConfigurableCodeGenColumns(columns),
	}, nil
}

// ListCodeGenDatabaseColumn 查询指定数据库表的字段元数据。
func (c *CodeGenColumnCase) ListCodeGenDatabaseColumn(ctx context.Context, tableName string) (*systemadminv1.ListCodeGenDatabaseColumnResponse, error) {
	databaseColumns, err := c.listDatabaseColumns(ctx, tableName)
	if err != nil {
		return nil, err
	}
	columns := make([]*systemadminv1.CodeGenDatabaseColumn, 0, len(databaseColumns))
	for _, item := range databaseColumns {
		columnComment := item.Comment
		// 数据库未配置字段注释时回退显示字段名。
		if columnComment == "" {
			columnComment = item.Name
		}
		columnType := item.ColumnType
		// 数据库未返回完整类型时回退到基础数据类型。
		if columnType == "" {
			columnType = item.DataType
		}
		columns = append(columns, &systemadminv1.CodeGenDatabaseColumn{
			Name:       item.Name,
			Comment:    columnComment,
			DbType:     item.DataType,
			ColumnType: columnType,
			IsPrimary:  item.ColumnKey == "PRI",
			IsNullable: item.IsNullable == "YES",
		})
	}
	return &systemadminv1.ListCodeGenDatabaseColumnResponse{Columns: columns}, nil
}

// DeleteByTableIDs 删除多个代码生成表配置关联的字段配置。
func (c *CodeGenColumnCase) DeleteByTableIDs(ctx context.Context, tableIDs []int64) error {
	if len(tableIDs) == 0 {
		return nil
	}
	query := c.Query(ctx).CodeGenColumn
	return c.Delete(ctx, repository.Where(query.TableID.In(tableIDs...)))
}

// SaveCodeGenColumn 按最新数据库字段同步代码生成字段配置。
func (c *CodeGenColumnCase) SaveCodeGenColumn(ctx context.Context, req *systemadminv1.SaveCodeGenColumnRequest) error {
	table, err := c.codeGenTableRepo.FindByID(ctx, req.GetTableId())
	if err != nil {
		return err
	}
	var databaseColumns []dto.CodeGenDatabaseColumn
	databaseColumns, err = c.listDatabaseColumns(ctx, table.Name)
	if err != nil {
		return err
	}
	// 保存字段配置前必须确认目标表仍有可用字段。
	if len(databaseColumns) == 0 {
		return errorsx.ResourceNotFound("数据库表字段不存在")
	}
	var defaultSources map[string]codeGenColumnDefaultSource
	defaultSources, err = c.listCodeGenColumnDefaultSources(ctx, table.Name, databaseColumns)
	if err != nil {
		return err
	}
	requestColumns := make(map[string]*systemadminv1.CodeGenColumn, len(req.GetCodeGenColumns()))
	for _, column := range req.GetCodeGenColumns() {
		if column == nil || column.GetName() == "" {
			return errorsx.InvalidArgument("字段名不能为空")
		}
		// 同一个字段只能保存一份配置，避免覆盖顺序影响最终结果。
		if _, exists := requestColumns[column.GetName()]; exists {
			return errorsx.InvalidArgument("字段" + column.GetName() + "配置重复")
		}
		requestColumns[column.GetName()] = column
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.Query(ctx).CodeGenColumn
		opts := make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(query.TableID.Eq(req.GetTableId())))
		opts = append(opts, repository.Order(query.ID.Asc()))
		var savedColumns []*models.CodeGenColumn
		savedColumns, err = c.List(ctx, opts...)
		if err != nil {
			return err
		}
		savedByName := make(map[string]*models.CodeGenColumn, len(savedColumns))
		deleteIDs := make([]int64, 0)
		for _, saved := range savedColumns {
			// 历史重复配置只保留最早记录，其余记录在本轮同步中删除。
			if _, exists := savedByName[saved.Name]; exists {
				deleteIDs = append(deleteIDs, saved.ID)
				continue
			}
			savedByName[saved.Name] = saved
		}
		for index, databaseColumn := range databaseColumns {
			saved := savedByName[databaseColumn.Name]
			column := requestColumns[databaseColumn.Name]
			// 前端不展示的系统字段沿用已有配置；新增字段才使用元数据默认值。
			if column == nil {
				column = newDefaultCodeGenColumn(req.GetTableId(), databaseColumn, int32(index+1), defaultSources[databaseColumn.Name])
				c.mergeSavedCodeGenColumn(column, saved)
			}
			normalizeCodeGenColumnConfig(column, databaseColumn, defaultSources[databaseColumn.Name])
			if err = validateCodeGenColumnConfig(column, databaseColumn); err != nil {
				return err
			}
			item := c.mapper.ToEntity(column)
			item.TableID = req.GetTableId()
			item.Name = databaseColumn.Name
			// 新字段或历史零排序按数据库字段位置初始化，已有字段沿用页面提交或数据库保存的排序。
			if item.Sort <= 0 {
				item.Sort = int32(index + 1)
			}
			if saved == nil {
				if err = c.Create(ctx, item); err != nil {
					return err
				}
			} else {
				item.ID = saved.ID
				_, err = query.WithContext(ctx).Where(query.ID.Eq(item.ID)).UpdateSimple(
					query.Comment.Value(item.Comment),
					query.QueryConfig.Value(item.QueryConfig),
					query.ListConfig.Value(item.ListConfig),
					query.FormConfig.Value(item.FormConfig),
					query.Sort.Value(item.Sort),
					query.I18NConfig.Value(item.I18NConfig),
				)
				if err != nil {
					return err
				}
			}
			delete(savedByName, databaseColumn.Name)
			delete(requestColumns, databaseColumn.Name)
		}
		// 请求中出现非数据库字段时拒绝保存，避免写入失效配置。
		for columnName := range requestColumns {
			return errorsx.InvalidArgument("字段" + columnName + "不存在")
		}
		// 数据库已删除的字段配置以及历史重复配置同步删除。
		for _, saved := range savedByName {
			deleteIDs = append(deleteIDs, saved.ID)
		}
		return c.DeleteByIDs(ctx, deleteIDs)
	})
}

// listCodeGenColumns 查询数据库字段与已保存生成配置的完整合并结果。
func (c *CodeGenColumnCase) listCodeGenColumns(ctx context.Context, tableID int64) ([]*systemadminv1.CodeGenColumn, error) {
	if tableID <= 0 {
		return nil, errorsx.InvalidArgument("代码生成表配置ID不能为空")
	}
	table, err := c.codeGenTableRepo.FindByID(ctx, tableID)
	if err != nil {
		return nil, err
	}
	var databaseColumns []dto.CodeGenDatabaseColumn
	databaseColumns, err = c.listDatabaseColumns(ctx, table.Name)
	if err != nil {
		return nil, err
	}
	var defaultSources map[string]codeGenColumnDefaultSource
	defaultSources, err = c.listCodeGenColumnDefaultSources(ctx, table.Name, databaseColumns)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).CodeGenColumn
	opts := make([]repository.QueryOption, 0, 3)
	opts = append(opts, repository.Where(query.TableID.Eq(tableID)))
	opts = append(opts, repository.Order(query.Sort.Asc()))
	opts = append(opts, repository.Order(query.ID.Asc()))
	var savedColumns []*models.CodeGenColumn
	savedColumns, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return c.mergeCodeGenColumns(tableID, databaseColumns, savedColumns, defaultSources), nil
}

// listDatabaseColumns 查询数据库字段生成所需的完整元数据。
func (c *CodeGenColumnCase) listDatabaseColumns(ctx context.Context, tableName string) ([]dto.CodeGenDatabaseColumn, error) {
	if tableName == "" {
		return nil, errorsx.InvalidArgument("数据库表名不能为空")
	}
	if !codeGenDatabaseTableNamePattern.MatchString(tableName) {
		return nil, errorsx.InvalidArgument("数据库表名格式不正确")
	}
	var columns []dto.CodeGenDatabaseColumn
	// information_schema 没有业务生成模型，表名经白名单校验后使用参数化查询读取字段元数据。
	err := c.dbClient.DB.WithContext(ctx).
		Table("information_schema.columns").
		Select("column_name, column_comment, data_type, column_type, column_key, is_nullable, extra, ordinal_position, character_maximum_length, numeric_precision, numeric_scale, column_default").
		Where("table_schema = DATABASE()").
		Where("table_name = ?", tableName).
		Order("ordinal_position").
		Find(&columns).Error
	return columns, err
}

// listCodeGenColumnDefaultSources 查询字段默认选项对应的字典或数据表来源。
func (c *CodeGenColumnCase) listCodeGenColumnDefaultSources(ctx context.Context, tableName string, databaseColumns []dto.CodeGenDatabaseColumn) (map[string]codeGenColumnDefaultSource, error) {
	defaultSources := make(map[string]codeGenColumnDefaultSource, len(databaseColumns))
	if len(databaseColumns) == 0 {
		return defaultSources, nil
	}
	candidateCodes := make([]string, 0, len(databaseColumns))
	seenCodes := make(map[string]struct{}, len(databaseColumns))
	for _, column := range databaseColumns {
		candidateCode := strings.ToLower(tableName + "_" + column.Name)
		if _, exists := seenCodes[candidateCode]; exists {
			continue
		}
		seenCodes[candidateCode] = struct{}{}
		candidateCodes = append(candidateCodes, candidateCode)
	}
	dictQuery := c.baseDictRepo.Query(ctx).BaseDict
	dictOpts := make([]repository.QueryOption, 0, 1)
	dictOpts = append(dictOpts, repository.Where(dictQuery.Code.In(candidateCodes...)))
	var dictionaries []*models.BaseDict
	var err error
	dictionaries, err = c.baseDictRepo.List(ctx, dictOpts...)
	if err != nil {
		return nil, err
	}
	dictionaryCodes := make(map[string]struct{}, len(dictionaries))
	for _, dictionary := range dictionaries {
		if dictionary != nil {
			dictionaryCodes[strings.ToLower(dictionary.Code)] = struct{}{}
		}
	}
	needsTableMatch := false
	for _, column := range databaseColumns {
		candidateCode := strings.ToLower(tableName + "_" + column.Name)
		if _, exists := dictionaryCodes[candidateCode]; exists {
			defaultSources[column.Name] = codeGenColumnDefaultSource{dictionaryCode: candidateCode}
			continue
		}
		if strings.HasSuffix(strings.ToLower(column.Name), "_id") {
			needsTableMatch = true
		}
	}
	if !needsTableMatch {
		return defaultSources, nil
	}
	tableInfos, err := c.listCodeGenOptionTables(ctx)
	if err != nil {
		return nil, err
	}
	for _, column := range databaseColumns {
		if _, exists := defaultSources[column.Name]; exists {
			continue
		}
		matchedTableName := findCodeGenOptionTable(strings.ToLower(column.Name), tableInfos)
		if matchedTableName != "" {
			defaultSources[column.Name] = codeGenColumnDefaultSource{tableName: matchedTableName}
		}
	}
	return defaultSources, nil
}

// listCodeGenOptionTables 查询用于匹配字段关联关系的数据库表名。
func (c *CodeGenColumnCase) listCodeGenOptionTables(ctx context.Context) ([]dto.CodeGenDatabaseTable, error) {
	var tableInfos []dto.CodeGenDatabaseTable
	err := c.dbClient.DB.WithContext(ctx).
		Table("information_schema.tables").
		Select("table_name, table_comment").
		Where("table_schema = DATABASE()").
		Where("table_type = ?", "BASE TABLE").
		Order("table_name").
		Find(&tableInfos).Error
	return tableInfos, err
}

// mergeCodeGenColumns 合并数据库字段元数据与已保存配置。
func (c *CodeGenColumnCase) mergeCodeGenColumns(tableID int64, databaseColumns []dto.CodeGenDatabaseColumn, savedColumns []*models.CodeGenColumn, defaultSources map[string]codeGenColumnDefaultSource) []*systemadminv1.CodeGenColumn {
	savedByName := make(map[string]*models.CodeGenColumn, len(savedColumns))
	for _, column := range savedColumns {
		savedByName[column.Name] = column
	}
	columns := make([]*systemadminv1.CodeGenColumn, 0, len(databaseColumns))
	for index, databaseColumn := range databaseColumns {
		column := newDefaultCodeGenColumn(tableID, databaseColumn, int32(index+1), defaultSources[databaseColumn.Name])
		// 已保存配置覆盖可编辑部分和排序，数据库字段属性始终以实时元数据为准。
		c.mergeSavedCodeGenColumn(column, savedByName[databaseColumn.Name])
		normalizeCodeGenColumnConfig(column, databaseColumn, defaultSources[databaseColumn.Name])
		columns = append(columns, column)
	}
	// 稳定排序保证相同排序值或历史零值仍保留数据库字段顺序。
	slices.SortStableFunc(columns, func(left *systemadminv1.CodeGenColumn, right *systemadminv1.CodeGenColumn) int {
		if left.GetSort() < right.GetSort() {
			return -1
		}
		if left.GetSort() > right.GetSort() {
			return 1
		}
		return 0
	})
	return columns
}

// mergeSavedCodeGenColumn 将已有字段配置合并到最新数据库字段模型。
func (c *CodeGenColumnCase) mergeSavedCodeGenColumn(column *systemadminv1.CodeGenColumn, saved *models.CodeGenColumn) {
	if saved == nil {
		return
	}
	savedColumn := c.mapper.ToDTO(saved)
	column.Id = savedColumn.GetId()
	column.Comment = savedColumn.GetComment()
	column.QueryConfig = savedColumn.GetQueryConfig()
	column.ListConfig = savedColumn.GetListConfig()
	column.FormConfig = savedColumn.GetFormConfig()
	if savedColumn.GetSort() > 0 {
		column.Sort = savedColumn.GetSort()
	}
}

// filterConfigurableCodeGenColumns 过滤字段配置接口不允许维护的数据库字段。
func filterConfigurableCodeGenColumns(columns []*systemadminv1.CodeGenColumn) []*systemadminv1.CodeGenColumn {
	configurableColumns := make([]*systemadminv1.CodeGenColumn, 0, len(columns))
	for _, column := range columns {
		// 主键和软删除字段由基础设施维护，不通过字段配置接口返回。
		if column.GetIsPrimary() || column.GetName() == "deleted_at" {
			continue
		}
		configurableColumns = append(configurableColumns, column)
	}
	return configurableColumns
}

// newDefaultCodeGenColumn 根据数据库字段元数据创建默认生成配置。
func newDefaultCodeGenColumn(tableID int64, item dto.CodeGenDatabaseColumn, sort int32, defaultSource codeGenColumnDefaultSource) *systemadminv1.CodeGenColumn {
	columnComment := item.Comment
	// 数据库未配置字段注释时使用字段名作为展示名称。
	if columnComment == "" {
		columnComment = item.Name
	}
	lengthValue := item.CharacterMaximumLength
	// 数值字段没有字符长度时使用数值精度。
	if !lengthValue.Valid {
		lengthValue = item.NumericPrecision
	}
	var columnLength int32
	// 超出 Proto int32 范围的数据库长度不写入生成配置。
	if lengthValue.Valid && lengthValue.Int64 <= math.MaxInt32 {
		columnLength = int32(lengthValue.Int64)
	}
	var columnScale int32
	// 超出 Proto int32 范围的小数位不写入生成配置。
	if item.NumericScale.Valid && item.NumericScale.Int64 <= math.MaxInt32 {
		columnScale = int32(item.NumericScale.Int64)
	}
	isPrimary := item.ColumnKey == "PRI"
	isAutoIncrement := strings.Contains(strings.ToLower(item.Extra), "auto_increment")
	protoType := inferCodeGenProtoType(item.ColumnType)
	tsType := "string"
	// TypeScript 数值与布尔类型跟随 Proto 标量类型映射。
	if protoType == "bool" {
		tsType = "boolean"
	} else if protoType == "int64" || protoType == "int32" || protoType == "double" {
		tsType = "number"
	}
	column := &systemadminv1.CodeGenColumn{
		TableId:         tableID,
		Name:            item.Name,
		Comment:         columnComment,
		DbType:          item.ColumnType,
		DbLength:        columnLength,
		DbScale:         columnScale,
		IsPrimary:       isPrimary,
		IsAutoIncrement: isAutoIncrement,
		IsNullable:      item.IsNullable == "YES",
		GoType:          inferCodeGenGoType(item.ColumnType),
		ProtoType:       protoType,
		TsType:          tsType,
		Sort:            sort,
	}
	normalizeCodeGenColumnConfig(column, item, defaultSource)
	return column
}

// normalizeCodeGenColumnConfig 补齐缺失的结构化字段配置。
func normalizeCodeGenColumnConfig(column *systemadminv1.CodeGenColumn, item dto.CodeGenDatabaseColumn, defaultSource codeGenColumnDefaultSource) {
	// 旧配置没有保存字段描述时，完整沿用数据库原始注释。
	if column.Comment == "" {
		column.Comment = item.Comment
		if column.Comment == "" {
			column.Comment = item.Name
		}
	}
	// 缺失的查询配置按字段名称和数据库类型推导。
	if column.QueryConfig == nil {
		column.QueryConfig = defaultCodeGenQueryConfig(item.Name, item.ColumnType, defaultSource)
	}
	// 查询选项独立保存，不能与列表或表单配置共用。
	if column.QueryConfig.Option == nil {
		column.QueryConfig.Option = &systemadminv1.CodeGenColumnOptionConfig{}
	}
	// 缺失的列表配置按字段类型选择基础展示组件。
	if column.ListConfig == nil {
		component, option := defaultCodeGenListConfig(item, defaultSource)
		column.ListConfig = &systemadminv1.CodeGenColumnListConfig{
			Enabled:   !isManagedCodeGenColumn(item.Name) && !isCodeGenPasswordColumn(item.Name),
			Component: component,
			Option:    option,
		}
	}
	// 列表选项独立保存，不能与查询或表单配置共用。
	if column.ListConfig.Option == nil {
		column.ListConfig.Option = &systemadminv1.CodeGenColumnOptionConfig{}
	}
	// 缺失的表单配置按数据库约束推导。
	if column.FormConfig == nil {
		column.FormConfig = defaultCodeGenFormConfig(item, defaultSource)
	}
	// 表单选项独立保存，不能与查询或列表配置共用。
	if column.FormConfig.Option == nil {
		column.FormConfig.Option = &systemadminv1.CodeGenColumnOptionConfig{}
	}
	// 密码字段即使沿用历史配置也必须使用密码组件，避免回显密文或按普通文本录入。
	if isCodeGenPasswordColumn(item.Name) && column.FormConfig.GetEnabled() {
		column.FormConfig.Component = "password"
	}
}

// validateCodeGenColumnConfig 校验结构化字段配置的业务完整性。
func validateCodeGenColumnConfig(column *systemadminv1.CodeGenColumn, databaseColumn dto.CodeGenDatabaseColumn) error {
	err := validateCodeGenOptionConfig(column.GetName(), "查询", column.GetQueryConfig().GetOption())
	if err != nil {
		return err
	}
	err = validateCodeGenListOptionConfig(column.GetName(), column.GetListConfig())
	if err != nil {
		return err
	}
	err = validateCodeGenFormOptionConfig(column.GetName(), column.GetFormConfig())
	if err != nil {
		return err
	}
	formConfig := column.GetFormConfig()
	// 多选树形值以 JSON 数组存储，避免数组直接写入标量字段。
	if formConfig.GetMultiple() {
		if !formConfig.GetEnabled() || formConfig.GetComponent() != "tree-select" || formConfig.GetOption().GetKind() != "tree" {
			return errorsx.InvalidArgument("字段" + column.GetName() + "的表单多选仅支持树形选择")
		}
		if !strings.EqualFold(strings.TrimSpace(databaseColumn.DataType), "json") {
			return errorsx.InvalidArgument("字段" + column.GetName() + "的表单多选仅支持JSON字段")
		}
	}
	return nil
}

// validateCodeGenListOptionConfig 校验列表组件与选项配置是否匹配。
func validateCodeGenListOptionConfig(columnName string, config *systemadminv1.CodeGenColumnListConfig) error {
	option := config.GetOption()
	// 列表只允许使用已经确认的七种展示组件。
	switch config.GetComponent() {
	case "text", "image", "money", "date":
		if option.GetKind() != "" || option.GetSourceType() != "" || option.GetSourceValue() != "" || option.GetLabelField() != "" || option.GetValueField() != "" || option.GetParentField() != "" || option.GetActiveValue() != "" || option.GetInactiveValue() != "" {
			return errorsx.InvalidArgument("字段" + columnName + "的列表组件不需要选项配置")
		}
		return nil
	case "switch":
		if !config.GetEnabled() {
			return validateCodeGenOptionConfig(columnName, "列表", option)
		}
		return validateCodeGenSwitchOptionConfig(columnName, "列表", option)
	case "select":
		if !config.GetEnabled() {
			return validateCodeGenOptionConfig(columnName, "列表", option)
		}
		if option.GetKind() != "option" {
			return errorsx.InvalidArgument("字段" + columnName + "的列表下拉选项配置不完整")
		}
	case "tree-select":
		if !config.GetEnabled() {
			return validateCodeGenOptionConfig(columnName, "列表", option)
		}
		if option.GetKind() != "tree" {
			return errorsx.InvalidArgument("字段" + columnName + "的列表树形选项配置不完整")
		}
	default:
		return errorsx.InvalidArgument("字段" + columnName + "的列表组件不支持")
	}
	return validateCodeGenOptionConfig(columnName, "列表", option)
}

// validateCodeGenFormOptionConfig 校验表单组件与选项配置是否匹配。
func validateCodeGenFormOptionConfig(columnName string, config *systemadminv1.CodeGenColumnFormConfig) error {
	option := config.GetOption()
	// 未启用的表单组件不需要保留选项配置。
	if !config.GetEnabled() {
		return validateCodeGenOptionConfig(columnName, "表单", option)
	}
	// 开关必须配置字典与两个可区分的状态值。
	if config.GetComponent() == "switch" {
		return validateCodeGenSwitchOptionConfig(columnName, "表单", option)
	}
	// 字典选择组件只读取字典编码，不能使用静态数据或数据表来源。
	if config.GetComponent() == "dict" && (option.GetKind() != "option" || option.GetSourceType() != "dict" || option.GetSourceValue() == "") {
		return errorsx.InvalidArgument("字段" + columnName + "的表单字典选择配置不完整")
	}
	return validateCodeGenOptionConfig(columnName, "表单", option)
}

// validateCodeGenSwitchOptionConfig 校验列表或表单开关使用的字典和值配置。
func validateCodeGenSwitchOptionConfig(columnName string, scope string, option *systemadminv1.CodeGenColumnOptionConfig) error {
	// 开关必须绑定字典，并配置可区分的开启、关闭值。
	if option.GetKind() != "switch" || option.GetSourceType() != "dict" || option.GetSourceValue() == "" || option.GetActiveValue() == "" || option.GetInactiveValue() == "" {
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "开关配置不完整")
	}
	// 开关两个状态值不能相同。
	if option.GetActiveValue() == option.GetInactiveValue() {
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "开关开启值和关闭值不能相同")
	}
	return nil
}

// validateCodeGenOptionConfig 校验查询、列表或表单自己的选项配置。
func validateCodeGenOptionConfig(columnName string, scope string, option *systemadminv1.CodeGenColumnOptionConfig) error {
	// 未启用选项能力时不允许只残留部分来源字段。
	if option.GetKind() == "" {
		if option.GetSourceType() != "" || option.GetSourceValue() != "" || option.GetLabelField() != "" || option.GetValueField() != "" || option.GetParentField() != "" || option.GetActiveValue() != "" || option.GetInactiveValue() != "" || option.GetLazy() {
			return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "选项形态不能为空")
		}
		return nil
	}
	// 开关值只能由开关选项配置维护。
	if option.GetKind() != "switch" && (option.GetActiveValue() != "" || option.GetInactiveValue() != "") {
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "选项不能配置开关值")
	}
	if option.GetLazy() && option.GetKind() != "tree" {
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "仅树形选项支持懒加载")
	}
	// 树形选项只能使用数据库表构建真实父子关系。
	if option.GetKind() == "tree" && option.GetSourceType() != "table" {
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "树形选项只能使用数据表来源")
	}
	// 启用选项能力后，来源只能是静态数据、字典或数据表。
	switch option.GetSourceType() {
	case "static", "dict", "table":
	default:
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "选项来源配置不完整")
	}
	// 所有选项来源都必须配置具体来源值。
	if option.GetSourceValue() == "" {
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "选项来源值不能为空")
	}
	// 数据表选项必须明确展示字段、值字段以及树形父级字段。
	if option.GetSourceType() == "table" && (option.GetLabelField() == "" || option.GetValueField() == "" || option.GetKind() == "tree" && option.GetParentField() == "") {
		return errorsx.InvalidArgument("字段" + columnName + "的" + scope + "数据表选项字段配置不完整")
	}
	return nil
}

// defaultCodeGenQueryConfig 创建默认查询配置。
func defaultCodeGenQueryConfig(columnName string, dbType string, defaultSource codeGenColumnDefaultSource) *systemadminv1.CodeGenColumnQueryConfig {
	isStatus := isCodeGenStatusColumn(columnName)
	hasOptionSource := defaultSource.dictionaryCode != "" || defaultSource.tableName != ""
	enabled := columnName == "name" || columnName == "title" || columnName == "code" || isStatus || hasOptionSource
	operator := "like"
	component := "input"
	option := &systemadminv1.CodeGenColumnOptionConfig{}
	// 日期时间字段默认使用区间查询。
	if isCodeGenDateTimeType(dbType) {
		operator = "between"
		component = "date-picker"
	} else if isCodeGenBoolType(dbType) || isCodeGenNumericType(dbType) || isStatus || hasOptionSource {
		operator = "eq"
		component = "input-number"
	}
	// 布尔值、状态字段和命中字典或数据表的字段使用选项组件表达有限值集合。
	if isCodeGenBoolType(dbType) || isStatus || hasOptionSource {
		component = "select"
		option = defaultCodeGenColumnOption("option", defaultSource)
		if option.GetSourceValue() == "" {
			option = defaultCodeGenStatusOptionConfig("option")
		}
	}
	return &systemadminv1.CodeGenColumnQueryConfig{
		Enabled:   enabled,
		Operator:  operator,
		Component: component,
		Option:    option,
	}
}

// defaultCodeGenListConfig 创建默认列表展示配置。
func defaultCodeGenListConfig(item dto.CodeGenDatabaseColumn, defaultSource codeGenColumnDefaultSource) (string, *systemadminv1.CodeGenColumnOptionConfig) {
	if isCodeGenStatusColumn(item.Name) {
		return "switch", defaultCodeGenStatusOptionConfig("switch")
	}
	if defaultSource.dictionaryCode != "" || defaultSource.tableName != "" {
		return "select", defaultCodeGenColumnOption("option", defaultSource)
	}
	if isCodeGenDateTimeType(item.ColumnType) {
		return "date", &systemadminv1.CodeGenColumnOptionConfig{}
	}
	return "text", &systemadminv1.CodeGenColumnOptionConfig{}
}

// defaultCodeGenFormConfig 创建默认表单录入配置。
func defaultCodeGenFormConfig(item dto.CodeGenDatabaseColumn, defaultSource codeGenColumnDefaultSource) *systemadminv1.CodeGenColumnFormConfig {
	isPrimary := item.ColumnKey == "PRI"
	isAutoIncrement := strings.Contains(strings.ToLower(item.Extra), "auto_increment")
	enabled := !isPrimary && !isAutoIncrement && !isManagedCodeGenColumn(item.Name)
	component := "input"
	option := &systemadminv1.CodeGenColumnOptionConfig{}
	// 表单组件根据数据库字段类型选择，优先匹配更具体的布尔和数值类型。
	if isCodeGenPasswordColumn(item.Name) {
		component = "password"
	} else if isCodeGenStatusColumn(item.Name) || isCodeGenBoolType(item.ColumnType) {
		component = "switch"
		option = defaultCodeGenStatusOptionConfig("switch")
	} else if defaultSource.dictionaryCode != "" || defaultSource.tableName != "" {
		component = "select"
		if defaultSource.dictionaryCode != "" {
			component = "dict"
		}
		option = defaultCodeGenColumnOption("option", defaultSource)
	} else if isCodeGenNumericType(item.ColumnType) {
		component = "input-number"
	} else if isCodeGenDateTimeType(item.ColumnType) {
		component = "date-picker"
	} else if strings.Contains(strings.ToLower(item.ColumnType), "text") {
		component = "textarea"
	}
	return &systemadminv1.CodeGenColumnFormConfig{
		Enabled:   enabled,
		Component: component,
		Required:  enabled && item.IsNullable != "YES",
		Option:    option,
	}
}

// defaultCodeGenColumnOption 创建普通字段的默认选项来源配置。
func defaultCodeGenColumnOption(kind string, defaultSource codeGenColumnDefaultSource) *systemadminv1.CodeGenColumnOptionConfig {
	option := &systemadminv1.CodeGenColumnOptionConfig{
		Kind: kind,
	}
	if defaultSource.dictionaryCode != "" {
		option.SourceType = "dict"
		option.SourceValue = defaultSource.dictionaryCode
		option.LabelField = "label"
		option.ValueField = "value"
	} else if defaultSource.tableName != "" {
		option.SourceType = "table"
		option.SourceValue = defaultSource.tableName
	}
	return option
}

// defaultCodeGenStatusOptionConfig 创建状态字段使用的字典选项默认值。
func defaultCodeGenStatusOptionConfig(kind string) *systemadminv1.CodeGenColumnOptionConfig {
	option := defaultCodeGenColumnOption(kind, codeGenColumnDefaultSource{dictionaryCode: "status"})
	// 列表和表单开关需要显式保存两个状态值，查询下拉不需要。
	if kind == "switch" {
		option.ActiveValue = "1"
		option.InactiveValue = "2"
	}
	return option
}

// inferCodeGenGoType 推断 Go 字段类型。
func inferCodeGenGoType(dbType string) string {
	// tinyint(1) 等布尔字段优先映射为 bool。
	if isCodeGenBoolType(dbType) {
		return "bool"
	}
	lowerType := strings.ToLower(dbType)
	// bigint 需要保留 64 位整数范围。
	if strings.Contains(lowerType, "bigint") {
		return "int64"
	}
	// 其余数值字段按是否包含小数映射。
	if isCodeGenNumericType(lowerType) {
		if strings.Contains(lowerType, "decimal") || strings.Contains(lowerType, "float") || strings.Contains(lowerType, "double") {
			return "float64"
		}
		return "int32"
	}
	// Go 层保留数据库日期时间语义。
	if isCodeGenDateTimeType(lowerType) {
		return "time.Time"
	}
	return "string"
}

// inferCodeGenProtoType 推断 Proto 字段类型。
func inferCodeGenProtoType(dbType string) string {
	lowerType := strings.ToLower(dbType)
	// Proto 标量类型按数据库类型范围依次匹配。
	if isCodeGenBoolType(lowerType) {
		return "bool"
	}
	if strings.Contains(lowerType, "bigint") {
		return "int64"
	}
	if strings.Contains(lowerType, "int") {
		return "int32"
	}
	if strings.Contains(lowerType, "decimal") || strings.Contains(lowerType, "float") || strings.Contains(lowerType, "double") {
		return "double"
	}
	return "string"
}

// isManagedCodeGenColumn 判断字段是否由基础设施维护。
func isManagedCodeGenColumn(columnName string) bool {
	return columnName == "created_by" || columnName == "updated_by" || columnName == "created_at" || columnName == "updated_at" || columnName == "deleted_at"
}

// isCodeGenPasswordColumn 判断字段是否属于密码字段。
func isCodeGenPasswordColumn(columnName string) bool {
	lowerName := strings.ToLower(columnName)
	return lowerName == "password" || lowerName == "pwd" || lowerName == "passwd" ||
		strings.HasSuffix(lowerName, "_password") || strings.HasSuffix(lowerName, "_pwd") || strings.HasSuffix(lowerName, "_passwd")
}

// isCodeGenStatusColumn 判断字段是否默认按状态处理。
func isCodeGenStatusColumn(columnName string) bool {
	return columnName == "status" || columnName == "state" ||
		strings.HasSuffix(columnName, "_status") || strings.HasSuffix(columnName, "_state")
}

// isCodeGenBoolType 判断数据库字段是否表示布尔值。
func isCodeGenBoolType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return lowerType == "bool" || lowerType == "boolean" || strings.Contains(lowerType, "tinyint(1)")
}

// isCodeGenNumericType 判断数据库字段是否表示数值。
func isCodeGenNumericType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return strings.Contains(lowerType, "int") || strings.Contains(lowerType, "decimal") || strings.Contains(lowerType, "float") || strings.Contains(lowerType, "double")
}

// isCodeGenDateTimeType 判断数据库字段是否表示日期时间。
func isCodeGenDateTimeType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return strings.Contains(lowerType, "date") || strings.Contains(lowerType, "time")
}

// findCodeGenOptionTable 按字段去掉_id后的名称唯一匹配数据库表后缀。
func findCodeGenOptionTable(columnName string, tableInfos []dto.CodeGenDatabaseTable) string {
	if !strings.HasSuffix(columnName, "_id") {
		return ""
	}
	targetSuffix := "_" + strings.TrimSuffix(columnName, "_id")
	matchedTableName := ""
	matchCount := 0
	for _, tableInfo := range tableInfos {
		tableName := strings.ToLower(tableInfo.TableName)
		if tableName != strings.TrimPrefix(targetSuffix, "_") && !strings.HasSuffix(tableName, targetSuffix) {
			continue
		}
		matchedTableName = tableInfo.TableName
		matchCount++
	}
	if matchCount != 1 {
		return ""
	}
	return matchedTableName
}
