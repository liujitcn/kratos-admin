package logging

import (
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

// TestExtractResponseWithStructMapValue 确认 google.protobuf.Struct 动态 map 字段不会触发反射 panic。
func TestExtractResponseWithStructMapValue(t *testing.T) {
	config, err := structpb.NewStruct(map[string]interface{}{
		"enabled": true,
		"options": map[string]interface{}{
			"threshold": 0.9,
		},
	})
	if err != nil {
		t.Fatalf("创建 Struct 响应失败: %v", err)
	}

	response := extractResponse(config)
	var body map[string]interface{}
	if err = json.Unmarshal([]byte(response), &body); err != nil {
		t.Fatalf("响应日志不是合法 JSON: %v", err)
	}
	if body["enabled"] != true {
		t.Fatalf("响应日志内容不符合预期: %s", response)
	}
}
