//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend"
	kratoscore "github.com/liujitcn/kratos-core"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// NewApp 通过 Core、Admin 和宿主 ProviderSet 注入并创建应用。
func NewApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		kratoscore.ProviderSet,
		backend.ProviderSet,
		hostProviderSet,
	))
}
