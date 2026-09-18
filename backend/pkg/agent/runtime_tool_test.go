package agent

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
)

// TestSelectFallbackToolInfos 验证内部工具不匹配时保底暴露联网搜索工具。
func TestSelectFallbackToolInfos(t *testing.T) {
	infos := []*tool.Info{
		{Name: "system_admin_v1_base_user_service_page_base_user", Desc: "分页查询用户列表"},
		{Name: "system_admin_v1_base_dict_service_option_base_dict", Desc: "查询字典列表"},
		{Name: aiWebSearchToolName, Desc: "联网搜索公开信息并返回结果摘要"},
	}
	result := selectFallbackToolInfos(infos)
	if len(result) != 1 {
		t.Fatalf("保底工具数量 = %d, 期望 1", len(result))
	}
	if result[0].Name != aiWebSearchToolName {
		t.Fatalf("保底工具名 = %q, 期望联网搜索工具", result[0].Name)
	}
}

// TestSelectFallbackToolInfosDisabled 验证禁用搜索工具后不再保底返回。
func TestSelectFallbackToolInfosDisabled(t *testing.T) {
	infos := []*tool.Info{
		{Name: "system_admin_v1_base_user_service_page_base_user", Desc: "分页查询用户列表"},
	}
	if result := selectFallbackToolInfos(infos); result != nil {
		t.Fatalf("禁用搜索后保底结果 = %v, 期望 nil", result)
	}
}

// TestSelectToolInfosFallback 验证无关问题走保底搜索工具而不是返回空工具集。
func TestSelectToolInfosFallback(t *testing.T) {
	infos := make([]*tool.Info, 0, maxModelToolsPerRequest+1)
	for i := 0; i <= maxModelToolsPerRequest; i++ {
		infos = append(infos, &tool.Info{Name: "system_admin_v1_base_user_service_page_base_user", Desc: "分页查询用户列表"})
	}
	infos = append(infos, &tool.Info{Name: aiWebSearchToolName, Desc: "联网搜索公开信息并返回结果摘要"})
	result := selectToolInfos(RuntimeInput{Content: "今天有什么热点新闻？"}, infos)
	if len(result) != 1 || result[0].Name != aiWebSearchToolName {
		t.Fatalf("无关问题工具集 = %v, 期望只保底联网搜索工具", result)
	}
}
