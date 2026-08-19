# Agent 公共 API

其他 Backend 模块不需要引用 `internal` 路径，直接使用 `pkg/agent` 即可接入 AI。

## 添加工具

优先使用 Eino 的 `InferTool`，输入结构会自动生成工具参数 schema：

```go
type SearchRequest struct {
	Keyword string `json:"keyword" jsonschema_description:"搜索关键词"`
}

searchTool, err := agent.InferTool[SearchRequest, string](
	"search_records",
	"搜索业务记录",
	func(ctx context.Context, request SearchRequest) (string, error) {
		return searchRecords(ctx, request.Keyword)
	},
)
if err != nil {
	return err
}
```

创建 Runtime 时传入工具，或在运行时追加：

```go
client := agent.NewResponsesClient(modelConfig)
runtime := agent.NewRuntime(agent.RuntimeConfig{
	Client:     client,
	AdminTools: []agent.Tool{searchTool},
})
runtime.RegisterTool("admin", anotherTool)
```

不接入权限系统时 `Checker` 保持 `nil`；需要按终端控制工具时实现
`agent.ToolAccessChecker`。

## 结构化输出

评论审核、内容提取等业务可以复用公共聊天客户端和结构化运行器，不需要引用
`internal/biz/agent`：

```go
client := agent.NewChatClient(modelConfig)
runner := agent.NewStructuredRunner(client)
schema, err := agent.SchemaFor[ReviewResult]()
if err != nil {
	return err
}
err = runner.Generate(ctx, instruction, []*agent.Part{
	agent.TextPart(content),
	agent.ImageURLPart(imageURL),
}, schema, &result)
```
