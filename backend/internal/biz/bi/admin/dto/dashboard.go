package dto

import (
	"encoding/json"

	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
)

// DashboardSchema 保存卡片引用与布局，发布时由服务端补齐卡片配置快照。
type DashboardSchema struct {
	SchemaVersion int             `json:"schemaVersion"`
	Name          string          `json:"name"`
	Theme         json.RawMessage `json:"theme,omitempty"`
	Cards         []DashboardCard `json:"cards"`
}

// DashboardCard 表示一张卡片在画布中的位置及其发布快照。
type DashboardCard struct {
	ID            string        `json:"id"`
	CardID        int64         `json:"card_id"`
	QueryRecordID int64         `json:"query_record_id,omitempty"` // 原始智能问数记录
	X             int           `json:"x"`
	Y             int           `json:"y"`
	W             int           `json:"w"`
	H             int           `json:"h"`
	Name          string        `json:"name,omitempty"`
	ChartType     string        `json:"chart_type,omitempty"`
	ChartJSON     string        `json:"chart_json,omitempty"`
	Query         *bi.QuerySpec `json:"query,omitempty"`
}
