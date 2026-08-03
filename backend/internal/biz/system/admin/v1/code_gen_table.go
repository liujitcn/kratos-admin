package biz

import (
	"context"
	"regexp"

	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/v1/dto"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"

	"github.com/liujitcn/go-utils/mapper"
	_string "github.com/liujitcn/go-utils/string"
	"github.com/liujitcn/gorm-kit/repository"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
)

var codeGenBusinessModulePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

const (
	// codeGenPageTypeNormal 表示普通表格页面。
	codeGenPageTypeNormal = "normal"
)

// CodeGenTableCase 管理代码生成表配置。
type CodeGenTableCase struct {
	*data.CodeGenTableRepository
	dbClient          *databaseGorm.Client // 数据库元数据客户端
	tx                data.Transaction
	baseDictRepo      *data.BaseDictRepository
	baseDictItemRepo  *data.BaseDictItemRepository
	baseMenuCase      *BaseMenuCase
	codeGenColumnCase *CodeGenColumnCase
	codeGenProtoCase  *CodeGenProtoCase
	formMapper        *mapper.CopierMapper[systemadminv1.CodeGenTableForm, models.CodeGenTable]
	mapper            *mapper.CopierMapper[systemadminv1.CodeGenTable, models.CodeGenTable]
}

// NewCodeGenTableCase 创建代码生成表配置业务实例。
func NewCodeGenTableCase(
	codeGenTableRepo *data.CodeGenTableRepository,
	dbClient *databaseGorm.Client,
	tx data.Transaction,
	baseDictRepo *data.BaseDictRepository,
	baseDictItemRepo *data.BaseDictItemRepository,
	baseMenuCase *BaseMenuCase,
	codeGenColumnCase *CodeGenColumnCase,
	codeGenProtoCase *CodeGenProtoCase,
) *CodeGenTableCase {
	formMapper := mapper.NewCopierMapper[systemadminv1.CodeGenTableForm, models.CodeGenTable]()
	formMapper.AppendConverters(mapper.NewJSONTypeConverter[*systemadminv1.CodeGenLeftTreeConfig]().NewConverterPair())
	formMapper.AppendConverters(mapper.NewJSONTypeConverter[map[string]*systemadminv1.CodeGenLocaleConfig]().NewConverterPair())
	formMapper.AppendConverters(mapper.NewGenericTypeConverterPair(
		false,
		int32(0),
		func(value bool) int32 {
			if value {
				return 1
			}
			return 0
		},
		func(value int32) bool {
			return value == 1
		},
	))
	tableMapper := mapper.NewCopierMapper[systemadminv1.CodeGenTable, models.CodeGenTable]()
	tableMapper.AppendConverters(mapper.NewJSONTypeConverter[map[string]*systemadminv1.CodeGenLocaleConfig]().NewConverterPair())
	return &CodeGenTableCase{
		CodeGenTableRepository: codeGenTableRepo,
		dbClient:               dbClient,
		tx:                     tx,
		baseDictRepo:           baseDictRepo,
		baseDictItemRepo:       baseDictItemRepo,
		baseMenuCase:           baseMenuCase,
		codeGenColumnCase:      codeGenColumnCase,
		codeGenProtoCase:       codeGenProtoCase,
		formMapper:             formMapper,
		mapper:                 tableMapper,
	}
}

// PageCodeGenTable 查询代码生成表配置分页数据。
func (c *CodeGenTableCase) PageCodeGenTable(ctx context.Context, req *systemadminv1.PageCodeGenTableRequest) (*systemadminv1.PageCodeGenTableResponse, error) {
	query := c.Query(ctx).CodeGenTable
	opts := make([]repository.QueryOption, 0, 4)
	opts = append(opts, repository.Order(query.CreatedAt.Desc()))
	if req.Name != nil {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	if req.BusinessModule != nil {
		opts = append(opts, repository.Where(query.BusinessModule.Eq(req.GetBusinessModule())))
	}
	if req.PageType != nil {
		opts = append(opts, repository.Where(query.PageType.Eq(req.GetPageType())))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(int32(req.GetStatus()))))
	}
	list, total, err := c.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	codeGenTables := make([]*systemadminv1.CodeGenTable, 0, len(list))
	for _, item := range list {
		table := c.mapper.ToDTO(item)
		table.RestoreAvailable = RestoreAvailable(item.ID)
		codeGenTables = append(codeGenTables, table)
	}
	return &systemadminv1.PageCodeGenTableResponse{CodeGenTables: codeGenTables, Total: int32(total)}, nil
}

