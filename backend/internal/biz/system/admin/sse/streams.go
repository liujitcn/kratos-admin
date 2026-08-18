package sse

import (
	"context"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	sseTransport "github.com/liujitcn/kratos-kit/transport/sse"
)

// Streams 聚合 Admin 提供的全部业务 SSE 流。
type Streams struct {
	codegen       *Codegen
	opsMonitoring *OpsMonitoring
}

// NewStreams 创建 Admin SSE 流集合。
func NewStreams(codegen *Codegen, opsMonitoring *OpsMonitoring) ([]sseTransport.SSEStream, func()) {
	return []sseTransport.SSEStream{
			codegen, opsMonitoring,
		}, func() {
			if opsMonitoring != nil {
				opsMonitoring.Stop()
			}
		}
}

// SetPublisher 设置 Admin SSE 流的统一发布能力。
func (s *Streams) SetPublisher(publisher codegen.Publisher) {
	if s == nil {
		return
	}
	if s.codegen != nil {
		s.codegen.SetPublisher(publisher)
	}
	if s.opsMonitoring != nil {
		s.opsMonitoring.SetPublisher(publisher)
	}
}

// Register 将 Admin 的全部业务 SSE 流注册到 Core Server。
func (s *Streams) Register(server *sseTransport.Server) error {
	if s == nil {
		return nil
	}
	streams := make([]sseTransport.SSEStream, 0, 2)
	if s.codegen != nil {
		streams = append(streams, s.codegen)
	}
	if s.opsMonitoring != nil {
		streams = append(streams, s.opsMonitoring)
	}
	if len(streams) == 0 {
		return nil
	}
	err := server.RegisterStream(streams...)
	if err != nil {
		return err
	}

	// 设备监控不为空，开始监控
	if s.opsMonitoring != nil {
		s.opsMonitoring.Start(context.Background())
	}
	return nil
}
