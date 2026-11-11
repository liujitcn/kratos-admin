package aiplugin

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
)

// Manifest 声明由组合根显式装配的本地插件；配置不能动态装载 Go 代码。
type Manifest struct {
	Code        string        // 插件稳定编码。
	Name        string        // 展示名称。
	Description string        // 功能说明。
	Source      string        // API 或 LOCAL。
	Revision    int64         // 契约修订号。
	Tools       []ToolBinding // 显式绑定的执行器。
}

// ToolBinding 将确定终端的工具绑定到插件和效果声明。
type ToolBinding struct {
	Terminal  string                                        // admin 或 app。
	Operation string                                        // API 工具的 Proto operation；LOCAL 留空。
	Effect    string                                        // read、write 或 unknown；未知效果必须确认。
	Tool      tool.BaseTool                                 // 标准或增强型 Eino 工具。
	Validate  func(context.Context, string) (string, error) // LOCAL 参数和宿主业务权限校验。
}
