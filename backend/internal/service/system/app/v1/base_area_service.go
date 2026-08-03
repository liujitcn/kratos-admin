package app

import (
	"context"
	"fmt"

	systemappv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/app/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/app/v1"

	"github.com/go-kratos/kratos/v3/log"
)

// BaseAreaService 行政区域服务
type BaseAreaService struct {
	systemappv1.UnimplementedBaseAreaServiceServer
	baseAreaCase *biz.BaseAreaCase
}

// NewBaseAreaService 创建行政区域服务
func NewBaseAreaService(
	baseAreaCase *biz.BaseAreaCase,
) *BaseAreaService {
	var ss = BaseAreaService{
		baseAreaCase: baseAreaCase,
	}
	return &ss
}

// TreeBaseArea 查询行政区域树形列表
func (s *BaseAreaService) TreeBaseArea(ctx context.Context, req *systemappv1.TreeBaseAreaRequest) (*systemappv1.TreeBaseAreaResponse, error) {
	tree, err := s.baseAreaCase.TreeBaseArea(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("TreeBaseArea %v", err))
		return nil, errorsx.WrapInternal(err, "查询行政区域树形列表失败")
	}

	return &systemappv1.TreeBaseAreaResponse{Areas: tree.GetList()}, nil
}
