package admin

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	kratosErrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/adapter/kit"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	queryCompiler "github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/query"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-core/errorsx"
	"github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var biSourceIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// BiSourceCase 管理租户可配置的数据源，并将其接入 BI 查询注册表。
type BiSourceCase struct {
	*biz.BaseCase
	tx             data.Transaction
	sourceRepo     *data.BiSourceRepository
	registry       *bi.Registry
	redactResolver *kit.RedactPolicyResolver
	cardRepo       *data.BiCardRepository
	dashboardRepo  *data.BiDashboardRepository
	versionRepo    *data.BiDashboardVersionRepository
	authorizer     engine.Engine
}

// NewBiSourceCase 创建 BI 数据源配置业务实例。
func NewBiSourceCase(baseCase *biz.BaseCase, tx data.Transaction, sourceRepo *data.BiSourceRepository, registry *bi.Registry, redactResolver *kit.RedactPolicyResolver, cardRepo *data.BiCardRepository, dashboardRepo *data.BiDashboardRepository, versionRepo *data.BiDashboardVersionRepository, authorizer engine.Engine) *BiSourceCase {
	return &BiSourceCase{BaseCase: baseCase, tx: tx, sourceRepo: sourceRepo, registry: registry, redactResolver: redactResolver, cardRepo: cardRepo, dashboardRepo: dashboardRepo, versionRepo: versionRepo, authorizer: authorizer}
}

// List 返回当前访问范围内的静态和可配置数据源目录。
func (c *BiSourceCase) List(ctx context.Context, scope bi.Scope) ([]bi.SourceDescriptor, error) {
	result, err := c.registry.List(ctx, scope)
	if err != nil {
		return nil, err
	}
	query := c.sourceRepo.Query(ctx).BiSource
	var items []*models.BiSource
	items, err = c.sourceRepo.List(ctx,
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
	execution, err := c.compile(ctx, scope, spec)
	if err != nil {
		return bi.QueryResult{}, err
	}
	return c.executeDefinition(ctx, scope, spec, execution, 0, time.Now())
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
	var items []*models.BiSource
	var total int64
	items, total, err = c.sourceRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BiSource, 0, len(items))
	for _, item := range items {
		var response *adminv1.BiSource
		response, err = c.sourceResponse(ctx, item)
		if err != nil {
			return nil, err
		}
		result = append(result, response)
	}
	return &adminv1.PageBiSourceResponse{Sources: result, Total: int32(total)}, nil
}

// GetBiSource 查询当前租户的数据源配置详情。
func (c *BiSourceCase) GetBiSource(ctx context.Context, req *adminv1.GetBiSourceRequest) (*adminv1.BiSource, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var item *models.BiSource
	item, err = c.findSource(ctx, authInfo.TenantId, req.GetId())
	if err != nil {
		return nil, err
	}
	return c.sourceResponse(ctx, item)
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
	return c.sourceResponse(ctx, item)
}

