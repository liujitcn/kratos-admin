package biz

import (
	"context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-core/biz"
	coresse "github.com/liujitcn/kratos-core/sse"

	"google.golang.org/protobuf/types/known/emptypb"
)

// SseCase 处理 SSE 公共业务。
type SseCase struct {
	*biz.BaseCase
	sse *coresse.SSE
}

// NewSseCase 创建 SSE 业务实例。
func NewSseCase(baseCase *biz.BaseCase, sse *coresse.SSE) *SseCase {
	return &SseCase{
		BaseCase: baseCase,
		sse:      sse,
	}
}

// SubscribeSse 订阅 SSE 事件流。
func (h *SseCase) SubscribeSse(ctx context.Context, req *basev1.SubscribeSseRequest) (*emptypb.Empty, error) {
	err := h.sse.Serve(ctx, req.GetStream(), req.GetChannelId())
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
