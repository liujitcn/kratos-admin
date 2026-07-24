# Eino Agent Adapter

`pkg/agent/eino` 是当前项目对 CloudWeGo Eino 的适配层。业务包不直接堆 Eino 细节，而是通过这里按 Quick Start 的组件路线使用 ChatModel、Agent/Runner、Tool、Middleware、Callback 和 Workflow。

这个目录的目标有三个：

1. 收拢 Eino 原生 import，避免 `ai`、`server` 等业务包直接感知第三方框架细节。
2. 按组件能力分包，让“模型调用、消息转换、工具执行、运行循环、统计记录、固定流程”各有清晰边界。
3. 给后续接入其他 Agent 框架预留形态：可以按同样能力边界新增平行适配层，而不是继续把框架代码散落到业务运行时里。

## 推荐阅读顺序

第一次看这个目录时，建议按下面顺序读。这个顺序从基础类型到完整运行链路，读完基本可以回答“AI 助手和结构化任务怎么运行、固定流程在哪定义”。

| 顺序 | 位置 | 先看什么 | 读完应理解什么 |
| --- | --- | --- | --- |
| 1 | `model` | `client.go`、`options.go` | 模型客户端怎么创建，Responses 服务端工具选项在哪里注入。 |
| 2 | `message` | `message.go` | 项目内如何创建系统/用户/助手/工具消息，如何处理多模态、工具调用、流式合并和 token。 |
| 3 | `callback` | `recorder.go` | Eino Callback 如何接入 ADK 执行链，并把模型调用、Token、错误、服务端工具统一记录。 |
| 4 | `middleware` | `model.go`、`tool.go`、`error.go` | ADK 调用链上如何做模型重试、工具筛选、Responses 服务端工具注入、工具错误 JSON 和工具指标记录。 |
| 5 | `tool` | `tool.go` | 生成工具与手写运行时之间的接口、工具目录、直接工具调用协议。 |
| 6 | `adk` | `runner.go` | AI 助手主循环：如何创建 ChatModelAgent、挂中间件、消费事件流、回传 SSE 文本。 |
| 7 | `structured` | `runner.go`、`prompt.go`、`part.go` | 结构化任务如何组 Schema Prompt、发起模型调用并解析 JSON。 |
| 8 | `workflow` | `flow.go` | 已注册模块的固定流程如何通过 Eino Workflow/Graph 路由并执行。 |
| 9 | 上层调用方 | `service/base/agent/ai` | 基础 AI 运行时如何只保留协议、提示词、会话和响应结构，把 Eino 能力交给本目录。 |

## 主链路

### AI 助手对话链路

1. `service/base/agent/ai.Runtime` 负责读取会话、历史消息、附件、工具开关和前端 SSE 协议。
2. `ai.Runtime` 使用 `message` 构造 Eino 可消费的 `AgenticMessage`。
3. `ai.Runtime.toolInfos` 先按终端、工具启用状态和业务策略选出本轮候选工具。
4. `adk.Runner` 创建 Eino ADK `ChatModelAgent`，把模型、工具池和中间件挂进去。
5. `middleware.ToolFilterHandler` 在模型调用前做最终工具裁剪，保证模型只看到本轮允许的工具。
6. `middleware.ResponsesServerToolHandler` 给模型调用注入 Responses 服务端工具选项，并记录服务端工具事件。
7. `adk.Runner` 通过 `adk.WithCallbacks(callback.NewHandler())` 接入 Eino 原生 Callback，模型耗时、Token、错误和服务端工具写入 `callback.Recorder`。
8. `middleware.ToolMetricsHandler` 把工具耗时、输入、输出和错误写入 `callback.Recorder`。
9. `adk.Runner` 消费 ADK 事件流，把可见文本增量透传给 `ai.Runtime` 的 SSE 回调。
10. `ai.Runtime` 把最终消息、Token 和工具记录转换回现有前端协议。

### 工具直接调用链路

1. 前端或运行时拿到模型输出中的工具调用。
2. `ai.Runtime.InvokeTool` 把工具名和 JSON 参数传给 `tool.ExecuteCall`。
3. `tool.ExecuteCall` 校验工具是否在本轮启用列表中，再调用生成工具或手写工具的 `InvokableRun`。
4. 成功结果原样作为工具输出；失败结果统一走 `middleware.MarshalToolError`，输出稳定 JSON。
5. 调用方继续沿用当前会话、消息、SSE 和工具记录协议。

### 结构化任务链路

1. 上层运行时只组织结构化任务输入和提示词。
2. `structured.Runner` 使用 `structured.Part` 拼多模态输入，使用 `structured.SchemaPrompt` 生成结构化输出约束。
3. `model.ChatClient` 发起模型请求。
4. `structured.DecodeContent` 从模型返回中提取 JSON 并反序列化到业务结构体。

### 固定流程链路

