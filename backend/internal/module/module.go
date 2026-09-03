package module

import (
	"context"
	"fmt"
	"os"

	"github.com/go-kratos/kratos/v3/middleware"
	kratosGRPC "github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/go-kratos/kratos/v3/transport/http"
	biz "github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/logstream"
	"github.com/liujitcn/kratos-admin/backend/internal/server/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/log"
	serverlogstream "github.com/liujitcn/kratos-admin/backend/internal/server/middleware/logstream"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/oauth"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/passwordpolicy"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/securityheaders"
	"github.com/liujitcn/kratos-admin/backend/internal/server/middleware/sessionpolicy"
	"github.com/liujitcn/kratos-admin/backend/internal/server/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/server/system/app/v1"
	"github.com/liujitcn/kratos-core/module"
	"github.com/liujitcn/kratos-core/queue"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/transport/mcp"
	"google.golang.org/grpc"
)

// Module 聚合 Admin 注册到 Core 宿主的协议服务。
type Module struct {
	baseServices  *base.Services
	adminServices *admin.Services
	appServices   *app.Services
}

var _ module.Module = (*Module)(nil)

// NewModules 创建包含 Admin 服务的模块集合。
func NewModules(
	baseServices *base.Services,
	adminServices *admin.Services,
	appServices *app.Services,
	baseConfigCase *biz.BaseConfigCase,
	baseLoginPolicyCase *biz.BaseLoginPolicyCase,
) (module.Modules, error) {
	// 迁移可能新增系统配置，模块启动前刷新缓存，避免认证策略沿用旧快照。
	var err error
	err = baseConfigCase.RefreshBaseConfig(context.Background())
	if err != nil {
		return nil, err
	}
	err = baseConfigCase.RefreshHiddenBaseConfig(context.Background())
	if err != nil {
		return nil, err
	}
	err = baseLoginPolicyCase.RefreshBaseLoginPolicy(context.Background())
	if err != nil {
		return nil, err
	}
	err = logstream.InitializeRuntimeLogging()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "启动运行日志采集失败: %v\n", err)
	}
	return module.Modules{
		&Module{
			baseServices:  baseServices,
			adminServices: adminServices,
			appServices:   appServices,
		},
	}, nil
}

// NewQueueConsumers 提供 Admin 自身投递事件的队列消费者集合。
func NewQueueConsumers(baseMessageCase *biz.BaseMessageCase, logConsumer queueData.ConsumerFunc) queue.Consumers {
	return queue.Consumers{
		{Stream: "base.message.dispatch", Handler: baseMessageCase.HandleDispatchMessage},
		{Stream: logmiddleware.AdminEventStream(), Handler: logConsumer},
	}
}

// RegisterGRPC 注册 Admin 的全部 gRPC 服务。
func (m *Module) RegisterGRPC(registrar grpc.ServiceRegistrar) {
	if server, ok := registrar.(*kratosGRPC.Server); ok {
		policyMiddleware := passwordpolicy.NewMiddleware(m.adminServices.BaseUserRepository, m.adminServices.BaseCase.Cache)
		sessionMiddleware := sessionpolicy.NewMiddleware(m.adminServices.BaseCase, m.adminServices.UserToken)
		server.Use("/*", middleware.Chain(m.adminServices.LogMiddleware, sessionMiddleware, policyMiddleware))
		server.Use("/system.admin.v1.RuntimeLogService/*", middleware.Chain(m.adminServices.LogMiddleware, sessionMiddleware, policyMiddleware, serverlogstream.RuntimeAccessMiddleware()))
	}
	m.baseServices.RegisterGRPC(registrar)
	m.adminServices.RegisterGRPC(registrar)
	m.appServices.RegisterGRPC(registrar)
}

// RegisterHTTP 注册 Admin 的全部 HTTP 服务。
func (m *Module) RegisterHTTP(server *http.Server) {
	m.baseServices.RegisterHTTP(server)
	m.adminServices.RegisterHTTP(server)
	m.appServices.RegisterHTTP(server)
	policyMiddleware := passwordpolicy.NewMiddleware(m.adminServices.BaseUserRepository, m.adminServices.BaseCase.Cache)
	sessionMiddleware := sessionpolicy.NewMiddleware(m.adminServices.BaseCase, m.adminServices.UserToken)
	server.Use("/system.admin.v1.RuntimeLogService/*", middleware.Chain(
		oauth.NewIPMiddleware(m.adminServices.OauthClientRepository),
		oauth.NewClientMiddleware(m.adminServices.OauthClientRepository, m.adminServices.BaseAPICase),
		m.adminServices.LogMiddleware,
		sessionMiddleware,
		policyMiddleware,
		serverlogstream.RuntimeAccessMiddleware(),
	))
	// 外部开放授权接口需要在 Proto HTTP 绑定前解密请求体，并在响应写出前加密数据。
	// 模块路由已全部注册，包裹底层 Handler 可以覆盖所有动态生成的 HTTP 路由。
	server.Server.Handler = securityheaders.NewHandler(oauth.NewCryptoFilter(
		m.adminServices.OauthClientRepository,
		m.adminServices.Authenticator,
		m.adminServices.OauthCredentialProtector,
	)(server.Server.Handler))
}

// RegisterMCP 注册 Admin 的全部 MCP 工具。
func (m *Module) RegisterMCP(server *mcp.Server) {
	m.baseServices.RegisterMCP(server)
	m.adminServices.RegisterMCP(server)
	m.appServices.RegisterMCP(server)
}
