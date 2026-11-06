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

func TestRegistryRegisterListAndExecute(t *testing.T) {
	registry := NewRegistry()
	adapter := testSourceAdapter{}
	if err := registry.Register(adapter); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	if err := registry.Register(adapter); err == nil {
		t.Fatal("expected duplicate registration error")
	}
	descriptors, err := registry.List(context.Background(), Scope{})
	if err != nil || len(descriptors) != 1 {
		t.Fatalf("list descriptors: len=%d err=%v", len(descriptors), err)
	}
	result, err := registry.Execute(context.Background(), Scope{}, QuerySpec{SourceCode: adapter.Code()})
	if err != nil || len(result.Rows) != 1 {
		t.Fatalf("execute adapter: rows=%d err=%v", len(result.Rows), err)
	}
}
