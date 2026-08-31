package biz

import (
	"context"
	"strings"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/mapper"
	"github.com/liujitcn/go-utils/stringcase"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

const (
	codeGenTriggerCRUD         = "crud"
	codeGenTriggerPageTree     = "page_tree"
	codeGenTriggerLeftTree     = "left_tree"
	codeGenTriggerEntityOption = "entity_option"
	codeGenTriggerFieldOption  = "field_option"
	codeGenTriggerFieldStatus  = "field_status"
	codeGenAPIKindCRUD         = "crud"
	codeGenAPIKindList         = "list"
	codeGenAPIKindOption       = "option"
	codeGenAPIKindTree         = "tree"
	codeGenAPIKindStatus       = "status"
)

// codeGenProtoConfigField 表示生成模板依赖的配置字段。
type codeGenProtoConfigField struct {
	label string
	value string
}

// CodeGenProtoCase 管理代码生成 Proto 接口配置。
type CodeGenProtoCase struct {
	*biz.BaseCase
	*data.CodeGenProtoRepository
	tx                data.Transaction
	baseAPIRepo       *data.BaseAPIRepository
	codeGenTableRepo  *data.CodeGenTableRepository
	codeGenColumnCase *CodeGenColumnCase
	mapper            *mapper.CopierMapper[adminv1.CodeGenProto, models.CodeGenProto]
	tableMapper       *mapper.CopierMapper[adminv1.CodeGenTableForm, models.CodeGenTable]
}

// NewCodeGenProtoCase 创建代码生成 Proto 接口业务实例。
func NewCodeGenProtoCase(
	baseCase *biz.BaseCase,
	codeGenProtoRepo *data.CodeGenProtoRepository,
	tx data.Transaction,
	baseAPIRepo *data.BaseAPIRepository,
	codeGenTableRepo *data.CodeGenTableRepository,
	codeGenColumnCase *CodeGenColumnCase,
) *CodeGenProtoCase {
	protoMapper := mapper.NewCopierMapper[adminv1.CodeGenProto, models.CodeGenProto]()
	protoMapper.AppendConverters(mapper.NewJSONTypeConverter[*adminv1.CodeGenProtoConfig]().NewConverterPair())
	protoMapper.AppendConverters(mapper.NewGenericTypeConverterPair(
		false,
		int32(0),
		func(value bool) int32 {
			// 数据库存储使用 1 表示启用状态。
			if value {
				return 1
			}
			return 0
		},
		func(value int32) bool {
			return value == 1
		},
	))
	tableMapper := mapper.NewCopierMapper[adminv1.CodeGenTableForm, models.CodeGenTable]()
	tableMapper.AppendConverters(mapper.NewJSONTypeConverter[*adminv1.CodeGenLeftTreeConfig]().NewConverterPair())
	return &CodeGenProtoCase{
		BaseCase:               baseCase,
		CodeGenProtoRepository: codeGenProtoRepo,
		tx:                     tx,
		baseAPIRepo:            baseAPIRepo,
		codeGenTableRepo:       codeGenTableRepo,
		codeGenColumnCase:      codeGenColumnCase,
		mapper:                 protoMapper,
		tableMapper:            tableMapper,
	}
}

// ListCodeGenProto 查询当前生成配置需要的 Proto 接口。
func (c *CodeGenProtoCase) ListCodeGenProto(ctx context.Context, tableID int64) (*adminv1.ListCodeGenProtoResponse, error) {
	table, err := c.codeGenTableRepo.FindByID(ctx, tableID)
	if err != nil {
		return nil, err
	}
	var columns []*adminv1.CodeGenColumn
	columns, err = c.codeGenColumnCase.listCodeGenColumns(ctx, tableID)
	if err != nil {
		return nil, err
	}
	query := c.Query(ctx).CodeGenProto
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.TableID.Eq(tableID)))
	opts = append(opts, repository.Order(query.Sort.Asc()))
	var savedProtos []*models.CodeGenProto
	savedProtos, err = c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	var checks []*adminv1.CodeGenProtoCheck
	checks, err = c.inspectCodeGenProtos(ctx, table, c.tableMapper.ToDTO(table), columns, savedProtos)
	if err != nil {
		return nil, err
	}
	return &adminv1.ListCodeGenProtoResponse{
		CodeGenProtos: checks,
	}, nil
}

