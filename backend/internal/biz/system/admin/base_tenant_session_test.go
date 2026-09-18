package biz

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/data"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	authdata "github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/cache/memory"
	kitgorm "github.com/liujitcn/kratos-kit/database/gorm"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestRevokeTenantUserTokens 验证停用租户只撤销该租户用户的全部登录会话。
func TestRevokeTenantUserTokens(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	var connection *sql.DB
	connection, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() {
		if err = connection.Close(); err != nil {
			t.Error(err)
		}
	})
	if err = db.AutoMigrate(&models.BaseUser{}); err != nil {
		t.Fatal(err)
	}
	users := []*models.BaseUser{
		{ID: 1, TenantID: 7, UserName: "tenant-seven", UserCode: "tenant-seven", RoleID: 1, DeptID: 1, Status: 1},
		{ID: 2, TenantID: 8, UserName: "tenant-eight", UserCode: "tenant-eight", RoleID: 1, DeptID: 1, Status: 1},
	}
	for _, user := range users {
		if err = db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	cacheStore, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	userToken := authdata.NewUserToken(cacheStore, sessionIdentityCreator{}, "access:", "refresh:", time.Hour, 24*time.Hour)
	for _, user := range users {
		if _, _, err = userToken.GenerateTokenForSession(&authdata.UserTokenPayload{UserId: user.ID}, "session"); err != nil {
			t.Fatal(err)
		}
	}
	var store *data.Data
	store, err = data.NewData(map[string]*kitgorm.Client{kitgorm.DefaultClientName: {DB: db}})
	if err != nil {
		t.Fatal(err)
	}
	baseCase := &BaseTenantCase{baseUserRepo: data.NewBaseUserRepository(store), userToken: userToken}
	if err = baseCase.revokeTenantUserTokens(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if sessions, err := userToken.GetTokenSessions(1); err != nil || len(sessions) != 0 {
		t.Fatalf("租户 7 会话未全部撤销: sessions=%v err=%v", sessions, err)
	}
	if sessions, err := userToken.GetTokenSessions(2); err != nil || len(sessions) != 1 {
		t.Fatalf("租户 8 会话被错误撤销: sessions=%v err=%v", sessions, err)
	}
}
