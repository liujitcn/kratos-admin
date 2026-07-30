package openapi

import (
	"strings"
	"testing"
)

// TestRegistryOperationOwnership 验证接口归属查询和跨文档冲突检查。
func TestRegistryOperationOwnership(t *testing.T) {
	first := Document{
		Key:  "first",
		Name: "First API",
		Data: []byte("openapi: 3.0.3\npaths:\n  /items:\n    get:\n      responses: {}\n"),
	}
	registry, err := NewRegistry(first)
	if err != nil {
		t.Fatalf("创建 OpenAPI 注册表失败: %v", err)
	}

	document, exists := registry.DocumentByOperation("/items", "GET")
	if !exists || document.Key != first.Key {
		t.Fatalf("接口归属错误: exists=%v document=%+v", exists, document)
	}

	err = registry.Register(Document{
		Key:  "second",
		Name: "Second API",
		Data: []byte("openapi: 3.0.3\npaths:\n  /items:\n    get:\n      responses: {}\n"),
	})
	if err == nil || !strings.Contains(err.Error(), "重复接口") {
		t.Fatalf("重复接口未被拒绝: %v", err)
	}
	if len(registry.Documents()) != 1 {
		t.Fatalf("失败注册不应改变文档快照: %d", len(registry.Documents()))
	}
}
