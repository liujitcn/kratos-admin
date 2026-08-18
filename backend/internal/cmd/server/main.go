package main

import (
	"context"

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
	ctx := bootstrap.NewContext(
		context.Background(),
		&bootstrapConfigv1.AppInfo{
			Project: _const.Project,
			AppId:   _const.AppID,
			Name:    _const.Name,
			Version: _const.Version,
		},
	)
	if err := bootstrap.RunApp(ctx, NewAdminApp); err != nil {
		panic(err)
	}
}
