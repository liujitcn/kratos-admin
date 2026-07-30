package startup

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// TestManagerStartRollback 验证启动失败时只逆序清理已启动钩子。
func TestManagerStartRollback(t *testing.T) {
	calls := make([]string, 0)
	manager := NewManager()
	var err error
	err = manager.Register(
		Hook{
			Name: "first",
			Start: func(context.Context) error {
				calls = append(calls, "start:first")
				return nil
			},
			Stop: func(context.Context) error {
				calls = append(calls, "stop:first")
				return nil
			},
		},
		Hook{
			Name: "second",
			Start: func(context.Context) error {
				calls = append(calls, "start:second")
				return errors.New("failed")
			},
		},
	)
	if err != nil {
		t.Fatalf("注册启动钩子失败: %v", err)
	}

	err = manager.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "second") {
		t.Fatalf("启动错误不符合预期: %v", err)
	}
	want := []string{"start:first", "start:second", "stop:first"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("启动钩子调用顺序错误: got %v want %v", calls, want)
	}

	err = manager.Stop(context.Background())
	if err != nil {
		t.Fatalf("重复清理启动钩子失败: %v", err)
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("重复停止不应再次执行钩子: got %v want %v", calls, want)
	}
}
