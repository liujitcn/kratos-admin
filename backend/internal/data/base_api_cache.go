package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/gorm-kit/repository"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-kit/cache"
)

const baseAPICacheRevisionKey = "shared:api:tools:revision"

// LoadBaseAPICache 读取按工具名分组的API元数据快照；不缓存用户授权，执行前仍须核实数据库开关。
func LoadBaseAPICache(ctx context.Context, repo *data.BaseAPIRepository, store cache.Cache) (map[string][]*models.BaseAPI, error) {
	revision, enabled := cache.ReadRevision(store, baseAPICacheRevisionKey)
	key := "shared:api:tools:data:" + revision
	var err error
	if enabled {
		var value string
		value, err = store.Get(key)
		if err == nil {
			var cached map[string][]*models.BaseAPI
			err = json.Unmarshal([]byte(value), &cached)
			if err == nil && cached != nil {
				return cached, nil
			}
		}
	}
	query := repo.Query(ctx).BaseAPI
	opts := []repository.QueryOption{repository.Order(query.ID.Asc())}
	var rows []*models.BaseAPI
	rows, err = repo.List(ctx, opts...)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]*models.BaseAPI)
	for _, row := range rows {
		result[row.ToolName] = append(result[row.ToolName], row)
	}
	if enabled {
		var payload []byte
		payload, err = json.Marshal(result)
		if err != nil {
			return nil, err
		}
		// 旧版本填充只写旧键，事务提交后版本递增不会被并发回填覆盖。
		if err = store.Set(key, string(payload), time.Minute); err != nil {
			log.Warn("保存API工具缓存失败", "error", err)
		}
	}
	return result, nil
}

// InvalidateBaseAPICache 在API配置事务提交或资源同步后递增共享版本，使各节点读取新快照。
func InvalidateBaseAPICache(store cache.Cache) error {
	return cache.IncrementRevision(store, baseAPICacheRevisionKey)
}
