package module

import (
	"os"
	"strings"
	"testing"
)

// TestCodegenManagerInjected 验证独立 Wire 入口不会分别创建任务管理器，必须接收宿主共享实例。
func TestCodegenManagerInjected(t *testing.T) {
	content, err := os.ReadFile("wire_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "codegen.NewManager()") {
		t.Fatal("服务与 SSE 的独立注入入口创建了不同 Manager，SSE 无法识别服务创建的任务")
	}
	for _, name := range []string{"BuildModules", "BuildStreams"} {
		start := strings.Index(string(content), "func "+name+"(")
		if start < 0 {
			t.Fatalf("缺失 %s", name)
		}
		signature := strings.SplitN(string(content)[start:], ") (", 2)[0]
		if !strings.Contains(signature, "*codegen.Manager") {
			t.Errorf("%s 未接收共享任务管理器", name)
		}
	}
}
