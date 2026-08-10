package base

import (
	"context"
	"fmt"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/base"
	"github.com/liujitcn/kratos-core/pkg/errorsx"

	"github.com/go-kratos/kratos/v3/log"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SseService Base SSE 服务。
type SseService struct {
	basev1.UnimplementedSseServiceServer
	sseCase *biz.SseCase
}

// NewSseService 创建 Base SSE 服务。
func NewSseService(
	sseCase *biz.SseCase,
) *SseService {
	var ss = SseService{
		sseCase: sseCase,
	}
	return &ss
}

// SubscribeSse 订阅 SSE 事件流。
func (s *SseService) SubscribeSse(ctx context.Context, req *basev1.SubscribeSseRequest) (*emptypb.Empty, error) {
	res, err := s.sseCase.SubscribeSse(ctx, req)
	if err != nil {
		log.Error(fmt.Sprintf("SubscribeSse %v", err))
		return nil, errorsx.WrapInternal(err, "订阅SSE事件流失败")
	}
	return res, nil
}
