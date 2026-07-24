package main

import (
	host "github.com/liujitcn/kratos-admin/backend/server"
	baseserver "github.com/liujitcn/kratos-admin/backend/server/base"
	systemadminserver "github.com/liujitcn/kratos-admin/backend/server/system/admin"
	systemappserver "github.com/liujitcn/kratos-admin/backend/server/system/app"
)

// newModules 汇总当前进程启用的基础与系统模块。
func newModules(
	baseModule baseserver.Services,
	systemAdminModule systemadminserver.Services,
	systemAppModule systemappserver.Services,
) (host.Modules, error) {
	return host.Modules{
		baseModule,
		systemAdminModule,
		systemAppModule,
	}, nil
}
