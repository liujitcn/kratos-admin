package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionregistry"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionstate"
	"github.com/liujitcn/kratos-core/biz"
	_const "github.com/liujitcn/kratos-core/const"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/cache"
)

// TestPageOnlineBaseSessionsFiltersExpired 校验在线列表排除失效会话且沿用应用用户的免超时策略。
func TestPageOnlineBaseSessionsFiltersExpired(t *testing.T) {
	now := time.Now().UTC()
	for _, testCase := range []struct {
		name          string
		role          string
		state         *sessionstate.State
		revoke        bool
		accessExpired bool
		tokensExpired bool
		wantTotal     int32
	}{
		{name: "有效管理会话", role: _const.BASE_ROLE_CODE_SUPER, state: &sessionstate.State{CreatedAt: now, LastActiveAt: now}, wantTotal: 1},
		{name: "管理会话空闲超时", role: _const.BASE_ROLE_CODE_ADMIN, state: &sessionstate.State{CreatedAt: now.Add(-time.Hour), LastActiveAt: now.Add(-time.Hour)}},
		{name: "管理会话超过最长生命周期", role: _const.BASE_ROLE_CODE_TENANT, state: &sessionstate.State{CreatedAt: now.Add(-24 * time.Hour), LastActiveAt: now}},
		{name: "管理会话状态缺失", role: _const.BASE_ROLE_CODE_SUPER},
		{name: "会话已撤销", role: _const.BASE_ROLE_CODE_SUPER, state: &sessionstate.State{CreatedAt: now, LastActiveAt: now}, revoke: true},
		{name: "访问与刷新令牌均到期", role: _const.BASE_ROLE_CODE_USER, tokensExpired: true},
		{name: "普通应用用户不受管理会话超时限制", role: _const.BASE_ROLE_CODE_USER, wantTotal: 1},
		{name: "认证应用用户不受管理会话超时限制", role: _const.BASE_ROLE_CODE_AUTHUSER, wantTotal: 1},
		{name: "应用用户遗留管理状态超时仍按当前身份豁免", role: _const.BASE_ROLE_CODE_USER, state: &sessionstate.State{CreatedAt: now.Add(-24 * time.Hour), LastActiveAt: now}, wantTotal: 1},
		{name: "访问令牌到期但会话可刷新", role: _const.BASE_ROLE_CODE_ADMIN, state: &sessionstate.State{CreatedAt: now, LastActiveAt: now}, accessExpired: true, wantTotal: 1},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			caseValue := newSessionTestCase(t)
			if testCase.accessExpired {
				caseValue.userToken = data.NewUserToken(caseValue.Cache, &sessionTestIssuer{}, "test:access:", "test:refresh:", -time.Second, time.Hour)
			}
			if testCase.tokensExpired {
				caseValue.userToken = data.NewUserToken(caseValue.Cache, &sessionTestIssuer{}, "test:access:", "test:refresh:", -time.Second, -time.Second)
			}
			record := seedSession(t, caseValue, 10, "0000", testCase.role, testCase.state)
			var err error
			if testCase.revoke {
				err = caseValue.userToken.RemoveTokenForSession(record.UserID, record.SessionID, record.AccessToken, record.RefreshToken)
				if err != nil {
					t.Fatal(err)
				}
			}
			var response *adminv1.PageOnlineBaseSessionsResponse
			response, err = caseValue.PageOnlineBaseSessions(sessionTestContext("0000"), &adminv1.PageOnlineBaseSessionsRequest{})
			if err != nil {
				t.Fatal(err)
			}
			if response.Total != testCase.wantTotal || len(response.Sessions) != int(testCase.wantTotal) {
				t.Fatalf("有效会话总数=%d，列表条数=%d，期望=%d", response.Total, len(response.Sessions), testCase.wantTotal)
			}
			var saved sessionregistry.Record
			saved, err = sessionregistry.Get(caseValue.Cache, record.SessionID)
			if err != nil || saved.SessionID != record.SessionID {
				t.Fatalf("列表查询不应删除会话记录: %v", err)
			}
			if testCase.state != nil {
				var state sessionstate.State
				state, err = sessionstate.Read(caseValue.Cache, record.SessionID)
				if err != nil || !reflect.DeepEqual(state, *testCase.state) {
					t.Fatalf("列表查询不应延长会话活动时间: state=%+v, err=%v", state, err)
				}
			}
		})
	}
}

