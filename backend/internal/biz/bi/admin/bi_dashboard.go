package admin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"time"

	"github.com/liujitcn/gorm-kit/repository"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	dataAccess "github.com/liujitcn/kratos-admin/backend/internal/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-core/errorsx"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	biDashboardStatusDraft      int32 = 1
	biDashboardStatusPublished  int32 = 2
	biDashboardVersionDraft     int32 = 1
	biDashboardVersionPublished int32 = 2
	biDashboardMaxCards               = 20
	biDashboardPermissionView   int32 = 1
	biDashboardPermissionEdit   int32 = 2
)

// BiDashboardCase 管理卡片组合、草稿和发布快照。
type BiDashboardCase struct {
	*biz.BaseCase
	tx            data.Transaction
	dashboardRepo *data.BiDashboardRepository
	versionRepo   *data.BiDashboardVersionRepository
	cardRepo      *data.BiCardRepository
	biQueryCase   *BiQueryCase
}

// NewBiDashboardCase 创建仪表盘组合业务实例。
func NewBiDashboardCase(baseCase *biz.BaseCase, tx data.Transaction, dashboardRepo *data.BiDashboardRepository, versionRepo *data.BiDashboardVersionRepository, cardRepo *data.BiCardRepository, biQueryCase *BiQueryCase) *BiDashboardCase {
	return &BiDashboardCase{BaseCase: baseCase, tx: tx, dashboardRepo: dashboardRepo, versionRepo: versionRepo, cardRepo: cardRepo, biQueryCase: biQueryCase}
}

// PageBiDashboard 返回自己的草稿和当前租户、项目内已发布的业务仪表盘。
func (c *BiDashboardCase) PageBiDashboard(ctx context.Context, req *adminv1.PageBiDashboardRequest) (*adminv1.PageBiDashboardResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	query := c.dashboardRepo.Query(ctx).BiDashboard
	opts := []repository.QueryOption{repository.Where(field.Or(query.OwnerID.Eq(authInfo.UserId), query.PublishedVersion.Gt(0))), repository.Order(query.UpdatedAt.Desc(), query.ID.Desc())}
	if !hasAllProjectsScope(authInfo) {
		opts = append(opts, repository.Where(query.ProjectID.In(allowedProjectIDs(authInfo)...)))
	}
	if req.GetName() != "" {
		opts = append(opts, repository.Where(query.Name.Like("%"+req.GetName()+"%")))
	}
	var items []*models.BiDashboard
	var total int64
	items, total, err = c.dashboardRepo.Page(ctx, req.GetPageNum(), req.GetPageSize(), opts...)
	if err != nil {
		return nil, err
	}
	result := make([]*adminv1.BiDashboard, 0, len(items))
	for _, item := range items {
		current := toBiDashboard(item, "")
		current.RequiresRepublish, err = c.requiresRepublish(ctx, item)
		if err != nil {
			return nil, err
		}
		current.CanEdit = item.OwnerID == authInfo.UserId
		result = append(result, current)
	}
	return &adminv1.PageBiDashboardResponse{Dashboards: result, Total: int32(total)}, nil
}

// GetBiDashboard 仅向所有者返回编辑草稿，其他访问者读取当前发布快照。
func (c *BiDashboardCase) GetBiDashboard(ctx context.Context, req *adminv1.GetBiDashboardRequest) (*adminv1.BiDashboard, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var item *models.BiDashboard
	item, err = c.findAccessibleDashboard(ctx, authInfo, req.GetId(), biDashboardPermissionView)
	if err != nil {
		return nil, err
	}
	number := item.PublishedVersion
	if item.OwnerID == authInfo.UserId {
		number = item.DraftVersion
	}
	var version *models.BiDashboardVersion
	version, err = c.findVersion(ctx, item.ID, number)
	if err != nil {
		return nil, err
	}
	result := toBiDashboard(item, version.SchemaJSON)
	result.RequiresRepublish, err = c.requiresRepublish(ctx, item)
	if err != nil {
		return nil, err
	}
	result.CanEdit = item.OwnerID == authInfo.UserId
	return result, nil
}

