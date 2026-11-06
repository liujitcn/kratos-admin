package data

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"gorm.io/gorm/clause"
)

// SaveBiWorkbench 按租户和用户原子保存个人配置，避免首次并发保存产生重复记录。
func SaveBiWorkbench(ctx context.Context, repo *data.BiWorkbenchRepository, item *models.BiWorkbench) error {
	query := repo.Query(ctx).BiWorkbench
	return query.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"config_json", "updated_at"}),
	}).Create(item)
}
