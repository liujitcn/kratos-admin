package username

import (
	"strings"

	"github.com/liujitcn/kratos-core/errorsx"
)

// Validate 拒绝常见管理账号及产品保留名称，保留原始输入而不改写登录标识。
func Validate(value string) error {
	// MySQL 常见 PAD SPACE 排序规则会忽略尾空格；仅拒绝首尾空白，不改写账号。
	if strings.TrimSpace(value) != value {
		return errorsx.InvalidArgument("用户账号首尾不能包含空白字符")
	}
	switch strings.ToLower(value) {
	case "admin", "administrator", "sa", "root", "sysadmin", "newcapec", "newcap", "webadmin", "guest", "superadmin", "user", "system", "guanliyuan", "kratosadmin", "kratos-admin":
		return errorsx.InvalidArgument("用户账号不能使用保留名称")
	default:
		return nil
	}
}
