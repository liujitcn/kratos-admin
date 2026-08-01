package module

import (
	"github.com/liujitcn/kratos-admin/backend/core"
	gormmigration "github.com/liujitcn/kratos-kit/database/gorm/migration"
)

// MigrationContributor 表示可向 Admin 数据库贡献版本化迁移的模块。
type MigrationContributor interface {
	// MigrationContributors 返回模块提供的数据库迁移贡献者。
	MigrationContributors() gormmigration.AdditionalMigrations
}

// MigrationContributors 汇总外部模块贡献的数据库迁移。
func MigrationContributors(modules core.Modules) gormmigration.AdditionalMigrations {
	migrations := make(gormmigration.AdditionalMigrations, 0)
	for _, module := range modules {
		contributor, ok := module.(MigrationContributor)
		if !ok {
			continue
		}
		migrations = append(migrations, contributor.MigrationContributors()...)
	}
	return migrations
}
