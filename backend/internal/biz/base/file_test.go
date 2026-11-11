package biz

import "testing"

// TestMetadataStoreForObjectPathSeparatesAiFiles 验证 AI 对象路径只进入 AI 文件元数据表。
func TestMetadataStoreForObjectPathSeparatesAiFiles(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "ai", path: "ai/docs/2026/11/11/file.txt", want: "ai"},
		{name: "base", path: "avatar/images/2026/11/11/file.png", want: "base"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := metadataStoreForObjectPath(test.path); got != test.want {
				t.Fatalf("metadata store = %q, want %q", got, test.want)
			}
		})
	}
}