// DeleteByTableIDs 删除多个代码生成表配置关联的 Proto 接口配置。
func (c *CodeGenProtoCase) DeleteByTableIDs(ctx context.Context, tableIDs []int64) error {
	if len(tableIDs) == 0 {
		return nil
	}
	query := c.Query(ctx).CodeGenProto
	return c.Delete(ctx, repository.Where(query.TableID.In(tableIDs...)))
}

// SaveCodeGenProto 按当前 Proto 能力清单同步接口配置。
func (c *CodeGenProtoCase) SaveCodeGenProto(ctx context.Context, req *adminv1.SaveCodeGenProtoRequest) error {
	table, err := c.codeGenTableRepo.FindByID(ctx, req.GetTableId())
	if err != nil {
		return err
	}
	var checks *adminv1.ListCodeGenProtoResponse
	checks, err = c.ListCodeGenProto(ctx, req.GetTableId())
	if err != nil {
		return err
	}
	protos := make([]*models.CodeGenProto, 0, len(req.GetCodeGenProtos()))
	columnNamesByTable := make(map[string]map[string]struct{})
	for index, input := range req.GetCodeGenProtos() {
		if input == nil {
			return errorsx.InvalidArgument("Proto接口配置不能为空")
		}
		if index >= len(checks.GetCodeGenProtos()) {
			return errorsx.InvalidArgument("Proto接口配置不在当前检查结果中")
		}
		check := checks.GetCodeGenProtos()[index]
		if input.GetTriggerType() != check.GetTriggerType() ||
			input.GetApiKind() != check.GetApiKind() ||
			input.GetSort() != 0 && input.GetSort() != check.GetSort() {
			return errorsx.InvalidArgument("Proto接口配置顺序或类型不正确")
		}
		// 数据库只保存缺失且勾选生成的接口；已存在或取消勾选的接口清理历史配置。
		if check.GetExists() || !input.GetGenerateWhenMissing() {
			continue
		}
		proto := &adminv1.CodeGenProto{
			TableId:             req.GetTableId(),
			TriggerType:         check.GetTriggerType(),
			ApiKind:             check.GetApiKind(),
			Config:              mergeCodeGenProtoConfig(check.GetApiKind(), input.GetConfig(), check.GetConfig()),
			GenerateWhenMissing: input.GetGenerateWhenMissing(),
			Sort:                input.GetSort(),
		}
		if proto.Sort == 0 {
			proto.Sort = int32(index + 1)
		}
		targetTableName := codeGenProtoTargetTableName(table, check.GetTargetEntityName())
		if err = c.validateCodeGenProtoColumns(ctx, targetTableName, check.GetMethodName(), proto, columnNamesByTable); err != nil {
			return err
		}
		item := c.mapper.ToEntity(proto)
		item.ID = 0
		protos = append(protos, item)
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		query := c.Query(ctx).CodeGenProto
		opts := make([]repository.QueryOption, 0, 2)
		opts = append(opts, repository.Where(query.TableID.Eq(req.GetTableId())))
		opts = append(opts, repository.Order(query.ID.Asc()))
		var savedProtos []*models.CodeGenProto
		savedProtos, err = c.List(ctx, opts...)
		if err != nil {
			return err
		}
		usedIDs := make(map[int64]struct{}, len(protos))
		for _, proto := range protos {
			var saved *models.CodeGenProto
			for _, candidate := range savedProtos {
				if _, used := usedIDs[candidate.ID]; used || candidate.Sort != proto.Sort {
					continue
				}
				saved = candidate
				break
			}
			if saved == nil {
				if err = c.Create(ctx, proto); err != nil {
					return err
				}
				continue
			}
			proto.ID = saved.ID
			usedIDs[saved.ID] = struct{}{}
			_, err = query.WithContext(ctx).Where(query.ID.Eq(proto.ID)).UpdateSimple(
				query.TriggerType.Value(proto.TriggerType),
				query.APIKind.Value(proto.APIKind),
				query.GenerateWhenMissing.Value(proto.GenerateWhenMissing),
				query.Config.Value(proto.Config),
				query.Sort.Value(proto.Sort),
			)
			if err != nil {
				return err
			}
		}
		deleteIDs := make([]int64, 0, len(savedProtos)-len(usedIDs))
		for _, saved := range savedProtos {
			// 当前推导清单不再需要、接口已经存在或历史重复的配置统一删除。
			if _, used := usedIDs[saved.ID]; !used {
				deleteIDs = append(deleteIDs, saved.ID)
			}
		}
		return c.DeleteByIDs(ctx, deleteIDs)
	})
}

