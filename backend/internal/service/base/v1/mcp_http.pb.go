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

const OperationMcpServiceHandleMcp = "/base.v1.McpService/HandleMcp"

type McpServiceHTTPServer interface {
	// HandleMcp 处理MCP Streamable HTTP请求
	HandleMcp(context.Context, *basev1.HandleMcpRequest) (*emptypb.Empty, error)
}

func RegisterMcpServiceHTTPServer(s *http.Server, srv McpServiceHTTPServer) {
	r := s.Route("/")
	r.GET("/mcp/{terminal}", _McpService_HandleMcp0_HTTP_Handler(srv))
	r.DELETE("/mcp/{terminal}", _McpService_HandleMcp1_HTTP_Handler(srv))
	r.POST("/mcp/{terminal}", _McpService_HandleMcp2_HTTP_Handler(srv))
}

func _McpService_HandleMcp0_HTTP_Handler(srv McpServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in basev1.HandleMcpRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationMcpServiceHandleMcp)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.HandleMcp(ctx, req.(*basev1.HandleMcpRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		_ = out.(*emptypb.Empty)
		return nil
	}
}

func _McpService_HandleMcp1_HTTP_Handler(srv McpServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in basev1.HandleMcpRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationMcpServiceHandleMcp)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.HandleMcp(ctx, req.(*basev1.HandleMcpRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		_ = out.(*emptypb.Empty)
		return nil
	}
}

func _McpService_HandleMcp2_HTTP_Handler(srv McpServiceHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in basev1.HandleMcpRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		if err := ctx.BindVars(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationMcpServiceHandleMcp)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.HandleMcp(ctx, req.(*basev1.HandleMcpRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		_ = out.(*emptypb.Empty)
		return nil
	}
}
