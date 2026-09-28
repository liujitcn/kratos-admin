package main

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/go-kratos/kratos/v3/log"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/logstream"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

	"github.com/liujitcn/kratos-kit/bootstrap"

	//_ "github.com/liujitcn/kratos-kit/database/gorm/driver/bigquery"
	_ "github.com/liujitcn/kratos-kit/database/gorm/driver/mysql"
	//_ "github.com/liujitcn/kratos-kit/database/gorm/driver/oracle"
	//_ "github.com/liujitcn/kratos-kit/database/gorm/driver/postgres"
	//_ "github.com/liujitcn/kratos-kit/database/gorm/driver/sqlite"
	//_ "github.com/liujitcn/kratos-kit/database/gorm/driver/sqlserver"

	//_ "github.com/liujitcn/kratos-kit/config/apollo"
	//_ "github.com/liujitcn/kratos-kit/config/consul"
	//_ "github.com/liujitcn/kratos-kit/config/etcd"
	//_ "github.com/liujitcn/kratos-kit/config/kubernetes"
	//_ "github.com/liujitcn/kratos-kit/config/nacos"
	//_ "github.com/liujitcn/kratos-kit/config/polaris"

	//_ "github.com/liujitcn/kratos-kit/logger/aliyun"
	//_ "github.com/liujitcn/kratos-kit/logger/fluent"
	//_ "github.com/liujitcn/kratos-kit/logger/logrus"
	//_ "github.com/liujitcn/kratos-kit/logger/tencent"
	_ "github.com/liujitcn/kratos-kit/logger/zap"
	//_ "github.com/liujitcn/kratos-kit/logger/zerolog"
	_ "github.com/liujitcn/kratos-kit/registry/consul"
	//_ "github.com/liujitcn/kratos-kit/registry/etcd"
	//_ "github.com/liujitcn/kratos-kit/registry/eureka"
	//_ "github.com/liujitcn/kratos-kit/registry/kubernetes"
	//_ "github.com/liujitcn/kratos-kit/registry/nacos"
	//_ "github.com/liujitcn/kratos-kit/registry/polaris"
	//_ "github.com/liujitcn/kratos-kit/registry/servicecomb"
	//_ "github.com/liujitcn/kratos-kit/registry/zookeeper"
)

// main 启动 Admin 宿主应用。
func main() {
	defer func() {
		recovered := recover()
		stopErr := logstream.StopConsoleCapture()
		if stopErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "恢复启动日志输出失败: %v\n", stopErr)
		}
		if recovered != nil {
			// re-panic 后运行时只会打印本 defer 位置的堆栈，先输出原始 panic 堆栈便于排查。
			_, _ = fmt.Fprintf(os.Stderr, "panic: %v\n%s", recovered, debug.Stack())
			panic(recovered)
		}
	}()
	ctx := bootstrap.NewContext(
		context.Background(),
		&configv1.AppInfo{
			Project: _const.Project,
			AppId:   _const.AppID,
			Name:    _const.Name,
			Version: _const.Version,
		},
	)
	if err := bootstrap.RunApp(ctx, NewApp); err != nil {
		stopErr := logstream.StopConsoleCapture()
		if stopErr != nil {
			_, _ = fmt.Fprintf(os.Stderr, "恢复启动日志输出失败: %v\n", stopErr)
		}
		// 显式记录启动失败原因，保证端口占用等错误在控制台和日志文件中可见。
		log.Error("服务启动失败", "error", err)
		panic(err)
	}
}
