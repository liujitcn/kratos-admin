package base

import (
	"os"
	"strings"
	"testing"
	"time"
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
	objectPath := info.Path + "/" + info.Name
	if !strings.HasPrefix(objectPath, expectedPath+"/") {
		t.Fatalf("上传对象路径 = %q, want prefix %q", objectPath, expectedPath+"/")
	}
	if strings.Contains(objectPath, "kratos") || strings.Contains(objectPath, "tenant") {
		t.Fatalf("上传对象路径包含已移除的目录层: %q", info.Path)
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
