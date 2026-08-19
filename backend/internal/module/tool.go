package module

import (
	baseAI "github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	adminServer "github.com/liujitcn/kratos-admin/backend/internal/server/system/admin/v1"
	appServer "github.com/liujitcn/kratos-admin/backend/internal/server/system/app/v1"
)

// ParseAdminAgentTools 创建管理端服务导出的 Eino 工具。
func ParseAdminAgentTools(services adminServer.Services) (baseAI.AdminTools, error) {
	return services.AgentTools()
}

// ParseAppAgentTools 创建应用端服务导出的 Eino 工具。
func ParseAppAgentTools(services appServer.Services) (baseAI.AppTools, error) {
	return services.AppAgentTools()
}
