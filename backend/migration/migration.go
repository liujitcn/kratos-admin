package migration

import (
	"embed"

	gormmigration "github.com/liujitcn/kratos-kit/database/gorm/migration"
)

// ModuleName 表示 kratos-admin 基础迁移模块名称。
const ModuleName gormmigration.ModuleName = "kratos-admin"

// NewMigrations 返回基础应用提供的数据库迁移贡献者集合。
func NewMigrations() gormmigration.AdditionalMigrations {
	return gormmigration.AdditionalMigrations{
		BaseContributor(),
	}
}

// BaseContributor 返回基础应用的数据库迁移贡献者。
func BaseContributor() gormmigration.Contributor {
	return baseContributor{}
}

type baseContributor struct{}

// Name 返回基础应用的迁移模块注册名称。
func (baseContributor) Name() gormmigration.ModuleName {
	return ModuleName
}

// Migrations 返回基础应用的版本化迁移资源。
func (baseContributor) Migrations() []gormmigration.Migration {
	return []gormmigration.Migration{
		{
			FS:   baseMigrationFS,
			Path: "assets/mysql",
		},
	}
}

//go:embed assets/mysql/*
var baseMigrationFS embed.FS
