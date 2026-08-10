package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/dto"
	"github.com/liujitcn/kratos-core/pkg/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/grpc"
)

const _ = grpc.SupportPackageIsVersion7

// AiMessageService AI 助手消息公共服务。
type AiMessageService struct {
	basev1.UnimplementedAiMessageServiceServer
	aiMessageCase *biz.AiMessageCase
}

// NewAiMessageService 创建 AI 助手消息公共服务。
func NewAiMessageService(aiMessageCase *biz.AiMessageCase) *AiMessageService {
	return &AiMessageService{
		aiMessageCase: aiMessageCase,
	}
}

// UpdateAiMessage 更新 AI 助手消息并重新生成输出。
func (s *AiMessageService) UpdateAiMessage(ctx context.Context, req *basev1.UpdateAiMessageRequest) (*basev1.SendAiMessageResponse, error) {
	res, err := s.aiMessageCase.UpdateAiMessage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("UpdateAiMessage %v", err))
		return nil, errorsx.WrapInternal(err, "更新AI助手消息失败")
	}
	return res, nil
}

// DeleteAiMessage 删除 AI 助手消息。
func (s *AiMessageService) DeleteAiMessage(ctx context.Context, req *basev1.DeleteAiMessageRequest) (*basev1.DeleteAiMessageResponse, error) {
	err := s.aiMessageCase.DeleteAiMessage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("DeleteAiMessage %v", err))
		return nil, errorsx.WrapInternal(err, "删除AI助手消息失败")
	}
	return &basev1.DeleteAiMessageResponse{}, nil
}

// SendAiMessage 发送 AI 助手消息。
func (s *AiMessageService) SendAiMessage(ctx context.Context, req *basev1.SendAiMessageRequest) (*basev1.SendAiMessageResponse, error) {
	res, err := s.aiMessageCase.SendAiMessage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SendAiMessage %v", err))
		return nil, errorsx.WrapInternal(err, "发送AI助手消息失败")
	}
	return res, nil
}

// RetryAiUserMessage 重试失败的 AI 助手消息。
func (s *AiMessageService) RetryAiUserMessage(ctx context.Context, req *basev1.RetryAiUserMessageRequest) (*basev1.SendAiMessageResponse, error) {
	res, err := s.aiMessageCase.RetryAiUserMessage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("RetryAiUserMessage %v", err))
		return nil, errorsx.WrapInternal(err, "重试AI助手消息失败")
	}
	return res, nil
}

// RegenerateAiMessage 重新生成 AI 助手输出。
func (s *AiMessageService) RegenerateAiMessage(ctx context.Context, req *basev1.RegenerateAiMessageRequest) (*basev1.SendAiMessageResponse, error) {
	res, err := s.aiMessageCase.RegenerateAiMessage(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("RegenerateAiMessage %v", err))
		return nil, errorsx.WrapInternal(err, "重新生成AI助手输出失败")
	}
	return res, nil
}

// StreamAiMessage 流式发送 AI 助手消息。
func (s *AiMessageService) StreamAiMessage(ctx context.Context, req *basev1.SendAiMessageRequest, emitter dto.AiStreamEmitter) error {
	err := s.aiMessageCase.StreamAiMessage(ctx, req, emitter)
	if err != nil {
		log.Error(fmt.Sprintf("StreamAiMessage %v", err))
		return errorsx.WrapInternal(err, "流式发送AI助手消息失败")
	}
	return nil
}
