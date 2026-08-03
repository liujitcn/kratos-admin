package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
)

// Stream 描述一个可由外部身份适配器解析的 SSE 流。
type Stream interface {
	ID() string
	Resolve(channelID string, subjectID int64) (string, error)
}

// Registry 保存当前进程已启用模块声明的 SSE 流。
type Registry struct {
	mu      sync.RWMutex
	streams map[string]Stream
}

// NewRegistry 创建空的 SSE 流注册表。
func NewRegistry() *Registry {
	return &Registry{streams: make(map[string]Stream)}
}

// Register 注册 SSE 流，并拒绝空对象、空标识和重复标识。
func (r *Registry) Register(streams ...Stream) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	registered := make(map[string]struct{}, len(streams))
	for _, stream := range streams {
		if stream == nil {
			return fmt.Errorf("SSE流不能为空")
		}
		streamID := stream.ID()
		if streamID == "" {
			return fmt.Errorf("SSE流标识不能为空")
		}
		if _, exists := r.streams[streamID]; exists {
			return fmt.Errorf("SSE流标识重复: %s", streamID)
		}
		if _, exists := registered[streamID]; exists {
			return fmt.Errorf("SSE流标识重复: %s", streamID)
		}
		registered[streamID] = struct{}{}
	}
	for _, stream := range streams {
		r.streams[stream.ID()] = stream
	}
	return nil
}

// Resolve 解析订阅请求对应的传输流标识。
func (r *Registry) Resolve(streamID, channelID string, subjectID int64) (string, bool, error) {
	r.mu.RLock()
	stream, exists := r.streams[streamID]
	r.mu.RUnlock()
	if !exists {
		return "", false, nil
	}
	var err error
	var transportID string
	transportID, err = stream.Resolve(channelID, subjectID)
	if err != nil {
		return "", true, err
	}
	return transportID, true, nil
}

// Publisher 将结构化消息发布到已声明的 SSE 流。
type Publisher struct {
	server *sseServer.Server
}

// NewPublisher 创建 SSE JSON 发布器。
func NewPublisher(server *sseServer.Server) *Publisher {
	return &Publisher{server: server}
}

// PublishJSON 编码并发布一条 SSE JSON 消息。
func (p *Publisher) PublishJSON(ctx context.Context, streamID, eventID string, payload any) error {
	if p == nil || p.server == nil {
		return fmt.Errorf("SSE服务未初始化")
	}
	if streamID == "" {
		return fmt.Errorf("SSE流标识不能为空")
	}
	if eventID == "" {
		return fmt.Errorf("SSE事件标识不能为空")
	}
	var data []byte
	var err error
	data, err = json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("编码SSE消息失败: %w", err)
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	transportID := sseServer.StreamID(streamID)
	if p.server.GetStream(transportID) == nil {
		return fmt.Errorf("SSE流不存在: %s", streamID)
	}
	p.server.Publish(ctx, transportID, &sseServer.Event{
		Event: []byte(eventID),
		Data:  data,
	})
	return nil
}

// TryPublishJSON 编码并尽力发布一条 SSE JSON 消息。
func (p *Publisher) TryPublishJSON(ctx context.Context, streamID, eventID string, payload any) {
	if p == nil || p.server == nil || streamID == "" || eventID == "" {
		return
	}
	var data []byte
	var err error
	data, err = json.Marshal(payload)
	if err != nil {
		return
	}
	p.server.TryPublish(ctx, sseServer.StreamID(streamID), &sseServer.Event{
		Event: []byte(eventID),
		Data:  data,
	})
}