1. `service/base/agent/ai` 通过 `workflow.Registry` 管理当前启用模块提供的固定流程。
2. `workflow.Registry` 内部编译 `compose.Workflow`，`Lookup` 用于查询动作所属流程、步骤和入口定义。
3. `workflow.Registry` 同时编译 `compose.Graph`，`Run` 会按 `flow + action_type` 选择动作节点，输出命中的动作元信息。
4. Runner 作为 typed handler 执行已注册模块的流程，业务输入通过当前动作的 payload 传递。
5. 前端动作携带自己的 `flow + action_type + payload`，Graph 按请求上下文路由，流程之间不会共享运行状态。

## 子包说明

### `adk`

`adk` 是 AI 助手的 Eino ADK 入口，核心文件是 `runner.go`。

- 负责创建 Eino ADK `ChatModelAgent`。
- 负责把模型、工具池、中间件、模型重试配置和最大循环次数组合起来。
- 负责消费 ADK 事件流，区分流式消息、非流式消息、工具中间消息和最终助手消息。
- 负责把可见文本增量回调给 SSE，不直接关心前端响应结构。

它不负责会话落库、前端协议、业务工具筛选策略和结构化任务输入，这些仍留在对应业务包中。

### `middleware`

`middleware` 是 ADK 调用链上的横切能力集合。

- `model.go` 放模型相关中间件：模型重试配置、Responses 服务端工具注入。
- `tool.go` 放工具相关中间件：工具定义筛选、工具执行耗时记录、工具错误记录。
- `error.go` 放工具错误 JSON 的稳定输出，保证 ADK 工具调用和直接工具调用使用同一错误格式。

后续新增“工具黑白名单、模型降级、工具熔断、审计采样”等横切能力时，优先放在这里，不继续堆到 `ai/runtime.go`。

### `callback`

`callback` 是 Eino Callback 与项目内调用记录的桥接层，核心文件是 `recorder.go`。

- `NewHandler` 创建 Eino `callbacks.Handler`，通过 `adk.WithCallbacks` 接入 ADK 执行链。
- `Recorder` 通过 context 贯穿一次 Agent 调用。
- `RecordModel` 记录模型调用模式、耗时、Token 和错误。
- `RecordTool` 记录函数工具调用的名称、标题、输入、输出、耗时、状态和错误。
- `RecordServerTools` 记录 Responses 服务端工具，例如联网搜索。
- `TotalToken` 汇总多次模型调用的 token，用于回填现有助手协议。

它只记录事实，不决定前端如何展示。展示字段仍由 `ai.Runtime` 转换成当前协议。

### `workflow`

`workflow` 是固定流程的 Eino Workflow/Graph 适配层，核心文件是 `flow.go`。流程名称和动作由启用模块提供，基础层只负责校验、索引和执行编排。

注册表会编译两条 Eino 编排：

- `compose.Workflow`：服务 `Lookup`，用于动作定义查询、入口解析和跨流程校验。
- `compose.Graph`：服务 `Run`，用于按 `flow + action_type` 选择具体动作节点，再由调用方传入 typed handler 执行业务动作。

动作查询分两类：

- `Action(flow, actionType)`：查询指定流程内动作，适合前端按钮回传了流程上下文的场景。
- `UniqueAction(actionType)`：只查询全局唯一动作；如果同一个动作类型出现在多个流程中会返回未命中，避免列表型流程漏传 `flow` 后误入第一个流程。

当前启用流程的每一步已经由 Graph 路由到独立节点；Graph 输入只保留流程动作和 payload，不携带运行时对象，后续如果要加入人工确认、Interrupt/Resume 或跨 Agent 协作，可以继续在这里扩展 Graph。

### `model`

`model` 负责模型客户端和模型调用选项。

- `client.go` 封装当前聊天模型客户端，给 `adk` 和 `structured` 共用。
- `options.go` 封装 Responses 服务端工具选项，避免上层到处写模型厂商参数。

业务包不应直接拼模型 Option；需要新增模型能力时，优先在这里形成项目内选项。

### `message`

`message` 是 Eino `AgenticMessage` 的项目内门面。

- 创建系统消息、用户消息、助手消息和工具结果消息。
- 支持文本、图片、多模态输入片段。
- 提取助手纯文本、工具调用、服务端工具事件和 token usage。
- 合并流式 chunk，给 `adk.Runner` 生成最终助手消息。

只要业务代码需要构造或读取 Agent 消息，优先通过这里，不直接操作 Eino schema 的复杂字段。

### `tool`

`tool` 连接生成 Agent Tool、手写运行时和 ADK 工具池。

- 通过类型别名暴露项目内使用的工具接口和工具定义。
- `NewCatalogTool` 创建工具目录工具，让模型能查询当前终端完整工具列表和启用状态。
- `ExecuteCall` 提供直接工具调用入口，保持与 ADK 工具调用相同的成功/失败输出协议。
- `NameSet`、`HasInfo`、`Title` 等函数用于工具集合判断和展示标题解析。

这里负责工具协议和调用适配，不负责某个具体业务工具的实现。

