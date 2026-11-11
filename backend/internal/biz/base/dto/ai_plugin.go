package dto

import (
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-admin/backend/pkg/aiplugin"
	"github.com/liujitcn/kratos-kit/auth/data"
	"golang.org/x/oauth2"
)

// AiPluginOAuthState 在服务端TTL缓存绑定一次性授权、身份及配置版本。
type AiPluginOAuthState struct {
	Identity             data.UserTokenPayload `json:"identity"`
	ConnectionID         int64                 `json:"connection_id"`
	ConnectionRevision   int64                 `json:"connection_revision"`
	PluginRevision       int64                 `json:"plugin_revision"`
	ApprovedRevision     string                `json:"approved_revision"`
	InstallationRevision int64                 `json:"installation_revision"`
	Verifier             string                `json:"verifier"`
}

// AiPluginCredential 是仅保存在加密信封中的认证资料。
type AiPluginCredential struct {
	Token string        `json:"token,omitempty"`
	OAuth *oauth2.Token `json:"oauth,omitempty"`
}

// AiPluginResolvedTool 保存请求工具和实际执行前必须重新核实的引用。
type AiPluginResolvedTool struct {
	Plugin       *models.AiPlugin
	Installation *models.AiPluginInstallation
	Snapshot     *models.AiPluginTool
	Policy       *models.AiPluginToolPolicy
	Connection   *models.AiPluginConnection
	Binding      *aiplugin.ToolBinding
	Name         string
	Revision     string
}
