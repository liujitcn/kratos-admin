package script

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

// TestRegistryRunDependencyOrder 验证启动脚本按依赖顺序执行。
func TestRegistryRunDependencyOrder(t *testing.T) {
	calls := make([]string, 0)
	registry := NewRegistry()
	var err error
	err = registry.Register(
		Script{
			Name:         "schema",
			Dependencies: []string{"database"},
			Run: func(context.Context) error {
				calls = append(calls, "schema")
				return nil
			},
		},
		Script{
			Name: "database",
			Run: func(context.Context) error {
				calls = append(calls, "database")
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("注册启动脚本失败: %v", err)
	}
	err = registry.Run(context.Background())
	if err != nil {
		t.Fatalf("执行启动脚本失败: %v", err)
	}
	want := []string{"database", "schema"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("脚本执行顺序错误: got %v want %v", calls, want)
	}
}

// TestRegistryRunCycle 验证循环依赖在执行前被拒绝。
func TestRegistryRunCycle(t *testing.T) {
	registry := NewRegistry()
	var err error
	err = registry.Register(
		Script{Name: "first", Dependencies: []string{"second"}, Run: func(context.Context) error { return nil }},
		Script{Name: "second", Dependencies: []string{"first"}, Run: func(context.Context) error { return nil }},
	)
	if err != nil {
		t.Fatalf("注册启动脚本失败: %v", err)
	}
	err = registry.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "循环依赖") {
		t.Fatalf("循环依赖未被拒绝: %v", err)
	}
}
