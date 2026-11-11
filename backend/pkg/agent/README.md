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
client := agent.NewAssistantClient(modelConfig)
runtime := agent.NewRuntime(agent.RuntimeConfig{
	Client:     client,
	AdminTools: []agent.Tool{searchTool},
})
runtime.RegisterTool("admin", anotherTool)
```

宿主可通过 `RuntimeConfig.LocalizeMessage` 注入基于请求语言的消息本地化器，用于渲染工具禁用等面向用户的提示；未配置时使用英文兜底文案。

不接入权限系统时 `Checker` 保持 `nil`；需要按终端控制工具时实现
`agent.ToolAccessChecker`。

`agent.Tool` 是 Eino `tool.BaseTool`，支持标准字符串和增强结构化工具。工具发现使用 Eino 原生 `tool_search`，只检索当前已授权工具，不再维护固定数量的关键词候选。

按请求选择插件或连接时，先解析工具和逐次授权器，然后绑定上下文：

```go
ctx = agent.ContextWithToolPool(ctx, agent.ToolPool{
 Tools: selectedTools,
 Checker: checker,
 Revision: configurationRevision,
})
response, err := runtime.Run(ctx, input)
```

显式空 `Tools` 允许普通聊天；未绑定请求池时沿用静态登记和 `RuntimeConfig.Checker`。`RegisteredTools(terminal)` 返回静态工具切片副本，供宿主解析内置插件。`Revision` 必须绑定插件、安装策略、工具契约和连接修订，并在审批恢复前重算；变化会拒绝恢复，禁止重放旧确认。请求池中的执行器只绑定服务端已选连接，不能由模型替换。

Runtime 会把调用终端写入工具上下文；工具通过 `agent.TerminalFromContext(ctx)` 读取 `admin` 或 `app`。其他受信工具入口可用 `agent.ContextWithTerminal` 传入终端，数据权限仍由认证身份和业务规则决定。

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

Runtime 也会通过 `agent.ContextWithAssistantClient` 传递本轮已解析的模型客户端。BI 等业务工具可用 `agent.AssistantClientFromContext` 沿用本轮模型；直接业务请求再按自身的模型参数解析。

## 人工确认

宿主的 `Checker.PrepareToolCall` 必须在实际执行前重新授权、补全租户并按 Proto 校验参数，返回是否需要人工确认。配置 `RuntimeConfig.CheckPointStore`，并给 `RuntimeInput.CheckPointID` 分配服务端运行编号后，写调用会通过 Eino `StatefulInterrupt` 返回 `Response.Pending`。已配置 Checker 的受保护 Runtime 会拒绝没有持久化上下文的直接 `InvokeTool` 写调用；未配置 Checker 的 SDK 宿主须自行提供工具授权和审批边界。

宿主负责持久化 `Pending.State` 和全部 `Pending.Calls`，绑定用户、租户、会话、消息、一次性确认编号与有效期。确认接口必须先原子领取执行权，再调用 `Runtime.Resume`；不得允许客户端提交工具名称、参数或 Eino 中断地址。恢复后仍会校验最新权限和冻结参数，新的写调用需要再次确认。当前管理端的拒绝操作直接结束该轮，不再调用模型。

`CheckPointStore` 本身不提供业务幂等；不得自动重放执行中的运行。附件随 checkpoint 保存，此模式使用内联图片；PDF 在 Responses 协议下内联传入，Chat Completions 协议使用业务层提取的 `Attachment.Content` 文本。模型供应商须支持对应输入格式。未经沙箱隔离的 shell、Python 等执行类工具禁止注册到业务 Agent。

持久化模式在调用模型前限制 JSON 编码后的输入为 28 MiB（包括系统提示、历史、正文和附件编码）。当前数据库适配器使用现有 zstd 依赖压缩重复上下文，未压缩快照上限 192 MiB，压缩后上限 48 MiB；要求 LONGBLOB 与至少 64 MiB 的 MySQL `max_allowed_packet`。管理端附件合计上限为 20 MiB，文本和历史仍占用输入额度。
