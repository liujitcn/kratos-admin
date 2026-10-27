package dto

// WorkflowStartInput 保存首次推进所需的宿主关联确认，避免进程中断丢失恢复点。
type WorkflowStartInput struct {
	BindingRequired  bool `json:"binding_required"`
	BindingConfirmed bool `json:"binding_confirmed"`
}

// WorkflowScriptOutput 定义工作流各执行阶段的脚本输出契约。
type WorkflowScriptOutput struct {
	Allowed   *bool                  `json:"allowed,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Variables map[string]any         `json:"variables,omitempty"`
	UserIDs   []int64                `json:"user_ids,omitempty"`
	Actions   []WorkflowScriptAction `json:"actions,omitempty"`
}

// WorkflowScriptAction 定义宿主白名单内的业务动作及参数。
type WorkflowScriptAction struct {
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload"`
}

// WorkflowScriptNotification 定义内置站内信动作的字段和显式用户受众。
type WorkflowScriptNotification struct {
	Title   string  `json:"title"`
	Content string  `json:"content"`
	UserIDs []int64 `json:"user_ids"`
}
