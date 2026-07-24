package queue

import (
	"bytes"
	"encoding/json"
	"fmt"

	_const "github.com/liujitcn/kratos-admin/backend/pkg/const"

	"github.com/go-kratos/kratos/v3/log"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
)

// AddQueue 向运行时队列追加异步消息。
func AddQueue(queueName _const.Queue, data any) bool {
	queueID := string(queueName)
	q := sdk.Runtime.GetQueue()
	// 运行时队列未初始化时，直接跳过异步投递。
	if q == nil {
		return false
	}

	rawBody, err := json.Marshal(data)
	// 队列消息体无法序列化时，只记录日志，不影响主流程。
	if err != nil {
		log.Error(fmt.Sprintf("build queue message data error, %s", err.Error()))
		return false
	}
	messageData := map[string]any{
		"data": string(rawBody),
	}

	var message queueData.Message
	message, err = sdk.Runtime.GetStreamMessage("", messageData)
	// 底层消息对象构造失败时，只记录日志，不影响主流程。
	if err != nil {
		log.Error(fmt.Sprintf("GetStreamMessage error, %s", err.Error()))
		return false
	}

	err = q.Append(queueID, message)
	// 队列追加失败时，只记录日志，不影响主流程。
	if err != nil {
		log.Error(fmt.Sprintf("Append message error, %s", err.Error()))
		return false
	}
	return true
}

// DecodeQueueData 解析队列消息中的 data 字段，并兼容内存队列与 Redis 队列两种载荷形态。
func DecodeQueueData[T any](message queueData.Message) (*T, error) {
	rawData, ok := message.Values["data"]
	// 队列消息里没有 data 字段时，说明当前消息无需继续处理。
	if !ok || rawData == nil {
		return nil, nil
	}

	switch value := rawData.(type) {
	// Redis 队列会把复杂对象保存成 JSON 字符串，这里直接按目标类型解析。
	case string:
		return decodeQueueDataBytes[T]([]byte(value))
	// 少数字节载荷场景复用同一套 JSON 解析逻辑。
	case []byte:
		return decodeQueueDataBytes[T](value)
	default:
		// 内存队列仍可能直接传递结构体对象，这里先转成 JSON 再统一解析。
		rawBody, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return decodeQueueDataBytes[T](rawBody)
	}
}

// decodeQueueDataBytes 将 JSON 字节载荷还原成业务消息对象。
func decodeQueueDataBytes[T any](rawBody []byte) (*T, error) {
	trimmedBody := bytes.TrimSpace(rawBody)
	// 空载荷或 null 载荷都视为当前消息没有有效业务数据。
	if len(trimmedBody) == 0 || bytes.Equal(trimmedBody, []byte("null")) {
		return nil, nil
	}

	var data T
	err := json.Unmarshal(trimmedBody, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}
