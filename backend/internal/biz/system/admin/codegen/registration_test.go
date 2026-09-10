package codegen

import (
	"strings"
	"testing"
)

// TestRegistrationReusesImports 验证已有协议别名和默认服务包名被复用，重复生成不追加注册。
func TestRegistrationReusesImports(t *testing.T) {
	target, _ := ProtoTargetForBusinessModule("system")
	source := `package admin
import (
 adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
 "github.com/liujitcn/kratos-admin/backend/internal/service/system/admin/v1"
)
type Services struct {}
func (s Services) RegisterGRPC(srv any) {}
func (s Services) RegisterHTTP(srv any) {}
func (s Services) RegisterMCP(mcpSrv any) {}
`
	result := appendProtoServiceRegistration(source, target, "TenantProject")
	for _, expected := range []string{"*admin.TenantProjectService", "adminv1.RegisterTenantProjectServiceServer", "adminv1.RegisterTenantProjectServiceHTTPServer", "adminv1.RegisterTenantProjectServiceMCPTools"} {
		if !strings.Contains(result, expected) {
			t.Fatalf("缺少正确的包引用 %s: %s", expected, result)
		}
	}
	if strings.Contains(result, "systemadmin") {
		t.Fatal("生成器使用了不存在的包别名")
	}
	if next := appendProtoServiceRegistration(result, target, "TenantProject"); next != result {
		t.Fatal("重复生成不应改变注册文件")
	}
}

// TestProviderRegistrationAppendsLast 验证末尾追加构造函数不会因逗号位置错误被静默跳过。
func TestProviderRegistrationAppendsLast(t *testing.T) {
	for _, suffix := range []string{"Case", "Service"} {
		source := "package admin\nimport \"github.com/google/wire\"\nvar ProviderSet = wire.NewSet(\n\tNewBaseTenant" + suffix + ",\n)\n"
		provider := "NewTenantProject" + suffix
		result := appendProviderSetItems(source, provider)
		if !strings.Contains(result, provider+",") {
			t.Fatalf("缺少依赖注册 %s: %s", provider, result)
		}
		if next := appendProviderSetItems(result, provider); next != result {
			t.Fatal("重复生成不应改变依赖注册")
		}
	}
}

// TestCodeGenProgressIncludesModuleWire 验证生成进度按模块装配、独立入口的顺序展示 Wire 步骤。
func TestCodeGenProgressIncludesModuleWire(t *testing.T) {
	steps := BuildProgressSteps(nil, false, true, LocaleState{})
	var commands []string
	for _, step := range steps {
		commands = append(commands, step.GetId())
	}
	expected := "command:gorm-gen,command:api,command:openapi,command:ts,command:public-wire,command:wire,command:fmt"
	if actual := strings.Join(commands, ","); actual != expected {
		t.Fatalf("生成命令顺序 = %s，期望 %s", actual, expected)
	}
}
