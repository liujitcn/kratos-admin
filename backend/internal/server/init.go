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
