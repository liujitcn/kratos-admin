package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm"
)

var biSourceIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// BiSourceCase 管理租户可配置的数据源，并将其接入 BI 查询注册表。
type BiSourceCase struct {
	*biz.BaseCase
	tx             data.Transaction
	sourceRepo     *data.BiSourceRepository
	registry       *bi.Registry
	redactResolver *kit.RedactPolicyResolver
}

// NewBiSourceCase 创建 BI 数据源配置业务实例。
func NewBiSourceCase(baseCase *biz.BaseCase, tx data.Transaction, sourceRepo *data.BiSourceRepository, registry *bi.Registry, redactResolver *kit.RedactPolicyResolver) *BiSourceCase {
	return &BiSourceCase{BaseCase: baseCase, tx: tx, sourceRepo: sourceRepo, registry: registry, redactResolver: redactResolver}
}

// List 返回当前访问范围内的静态和可配置数据源目录。
func (c *BiSourceCase) List(ctx context.Context, scope bi.Scope) ([]bi.SourceDescriptor, error) {
	var err error
	result, err := c.registry.List(ctx, scope)
	if err != nil {
		return nil, err
	}
	query := c.sourceRepo.Query(ctx).BiSource
	items, err := c.sourceRepo.List(ctx,
		repository.Where(query.TenantID.Eq(scope.TenantID)),
		repository.Where(query.Status.Eq(_const.STATUS_STATUS_ENABLE)),
		repository.Order(query.Code.Asc()),
	)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		var schema dto.ConfiguredSourceSchema
		err = json.Unmarshal([]byte(item.SchemaJSON), &schema)
		if err != nil {
			return nil, fmt.Errorf("解析 BI 数据源 %s 语义配置失败: %w", item.Code, err)
		}
		result = append(result, schema.Descriptor(item.Code, item.Name, item.Description, item.Version, item.MaxRows))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result, nil
}

// Execute 执行静态 Adapter 或可配置语义数据源查询。
func (c *BiSourceCase) Execute(ctx context.Context, scope bi.Scope, spec bi.QuerySpec) (bi.QueryResult, error) {
	if c.registry.Has(spec.SourceCode) {
		return c.registry.Execute(ctx, scope, spec)
	}
	item, schema, err := c.findEnabledSource(ctx, scope.TenantID, spec.SourceCode)
	if err != nil {
		return bi.QueryResult{}, err
	}
	var database *configuredDatabase
	database, err = c.openSource(ctx, item)
	if err != nil {
		return bi.QueryResult{}, err
	}
	defer database.Close()
	return executeConfiguredSource(ctx, database, item, schema, scope, spec)
}

// openSource 校验宿主连接权限后打开数据源，已有配置执行时同样不能绕过权限。
func (c *BiSourceCase) openSource(ctx context.Context, item *models.BiSource) (*configuredDatabase, error) {
	err := c.requireHostClientAccess(ctx, item.Host)
	if err != nil {
		return nil, err
	}
	var password string
	password, err = c.sourcePassword(ctx, item)
	if err != nil {
		return nil, err
	}
	return openConfiguredSource(ctx, c.GormClients, item, password)
}

// requireHostClientAccess 仅允许默认租户使用宿主数据库，普通租户必须提供独立连接凭据。
func (c *BiSourceCase) requireHostClientAccess(ctx context.Context, host string) error {
	if host != "" {
		return nil
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return err
	}
	if authInfo.TenantCode != databaseGorm.DefaultTenantCode {
		return errorsx.PermissionDenied("只有默认租户可以使用服务端数据库客户端")
	}
	return nil
}

