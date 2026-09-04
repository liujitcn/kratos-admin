package codegen

import "testing"

func TestFrontendPackageNameForBusinessModule(t *testing.T) {
	tests := []struct {
		name   string
		module string
		want   string
	}{
		{name: "system module", module: "system", want: "@liujitcn/kratos-admin-system"},
		{name: "generated business module", module: "app", want: "@app/admin-module"},
		{name: "custom business module", module: "shop", want: "@shop/admin-module"},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			if got := frontendPackageNameForBusinessModule(testCase.module); got != testCase.want {
				t.Fatalf("frontendPackageNameForBusinessModule(%q) = %q, want %q", testCase.module, got, testCase.want)
			}
		})
	}
}

func TestProtoTargetForBusinessModuleUsesFrontendPackageName(t *testing.T) {
	appTarget, ok := ProtoTargetForBusinessModule("app")
	if !ok {
		t.Fatal("ProtoTargetForBusinessModule(app) returned false")
	}
	if appTarget.FrontendPackageName != "@app/admin-module" {
		t.Fatalf("app frontend package = %q, want %q", appTarget.FrontendPackageName, "@app/admin-module")
	}

	systemTarget, ok := ProtoTargetForBusinessModule("system")
	if !ok {
		t.Fatal("ProtoTargetForBusinessModule(system) returned false")
	}
	if systemTarget.FrontendPackageName != "@liujitcn/kratos-admin-system" {
		t.Fatalf("system frontend package = %q, want %q", systemTarget.FrontendPackageName, "@liujitcn/kratos-admin-system")
	}
}

func TestFrontendImportPathsUseGeneratedModulePackage(t *testing.T) {
	protoPath := "backend/api/proto/app/admin/v1/test_item.proto"
	if got := frontendRPCImportPath(protoPath); got != "@app/admin-module/rpc/app/admin/v1/test_item" {
		t.Fatalf("frontend RPC import = %q, want %q", got, "@app/admin-module/rpc/app/admin/v1/test_item")
	}

	method := &Proto{ProtoFilePath: protoPath, TargetEntityName: "TestItem"}
	if got := frontendAPIImportPathForMethod(method); got != "@app/admin-module/api/app/test_item" {
		t.Fatalf("frontend API import = %q, want %q", got, "@app/admin-module/api/app/test_item")
	}
}
