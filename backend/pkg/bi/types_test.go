package bi

import (
	"context"
	"testing"
)

type testSourceAdapter struct{}

func (testSourceAdapter) Code() string { return "test.source" }

func (testSourceAdapter) Descriptor(context.Context, Scope) (SourceDescriptor, error) {
	return SourceDescriptor{Code: "test.source", Name: "测试数据源"}, nil
}

func (testSourceAdapter) Execute(context.Context, Scope, QuerySpec) (QueryResult, error) {
	return QueryResult{Rows: [][]string{{"ok"}}}, nil
}

// TestRegistryRegisterListAndExecute 验证数据源注册、重复检测和查询执行。
func TestRegistryRegisterListAndExecute(t *testing.T) {
	registry := NewRegistry()
	adapter := testSourceAdapter{}
	err := registry.Register(adapter)
	if err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	err = registry.Register(adapter)
	if err == nil {
		t.Fatal("expected duplicate registration error")
	}
	var descriptors []SourceDescriptor
	descriptors, err = registry.List(t.Context(), Scope{})
	if err != nil || len(descriptors) != 1 {
		t.Fatalf("list descriptors: len=%d err=%v", len(descriptors), err)
	}
	var result QueryResult
	result, err = registry.Execute(t.Context(), Scope{}, QuerySpec{SourceCode: adapter.Code()})
	if err != nil || len(result.Rows) != 1 {
		t.Fatalf("execute adapter: rows=%d err=%v", len(result.Rows), err)
	}
}
