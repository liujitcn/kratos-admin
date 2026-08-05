# Eino Agent 适配层

`internal/agent` 隔离 CloudWeGo Eino 细节，供 `internal/biz/base/ai` 使用。它不负责会话落库、终端权限和前端协议，这些仍属于业务运行时。

## 子包

| 子包 | 职责 |
| --- | --- |
| `model` | Chat/Responses 模型客户端和模型选项。 |
| `message` | `AgenticMessage`、多模态内容、工具调用、流式合并和 Token 提取。 |
| `adk` | 创建 `TypedChatModelAgent`、运行事件循环并回调可见文本增量。 |
| `middleware` | 模型重试、Responses 服务端工具、工具筛选、错误 JSON 和指标。 |
| `callback` | 记录模型调用、Token、函数工具、服务端工具、耗时和错误。 |
| `tool` | Eino Tool 类型门面、工具目录和直接调用。 |
| `structured` | JSON Schema 提示、结构化模型调用和结果反序列化。 |
| `workflow` | 固定流程定义、动作查询和基于 Eino Workflow/Graph 的路由执行。 |

## 对话链路

1. `biz/base/ai.Runtime` 组装历史、附件、提示词和候选工具。
2. `message` 转成 Eino `AgenticMessage`。
3. `adk.Runner` 创建 ChatModelAgent，挂载 Callback 和 Middleware。
4. `ToolFilterHandler` 在模型调用前只保留本轮允许的工具。
5. `ResponsesServerToolHandler` 注入服务端工具选项。
6. `ToolMetricsHandler` 和 `callback.Recorder` 记录模型与工具事实。
7. Runner 消费事件流，将可见文本增量回调给业务 SSE emitter。
8. 业务运行时把最终内容、Token 和工具记录保存为当前协议。

`adk` 不处理数据库、租户或前端消息结构；`callback` 只记录调用事实，不决定展示。

## 直接工具调用

`biz/base/ai.Runtime.InvokeTool` 按终端取工具并交给 `tool.ExecuteCall`。执行前必须确认工具在本轮启用列表中；失败统一通过 `middleware.MarshalToolError` 返回稳定 JSON。Agent 循环和直接调用使用同一套工具开关与错误格式。

## 结构化任务

`structured.Runner` 使用 `Part` 组织文本或图片输入，通过 JSON Schema 构造约束提示，调用 `model.ChatClient` 后解析 JSON 到业务结构。适合无需 Agent 工具循环的单次结构化生成。

## 固定流程

`workflow.Registry` 校验并索引调用方提供的 `Definition`：

- `Lookup` 和 `Action` 查询流程、入口和动作。
- `UniqueAction` 只返回全局唯一动作，避免缺少 flow 时误匹配。
- `Run` 按 `flow + action_type` 路由，再调用业务传入的 typed handler。

流程定义和业务执行器由上层模块提供；适配层不内置具体业务流程。

## 边界

- 新的 Eino import、模型厂商选项和 Agent Middleware 优先收敛到本目录。
- 业务工具筛选、数据库状态、终端权限和会话事务保留在 `internal/biz/base/ai`。
- 工具由生成代码或明确的运行时适配器提供，不在此复制 Proto 服务逻辑。
- 更换 Agent 框架时可按同样的 model/message/tool/runner/workflow 边界增加平行适配，不改变上层会话协议。
