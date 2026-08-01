package module

import "google.golang.org/grpc"

// RuntimeReadyContributor 表示需要接收 Admin 进程内 gRPC 连接的扩展模块。
type RuntimeReadyContributor interface {
	// RuntimeReady 在 Admin 运行时完成装配后绑定进程内 gRPC 连接。
	RuntimeReady(grpc.ClientConnInterface) error
}
