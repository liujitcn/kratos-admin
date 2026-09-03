package biz

import "testing"

// TestObjectFilePath 验证浏览器路径和 OSS 对象路径的转换。
func TestObjectFilePath(t *testing.T) {
	path, err := objectFilePath("/image/images/2026/01/01/a.png")
	if err != nil {
		t.Fatal(err)
	}
	if path != "image/images/2026/01/01/a.png" {
		t.Fatalf("unexpected object path %q", path)
	}
	path, err = objectFilePath("/data/image/images/2026/01/01/a.png")
	if err != nil {
		t.Fatal(err)
	}
	if path != "image/images/2026/01/01/a.png" {
		t.Fatalf("unexpected public object path %q", path)
	}
	if _, err = objectFilePath("/image/../secret.txt"); err == nil {
		t.Fatal("expected path traversal to be rejected")
	}
}

// TestPublicFileURL 验证上传响应使用统一的数据访问路径。
func TestPublicFileURL(t *testing.T) {
	if path := publicFileURL("image/images/2026/01/01/a.png"); path != "/data/image/images/2026/01/01/a.png" {
		t.Fatalf("unexpected public file path %q", path)
	}
	if path := publicFileURL("https://cdn.example.com/a.png"); path != "https://cdn.example.com/a.png" {
		t.Fatalf("unexpected external file URL %q", path)
	}
}

// TestValidateFileContentExtension 验证危险扩展名和伪装内容会被拒绝。
func TestValidateFileContentExtension(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
	if err := validateFileContent("image.png", pngHeader); err != nil {
		t.Fatal(err)
	}
	if err := validateFileContent("script.exe", []byte("content")); err == nil {
		t.Fatal("expected executable extension to be rejected")
	}
	if err := validateFileContent("fake.jpg", []byte("%PDF-1.7")); err == nil {
		t.Fatal("expected mismatched content type to be rejected")
	}
}

// TestFileContentMatchesExtension 验证常用文档与媒体类型映射。
func TestFileContentMatchesExtension(t *testing.T) {
	tests := []struct {
		extension   string
		contentType string
		matched     bool
	}{
		{extension: ".jpg", contentType: "image/jpeg", matched: true},
		{extension: ".jpg", contentType: "application/pdf", matched: false},
		{extension: ".pdf", contentType: "application/pdf", matched: true},
		{extension: ".docx", contentType: "application/zip", matched: true},
		{extension: ".txt", contentType: "text/html; charset=utf-8", matched: false},
	}
	for _, test := range tests {
		if actual := fileContentMatchesExtension(test.extension, test.contentType); actual != test.matched {
			t.Fatalf("扩展名 %s 和类型 %s 匹配结果错误: %v", test.extension, test.contentType, actual)
		}
	}
}