// PageBiSource 分页查询当前租户的数据源配置。
func (c *BiSourceCase) PageBiSource(ctx context.Context, req *adminv1.PageBiSourceRequest) (*adminv1.PageBiSourceResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.sourceRepo.Query(ctx).BiSource
	opts := []repository.QueryOption{
		repository.Where(query.TenantID.Eq(authInfo.TenantId)),
		repository.Order(query.UpdatedAt.Desc()),
		repository.Order(query.ID.Desc()),
	}
	if code := strings.TrimSpace(req.GetCode()); code != "" {
		opts = append(opts, repository.Where(query.Code.Like("%"+code+"%")))
	}
	if name := strings.TrimSpace(req.GetName()); name != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+name+"%")))
	}
	if sourceName := strings.TrimSpace(req.GetSourceName()); sourceName != "" {
		opts = append(opts, repository.Where(query.SourceName.Eq(sourceName)))
	}
	if req.Status != nil {
		opts = append(opts, repository.Where(query.Status.Eq(req.GetStatus())))
	}
	items, total, err := c.sourceRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BiSource, 0, len(items))
	for _, item := range items {
		result = append(result, toBiSource(item))
	}
	return &adminv1.PageBiSourceResponse{Sources: result, Total: int32(total)}, nil
}

// GetBiSource 查询当前租户的数据源配置详情。
func (c *BiSourceCase) GetBiSource(ctx context.Context, req *adminv1.GetBiSourceRequest) (*adminv1.BiSource, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	item, err := c.findSource(ctx, authInfo.TenantId, req.GetId())
	if err != nil {
		return nil, err
	}
	return toBiSource(item), nil
}

// CreateBiSource 创建当前租户的数据源配置。
func (c *BiSourceCase) CreateBiSource(ctx context.Context, req *adminv1.CreateBiSourceRequest) (*adminv1.BiSource, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	form := req.GetSource()
	if form == nil {
		return nil, errorsx.InvalidArgument("BI 数据源配置不能为空")
	}
	if err = c.validateForm(ctx, form); err != nil {
		return nil, err
	}
	if c.registry.Has(form.GetCode()) {
		return nil, errorsx.Conflict("数据源编码已被静态 Adapter 占用")
	}
	now := time.Now()
	item := &models.BiSource{
		TenantID: authInfo.TenantId, Code: form.GetCode(), Name: form.GetName(), Description: form.GetDescription(),
		SourceName: sourceName(form), Driver: form.GetDriver(), Host: form.GetHost(), Port: form.GetPort(), DatabaseName: form.GetDatabaseName(), Username: form.GetUsername(), Password: form.GetPassword().GetText(), SslMode: form.GetSslMode(), SchemaName: form.GetSchemaName(),
		SchemaJSON: "{}", Status: _const.STATUS_STATUS_DISABLE, Version: 1, MaxRows: normalizeMaxRows(form.GetMaxRows()),
		CreatedBy: authInfo.UserId, UpdatedBy: authInfo.UserId, CreatedAt: now, UpdatedAt: now,
	}
	if err = c.sourceRepo.Create(ctx, item); err != nil {
		if errorsx.IsDuplicateKey(err) {
			return nil, errorsx.Conflict("当前租户的数据源编码已存在").WithCause(err)
		}
		return nil, err
	}
	return toBiSource(item), nil
}

