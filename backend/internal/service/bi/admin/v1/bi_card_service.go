package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/bi/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/bi/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// BiCardService 提供数据卡片和个人工作台接口。
type BiCardService struct {
	adminv1.UnimplementedBiCardServiceServer
	biCardCase *admin.BiCardCase
}

// NewBiCardService 创建数据卡片服务。
func NewBiCardService(biCardCase *admin.BiCardCase) *BiCardService {
	return &BiCardService{biCardCase: biCardCase}
}

// PageBiCard 分页查询数据卡片。
func (s *BiCardService) PageBiCard(ctx context.Context, req *adminv1.PageBiCardRequest) (*adminv1.PageBiCardResponse, error) {
	result, err := s.biCardCase.PageBiCard(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageBiCard %v", err))
		return nil, errorsx.WrapInternal(err, "分页查询数据卡片失败")
	}
	return result, nil
}

// SaveBiCard 保存数据卡片。
func (s *BiCardService) SaveBiCard(ctx context.Context, req *adminv1.SaveBiCardRequest) (*adminv1.BiCard, error) {
	result, err := s.biCardCase.SaveBiCard(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SaveBiCard %v", err))
		return nil, errorsx.WrapInternal(err, "保存数据卡片失败")
	}
	return result, nil
}

// DeleteBiCard 删除数据卡片。
func (s *BiCardService) DeleteBiCard(ctx context.Context, req *adminv1.DeleteBiCardRequest) (*adminv1.DeleteBiCardResponse, error) {
	result, err := s.biCardCase.DeleteBiCard(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("DeleteBiCard %v", err))
		return nil, errorsx.WrapInternal(err, "删除数据卡片失败")
	}
	return result, nil
}

// GetBiCardData 执行卡片查询。
func (s *BiCardService) GetBiCardData(ctx context.Context, req *adminv1.GetBiCardDataRequest) (*adminv1.BiQueryResult, error) {
	result, err := s.biCardCase.GetBiCardData(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBiCardData %v", err))
		return nil, errorsx.WrapInternal(err, "执行卡片查询失败")
	}
	return result, nil
}

// GetBiWorkbench 获取个人工作台。
func (s *BiCardService) GetBiWorkbench(ctx context.Context, req *adminv1.GetBiWorkbenchRequest) (*adminv1.BiWorkbench, error) {
	result, err := s.biCardCase.GetBiWorkbench(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("GetBiWorkbench %v", err))
		return nil, errorsx.WrapInternal(err, "获取个人工作台失败")
	}
	return result, nil
}

// SaveBiWorkbench 保存个人工作台。
func (s *BiCardService) SaveBiWorkbench(ctx context.Context, req *adminv1.SaveBiWorkbenchRequest) (*adminv1.BiWorkbench, error) {
	result, err := s.biCardCase.SaveBiWorkbench(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SaveBiWorkbench %v", err))
		return nil, errorsx.WrapInternal(err, "保存个人工作台失败")
	}
	return result, nil
}
