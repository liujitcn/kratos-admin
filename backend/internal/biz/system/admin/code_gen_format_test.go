package biz

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestFormatCodeGenChangesScope 验证 make fmt 收到的清单只包含本次改写文件，且支持空格路径。
func TestFormatCodeGenChangesScope(t *testing.T) {
	makefile, err := os.ReadFile(filepath.Join("../../../../", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Chdir(root)
	backend := filepath.Join(root, "backend")
	bin := filepath.Join(root, "bin")
	for _, path := range []string{backend, bin} {
		if err = os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{
		"backend/Makefile":         string(makefile),
		"backend/changed file.go":  "package sample\n",
		"backend/untouched.go":     "package untouched\n",
		"bin/goimports":            "#!/bin/sh\nshift\nfor file do printf '// formatted\\n' >> \"$file\"; done\n",
		"bin/normalize-go-imports": "#!/bin/sh\nexit 0\n",
	}
	for path, content := range files {
		if err = os.WriteFile(filepath.Join(root, path), []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GO_BIN_DIR", bin)
	var before map[string]codeGenRestoreWorkspaceFile
	before, err = captureCodeGenWorkspaceSnapshot(nil)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(backend, "changed file.go"), []byte("package changed\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, err = formatCodeGenChanges(context.Background(), backend, before)
	if err != nil {
		t.Fatal(err)
	}
	for path, expected := range map[string]string{"changed file.go": "package changed\n// formatted\n// formatted\n", "untouched.go": "package untouched\n"} {
		var content []byte
		content, err = os.ReadFile(filepath.Join(backend, path))
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != expected {
			t.Fatalf("文件 %s 超出格式化范围或未格式化: %s", path, content)
		}
	}
}
