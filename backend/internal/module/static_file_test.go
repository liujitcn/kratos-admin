package module

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestStaticFileDirectoryListingBlocked 验证精确文件可访问且目录索引被拒绝。
func TestStaticFileDirectoryListingBlocked(t *testing.T) {
	rootDirectory := t.TempDir()
	filePath := filepath.Join(rootDirectory, "1", "image", "avatar.png")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("创建测试目录失败: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("image-content"), 0o644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	fileHandler := http.StripPrefix("/data/", http.FileServer(http.Dir(rootDirectory)))
	handler := blockStaticDirectoryListing(fileHandler)
	request := httptest.NewRequest(http.MethodGet, "/data/1/image/avatar.png", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "image-content" {
		t.Fatalf("精确静态文件响应 = (%d, %q), want (%d, %q)", recorder.Code, recorder.Body.String(), http.StatusOK, "image-content")
	}

	request = httptest.NewRequest(http.MethodGet, "/data/1/image/", nil)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("目录静态请求状态码 = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}