// validateCodeGenProtoColumns 校验待生成 Proto 接口的类型配置和数据库字段。
func (c *CodeGenProtoCase) validateCodeGenProtoColumns(ctx context.Context, tableName string, methodName string, proto *adminv1.CodeGenProto, columnNamesByTable map[string]map[string]struct{}) error {
	// 未勾选生成的接口不消费类型配置，无需校验字段。
	if !proto.GetGenerateWhenMissing() {
		return nil
	}
	fields, supported := codeGenProtoConfigFields(proto.GetApiKind(), proto.GetConfig())
	// 未知接口类型没有可用的生成模板。
	if !supported {
		return errorsx.InvalidArgument("Proto接口" + methodName + "的接口类型不支持")
	}
	// 基础接口没有类型配置字段，无需读取数据库元数据。
	if len(fields) == 0 {
		return nil
	}
	columnNames, err := c.loadCodeGenProtoNames(ctx, tableName, columnNamesByTable)
	if err != nil {
		return err
	}
	for _, field := range fields {
		if field.value == "" {
			return errorsx.InvalidArgument("请选择Proto接口" + methodName + "的" + field.label)
		}
		if _, exists := columnNames[field.value]; !exists {
			return errorsx.InvalidArgument("Proto接口" + methodName + "的" + field.label + field.value + "不属于目标表" + tableName)
		}
	}
	return nil
}

