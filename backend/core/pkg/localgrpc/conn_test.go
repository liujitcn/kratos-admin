package localgrpc

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type echoService struct{}

// echo 返回带固定前缀的测试消息。
func (echoService) echo(_ context.Context, request *wrapperspb.StringValue) (*wrapperspb.StringValue, error) {
	return wrapperspb.String("echo:" + request.GetValue()), nil
}

// TestConnInvokesRegisteredUnaryMethod 验证已注册 unary 方法可通过进程内连接调用。
func TestConnInvokesRegisteredUnaryMethod(t *testing.T) {
	conn := NewConn()
	service := echoService{}
	conn.RegisterService(&grpc.ServiceDesc{
		ServiceName: "test.EchoService",
		HandlerType: (*echoService)(nil),
		Methods: []grpc.MethodDesc{
			{
				MethodName: "Echo",
				Handler: func(service any, ctx context.Context, decode func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
					request := new(wrapperspb.StringValue)
					err := decode(request)
					if err != nil {
						return nil, err
					}
					handler := func(ctx context.Context, request any) (any, error) {
						return service.(echoService).echo(ctx, request.(*wrapperspb.StringValue))
					}
					if interceptor == nil {
						return handler(ctx, request)
					}
					return interceptor(ctx, request, &grpc.UnaryServerInfo{
						Server:     service,
						FullMethod: "/test.EchoService/Echo",
					}, handler)
				},
			},
		},
	}, service)

	reply := new(wrapperspb.StringValue)
	err := conn.Invoke(context.Background(), "/test.EchoService/Echo", wrapperspb.String("value"), reply)
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if reply.GetValue() != "echo:value" {
		t.Fatalf("Invoke() reply = %q", reply.GetValue())
	}
}
