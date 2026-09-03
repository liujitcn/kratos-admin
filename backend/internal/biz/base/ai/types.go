package ai

import (
	"github.com/liujitcn/kratos-admin/backend/internal/biz/agent/tool"
	"github.com/liujitcn/kratos-admin/backend/pkg/agent"
)

const (
	// TerminalApp 表示应用端终端值。
	TerminalApp int32 = 1
	// TerminalAdmin 表示管理端终端值。
	TerminalAdmin int32 = 2

	previewSize             = 18
	maxAttachmentTextLength = 4000
)

const (
	// RoleUser 表示用户历史消息角色。
	RoleUser = agent.RoleUser
	// RoleAI 表示助手历史消息角色。
	RoleAI = agent.RoleAI
	// KindText 表示普通文本消息类型。
	KindText = agent.KindText
)

// Message 是 AI 历史消息。
type Message = agent.Message

// Attachment 是 AI 模型可消费的附件。
type Attachment = agent.Attachment

// Response 是 AI 单轮回复结果。
type Response = agent.Response

// TokenUsage 是 AI token 使用量。
type TokenUsage = agent.TokenUsage

// ToolUsage 是 AI 工具调用记录。
type ToolUsage = agent.ToolUsage

// ToolInvokeResult 是直接调用工具后的结果。
type ToolInvokeResult = agent.ToolInvokeResult

// ToolAccessChecker 判断 Agent 工具是否允许在当前终端暴露。
type ToolAccessChecker = agent.ToolAccessChecker

// ToolConfig 表示 Agent 工具运行时配置。
type ToolConfig = agent.ToolConfig

// RuntimeInput 表示 AI 助手运行时输入。
type RuntimeInput = agent.RuntimeInput

// AdminTools 表示管理端 AI 工具集合。
type AdminTools []tool.Invokable

// AppTools 表示应用端 AI 工具集合。
type AppTools []tool.Invokable
