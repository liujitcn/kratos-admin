package admin

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin/dto"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-core/errorsx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// validatePublishedExecutions 在执行任一卡片前及整盘返回前验证完整快照，历史版本同样受约束。
func (c *BiDashboardCase) validatePublishedExecutions(ctx context.Context, scope bi.Scope, layout *dto.DashboardSchema) error {
	for _, placed := range layout.Cards {
		if placed.Query == nil || placed.Execution == nil {
			return errorsx.Conflict("公共指标或模型已更新，请重新发布仪表盘")
		}
		if err := c.biQueryCase.biSourceCase.checkExecution(ctx, scope, *placed.Query, placed.Execution); err != nil {
			return err
		}
	}
	return nil
}

// requiresRepublish 比较当前发布指针的实际模型修订，不因密码或画布修改误报。
func (c *BiDashboardCase) requiresRepublish(ctx context.Context, item *models.BiDashboard) (bool, error) {
	if item.PublishedVersion == 0 {
		return false, nil
	}
	version, err := c.findVersion(ctx, item.ID, item.PublishedVersion)
	if err != nil {
		return false, err
	}
	var layout dto.DashboardSchema
	if err = json.Unmarshal([]byte(version.SchemaJSON), &layout); err != nil || layout.SchemaVersion != 4 {
		return true, nil
	}
	for _, placed := range layout.Cards {
		if placed.Query == nil || placed.Execution == nil {
			return true, nil
		}
		sourceCase := c.biQueryCase.biSourceCase
		if sourceCase.registry.Has(placed.Query.SourceCode) {
			var current *dto.CardExecution
			current, err = sourceCase.compile(ctx, bi.Scope{TenantID: item.TenantID, ProjectID: item.ProjectID, UserID: item.OwnerID, Terminal: "admin"}, *placed.Query)
			if err != nil {
				return false, err
			}
			if current.Hash != placed.Execution.Hash {
				return true, nil
			}
			continue
		}
		query := sourceCase.sourceRepo.Query(ctx).BiSource
		var source *models.BiSource
		source, err = sourceCase.sourceRepo.Find(ctx, repository.Where(query.TenantID.Eq(item.TenantID), query.Code.Eq(placed.Query.SourceCode)))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		if err != nil {
			return false, err
		}
		var model dto.ConfiguredSourceSchema
		if err = json.Unmarshal([]byte(source.SchemaJSON), &model); err != nil {
			return false, errorsx.Internal("读取公共模型失败").WithCause(err)
		}
		if !executionMatchesModel(placed.Execution, model) {
			return true, nil
		}
	}
	return false, nil
}

// lockPublishInputs 按源 ID、卡片 ID 升序加锁，仪表盘与版本锁在调用方随后取得。
func (c *BiDashboardCase) lockPublishInputs(ctx context.Context, tenantID int64, schemaJSON string) error {
	layout, err := parseDashboardSchema(schemaJSON)
	if err != nil {
		return err
	}
	ids := make([]int64, 0, len(layout.Cards))
	for _, placed := range layout.Cards {
		ids = append(ids, placed.CardID)
	}
	if len(ids) == 0 {
		return nil
	}
	query := c.cardRepo.Query(ctx).BiCard
	opts := []repository.QueryOption{repository.Where(query.TenantID.Eq(tenantID), query.ID.In(ids...)), repository.Order(query.ID.Asc())}
	var cards []*models.BiCard
	cards, err = c.cardRepo.List(ctx, opts...)
	if err != nil {
		return err
	}
	codes := make([]string, 0, len(cards))
	for _, card := range cards {
		if !slices.Contains(codes, card.SourceCode) {
			codes = append(codes, card.SourceCode)
		}
	}
	sourceCase := c.biQueryCase.biSourceCase
	sourceQuery := sourceCase.sourceRepo.Query(ctx).BiSource
	var sources []*models.BiSource
	sources, err = sourceCase.sourceRepo.List(ctx, repository.Where(sourceQuery.TenantID.Eq(tenantID), sourceQuery.Code.In(codes...)), repository.Order(sourceQuery.ID.Asc()))
	if err != nil {
		return err
	}
	for _, source := range sources {
		if _, err = sourceCase.lockSource(ctx, tenantID, source.Code); err != nil {
			return err
		}
	}
	_, err = c.cardRepo.List(ctx, append(opts, repository.Clauses(clause.Locking{Strength: "UPDATE"}))...)
	return err
}