// inspectCodeGenProtos 推导当前配置需要的 Proto 接口并检查仓库与目标表状态。
func (c *CodeGenProtoCase) inspectCodeGenProtos(ctx context.Context, table *models.CodeGenTable, form *adminv1.CodeGenTableForm, columns []*adminv1.CodeGenColumn, savedProtos []*models.CodeGenProto) ([]*adminv1.CodeGenProtoCheck, error) {
	checks := buildExpectedCodeGenProtos(table, form, columns)
	businessNames, err := c.codeGenProtoBusinessNames(ctx, table, form, checks)
	if err != nil {
		return nil, err
	}
	serviceBusinessNames := make(map[string]string, len(checks))
	for _, check := range checks {
		key := check.GetProtoFilePath() + ":" + check.GetTargetEntityName()
		if _, exists := serviceBusinessNames[key]; exists {
			continue
		}
		serviceBusinessNames[key] = businessNames[check.GetTargetEntityName()+":"+check.GetTriggerType()]
	}
	columnNamesByTable := make(map[string]map[string]struct{})
	codeGenTable := &codegen.Table{
		BusinessModule: table.BusinessModule,
		EntityName:     stringcase.ToPascalCase(table.Name),
	}
	for _, check := range checks {
		businessName := businessNames[check.GetTargetEntityName()+":"+check.GetTriggerType()]
		serviceBusinessName := serviceBusinessNames[check.GetProtoFilePath()+":"+check.GetTargetEntityName()]
		check.MethodComment = codegen.GeneratedProtoMethodComment(
			businessName,
			check.GetTargetEntityName(),
			check.GetTriggerType(),
			check.GetApiKind(),
			check.GetMethodName(),
		)
		check.ServiceName, check.ServiceComment = codeGenProtoServiceMetadata(
			check.GetTargetEntityName(),
			check.GetTargetEntityName()+"Service",
			"Admin"+serviceBusinessName+"服务",
		)
		for _, saved := range savedProtos {
			if !savedCodeGenProtoMatches(saved, check) {
				continue
			}
			check.GenerateWhenMissing = saved.GenerateWhenMissing == 1
			savedProto := c.mapper.ToDTO(saved)
			check.Config = mergeCodeGenProtoConfig(check.GetApiKind(), savedProto.GetConfig(), check.GetConfig())
			break
		}
		check.Exists, check.Message, err = c.codeGenProtoExists(ctx, codeGenTable, check)
		if err != nil {
			return nil, err
		}
		// 仓库中已存在接口时不再进入后续生成流程。
		if check.Exists {
			check.GenerateWhenMissing = false
			continue
		}
		// 用户已取消选择时保留保存结果，不再做默认字段判断。
		if !check.GenerateWhenMissing {
			continue
		}
		targetTableName := codeGenProtoTargetTableName(table, check.GetTargetEntityName())
		// 默认字段只用于判断新接口能否默认勾选，不能回填到未保存的配置中。
		selectionConfig := mergeCodeGenProtoConfig(
			check.GetApiKind(),
			check.GetConfig(),
			defaultCodeGenProtoConfig(check.GetApiKind()),
		)
		fields, supported := codeGenProtoConfigFields(check.GetApiKind(), selectionConfig)
		// 未知接口类型没有可用生成模板，不能默认勾选。
		if !supported {
			check.GenerateWhenMissing = false
			continue
		}
		// 基础接口不依赖类型配置字段，保持默认勾选。
		if len(fields) == 0 {
			continue
		}
		var columnNames map[string]struct{}
		columnNames, err = c.loadCodeGenProtoNames(ctx, targetTableName, columnNamesByTable)
		if err != nil {
			return nil, err
		}
		// 目标表缺少当前配置需要的任一字段时取消默认勾选。
		if !hasCodeGenProtoConfigColumns(check.GetApiKind(), selectionConfig, columnNames) {
			check.GenerateWhenMissing = false
		}
	}
	return checks, nil
}

// codeGenProtoBusinessNames 加载 Proto 目标实体使用的业务描述，保持检查结果与生成器一致。
func (c *CodeGenProtoCase) codeGenProtoBusinessNames(ctx context.Context, table *models.CodeGenTable, form *adminv1.CodeGenTableForm, checks []*adminv1.CodeGenProtoCheck) (map[string]string, error) {
	tableNames := make(map[string]struct{}, len(checks))
	tableNameByCheck := make(map[string]string, len(checks))
	targetEntityByCheck := make(map[string]string, len(checks))
	leftTree := form.GetLeftTreeConfig()
	for _, check := range checks {
		tableName := codeGenProtoTargetTableName(table, check.GetTargetEntityName())
		if check.GetTriggerType() == codeGenTriggerLeftTree && leftTree.GetTableName() != "" {
			tableName = leftTree.GetTableName()
		}
		key := check.GetTargetEntityName() + ":" + check.GetTriggerType()
		tableNameByCheck[key] = tableName
		targetEntityByCheck[key] = check.GetTargetEntityName()
		if tableName != "" {
			tableNames[tableName] = struct{}{}
		}
	}
	commentsByTableName := make(map[string]string, len(tableNames))
	for tableName := range tableNames {
		commentsByTableName[tableName] = ""
	}
	var databaseTables []dto.CodeGenDatabaseTable
	var err error
	tableNameList := make([]string, 0, len(tableNames))
	for tableName := range tableNames {
		tableNameList = append(tableNameList, tableName)
	}
	if len(tableNames) > 0 {
		database := c.GormClients[gorm.DefaultClientName]
		err = database.DB.WithContext(ctx).
			Table("information_schema.tables").
			Select("table_name, table_comment").
			Where("table_schema = DATABASE()").
			Where("table_type = ?", "BASE TABLE").
			Where("table_name IN ?", tableNameList).
			Find(&databaseTables).Error
		if err != nil {
			return nil, err
		}
	}
	for _, databaseTable := range databaseTables {
		commentsByTableName[databaseTable.TableName] = databaseTable.TableComment
	}
	if table.Comment != "" {
		commentsByTableName[table.Name] = table.Comment
	}
	if len(tableNames) > 0 {
		query := c.codeGenTableRepo.Query(ctx).CodeGenTable
		var configuredTables []*models.CodeGenTable
		configuredTables, err = c.codeGenTableRepo.List(ctx, repository.Where(query.Name.In(tableNameList...)))
		if err != nil {
			return nil, err
		}
		for _, configuredTable := range configuredTables {
			if configuredTable.Comment != "" {
				commentsByTableName[configuredTable.Name] = configuredTable.Comment
			}
		}
	}
	businessNames := make(map[string]string, len(tableNameByCheck))
	for key, tableName := range tableNameByCheck {
		comment := commentsByTableName[tableName]
		if strings.HasSuffix(key, ":"+codeGenTriggerLeftTree) && leftTree.GetComment() != "" {
			comment = leftTree.GetComment()
		}
		businessNames[key] = codegen.DefaultString(comment, targetEntityByCheck[key])
	}
	return businessNames, nil
}

