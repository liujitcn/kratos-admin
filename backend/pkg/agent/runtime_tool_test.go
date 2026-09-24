package agent

import (
	"strings"
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
	result, matched := selectToolInfos(RuntimeInput{Content: "今天有什么热点新闻？"}, infos)
	if matched {
		t.Fatal("无关问题不应标记为命中内部工具")
	}
	if len(result) != 1 || result[0].Name != aiWebSearchToolName {
		t.Fatalf("无关问题工具集 = %v, 期望只保底联网搜索工具", result)
	}
}

// TestSelectToolInfosPrefersInternalTool 验证命中内部工具时不向模型暴露联网搜索。
func TestSelectToolInfosPrefersInternalTool(t *testing.T) {
	internalInfo := &tool.Info{Name: "base_user_list", Desc: "分页查询用户列表"}
	infos := []*tool.Info{
		internalInfo,
		{Name: aiWebSearchToolName, Desc: "联网搜索公开信息并返回结果摘要"},
	}
	result, matched := selectToolInfos(RuntimeInput{Content: "查询用户列表"}, infos)
	if !matched {
		t.Fatal("匹配内部工具时应设置命中标记")
	}
	if len(result) != 1 || result[0].Name != internalInfo.Name {
		t.Fatalf("工具集 = %v, 期望只包含 %q", result, internalInfo.Name)
	}
}

// TestSelectToolInfosPrefersRevenueToolForRelativeDate 验证明确的营收日期查询优先命中内部工具。
func TestSelectToolInfosPrefersRevenueToolForRelativeDate(t *testing.T) {
	internalInfo := &tool.Info{Name: "manager_admin_v1_project_analytics_service_summary_project_overview", Desc: "SCP营收分析服务，支持今日营收/今天营收（当天）、明日营收/明天营收（次日）和本月营收（MONTH）。其他单日相对日期按会话业务日期及时区换算，单日将 start_date、end_date 设为同一天；查询营收概览指标。"}
	infos := []*tool.Info{
		internalInfo,
		{Name: aiWebSearchToolName, Desc: "联网搜索公开信息并返回结果摘要"},
	}
	for _, content := range []string{"查一下今日营收", "查一下今天的营收", "查一下明天营收", "查一下本月营收"} {
		result, matched := selectToolInfos(RuntimeInput{Content: content}, infos)
		if !matched {
			t.Errorf("问题 %q 应命中内部营收工具", content)
			continue
		}
		if len(result) != 1 || result[0].Name != internalInfo.Name {
			t.Errorf("问题 %q 的工具集 = %v, 期望只包含营收工具 %q", content, result, internalInfo.Name)
		}
	}
}

// TestSelectToolInfosPrefersInternalToolWithLargeCatalog 验证工具数超过模型上限时仍优先命中内部工具。
func TestSelectToolInfosPrefersInternalToolWithLargeCatalog(t *testing.T) {
	internalInfo := &tool.Info{Name: "base_user_list", Desc: "分页查询用户列表"}
	infos := []*tool.Info{internalInfo, {Name: aiWebSearchToolName, Desc: "联网搜索公开信息"}}
	for i := 0; i < maxModelToolsPerRequest; i++ {
		infos = append(infos, &tool.Info{Name: "unrelated_tool", Desc: "执行无关操作"})
	}
	result, matched := selectToolInfos(RuntimeInput{Content: "查询用户列表"}, infos)
	if !matched {
		t.Fatal("匹配内部工具时应设置命中标记")
	}
	if len(result) != 1 || result[0].Name != internalInfo.Name {
		t.Fatalf("工具集 = %v, 期望只包含 %q", result, internalInfo.Name)
	}
}

// TestSelectToolInfosNoInternalMatchKeepsSearch 验证未命中内部工具时保留联网搜索回退。
func TestSelectToolInfosNoInternalMatchKeepsSearch(t *testing.T) {
	infos := []*tool.Info{
		{Name: "base_user_list", Desc: "分页查询用户列表"},
		{Name: aiWebSearchToolName, Desc: "联网搜索公开信息并返回结果摘要"},
	}
	result, matched := selectToolInfos(RuntimeInput{Content: "今天有什么热点新闻？"}, infos)
	if matched {
		t.Fatal("未命中内部工具时不应设置命中标记")
	}
	if len(result) != len(infos) || result[1].Name != aiWebSearchToolName {
		t.Fatalf("工具集 = %v, 期望保留联网搜索工具", result)
	}
}

