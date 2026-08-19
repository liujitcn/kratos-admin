package module

import (
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/ai"
	"github.com/liujitcn/kratos-admin/backend/internal/server/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/server/system/app/v1"
)

// ParseAdminAgentTools 创建管理端服务导出的 Eino 工具。
func ParseAdminAgentTools(services admin.Services) (ai.AdminTools, error) {
	return services.AgentTools()
}

// ParseAppAgentTools 创建应用端服务导出的 Eino 工具。
func ParseAppAgentTools(services app.Services) (ai.AppTools, error) {
	return services.AppAgentTools()
}
