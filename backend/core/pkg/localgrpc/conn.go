// Package localgrpc 提供与生成 gRPC 客户端兼容的进程内调用连接。
package localgrpc

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

var (
	_ grpc.ServiceRegistrar    = (*Conn)(nil)
	_ grpc.ClientConnInterface = (*Conn)(nil)
)

type unaryMethod struct {
	description *grpc.MethodDesc
	service     any
}

// Conn 将注册的 unary gRPC 服务作为进程内生成客户端连接使用。
//
// 注册应在首次调用前完成。当前不支持 streaming RPC。
type Conn struct {
	mu          sync.RWMutex
	methods     map[string]unaryMethod
	interceptor grpc.UnaryServerInterceptor
}

// Option 配置进程内 gRPC 连接。
type Option func(*Conn)

// WithUnaryInterceptor 设置进程内调用使用的 unary 拦截器。
func WithUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) Option {
	return func(conn *Conn) {
		conn.interceptor = interceptor
	}
}

// NewConn 创建进程内 gRPC 连接。
func NewConn(options ...Option) *Conn {
	conn := &Conn{
		methods: make(map[string]unaryMethod),
	}
	for _, option := range options {
		option(conn)
	}
	return conn
}

// RegisterService 注册生成代码声明的 gRPC 服务。
func (c *Conn) RegisterService(description *grpc.ServiceDesc, service any) {
	if description == nil {
		panic("localgrpc: gRPC 服务描述不能为空")
	}
	if service == nil {
		panic(fmt.Sprintf("localgrpc: gRPC 服务 %s 的实现不能为空", description.ServiceName))
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for index := range description.Methods {
		method := &description.Methods[index]
		fullMethod := "/" + description.ServiceName + "/" + method.MethodName
		if _, exists := c.methods[fullMethod]; exists {
			panic(fmt.Sprintf("localgrpc: gRPC 方法 %s 重复注册", fullMethod))
		}
		c.methods[fullMethod] = unaryMethod{
			description: method,
			service:     service,
		}
	}
}

// Invoke 调用已注册的进程内 unary gRPC 方法。
func (c *Conn) Invoke(ctx context.Context, method string, args, reply any, _ ...grpc.CallOption) error {
	c.mu.RLock()
	registered, exists := c.methods[method]
	interceptor := c.interceptor
	c.mu.RUnlock()
	if !exists {
		return status.Errorf(codes.Unimplemented, "进程内 gRPC 方法未注册: %s", method)
	}

	response, err := registered.description.Handler(
		registered.service,
		ctx,
		func(request any) error {
			return copyMessage(request, args)
		},
		interceptor,
	)
	if err != nil {
		return err
	}
	return copyMessage(reply, response)
}

// NewStream 返回当前不支持 streaming RPC 的明确错误。
func (c *Conn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, status.Error(codes.Unimplemented, "进程内 gRPC 连接暂不支持 streaming RPC")
}

// copyMessage 复制同类型 protobuf 消息并隔离调用双方的可变状态。
func copyMessage(target, source any) error {
	targetMessage, targetOK := target.(proto.Message)
	sourceMessage, sourceOK := source.(proto.Message)
	if !targetOK || !sourceOK {
		return fmt.Errorf("进程内 gRPC 仅支持 protobuf 消息，来源 %T，目标 %T", source, target)
	}
	if targetMessage.ProtoReflect().Descriptor().FullName() != sourceMessage.ProtoReflect().Descriptor().FullName() {
		return fmt.Errorf("进程内 gRPC 消息类型不一致，来源 %s，目标 %s", sourceMessage.ProtoReflect().Descriptor().FullName(), targetMessage.ProtoReflect().Descriptor().FullName())
	}
	proto.Reset(targetMessage)
	proto.Merge(targetMessage, sourceMessage)
	return nil
}