// UpdateBiSource 更新当前租户的数据源配置。
func (c *BiSourceCase) UpdateBiSource(ctx context.Context, req *adminv1.UpdateBiSourceRequest) (*adminv1.BiSource, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	form := req.GetSource()
	if form == nil || form.GetId() <= 0 {
		return nil, errorsx.InvalidArgument("BI 数据源配置ID无效")
	}
	var item *models.BiSource
	item, err = c.findSource(ctx, authInfo.TenantId, form.GetId())
	if err != nil {
		return nil, err
	}
	if err = c.validateForm(ctx, form); err != nil {
		return nil, err
	}
	if c.registry.Has(form.GetCode()) && form.GetCode() != item.Code {
		return nil, errorsx.Conflict("数据源编码已被静态 Adapter 占用")
	}
	connectionChanged := item.SourceName != sourceName(form) || item.Driver != form.GetDriver() || item.Host != form.GetHost() || item.Port != form.GetPort() || item.DatabaseName != form.GetDatabaseName() || item.Username != form.GetUsername() || item.SslMode != form.GetSslMode() || item.SchemaName != form.GetSchemaName()
	item.Code = form.GetCode()
	item.Name = form.GetName()
	item.Description = form.GetDescription()
	item.SourceName = sourceName(form)
	item.Driver = form.GetDriver()
	item.Host = form.GetHost()
	item.Port = form.GetPort()
	item.DatabaseName = form.GetDatabaseName()
	item.Username = form.GetUsername()
	item.SslMode = form.GetSslMode()
	if password := form.GetPassword().GetText(); password != "" {
		item.Password = password
	}
	item.SchemaName = form.GetSchemaName()
	if connectionChanged {
		item.TableName_ = ""
		item.TenantColumn = ""
		item.SchemaJSON = "{}"
		item.Status = _const.STATUS_STATUS_DISABLE
	}
	item.Version++
	item.MaxRows = normalizeMaxRows(form.GetMaxRows())
	item.UpdatedBy = authInfo.UserId
	item.UpdatedAt = time.Now()
	if err = c.sourceRepo.UpdateByID(ctx, item); err != nil {
		if errorsx.IsDuplicateKey(err) {
			return nil, errorsx.Conflict("当前租户的数据源编码已存在").WithCause(err)
		}
		return nil, err
	}
	return toBiSource(item), nil
}

// DeleteBiSource 删除当前租户的数据源配置。
func (c *BiSourceCase) DeleteBiSource(ctx context.Context, req *adminv1.DeleteBiSourceRequest) (*adminv1.DeleteBiSourceResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var item *models.BiSource
	item, err = c.findSource(ctx, authInfo.TenantId, req.GetId())
	if err != nil {
		return nil, err
	}
	if err = c.sourceRepo.DeleteByIDs(ctx, []int64{item.ID}); err != nil {
		return nil, err
	}
	return &adminv1.DeleteBiSourceResponse{}, nil
}

// SetBiSourceStatus 修改当前租户的数据源启用状态。
func (c *BiSourceCase) SetBiSourceStatus(ctx context.Context, req *adminv1.SetBiSourceStatusRequest) (*adminv1.BiSource, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetStatus() != _const.STATUS_STATUS_ENABLE && req.GetStatus() != _const.STATUS_STATUS_DISABLE {
		return nil, errorsx.InvalidArgument("数据源状态无效")
	}
	var item *models.BiSource
	item, err = c.findSource(ctx, authInfo.TenantId, req.GetId())
	if err != nil {
		return nil, err
	}
	if req.GetStatus() == _const.STATUS_STATUS_ENABLE && !hasConfiguredSourceModel(item.SchemaJSON) {
		return nil, errorsx.InvalidArgument("请先完成数据源表结构和语义模型配置")
	}
	item.Status = req.GetStatus()
	item.UpdatedBy = authInfo.UserId
	item.UpdatedAt = time.Now()
	if err = c.sourceRepo.UpdateByID(ctx, item); err != nil {
		return nil, err
	}
	return toBiSource(item), nil
}