// TestSelectToolInfosKeepsInternalToolForFollowUp 验证短追问继续使用历史内部工具且不暴露联网搜索。
func TestSelectToolInfosKeepsInternalToolForFollowUp(t *testing.T) {
	internalInfo := &tool.Info{Name: "base_user_list", Desc: "分页查询用户列表"}
	infos := []*tool.Info{
		internalInfo,
		{Name: aiWebSearchToolName, Desc: "联网搜索公开信息并返回结果摘要"},
	}
	input := RuntimeInput{
		Content: "继续",
		History: []Message{{Tools: []ToolUsage{{Name: internalInfo.Name}}}},
	}
	result, matched := selectToolInfos(input, infos)
	if !matched {
		t.Fatal("延续历史内部工具时应设置命中标记")
	}
	if len(result) != 1 || result[0].Name != internalInfo.Name {
		t.Fatalf("工具集 = %v, 期望只包含历史内部工具 %q", result, internalInfo.Name)
	}
}

// TestSelectToolInfosKeepsInternalToolForDateFollowUp 验证“明天呢”沿用历史内部工具且不暴露联网搜索。
func TestSelectToolInfosKeepsInternalToolForDateFollowUp(t *testing.T) {
	internalInfo := &tool.Info{Name: "revenue_summary", Desc: "查询营收概览。日期条件：今天/今日、明天/明日、本月。"}
	infos := []*tool.Info{
		internalInfo,
		{Name: aiWebSearchToolName, Desc: "联网搜索公开信息并返回结果摘要"},
	}
	input := RuntimeInput{
		Content: "明天呢",
		History: []Message{{Tools: []ToolUsage{{Name: internalInfo.Name}}}},
	}
	result, matched := selectToolInfos(input, infos)
	if !matched {
		t.Fatal("日期追问应沿用历史内部工具")
	}
	if len(result) != 1 || result[0].Name != internalInfo.Name {
		t.Fatalf("工具集 = %v, 期望只包含历史营收工具 %q", result, internalInfo.Name)
	}
}

// TestSelectToolInfosKeepsSearchForUnmatchedShortFollowUp 验证未命中历史工具描述的短追问保留联网搜索。
func TestSelectToolInfosKeepsSearchForUnmatchedShortFollowUp(t *testing.T) {
	internalInfo := &tool.Info{Name: "revenue_summary", Desc: "查询营收概览"}
	searchInfo := &tool.Info{Name: aiWebSearchToolName, Desc: "联网搜索公开信息并返回结果摘要"}
	input := RuntimeInput{
		Content: "天气呢",
		History: []Message{{Tools: []ToolUsage{{Name: internalInfo.Name}}}},
	}
	result, matched := selectToolInfos(input, []*tool.Info{internalInfo, searchInfo})
	if matched {
		t.Fatal("未匹配历史工具描述的短追问不应屏蔽联网搜索")
	}
	if len(result) != 2 || result[0].Name != internalInfo.Name || result[1].Name != searchInfo.Name {
		t.Fatalf("工具集 = %v, 期望同时包含历史工具和联网搜索", result)
	}
}

// TestSelectToolInfosKeepsSearchForPublicTemporalQuestion 验证带公开信息意图的日期问题仍保留联网搜索。
func TestSelectToolInfosKeepsSearchForPublicTemporalQuestion(t *testing.T) {
	infos := []*tool.Info{
		{Name: "revenue_summary", Desc: "查询营收概览"},
		{Name: aiWebSearchToolName, Desc: "联网搜索公开信息并返回结果摘要"},
	}
	input := RuntimeInput{
		Content: "明天有什么热点新闻？",
		History: []Message{{Tools: []ToolUsage{{Name: "revenue_summary"}}}},
	}
	result, matched := selectToolInfos(input, infos)
	if matched {
		t.Fatal("公开新闻问题不应因历史内部工具而标记为命中")
	}
	if len(result) != len(infos) || result[1].Name != aiWebSearchToolName {
		t.Fatalf("工具集 = %v, 期望保留联网搜索工具", result)
	}
}

// TestResolvePromptKeepsDateRulesGeneric 验证系统提示词提供运行时日期且不固化业务日期映射。
func TestResolvePromptKeepsDateRulesGeneric(t *testing.T) {
	prompt := (&Runtime{}).resolvePrompt(RuntimeInput{})
	for _, expected := range []string{"当前业务日期：", "业务时区：", "遵循该工具的描述和参数默认值", "只修改用户补充的条件"} {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("系统提示词缺少 %q", expected)
		}
	}
	for _, unexpected := range []string{"明天呢", "本月呢", "后天、本周"} {
		if strings.Contains(prompt, unexpected) {
			t.Fatalf("系统提示词不应固化业务日期示例 %q", unexpected)
		}
	}
}
