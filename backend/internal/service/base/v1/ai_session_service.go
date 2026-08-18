package base

import (
	"context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// AiSessionService AI 助手会话服务。
type AiSessionService struct {
	basev1.UnimplementedAiSessionServiceServer
	aiSessionCase *biz.AiSessionCase
	aiMessageCase *biz.AiMessageCase
}

// NewAiSessionService 创建 AI 助手会话服务。
func NewAiSessionService(aiSessionCase *biz.AiSessionCase, aiMessageCase *biz.AiMessageCase) *AiSessionService {
	return &AiSessionService{
		aiSessionCase: aiSessionCase,
		aiMessageCase: aiMessageCase,
	}
}

// ListAiMessage 查询 AI 助手消息列表。
func (s *AiSessionService) ListAiMessage(ctx context.Context, req *basev1.ListAiMessageRequest) (*basev1.ListAiMessageResponse, error) {
	res, err := s.aiMessageCase.ListAiMessage(ctx, req)
	if err != nil {
		log.Error("ListAiMessage", "error", err)
		return nil, errorsx.WrapInternal(err, "查询AI助手消息失败")
	}
	return res, nil
}

// ListAiSession 查询 AI 助手会话列表。
func (s *AiSessionService) ListAiSession(ctx context.Context, req *basev1.ListAiSessionRequest) (*basev1.ListAiSessionResponse, error) {
	res, err := s.aiSessionCase.ListAiSession(ctx, req)
	if err != nil {
		log.Error("ListAiSession", "error", err)
		return nil, errorsx.WrapInternal(err, "查询AI助手会话失败")
	}
	return res, nil
}

// CreateAiSession 创建 AI 助手会话。
func (s *AiSessionService) CreateAiSession(ctx context.Context, req *basev1.CreateAiSessionRequest) (*basev1.CreateAiSessionResponse, error) {
	res, err := s.aiSessionCase.CreateAiSession(ctx, req)
	if err != nil {
		log.Error("CreateAiSession", "error", err)
		return nil, errorsx.WrapInternal(err, "创建AI助手会话失败")
	}
	return &basev1.CreateAiSessionResponse{Session: res}, nil
}

// CreateAiSessionBranch 从指定消息创建 AI 助手分支会话。
func (s *AiSessionService) CreateAiSessionBranch(ctx context.Context, req *basev1.CreateAiSessionBranchRequest) (*basev1.CreateAiSessionBranchResponse, error) {
	res, err := s.aiSessionCase.CreateAiSessionBranch(ctx, req)
	if err != nil {
		log.Error("CreateAiSessionBranch", "error", err)
		return nil, errorsx.WrapInternal(err, "创建AI助手分支会话失败")
	}
	return res, nil
}

// UpdateAiSession 更新 AI 助手会话。
func (s *AiSessionService) UpdateAiSession(ctx context.Context, req *basev1.UpdateAiSessionRequest) (*basev1.UpdateAiSessionResponse, error) {
	res, err := s.aiSessionCase.UpdateAiSession(ctx, req)
	if err != nil {
		log.Error("UpdateAiSession", "error", err)
		return nil, errorsx.WrapInternal(err, "更新AI助手会话失败")
	}
	return &basev1.UpdateAiSessionResponse{Session: res}, nil
}

// DeleteAiSession 删除 AI 助手会话。
func (s *AiSessionService) DeleteAiSession(ctx context.Context, req *basev1.DeleteAiSessionRequest) (*basev1.DeleteAiSessionResponse, error) {
	_, err := s.aiSessionCase.DeleteAiSession(ctx, req)
	if err != nil {
		log.Error("DeleteAiSession", "error", err)
		return nil, errorsx.WrapInternal(err, "删除AI助手会话失败")
	}
	return &basev1.DeleteAiSessionResponse{}, nil
}
