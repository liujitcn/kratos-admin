package module

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
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

// TestStaticFileAccessMode 验证公开文件免认证、授权文件必须携带令牌。
func TestStaticFileAccessMode(t *testing.T) {
	fileHandler := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("image-content"))
	})
	findFile := func(_ context.Context, objectPath string) (*models.BaseFile, error) {
		if objectPath == "public/logo.png" {
			return &models.BaseFile{AccessMode: int32(basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_PUBLIC)}, nil
		}
		return &models.BaseFile{AccessMode: int32(basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_AUTHORIZED)}, nil
	}
	handler := newStaticFileAccessHandler(fileHandler, findFile, nil, nil)

	request := httptest.NewRequest(http.MethodGet, "/data/public/logo.png", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "image-content" {
		t.Fatalf("公开文件响应 = (%d, %q), want (%d, %q)", recorder.Code, recorder.Body.String(), http.StatusOK, "image-content")
	}

	request = httptest.NewRequest(http.MethodGet, "/data/private/avatar.png", nil)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("授权文件无令牌状态码 = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
}
