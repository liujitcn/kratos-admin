package kratosadmin

import (
	"github.com/liujitcn/kratos-admin/backend/core"
	"github.com/liujitcn/kratos-admin/backend/core/pkg/projectdoc"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/migration"
	backendmodule "github.com/liujitcn/kratos-admin/backend/module"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	gormmigration "github.com/liujitcn/kratos-kit/database/gorm/migration"
)

// Option 配置 Backend 模块及独立应用装配。
type Option func(*options)

type options struct {
	additionalModules          AdditionalModules
	configuredProjectDocuments projectdoc.ConfiguredDocuments
	databaseOptions            []databaseGorm.ClientOption
	migrations                 gormmigration.AdditionalMigrations
}

// WithAdditionalModules 追加由同一启动器挂载的扩展模块。
func WithAdditionalModules(modules ...core.Module) Option {
	return func(opts *options) {
		opts.additionalModules = append(opts.additionalModules, modules...)
	}
}

// WithProjectDocuments 追加由宿主项目直接提供的文档。
func WithProjectDocuments(documents ...projectdoc.Document) Option {
	return func(opts *options) {
		opts.configuredProjectDocuments = append(opts.configuredProjectDocuments, documents...)
	}
}

// WithDatabaseOptions 追加 Backend 数据库客户端选项。
func WithDatabaseOptions(values ...databaseGorm.ClientOption) Option {
	return func(opts *options) {
		opts.databaseOptions = append(opts.databaseOptions, values...)
	}
}

// WithMigrations 追加与 Backend 数据库一起执行的迁移贡献者。
func WithMigrations(contributors ...gormmigration.Contributor) Option {
	return func(opts *options) {
		opts.migrations = append(opts.migrations, contributors...)
	}
}

// newOptions 合并默认迁移和调用方提供的装配选项。
func newOptions(optionValues []Option) options {
	opts := options{
		databaseOptions: []databaseGorm.ClientOption{databaseGorm.WithMigrateModels(data.Models()...)},
		migrations:      migration.NewMigrations(),
	}
	for _, option := range optionValues {
		option(&opts)
	}
	opts.migrations = append(opts.migrations, backendmodule.MigrationContributors(core.Modules(opts.additionalModules))...)
	return opts
}
