package data

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// CreateBiDashboardDraft 创建草稿版本，省略发布时间以保存 NULL，避免 MySQL 拒绝零日期。
func CreateBiDashboardDraft(ctx context.Context, repo *data.BiDashboardVersionRepository, item *models.BiDashboardVersion) error {
	query := repo.Query(ctx).BiDashboardVersion
	return query.WithContext(ctx).Omit(query.PublishedAt).Create(item)
}

// UpdateBiDashboardState 按原草稿版本原子更新状态，防止保存与发布互相覆盖。
func UpdateBiDashboardState(ctx context.Context, repo *data.BiDashboardRepository, item *models.BiDashboard, previous int64) (int64, error) {
	query := repo.Query(ctx).BiDashboard
	result, err := query.WithContext(ctx).Where(query.ID.Eq(item.ID), query.TenantID.Eq(item.TenantID), query.DraftVersion.Eq(previous)).Select(query.Name, query.Description, query.Status, query.DraftVersion, query.PublishedVersion, query.UpdatedBy, query.UpdatedAt).Updates(item)
	return result.RowsAffected, err
}

// PublishBiDashboardVersion 按原状态原子写入发布快照，防止重复发布改写同一版本。
func PublishBiDashboardVersion(ctx context.Context, repo *data.BiDashboardVersionRepository, item *models.BiDashboardVersion, previous int32) (int64, error) {
	query := repo.Query(ctx).BiDashboardVersion
	result, err := query.WithContext(ctx).Where(query.ID.Eq(item.ID), query.Status.Eq(previous)).Select(query.SchemaJSON, query.SchemaHash, query.Status, query.PublishedBy, query.PublishedAt).Updates(item)
	return result.RowsAffected, err
}
