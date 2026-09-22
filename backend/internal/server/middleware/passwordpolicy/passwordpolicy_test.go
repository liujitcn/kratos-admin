package passwordpolicy

import "testing"

func TestPasswordChangeOperationAllowsCurrentPasswordPolicy(t *testing.T) {
	if !passwordChangeOperation("/system.admin.v1.AuthService/GetCurrentPasswordPolicy") {
		t.Fatal("强制改密会话应允许读取当前用户密码策略")
	}
	if passwordChangeOperation("/system.admin.v1.BaseLoginPolicyService/PageBaseLoginPolicy") {
		t.Fatal("强制改密会话不应放行登录策略管理接口")
	}
}
