package username

import "testing"

// TestReservedUsernames 验证保留名称不可通过大小写绕过，正常账号与可配置产品名保持可用。
func TestReservedUsernames(t *testing.T) {
	t.Setenv("SECURITY_RESERVED_USERNAMES", "CampusService, productsys")
	for _, name := range []string{"admin", "ADMIN", "Root", "guanliyuan", "CampusService", "productsys", ""} {
		if Validate(name) == nil {
			t.Fatalf("保留或无效名称未拒绝: %q", name)
		}
	}
	for _, name := range []string{PlatformAdministrator, TenantAdministrator, "zhangsan01", "super"} {
		if err := Validate(name); err != nil {
			t.Fatalf("正常账号被拒绝: %s: %v", name, err)
		}
	}
}
