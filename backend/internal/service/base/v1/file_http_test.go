package base

import (
	"os"
	"strings"
	"testing"
	"time"

	basev1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/base/v1"
)

// TestConvertUploadFileInfo 验证上传对象路径按业务、分类和日期分层。
func TestConvertUploadFileInfo(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "upload-*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.WriteString("image-content"); err != nil {
		t.Fatal(err)
	}
	if _, err = file.Seek(0, 0); err != nil {
		t.Fatal(err)
	}

	info, err := convertUploadFileInfo(file, "message", "image/jpeg", "image.jpg")
	if err != nil {
		t.Fatal(err)
	}
	expectedPath := "message/images/" + time.Now().Format("2006/01/02")
	if info.Path != expectedPath {
		t.Fatalf("上传对象目录 = %q, want %q", info.Path, expectedPath)
	}
	if info.Name != "image.jpg" {
		t.Fatalf("原始文件名 = %q, want %q", info.Name, "image.jpg")
	}
	if strings.Contains(info.Path, "kratos") || strings.Contains(info.Path, "tenant") {
		t.Fatalf("上传对象路径包含已移除的目录层: %q", info.Path)
	}
}

// TestParseUploadAccessMode 验证上传访问方式的默认值、枚举名称和非法值处理。
func TestParseUploadAccessMode(t *testing.T) {
	tests := []struct {
		value string
		want  basev1.BaseFileAccessMode
		valid bool
	}{
		{value: "", want: basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_UNSPECIFIED, valid: true},
		{value: "1", want: basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_PUBLIC, valid: true},
		{value: "PUBLIC", want: basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_PUBLIC, valid: true},
		{value: "2", want: basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_AUTHORIZED, valid: true},
		{value: "AUTHORIZED", want: basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_AUTHORIZED, valid: true},
		{value: "3", want: basev1.BaseFileAccessMode_BASE_FILE_ACCESS_MODE_UNSPECIFIED, valid: false},
	}
	for _, test := range tests {
		got, err := parseUploadAccessMode(test.value)
		if test.valid && err != nil {
			t.Errorf("访问方式 %q 返回错误: %v", test.value, err)
		}
		if !test.valid && err == nil {
			t.Errorf("访问方式 %q 应返回错误", test.value)
		}
		if got != test.want {
			t.Errorf("访问方式 %q = %v, want %v", test.value, got, test.want)
		}
	}
}

// TestNormalizeUploadBusinessType 验证业务类型不能注入多级或逃逸路径。
func TestNormalizeUploadBusinessType(t *testing.T) {
	if businessType, err := normalizeUploadBusinessType(""); err != nil || businessType != "file" {
		t.Fatalf("默认业务类型 = %q, err = %v", businessType, err)
	}
	for _, value := range []string{"../message", "message/image", `message\\image`, "message\x00image"} {
		if _, err := normalizeUploadBusinessType(value); err == nil {
			t.Fatalf("业务类型 %q 应被拒绝", value)
		}
	}
}
