package biz

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/i18n"
	corei18n "github.com/liujitcn/kratos-core/resource/i18n"
)

// TestOpsMonitoringFailureMessagesHaveAllLocales 验证监控错误文案覆盖全部内置语言。
func TestOpsMonitoringFailureMessagesHaveAllLocales(t *testing.T) {
	catalog, err := corei18n.NewI18n("ops-monitoring-test", i18n.Assets())
	if err != nil {
		t.Fatal(err)
	}
	locales := []string{"zh-CN", "en-US", "ja-JP", "zh-TW"}
	keys := []string{
		"connection_check_failed",
		"connection_close_failed",
		"connection_pool_unavailable",
		"dispatch_status_unavailable",
		"redis_configuration_invalid",
	}
	for _, locale := range locales {
		for _, key := range keys {
			if got := monitoringText(catalog, locale, key); got == "system.admin.ops_monitoring."+key {
				t.Errorf("locale %s missing monitoring message %s", locale, key)
			}
		}
	}
}
