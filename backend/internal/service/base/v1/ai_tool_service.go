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

// AiToolService AI 助手工具服务。
type AiToolService struct {
	basev1.UnimplementedAiToolServiceServer
	aiToolCase *biz.AiToolCase
}

// NewAiToolService 创建 AI 助手工具服务。
func NewAiToolService(aiToolCase *biz.AiToolCase) *AiToolService {
	return &AiToolService{aiToolCase: aiToolCase}
}

// ListAiShortcut 查询 AI 助手快捷入口列表。
func (s *AiToolService) ListAiShortcut(ctx context.Context, req *basev1.ListAiShortcutRequest) (*basev1.ListAiShortcutResponse, error) {
	res, err := s.aiToolCase.ListAiShortcut(ctx, req)
	if err != nil {
		log.Error("ListAiShortcut", "error", err)
		return nil, errorsx.WrapInternal(err, "查询AI助手快捷入口失败")
	}
	return res, nil
}
