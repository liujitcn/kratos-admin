package adk

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// InitialDynamicToolState 复用动态发现中间件取得首轮实际可见定义和目录提示，不加载或执行业务工具。
func InitialDynamicToolState(ctx context.Context, tools []tool.BaseTool) (*adk.TypedChatModelAgentState[*schema.AgenticMessage], error) {
	state := &adk.TypedChatModelAgentState[*schema.AgenticMessage]{}
	if len(tools) == 0 {
		return state, nil
	}
	search, err := toolsearch.NewTyped[*schema.AgenticMessage](ctx, &toolsearch.Config{DynamicTools: tools, UseModelToolSearch: false})
	if err != nil {
		return nil, err
	}
	var runCtx *adk.ChatModelAgentContext
	_, runCtx, err = search.BeforeAgent(ctx, &adk.ChatModelAgentContext{})
	if err != nil {
		return nil, err
	}
	for _, item := range runCtx.Tools {
		var info *schema.ToolInfo
		info, err = item.Info(ctx)
		if err != nil {
			return nil, err
		}
		state.ToolInfos = append(state.ToolInfos, info)
	}
	// 纯预算预览隔离 ADK 运行局部状态，避免读取或初始化外层 Agent 的发现状态。
	_, state, err = search.BeforeModelRewriteState(context.Background(), state, nil)
	return state, err
}
