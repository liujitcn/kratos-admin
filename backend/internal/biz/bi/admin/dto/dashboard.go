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
	Columns                 []bi.ResultColumn `json:"columns,omitempty"`
	Execution               *CardExecution    `json:"execution,omitempty"`
	CardVersion             int64             `json:"card_version,omitempty"`
	ValidationQueryRecordID int64             `json:"validation_query_record_id,omitempty"`
	ID                      string            `json:"id"`
	CardID                  int64             `json:"card_id"`
	QueryRecordID           int64             `json:"query_record_id,omitempty"` // 原始智能问数记录
	X                       int               `json:"x"`
	Y                       int               `json:"y"`
	W                       int               `json:"w"`
	H                       int               `json:"h"`
	Name                    string            `json:"name,omitempty"`
	ChartType               string            `json:"chart_type,omitempty"`
	ChartJSON               string            `json:"chart_json,omitempty"`
	Query                   *bi.QuerySpec     `json:"query,omitempty"`
}

// DashboardDisplay 是对所有查看接口开放的布局白名单，与存储快照分离。
type DashboardDisplay struct {
	SchemaVersion int                    `json:"schemaVersion"`
	Name          string                 `json:"name"`
	Cards         []DashboardDisplayCard `json:"cards"`
}

// DashboardDisplayCard 只包含布局、受控样式和结果字段的展示信息。
type DashboardDisplayCard struct {
	ID          string            `json:"id"`
	CardID      int64             `json:"card_id"`
	X           int               `json:"x"`
	Y           int               `json:"y"`
	W           int               `json:"w"`
	H           int               `json:"h"`
	Name        string            `json:"name,omitempty"`
	ChartType   string            `json:"chart_type,omitempty"`
	ChartJSON   string            `json:"chart_json,omitempty"`
	CardVersion int64             `json:"card_version,omitempty"`
	Columns     []bi.ResultColumn `json:"columns,omitempty"`
}

// Display 显式构造展示布局，新增内部字段也不会意外进入查看响应。
func (s DashboardSchema) Display() DashboardDisplay {
	result := DashboardDisplay{SchemaVersion: 4, Name: s.Name, Cards: make([]DashboardDisplayCard, 0, len(s.Cards))}
	for _, card := range s.Cards {
		item := DashboardDisplayCard{ID: card.ID, CardID: card.CardID, X: card.X, Y: card.Y, W: card.W, H: card.H, Name: card.Name, ChartType: card.ChartType, ChartJSON: card.ChartJSON, CardVersion: card.CardVersion}
		if card.Execution != nil {
			item.Columns = card.Execution.Columns
		}
		result.Cards = append(result.Cards, item)
	}
	return result
}
