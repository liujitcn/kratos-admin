package server

import "github.com/google/wire"

// ProviderSet 汇总服务端依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewHTTPMiddleware,
	NewMCPHandler,
	NewMCPToolsReady,
	NewAgentToolsReady,
	NewGRPCMiddleware,
	NewGRPCServer,
	NewHTTPServer,
)