// loadCodeGenProtoNames 加载并缓存目标表字段名。
func (c *CodeGenProtoCase) loadCodeGenProtoNames(ctx context.Context, tableName string, columnNamesByTable map[string]map[string]struct{}) (map[string]struct{}, error) {
	columnNames := columnNamesByTable[tableName]
	// 同一目标表已加载时直接复用字段集合。
	if columnNames != nil {
		return columnNames, nil
	}
	databaseColumns, err := c.codeGenColumnCase.listDatabaseColumns(ctx, tableName)
	if err != nil {
		return nil, err
	}
	columnNames = make(map[string]struct{}, len(databaseColumns))
	for _, column := range databaseColumns {
		columnNames[column.Name] = struct{}{}
	}
	columnNamesByTable[tableName] = columnNames
	return columnNames, nil
}

// buildExpectedCodeGenProtos 根据表与字段配置推导所需 Proto 接口。
func buildExpectedCodeGenProtos(table *models.CodeGenTable, form *adminv1.CodeGenTableForm, columns []*adminv1.CodeGenColumn) []*adminv1.CodeGenProtoCheck {
	protoPath := defaultCodeGenProtoPath(table)
	entity := stringcase.ToPascalCase(table.Name)
	checks := make([]*adminv1.CodeGenProtoCheck, 0, 10)
	// 树形页面使用树接口，普通页面使用分页与平铺选项接口。
	if table.PageType == "tree" || table.PageType == codegen.PageTypeTreeLazy {
		checks = append(checks,
			newCodeGenProtoCheck(table.ID, codeGenTriggerPageTree, codeGenAPIKindTree, entity, "Tree"+entity, protoPath),
			newCodeGenProtoCheck(table.ID, codeGenTriggerEntityOption, codeGenAPIKindTree, entity, "Option"+entity, protoPath),
		)
	} else {
		checks = append(checks,
			newCodeGenProtoCheck(table.ID, codeGenTriggerCRUD, codeGenAPIKindList, entity, "Page"+entity, protoPath),
			newCodeGenProtoCheck(table.ID, codeGenTriggerEntityOption, codeGenAPIKindOption, entity, "Option"+entity, protoPath),
		)
	}
	checks = append(checks,
		newCodeGenProtoCheck(table.ID, codeGenTriggerCRUD, codeGenAPIKindCRUD, entity, "Get"+entity, protoPath),
		newCodeGenProtoCheck(table.ID, codeGenTriggerCRUD, codeGenAPIKindCRUD, entity, "Create"+entity, protoPath),
		newCodeGenProtoCheck(table.ID, codeGenTriggerCRUD, codeGenAPIKindCRUD, entity, "Update"+entity, protoPath),
		newCodeGenProtoCheck(table.ID, codeGenTriggerCRUD, codeGenAPIKindCRUD, entity, "Delete"+entity, protoPath),
	)
	leftTree := form.GetLeftTreeConfig()
	// 左树页面的数据表来源需要在目标实体上提供树形选项接口。
	if table.PageType == "left_tree" && leftTree.GetTableName() != "" {
		target := stringcase.ToPascalCase(leftTree.GetTableName())
		check := newCodeGenProtoCheck(
			table.ID,
			codeGenTriggerLeftTree,
			codeGenAPIKindTree,
			target,
			"Option"+target,
			defaultTargetCodeGenProtoPath(table, target),
		)
		check.Config = &adminv1.CodeGenProtoConfig{
			ParentColumn: leftTree.GetParentColumn(),
			LabelColumn:  leftTree.GetLabelColumn(),
			ValueColumn:  leftTree.GetValueColumn(),
			Lazy:         leftTree.GetLazy(),
		}
		checks = append(checks, check)
	}
	for _, column := range columns {
		listConfig := column.GetListConfig()
		// 列表使用开关时为当前字段补充状态变更 RPC。
		if listConfig.GetEnabled() && listConfig.GetComponent() == "switch" {
			checks = append(checks, newCodeGenProtoCheck(
				table.ID,
				codeGenTriggerFieldStatus,
				codeGenAPIKindStatus,
				entity,
				"Set"+entity+stringcase.ToPascalCase(column.GetName()),
				protoPath,
			))
		}
		var options []*adminv1.CodeGenColumnOptionConfig
		// 只检查已启用配置中的数据表选项，查询、列表和表单彼此独立。
		if column.GetQueryConfig().GetEnabled() {
			options = append(options, column.GetQueryConfig().GetOption())
		}
		if column.GetListConfig().GetEnabled() {
			options = append(options, column.GetListConfig().GetOption())
		}
		if column.GetFormConfig().GetEnabled() {
			options = append(options, column.GetFormConfig().GetOption())
		}
		for _, option := range options {
			// 只有数据表来源的选项字段才需要检查目标实体接口。
			if option.GetKind() == "" || option.GetSourceType() != "table" {
				continue
			}
			target := stringcase.ToPascalCase(option.GetSourceValue())
			// 来源未声明实体名时回退到当前实体。
			if target == "" {
				target = entity
			}
			apiKind := codeGenAPIKindOption
			// 树形选项需要使用树接口类型。
			if option.GetKind() == "tree" {
				apiKind = codeGenAPIKindTree
			}
			check := newCodeGenProtoCheck(
				table.ID,
				codeGenTriggerFieldOption,
				apiKind,
				target,
				"Option"+target,
				defaultTargetCodeGenProtoPath(table, target),
			)
			check.Config = &adminv1.CodeGenProtoConfig{
				ParentColumn: option.GetParentField(),
				LabelColumn:  option.GetLabelField(),
				ValueColumn:  option.GetValueField(),
				Lazy:         option.GetLazy(),
			}
			checks = append(checks, check)
		}
	}
	list := make([]*adminv1.CodeGenProtoCheck, 0, len(checks))
	seen := make(map[string]struct{}, len(checks))
	for _, check := range checks {
		key := codeGenProtoKey(check.GetProtoFilePath(), check.GetTargetEntityName(), check.GetMethodName())
		// 同一目标服务的同名 RPC 只保留一份检查项。
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		check.Sort = int32(len(list) + 1)
		list = append(list, check)
	}
	return list
}

