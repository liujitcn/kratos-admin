package localgrpc

import (
	"context"
	"fmt"
	"io"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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

type streamMethod struct {
	description *grpc.StreamDesc
	service     any
}

// Conn 将注册的 gRPC 服务作为进程内生成客户端连接使用。
//
// 注册应在首次调用前完成。
type Conn struct {
	mu          sync.RWMutex
	methods     map[string]unaryMethod
	streams     map[string]streamMethod
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
		streams: make(map[string]streamMethod),
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
		if _, exists := c.streams[fullMethod]; exists {
			panic(fmt.Sprintf("localgrpc: gRPC 方法 %s 重复注册", fullMethod))
		}
		c.methods[fullMethod] = unaryMethod{
			description: method,
			service:     service,
		}
	}
	for index := range description.Streams {
		stream := &description.Streams[index]
		fullMethod := "/" + description.ServiceName + "/" + stream.StreamName
		if _, exists := c.methods[fullMethod]; exists {
			panic(fmt.Sprintf("localgrpc: gRPC 方法 %s 重复注册", fullMethod))
		}
		if _, exists := c.streams[fullMethod]; exists {
			panic(fmt.Sprintf("localgrpc: gRPC 方法 %s 重复注册", fullMethod))
		}
		c.streams[fullMethod] = streamMethod{
			description: stream,
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

// NewStream 创建可直接调用本地服务处理器的进程内 streaming RPC。
func (c *Conn) NewStream(ctx context.Context, _ *grpc.StreamDesc, method string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
	c.mu.RLock()
	registered, exists := c.streams[method]
	c.mu.RUnlock()
	if !exists {
		return nil, status.Errorf(codes.Unimplemented, "进程内 gRPC 流式方法未注册: %s", method)
	}
	stream := newLocalStream(ctx, registered.description)
	go func() {
		err := registered.description.Handler(registered.service, stream)
		stream.finish(err)
	}()
	return &localClientStream{localStream: stream}, nil
}

type localStream struct {
	ctx             context.Context
	description     *grpc.StreamDesc
	clientMessages  chan proto.Message
	serverMessages  chan proto.Message
	clientSendDone  chan struct{}
	done            chan struct{}
	headerReady     chan struct{}
	mu              sync.Mutex
	headerOnce      sync.Once
	finishedErr     error
	finished        bool
	clientSendClose bool
	header          metadata.MD
	trailer         metadata.MD
	headerSent      bool
}

type localClientStream struct {
	*localStream
}

// newLocalStream 创建本地流的双向消息通道。
func newLocalStream(ctx context.Context, description *grpc.StreamDesc) *localStream {
	return &localStream{
		ctx:            ctx,
		description:    description,
		clientMessages: make(chan proto.Message, 1),
		serverMessages: make(chan proto.Message, 1),
		clientSendDone: make(chan struct{}),
		done:           make(chan struct{}),
		headerReady:    make(chan struct{}),
	}
}

// finish 标记服务端流处理完成并保存处理结果。
func (s *localStream) finish(err error) {
	s.mu.Lock()
	s.finishedErr = err
	s.finished = true
	s.mu.Unlock()
	s.markHeaderReady()
	close(s.done)
}

// markHeaderReady 标记流响应头已经发送或流已经结束。
func (s *localStream) markHeaderReady() {
	s.headerOnce.Do(func() {
		close(s.headerReady)
	})
}

// result 返回服务端流处理结果。
func (s *localStream) result() error {
	s.mu.Lock()
	err := s.finishedErr
	s.mu.Unlock()
	return err
}

// SetHeader 累积服务端流响应头。
func (s *localStream) SetHeader(md metadata.MD) error {
	s.mu.Lock()
	if s.finished || s.headerSent {
		s.mu.Unlock()
		return status.Error(codes.Internal, "进程内 gRPC 流式响应头已发送")
	}
	s.header = metadata.Join(s.header, md)
	s.mu.Unlock()
	return nil
}

// SendHeader 发送服务端流响应头。
func (s *localStream) SendHeader(md metadata.MD) error {
	s.mu.Lock()
	if s.finished || s.headerSent {
		s.mu.Unlock()
		return status.Error(codes.Internal, "进程内 gRPC 流式响应头重复发送")
	}
	s.header = metadata.Join(s.header, md)
	s.headerSent = true
	s.mu.Unlock()
	s.markHeaderReady()
	return nil
}

// SetTrailer 累积服务端流尾部元数据。
func (s *localStream) SetTrailer(md metadata.MD) {
	s.mu.Lock()
	if s.finished {
		s.mu.Unlock()
		return
	}
	s.trailer = metadata.Join(s.trailer, md)
	s.mu.Unlock()
}

// Context 返回本地流上下文。
func (s *localStream) Context() context.Context {
	return s.ctx
}

// SendMsg 发送服务端流消息。
func (s *localStream) SendMsg(message any) error {
	protobufMessage, ok := message.(proto.Message)
	if !ok {
		return fmt.Errorf("进程内 gRPC 仅支持 protobuf 消息，来源 %T", message)
	}
	s.mu.Lock()
	if s.finished {
		err := s.finishedErr
		s.mu.Unlock()
		if err != nil {
			return err
		}
		return io.EOF
	}
	s.headerSent = true
	s.mu.Unlock()
	s.markHeaderReady()
	clonedMessage := proto.Clone(protobufMessage)
	select {
	case s.serverMessages <- clonedMessage:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	case <-s.done:
		if err := s.result(); err != nil {
			return err
		}
		return io.EOF
	}
}

// RecvMsg 接收客户端发送给服务端的流消息。
func (s *localStream) RecvMsg(message any) error {
	receivedMessage, err := s.receiveClientMessage()
	if err != nil {
		return err
	}
	return copyMessage(message, receivedMessage)
}

// SendMsg 发送客户端流消息。
func (s *localClientStream) SendMsg(message any) error {
	protobufMessage, ok := message.(proto.Message)
	if !ok {
		return fmt.Errorf("进程内 gRPC 仅支持 protobuf 消息，来源 %T", message)
	}
	s.mu.Lock()
	clientSendClose := s.clientSendClose
	s.mu.Unlock()
	if clientSendClose {
		return io.EOF
	}
	clonedMessage := proto.Clone(protobufMessage)
	select {
	case s.clientMessages <- clonedMessage:
		return nil
	case <-s.Context().Done():
		return s.Context().Err()
	case <-s.done:
		if err := s.result(); err != nil {
			return err
		}
		return io.EOF
	}
}

// RecvMsg 接收服务端发送给客户端的流消息。
func (s *localClientStream) RecvMsg(message any) error {
	receivedMessage, err := s.receiveServerMessage()
	if err != nil {
		return err
	}
	return copyMessage(message, receivedMessage)
}

// CloseSend 关闭客户端发送方向。
func (s *localClientStream) CloseSend() error {
	s.mu.Lock()
	if s.clientSendClose {
		s.mu.Unlock()
		return nil
	}
	s.clientSendClose = true
	close(s.clientSendDone)
	s.mu.Unlock()
	return nil
}

// Header 返回服务端流响应头。
func (s *localClientStream) Header() (metadata.MD, error) {
	select {
	case <-s.headerReady:
	case <-s.done:
	case <-s.Context().Done():
		return nil, s.Context().Err()
	}
	s.mu.Lock()
	header := metadata.Join(s.header)
	s.mu.Unlock()
	return header, nil
}

// Trailer 返回服务端流尾部元数据。
func (s *localClientStream) Trailer() metadata.MD {
	s.mu.Lock()
	trailer := metadata.Join(s.trailer)
	s.mu.Unlock()
	return trailer
}

// receiveClientMessage 接收客户端发送给服务端的下一条消息，并在发送端关闭后返回 EOF。
func (s *localStream) receiveClientMessage() (proto.Message, error) {
	select {
	case receivedMessage := <-s.clientMessages:
		return receivedMessage, nil
	default:
	}
	select {
	case receivedMessage := <-s.clientMessages:
		return receivedMessage, nil
	case <-s.clientSendDone:
		select {
		case receivedMessage := <-s.clientMessages:
			return receivedMessage, nil
		default:
			return nil, io.EOF
		}
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	case <-s.done:
		if err := s.result(); err != nil {
			return nil, err
		}
		select {
		case receivedMessage := <-s.clientMessages:
			return receivedMessage, nil
		default:
			return nil, io.EOF
		}
	}
}

// receiveServerMessage 接收服务端发送给客户端的下一条消息，并在服务端完成后返回 EOF。
func (s *localStream) receiveServerMessage() (proto.Message, error) {
	select {
	case receivedMessage := <-s.serverMessages:
		return receivedMessage, nil
	default:
	}
	select {
	case receivedMessage := <-s.serverMessages:
		return receivedMessage, nil
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	case <-s.done:
		if err := s.result(); err != nil {
			return nil, err
		}
		select {
		case receivedMessage := <-s.serverMessages:
			return receivedMessage, nil
		default:
			return nil, io.EOF
		}
	}
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
