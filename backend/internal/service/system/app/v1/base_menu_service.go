package app

import (
	"context"
	"fmt"

	systemappv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/app/v1"

	"github.com/go-kratos/kratos/v3/log"
)

// BaseMenuService 移动端菜单服务。
type BaseMenuService struct {
	systemappv1.UnimplementedBaseMenuServiceServer
	baseMenuCase *biz.BaseMenuCase
}

// NewBaseMenuService 创建移动端菜单服务。
func NewBaseMenuService(baseMenuCase *biz.BaseMenuCase) *BaseMenuService {
	return &BaseMenuService{baseMenuCase: baseMenuCase}
}

// ListBaseMenu 查询移动端菜单。
func (s *BaseMenuService) ListBaseMenu(ctx context.Context, _ *systemappv1.ListBaseMenuRequest) (*systemappv1.ListBaseMenuResponse, error) {
	items, err := s.baseMenuCase.ListBaseMenu(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("ListBaseMenu %v", err))
		return nil, errorsx.WrapInternal(err, "查询移动端菜单失败")
	}
	return &systemappv1.ListBaseMenuResponse{Items: items}, nil
}
