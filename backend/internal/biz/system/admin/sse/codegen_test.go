package sse

import (
	"testing"

	"github.com/liujitcn/kratos-admin/backend/internal/biz/system/admin/codegen"
	coresse "github.com/liujitcn/kratos-core/sse"
)

// TestCodegenSharedManagerOwnership 验证共享任务管理器可解析本人的任务，且仍拒绝其他用户。
func TestCodegenSharedManagerOwnership(t *testing.T) {
	manager := codegen.NewManager()
	task, created := manager.Create(42, nil, codegen.LocaleState{})
	if !created {
		t.Fatal("创建任务失败")
	}
	registry := coresse.NewRegistry()
	err := registry.Register(NewCodegen(manager))
	if err != nil {
		t.Fatal(err)
	}
	var stream string
	var found bool
	stream, found, err = registry.Resolve(codegen.SSEStreamCodeGen, task.TaskId, 42)
	if err != nil || !found || stream != codegen.StreamID(task.TaskId) {
		t.Fatalf("创建者不能订阅任务: %s %v", stream, err)
	}
	_, _, err = registry.Resolve(codegen.SSEStreamCodeGen, task.TaskId, 43)
	if err == nil {
		t.Fatal("其他用户不应订阅任务")
	}
}
