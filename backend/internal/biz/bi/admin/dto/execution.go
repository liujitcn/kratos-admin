package dto

import "github.com/liujitcn/kratos-admin/backend/pkg/bi"

const (
	// QueryKindAsk 是自然语言智能问数。
	QueryKindAsk int32 = 1
	// QueryKindPreview 是普通语义预览。
	QueryKindPreview int32 = 2
	// QueryKindCardSQLPreview 是绑定候选公式的人工预览。
	QueryKindCardSQLPreview int32 = 3
	// QueryKindCard 是保存卡片的执行。
	QueryKindCard int32 = 4
	// QueryKindDashboard 是发布快照的执行。
	QueryKindDashboard int32 = 5
)

// CardExecution 是服务端生成的唯一当前执行定义，不包含解析后的身份和相对时间。
type CardExecution struct {
	Limit          int32                `json:"limit"`
	Version        int                  `json:"version"`
	Kind           string               `json:"kind"`
	Driver         string               `json:"driver,omitempty"`
	ModelRevision  int64                `json:"model_revision"`
	AdapterVersion int32                `json:"adapter_version,omitempty"`
	SQL            string               `json:"sql,omitempty"`
	Parameters     []ExecutionParameter `json:"parameters,omitempty"`
	Metrics        []MetricRevision     `json:"metrics"`
	Columns        []bi.ResultColumn    `json:"columns"`
	QueryHash      string               `json:"query_hash"`
	Hash           string               `json:"hash"`
}

// ExecutionParameter 记录参数的受控来源及顺序，执行时才解析实际值。
type ExecutionParameter struct {
	Position    int    `json:"position"`
	Kind        string `json:"kind"`
	Field       string `json:"field,omitempty"`
	FilterIndex int    `json:"filter_index,omitempty"`
	ValueIndex  int    `json:"value_index,omitempty"`
	Contains    bool   `json:"contains,omitempty"`
}

// MetricRevision 固定公共指标的稳定编码与最近定义版本。
type MetricRevision struct {
	Code     string `json:"code"`
	Revision int64  `json:"revision"`
}

// CardPreviewContext 冻结预览的来源、版本和精确公共指标候选。
type CardPreviewContext struct {
	SourceVersion int32                     `json:"source_version"`
	CardID        int64                     `json:"card_id"`
	CardVersion   int64                     `json:"card_version"`
	QueryRecordID int64                     `json:"query_record_id"`
	CandidateHash string                    `json:"candidate_hash"`
	Changes       []ConfiguredSourceMeasure `json:"changes"`
	Before        []ConfiguredSourceMeasure `json:"before"`
}
