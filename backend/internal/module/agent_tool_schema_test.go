package module

import (
	"testing"

	"github.com/cloudwego/eino/components/tool"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

// TestRecursiveProviderAgentToolsCanBeConstructed 验证含递归 Struct 字段的 Provider 请求可生成 Agent Tool。
func TestRecursiveProviderAgentToolsCanBeConstructed(t *testing.T) {
	tests := []struct {
		name string
		new  func() (tool.InvokableTool, error)
	}{
		{
			name: "ai provider create",
			new:  func() (tool.InvokableTool, error) { return adminv1.NewAiProviderServiceCreateAiProviderAgentTool(nil) },
		},
		{
			name: "ai provider update",
			new:  func() (tool.InvokableTool, error) { return adminv1.NewAiProviderServiceUpdateAiProviderAgentTool(nil) },
		},
		{
			name: "message provider create",
			new: func() (tool.InvokableTool, error) {
				return adminv1.NewBaseMessageProviderServiceCreateBaseMessageProviderAgentTool(nil)
			},
		},
		{
			name: "message provider update",
			new: func() (tool.InvokableTool, error) {
				return adminv1.NewBaseMessageProviderServiceUpdateBaseMessageProviderAgentTool(nil)
			},
		},
		{
			name: "message template create",
			new: func() (tool.InvokableTool, error) {
				return adminv1.NewBaseMessageTemplateServiceCreateBaseMessageTemplateAgentTool(nil)
			},
		},
		{
			name: "message template update",
			new: func() (tool.InvokableTool, error) {
				return adminv1.NewBaseMessageTemplateServiceUpdateBaseMessageTemplateAgentTool(nil)
			},
		},
		{
			name: "oauth provider create",
			new: func() (tool.InvokableTool, error) {
				return adminv1.NewBaseOauthProviderServiceCreateBaseOauthProviderAgentTool(nil)
			},
		},
		{
			name: "oauth provider update",
			new: func() (tool.InvokableTool, error) {
				return adminv1.NewBaseOauthProviderServiceUpdateBaseOauthProviderAgentTool(nil)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := test.new()
			if err != nil {
				t.Fatal(err)
			}
			if actual == nil {
				t.Fatal("Agent Tool 未创建")
			}
		})
	}
}