// UpdateBiSource 在源锁内更新连接信息，不允许编码重置或绕过派生卡片同步。
func (c *BiSourceCase) UpdateBiSource(ctx context.Context, req *adminv1.UpdateBiSourceRequest) (*adminv1.BiSource, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	form := req.GetSource()
	if err = c.validateForm(ctx, form); err != nil {
		return nil, err
	}
	var initial *models.BiSource
	initial, err = c.findSource(ctx, authInfo.TenantId, form.GetId())
	if err != nil {
		return nil, err
	}
	if initial.Code != form.GetCode() {
		return nil, errorsx.InvalidArgument("数据源编码创建后不可修改")
	}
	var item *models.BiSource
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		item, err = c.lockSource(txCtx, authInfo.TenantId, initial.Code)
		if err != nil {
			return err
		}
		connectionChanged := item.SourceName != sourceName(form) || item.Driver != form.GetDriver() || item.Host != form.GetHost() || item.Port != form.GetPort() || item.DatabaseName != form.GetDatabaseName() || item.Username != form.GetUsername() || item.SslMode != form.GetSslMode() || item.SchemaName != form.GetSchemaName()
		if connectionChanged {
			if err = c.requireUnusedSource(txCtx, item); err != nil {
				return err
			}
			item.Status = _const.STATUS_STATUS_DISABLE
			var previous dto.ConfiguredSourceSchema
			if err = json.Unmarshal([]byte(item.SchemaJSON), &previous); err != nil {
				return err
			}
			var payload []byte
			payload, err = marshalConfiguredSourceModel(dto.ConfiguredSourceSchema{ModelRevision: previous.ModelRevision + 1})
			if err != nil {
				return err
			}
			item.SchemaJSON = string(payload)
			item.TableName_, item.TenantColumn = "", ""
		}
		if item.MaxRows != normalizeMaxRows(form.GetMaxRows()) && hasConfiguredSourceModel(item.SchemaJSON) {
			var model dto.ConfiguredSourceSchema
			if err = json.Unmarshal([]byte(item.SchemaJSON), &model); err != nil {
				return err
			}
			model.ModelRevision++
			item.MaxRows = normalizeMaxRows(form.GetMaxRows())
			_, err = c.synchronizeModel(txCtx, item, model, 0, true)
			if err != nil {
				return err
			}
			var payload []byte
			payload, err = marshalConfiguredSourceModel(model)
			if err != nil {
				return err
			}
			item.SchemaJSON = string(payload)
		}
		item.Name = form.GetName()
		item.Description = form.GetDescription()
		item.SourceName = sourceName(form)
		item.Driver = form.GetDriver()
		item.Host = form.GetHost()
		item.Port = form.GetPort()
		item.DatabaseName = form.GetDatabaseName()
		item.Username = form.GetUsername()
		item.SslMode = form.GetSslMode()
		item.SchemaName = form.GetSchemaName()
		item.MaxRows = normalizeMaxRows(form.GetMaxRows())
		if form.GetPassword().GetText() != "" {
			item.Password = form.GetPassword().GetText()
		}
		item.Version++
		item.UpdatedBy = authInfo.UserId
		item.UpdatedAt = time.Now()
		return c.sourceRepo.UpdateByID(txCtx, item)
	})
	if err != nil {
		return nil, err
	}
	return c.sourceResponse(ctx, item)
}

// DeleteBiSource 在源锁内检查现存卡片和当前发布版本，保留软删除编码唯一约束。
func (c *BiSourceCase) DeleteBiSource(ctx context.Context, req *adminv1.DeleteBiSourceRequest) (*adminv1.DeleteBiSourceResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var initial *models.BiSource
	initial, err = c.findSource(ctx, authInfo.TenantId, req.GetId())
	if err != nil {
		return nil, err
	}
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		var item *models.BiSource
		item, err = c.lockSource(txCtx, authInfo.TenantId, initial.Code)
		if err != nil {
			return err
		}
		if err = c.requireUnusedSource(txCtx, item); err != nil {
			return err
		}
		return c.sourceRepo.DeleteByIDs(txCtx, []int64{item.ID})
	})
	if err != nil {
		return nil, err
	}
	return &adminv1.DeleteBiSourceResponse{}, nil
}

// SetBiSourceStatus 在源锁内修改可用性并增加并发版本，不覆盖并发提交的公共模型。
func (c *BiSourceCase) SetBiSourceStatus(ctx context.Context, req *adminv1.SetBiSourceStatusRequest) (*adminv1.BiSource, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetStatus() != _const.STATUS_STATUS_ENABLE && req.GetStatus() != _const.STATUS_STATUS_DISABLE {
		return nil, errorsx.InvalidArgument("数据源状态无效")
	}
	var initial *models.BiSource
	initial, err = c.findSource(ctx, authInfo.TenantId, req.GetId())
	if err != nil {
		return nil, err
	}
	var item *models.BiSource
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		item, err = c.lockSource(txCtx, authInfo.TenantId, initial.Code)
		if err != nil {
			return err
		}
		if req.GetStatus() == _const.STATUS_STATUS_ENABLE && !hasConfiguredSourceModel(item.SchemaJSON) {
			return errorsx.InvalidArgument("请先完成数据源表结构和语义模型配置")
		}
		item.Status = req.GetStatus()
		item.Version++
		item.UpdatedBy = authInfo.UserId
		item.UpdatedAt = time.Now()
		return c.sourceRepo.UpdateByID(txCtx, item)
	})
	if err != nil {
		return nil, err
	}
	return c.sourceResponse(ctx, item)
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
	slices.SortFunc(items, func(left, right *adminv1.BiSourceDatabase) int { return cmp.Compare(left.Name, right.Name) })
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
		return &adminv1.TestBiSourceResponse{Success: false, Message: testSourceFailureMessage(err)}, nil
	}
	defer database.Close()
	return &adminv1.TestBiSourceResponse{Success: true, Message: "数据源连接测试通过"}, nil
}