// newCodeGenProtoCheck 创建默认勾选生成的 Proto 接口检查项。
func newCodeGenProtoCheck(tableID int64, triggerType string, apiKind string, targetEntity string, methodName string, protoPath string) *adminv1.CodeGenProtoCheck {
	return &adminv1.CodeGenProtoCheck{
		TableId:             tableID,
		TriggerType:         triggerType,
		ApiKind:             apiKind,
		TargetEntityName:    targetEntity,
		MethodName:          methodName,
		ProtoFilePath:       protoPath,
		Config:              &adminv1.CodeGenProtoConfig{},
		GenerateWhenMissing: true,
	}
}

// defaultCodeGenProtoConfig 返回接口类型用于默认选择判断的约定字段。
func defaultCodeGenProtoConfig(apiKind string) *adminv1.CodeGenProtoConfig {
	// 不同生成模板只保留自身消费的配置字段。
	switch apiKind {
	case codeGenAPIKindOption:
		return &adminv1.CodeGenProtoConfig{LabelColumn: "name", ValueColumn: "id"}
	case codeGenAPIKindTree:
		return &adminv1.CodeGenProtoConfig{ParentColumn: "parent_id", LabelColumn: "name", ValueColumn: "id"}
	case codeGenAPIKindStatus:
		return &adminv1.CodeGenProtoConfig{StatusColumn: "status"}
	default:
		return &adminv1.CodeGenProtoConfig{}
	}
}

