package username

import (
	"os"
	"strings"
	"unicode/utf8"

	"github.com/liujitcn/kratos-core/errorsx"
)

const (
	// PlatformAdministrator 是新装平台管理员的合规登录名。
	PlatformAdministrator = "platformoperator"
	// TenantAdministrator 是新建租户管理员的合规登录名。
	TenantAdministrator = "tenantoperator"
)

// Validate 统一校验创建、导入及个人改名入口，部署可补充产品正式名称和缩写。
func Validate(name string) error {
	if name == "" || utf8.RuneCountInString(name) > 50 {
		return errorsx.InvalidArgument("用户账号不能为空且不超过50个字符")
	}
	reserved := []string{"admin", "administrator", "sa", "root", "sysadmin", "newcapec", "newcap", "webadmin", "guest", "superadmin", "user", "system", "guanliyuan", "shuikong", "shuiwu", "xysh", "xyshfw"}
	for _, item := range append(reserved, strings.Split(os.Getenv("SECURITY_RESERVED_USERNAMES"), ",")...) {
		// 部署名单允许逗号后的排版空白，不修改用户实际提交的账号。
		if strings.EqualFold(name, strings.TrimSpace(item)) {
			return errorsx.InvalidArgument("用户账号不允许使用系统保留名称")
		}
	}
	return nil
}
