package static

import (
	"io/fs"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// TestSPAHandlerFallback 验证静态文件直出且前端路由回退到入口页面。
func TestSPAHandlerFallback(t *testing.T) {
	webFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("index")},
		"app.js":     &fstest.MapFile{Data: []byte("asset")},
	}
	var filesystem fs.FS = webFS
	handler := newHandler(filesystem, "/admin", true)

	assetRequest := httptest.NewRequest(stdhttp.MethodGet, "/admin/app.js", nil)
	assetResponse := httptest.NewRecorder()
	handler.ServeHTTP(assetResponse, assetRequest)
	if assetResponse.Code != stdhttp.StatusOK || assetResponse.Body.String() != "asset" {
		t.Fatalf("静态文件响应错误: code=%d body=%q", assetResponse.Code, assetResponse.Body.String())
	}

	fallbackRequest := httptest.NewRequest(stdhttp.MethodGet, "/admin/orders/1", nil)
	fallbackResponse := httptest.NewRecorder()
	handler.ServeHTTP(fallbackResponse, fallbackRequest)
	if fallbackResponse.Code != stdhttp.StatusOK || fallbackResponse.Body.String() != "index" {
		t.Fatalf("SPA 回退响应错误: code=%d body=%q", fallbackResponse.Code, fallbackResponse.Body.String())
	}
}
