package biz

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"testing"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/sessionregistry"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	"github.com/liujitcn/kratos-core/biz"
	configv1 "github.com/liujitcn/kratos-kit/api/gen/go/config/v1"
	"github.com/liujitcn/kratos-kit/auth/authn/engine"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/cache/memory"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// loginBarrierIdentity 在第一份令牌签发期间暂停，以固定两个请求的重叠时序。
type loginBarrierIdentity struct {
	once    sync.Once
	entered chan struct{}
	resume  chan struct{}
}

// CreateIdentity 暂停首次签发后序列化认证声明。
func (a *loginBarrierIdentity) CreateIdentity(claims engine.AuthClaims) (string, error) {
	a.once.Do(func() { close(a.entered); <-a.resume })
	raw, err := json.Marshal(claims)
	return string(raw), err
}

// TestSingleSessionLoginRejectsOverlappingIssuance 验证禁止并发登录时重叠签发被拒绝，后续登录撤销旧会话。
func TestSingleSessionLoginRejectsOverlappingIssuance(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var connection *sql.DB
	connection, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err = connection.Close(); err != nil {
			t.Error(err)
		}
	})
	err = db.AutoMigrate(&models.BaseRole{}, &models.BaseDept{}, &models.BaseTenant{})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create(&models.BaseRole{ID: 1, Code: "user", Name: "普通用户", Status: 1}).Error
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create(&models.BaseDept{ID: 1, Name: "默认部门", Status: 1}).Error
	if err != nil {
		t.Fatal(err)
	}
	err = db.Create(&models.BaseTenant{ID: 1, Code: "default", Status: 1}).Error
	if err != nil {
		t.Fatal(err)
	}
	dataStore, err := data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	var manager *sessionregistry.LoginLocker
	var closeLocker func()
	manager, closeLocker, err = sessionregistry.NewLoginLocker(&configv1.Bootstrap{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeLocker()
	identity := &loginBarrierIdentity{entered: make(chan struct{}), resume: make(chan struct{})}
	tokens := authdata.NewUserToken(store, identity, "access:", "refresh:", time.Hour, 24*time.Hour)
	login := &LoginCase{
		BaseCase: &biz.BaseCase{Cache: store}, loginLocker: manager, userToken: tokens,
		baseRoleCase:   &BaseRoleCase{BaseRoleRepository: data.NewBaseRoleRepository(dataStore)},
		baseDeptCase:   &BaseDeptCase{BaseDeptRepository: data.NewBaseDeptRepository(dataStore)},
		baseTenantRepo: data.NewBaseTenantRepository(dataStore),
	}
	user := &models.BaseUser{ID: 1, Status: 1, TenantID: 1, RoleID: 1, DeptID: 1}
	responses := make(chan *basev1.LoginResponse, 1)
	errors := make(chan error, 1)
	go func() {
		response, issueErr := login.IssueUserToken(context.Background(), user)
		responses <- response
		errors <- issueErr
	}()
	select {
	case <-identity.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("首次登录未进入签发流程")
	}
	_, err = login.IssueUserToken(context.Background(), user)
	close(identity.resume)
	if err == nil {
		t.Error("并发签发同账号令牌必须被拒绝")
	}
	first := <-responses
	if err = <-errors; err != nil {
		t.Fatal(err)
	}
	var next *basev1.LoginResponse
	next, err = login.IssueUserToken(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	if tokens.IsAccessTokenValid(1, first.AccessToken) || !tokens.IsAccessTokenValid(1, next.AccessToken) {
		t.Fatal("后续单会话登录必须只保留新会话")
	}
}
