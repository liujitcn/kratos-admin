package admin

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/pkg/bi"
	"github.com/liujitcn/kratos-kit/auth/data"
	"github.com/liujitcn/kratos-kit/database/gorm"
)

func TestProjectIDForAuthRequiresTenantProjectScope(t *testing.T) {
	tests := []struct {
		name      string
		authInfo  *data.UserTokenPayload
		wantID    int64
		wantError bool
	}{
		{name: "default tenant", authInfo: &data.UserTokenPayload{TenantCode: gorm.DefaultTenantCode}, wantID: 0},
		{name: "missing tenant scope", authInfo: &data.UserTokenPayload{TenantCode: "tenant-a", TenantId: 1}, wantError: true},
		{name: "single project", authInfo: &data.UserTokenPayload{TenantCode: "tenant-a", TenantId: 1, TenantProjects: []data.TenantProjectScope{{TenantId: 1, ProjectId: []int64{42}}}}, wantID: 42},
		{name: "all projects", authInfo: &data.UserTokenPayload{TenantCode: "tenant-a", TenantId: 1, TenantProjects: []data.TenantProjectScope{{TenantId: 1, ProjectId: []int64{0}}}}, wantID: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			projectID, err := projectIDForAuth(test.authInfo)
			if (err != nil) != test.wantError {
				t.Fatalf("projectIDForAuth() error = %v, wantError %v", err, test.wantError)
			}
			if err == nil && projectID != test.wantID {
				t.Fatalf("projectIDForAuth() = %d, want %d", projectID, test.wantID)
			}
		})
	}
}

func TestApplyResultOrdersSortsNumericValuesAndLimits(t *testing.T) {
	result := applyResultOrders(bi.QueryResult{
		Columns: []bi.ResultColumn{{Code: "amount"}},
		Rows:    [][]string{{"2"}, {"10"}, {"5"}},
	}, []bi.OrderSpec{{Field: "amount", Direction: "desc"}}, 2)
	if len(result.Rows) != 2 || result.Rows[0][0] != "10" || result.Rows[1][0] != "5" || !result.Truncated {
		t.Fatalf("unexpected ordered result: %#v", result)
	}
}

// TestUnsortedResultLimit 验证未排序的外部 Adapter 结果也不能突破请求行数。
func TestUnsortedResultLimit(t *testing.T) {
	result := applyResultOrders(bi.QueryResult{Rows: [][]string{{"2"}, {"10"}, {"5"}}}, nil, 2)
	if len(result.Rows) != 2 || result.Rows[0][0] != "2" || !result.Truncated {
		t.Fatalf("未排序结果未按请求截断: %#v", result)
	}
}
