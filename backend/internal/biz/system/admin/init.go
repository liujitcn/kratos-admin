package biz

import "github.com/google/wire"

// ProviderSet 汇总 system.admin 业务依赖注入提供者。
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
	NewBaseAuditLogCase,
	NewBaseDashboardCase,
	NewBaseFileCase,
	NewBaseLanguageCase,
	NewBaseMenuCase,
	NewBasePostCase,
	NewBaseRoleCase,
	NewBaseTenantCase,
	NewBaseI18nCase,
	NewBaseUserCase,
	NewCasbinRuleCase,
	NewCodeGenCase,
	NewCodeGenColumnCase,
	NewCodeGenProtoCase,
	NewCodeGenTableCase,
	NewOpsMonitoringCase,
	NewRuntimeLogCase,
	NewBaseMigrationCase,
	NewBaseSessionCase,
	NewBaseLoginPolicyCase,
	NewOauthClientCase,
)

// MessageProviderSet 汇总站内信管理和投递业务依赖。
var MessageProviderSet = wire.NewSet(
	NewBaseMessageCategoryCase,
	NewBaseMessageCase,
)