// OptionBiSourceDatabase 仅向默认租户返回服务端已经装配的数据库客户端。
func (c *BiSourceCase) OptionBiSourceDatabase(ctx context.Context, _ *adminv1.OptionBiSourceDatabaseRequest) (*adminv1.OptionBiSourceDatabaseResponse, error) {
	err := c.requireHostClientAccess(ctx, "")
	if err != nil {
		return nil, err
	}
	items := make([]*adminv1.BiSourceDatabase, 0, len(c.GormClients))
	for name, client := range c.GormClients {
		if client == nil {
			continue
		}
		items = append(items, &adminv1.BiSourceDatabase{Name: name, Driver: client.Driver()})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return &adminv1.OptionBiSourceDatabaseResponse{Databases: items}, nil
}

// TestBiSource 测试数据源连接。
func (c *BiSourceCase) TestBiSource(ctx context.Context, req *adminv1.TestBiSourceRequest) (*adminv1.TestBiSourceResponse, error) {
	if req.GetSource() == nil {
		return nil, errorsx.InvalidArgument("BI 数据源配置不能为空")
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	form := req.GetSource()
	if err = c.validateForm(ctx, form); err != nil {
		return nil, err
	}
	password := form.GetPassword().GetText()
	if password == "" && form.GetId() > 0 {
		var item *models.BiSource
		item, err = c.findSource(ctx, authInfo.TenantId, form.GetId())
		if err != nil {
			return nil, err
		}
		password, err = c.sourcePassword(ctx, item)
		if err != nil {
			return nil, err
		}
	}
	var database *configuredDatabase
	database, err = openConfiguredSource(ctx, c.GormClients, &models.BiSource{SourceName: sourceName(form), Driver: form.GetDriver(), Host: form.GetHost(), Port: form.GetPort(), DatabaseName: form.GetDatabaseName(), Username: form.GetUsername(), Password: password, SslMode: form.GetSslMode()}, password)
	if err != nil {
		return &adminv1.TestBiSourceResponse{Message: err.Error()}, nil
	}
	defer database.Close()
	return &adminv1.TestBiSourceResponse{Success: true, Message: "数据源连接测试通过"}, nil
}

// ListBiSourceTables 查询当前租户数据源中的表和字段结构。
func (c *BiSourceCase) ListBiSourceTables(ctx context.Context, req *adminv1.ListBiSourceTablesRequest) (*adminv1.ListBiSourceTablesResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var item *models.BiSource
	item, err = c.findSource(ctx, authInfo.TenantId, req.GetId())
	if err != nil {
		return nil, err
	}
	var database *configuredDatabase
	database, err = c.openSource(ctx, item)
	if err != nil {
		return nil, err
	}
	defer database.Close()
	var tables []*adminv1.BiSourceSchemaTable
	tables, err = discoverSourceTables(ctx, database, item)
	if err != nil {
		return nil, err
	}
	return &adminv1.ListBiSourceTablesResponse{Tables: tables}, nil
}

// SaveBiSourceModel 校验并保存数据源多表语义模型。
func (c *BiSourceCase) SaveBiSourceModel(ctx context.Context, req *adminv1.SaveBiSourceModelRequest) (*adminv1.BiSource, error) {
	if req.GetModel() == nil {
		return nil, errorsx.InvalidArgument("数据源语义模型不能为空")
	}
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var item *models.BiSource
	item, err = c.findSource(ctx, authInfo.TenantId, req.GetId())
	if err != nil {
		return nil, err
	}
	var database *configuredDatabase
	database, err = c.openSource(ctx, item)
	if err != nil {
		return nil, err
	}
	defer database.Close()
	var discovered []*adminv1.BiSourceSchemaTable
	discovered, err = discoverSourceTables(ctx, database, item)
	if err != nil {
		return nil, err
	}
	var model dto.ConfiguredSourceSchema
	model, err = configuredSourceModel(req.GetModel(), discovered)
	if err != nil {
		return nil, err
	}
	var schemaJSON []byte
	schemaJSON, err = marshalConfiguredSourceModel(model)
	if err != nil {
		return nil, err
	}
	baseTable := findSchemaTable(discovered, model.BaseTable)
	item.SchemaName = baseTable.SchemaName
	item.TableName_ = baseTable.Name
	baseIndex := findModelTable(model.Tables, model.BaseTable)
	item.TenantColumn = model.Tables[baseIndex].TenantColumn
	item.SchemaJSON = string(schemaJSON)
	item.Status = _const.STATUS_STATUS_ENABLE
	item.Version++
	item.UpdatedBy = authInfo.UserId
	item.UpdatedAt = time.Now()
	if err = c.sourceRepo.UpdateByID(ctx, item); err != nil {
		return nil, err
	}
	return toBiSource(item), nil
}

// findEnabledSource 查询当前租户已启用且已配置模型的数据源。
func (c *BiSourceCase) findEnabledSource(ctx context.Context, tenantID int64, code string) (*models.BiSource, dto.ConfiguredSourceSchema, error) {
	query := c.sourceRepo.Query(ctx).BiSource
	item, err := c.sourceRepo.Find(ctx,
		repository.Where(query.TenantID.Eq(tenantID)),
		repository.Where(query.Code.Eq(code)),
		repository.Where(query.Status.Eq(_const.STATUS_STATUS_ENABLE)),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, dto.ConfiguredSourceSchema{}, errorsx.ResourceNotFound("BI 数据源不存在或未启用")
		}
		return nil, dto.ConfiguredSourceSchema{}, err
	}
	var schema dto.ConfiguredSourceSchema
	err = json.Unmarshal([]byte(item.SchemaJSON), &schema)
	if err != nil {
		return nil, dto.ConfiguredSourceSchema{}, errorsx.Internal("解析 BI 数据源语义配置失败").WithCause(err)
	}
	if !hasConfiguredSourceModel(item.SchemaJSON) {
		return nil, dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("BI 数据源语义模型未配置")
	}
	return item, schema, nil
}

func (c *BiSourceCase) findSource(ctx context.Context, tenantID, id int64) (*models.BiSource, error) {
	query := c.sourceRepo.Query(ctx).BiSource
	item, err := c.sourceRepo.Find(ctx, repository.Where(query.ID.Eq(id)), repository.Where(query.TenantID.Eq(tenantID)))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("BI 数据源不存在")
		}
		return nil, err
	}
	return item, nil
}