// ListBiSourceTables 查询当前租户数据源中的表和字段结构。
func (c *BiSourceCase) ListBiSourceTables(ctx context.Context, req *adminv1.ListBiSourceTablesRequest) (*adminv1.ListBiSourceTablesResponse, error) {
	err := c.requireModelPermission(ctx)
	if err != nil {
		return nil, err
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
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

// PageBiSourceTableData 分页查询当前租户数据源中指定表的数据。
func (c *BiSourceCase) PageBiSourceTableData(ctx context.Context, req *adminv1.PageBiSourceTableDataRequest) (*adminv1.PageBiSourceTableDataResponse, error) {
	err := c.requireModelPermission(ctx)
	if err != nil {
		return nil, err
	}
	var authInfo *authData.UserTokenPayload
	authInfo, err = c.GetAuthInfo(ctx)
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
	var table *adminv1.BiSourceSchemaTable
	table, err = findSourceTableDataTarget(tables, req.GetTableKey(), req.GetTenantColumn())
	if err != nil {
		return nil, err
	}
	return pageSourceTableData(ctx, database, table, req.GetTenantColumn(), authInfo.TenantId, req.GetPageNum(), req.GetPageSize())
}

// SaveBiSourceModel 在同一元数据事务中保存公共模型并同步所有引用卡片。
func (c *BiSourceCase) SaveBiSourceModel(ctx context.Context, req *adminv1.SaveBiSourceModelRequest) (*adminv1.BiSource, error) {
	prepared, candidate, err := c.prepareModel(ctx, req)
	if err != nil {
		return nil, err
	}
	var item *models.BiSource
	var impact *adminv1.BiModelImpact
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		item, err = c.lockSource(txCtx, prepared.TenantID, prepared.Code)
		if err != nil {
			return err
		}
		if item.Version != req.GetExpectedSourceVersion() {
			return errorsx.Conflict("数据源版本已变化，请刷新后重新预览")
		}
		impact, err = c.synchronizeModel(txCtx, item, candidate, 0, true)
		if err != nil {
			return err
		}
		var payload []byte
		payload, err = marshalConfiguredSourceModel(candidate)
		if err != nil {
			return err
		}
		base := candidate.Tables[findModelTable(candidate.Tables, candidate.BaseTable)]
		item.SchemaName, item.TableName_, item.TenantColumn = base.SchemaName, base.Name, base.TenantColumn
		item.SchemaJSON = string(payload)
		item.Version++
		item.Status = _const.STATUS_STATUS_ENABLE
		item.UpdatedAt = time.Now()
		var authInfo *authData.UserTokenPayload
		authInfo, err = c.GetAuthInfo(txCtx)
		if err != nil {
			return err
		}
		item.UpdatedBy = authInfo.UserId
		return c.sourceRepo.UpdateByID(txCtx, item)
	})
	if err != nil {
		return nil, err
	}
	result := toBiSource(item)
	result.SyncSummary = impact
	return result, nil
}

