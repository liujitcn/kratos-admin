//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v3"
	"github.com/google/wire"
	"github.com/liujitcn/kratos-admin/backend/internal/module"
	kratoscore "github.com/liujitcn/kratos-core"
	"github.com/liujitcn/kratos-kit/bootstrap"
)

// NewApp 通过 Core 根 ProviderSet 注入并创建应用。
func NewAdminApp(ctx *bootstrap.Context) (*kratos.App, func(), error) {
	panic(wire.Build(
		kratoscore.ProviderSet,
		module.ProviderSet,
	))
}
