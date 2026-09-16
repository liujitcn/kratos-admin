package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httptransport "github.com/go-kratos/kratos/v3/transport/http"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

type tenantProjectRouteStub struct {
	adminv1.BaseTenantProjectServiceHTTPServer
}

type tenantProjectGrantRouteStub struct {
	adminv1.BaseTenantProjectGrantServiceHTTPServer
	pageCalled bool
}

// PageBaseTenantProjectGrant 返回空分页结果并记录路由命中。
func (s *tenantProjectGrantRouteStub) PageBaseTenantProjectGrant(context.Context, *adminv1.PageBaseTenantProjectGrantRequest) (*adminv1.PageBaseTenantProjectGrantResponse, error) {
	s.pageCalled = true
	return &adminv1.PageBaseTenantProjectGrantResponse{}, nil
}

// TestRegisterTenantProjectHTTP 防止项目授权静态路径被项目详情动态路径抢先匹配。
func TestRegisterTenantProjectHTTP(t *testing.T) {
	server := httptransport.NewServer()
	grant := &tenantProjectGrantRouteStub{}
	registerTenantProjectHTTP(server, &tenantProjectRouteStub{}, grant)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/base/tenant/project/grant?page_num=1&page_size=10", nil)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("项目授权分页路由返回状态码 %d，响应：%s", response.Code, strings.TrimSpace(response.Body.String()))
	}
	if !grant.pageCalled {
		t.Fatal("项目授权分页路由未命中授权服务")
	}
}