// codeGenProtoConfigFields 返回接口类型生成模板依赖的配置字段。
func codeGenProtoConfigFields(apiKind string, config *adminv1.CodeGenProtoConfig) ([]codeGenProtoConfigField, bool) {
	// 每种接口类型只返回生成模板实际消费的固定字段。
	switch apiKind {
	case codeGenAPIKindCRUD, codeGenAPIKindList:
		return nil, true
	case codeGenAPIKindOption:
		return []codeGenProtoConfigField{
			{label: "显示字段", value: config.GetLabelColumn()},
			{label: "值字段", value: config.GetValueColumn()},
		}, true
	case codeGenAPIKindTree:
		return []codeGenProtoConfigField{
			{label: "父节点字段", value: config.GetParentColumn()},
			{label: "显示字段", value: config.GetLabelColumn()},
			{label: "值字段", value: config.GetValueColumn()},
		}, true
	case codeGenAPIKindStatus:
		return []codeGenProtoConfigField{
			{label: "状态字段", value: config.GetStatusColumn()},
		}, true
	default:
		return nil, false
	}
}

// hasCodeGenProtoConfigColumns 判断目标表是否包含接口配置所需的全部字段。
func hasCodeGenProtoConfigColumns(apiKind string, config *adminv1.CodeGenProtoConfig, columnNames map[string]struct{}) bool {
	fields, supported := codeGenProtoConfigFields(apiKind, config)
	// 未知类型不能进入默认生成选择。
	if !supported {
		return false
	}
	for _, field := range fields {
		if field.value == "" {
			return false
		}
		if _, exists := columnNames[field.value]; !exists {
			return false
		}
	}
	return true
}

// mergeCodeGenProtoConfig 合并用户配置与推导默认值，并按接口类型裁剪无关字段。
func mergeCodeGenProtoConfig(apiKind string, preferred *adminv1.CodeGenProtoConfig, fallback *adminv1.CodeGenProtoConfig) *adminv1.CodeGenProtoConfig {
	config := &adminv1.CodeGenProtoConfig{}
	// 每种接口类型只合并生成模板需要的字段。
	switch apiKind {
	case codeGenAPIKindOption:
		config.LabelColumn = preferred.GetLabelColumn()
		config.ValueColumn = preferred.GetValueColumn()
		if config.LabelColumn == "" {
			config.LabelColumn = fallback.GetLabelColumn()
		}
		if config.ValueColumn == "" {
			config.ValueColumn = fallback.GetValueColumn()
		}
	case codeGenAPIKindTree:
		config.ParentColumn = preferred.GetParentColumn()
		config.LabelColumn = preferred.GetLabelColumn()
		config.ValueColumn = preferred.GetValueColumn()
		config.Lazy = preferred.GetLazy()
		if config.ParentColumn == "" {
			config.ParentColumn = fallback.GetParentColumn()
		}
		if config.LabelColumn == "" {
			config.LabelColumn = fallback.GetLabelColumn()
		}
		if config.ValueColumn == "" {
			config.ValueColumn = fallback.GetValueColumn()
		}
	case codeGenAPIKindStatus:
		config.StatusColumn = preferred.GetStatusColumn()
		if config.StatusColumn == "" {
			config.StatusColumn = fallback.GetStatusColumn()
		}
	}
	return config
}

