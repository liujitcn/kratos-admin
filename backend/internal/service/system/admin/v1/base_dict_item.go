package admin

import (
	"context"
	"fmt"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// BaseDictItemService Admin字典项服务
type BaseDictItemService struct {
	adminv1.UnimplementedBaseDictItemServiceServer
	baseDictItemCase *biz.BaseDictItemCase
}

// NewBaseDictItemService 创建Admin字典项服务
func NewBaseDictItemService(baseDictItemCase *biz.BaseDictItemCase) *BaseDictItemService {
	return &BaseDictItemService{baseDictItemCase: baseDictItemCase}
}

// PageBaseDictItem 查询字典属性分页列表
func (s *BaseDictItemService) PageBaseDictItem(ctx context.Context, req *adminv1.PageBaseDictItemRequest) (*adminv1.PageBaseDictItemResponse, error) {
	page, err := s.baseDictItemCase.PageBaseDictItem(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBaseDictItem %v", err))
		return nil, errorsx.WrapInternal(err, "查询字典属性分页列表失败")
	}
	return page, nil
}

// GetBaseDictItem 查询字典属性
func (s *BaseDictItemService) GetBaseDictItem(ctx context.Context, req *adminv1.GetBaseDictItemRequest) (*adminv1.BaseDictItemForm, error) {
	baseDictItem, err := s.baseDictItemCase.GetBaseDictItem(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetBaseDictItem %v", err))
		return nil, errorsx.WrapInternal(err, "查询字典属性失败")
	}
	return baseDictItem, nil
}

// CreateBaseDictItem 创建字典属性
func (s *BaseDictItemService) CreateBaseDictItem(ctx context.Context, req *adminv1.CreateBaseDictItemRequest) (*emptypb.Empty, error) {
	err := s.baseDictItemCase.CreateBaseDictItem(ctx, req.GetBaseDictItem())
	if err != nil {
		log.Error(fmt.Sprintf("CreateBaseDictItem %v", err))
		return nil, errorsx.WrapInternal(err, "创建字典属性失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateBaseDictItem 更新字典属性
func (s *BaseDictItemService) UpdateBaseDictItem(ctx context.Context, req *adminv1.UpdateBaseDictItemRequest) (*emptypb.Empty, error) {
	err := s.baseDictItemCase.UpdateBaseDictItem(ctx, req.GetBaseDictItem())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateBaseDictItem %v", err))
		return nil, errorsx.WrapInternal(err, "更新字典属性失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteBaseDictItem 删除字典属性
func (s *BaseDictItemService) DeleteBaseDictItem(ctx context.Context, req *adminv1.DeleteBaseDictItemRequest) (*emptypb.Empty, error) {
	err := s.baseDictItemCase.DeleteBaseDictItem(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBaseDictItem %v", err))
		return nil, errorsx.WrapInternal(err, "删除字典属性失败")
	}
	return new(emptypb.Empty), nil
}

// SetBaseDictItemStatus 设置状态
func (s *BaseDictItemService) SetBaseDictItemStatus(ctx context.Context, req *adminv1.SetBaseDictItemStatusRequest) (*emptypb.Empty, error) {
	err := s.baseDictItemCase.SetBaseDictItemStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetBaseDictItemStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置状态失败")
	}
	return new(emptypb.Empty), nil
}
