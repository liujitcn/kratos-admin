package _const

const (
	// BASE_DEPT_ID_APP_USER 表示移动端注册用户固定所属部门 ID。
	BASE_DEPT_ID_APP_USER int64 = 4
)

// OAuthClientTenantCode 返回开放授权客户端使用的数据权限租户编码。
// 默认租户编码拥有跨租户能力，客户端绑定默认租户时使用独立编码避免绕过租户隔离。
func OAuthClientTenantCode(tenantCode string, clientID string) string {
	if tenantCode == "0000" {
		return "oauth:" + clientID
	}
	return tenantCode
}