// validateForm 校验连接配置及宿主数据库客户端访问权限。
func (c *BiSourceCase) validateForm(ctx context.Context, form *adminv1.BiSourceForm) error {
	err := c.requireHostClientAccess(ctx, form.GetHost())
	if err != nil {
		return err
	}
	if form.GetSourceName() == "" && form.GetHost() == "" {
		return errorsx.InvalidArgument("请选择服务端数据库客户端或填写独立数据库连接")
	}
	if form.GetSourceName() != "" && c.GormClients[form.GetSourceName()] == nil && form.GetHost() == "" {
		return errorsx.InvalidArgument("服务端数据库客户端不存在")
	}
	if form.GetHost() != "" && !supportedSourceDriver(form.GetDriver()) {
		return errorsx.InvalidArgument("数据库驱动不支持")
	}
	if form.GetHost() != "" && (form.GetPort() <= 0 || form.GetDatabaseName() == "" || form.GetUsername() == "") {
		return errorsx.InvalidArgument("独立数据库连接的主机、端口、数据库名和用户名不能为空")
	}
	if form.GetHost() != "" && form.GetPassword().GetText() == "" && form.GetId() == 0 {
		return errorsx.InvalidArgument("新建独立数据库连接时密码不能为空")
	}
	if form.GetSchemaName() != "" && !biSourceIdentifierPattern.MatchString(form.GetSchemaName()) {
		return errorsx.InvalidArgument("数据库 Schema 不合法")
	}
	return nil
}

