package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
)

// CacheService 提供运行时缓存查询接口。
type CacheService struct {
	adminv1.UnimplementedCacheServiceServer
	cacheCase *biz.CacheCase
}

// NewCacheService 创建运行时缓存查询服务。
func NewCacheService(cacheCase *biz.CacheCase) *CacheService {
	return &CacheService{cacheCase: cacheCase}
}

// PageCache 分页查询当前进程已观测到的缓存条目。
func (s *CacheService) PageCache(ctx context.Context, req *adminv1.PageCacheRequest) (*adminv1.PageCacheResponse, error) {
	response, err := s.cacheCase.PageCache(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageCache %v", err))
		return nil, errorsx.WrapInternal(err, "查询缓存条目失败")
	}
	return response, nil
}
