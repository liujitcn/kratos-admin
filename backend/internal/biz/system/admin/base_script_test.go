package biz

import (
	"strings"
	"testing"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
)

// TestValidateScriptRunRequest 验证脚本试运行只允许引用已保存脚本。
func TestValidateScriptRunRequest(t *testing.T) {
	tests := []struct {
		name string
		req  *adminv1.TestRunBaseScriptRequest
		want string
	}{
		{name: "draft", req: &adminv1.TestRunBaseScriptRequest{Source: "return 1"}, want: "必须引用"},
		{name: "saved with source", req: &adminv1.TestRunBaseScriptRequest{Id: 1, Source: "return 1"}, want: "不能提交"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateScriptRunRequest(test.req)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("请求未被拒绝: %v", err)
			}
		})
	}
	if err := validateScriptRunRequest(&adminv1.TestRunBaseScriptRequest{Id: 1}); err != nil {
		t.Fatalf("已保存脚本被错误拒绝: %v", err)
	}
}
