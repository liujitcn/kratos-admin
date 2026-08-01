package core

import (
	"context"
	"net/url"
	"sync"

	kratosTransport "github.com/go-kratos/kratos/v3/transport"
)

// managedServer 为核心服务提供幂等的停止边界，协调 Kratos 和宿主清理流程。
type managedServer struct {
	server   kratosTransport.Server
	stopOnce sync.Once
	stopErr  error
}

// Start 启动底层服务。
func (s *managedServer) Start(ctx context.Context) error {
	return s.server.Start(ctx)
}

// Stop 停止底层服务，并保证底层 Stop 只执行一次。
func (s *managedServer) Stop(ctx context.Context) error {
	s.stopOnce.Do(func() {
		s.stopErr = s.server.Stop(ctx)
	})
	return s.stopErr
}

// managedEndpointerServer 保留底层服务的注册中心端点能力。
type managedEndpointerServer struct {
	*managedServer
	endpointer kratosTransport.Endpointer
}

// Endpoint 返回底层服务的注册中心端点。
func (s *managedEndpointerServer) Endpoint() (*url.URL, error) {
	return s.endpointer.Endpoint()
}

// wrapManagedServer 为服务添加幂等停止能力，并保留可选的端点接口。
func wrapManagedServer(server kratosTransport.Server) kratosTransport.Server {
	managed := &managedServer{server: server}
	endpointer, ok := server.(kratosTransport.Endpointer)
	if !ok {
		return managed
	}
	return &managedEndpointerServer{
		managedServer: managed,
		endpointer:    endpointer,
	}
}
