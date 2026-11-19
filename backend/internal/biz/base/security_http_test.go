package biz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/securityconfig"
)

// TestSecurityConfiguredClientAllowsExplicitPrivateCIDR 验证显式授权的私网 CIDR 可访问。
func TestSecurityConfiguredClientAllowsExplicitPrivateCIDR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()
	previous := securityconfig.Current()
	configured := previous
	configured.AIAllowedCIDRs = []string{"127.0.0.1/32"}
	securityconfig.Set(configured)
	t.Cleanup(func() { securityconfig.Set(previous) })

	client, err := securityconfig.Current().AIHTTPClient(time.Second)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
}