// SaveBiDashboard 保存卡片引用与布局，允许空草稿并通过版本号拒绝覆盖其他编辑。
func (c *BiDashboardCase) SaveBiDashboard(ctx context.Context, req *adminv1.SaveBiDashboardRequest) (*adminv1.BiDashboard, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	input := req.GetDashboard()
	if input.GetName() == "" || len([]rune(input.GetName())) > 128 || input.GetCode() == "" || len(input.GetCode()) > 128 {
		return nil, errorsx.InvalidArgument("请填写有效的仪表盘名称和编码")
	}
	projectID := input.GetProjectId()
	var result *models.BiDashboard
	var schemaJSON string
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if input.GetId() > 0 {
			result, err = c.findAccessibleDashboard(txCtx, authInfo, input.GetId(), biDashboardPermissionEdit)
			if err != nil {
				return err
			}
			if input.GetDraftVersion() != result.DraftVersion {
				return errorsx.Conflict("仪表盘草稿版本已变化，请刷新后重试")
			}
			projectID = result.ProjectID
		} else {
			projectID, err = projectIDForAuth(authInfo)
			if err != nil {
				return err
			}
			result = &models.BiDashboard{TenantID: authInfo.TenantId, ProjectID: projectID, OwnerID: authInfo.UserId, Code: input.GetCode(), CreatedBy: authInfo.UserId, CreatedAt: time.Now()}
		}
		var layout *dto.DashboardSchema
		layout, err = c.resolveSchema(txCtx, authInfo, projectID, input.GetSchemaJson(), false)
		if err != nil {
			return err
		}
		layout.Name = input.GetName()
		var payload []byte
		payload, err = json.Marshal(layout)
		if err != nil {
			return errorsx.Internal("保存仪表盘布局失败").WithCause(err)
		}
		schemaJSON = string(payload)
		previous := result.DraftVersion
		result.Name = input.GetName()
		result.Description = input.GetDescription()
		result.Status = biDashboardStatusDraft
		result.DraftVersion++
		result.UpdatedBy = authInfo.UserId
		result.UpdatedAt = time.Now()
		if result.ID == 0 {
			err = c.dashboardRepo.Create(txCtx, result)
		} else {
			var affected int64
			affected, err = dataAccess.UpdateBiDashboardState(txCtx, c.dashboardRepo, result, previous)
			if err == nil && affected != 1 {
				return errorsx.Conflict("仪表盘草稿版本已变化，请刷新后重试")
			}
		}
		if err != nil {
			return err
		}
		return dataAccess.CreateBiDashboardDraft(txCtx, c.versionRepo, &models.BiDashboardVersion{DashboardID: result.ID, TenantID: authInfo.TenantId, Version: result.DraftVersion, SchemaJSON: schemaJSON, SchemaHash: hashSchema(schemaJSON), Status: biDashboardVersionDraft, CreatedBy: authInfo.UserId, CreatedAt: time.Now()})
	})
	if err != nil {
		return nil, err
	}
	response := toBiDashboard(result, schemaJSON)
	response.RequiresRepublish, err = c.requiresRepublish(ctx, result)
	if err != nil {
		return nil, err
	}
	response.CanEdit = true
	return response, nil
}

