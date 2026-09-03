package task

import (
	"testing"
)

// TestNewTaskRegistersCurrentTargets 验证任务注册使用当前调用目标。
func TestNewTaskRegistersCurrentTargets(t *testing.T) {
	tasks := NewTask(nil, nil, nil, nil, nil)
	if len(tasks) != 5 {
		t.Fatalf("task count = %d, want 5", len(tasks))
	}

	wantNames := map[string]bool{
		"system.admin.BaseI18n":            false,
		"system.admin.BaseMessageDispatch": false,
		"system.admin.BaseTableArchive":    false,
		"system.admin.BaseLogFallback":     false,
		"system.admin.BaseTableBackup":     false,
	}
	for _, item := range tasks {
		if item == nil {
			continue
		}
		if _, ok := wantNames[item.Name]; ok {
			wantNames[item.Name] = true
		}
	}
	for name, found := range wantNames {
		if !found {
			t.Errorf("missing registered task %q", name)
		}
	}
}
