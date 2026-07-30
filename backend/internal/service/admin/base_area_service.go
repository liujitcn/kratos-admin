package admin

import (
	"context"

	commonv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/common/v1"
	systemadminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/admin"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const _ = grpc.SupportPackageIsVersion7

// BaseAreaService Admin行政区域服务。
type BaseAreaService struct {
	systemadminv1.UnimplementedBaseAreaServiceServer
	baseAreaCase *biz.BaseAreaCase
}

// NewBaseAreaService 创建Admin行政区域服务。
func NewBaseAreaService(baseAreaCase *biz.BaseAreaCase) *BaseAreaService {
	return &BaseAreaService{baseAreaCase: baseAreaCase}
}

// OptionBaseArea 查询选项失败。
func (s *BaseAreaService) OptionBaseArea(ctx context.Context, req *systemadminv1.OptionBaseAreaRequest) (*commonv1.TreeOptionResponse, error) {
	res, err := s.baseAreaCase.OptionBaseArea(ctx, req)
	if err != nil {
		log.Error("OptionBaseArea", "error", err)
		return nil, errorsx.WrapInternal(err, "查询选项失败")
	}
	return res, nil
}

// TreeBaseArea 查询行政区域树形列表失败。
func (s *BaseAreaService) TreeBaseArea(ctx context.Context, req *systemadminv1.TreeBaseAreaRequest) (*systemadminv1.TreeBaseAreaResponse, error) {
	res, err := s.baseAreaCase.TreeBaseArea(ctx, req)
	if err != nil {
		log.Error("TreeBaseArea", "error", err)
		return nil, errorsx.WrapInternal(err, "查询行政区域树形列表失败")
	}
	return res, nil
}

// GetBaseArea 查询行政区域失败。
func (s *BaseAreaService) GetBaseArea(ctx context.Context, req *systemadminv1.GetBaseAreaRequest) (*systemadminv1.BaseAreaForm, error) {
	res, err := s.baseAreaCase.GetBaseArea(ctx, req.GetId())
	if err != nil {
		log.Error("GetBaseArea", "error", err)
		return nil, errorsx.WrapInternal(err, "查询行政区域失败")
	}
	return res, nil
}

// CreateBaseArea 创建行政区域失败。
func (s *BaseAreaService) CreateBaseArea(ctx context.Context, req *systemadminv1.CreateBaseAreaRequest) (*emptypb.Empty, error) {
	err := s.baseAreaCase.CreateBaseArea(ctx, req.GetBaseArea())
	if err != nil {
		log.Error("CreateBaseArea", "error", err)
		return nil, errorsx.WrapInternal(err, "创建行政区域失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseArea 更新行政区域失败。
func (s *BaseAreaService) UpdateBaseArea(ctx context.Context, req *systemadminv1.UpdateBaseAreaRequest) (*emptypb.Empty, error) {
	err := s.baseAreaCase.UpdateBaseArea(ctx, req.GetId(), req.GetBaseArea())
	if err != nil {
		log.Error("UpdateBaseArea", "error", err)
		return nil, errorsx.WrapInternal(err, "更新行政区域失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseArea 删除行政区域失败。
func (s *BaseAreaService) DeleteBaseArea(ctx context.Context, req *systemadminv1.DeleteBaseAreaRequest) (*emptypb.Empty, error) {
	err := s.baseAreaCase.DeleteBaseArea(ctx, req.GetIds())
	if err != nil {
		log.Error("DeleteBaseArea", "error", err)
		return nil, errorsx.WrapInternal(err, "删除行政区域失败")
	}
	return new(emptypb.Empty), nil
}