// ListCodeGenDatabaseTable 查询当前数据库表元数据。
func (c *CodeGenTableCase) ListCodeGenDatabaseTable(ctx context.Context) (*systemadminv1.ListCodeGenDatabaseTableResponse, error) {
	query := c.Query(ctx).CodeGenTable
	opts := make([]repository.QueryOption, 0, 1)
	opts = append(opts, repository.Order(query.Name.Asc()))
	list, err := c.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	usedTableNames := make(map[string]bool, len(list))
	for _, item := range list {
		usedTableNames[item.Name] = true
	}
	var tableInfos []dto.CodeGenDatabaseTable
	tableInfos, err = c.listDatabaseTables(ctx, nil)
	if err != nil {
		return nil, err
	}
	tables := make([]*systemadminv1.CodeGenDatabaseTable, 0, len(tableInfos))
	for _, tableInfo := range tableInfos {
		tables = append(tables, &systemadminv1.CodeGenDatabaseTable{
			Name:     tableInfo.TableName,
			Comment:  tableInfo.TableComment,
			Disabled: usedTableNames[tableInfo.TableName],
		})
	}
	return &systemadminv1.ListCodeGenDatabaseTableResponse{Tables: tables}, nil
}

// GetCodeGenTable 查询代码生成表配置。
func (c *CodeGenTableCase) GetCodeGenTable(ctx context.Context, id int64) (*systemadminv1.CodeGenTableForm, error) {
	item, err := c.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	form := c.formMapper.ToDTO(item)
	if form.LeftTreeConfig == nil {
		form.LeftTreeConfig = &systemadminv1.CodeGenLeftTreeConfig{}
	}
	return form, nil
}

// ValidateBusinessModule 校验业务模块是否为启用的数据字典项。
func (c *CodeGenTableCase) ValidateBusinessModule(ctx context.Context, module string) error {
	if !codeGenBusinessModulePattern.MatchString(module) {
		return errorsx.InvalidArgument("业务模块格式不正确")
	}
	dictQuery := c.baseDictRepo.Query(ctx).BaseDict
	dict, err := c.baseDictRepo.Find(ctx, repository.Where(dictQuery.Code.Eq("business_module")), repository.Where(dictQuery.Status.Eq(_const.STATUS_ENABLE)))
	if err != nil {
		return errorsx.InvalidArgument("业务模块字典不存在").WithCause(err)
	}
	itemQuery := c.baseDictItemRepo.Query(ctx).BaseDictItem
	_, err = c.baseDictItemRepo.Find(ctx, repository.Where(itemQuery.DictID.Eq(dict.ID)), repository.Where(itemQuery.Value.Eq(module)), repository.Where(itemQuery.Status.Eq(_const.STATUS_ENABLE)))
	if err != nil {
		return errorsx.InvalidArgument("请选择启用的业务模块").WithCause(err)
	}
	return nil
}

// CreateCodeGenTable 创建代码生成表配置。
func (c *CodeGenTableCase) CreateCodeGenTable(ctx context.Context, req *systemadminv1.CodeGenTableForm) error {
	item, err := c.codeGenTableFormToModel(ctx, req)
	if err != nil {
		return err
	}
	item.ID = 0
	err = c.Create(ctx, item)
	if err != nil {
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("业务表已被代码生成表配置选择", "code_gen_table", "", "unique_code_gen_table").WithCause(err)
		}
		return err
	}
	return nil
}

