package username

import "testing"

// TestReservedUsernames 验证数据库配置的保留名称不可通过大小写绕过，正常账号保持可用。
func TestReservedUsernames(t *testing.T) {
	reserved := "admin,administrator,sa,root,sysadmin,newcapec,newcap,webadmin,guest,superadmin,user,system,guanliyuan,shuikong,shuiwu,xysh,xyshfw,CampusService, productsys"
	for _, name := range []string{"admin", "ADMIN", "Root", "guanliyuan", "CampusService", "productsys", ""} {
		if Validate(name, reserved) == nil {
			t.Fatalf("保留或无效名称未拒绝: %q", name)
		}
	}
	for _, name := range []string{PlatformAdministrator, TenantAdministrator, "zhangsan01", "super"} {
		if err := Validate(name, reserved); err != nil {
			t.Fatalf("正常账号被拒绝: %s: %v", name, err)
		}
	}
}
