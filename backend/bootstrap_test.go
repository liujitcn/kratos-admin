package backend_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestExternalHostWire 验证真实宿主 ProviderSet 在 Admin 路径之外生成代码后可以编译。
func TestExternalHostWire(t *testing.T) {
	wirePath, err := exec.LookPath("wire")
	if err != nil {
		t.Skip("未安装 wire，请安装 Wire 并将 Go 工具目录加入 PATH")
	}
	var backendDir string
	backendDir, err = os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	hostDir := t.TempDir()
	for _, name := range []string{"wire.go", "providers.go"} {
		var content []byte
		content, err = os.ReadFile(filepath.Join(backendDir, "internal", "cmd", "server", name))
		if err != nil {
			t.Fatal(err)
		}
		err = os.WriteFile(filepath.Join(hostDir, name), content, 0600)
		if err != nil {
			t.Fatal(err)
		}
	}
	var moduleFile []byte
	moduleFile, err = os.ReadFile(filepath.Join(backendDir, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	hostModule := strings.Replace(string(moduleFile), "module github.com/liujitcn/kratos-admin/backend", "module example.com/admin-host", 1)
	hostModule += "\nrequire github.com/liujitcn/kratos-admin/backend v0.0.40\n"
	err = os.WriteFile(filepath.Join(hostDir, "go.mod"), []byte(hostModule), 0600)
	if err != nil {
		t.Fatal(err)
	}
	workspacePath := filepath.Join(hostDir, "go.work")
	workspace := fmt.Sprintf("go 1.27.0\n\nuse (\n%q\n%q\n%q\n%q\n)\n", backendDir, filepath.Join(backendDir, "api"), filepath.Join(backendDir, "client"), hostDir)
	err = os.WriteFile(workspacePath, []byte(workspace), 0600)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOWORK", workspacePath)
	command := exec.Command(wirePath, ".")
	command.Dir = hostDir
	var output []byte
	output, err = command.CombinedOutput()
	if err != nil {
		t.Fatalf("外部宿主 Wire 生成失败: %v\n%s", err, output)
	}
	var generated []byte
	generated, err = os.ReadFile(filepath.Join(hostDir, "wire_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(generated), "github.com/liujitcn/kratos-admin/backend/internal/") {
		t.Fatal("外部宿主 Wire 生成代码泄露了 Admin internal 包导入")
	}
	command = exec.Command("go", "test", "-run", "^$", ".")
	command.Dir = hostDir
	output, err = command.CombinedOutput()
	if err != nil {
		t.Fatalf("外部宿主编译失败: %v\n%s", err, output)
	}
}