// PublishBiDashboard 固定本次卡片配置与查询定义，后续卡片编辑不影响已发布页面。
func (c *BiDashboardCase) PublishBiDashboard(ctx context.Context, req *adminv1.PublishBiDashboardRequest) (*adminv1.BiDashboard, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var initial *models.BiDashboard
	initial, err = c.findAccessibleDashboard(ctx, authInfo, req.GetId(), biDashboardPermissionEdit)
	if err != nil {
		return nil, err
	}
	var initialVersion *models.BiDashboardVersion
	initialVersion, err = c.findVersion(ctx, initial.ID, req.GetVersion())
	if err != nil {
		return nil, err
	}
	var result *models.BiDashboard
	var schemaJSON string
	err = c.tx.Transaction(ctx, func(txCtx context.Context) error {
		if err = c.lockPublishInputs(txCtx, authInfo.TenantId, initialVersion.SchemaJSON); err != nil {
			return err
		}
		result, err = c.findAccessibleDashboard(txCtx, authInfo, req.GetId(), biDashboardPermissionEdit)
		if err != nil {
			return err
		}
		if result.DraftVersion != req.GetVersion() {
			return errorsx.Conflict("仪表盘草稿版本已变化")
		}
		var version *models.BiDashboardVersion
		version, err = c.findVersion(txCtx, result.ID, result.DraftVersion)
		if err != nil {
			return err
		}
		if version.Status != biDashboardVersionDraft {
			return errorsx.Conflict("当前版本已经发布")
		}
		var layout *dto.DashboardSchema
		layout, err = c.resolveSchema(txCtx, authInfo, result.ProjectID, version.SchemaJSON, true)
		if err != nil {
			return err
		}
		if len(layout.Cards) == 0 {
			return errorsx.InvalidArgument("请先加入至少一张数据卡片")
		}
		var payload []byte
		payload, err = json.Marshal(layout)
		if err != nil {
			return errorsx.Internal("保存发布快照失败").WithCause(err)
		}
		schemaJSON = string(payload)
		version.SchemaJSON = schemaJSON
		version.SchemaHash = hashSchema(schemaJSON)
		version.Status = biDashboardVersionPublished
		version.PublishedBy = authInfo.UserId
		version.PublishedAt = new(time.Now())
		var affected int64
		affected, err = dataAccess.PublishBiDashboardVersion(txCtx, c.versionRepo, version, biDashboardVersionDraft)
		if err != nil {
			return err
		}
		if affected != 1 {
			return errorsx.Conflict("当前版本已经发布")
		}
		result.Status = biDashboardStatusPublished
		result.PublishedVersion = result.DraftVersion
		result.UpdatedAt = time.Now()
		result.UpdatedBy = authInfo.UserId
		affected, err = dataAccess.UpdateBiDashboardState(txCtx, c.dashboardRepo, result, req.GetVersion())
		if err != nil {
			return err
		}
		if affected != 1 {
			return errorsx.Conflict("仪表盘草稿版本已变化")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	response := toBiDashboard(result, schemaJSON)
	response.RequiresRepublish, err = c.requiresRepublish(ctx, result)
	if err != nil {
		return nil, err
	}
	response.CanEdit = true
	return response, nil
}

// DeleteBiDashboard 删除自己的仪表盘，入口配置在读取时过滤失效引用。
func (c *BiDashboardCase) DeleteBiDashboard(ctx context.Context, req *adminv1.DeleteBiDashboardRequest) (*adminv1.DeleteBiDashboardResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var item *models.BiDashboard
	item, err = c.findAccessibleDashboard(ctx, authInfo, req.GetId(), biDashboardPermissionEdit)
	if err != nil {
		return nil, err
	}
	query := c.dashboardRepo.Query(ctx).BiDashboard
	if err = c.dashboardRepo.Delete(ctx, repository.Where(query.ID.Eq(item.ID))); err != nil {
		return nil, err
	}
	return &adminv1.DeleteBiDashboardResponse{}, nil
}

// GetBiDashboardData 按访问者当前权限执行发布快照内的卡片查询。
func (c *BiDashboardCase) GetBiDashboardData(ctx context.Context, req *adminv1.GetBiDashboardDataRequest) (*adminv1.BiDashboardDataResponse, error) {
	authInfo, err := c.GetAuthInfo(ctx)
	if err != nil {
		return nil, err
	}
	var item *models.BiDashboard
	item, err = c.findAccessibleDashboard(ctx, authInfo, req.GetId(), biDashboardPermissionView)
	if err != nil {
		return nil, err
	}
	number := req.GetVersion()
	if number == 0 {
		number = item.PublishedVersion
	}
	if number == 0 {
		return nil, errorsx.InvalidArgument("仪表盘尚未发布")
	}
	var version *models.BiDashboardVersion
	version, err = c.findVersion(ctx, item.ID, number)
	if err != nil {
		return nil, err
	}
	if version.Status != biDashboardVersionPublished {
		return nil, errorsx.PermissionDenied("只能查看已发布的仪表盘版本")
	}
	var layout *dto.DashboardSchema
	layout, err = parseDashboardSchema(version.SchemaJSON)
	if err != nil {
		return nil, err
	}
	scope := bi.Scope{TenantID: authInfo.TenantId, ProjectID: item.ProjectID, UserID: authInfo.UserId, Terminal: "admin"}
	if err = c.validatePublishedExecutions(ctx, scope, layout); err != nil {
		return nil, err
	}
	results := make([]*adminv1.BiDashboardCardResult, 0, len(layout.Cards))
	for _, placed := range layout.Cards {
		if placed.Query == nil {
			return nil, errorsx.Internal("已发布卡片缺少查询快照")
		}
		var spec bi.QuerySpec
		spec, _, err = c.biQueryCase.validate(ctx, scope, *placed.Query)
		if err != nil {
			return nil, err
		}
		var value bi.QueryResult
		value, err = c.biQueryCase.execute(ctx, scope, spec, &dto.BiQueryRecordContext{Kind: dto.QueryKindDashboard, CardID: placed.CardID, Execution: placed.Execution, DashboardID: item.ID, DashboardVersion: number, WidgetID: placed.ID})
		if err != nil {
			return nil, err
		}
		results = append(results, &adminv1.BiDashboardCardResult{ItemId: placed.ID, Result: toProtoQueryResult(value)})
	}
	if err = c.validatePublishedExecutions(ctx, scope, layout); err != nil {
		return nil, err
	}
	return &adminv1.BiDashboardDataResponse{SchemaJson: toBiDashboard(item, version.SchemaJSON).SchemaJson, Cards: results}, nil
}

// findAccessibleDashboard 限制项目与草稿所有权（租户隔离由数据隔离回调注入），发布页面沿用项目与菜单授权。
func (c *BiDashboardCase) findAccessibleDashboard(ctx context.Context, authInfo *authData.UserTokenPayload, id int64, permission int32) (*models.BiDashboard, error) {
	query := c.dashboardRepo.Query(ctx).BiDashboard
	opts := []repository.QueryOption{repository.Where(query.ID.Eq(id))}
	if permission == biDashboardPermissionEdit {
		opts = append(opts, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
	}
	item, err := c.dashboardRepo.Find(ctx, opts...)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ResourceNotFound("仪表盘不存在")
	}
	if err != nil {
		return nil, err
	}
	if !projectAllowed(authInfo, item.ProjectID) {
		return nil, errorsx.PermissionDenied("无权访问仪表盘所属项目")
	}
	if item.OwnerID != authInfo.UserId && (permission == biDashboardPermissionEdit || item.PublishedVersion == 0) {
		return nil, errorsx.PermissionDenied("无权访问仪表盘草稿")
	}
	return item, nil
}

// findVersion 按仪表盘编号读取独立版本。
func (c *BiDashboardCase) findVersion(ctx context.Context, id, version int64) (*models.BiDashboardVersion, error) {
	query := c.versionRepo.Query(ctx).BiDashboardVersion
	item, err := c.versionRepo.Find(ctx, repository.Where(query.DashboardID.Eq(id), query.Version.Eq(version)), repository.Clauses(clause.Locking{Strength: "SHARE"}))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorsx.ResourceNotFound("仪表盘版本不存在")
	}
	return item, err
}

// resolveSchema 从可信卡片补齐发布快照，草稿只保留引用和布局。
func (c *BiDashboardCase) resolveSchema(ctx context.Context, authInfo *authData.UserTokenPayload, projectID int64, value string, publish bool) (*dto.DashboardSchema, error) {
	layout, err := parseDashboardSchema(value)
	if err != nil {
		return nil, err
	}
	query := c.cardRepo.Query(ctx).BiCard
	for index, placed := range layout.Cards {
		var item *models.BiCard
		opts := []repository.QueryOption{repository.Where(query.ID.Eq(placed.CardID))}
		if publish {
			opts = append(opts, repository.Clauses(clause.Locking{Strength: "UPDATE"}))
		}
		item, err = c.cardRepo.Find(ctx, opts...)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errorsx.ResourceNotFound("仪表盘引用的卡片不存在")
		}
		if err != nil {
			return nil, err
		}
		if !canAccessBiCard(authInfo, item, false) || item.ProjectID != projectID {
			return nil, errorsx.PermissionDenied("不能引用其他租户、项目或私有卡片")
		}
		placed = dto.DashboardCard{ID: placed.ID, CardID: placed.CardID, X: placed.X, Y: placed.Y, W: placed.W, H: placed.H}
		placed.Name = ""
		placed.ChartType = ""
		placed.ChartJSON = ""
		placed.Query = nil
		placed.QueryRecordID = 0
		placed.Execution = nil
		placed.CardVersion = 0
		placed.ValidationQueryRecordID = 0
		if publish {
			var spec bi.QuerySpec
			if err = json.Unmarshal([]byte(item.QueryJSON), &spec); err != nil {
				return nil, errorsx.Internal("读取卡片查询失败").WithCause(err)
			}
			scope := bi.Scope{TenantID: authInfo.TenantId, ProjectID: projectID, UserID: authInfo.UserId, Terminal: "admin"}
			ownerScope := scope
			ownerScope.UserID = item.OwnerID
			if _, err = c.biQueryCase.cardSourceRecord(ctx, ownerScope, item.QueryRecordID, spec); err != nil {
				return nil, err
			}
			var execution *dto.CardExecution
			execution, err = decodeExecution(item.ExecutionJSON)
			if err != nil {
				return nil, err
			}
			var compiled *dto.CardExecution
			compiled, err = c.biQueryCase.biSourceCase.compile(ctx, scope, spec)
			if err != nil {
				return nil, err
			}
			if execution.Hash != compiled.Hash {
				return nil, errorsx.Conflict("卡片公共指标已变化，请刷新卡片")
			}
			if item.ValidationQueryRecordID > 0 {
				recordQuery := c.biQueryCase.biQueryRepo.Query(ctx).BiQuery
				var record *models.BiQuery
				record, err = c.biQueryCase.biQueryRepo.Find(ctx, repository.Where(recordQuery.ID.Eq(item.ValidationQueryRecordID), recordQuery.TenantID.Eq(item.TenantID), recordQuery.ProjectID.Eq(item.ProjectID), recordQuery.UserID.Eq(item.OwnerID), recordQuery.Kind.Eq(dto.QueryKindCardSQLPreview), recordQuery.Status.Eq(int32(adminv1.BiQueryStatus_BI_QUERY_STATUS_SUCCESS))))
				if err != nil {
					return nil, errorsx.Conflict("卡片人工预览来源无效")
				}
				var preview dto.CardPreviewContext
				if err = json.Unmarshal([]byte(record.ContextJSON), &preview); err != nil {
					return nil, errorsx.Conflict("卡片人工预览来源无效")
				}
				if preview.QueryRecordID != item.QueryRecordID || (preview.CardID != 0 && preview.CardID != item.ID) || record.AppliedCardID != item.ID {
					return nil, errorsx.Conflict("卡片人工预览来源无效")
				}
			}
			if _, err = validateCardChart(item.ChartType, item.ChartJSON, spec, bi.QueryResult{Columns: execution.Columns}); err != nil {
				return nil, err
			}
			placed.Execution = execution
			placed.CardVersion = item.Version
			placed.ValidationQueryRecordID = item.ValidationQueryRecordID
			placed.QueryRecordID = item.QueryRecordID
			placed.Name = item.Name
			placed.ChartType = item.ChartType
			placed.ChartJSON = item.ChartJSON
			placed.Query = &spec
		}
		layout.Cards[index] = placed
	}
	return layout, nil
}

// toBiDashboard 转换仪表盘记录，编辑权限由调用方补齐。
func toBiDashboard(item *models.BiDashboard, schemaJSON string) *adminv1.BiDashboard {
	var layout dto.DashboardSchema
	if schemaJSON != "" {
		if err := json.Unmarshal([]byte(schemaJSON), &layout); err != nil {
			schemaJSON = ""
		} else {
			payload, _ := json.Marshal(layout.Display()) // 白名单结构只包含 JSON 原生值。
			schemaJSON = string(payload)
		}
	}
	return &adminv1.BiDashboard{Id: item.ID, TenantId: item.TenantID, ProjectId: item.ProjectID, Code: item.Code, Name: item.Name, Description: item.Description, OwnerId: item.OwnerID, Status: item.Status, DraftVersion: item.DraftVersion, PublishedVersion: item.PublishedVersion, SchemaJson: schemaJSON, UpdatedAt: item.UpdatedAt.Format(time.RFC3339)}
}

// hashSchema 为发布快照生成完整性摘要。
func hashSchema(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

// parseDashboardSchema 拒绝旧数据集结构并验证卡片布局边界。
func parseDashboardSchema(value string) (*dto.DashboardSchema, error) {
	var layout dto.DashboardSchema
	if !json.Valid([]byte(value)) || len(value) > 1<<20 {
		return nil, errorsx.InvalidArgument("仪表盘布局格式无效")
	}
	decoder := json.NewDecoder(bytes.NewBufferString(value))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&layout)
	if err != nil {
		return nil, errorsx.InvalidArgument("仪表盘布局格式无效").WithCause(err)
	}
	if layout.SchemaVersion != 4 || len(layout.Cards) > biDashboardMaxCards {
		return nil, errorsx.InvalidArgument("仪表盘版本无效或超过20张卡片")
	}
	seen := make(map[int64]bool, len(layout.Cards))
	ids := make(map[string]bool, len(layout.Cards))
	for _, item := range layout.Cards {
		if item.CardID <= 0 || item.ID == "" || len(item.ID) > 64 || seen[item.CardID] || ids[item.ID] {
			return nil, errorsx.InvalidArgument("卡片引用无效或重复")
		}
		if item.X < 0 || item.Y < 0 || item.Y > 1000 || item.W < 1 || item.W > 12 || item.X+item.W > 12 || item.H < 1 || item.H > 24 {
			return nil, errorsx.InvalidArgument("卡片布局超出画布范围")
		}
		seen[item.CardID] = true
		ids[item.ID] = true
	}
	return &layout, nil
}

// projectAllowed 校验当前用户是否具有目标项目范围。
func projectAllowed(authInfo *authData.UserTokenPayload, projectID int64) bool {
	if projectID == 0 {
		return hasAllProjectsScope(authInfo)
	}
	return hasAllProjectsScope(authInfo) || slices.Contains(allowedProjectIDs(authInfo), projectID)
}

// hasAllProjectsScope 判断认证范围是否明确包含当前租户的全部项目。
func hasAllProjectsScope(authInfo *authData.UserTokenPayload) bool {
	if authInfo.TenantCode == databaseGorm.DefaultTenantCode {
		return true
	}
	for _, scope := range authInfo.TenantProjects {
		if scope.TenantId == authInfo.TenantId && slices.Contains(scope.ProjectId, int64(0)) {
			return true
		}
	}
	return false
}

// allowedProjectIDs 返回当前租户内明确授权的项目编号。
func allowedProjectIDs(authInfo *authData.UserTokenPayload) []int64 {
	var ids []int64
	for _, scope := range authInfo.TenantProjects {
		if scope.TenantId != authInfo.TenantId {
			continue
		}
		for _, id := range scope.ProjectId {
			if id > 0 && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	return ids
}
