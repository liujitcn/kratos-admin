// Package sse 提供模块无关的 SSE 流注册与 JSON 发布能力。
package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	sseServer "github.com/liujitcn/kratos-kit/transport/sse"
)

// Stream 描述一个可由认证用户订阅的 SSE 流。
type Stream interface {
	ID() string
	Resolve(channelID string, userID int64) (string, error)
}

// Registry 保存当前进程已启用模块声明的 SSE 流。
type Registry struct {
	mu      sync.RWMutex
	streams map[string]Stream
}

// NewRegistry 创建空的 SSE 流注册表。
func NewRegistry() *Registry {
	return &Registry{
		streams: make(map[string]Stream),
	}
}

// NewPublisher 创建 SSE JSON 发布器。
func NewPublisher(server *sseServer.Server) *Publisher {
	return &Publisher{server: server}
}

// Register 注册一个 SSE 流，并拒绝空标识和重复标识。
func (r *Registry) Register(stream Stream) error {
	streamID := stream.ID()
	if streamID == "" {
		return fmt.Errorf("SSE流标识不能为空")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.streams[streamID]; exists {
		return fmt.Errorf("SSE流标识重复: %s", streamID)
	}
	r.streams[streamID] = stream
	return nil
}

// Publisher 将结构化消息发布到已声明的 SSE 流。
type Publisher struct {
	server *sseServer.Server
}

// Resolve 解析订阅请求对应的传输流标识。
func (r *Registry) Resolve(streamID, channelID string, userID int64) (string, bool, error) {
	r.mu.RLock()
	stream, exists := r.streams[streamID]
	r.mu.RUnlock()
	if !exists {
		return "", false, nil
	}
	transportID, err := stream.Resolve(channelID, userID)
	if err != nil {
		return "", true, err
	}
	return transportID, true, nil
}

// PublishJSON 编码并发布一条 SSE JSON 消息。
func (p *Publisher) PublishJSON(ctx context.Context, streamID, eventID string, payload any) error {
	if streamID == "" {
		return fmt.Errorf("SSE流标识不能为空")
	}
	if eventID == "" {
		return fmt.Errorf("SSE事件标识不能为空")
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("编码SSE消息失败: %w", err)
	}
	p.server.Publish(ctx, sseServer.StreamID(streamID), &sseServer.Event{
		Event: []byte(eventID),
		Data:  data,
	})
	return nil
}

// TryPublishJSON 编码并尽力发布一条 SSE JSON 消息。
func (p *Publisher) TryPublishJSON(ctx context.Context, streamID, eventID string, payload any) {
	if streamID == "" || eventID == "" {
		return
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	p.server.TryPublish(ctx, sseServer.StreamID(streamID), &sseServer.Event{
		Event: []byte(eventID),
		Data:  data,
	})
}