// configuredSourceModel 将请求模型解析为数据库已发现且可执行的语义定义。
func configuredSourceModel(input *adminv1.BiSourceModel, discovered []*adminv1.BiSourceSchemaTable) (dto.ConfiguredSourceSchema, error) {
	if input.GetBaseTable() == "" || len(input.GetTables()) == 0 || len(input.GetTables()) > 8 {
		return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("请选择1到8张数据表并指定事实表")
	}
	discoveredByKey := make(map[string]*adminv1.BiSourceSchemaTable, len(discovered))
	for _, table := range discovered {
		discoveredByKey[table.GetKey()] = table
	}
	model := dto.ConfiguredSourceSchema{BaseTable: input.GetBaseTable()}
	selected := make(map[string]*adminv1.BiSourceSchemaTable, len(input.GetTables()))
	columns := make(map[string]map[string]*adminv1.BiSourceSchemaColumn, len(input.GetTables()))
	for _, table := range input.GetTables() {
		metadata := discoveredByKey[table.GetKey()]
		if metadata == nil {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("选中的数据表已不存在，请重新发现表结构")
		}
		if _, exists := selected[table.GetKey()]; exists {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("数据表不能重复选择")
		}
		if table.GetTenantColumn() != "" && findSchemaColumn(metadata, table.GetTenantColumn()) == nil {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("租户隔离字段不属于选中的数据表")
		}
		selected[table.GetKey()] = metadata
		columns[table.GetKey()] = make(map[string]*adminv1.BiSourceSchemaColumn, len(metadata.GetColumns()))
		for _, column := range metadata.GetColumns() {
			columns[table.GetKey()][column.GetName()] = column
		}
		model.Tables = append(model.Tables, dto.ConfiguredSourceTable{Key: table.GetKey(), SchemaName: metadata.GetSchemaName(), Name: metadata.GetName(), TenantColumn: table.GetTenantColumn(), X: table.GetX(), Y: table.GetY()})
	}
	if selected[model.BaseTable] == nil {
		return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("事实表必须包含在选中的数据表中")
	}
	if len(input.GetRelations()) != len(model.Tables)-1 {
		return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("每张维度表都必须与事实表建立一条关系")
	}
	parents := make(map[string]int, len(model.Tables))
	children := make(map[string][]string, len(model.Tables))
	for _, relation := range input.GetRelations() {
		leftTable := selected[relation.GetLeftTable()]
		rightTable := selected[relation.GetRightTable()]
		leftColumn := columns[relation.GetLeftTable()][relation.GetLeftColumn()]
		rightColumn := columns[relation.GetRightTable()][relation.GetRightColumn()]
		if leftTable == nil || rightTable == nil || relation.GetLeftTable() == relation.GetRightTable() || leftColumn == nil || rightColumn == nil {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("表关联必须引用已选表中的有效字段")
		}
		if !rightColumn.GetUnique() || !compatibleSourceTypes(leftColumn.GetDataType(), rightColumn.GetDataType()) {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("维度表关联字段必须唯一且与事实表字段类型兼容")
		}
		joinType := strings.ToLower(relation.GetJoinType())
		if joinType != "inner" && joinType != "left" {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("只支持 inner 和 left 关联")
		}
		parents[relation.GetRightTable()]++
		if parents[relation.GetRightTable()] > 1 || relation.GetRightTable() == model.BaseTable {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("关系图必须是以事实表为根的树")
		}
		children[relation.GetLeftTable()] = append(children[relation.GetLeftTable()], relation.GetRightTable())
		model.Relations = append(model.Relations, dto.ConfiguredSourceRelation{LeftTable: relation.GetLeftTable(), LeftColumn: relation.GetLeftColumn(), RightTable: relation.GetRightTable(), RightColumn: relation.GetRightColumn(), JoinType: joinType})
	}
	reachable := map[string]bool{model.BaseTable: true}
	for changed := true; changed; {
		changed = false
		for parent, targets := range children {
			if !reachable[parent] {
				continue
			}
			for _, target := range targets {
				if !reachable[target] {
					reachable[target] = true
					changed = true
				}
			}
		}
	}
	if len(reachable) != len(selected) {
		return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("所有维度表都必须从事实表连通")
	}
	fieldCodes := make(map[string]struct{}, len(input.GetDimensions())+len(input.GetMeasures()))
	for _, dimension := range input.GetDimensions() {
		field := columns[dimension.GetTable()][dimension.GetColumn()]
		fieldType := bi.FieldType(strings.ToLower(dimension.GetType()))
		if selected[dimension.GetTable()] == nil || field == nil || validateSourceField(dimension.GetCode(), dimension.GetName(), dimension.GetColumn(), fieldType) != nil || sourceFieldType(field.GetDataType()) != fieldType {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("语义维度必须映射到已选表中的同类型字段")
		}
		if _, exists := fieldCodes[dimension.GetCode()]; exists {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("语义字段编码不能重复")
		}
		fieldCodes[dimension.GetCode()] = struct{}{}
		model.Dimensions = append(model.Dimensions, dto.ConfiguredSourceDimension{Code: dimension.GetCode(), Name: dimension.GetName(), Type: fieldType, Filterable: dimension.GetFilterable(), Table: dimension.GetTable(), Column: dimension.GetColumn()})
	}
	if len(input.GetMeasures()) == 0 {
		return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("至少配置一个语义指标")
	}
	for _, measure := range input.GetMeasures() {
		field := columns[measure.GetTable()][measure.GetColumn()]
		if measure.GetTable() != model.BaseTable || field == nil || validateSourceField(measure.GetCode(), measure.GetName(), measure.GetColumn(), bi.FieldTypeNumber) != nil {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("指标必须映射到事实表字段")
		}
		if _, exists := fieldCodes[measure.GetCode()]; exists {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("语义字段编码不能重复")
		}
		if len(measure.GetAggregations()) == 0 {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("指标至少需要一个聚合函数")
		}
		aggregations := make([]string, 0, len(measure.GetAggregations()))
		for _, aggregation := range measure.GetAggregations() {
			aggregation = strings.ToLower(aggregation)
			if !supportedAggregation(aggregation) || (aggregation != "count" && sourceFieldType(field.GetDataType()) != bi.FieldTypeNumber) {
				return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("指标聚合函数与数据库字段类型不匹配")
			}
			aggregations = append(aggregations, aggregation)
		}
		fieldCodes[measure.GetCode()] = struct{}{}
		model.Measures = append(model.Measures, dto.ConfiguredSourceMeasure{Code: measure.GetCode(), Name: measure.GetName(), Type: bi.FieldTypeNumber, Aggregations: aggregations, Table: measure.GetTable(), Column: measure.GetColumn()})
	}
	return model, nil
}

