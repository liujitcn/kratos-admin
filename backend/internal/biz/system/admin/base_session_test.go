package biz

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionregistry"
	"github.com/liujitcn/kratos-core/biz"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/cache/memory"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

// sessionIdentityCreator 序列化签发声明，用于测试实际令牌管理器。
type sessionIdentityCreator struct{}

// CreateIdentity 返回包含唯一声明的测试令牌。
func (sessionIdentityCreator) CreateIdentity(claims engine.AuthClaims) (string, error) {
	payload, err := json.Marshal(claims)
	return string(payload), err
}

// sessionTestTransport 为业务测试提供真实的请求头读取边界。
type sessionTestTransport struct {
	transport.Transporter
	header sessionTestHeader
}

// RequestHeader 返回测试请求头。
func (tr sessionTestTransport) RequestHeader() transport.Header { return tr.header }

// sessionTestHeader 提供当前测试会话的访问令牌。
type sessionTestHeader struct {
	transport.Header
	token string
}

// Get 返回 Authorization 请求头。
func (h sessionTestHeader) Get(key string) string {
	if key == "Authorization" {
		return "Bearer " + h.token
	}
	return ""
}

// TestCurrentSessionAfterOtherDeviceLogout 验证最新设备退出后仍可查询原设备的会话。
func TestCurrentSessionAfterOtherDeviceLogout(t *testing.T) {
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	manager := data.NewUserToken(store, sessionIdentityCreator{}, "access:", "refresh:", time.Hour, 24*time.Hour)
	user := &data.UserTokenPayload{UserId: 1, TenantId: 1, UserName: "alice", RoleCode: "super"}
	var access, refresh string
	access, refresh, err = manager.GenerateTokenForSession(user, "first")
	if err != nil {
		t.Fatal(err)
	}
	record := sessionregistry.Record{SessionID: "first", UserID: 1, AccessToken: access, RefreshToken: refresh}
	if err = sessionregistry.Register(store, record); err != nil {
		t.Fatal(err)
	}
	var secondAccess, secondRefresh string
	secondAccess, secondRefresh, err = manager.GenerateTokenForSession(user, "second")
	if err != nil {
		t.Fatal(err)
	}
	if err = manager.RemoveTokenForSession(1, "second", secondAccess, secondRefresh); err != nil {
		t.Fatal(err)
	}
	ctx := engine.ContextWithAuthClaims(context.Background(), user.MakeAuthClaims())
	ctx = transport.NewServerContext(ctx, sessionTestTransport{header: sessionTestHeader{token: access}})
	sessionCase := NewBaseSessionCase(&biz.BaseCase{Cache: store}, manager)
	var current *adminv1.BaseSession
	current, err = sessionCase.GetCurrentBaseSession(ctx)
	if err != nil {
		t.Fatalf("其他设备退出后当前会话应仍可查询: %v", err)
	}
	if current.SessionId != "first" || !current.Current {
		t.Fatalf("查询到了错误会话: %v", current)
	}
}

// TestListCurrentSessionsIsolatesOwner 验证本人列表包含多设备、标记当前设备并排除其他用户。
func TestListCurrentSessionsIsolatesOwner(t *testing.T) {
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	manager := data.NewUserToken(store, sessionIdentityCreator{}, "access:", "refresh:", time.Hour, 24*time.Hour)
	user := &data.UserTokenPayload{UserId: 1, TenantId: 1, RoleCode: "user", UserName: "alice"}
	currentAccess := ""
	for _, id := range []string{"first", "second", "other"} {
		owner := *user
		if id == "other" {
			owner.UserId = 2
		}
		var access, refresh string
		access, refresh, err = manager.GenerateTokenForSession(&owner, id)
		if err != nil {
			t.Fatal(err)
		}
		if id == "second" {
			currentAccess = access
		}
		err = sessionregistry.Register(store, sessionregistry.Record{SessionID: id, UserID: owner.UserId, AccessToken: access, RefreshToken: refresh, IssuedAt: time.Now()})
		if err != nil {
			t.Fatal(err)
		}
	}
	ctx := engine.ContextWithAuthClaims(context.Background(), user.MakeAuthClaims())
	ctx = transport.NewServerContext(ctx, sessionTestTransport{header: sessionTestHeader{token: currentAccess}})
	service := NewBaseSessionCase(&biz.BaseCase{Cache: store}, manager)
	var result *adminv1.ListCurrentBaseSessionsResponse
	result, err = service.ListCurrentBaseSessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sessions) != 2 || result.Sessions[0].SessionId != "second" || !result.Sessions[0].Current || result.Sessions[1].Current {
		t.Fatalf("本人多会话或当前设备标记错误: %v", result)
	}
	for _, session := range result.Sessions {
		if session.UserId != 1 {
			t.Fatal("泄露了其他用户会话")
		}
	}
}

// TestOnlineSessionTenantFilter 验证默认租户精确筛选，非默认租户不能指定其他租户。
func TestOnlineSessionTenantFilter(t *testing.T) {
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	manager := data.NewUserToken(store, sessionIdentityCreator{}, "access:", "refresh:", time.Hour, 24*time.Hour)
	for index, tenant := range []string{"tenant-a", "tenant-ab"} {
		user := &data.UserTokenPayload{UserId: int64(index + 1), TenantCode: tenant}
		var access, refresh string
		access, refresh, err = manager.GenerateTokenForSession(user, tenant)
		if err != nil {
			t.Fatal(err)
		}
		err = sessionregistry.Register(store, sessionregistry.Record{SessionID: tenant, TenantCode: tenant, UserID: user.UserId, AccessToken: access, RefreshToken: refresh})
		if err != nil {
			t.Fatal(err)
		}
	}
	service := NewBaseSessionCase(&biz.BaseCase{Cache: store}, manager)
	for _, test := range []struct{ viewer, filter, want string }{
		{gorm.DefaultTenantCode, "tenant-a", "tenant-a"},
		{"tenant-a", "tenant-ab", "tenant-a"},
	} {
		user := &data.UserTokenPayload{UserId: 1, RoleCode: "super", TenantCode: test.viewer}
		ctx := engine.ContextWithAuthClaims(context.Background(), user.MakeAuthClaims())
		var result *adminv1.PageOnlineBaseSessionsResponse
		result, err = service.PageOnlineBaseSessions(ctx, &adminv1.PageOnlineBaseSessionsRequest{TenantCode: test.filter, PageNum: 1, PageSize: 10})
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 1 || result.Sessions[0].TenantCode != test.want {
			t.Fatalf("租户筛选错误: %v", result)
		}
	}
}
