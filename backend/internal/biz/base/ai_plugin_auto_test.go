package biz

import (
	"strconv"
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
)

// TestAutomaticAppPluginSelections 验证自动引用不能扩大终端、效果或账号授权范围。
func TestAutomaticAppPluginSelections(t *testing.T) {
	for _, scenario := range []string{"allowed", "disabled", "unassembled", "admin_only", "write", "unavailable", "unreviewed", "foreign_account", "wrong_plugin", "expired", "disconnected"} {
		t.Run(scenario, func(t *testing.T) {
			definition := &basev1.AiPlugin{Id: "7", Source: "MCP", Enabled: true, Assembled: true, Installation: &basev1.AiPluginInstallation{Enabled: true, Terminals: []string{"app"}}}
			account := &basev1.AiPluginConnection{Id: "11", PluginId: "7", State: "connected", ExpiresAt: 200}
			tool := &basev1.AiPluginTool{PluginId: "7", ConnectionId: "11", State: "discovered", Effect: "read", EffectiveEnabled: true}
			accounts := []*basev1.AiPluginConnection{account}
			switch scenario {
			case "disabled":
				definition.Installation.Enabled = false
			case "unassembled":
				definition.Assembled = false
			case "admin_only":
				definition.Installation.Terminals = []string{"admin"}
			case "write":
				tool.Effect = "write"
			case "unavailable":
				definition.UnavailableReason = "撤销授权"
			case "unreviewed":
				tool.EffectiveEnabled = false
			case "foreign_account":
				accounts = nil
			case "wrong_plugin":
				account.PluginId = "8"
			case "expired":
				account.ExpiresAt = 100
			case "disconnected":
				account.State = "disconnected"
			}
			selections, err := automaticAppPluginSelections([]*basev1.AiPlugin{definition}, map[string][]*basev1.AiPluginTool{"7": {tool, tool}}, accounts, 100)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "allowed" {
				if len(selections) != 1 || selections[0].PluginId != "7" || selections[0].ConnectionId != "11" {
					t.Fatalf("未保持唯一的确定账号引用：%v", selections)
				}
			} else if len(selections) != 0 || selections == nil {
				t.Fatalf("无效能力必须返回明确的空选择，不可回退为默认工具：%v", selections)
			}
		})
	}
}

// TestAutomaticAppPluginSelectionsLimit 验证超过上限时明确失败，不静默截断账号范围。
func TestAutomaticAppPluginSelectionsLimit(t *testing.T) {
	definitions := make([]*basev1.AiPlugin, 0, 65)
	tools := make(map[string][]*basev1.AiPluginTool)
	for index := range 65 {
		id := strconv.Itoa(index + 1)
		definitions = append(definitions, &basev1.AiPlugin{Id: id, Source: "LOCAL", Enabled: true, Assembled: true, Installation: &basev1.AiPluginInstallation{Enabled: true, Terminals: []string{"app"}}})
		tools[id] = []*basev1.AiPluginTool{{PluginId: id, State: "discovered", Effect: "read", EffectiveEnabled: true}}
	}
	_, err := automaticAppPluginSelections(definitions, tools, nil, 100)
	if err == nil {
		t.Fatal("超过64个插件引用必须拒绝")
	}
}
