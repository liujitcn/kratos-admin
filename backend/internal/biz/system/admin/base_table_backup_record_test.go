package biz

import (
	"testing"
	"time"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// TestToBaseTableBackupRecordHidesUnverifiedTime 验证未完成校验的备份记录不返回校验时间。
func TestToBaseTableBackupRecordHidesUnverifiedTime(t *testing.T) {
	item := &models.BaseTableBackupRecord{Status: int32(adminv1.BaseTableBackupRecordStatus_BASE_TABLE_BACKUP_RECORD_STATUS_FAILED), VerifiedAt: time.Unix(0, 0).UTC()}
	result := toBaseTableBackupRecord(item)
	if result.VerifiedAt != "" {
		t.Fatalf("unverified backup should not expose verified_at, got %q", result.VerifiedAt)
	}

	item.Status = int32(adminv1.BaseTableBackupRecordStatus_BASE_TABLE_BACKUP_RECORD_STATUS_SUCCESS)
	item.VerifiedAt = time.Date(2026, time.September, 4, 19, 0, 0, 0, time.UTC)
	result = toBaseTableBackupRecord(item)
	if result.VerifiedAt != "2026-09-04T19:00:00Z" {
		t.Fatalf("successful backup should expose verified_at, got %q", result.VerifiedAt)
	}
}
