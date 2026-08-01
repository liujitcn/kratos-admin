// Package queue 提供队列生命周期、结构化消息发布和解码能力。
package queue

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	kitQueue "github.com/liujitcn/kratos-kit/queue"
	queueData "github.com/liujitcn/kratos-kit/queue/data"
)

// Consumer 描述模块贡献的具名队列消费者。
type Consumer struct {
	// Topic 是消费者监听的队列主题。
	Topic string
	// Handler 处理队列消息。
	Handler queueData.ConsumerFunc
}

// Runtime 将队列消费者接入 Kratos 服务生命周期。
type Runtime struct {
	queue     kitQueue.Queue
	mu        sync.Mutex
	topics    map[string]struct{}
	consumers []Consumer
	done      chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	started   atomic.Bool
}

// NewRuntime 创建队列运行时。
func NewRuntime(queue kitQueue.Queue) (*Runtime, error) {
	if queue == nil {
		return nil, fmt.Errorf("队列适配器不能为空")
	}
	return &Runtime{
		queue:  queue,
		topics: make(map[string]struct{}),
		done:   make(chan struct{}),
	}, nil
}

// Register 记录队列消费者定义，并拒绝空主题、空处理器和重复主题。
func (r *Runtime) Register(consumers ...Consumer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started.Load() {
		return fmt.Errorf("队列运行时已启动，不能追加消费者")
	}

	registered := make(map[string]struct{}, len(consumers))
	for _, consumer := range consumers {
		if consumer.Topic == "" {
			return fmt.Errorf("队列主题不能为空")
		}
		if consumer.Handler == nil {
			return fmt.Errorf("队列消费者不能为空: %s", consumer.Topic)
		}
		if _, exists := r.topics[consumer.Topic]; exists {
			return fmt.Errorf("队列主题重复: %s", consumer.Topic)
		}
		if _, exists := registered[consumer.Topic]; exists {
			return fmt.Errorf("队列主题重复: %s", consumer.Topic)
		}
		registered[consumer.Topic] = struct{}{}
	}
	for _, consumer := range consumers {
		r.topics[consumer.Topic] = struct{}{}
	}
	r.consumers = append(r.consumers, consumers...)
	return nil
}

// Start 注册队列消费者并启动消费循环。
func (r *Runtime) Start(context.Context) error {
	r.startOnce.Do(func() {
		r.mu.Lock()
		for _, consumer := range r.consumers {
			r.queue.Register(consumer.Topic, consumer.Handler)
		}
		r.started.Store(true)
		r.mu.Unlock()
		go func() {
			r.queue.Run()
			close(r.done)
		}()
	})
	return nil
}

// Stop 停止队列并等待消费循环退出。
func (r *Runtime) Stop(ctx context.Context) error {
	if !r.started.Load() {
		return nil
	}
	r.stopOnce.Do(func() {
		r.queue.Shutdown()
	})
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Publisher 将结构化数据发布到队列适配器。
type Publisher struct {
	queue kitQueue.Queue
}

// NewPublisher 创建队列发布器。
func NewPublisher(queue kitQueue.Queue) (*Publisher, error) {
	if queue == nil {
		return nil, fmt.Errorf("队列适配器不能为空")
	}
	return &Publisher{queue: queue}, nil
}

// Publish 将数据编码为 JSON 并发布到指定主题。
func (p *Publisher) Publish(ctx context.Context, topic string, data any) error {
	var err error
	err = ctx.Err()
	if err != nil {
		return err
	}
	if topic == "" {
		return fmt.Errorf("队列主题不能为空")
	}
	var rawBody []byte
	rawBody, err = json.Marshal(data)
	if err != nil {
		return fmt.Errorf("编码队列消息失败: %w", err)
	}
	err = p.queue.Append(topic, queueData.Message{
		Values: map[string]any{"data": string(rawBody)},
	})
	if err != nil {
		return fmt.Errorf("发布队列消息到 %q: %w", topic, err)
	}
	return nil
}

// Decode 解析队列消息中的 data 字段，并兼容字符串、字节和内存对象载荷。
func Decode[T any](message queueData.Message) (*T, error) {
	rawData, exists := message.Values["data"]
	if !exists || rawData == nil {
		return nil, nil
	}

	switch value := rawData.(type) {
	case string:
		return decodeBytes[T]([]byte(value))
	case []byte:
		return decodeBytes[T](value)
	default:
		var err error
		var rawBody []byte
		rawBody, err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return decodeBytes[T](rawBody)
	}
}

func decodeBytes[T any](rawBody []byte) (*T, error) {
	trimmedBody := bytes.TrimSpace(rawBody)
	if len(trimmedBody) == 0 || bytes.Equal(trimmedBody, []byte("null")) {
		return nil, nil
	}

	var data T
	var err error
	err = json.Unmarshal(trimmedBody, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}