// hasConfiguredSourceModel 判断数据源是否包含可执行的模型定义。
func hasConfiguredSourceModel(schemaJSON string) bool {
	var model dto.ConfiguredSourceSchema
	if json.Unmarshal([]byte(schemaJSON), &model) != nil {
		return false
	}
	return model.BaseTable != "" && len(model.Tables) > 0 && len(model.Measures) > 0
}

// marshalConfiguredSourceModel 序列化语义模型并限制数据库文本字段大小。
func marshalConfiguredSourceModel(model dto.ConfiguredSourceSchema) ([]byte, error) {
	encoded, err := json.Marshal(model)
	if err != nil {
		return nil, errorsx.Internal("生成数据源语义模型失败").WithCause(err)
	}
	if len(encoded) > 50000 {
		return nil, errorsx.InvalidArgument("数据源语义模型超过大小限制")
	}
	return encoded, nil
}

// findSchemaTable 按发现接口返回的结构标识查询表结构。
func findSchemaTable(tables []*adminv1.BiSourceSchemaTable, key string) *adminv1.BiSourceSchemaTable {
	for _, table := range tables {
		if table.GetKey() == key {
			return table
		}
	}
	return nil
}

// findModelTable 查询语义模型中指定表的位置。
func findModelTable(tables []dto.ConfiguredSourceTable, key string) int {
	for index, table := range tables {
		if table.Key == key {
			return index
		}
	}
	return -1
}

// findSchemaColumn 按字段名查询数据库列元数据。
func findSchemaColumn(table *adminv1.BiSourceSchemaTable, name string) *adminv1.BiSourceSchemaColumn {
	for _, column := range table.GetColumns() {
		if column.GetName() == name {
			return column
		}
	}
	return nil
}

