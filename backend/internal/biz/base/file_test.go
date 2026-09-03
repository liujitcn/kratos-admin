package biz

import "testing"

// TestTenantFilePath 验证租户文件路径隔离。
func TestTenantFilePath(t *testing.T) {
	path, err := tenantFilePath(7, "/base/avatar/2026/01/01/a.png")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/tenant/7/base/avatar/2026/01/01/a.png" {
		t.Fatalf("unexpected tenant path %q", path)
	}
	if _, err = tenantFilePath(7, "/tenant/8/base/avatar/a.png"); err == nil {
		t.Fatal("expected cross-tenant path to be rejected")
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
