package base

import (
	context "context"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"

	http "github.com/go-kratos/kratos/v3/transport/http"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the kratos package it is being compiled against.
var _ = new(context.Context)

const _ = http.SupportPackageIsVersion3

const OperationSseServiceSubscribeSse = "/base.v1.SseService/SubscribeSse"

type SseServiceHTTPServer interface {
	// SubscribeSse 订阅SSE事件流
	SubscribeSse(context.Context, *basev1.SubscribeSseRequest) (*emptypb.Empty, error)
}

func RegisterSseServiceHTTPServer(s *http.Server, srv SseServiceHTTPServer) {
	r := s.Route("/")
	r.GET("/events/{stream}", _SseService_SubscribeSse0_HTTP_Handler(srv))
}

func _SseService_SubscribeSse0_HTTP_Handler(srv SseServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in basev1.SubscribeSseRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationSseServiceSubscribeSse)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.SubscribeSse(ctx, req.(*basev1.SubscribeSseRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		_ = out.(*emptypb.Empty)
		return nil
	}
}
