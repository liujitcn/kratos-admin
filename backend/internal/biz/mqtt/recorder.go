package mqtt

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/log"
	"github.com/liujitcn/kratos-core/queue"
	"github.com/liujitcn/kratos-kit/transport/mqtt"
	queueTransport "github.com/liujitcn/kratos-kit/transport/queue"
)

const mqttMessageStream queueTransport.Stream = "base.mqtt.message"

// NewMessageRecorder 创建将 MQTT 消息事件投递到 Admin 队列的记录器。
func NewMessageRecorder() mqtt.Recorder {
	return messageRecorder{}
}

type messageRecorder struct{}

// Record 将 MQTT 消息事件异步投递给数据库消费者。
func (messageRecorder) Record(_ context.Context, record mqtt.MessageRecord) {
	if !queue.AddQueue(mqttMessageStream, record) {
		log.Warn(fmt.Sprintf("MQTT 消息记录入队失败，topic=%s", record.Topic))
	}
}
