package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/rag/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/rag/admin"
	"github.com/liujitcn/kratos-core/errorsx"
	"google.golang.org/protobuf/types/known/emptypb"
)

// AiKnowledgeService AI知识库管理服务。
type AiKnowledgeService struct {
	adminv1.UnimplementedAiKnowledgeServiceServer
	aiKnowledgeCase *biz.AiKnowledgeCase
}

// NewAiKnowledgeService 创建AI知识库管理服务。
func NewAiKnowledgeService(aiKnowledgeCase *biz.AiKnowledgeCase) *AiKnowledgeService {
	return &AiKnowledgeService{aiKnowledgeCase: aiKnowledgeCase}
}

// PageAiKnowledge 分页查询AI知识库。
func (s *AiKnowledgeService) PageAiKnowledge(ctx context.Context, req *adminv1.PageAiKnowledgeRequest) (*adminv1.PageAiKnowledgeResponse, error) {
	result, err := s.aiKnowledgeCase.PageAiKnowledge(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageAiKnowledge %v", err))
		return nil, errorsx.WrapInternal(err, "查询AI知识库失败")
	}
	return result, nil
}

// GetAiKnowledge 查询AI知识库表单。
func (s *AiKnowledgeService) GetAiKnowledge(ctx context.Context, req *adminv1.GetAiKnowledgeRequest) (*adminv1.AiKnowledgeForm, error) {
	result, err := s.aiKnowledgeCase.GetAiKnowledge(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("GetAiKnowledge %v", err))
		return nil, errorsx.WrapInternal(err, "查询AI知识库失败")
	}
	return result, nil
}

// CreateAiKnowledge 创建AI知识库。
func (s *AiKnowledgeService) CreateAiKnowledge(ctx context.Context, req *adminv1.CreateAiKnowledgeRequest) (*emptypb.Empty, error) {
	err := s.aiKnowledgeCase.CreateAiKnowledge(ctx, req.GetAiKnowledge())
	if err != nil {
		log.Error(fmt.Sprintf("CreateAiKnowledge %v", err))
		return nil, errorsx.WrapInternal(err, "创建AI知识库失败")
	}
	return new(emptypb.Empty), nil
}

// UpdateAiKnowledge 更新AI知识库。
func (s *AiKnowledgeService) UpdateAiKnowledge(ctx context.Context, req *adminv1.UpdateAiKnowledgeRequest) (*emptypb.Empty, error) {
	err := s.aiKnowledgeCase.UpdateAiKnowledge(ctx, req.GetAiKnowledge())
	if err != nil {
		log.Error(fmt.Sprintf("UpdateAiKnowledge %v", err))
		return nil, errorsx.WrapInternal(err, "更新AI知识库失败")
	}
	return new(emptypb.Empty), nil
}

// DeleteAiKnowledge 删除AI知识库。
func (s *AiKnowledgeService) DeleteAiKnowledge(ctx context.Context, req *adminv1.DeleteAiKnowledgeRequest) (*emptypb.Empty, error) {
	err := s.aiKnowledgeCase.DeleteAiKnowledge(ctx, req.GetId())
	if err != nil {
		log.Error(fmt.Sprintf("DeleteAiKnowledge %v", err))
		return nil, errorsx.WrapInternal(err, "删除AI知识库失败")
	}
	return new(emptypb.Empty), nil
}

// ListAiKnowledgeModels 查询已启用的embedding模型选项。
func (s *AiKnowledgeService) ListAiKnowledgeModels(ctx context.Context, _ *adminv1.ListAiKnowledgeModelsRequest) (*adminv1.ListAiKnowledgeModelsResponse, error) {
	result, err := s.aiKnowledgeCase.ListAiKnowledgeModels(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("ListAiKnowledgeModels %v", err))
		return nil, errorsx.WrapInternal(err, "查询embedding模型失败")
	}
	return result, nil
}
