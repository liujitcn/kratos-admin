package module

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/pkg/aiplugin"
)

// RegisterAIPlugin 显式登记本地执行器与归属，然后重建排除可选组件的内置快照。
func (m *Module) RegisterAIPlugin(manifest aiplugin.Manifest) error {
	err := m.aiPluginCase.RegisterManifest(manifest)
	if err != nil {
		return err
	}
	for _, binding := range manifest.Tools {
		err = m.aiRuntime.RegisterTools(binding.Terminal, binding.Tool)
		if err != nil {
			return err
		}
	}
	return m.aiPluginCase.SyncBuiltinTools(context.Background())
}
