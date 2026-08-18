package admin

import "github.com/google/wire"

// ProviderSet 汇总系统管理端服务依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewBaseAreaService,
	NewAuthService,
	NewBaseApiService,
	NewBaseConfigService,
	NewBaseDeptService,
	NewBaseDictService,
	NewBaseJobService,
	NewBaseLanguageService,
	NewBaseLogService,
	NewBaseMenuService,
	NewBasePostService,
	NewBaseRoleService,
	NewBaseTenantService,
	NewBaseThirdAccountService,
	NewBaseI18nService,
	NewBaseUserService,
	NewCodeGenService,
	NewCodeGenColumnService,
	NewCodeGenProtoService,
	NewCodeGenTableService,
	NewBaseMigrationService,
	NewOpsMonitoringService,
	NewProjectDocumentService,
)
