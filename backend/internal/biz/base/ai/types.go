package ai

import "context"

const (
	// TerminalApp 表示应用端终端值。
	TerminalApp int32 = 1
	// TerminalAdmin 表示管理端终端值。
	TerminalAdmin int32 = 2

	// RoleUser 表示用户消息角色。
	RoleUser = "user"
	// RoleAI 表示助手消息角色。
	RoleAI = "ai"
	// RoleSystem 表示系统消息角色，当前只在历史上下文兼容路径中使用。
	RoleSystem = "system"

	// KindText 表示普通文本消息类型。
	KindText = "text"

	// previewSize 表示会话摘要预览最大字符数。
	previewSize = 18
	// maxAttachmentTextLength 表示单个文本附件拼入模型上下文的最大字符数。
	maxAttachmentTextLength = 4000
)

// Message 表示写入 AI 助手上下文的历史消息。
//
// 该结构只保留模型构造上下文所需的最小信息，调用方从数据库消息或其他来源转换时，
// 不需要把附件、模型、降级等展示层元数据带入历史上下文。
type Message struct {
	// Role 消息角色，当前主要使用 user / ai，system 仅用于兼容历史上下文。
	Role string `json:"role"`
	// Content 消息正文，进入模型前会再次过滤空白内容。
	Content string `json:"content"`
	// Tools 当前历史轮次实际使用过的工具，仅用于短追问时延续候选工具。
	Tools []ToolUsage `json:"tools"`
}

// Attachment 表示 AI 助手运行时可消费的附件。
//
// 前端上传后传入的 proto 附件只包含文件元信息；业务层会读取 OSS 文件内容后填充
// Content 或 Bytes。运行时根据 MIME 类型决定将附件作为文本片段还是多模态图片输入。
type Attachment struct {
	// Name 附件名称，用于提示词中展示给模型。
	Name string `json:"name"`
	// Size 附件大小，模型无法直接读取文件时会作为辅助说明。
	Size int64 `json:"size"`
	// URL 附件地址，用于追踪来源；不会直接作为远端图片地址传给模型。
	URL string `json:"url"`
	// MIMEType 附件 MIME 类型，决定文本提取和图片输入路径。
	MIMEType string `json:"mimeType"`
	// Content 附件文本内容，通常来自文本、JSON、XML、CSV 等可直接读取的文件。
	Content string `json:"content"`
	// Bytes 附件原始字节，图片类附件会通过该字段作为视觉输入传给模型。
	Bytes []byte `json:"-"`
}

// Response 表示 AI 助手单轮回复结果。
//
// 该结构同时承载模型原始回复与本地降级回复。落库时会被序列化为包含正文和元信息的
// JSON，返回前端时再拆回正文、来源、模型名称和降级原因。
type Response struct {
	// Content 回复正文，面向前端展示。
	Content string `json:"content"`
	// Token 本次调用 token 消耗。
	Token TokenUsage `json:"token"`
	// Tools 本轮回复实际使用的工具列表。
	Tools []ToolUsage `json:"tools"`
	// Source 回复来源，例如 llm 或 fallback。
	Source string `json:"source"`
	// Model 使用的模型名称，便于前端展示和排障。
	Model string `json:"model"`
	// Fallback 标记本次回复是否由本地兜底逻辑生成。
	Fallback bool `json:"fallback"`
	// FallbackReason 记录触发降级的底层错误信息，仅用于排障和后台展示。
	FallbackReason string `json:"fallbackReason"`
	// Flow 记录本次回复所属的模块固定流程。
	Flow string `json:"flow"`
	// Step 记录本次回复所属的流程步骤。
	Step string `json:"step"`
	// BlocksJSON 记录模块定义的结构化内容 JSON。
	BlocksJSON string `json:"blocksJson"`
}

// TokenUsage 表示 AI 助手单轮真实 token 使用量。
type TokenUsage struct {
	// Input 输入 token 数。
	Input int32 `json:"input"`
	// Output 输出 token 数。
	Output int32 `json:"output"`
	// Cache 命中缓存 token 数。
	Cache int32 `json:"cache"`
	// Total 总 token 数。
	Total int32 `json:"total"`
}

// ToolUsage 表示 AI 助手单轮回复涉及的工具。
type ToolUsage struct {
	// Type 工具类型，例如 function 或 server。
	Type string `json:"type"`
	// Name 工具名称，用于排障和唯一识别。
	Name string `json:"name"`
	// Title 工具展示名称，优先使用生成工具描述。
	Title string `json:"title"`
	// Status 工具状态，例如 success、error。
	Status string `json:"status"`
	// Input 工具原始入参 JSON，用于后台展开排障。
	Input string `json:"input"`
	// Output 工具原始出参 JSON，用于后台展开排障。
	Output string `json:"output"`
}

// ToolAccessChecker 判断 Agent 工具是否允许在当前终端暴露。
type ToolAccessChecker interface {
	// ToolConfigs 返回当前终端允许暴露给 Agent 的工具配置。
	ToolConfigs(ctx context.Context, terminal string, names []string) (map[string]ToolConfig, error)
}

// ToolConfig 表示 Agent 工具运行时配置。
type ToolConfig struct {
	// Enabled 表示工具是否允许暴露给 Agent。
	Enabled bool
	// Prompts 表示覆盖生成工具描述和命中判断的业务提示词。
	Prompts []string
}

// RuntimeInput 表示 AI 助手运行时输入。
//
// 业务层在进入 Runtime 前完成鉴权、会话归属、附件读取、历史消息查询等工作；
// Runtime 只负责将这些输入组装成 Eino 消息和当前轮用户消息。
type RuntimeInput struct {
	// Terminal 终端标识，例如 admin 或 app，会注入到系统提示词。
	Terminal string
	// UserName 当前用户展示名称，会注入到系统提示词供模型理解上下文。
	UserName string
	// SessionTitle 当前会话标题，会注入到系统提示词供模型理解会话语境。
	SessionTitle string
	// SessionID 当前会话编号，预留给后续追踪、工具调用或日志串联。
	SessionID string
	// Summary 当前会话摘要，会注入到系统提示词作为压缩后的长期上下文。
	Summary string
	// Content 本轮用户文本内容。
	Content string
	// Attachments 本轮用户附件列表，已经由业务层读取过可用内容。
	Attachments []Attachment
	// History 会话历史消息，按时间正序传入。
	History []Message
}
