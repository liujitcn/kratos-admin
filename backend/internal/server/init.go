package server

import "github.com/google/wire"

// ModuleProviderSet 汇总可复用于外部启动器的模块运行时提供者。
var ModuleProviderSet = wire.NewSet(
	NewOpenAPIRegistry,
	NewOpenAPIReady,
	NewHTTPMiddleware,
	NewMCPHandler,
	NewMCPToolsReady,
	NewAgentToolsReady,
	NewGRPCMiddleware,
)

// ProviderSet 汇总 Backend 独立传输服务提供者。
var ProviderSet = wire.NewSet(
	ModuleProviderSet,
	NewGRPCServer,
	NewHTTPServer,
)