// findEnabledSource 查询当前租户已启用且已配置模型的数据源。
func (c *BiSourceCase) findEnabledSource(ctx context.Context, tenantID int64, code string) (*models.BiSource, dto.ConfiguredSourceSchema, error) {
	query := c.sourceRepo.Query(ctx).BiSource
	item, err := c.sourceRepo.Find(ctx,
		repository.Where(query.TenantID.Eq(tenantID)),
		repository.Where(query.Code.Eq(code)),
		repository.Where(query.Status.Eq(_const.STATUS_STATUS_ENABLE)),
		repository.Clauses(clause.Locking{Strength: "SHARE"}),
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

// findSource 按租户和编号查找数据源记录，记录不存在时返回资源未找到错误。
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
		fields := make(map[string]bi.FieldType, len(metadata.GetColumns()))
		for _, column := range metadata.GetColumns() {
			fields[column.GetName()] = sourceFieldType(column.GetDataType())
		}
		model.Tables = append(model.Tables, dto.ConfiguredSourceTable{Fields: fields, Key: table.GetKey(), SchemaName: metadata.GetSchemaName(), Name: metadata.GetName(), TenantColumn: table.GetTenantColumn(), X: table.GetX(), Y: table.GetY()})
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
	metricNames := make(map[string]bool, len(input.GetMeasures()))
	fields := model.Tables[findModelTable(model.Tables, model.BaseTable)].Fields
	for _, measure := range input.GetMeasures() {
		if !biSourceIdentifierPattern.MatchString(measure.GetCode()) || measure.GetName() == "" || strings.TrimSpace(measure.GetName()) != measure.GetName() || len([]rune(measure.GetName())) > 128 || len([]rune(measure.GetUnit())) > 32 || measure.GetType() != "number" {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("公共指标编码、名称、单位或类型无效")
		}
		if _, exists := fieldCodes[measure.GetCode()]; exists || metricNames[measure.GetName()] {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("同一数据源的指标编码和名称不能重复")
		}
		expression, err := queryCompiler.Parse(measure.GetExpression(), fields)
		if err != nil {
			return dto.ConfiguredSourceSchema{}, errorsx.InvalidArgument("公共指标公式无效: " + measure.GetCode()).WithCause(err)
		}
		fieldCodes[measure.GetCode()] = struct{}{}
		metricNames[measure.GetName()] = true
		model.Measures = append(model.Measures, dto.ConfiguredSourceMeasure{Code: measure.GetCode(), Name: measure.GetName(), Type: bi.FieldTypeNumber, Expression: expression.Canonical(), Unit: measure.GetUnit()})
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
	if len(encoded) > 4<<20 {
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

// validateSourceField 校验语义模型中单个字段定义：编码、名称、物理列必填且标识符合法，类型限定在支持的取值内。
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

// normalizeMaxRows 将数据源单次查询的返回行数收敛到安全区间，缺省 100 行、上限 1000 行，避免误配导致全表拉取。
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

// supportedSourceDriver 判断独立数据库连接的驱动类型是否在支持范围内（MySQL、Doris、TiDB、PostgreSQL）。
func supportedSourceDriver(value string) bool {
	switch strings.ToLower(value) {
	case "mysql", "doris", "tidb", "postgres":
		return true
	default:
		return false
	}
}

// sourcePassword 从脱敏存储中还原独立连接的密码；密码未配置或复用服务端客户端时无独立密码。
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

// testSourceFailureMessage 提取连接测试失败的简短原因，避免把完整错误链透出给前端。
func testSourceFailureMessage(err error) string {
	var ex *kratosErrors.Error
	if errors.As(err, &ex) && ex.Message != "" {
		return ex.Message
	}
	return "连接 BI 数据源失败"
}

// sourceResponse 统一限制连接管理出口中的物理模型信息，普通读权限仅得到配置元数据。
func (c *BiSourceCase) sourceResponse(ctx context.Context, item *models.BiSource) (*adminv1.BiSource, error) {
	result := toBiSource(item)
	err := c.requireModelPermission(ctx)
	if kratosErrors.Code(err) == 403 {
		result.SchemaJson = ""
		return result, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// findSourceTableDataTarget 从重新发现的表结构中校验预览目标和租户字段。
func findSourceTableDataTarget(tables []*adminv1.BiSourceSchemaTable, tableKey, tenantColumn string) (*adminv1.BiSourceSchemaTable, error) {
	var table *adminv1.BiSourceSchemaTable
	for _, item := range tables {
		if item.GetKey() == tableKey {
			table = item
			break
		}
	}
	if table == nil {
		return nil, errorsx.InvalidArgument("表结构不存在或已变化")
	}
	if tenantColumn == "" {
		return table, nil
	}
	for _, column := range table.GetColumns() {
		if column.GetName() == tenantColumn {
			return table, nil
		}
	}
	return nil, errorsx.InvalidArgument("租户字段不存在或已变化")
}

// pageSourceTableData 执行白名单表字段上的只读计数和分页查询。
func pageSourceTableData(ctx context.Context, database *configuredDatabase, table *adminv1.BiSourceSchemaTable, tenantColumn string, tenantID, pageNum, pageSize int64) (*adminv1.PageBiSourceTableDataResponse, error) {
	if pageNum < 1 || pageNum > 1000000 || pageSize < 1 || pageSize > 100 {
		return nil, errorsx.InvalidArgument("分页参数无效")
	}
	columns := table.GetColumns()
	if len(columns) == 0 {
		return nil, errorsx.InvalidArgument("表结构没有可预览字段")
	}
	for _, column := range columns {
		if !biSourceIdentifierPattern.MatchString(column.GetName()) {
			return nil, errorsx.InvalidArgument("表字段标识不合法")
		}
	}
	tableName := queryCompiler.QuoteSourceTable(database.driver, table.GetSchemaName(), table.GetName())
	whereSQL := ""
	args := make([]any, 0, 3)
	if tenantColumn != "" {
		whereSQL = " WHERE " + queryCompiler.QuoteSourceIdentifier(database.driver, tenantColumn) + " = " + queryCompiler.Placeholder(database.driver, 1)
		args = append(args, tenantID)
	}
	var total int64
	err := database.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+tableName+whereSQL, args...).Scan(&total) //nolint:forbidigo // 表和字段来自重新发现白名单，租户值使用参数绑定。
	if err != nil {
		return nil, errorsx.Internal("统计 BI 表数据失败").WithCause(err)
	}
	if total > math.MaxInt32 {
		return nil, errorsx.InvalidArgument("数据总数超过接口支持范围")
	}
	orderColumn := columns[0].GetName()
	for _, column := range columns {
		if column.GetPrimaryKey() {
			orderColumn = column.GetName()
			break
		}
	}
	selectParts := make([]string, 0, len(columns))
	for _, column := range columns {
		selectParts = append(selectParts, queryCompiler.QuoteSourceIdentifier(database.driver, column.GetName()))
	}
	limitIndex := len(args) + 1
	offsetIndex := limitIndex + 1
	args = append(args, pageSize, (pageNum-1)*pageSize)
	queryText := "SELECT " + strings.Join(selectParts, ", ") + " FROM " + tableName + whereSQL + " ORDER BY " + queryCompiler.QuoteSourceIdentifier(database.driver, orderColumn) + " LIMIT " + queryCompiler.Placeholder(database.driver, limitIndex) + " OFFSET " + queryCompiler.Placeholder(database.driver, offsetIndex)
	var rows *sql.Rows
	rows, err = database.db.QueryContext(ctx, queryText, args...) //nolint:forbidigo // 表和字段来自重新发现白名单，分页值使用参数绑定。
	if err != nil {
		return nil, errorsx.Internal("查询 BI 表数据失败").WithCause(err)
	}
	defer func() { _ = rows.Close() }()
	result := &adminv1.PageBiSourceTableDataResponse{Columns: columns, Total: int32(total)}
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(values))
		for index := range values {
			pointers[index] = &values[index]
		}
		if err = rows.Scan(pointers...); err != nil {
			return nil, errorsx.Internal("读取 BI 表数据失败").WithCause(err)
		}
		row := &adminv1.BiSourceTableDataRow{Values: make([]string, 0, len(values))}
		for _, value := range values {
			row.Values = append(row.Values, formatConfiguredValue(value))
		}
		result.Rows = append(result.Rows, row)
	}
	if err = rows.Err(); err != nil {
		return nil, errorsx.Internal("遍历 BI 表数据失败").WithCause(err)
	}
	return result, nil
}
