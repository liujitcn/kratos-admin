package config

import (
	"context"
	"fmt"
	"sort"

	"github.com/liujitcn/kratos-admin/backend/core/pkg/errorsx"

	bootstrapConfigv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/bootstrap"
	databaseGorm "github.com/liujitcn/kratos-kit/database/gorm"
	gormmigration "github.com/liujitcn/kratos-kit/database/gorm/migration"
)

// GetAppInfo 获取当前服务的应用信息。
func GetAppInfo(ctx *bootstrap.Context) *bootstrapConfigv1.AppInfo {
	return ctx.GetAppInfo()
}

// ParseAIModel 解析大模型配置。
func ParseAIModel(ctx *bootstrap.Context) *bootstrapConfigv1.AI_Model {
	cfg := ctx.GetConfig()
	// 未配置大模型参数时，返回空配置并由客户端保持关闭状态。
	if cfg == nil || cfg.GetAi() == nil || cfg.GetAi().GetModel() == nil {
		return &bootstrapConfigv1.AI_Model{}
	}
	return cfg.GetAi().GetModel()
}

// ParseOSS 解析对象存储配置。
func ParseOSS(ctx *bootstrap.Context) (*bootstrapConfigv1.Oss, error) {
	cfg := ctx.GetConfig()
	// 对象存储配置缺失时，直接返回错误。
	if cfg == nil || cfg.GetOss() == nil {
		return nil, errorsx.Internal("对象存储配置缺失")
	}
	return cfg.GetOss(), nil
}

// ParseTranslator 解析机器翻译配置。
func ParseTranslator(ctx *bootstrap.Context) *bootstrapConfigv1.Translator {
	cfg := ctx.GetConfig()
	if cfg == nil || cfg.GetTranslator() == nil {
		return &bootstrapConfigv1.Translator{}
	}
	return cfg.GetTranslator()
}

// ParseData 解析数据源配置。
func ParseData(ctx *bootstrap.Context) (*bootstrapConfigv1.Data, error) {
	cfg := ctx.GetConfig()
	// 数据源配置缺失时，直接返回错误。
	if cfg == nil || cfg.GetData() == nil {
		return nil, errorsx.Internal("数据源配置缺失")
	}
	return cfg.GetData(), nil
}

// NewDatabaseClient 创建全部数据库客户端并执行全部迁移贡献者。
func NewDatabaseClient(
	cfg *bootstrapConfigv1.Data,
	options []databaseGorm.ClientOption,
	runner *gormmigration.Runner,
	contributors gormmigration.AdditionalMigrations,
) (*databaseGorm.Client, func(), error) {
	configs, err := parseDatabaseConfigs(cfg)
	if err != nil {
		return nil, nil, err
	}
	names := make([]string, 0, len(configs))
	for name := range configs {
		names = append(names, name)
	}
	sort.Strings(names)
	clients := make(map[string]*databaseGorm.Client, len(configs))
	cleanups := make([]func(), 0, len(configs))
	cleanup := func() {
		for index := len(cleanups) - 1; index >= 0; index-- {
			cleanups[index]()
		}
	}
	for _, name := range names {
		clientOptions := append([]databaseGorm.ClientOption(nil), options...)
		clientOptions = append(clientOptions, databaseGorm.WithName(name))
		if name != gormmigration.DefaultTarget {
			// 非默认客户端由版本化脚本管理表结构，避免把管理后台模型自动建到业务库。
			clientOptions = append(clientOptions, databaseGorm.WithMigrateModels())
		}
		var client *databaseGorm.Client
		var clientCleanup func()
		client, clientCleanup, err = databaseGorm.NewGormClient(configs[name], clientOptions...)
		if err != nil {
			if clientCleanup != nil {
				clientCleanup()
			}
			cleanup()
			return nil, nil, err
		}
		cleanups = append(cleanups, clientCleanup)
		err = runner.SetClient(client)
		if err != nil {
			cleanup()
			return nil, nil, err
		}
		clients[name] = client
	}
	client, exists := clients[gormmigration.DefaultTarget]
	if !exists {
		cleanup()
		return nil, nil, fmt.Errorf("默认数据库客户端未配置")
	}
	// 每个贡献者都作为根模块执行一次，Runner 会按依赖递归执行前置模块。
	for _, contributor := range contributors {
		if contributor == nil {
			continue
		}
		err = runner.Run(context.Background(), contributor.Name())
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("执行迁移模块 %s: %w", contributor.Name(), err)
		}
	}
	return client, cleanup, nil
}

// parseDatabaseConfigs 合并兼容的单库配置和命名多数据源配置。
func parseDatabaseConfigs(cfg *bootstrapConfigv1.Data) (map[string]*bootstrapConfigv1.Data_Database, error) {
	if cfg == nil {
		return nil, errorsx.Internal("数据源配置缺失")
	}
	configs := make(map[string]*bootstrapConfigv1.Data_Database, len(cfg.GetDatabases())+1)
	for name, database := range cfg.GetDatabases() {
		if name == "" {
			return nil, errorsx.Internal("数据库名称不能为空")
		}
		if database == nil {
			return nil, errorsx.Internal(fmt.Sprintf("数据库配置不能为空: %s", name))
		}
		configs[name] = database
	}
	if database := cfg.GetDatabase(); database != nil {
		if _, exists := configs[gormmigration.DefaultTarget]; exists {
			return nil, errorsx.Internal("默认数据库配置重复")
		}
		configs[gormmigration.DefaultTarget] = database
	}
	if len(configs) == 0 {
		return nil, errorsx.Internal("数据库配置缺失")
	}
	return configs, nil
}

// ParseQueue 解析队列配置。
func ParseQueue(cfg *bootstrapConfigv1.Data) *bootstrapConfigv1.Data_Queue {
	return cfg.GetQueue()
}

// ParseRedis 解析 Redis 配置。
func ParseRedis(cfg *bootstrapConfigv1.Data) *bootstrapConfigv1.Data_Redis {
	return cfg.GetRedis()
}

// ParsePprof 解析性能分析配置。
func ParsePprof(ctx *bootstrap.Context) (*bootstrapConfigv1.Pprof, error) {
	cfg := ctx.GetConfig()
	// 性能分析配置缺失时，直接返回错误。
	if cfg == nil || cfg.GetPprof() == nil {
		return nil, errorsx.Internal("性能分析配置缺失")
	}
	return cfg.GetPprof(), nil
}

// ParseAuthnJWT 解析 JWT 认证配置。
func ParseAuthnJWT(ctx *bootstrap.Context) *bootstrapConfigv1.Authentication_Jwt {
	cfg := ctx.GetConfig()
	// 未配置 JWT 时，回退到项目默认认证参数。
	if cfg == nil || cfg.GetAuthn() == nil || cfg.GetAuthn().GetJwt() == nil {
		return &bootstrapConfigv1.Authentication_Jwt{
			Method: "HS256",
			Secret: "admin-base",
		}
	}
	return cfg.GetAuthn().GetJwt()
}

// ParseOAuth 解析 OAuth 配置。
func ParseOAuth(ctx *bootstrap.Context) *bootstrapConfigv1.OAuth {
	cfg := ctx.GetConfig()
	// 未配置 OAuth 时，回退为空配置，避免影响账号密码登录。
	if cfg == nil || cfg.GetOauth() == nil {
		return &bootstrapConfigv1.OAuth{}
	}
	return cfg.GetOauth()
}
