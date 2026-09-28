package _const

const (
	// MessageDeliveryTypeInbox 表示站内信收件记录。
	MessageDeliveryTypeInbox int32 = 1
	// MessageDeliveryTypeProvider 表示外部 Provider 投递记录。
	MessageDeliveryTypeProvider int32 = 2
	// MessageDeliveryStatusPending 表示等待 Provider 投递。
	MessageDeliveryStatusPending int32 = 1
	// MessageDeliveryStatusRunning 表示 Provider 正在投递。
	MessageDeliveryStatusRunning int32 = 2
	// MessageDeliveryStatusSucceeded 表示投递成功。
	MessageDeliveryStatusSucceeded int32 = 3
	// MessageDeliveryStatusFailed 表示投递失败。
	MessageDeliveryStatusFailed int32 = 4
	// MessageDeliveryStatusSkipped 表示投递被跳过。
	MessageDeliveryStatusSkipped int32 = 5
)