// sourceFieldType 将数据库类型映射为 BI 字段类型。
func sourceFieldType(value string) bi.FieldType {
	value = strings.ToLower(value)
	switch {
	case strings.Contains(value, "int"), strings.Contains(value, "decimal"), strings.Contains(value, "numeric"), strings.Contains(value, "real"), strings.Contains(value, "double"), strings.Contains(value, "float"):
		return bi.FieldTypeNumber
	case strings.Contains(value, "date"), strings.Contains(value, "time"):
		return bi.FieldTypeDate
	default:
		return bi.FieldTypeString
	}
}

// compatibleSourceTypes 判断关联字段是否属于相同的数据库类型类别。
func compatibleSourceTypes(left, right string) bool {
	return sourceFieldType(left) == sourceFieldType(right)
}

func validateSourceField(code, name, column string, fieldType bi.FieldType) error {
	if code == "" || name == "" || column == "" {
		return errorsx.InvalidArgument("字段编码、名称和物理列不能为空")
	}
	if !biSourceIdentifierPattern.MatchString(code) || !biSourceIdentifierPattern.MatchString(column) {
		return errorsx.InvalidArgument("字段编码或物理列名不合法")
	}
	if fieldType != bi.FieldTypeString && fieldType != bi.FieldTypeNumber && fieldType != bi.FieldTypeDate {
		return errorsx.InvalidArgument("字段类型不支持")
	}
	return nil
}

func supportedAggregation(value string) bool {
	switch strings.ToLower(value) {
	case "count", "sum", "avg", "min", "max":
		return true
	default:
		return false
	}
}

func normalizeMaxRows(maxRows int32) int32 {
	if maxRows <= 0 {
		return 100
	}
	if maxRows > 1000 {
		return 1000
	}
	return maxRows
}

// sourceName 返回服务端命名客户端编码，独立连接不绑定服务端客户端。
func sourceName(form *adminv1.BiSourceForm) string {
	if form.GetSourceName() != "" {
		return form.GetSourceName()
	}
	if form.GetHost() == "" {
		return form.GetCode()
	}
	return ""
}

func supportedSourceDriver(value string) bool {
	switch strings.ToLower(value) {
	case "mysql", "doris", "tidb", "postgres":
		return true
	default:
		return false
	}
}

func (c *BiSourceCase) sourcePassword(ctx context.Context, item *models.BiSource) (string, error) {
	if item.Password == "" || item.Host == "" {
		return "", nil
	}
	if c.redactResolver == nil {
		return "", errorsx.Internal("BI 数据源密码解密服务未初始化")
	}
	password, err := c.redactResolver.RevealStorageField(ctx, "bi_source", "password", item.ID)
	if err != nil {
		return "", errorsx.Internal("读取 BI 数据源密码失败").WithCause(err)
	}
	return password, nil
}

// toBiSource 将数据源模型转换为管理接口响应。
func toBiSource(item *models.BiSource) *adminv1.BiSource {
	return &adminv1.BiSource{
		Id: item.ID, TenantId: item.TenantID, Code: item.Code, Name: item.Name, Description: item.Description,
		SourceName: item.SourceName, Driver: item.Driver, Host: item.Host, Port: item.Port, DatabaseName: item.DatabaseName, Username: item.Username, PasswordConfigured: item.Password != "", SslMode: item.SslMode, SchemaName: item.SchemaName, TableName: item.TableName_, TenantColumn: item.TenantColumn,
		SchemaJson: item.SchemaJSON, Status: item.Status, Version: item.Version, MaxRows: item.MaxRows, ModelConfigured: hasConfiguredSourceModel(item.SchemaJSON), TableCount: configuredSourceTableCount(item.SchemaJSON),
		CreatedAt: item.CreatedAt.Format(time.RFC3339), UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
	}
}

// configuredSourceTableCount 返回语义模型选中的表数量。
func configuredSourceTableCount(schemaJSON string) int32 {
	var model dto.ConfiguredSourceSchema
	if json.Unmarshal([]byte(schemaJSON), &model) != nil {
		return 0
	}
	return int32(len(model.Tables))
}
