package dto

// MessageDispatchTask 表示 Redis Streams 中的消息投递任务定位信息。
// 消费者只依据编号重新读取数据库任务，并通过租户编号校验消息归属。
type MessageDispatchTask struct {
	DispatchID      int64 `json:"dispatch_id"`      // 消息投递任务编号。
	TenantID        int64 `json:"tenant_id"`        // 消息所属租户编号。
	ExpectedVersion int64 `json:"expected_version"` // 入队时的投递任务版本。
}
