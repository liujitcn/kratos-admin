package admin

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-core/errorsx"
)

// AiSessionService AI会话管理服务。
type AiSessionService struct {
	adminv1.UnimplementedAiSessionServiceServer
	aiSessionCase *biz.AiSessionCase
}

// NewAiSessionService 创建AI会话管理服务。
func NewAiSessionService(aiSessionCase *biz.AiSessionCase) *AiSessionService {
	return &AiSessionService{aiSessionCase: aiSessionCase}
}

// PageAiSession 分页查询AI会话。
func (s *AiSessionService) PageAiSession(ctx context.Context, req *adminv1.PageAiSessionRequest) (*adminv1.PageAiSessionResponse, error) {
	result, err := s.aiSessionCase.PageAiSession(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageAiSession %v", err))
		return nil, errorsx.WrapInternal(err, "查询AI会话失败")
	}
	return result, nil
}

// PageAiSessionMessage 分页查询AI会话消息。
func (s *AiSessionService) PageAiSessionMessage(ctx context.Context, req *adminv1.PageAiSessionMessageRequest) (*adminv1.PageAiSessionMessageResponse, error) {
	result, err := s.aiSessionCase.PageAiSessionMessage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("PageAiSessionMessage %v", err))
		return nil, errorsx.WrapInternal(err, "查询AI会话消息失败")
	}
	return result, nil
}