// defaultCodeGenProtoPath 返回当前实体默认 Proto 文件路径。
func defaultCodeGenProtoPath(table *models.CodeGenTable) string {
	target, _ := codegen.ProtoTargetForBusinessModule(table.BusinessModule)
	return codegen.ProtoFilePath(target.Directory, stringcase.ToPascalCase(table.Name))
}

// defaultTargetCodeGenProtoPath 返回关联实体默认 Proto 文件路径。
func defaultTargetCodeGenProtoPath(table *models.CodeGenTable, target string) string {
	// 关联目标就是当前实体时复用当前实体默认路径。
	if target == stringcase.ToPascalCase(table.Name) {
		return defaultCodeGenProtoPath(table)
	}
	if path, ok := codegen.ExistingProtoFilePath(target, "Option"+target, table.BusinessModule); ok {
		return path
	}
	targetConfig, _ := codegen.ProtoTargetForBusinessModule(table.BusinessModule)
	return codegen.ProtoFilePath(targetConfig.Directory, target)
}

// codeGenProtoTargetTableName 返回接口目标实体对应的数据库表名。
func codeGenProtoTargetTableName(table *models.CodeGenTable, targetEntity string) string {
	// 当前实体保留代码生成配置中的原始表名。
	if targetEntity == stringcase.ToPascalCase(table.Name) {
		return table.Name
	}
	return stringcase.ToSnakeCase(targetEntity)
}

// savedCodeGenProtoMatches 判断已保存配置是否对应当前检查项，并兼容旧版复数契约名。
func savedCodeGenProtoMatches(saved *models.CodeGenProto, check *adminv1.CodeGenProtoCheck) bool {
	return saved.Sort == check.GetSort()
}

// codeGenProtoKey 返回 Proto 接口配置稳定键。
func codeGenProtoKey(protoPath string, targetEntity string, methodName string) string {
	return protoPath + ":" + targetEntity + ":" + methodName
}

// codeGenProtoExists 根据 base_api 中的 path 与 operation 检查接口是否已存在。
func (c *CodeGenProtoCase) codeGenProtoExists(ctx context.Context, table *codegen.Table, check *adminv1.CodeGenProtoCheck) (bool, string, error) {
	method := &codegen.Proto{
		TriggerType:      check.GetTriggerType(),
		APIKind:          check.GetApiKind(),
		TargetEntityName: check.GetTargetEntityName(),
		MethodName:       check.GetMethodName(),
	}
	_, path, ok := codegen.GeneratedHTTPRoute(table, method)
	if !ok {
		return false, "无法推导接口路由", nil
	}
	operation := codegen.GeneratedRPCPath(table, method)
	query := c.baseAPIRepo.Query(ctx).BaseAPI
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.Path.Eq(path)))
	opts = append(opts, repository.Where(query.Operation.Eq(operation)))
	apis, err := c.baseAPIRepo.List(ctx, opts...)
	if err != nil {
		return false, "", err
	}
	if len(apis) > 0 {
		return true, "已存在", nil
	}
	return false, "缺少，可选择生成", nil
}

// codeGenProtoServiceMetadata 返回 Proto 检查项对应的服务名与服务描述。
func codeGenProtoServiceMetadata(targetEntity string, fallbackServiceName string, fallbackServiceComment string) (string, string) {
	serviceName := codegen.DefaultString(fallbackServiceName, targetEntity+"Service")
	return serviceName, codegen.DefaultString(fallbackServiceComment, "Admin"+targetEntity+"服务")
}
