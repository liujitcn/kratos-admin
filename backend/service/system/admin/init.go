package admin

import (
	"github.com/liujitcn/kratos-admin/backend/service/system/admin/biz"
	"github.com/liujitcn/kratos-admin/backend/service/system/admin/codegen"

	"github.com/google/wire"
)

// ProviderSet 汇总系统管理端服务依赖注入提供者。
var ProviderSet = wire.NewSet(

	NewBaseAreaService,
	codegen.NewManager,
	biz.NewAuthCase,
	biz.NewBaseAPICase,

	biz.NewBaseAreaCase,
	biz.NewBaseConfigCase,
	biz.NewBaseDeptCase,
	biz.NewBaseDictCase,
	biz.NewBaseDictItemCase,
	biz.NewBaseJobCase,
	biz.NewBaseJobLogCase,
	biz.NewBaseLogCase,
	biz.NewBaseMenuCase,
	biz.NewBasePostCase,
	biz.NewBaseRoleCase,
	biz.NewBaseTenantCase,
	biz.NewBaseUserCase,
	biz.NewCasbinRuleCase,
	biz.NewCodeGenCase,
	biz.NewCodeGenColumnCase,
	biz.NewCodeGenProtoCase,
	biz.NewCodeGenTableCase,
	NewAuthService,
	NewBaseApiService,
	NewBaseConfigService,
	NewBaseDeptService,
	NewBaseDictService,
	NewBaseJobService,
	NewBaseLogService,
	NewBaseMenuService,
	NewBasePostService,
	NewBaseRoleService,
	NewBaseTenantService,
	NewBaseUserService,
	NewCodeGenService,
	NewCodeGenColumnService,
	NewCodeGenProtoService,
	NewCodeGenTableService,
)