// UpdateCodeGenTable 更新代码生成表配置。
func (c *CodeGenTableCase) UpdateCodeGenTable(ctx context.Context, id int64, req *systemadminv1.CodeGenTableForm) error {
	item, err := c.codeGenTableFormToModel(ctx, req)
	if err != nil {
		return err
	}
	item.ID = id
	query := c.Query(ctx).CodeGenTable
	opts := make([]repository.QueryOption, 0, 2)
	opts = append(opts, repository.Where(query.ID.Eq(id)))
	opts = append(opts, repository.Select(
		query.Name,
		query.Comment,
		query.BusinessModule,
		query.ParentMenuID,
		query.PageType,
		query.ParentColumn,
		query.TreeLabelColumn,
		query.LeftTreeConfig,
		query.GenBackend,
		query.GenFrontend,
		query.GenSql,
		query.Status,
		query.Remark,
		query.I18NConfig,
	))
	err = c.Update(ctx, item, opts...)
	if err != nil {
		if errorsx.IsMySQLDuplicateKey(err) {
			return errorsx.UniqueConflict("业务表已被代码生成表配置选择", "code_gen_table", "", "unique_code_gen_table").WithCause(err)
		}
		return err
	}
	return nil
}

// DeleteCodeGenTable 删除代码生成表配置。
func (c *CodeGenTableCase) DeleteCodeGenTable(ctx context.Context, ids string) error {
	idList := _string.ConvertStringToInt64Array(ids)
	if len(idList) == 0 {
		return nil
	}
	return c.tx.Transaction(ctx, func(ctx context.Context) error {
		err := c.codeGenColumnCase.DeleteByTableIDs(ctx, idList)
		if err != nil {
			return err
		}
		err = c.codeGenProtoCase.DeleteByTableIDs(ctx, idList)
		if err != nil {
			return err
		}
		return c.DeleteByIDs(ctx, idList)
	})
}

// listDatabaseTables 查询当前数据库的表名与表描述，可按表名缩小范围。
func (c *CodeGenTableCase) listDatabaseTables(ctx context.Context, tableNames []string) ([]dto.CodeGenDatabaseTable, error) {
	query := c.dbClient.DB.WithContext(ctx).
		Table("information_schema.tables").
		Select("table_name, table_comment").
		Where("table_schema = DATABASE()").
		Where("table_type = ?", "BASE TABLE")
	if len(tableNames) > 0 {
		query = query.Where("table_name IN ?", tableNames)
	}
	var tableInfos []dto.CodeGenDatabaseTable
	err := query.Order("table_name").Find(&tableInfos).Error
	return tableInfos, err
}

// codeGenTableFormToModel 转换代码生成表配置保存模型，并校验生成所需的关联配置。
func (c *CodeGenTableCase) codeGenTableFormToModel(ctx context.Context, req *systemadminv1.CodeGenTableForm) (*models.CodeGenTable, error) {
	module := req.GetBusinessModule()
	err := c.ValidateBusinessModule(ctx, module)
	if err != nil {
		return nil, err
	}
	parentMenuID := req.GetParentMenuId()
	if parentMenuID <= 0 {
		return nil, errorsx.InvalidArgument("请选择父级菜单")
	}
	var menu *models.BaseMenu
	menu, err = c.baseMenuCase.FindByID(ctx, parentMenuID)
	if err != nil {
		return nil, errorsx.InvalidArgument("父级菜单不存在").WithCause(err)
	}
	if err = validateBaseMenuChild(menu, _const.BASE_MENU_TYPE_MENU); err != nil {
		return nil, err
	}
	item := c.formMapper.ToEntity(req)
	// 未指定页面类型时使用最通用的普通表格。
	if item.PageType == "" {
		item.PageType = codeGenPageTypeNormal
	}
	if req.GetLeftTreeConfig() == nil {
		item.LeftTreeConfig = ""
	}
	return item, nil
}
