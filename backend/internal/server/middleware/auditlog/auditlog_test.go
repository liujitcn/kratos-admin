package auditlog

import (
	"encoding/json"
	"testing"

	queueData "github.com/liujitcn/kratos-kit/queue/data"
	"github.com/liujitcn/kratos-kit/sdk"
)

type testQueue struct {
	stream  string
	message queueData.Message
}

// Append 记录测试期间投递的队列消息。
func (q *testQueue) Append(stream string, message queueData.Message) error {
	q.stream = stream
	q.message = message
	return nil
}

// Register 忽略测试不需要的消费者注册。
func (*testQueue) Register(string, queueData.ConsumerFunc) {}

// Run 忽略测试不需要的队列启动。
func (*testQueue) Run() {}

// Shutdown 忽略测试不需要的队列停止。
func (*testQueue) Shutdown() {}

// TestMiddlewareEmitsAdminEventToQueue 验证 Admin 业务审计事件只投递到异步入库队列。
func TestMiddlewareEmitsAdminEventToQueue(t *testing.T) {
	queue := &testQueue{}
	sdk.Runtime.SetQueue(queue)
	t.Cleanup(func() { sdk.Runtime.SetQueue(nil) })
	(&Middleware{}).emit("login", map[string]string{"request_id": "test"}, "/test")
	if queue.stream != string(adminEventStream) {
		t.Fatalf("unexpected stream: %s", queue.stream)
	}
	rawBody, ok := queue.message.Values["data"].(string)
	if !ok {
		t.Fatal("expected string queue payload")
	}
	var event adminEvent
	var err error
	err = json.Unmarshal([]byte(rawBody), &event)
	if err != nil {
		t.Fatal(err)
	}
	if event.Kind != "login" {
		t.Fatalf("unexpected event kind: %s", event.Kind)
	}
	if event.Payload == "" {
		t.Fatal("expected event payload")
	}
}
