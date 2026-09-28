package biz

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	commonv1 "github.com/liujitcn/kratos-core/api/gen/go/common/v1"
	"github.com/liujitcn/kratos-core/biz"
)

// TestValidateEnabledProviderRejectsIncompleteSenderConfig 验证启用 Provider 前会校验发送器配置。
func TestValidateEnabledProviderRejectsIncompleteSenderConfig(t *testing.T) {
	providerCase := &BaseMessageProviderCase{BaseCase: &biz.BaseCase{}}
	provider := &models.BaseMessageProvider{ID: 1, Provider: "email", Status: int32(commonv1.Status_STATUS_ENABLE), Config: "{}"}
	err := providerCase.validateEnabledProvider(provider)
	if err == nil {
		t.Fatal("enabled email provider without SMTP configuration should be rejected")
	}
	provider.Status = int32(commonv1.Status_STATUS_DISABLE)
	err = providerCase.validateEnabledProvider(provider)
	if err != nil {
		t.Fatalf("disabled provider should allow incomplete configuration: %v", err)
	}
}