### `structured`

`structured` 负责“非多轮工具循环”的结构化模型任务。

- `runner.go` 发起结构化模型调用并解析 JSON。
- `prompt.go` 生成 JSON Schema 提示词。
- `part.go` 组装文本、图片等多模态输入片段。

它和 `adk` 的区别是：`structured` 面向一次性结构化输出，不执行工具循环；`adk` 面向 AI 助手对话和工具循环。

## 功能对应关系

| 你要找的能力 | 当前体现位置 | 说明 |
| --- | --- | --- |
| ADK ChatModelAgent / Runner | `adk/runner.go` | 替换助手手写工具循环的主入口，负责创建 ChatModelAgent、挂工具和中间件、消费事件流。 |
| Middleware | `middleware/model.go`、`middleware/tool.go`、`middleware/error.go` | 工具筛选、模型重试、Responses 服务端工具注入、工具统计、工具错误 JSON 都在这里。 |
| Callback | `callback/recorder.go` | 通过 Eino `callbacks.Handler` 统一记录模型调用、Token、错误和服务端工具，供上层协议回填。 |
| Workflow/Graph | `workflow/flow.go` | `Lookup` 使用 Eino `compose.Workflow` 查询动作定义；`Run` 使用 Eino `compose.Graph` 路由已注册固定流程动作，并由 typed handler 执行业务步骤。 |
| 结构化输出 | `structured/runner.go`、`structured/prompt.go`、`structured/part.go` | 一次性结构化任务使用这里，不走 ADK 工具循环。 |
| 工具目录和直接调用 | `tool/tool.go` | 工具接口、工具定义、工具目录、直接执行和稳定输出协议。 |
| 模型客户端和选项 | `model/client.go`、`model/options.go` | 模型配置、Generate/Stream 入口、Responses 服务端工具参数。 |
| 消息转换 | `message/message.go` | 会话消息、多模态片段、工具调用、流式合并、Token 提取。 |

## 接入其他 Agent 框架时怎么做

后续如果接入其他 Agent 框架，建议沿用当前能力边界，而不是让业务包直接 import 新框架：

1. 新增平行适配目录，例如 `pkg/agent/{new_framework}`。
2. 保持 `model`、`message`、`tool`、`callback`、`middleware`、`workflow` 这些能力边界，不把横切能力塞回 `ai.Runtime`。
3. 先替换运行器入口，再逐步迁移模型选项、工具协议和统计记录。
4. 业务包继续只表达业务协议：会话、消息、SSE、工具记录、审核摘要、固定流程入口。
5. 如果新框架有 Graph/Workflow 引擎，优先接到 `workflow` 的流程定义后面，不直接改前端动作协议。

## 使用边界

- `service/base/agent/ai` 负责 AI 助手业务运行时、会话消息、工具候选策略、SSE 和前端需要的响应结构。
- 业务模块只负责输入、提示词和响应结构，不直接承担模型结构化输出细节。
- `server` 负责注册生成工具并交给运行时，不承担 Agent 编排。
- `pkg/agent/eino` 负责 Eino 适配能力，不写具体业务提示词和业务决策。
- Eino 原生 import 应尽量只出现在 `pkg/agent/eino` 和 `api/gen/*_agent_tool.go` 生成产物中。业务包如需模型、工具、结构化输出或固定流程能力，应通过本目录的项目内门面调用。

## 对照 CloudWeGo Quick Start

当前代码按 Quick Start 的渐进路线已经落地这些部分：

| Quick Start 能力 | 当前落地情况 |
| --- | --- |
| ChatModel 与 Message | `model` 创建 Eino AgenticModel，`message` 统一封装 AgenticMessage。 |
| Agent / Runner / AgentEvent | `adk.Runner` 创建 Eino ADK ChatModelAgent，通过 TypedRunner 消费 AgentEvent。 |
| Tool | `tool` 承接生成工具和目录工具，`adk.Runner` 把工具池交给 ChatModelAgent。 |
| Middleware | `middleware` 接入工具错误处理、工具筛选、Responses 服务端工具注入和模型重试。 |
| Callback / Trace | `callback.NewHandler` 通过 `adk.WithCallbacks` 接入 Eino Callback 链路，记录模型、工具、耗时、Token 和错误。 |
| Graph Tool / Workflow | `workflow.Lookup` 使用 `compose.Workflow`，`workflow.Run` 使用 `compose.Graph` 分支和动作节点承载已注册固定流程。 |

尚未接入的 Quick Start 后续能力：

- Memory：当前仍复用现有会话历史和摘要协议，未引入 Eino Memory 组件。
- Interrupt/Resume：当前确认动作由前端 action payload 驱动，未接入 Eino CheckPoint/Interrupt。
- Skill：当前没有把可复用知识包建成 `SKILL.md + reference` 形式。
- A2UI：当前继续复用项目已有 SSE、BlocksJSON 和工具记录协议，未替换成 Quick Start 的 A2UI 协议。
