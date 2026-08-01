package kratosadmin

import (
	"github.com/liujitcn/kratos-admin/backend/core"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/localgrpc"
	coreOpenAPI "github.com/liujitcn/kratos-admin/backend/core/pkg/openapi"
	adminbiz "github.com/liujitcn/kratos-admin/backend/internal/biz/admin"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/job"
	"github.com/liujitcn/kratos-admin/backend/internal/server"
	"github.com/liujitcn/kratos-kit/bootstrap"
	"google.golang.org/grpc"
)

// Module 表示可注册服务并提供进程内 gRPC 客户端连接的 Backend 模块。
type Module interface {
	core.Module
	ClientConn() grpc.ClientConnInterface
}

// AdditionalModules 表示与 Backend 一起装配的扩展模块集合。
type AdditionalModules []core.Module

// Runtime 封装 Backend 服务注册、本地客户端和运行时贡献。
type Runtime struct {
	modules             server.Modules
	clientConn          *localgrpc.Conn
	httpMiddlewares     server.HTTPMiddlewares
	grpcMiddlewares     server.GRPCMiddlewares
	cronServer          *job.CronServer
	openAPIRegistry     *coreOpenAPI.Registry
	projectDocumentCase *adminbiz.ProjectDocumentCase
}

var _ Module = (*Runtime)(nil)

// NewModule 创建可挂载到其他启动器的 Backend 模块。
func NewModule(ctx *bootstrap.Context, optionValues ...Option) (*Runtime, func(), error) {
	opts := newOptions(optionValues)
	return initModule(
		ctx,
		opts.additionalModules,
		opts.configuredProjectDocuments,
		opts.databaseOptions,
		opts.migrations,
	)
}
