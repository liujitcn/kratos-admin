package queue

import (
	"context"
	"testing"

	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

type queueAdapter struct {
	messages map[string]queueData.Message
}

// Append 保存测试消息。
func (q *queueAdapter) Append(topic string, message queueData.Message) error {
	q.messages[topic] = message
	return nil
}

// Register 保持测试消费者注册为空。
func (*queueAdapter) Register(string, queueData.ConsumerFunc) {}

// Run 保持测试消费循环为空。
func (*queueAdapter) Run() {}

// Shutdown 保持测试队列清理为空。
func (*queueAdapter) Shutdown() {}

// TestPublisherAndDecode 验证结构化消息发布后可以按目标类型解码。
func TestPublisherAndDecode(t *testing.T) {
	adapter := &queueAdapter{messages: make(map[string]queueData.Message)}
	publisher, err := NewPublisher(adapter)
	if err != nil {
		t.Fatalf("创建队列发布器失败: %v", err)
	}
	type payload struct {
		Name string `json:"name"`
	}
	err = publisher.Publish(context.Background(), "events", payload{Name: "created"})
	if err != nil {
		t.Fatalf("发布队列消息失败: %v", err)
	}
	var decoded *payload
	decoded, err = Decode[payload](adapter.messages["events"])
	if err != nil {
		t.Fatalf("解码队列消息失败: %v", err)
	}
	if decoded == nil || decoded.Name != "created" {
		t.Fatalf("队列消息内容错误: %+v", decoded)
	}
}
