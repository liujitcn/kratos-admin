package dto

// WorkbenchConfig 保存个人工作台的仪表盘入口顺序和数据卡片收藏。
type WorkbenchConfig struct {
	DashboardIDs []int64 `json:"dashboard_ids"`
	FavoriteIDs  []int64 `json:"favorite_ids"`
}

// CardChartConfig 保存统一图表编辑器的字段映射和样式，不接收任意引擎选项。
type CardChartConfig struct {
	Encoding CardChartEncoding `json:"encoding"`
	Style    CardChartStyle    `json:"style"`
}

// CardChartEncoding 只引用来源查询结果已经输出的字段。
type CardChartEncoding struct {
	Dimension string   `json:"dimension,omitempty"`
	Measure   string   `json:"measure,omitempty"`
	Measures  []string `json:"measures,omitempty"`
	Series    string   `json:"series,omitempty"`
}

// CardChartStyle 配置图表的受控展示选项。
type CardChartStyle struct {
	ShowLegend bool   `json:"showLegend"`
	ShowLabel  bool   `json:"showLabel"`
	Stack      bool   `json:"stack"`
	DataZoom   bool   `json:"dataZoom"`
	Palette    string `json:"palette,omitempty"`
}
