package dto

// AiBranchOriginBlock 保留分支创建时的来源标题和被选中回复摘要。
type AiBranchOriginBlock struct {
	// Type 标识 branch_origin 消息块。
	Type string `json:"type"`
	// Title 是来源会话创建分支时的标题。
	Title string `json:"title"`
	// Excerpt 是被选中回复的摘要。
	Excerpt string `json:"excerpt"`
}

// AiKnowledgeSource 记录注入本轮上下文的真实文档切片，引用编号按数组中的次序从 1 开始。
type AiKnowledgeSource struct {
	// Title 是文档名称。
	Title string `json:"title"`
	// Collection 是知识库名称。
	Collection string `json:"collection"`
	// Excerpt 是实际用于检索问答的片段。
	Excerpt string `json:"excerpt"`
}

// AiKnowledgeSourcesBlock 附带本轮知识引用的来源。
type AiKnowledgeSourcesBlock struct {
	// Type 标识 knowledge_sources 消息块。
	Type string `json:"type"`
	// Sources 按正文引用编号排列。
	Sources []AiKnowledgeSource `json:"sources"`
}

// AiApprovalOutcomeBlock 持久化人工决策与对应工具的实际执行状态。
type AiApprovalOutcomeBlock struct {
	// Type 标识 approval_outcome 消息块。
	Type string `json:"type"`
	// ApprovalID 区分同一轮的多次人工确认。
	ApprovalID string `json:"approval_id"`
	// State 取 approved、rejected 或 expired。
	State string `json:"state"`
	// Execution 取 succeeded、failed、not_run 或 unknown。
	Execution string `json:"execution"`
	// Retryable 表示能否安全地重新发起过期轮次。
	Retryable bool `json:"retryable"`
}

// AiReplyErrorBlock 记录服务端确认的失败类别。
type AiReplyErrorBlock struct {
	// Type 标识 reply_error 消息块。
	Type string `json:"type"`
	// Code 是固定的展示诊断码。
	Code string `json:"code"`
}

// AiGeneratedFileIssueBlock 记录未能转存的生成文件及已知的签名截止时间。
type AiGeneratedFileIssueBlock struct {
	// Type 标识 generated_file_issue 消息块。
	Type string `json:"type"`
	// Name 是产物文件的名称。
	Name string `json:"name"`
	// ExpiresAt 是工具给出的 RFC3339 有效期。
	ExpiresAt string `json:"expires_at,omitempty"`
}
