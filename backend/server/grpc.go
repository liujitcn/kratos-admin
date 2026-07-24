package server

import (
	"github.com/liujitcn/kratos-admin/backend/pkg/gen/data"
	appMiddleware "github.com/liujitcn/kratos-admin/backend/pkg/middleware"
	"github.com/liujitcn/kratos-admin/backend/pkg/middleware/logging"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
	"github.com/liujitcn/kratos-kit/rpc/middleware/requestid"
)

// GRPCMiddlewares 表示 GRPC 服务中间件链。
type GRPCMiddlewares []middleware.Middleware

// NewGRPCMiddleware 创建 GRPC 服务统一中间件链。
func NewGRPCMiddleware(
	ctx *bootstrap.Context,
	authenticator authnEngine.Authenticator,
	baseUserRepo *data.BaseUserRepository,
	authorizer authzEngine.Engine,
	userToken *authData.UserToken,
	jwtCfg *bootstrapConfigv1.Authentication_Jwt,
) GRPCMiddlewares {
	var ms GRPCMiddlewares
	cfg := ctx.GetConfig()
	// 先补齐请求标识，再进入访问日志中间件，确保日志能读取到统一 request_id。
	ms = append(ms, requestid.NewRequestIDMiddleware())
	// 开启日志中间件时，统一挂载请求日志与操作者解析逻辑。
	if cfg != nil && cfg.Server != nil && cfg.Server.Grpc != nil && cfg.Server.Grpc.Middleware != nil && cfg.Server.Grpc.Middleware.EnableLogging {
		ms = append(ms, logging.Server(ctx.GetLogger(), baseUserRepo, authenticator))
	}
	ms = append(ms, appMiddleware.NewAuthMiddleware(authenticator, authorizer, userToken, jwtCfg))
	ms = append(ms, appMiddleware.NewValidateMiddleware())
	return ms
}

// NewGRPCServer 创建 GRPC Server 并注册已启用业务模块。
func NewGRPCServer(
	ctx *bootstrap.Context,
	middlewares GRPCMiddlewares,
	modules Modules,
	_ MCPToolsReady,
	_ AgentToolsReady,
) (*grpc.Server, error) {
	cfg := ctx.GetConfig()
	// 未启用 GRPC 配置时，跳过 GRPC 服务创建。
	if cfg == nil || cfg.Server == nil || cfg.Server.Grpc == nil {
		return nil, nil
	}

	srv, err := rpc.CreateGrpcServer(cfg, middlewares...)
	if err != nil {
		return nil, err
	}
	modules.RegisterGRPC(srv)

	return srv, nil
}
