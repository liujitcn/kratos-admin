package plugin

import "testing"

// TestNetworkPolicyAllowsConfiguredPrivateHTTPMCP 验证平台批准的三个私网 MCP HTTP 端点可通过校验。
func TestNetworkPolicyAllowsConfiguredPrivateHTTPMCP(t *testing.T) {
	policy := NetworkPolicy{AllowedHosts: []string{"192.168.60.52"}, AllowPrivate: true}
	for _, endpoint := range []string{
		"http://192.168.60.52:8001/mcp",
		"http://192.168.60.52:8002/mcp",
		"http://192.168.60.52:8003/mcp",
	} {
		_, err := policy.ValidateURL(endpoint)
		if err != nil {
			t.Fatalf("应允许 %s: %v", endpoint, err)
		}
	}
}

// TestNetworkPolicyRejectsPrivateHTTPWithoutApproval 验证未开启私网策略时仍拒绝 HTTP MCP 端点。
func TestNetworkPolicyRejectsPrivateHTTPWithoutApproval(t *testing.T) {
	policy := NetworkPolicy{AllowedHosts: []string{"192.168.60.52"}}
	_, err := policy.ValidateURL("http://192.168.60.52:8001/mcp")
	if err == nil {
		t.Fatal("未开启私网策略时应拒绝 HTTP MCP 端点")
	}
}
