package server

import (
	"github.com/liujitcn/kratos-admin/backend/core"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	appMiddleware "github.com/liujitcn/kratos-admin/backend/internal/server/middleware"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/logging"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	authnEngine "github.com/liujitcn/kratos-kit/auth/authn/engine"
	authzEngine "github.com/liujitcn/kratos-kit/auth/authz/engine"
	authData "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"github.com/liujitcn/kratos-kit/rpc"
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
	ms = append(ms, appMiddleware.NewLocaleMiddleware())
	// request-id、recovery、tracing、metadata 等框架拦截器由 rpc.CreateGrpcServer 按配置挂载。
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

	allMiddlewares := append(GRPCMiddlewares(nil), middlewares...)
	allMiddlewares = append(allMiddlewares, core.Modules(modules).GRPCMiddlewares()...)
	srv, err := rpc.CreateGrpcServer(cfg, allMiddlewares...)
	if err != nil {
		return nil, err
	}
	modules.RegisterGRPC(srv)

	return srv, nil
}
