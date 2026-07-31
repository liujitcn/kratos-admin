package main

import (
	"context"

	"github.com/go-kratos/kratos/v3"
	kratosadmin "github.com/liujitcn/kratos-admin/backend"
	systemConfig "github.com/liujitcn/kratos-admin/backend/internal/config"
	_const "github.com/liujitcn/kratos-admin/backend/internal/const"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"

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
	//_ "github.com/liujitcn/kratos-kit/registry/consul"
	//_ "github.com/liujitcn/kratos-kit/registry/etcd"
	//_ "github.com/liujitcn/kratos-kit/registry/eureka"
	//_ "github.com/liujitcn/kratos-kit/registry/kubernetes"
	//_ "github.com/liujitcn/kratos-kit/registry/nacos"
	//_ "github.com/liujitcn/kratos-kit/registry/polaris"
	//_ "github.com/liujitcn/kratos-kit/registry/servicecomb"
	//_ "github.com/liujitcn/kratos-kit/registry/zookeeper"
)

var (
	// Project 表示当前服务所属项目标识。
	Project = _const.PROJECT_KEY
	// AppID 表示当前服务应用标识。
	AppID = "admin"
	// Name 表示当前服务展示名称。
	Name    = _const.PROJECT_NAME
	version = "1.0.0"
)

// main 作为服务启动入口，负责执行应用启动并在失败时中止进程。
func main() {
	ctx := bootstrap.NewContext(
		context.Background(),
		&bootstrapConfigv1.AppInfo{
			Project: Project,
			AppId:   AppID,
			Name:    Name,
			Version: version,
		},
	)
	appInfo := systemConfig.GetAppInfo(ctx)
	_const.BASE_PATH = appInfo.GetProject()

	// 应用启动失败时直接中止进程，避免服务以异常状态继续运行。
	if err := bootstrap.RunApp(ctx, initApp); err != nil {
		panic(err)
	}
}

// initApp 使用 Backend 根包的内部装配结果创建独立应用。
func initApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	return kratosadmin.NewApp(ctx)
}
