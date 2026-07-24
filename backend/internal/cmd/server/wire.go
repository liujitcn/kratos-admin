//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	kratosadmin "github.com/liujitcn/kratos-admin/backend"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// initApp 初始化 Kratos 应用实例。
func initApp(*bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		kratosadmin.ProviderSet,
		wire.Value(kratosadmin.AdditionalModules(nil)),
		kratosadmin.NewApp,
	))
}