// TestPageOnlineBaseSessionsPagination 校验先过滤失效会话再分页，并保留租户及关键字筛选。
func TestPageOnlineBaseSessionsPagination(t *testing.T) {
	caseValue := newSessionTestCase(t)
	now := time.Now().UTC()
	active := &sessionstate.State{CreatedAt: now, LastActiveAt: now}
	seedSession(t, caseValue, 10, "1000", _const.BASE_ROLE_CODE_ADMIN, &sessionstate.State{CreatedAt: now.Add(-time.Hour), LastActiveAt: now.Add(-time.Hour)})
	seedSession(t, caseValue, 11, "1000", _const.BASE_ROLE_CODE_ADMIN, active)
	seedSession(t, caseValue, 12, "1000", _const.BASE_ROLE_CODE_ADMIN, active)
	seedSession(t, caseValue, 13, "2000", _const.BASE_ROLE_CODE_ADMIN, active)
	for _, testCase := range []struct {
		name      string
		tenant    string
		request   *adminv1.PageOnlineBaseSessionsRequest
		wantTotal int32
		wantRows  int
	}{
		{name: "全局第一页", tenant: "0000", request: &adminv1.PageOnlineBaseSessionsRequest{PageNum: 1, PageSize: 2}, wantTotal: 3, wantRows: 2},
		{name: "全局第二页", tenant: "0000", request: &adminv1.PageOnlineBaseSessionsRequest{PageNum: 2, PageSize: 2}, wantTotal: 3, wantRows: 1},
		{name: "指定租户", tenant: "0000", request: &adminv1.PageOnlineBaseSessionsRequest{TenantCode: "1000"}, wantTotal: 2, wantRows: 2},
		{name: "关键字", tenant: "0000", request: &adminv1.PageOnlineBaseSessionsRequest{Keyword: "user11"}, wantTotal: 1, wantRows: 1},
		{name: "租户管理员不可切换租户", tenant: "1000", request: &adminv1.PageOnlineBaseSessionsRequest{TenantCode: "2000"}, wantTotal: 2, wantRows: 2},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			response, err := caseValue.PageOnlineBaseSessions(sessionTestContext(testCase.tenant), testCase.request)
			if err != nil {
				t.Fatal(err)
			}
			if response.Total != testCase.wantTotal || len(response.Sessions) != testCase.wantRows {
				t.Fatalf("分页总数=%d，列表条数=%d，期望=%d/%d", response.Total, len(response.Sessions), testCase.wantTotal, testCase.wantRows)
			}
		})
	}
}

// TestPageOnlineBaseSessionsCacheFailure 校验会话状态读取失败应报错，不能伪装成正常在线或离线。
func TestPageOnlineBaseSessionsCacheFailure(t *testing.T) {
	caseValue := newSessionTestCase(t)
	now := time.Now().UTC()
	seedSession(t, caseValue, 10, "0000", _const.BASE_ROLE_CODE_ADMIN, &sessionstate.State{CreatedAt: now, LastActiveAt: now})
	caseValue.Cache = sessionTestFaultCache{Cache: caseValue.Cache}
	_, err := caseValue.PageOnlineBaseSessions(sessionTestContext("0000"), &adminv1.PageOnlineBaseSessionsRequest{})
	if err == nil {
		t.Fatal("会话状态缓存故障应返回错误")
	}
}

// sessionTestIssuer 为测试生成独立访问令牌。
type sessionTestIssuer struct {
	sequence int
}

// CreateIdentity 生成无需外部 JWT 配置的测试令牌。
func (issuer *sessionTestIssuer) CreateIdentity(_ engine.AuthClaims) (string, error) {
	issuer.sequence++
	return fmt.Sprintf("test-access-%d", issuer.sequence), nil
}

// sessionTestFaultCache 仅模拟会话状态读取故障。
type sessionTestFaultCache struct {
	cache.Cache
}

// Get 保留普通缓存读取，仅使服务端会话状态不可读。
func (store sessionTestFaultCache) Get(key string) (string, error) {
	if strings.Contains(key, "auth:session:state:") {
		return "", errors.New("测试会话缓存不可用")
	}
	return store.Cache.Get(key)
}

// newSessionTestCase 创建使用真实内存缓存和会话令牌管理器的业务实例。
func newSessionTestCase(t *testing.T) *BaseSessionCase {
	t.Helper()
	store, cleanup, err := cache.NewCache(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	tokens := data.NewUserToken(store, &sessionTestIssuer{}, "test:access:", "test:refresh:", time.Hour, 24*time.Hour)
	return NewBaseSessionCase(&biz.BaseCase{Cache: store}, tokens)
}

// seedSession 写入真实会话令牌、刷新身份、展示记录和可选服务端状态。
func seedSession(t *testing.T, caseValue *BaseSessionCase, userID int64, tenant, role string, state *sessionstate.State) sessionregistry.Record {
	t.Helper()
	authInfo := &data.UserTokenPayload{UserId: userID, UserName: fmt.Sprintf("user%d", userID), TenantCode: tenant, RoleCode: role}
	record := sessionregistry.NewRecord(userID, authInfo.UserName, tenant, "127.0.0.1", "测试设备", "测试浏览器", "", "", time.Now())
	var err error
	record.AccessToken, record.RefreshToken, err = caseValue.userToken.GenerateTokenForSession(authInfo, record.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	var payload []byte
	payload, err = json.Marshal(authInfo)
	if err != nil {
		t.Fatal(err)
	}
	err = caseValue.Cache.Set(sessionregistry.RefreshTokenAuthKey(record.RefreshToken), string(payload), 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if state != nil {
		payload, err = json.Marshal(state)
		if err != nil {
			t.Fatal(err)
		}
		err = caseValue.Cache.Set("shared:auth:session:state:"+record.SessionID, string(payload), 24*time.Hour)
		if err != nil {
			t.Fatal(err)
		}
	}
	err = sessionregistry.Register(caseValue.Cache, record)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

// sessionTestContext 构造指定租户管理员的请求认证上下文。
func sessionTestContext(tenant string) context.Context {
	payload := &data.UserTokenPayload{UserId: 1, UserName: "operator", TenantCode: tenant, RoleCode: _const.BASE_ROLE_CODE_TENANT}
	return engine.ContextWithAuthClaims(context.Background(), payload.MakeAuthClaims())
}
