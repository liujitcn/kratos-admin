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

// AiProviderService AI供应商管理服务。
type AiProviderService struct {
	adminv1.UnimplementedAiProviderServiceServer
	aiProviderCase *biz.AiProviderCase
}

// NewAiProviderService 创建AI供应商管理服务。
func NewAiProviderService(aiProviderCase *biz.AiProviderCase) *AiProviderService {
	return &AiProviderService{aiProviderCase: aiProviderCase}
}

// PageAiProvider 分页查询AI供应商。
func (s *AiProviderService) PageAiProvider(ctx context.Context, req *adminv1.PageAiProviderRequest) (*adminv1.PageAiProviderResponse, error) {
	result, err := s.aiProviderCase.PageAiProvider(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageAiProvider %v", err))
		return nil, errorsx.WrapInternal(err, "查询AI供应商失败")
	}
	return result, nil
}

// GetAiProvider 查询AI供应商表单。
func (s *AiProviderService) GetAiProvider(ctx context.Context, req *adminv1.GetAiProviderRequest) (*adminv1.AiProviderForm, error) {
	result, err := s.aiProviderCase.GetAiProvider(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetAiProvider %v", err))
		return nil, errorsx.WrapInternal(err, "查询AI供应商失败")
	}
	return result, nil
}

// CreateAiProvider 创建AI供应商。
func (s *AiProviderService) CreateAiProvider(ctx context.Context, req *adminv1.CreateAiProviderRequest) (*emptypb.Empty, error) {
	err := s.aiProviderCase.CreateAiProvider(ctx, req.GetAiProvider())
	if err != nil {
		log.Error(fmt.Sprintf("CreateAiProvider %v", err))
		return nil, errorsx.WrapInternal(err, "创建AI供应商失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateAiProvider 更新AI供应商。
func (s *AiProviderService) UpdateAiProvider(ctx context.Context, req *adminv1.UpdateAiProviderRequest) (*emptypb.Empty, error) {
	err := s.aiProviderCase.UpdateAiProvider(ctx, req.GetAiProvider())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateAiProvider %v", err))
		return nil, errorsx.WrapInternal(err, "更新AI供应商失败")
	}
	return new(emptypb.Empty), nil
}

// TestAiProviderModels 测试供应商表单中的全部模型。
func (s *AiProviderService) TestAiProviderModels(ctx context.Context, req *adminv1.TestAiProviderModelsRequest) (*adminv1.TestAiProviderModelsResponse, error) {
	result, err := s.aiProviderCase.TestAiProviderModels(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("TestAiProviderModels %v", err))
		return nil, errorsx.WrapInternal(err, "测试AI模型失败")
	}
	return result, nil
}

// DeleteAiProvider 删除AI供应商。
func (s *AiProviderService) DeleteAiProvider(ctx context.Context, req *adminv1.DeleteAiProviderRequest) (*emptypb.Empty, error) {
	err := s.aiProviderCase.DeleteAiProvider(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteAiProvider %v", err))
		return nil, errorsx.WrapInternal(err, "删除AI供应商失败")
	}
	return new(emptypb.Empty), nil
}

// SetAiProviderStatus 设置AI供应商状态。
func (s *AiProviderService) SetAiProviderStatus(ctx context.Context, req *adminv1.SetAiProviderStatusRequest) (*emptypb.Empty, error) {
	err := s.aiProviderCase.SetAiProviderStatus(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SetAiProviderStatus %v", err))
		return nil, errorsx.WrapInternal(err, "设置AI供应商状态失败")
	}
	return new(emptypb.Empty), nil
}
