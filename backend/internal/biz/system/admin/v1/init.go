package biz

import "github.com/google/wire"

// ProviderSet 汇总 system.admin.v1 业务依赖注入提供者。
var ProviderSet = wire.NewSet(
	NewAuthCase,
	NewBaseAPICase,
	NewBaseAreaCase,
	NewBaseConfigCase,
	NewBaseDeptCase,
	NewBaseDictCase,
	NewBaseDictItemCase,
	NewBaseJobCase,
	NewBaseJobLogCase,
	NewBaseLanguageCase,
	NewBaseLogCase,
	NewBaseMenuCase,
	NewBasePostCase,
	NewBaseRoleCase,
	NewBaseTenantCase,
	NewBaseTranslationCase,
	NewBaseUserCase,
	NewCasbinRuleCase,
	NewCodeGenCase,
	NewCodeGenColumnCase,
	NewCodeGenProtoCase,
	NewCodeGenTableCase,
	NewProjectDocumentCase,
	NewBaseMigrationCase,
)
