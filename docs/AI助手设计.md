# AI 助手设计

## 文档目标

本文档说明 AI 助手在管理后台的产品定位、页面结构、快捷操作、会话消息和流程动作。

## 能力定位

AI 助手是管理后台的对话式工作入口，目标是把常见的系统运维、代码生成和资料查询流程收敛到同一套会话内：

- 管理后台面向运营、客服和管理员，强调高频工作流的快速触发和可追踪执行。
- 后端保持统一会话、消息、快捷入口和流程动作协议。
- 前端点击快捷入口时发送 `text + action`，不在浏览器伪造业务结果。

## 总体链路

~~~mermaid
flowchart LR
  Shortcut["快捷入口"] --> Composer["输入框 / 发送载荷"]
  UserMessage["用户消息"] --> MessageAPI["AI 助手消息接口"]
  Composer --> MessageAPI
  MessageAPI --> Stream["流式回复"]
  Stream --> AiMessage["助手消息"]
  AiMessage --> Blocks["结构化流程块"]
  Blocks --> Action["流程动作"]
  Action --> MessageAPI
~~~

- 快捷入口来自 `ListAiShortcut`，按当前终端和权限过滤后返回。
- 助手消息可以包含 Markdown、工具调用摘要、附件和结构化流程块。
- 结构化流程动作继续通过消息接口提交，保证流程上下文在会话内闭环。

## 后端契约与运行时

AI 公共接口位于 `backend/api/proto/base/v1/ai_tool.proto`、`ai_session.proto` 与 `ai_message.proto`，路径前缀为 `/api/v1/base/ai`。快捷入口由 `AiToolService` 提供，会话与消息列表由 `AiSessionService` 提供，发送、删除、重试和重新生成由 `AiMessageService` 提供。发送消息使用会话级 SSE 响应，不占用工作台通用 `/events` 流。

会话与消息分别持久化为 `ai_session`、`ai_message`。每轮消息保存输入、输出、附件、工具调用、Token、首 Token 耗时、总耗时和生成状态。失败消息可重试，成功输出可重新生成，任意锚点消息可创建持久化分支会话。

后端运行时位于 `service/base/agent/ai`，Eino 适配层位于 `pkg/agent/eino`。运行时按终端筛选启用的生成工具，再通过同一进程内服务实例执行；MCP 是否暴露由独立的 `mcp_status` 控制。公开实时信息可使用 Responses 的联网搜索工具，结构化任务使用独立的模型调用链。

## 管理后台设计

管理后台入口位于 `frontend/admin/src/views/base/ai/chat`，主要组件包括：

| 组件 | 职责 |
| --- | --- |
| `index.vue` | 会话列表、消息加载、发送、流式回复、流程动作分发 |
| `ChatPanel.vue` | 空态、消息列表、快捷操作、输入框、消息级操作 |
| `SessionPanel.vue` | 会话搜索、创建、切换、重命名和删除 |
| `FlowBlocks.vue` | 渲染结构化流程卡片和动作按钮 |
| `XSender.vue` | 文本、附件和发送状态 |

### 页面布局

~~~mermaid
flowchart TB
  Page["AI 助手页面"] --> Session["左侧会话栏"]
  Page --> Chat["右侧聊天区"]
  Chat --> Empty["空态欢迎区"]
  Chat --> Messages["消息列表"]
  Chat --> Quick["快捷操作区"]
  Chat --> Sender["输入区"]
~~~

- 空态展示欢迎语、快捷操作面板和输入框，适合新会话快速开始。
- 已有消息时，快捷操作以紧凑横向入口展示在输入框上方，保持随时可触达。
- 消息区保留工具调用、运行明细、复制、删除、编辑、重试和分支会话等管理能力。

### 快捷操作规则

- 快捷操作是否展示由“快捷入口加载中或已有可用入口”决定，不依赖当前会话是否为空。
- 空态面板使用两列网格，突出常用工作流。
- 会话中使用紧凑横向滚动条，降低占屏高度。
- 点击快捷操作复用普通发送链路，统一调用 `handleShortcutClick` 并补齐 `AiAction`。

## 数据与状态约定

| 数据 | 来源 | 前端处理 |
| --- | --- | --- |
| 快捷入口 | `ListAiShortcut` | 按 `sort` 排序，过滤无 key 且无标题或 prompt 的异常项 |
| 会话列表 | `ListAiSession` | 支持创建、切换、搜索、重命名、删除 |
| 消息列表 | `ListAiMessage` | 按当前会话缓存，切换会话时加载 |
| 流式回复 | 消息流接口 | 合并增量，维护生成态和失败态 |
| 流程动作 | 助手消息 blocks | 点击后携带 action 回传消息接口 |

快捷入口的展示状态只和快捷入口数据有关，不应绑定 `messages.length === 0`。空态和会话中可以使用不同视觉密度，但必须复用同一批终端快捷入口和同一套发送链路。

流程动作必须回传 `source_message_id`、`action_id`、`flow_version`。服务端只接受当前会话最新成功消息的 `blocks_json` 中仍然存在的动作，阻止历史消息中的操作被重复执行。

## 维护与验证

- 修改管理后台 AI 助手页面后，在 `frontend/admin` 执行 `pnpm lint:oxlint`，必要时执行 `pnpm type:check`。
- 涉及协议字段变化时，由后端重新生成 `frontend/admin/src/rpc`，不要手工修改生成类型。
- 涉及快捷入口种类、分组或默认提示调整时，应检查展示是否仍然能容纳最长标题。
- 页面视觉改动应至少检查空会话、有消息会话、发送中和快捷入口加载失败后的状态。
