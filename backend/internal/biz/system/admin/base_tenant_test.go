package biz

import (
	"testing"

	"github.com/liujitcn/go-utils/crypto"
	"github.com/liujitcn/kratos-admin/backend/internal/biz/base/loginpolicy"
	passwordPolicy "github.com/liujitcn/kratos-admin/backend/internal/biz/base/password"
)

// TestBuildTenantAdminCredentialsUsesGlobalPassword 验证已配置初始化密码时直接复用密码哈希。
func TestBuildTenantAdminCredentialsUsesGlobalPassword(t *testing.T) {
	configuredHash := "$2a$10$configured-password-hash"
	password, initialPassword, err := buildTenantAdminCredentials(loginpolicy.PasswordConfig{InitialPasswordHash: configuredHash})
	if err != nil {
		t.Fatalf("buildTenantAdminCredentials() error = %v", err)
	}
	if password != configuredHash {
		t.Fatalf("password = %q, want configured hash", password)
	}
	if initialPassword != "" {
		t.Fatalf("initialPassword = %q, want empty", initialPassword)
	}
}

// TestBuildTenantAdminCredentialsGeneratesUsablePassword 验证未配置初始化密码时生成可登录且符合策略的随机密码。
func TestBuildTenantAdminCredentialsGeneratesUsablePassword(t *testing.T) {
	config := loginpolicy.PasswordConfig{MinLength: 24, MinComplexityClasses: 4}
	password, initialPassword, err := buildTenantAdminCredentials(config)
	if err != nil {
		t.Fatalf("buildTenantAdminCredentials() error = %v", err)
	}
	if initialPassword == "" {
		t.Fatal("initialPassword is empty")
	}
	if len([]rune(initialPassword)) != int(config.MinLength) {
		t.Fatalf("generated password length = %d, want %d", len([]rune(initialPassword)), config.MinLength)
	}
	if err = passwordPolicy.ValidateComplexity(initialPassword, config); err != nil {
		t.Fatalf("generated password does not satisfy policy: %v", err)
	}
	if err = crypto.Verify(initialPassword, password); err != nil {
		t.Fatalf("generated password does not match hash: %v", err)
	}
}
