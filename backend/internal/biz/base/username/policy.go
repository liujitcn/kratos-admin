package username

import (
	"strings"
	"unicode/utf8"

	"github.com/liujitcn/kratos-core/errorsx"
)

const (
	// PlatformAdministrator 是平台管理员的登录名。
	PlatformAdministrator = "mgr"
	// TenantAdministrator 是新建租户管理员的登录名。
	TenantAdministrator = "tmgr"
)

// Validate 根据系统配置统一校验创建、导入及个人改名入口。
func Validate(name, reservedValue string) error {
	if name == "" || utf8.RuneCountInString(name) > 50 {
		return errorsx.InvalidArgument("用户账号不能为空且不超过50个字符")
	}
	for item := range strings.SplitSeq(reservedValue, ",") {
		// 配置名单允许逗号后的排版空白，不修改用户实际提交的账号。
		if strings.EqualFold(name, strings.TrimSpace(item)) {
			return errorsx.InvalidArgument("用户账号不允许使用系统保留名称")
		}
	}
	return nil
}
