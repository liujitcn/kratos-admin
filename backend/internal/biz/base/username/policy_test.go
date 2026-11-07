package username

import (
	"strings"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
)

// TestValidateReservedNames 验证保留名称及其大小写变体被拒绝而普通账号可用。
func TestValidateReservedNames(t *testing.T) {
	for _, name := range []string{"admin", "administrator", "sa", "root", "sysadmin", "newcapec", "newcap", "webadmin", "guest", "superadmin", "user", "system", "guanliyuan", "kratosadmin", "kratos-admin"} {
		for _, value := range []string{name, strings.ToUpper(name)} {
			err := Validate(value)
			if err == nil || errors.Code(err) != 400 {
				t.Errorf("保留账号 %q 必须返回参数错误，实际 %v", value, err)
			}
		}
	}
	for _, value := range []string{"alice", "tenant_42", "rooter", "user_2026", "cp00a2rv6412g89vmjq0"} {
		if err := Validate(value); err != nil {
			t.Errorf("正常账号 %q 被拒绝: %v", value, err)
		}
	}
}

// TestValidateRejectsBoundaryWhitespace 验证保留账号不能借首尾空白绕过且不额外限制中间字符。
func TestValidateRejectsBoundaryWhitespace(t *testing.T) {
	for _, value := range []string{"AdMiN ", " AdMiN", "alice\t", "\u3000alice", "alice\u00a0"} {
		err := Validate(value)
		if err == nil || errors.Code(err) != 400 {
			t.Errorf("首尾空白账号 %q 必须拒绝，实际 %v", value, err)
		}
	}
	if err := Validate("alice smith"); err != nil {
		t.Fatalf("中间字符不属于首尾空白限制: %v", err)
	}
}
