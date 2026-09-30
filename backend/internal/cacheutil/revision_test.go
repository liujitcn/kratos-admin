package cacheutil

import (
	"testing"

	"github.com/liujitcn/kratos-kit/cache/memory"
)

// TestReadRevisionUsesInitialValue 验证缺失版本键使用初始值，并可由原子递增更新。
func TestReadRevisionUsesInitialValue(t *testing.T) {
	store, cleanup, err := memory.NewMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	revision, enabled := ReadRevision(store, "app:test:revision")
	if !enabled || revision != "0" {
		t.Fatalf("ReadRevision() = %q, %v; want %q, true", revision, enabled, "0")
	}
	if err = IncrementRevision(store, "app:test:revision"); err != nil {
		t.Fatal(err)
	}
	revision, enabled = ReadRevision(store, "app:test:revision")
	if !enabled || revision != "1" {
		t.Fatalf("ReadRevision() after increment = %q, %v; want %q, true", revision, enabled, "1")
	}
}
