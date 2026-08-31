package biz

import "testing"

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

func TestValidateFileContentExtension(t *testing.T) {
	if err := validateFileContent("image.png", []byte("content")); err != nil {
		t.Fatal(err)
	}
	if err := validateFileContent("script.exe", []byte("content")); err == nil {
		t.Fatal("expected executable extension to be rejected")
	}
}
