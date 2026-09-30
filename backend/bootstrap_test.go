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
	// 临时宿主由 go.work 共享本地模块，移除复制来的相对 kratos-kit replace 块。
	replaceStart := strings.Index(hostModule, "\nreplace (\n")
	if replaceStart >= 0 {
		replaceEnd := strings.Index(hostModule[replaceStart:], "\n)\n")
		if replaceEnd < 0 {
			t.Fatal("Admin go.mod 中的 replace 块格式无效")
		}
		hostModule = hostModule[:replaceStart] + hostModule[replaceStart+replaceEnd+len("\n)\n"):]
	}
	// 临时宿主通过工作区复用 backend 模块的本地替换，不能再解析一份相对替换路径。
	hostModule = strings.Replace(hostModule, "\nreplace github.com/liujitcn/kratos-core => ../../kratos-core\n", "\n", 1)
	hostModule = strings.Replace(hostModule, "\nreplace github.com/liujitcn/kratos-core/api => ../../kratos-core/api\n", "\n", 1)
	hostModule = strings.Replace(hostModule, "\nreplace github.com/liujitcn/kratos-kit/secretcrypto => ../../kratos-kit/secretcrypto\n", "\n", 1)
	hostModule = strings.Replace(hostModule, "\nreplace github.com/liujitcn/kratos-kit/cache => ../../kratos-kit/cache\n", "\n", 1)
	hostModule = strings.Replace(hostModule, "\nreplace github.com/liujitcn/kratos-kit/api => ../../kratos-kit/api\n", "\n", 1)
	hostModule = strings.Replace(hostModule, "\nreplace github.com/liujitcn/kratos-kit/cache => ../../kratos-kit/cache\n", "\n", 1)
	hostModule = strings.Replace(hostModule, "\nreplace github.com/liujitcn/kratos-kit/oauth => ../../kratos-kit/oauth\n", "\n", 1)
	hostModule = strings.Replace(hostModule, "\nreplace github.com/liujitcn/kratos-kit/redact => ../../kratos-kit/redact\n", "\n", 1)
	hostModule = strings.Replace(hostModule, "\nreplace github.com/liujitcn/kratos-admin/backend/api => ./api\n", "\n", 1)
	hostModule += "\nrequire github.com/liujitcn/kratos-admin/backend v0.0.40\n"
	err = os.WriteFile(filepath.Join(hostDir, "go.mod"), []byte(hostModule), 0600)
	if err != nil {
		t.Fatal(err)
	}
	workspacePath := filepath.Join(hostDir, "go.work")
	coreDir := filepath.Clean(filepath.Join(backendDir, "../../kratos-core"))
	kitDir := filepath.Clean(filepath.Join(backendDir, "../../kratos-kit"))
	workspaceModules := []string{backendDir, filepath.Join(backendDir, "api"), filepath.Join(backendDir, "client"), hostDir, coreDir}
	kitWorkspace, err := os.ReadFile(filepath.Join(kitDir, "go.work"))
	if err != nil {
		t.Fatal(err)
	}
	inKitUseBlock := false
	for _, line := range strings.Split(string(kitWorkspace), "\n") {
		line = strings.TrimSpace(line)
		if line == "use (" {
			inKitUseBlock = true
			continue
		}
		if !inKitUseBlock || line == "" {
			continue
		}
		if line == ")" {
			break
		}
		modulePath := strings.Trim(line, "\"")
		if modulePath == "." {
			modulePath = kitDir
		} else {
			modulePath = filepath.Join(kitDir, modulePath)
		}
		if _, err = os.Stat(filepath.Join(modulePath, "go.mod")); err != nil {
			t.Fatalf("Kit workspace 模块不存在 %s: %v", modulePath, err)
		}
		workspaceModules = append(workspaceModules, modulePath)
	}
	var workspaceBuilder strings.Builder
	workspaceBuilder.WriteString("go 1.27.0\n\nuse (\n")
	for _, modulePath := range workspaceModules {
		_, err = fmt.Fprintf(&workspaceBuilder, "%q\n", modulePath)
		if err != nil {
			t.Fatal(err)
		}
	}
	workspaceBuilder.WriteString(")\n")
	workspace := workspaceBuilder.String()
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
