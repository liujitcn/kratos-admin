package dto

import "time"

// OpsLogRecord 保存运维监控所需的访问日志字段。
type OpsLogRecord struct {
	Path        string
	RequestTime time.Time
	CostTime    int64
	IsSuccess   bool
	StatusCode  int32
}

// OpsEndpointAggregate 聚合单个接口的访问指标。
type OpsEndpointAggregate struct {
	Route  string
	Costs  []int64
	Total  int
	Errors int
}

// OpsMinuteAggregate 聚合一分钟内的访问指标。
type OpsMinuteAggregate struct {
	At    time.Time
	Costs []int64
	Total int
}
