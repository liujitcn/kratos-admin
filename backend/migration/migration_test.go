package migration

import (
	"encoding/json"
	"io/fs"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TestBiSourceMenuPermissions 验证数据源页面只授予读取权限，写操作由对应按钮单独授权。
func TestBiSourceMenuPermissions(t *testing.T) {
	pattern := regexp.MustCompile(`(?m)VALUES \((91050[0-4]00),[^\n]*'(\[[^'\n]*\])'`)
	for _, driver := range []string{"mysql", "postgres"} {
		seed, err := fs.ReadFile(Assets(), "v0.0.1/"+driver+"/default_data.up.sql")
		if err != nil {
			t.Fatal(err)
		}
		menus := make(map[string][]string)
		for _, match := range pattern.FindAllStringSubmatch(string(seed), -1) {
			var operations []string
			if err = json.Unmarshal([]byte(match[2]), &operations); err != nil {
				t.Fatal(err)
			}
			menus[match[1]] = operations
		}
		if len(menus) != 5 {
			t.Fatalf("%s 数据源页面及四个操作按钮未完整初始化", driver)
		}
		for _, operation := range menus["91050000"] {
			method := strings.TrimPrefix(operation, "/bi.admin.v1.BiSourceService/")
			if method != "PageBiSource" && method != "GetBiSource" {
				t.Errorf("%s 仅授予页面权限时不应允许 %s", driver, operation)
			}
		}
		for menuID, methods := range map[string][]string{
			"91050000": {"PageBiSource", "GetBiSource"},
			"91050100": {"CreateBiSource", "TestBiSource", "ListBiSourceTables", "SaveBiSourceModel"},
			"91050200": {"UpdateBiSource", "TestBiSource", "ListBiSourceTables", "SaveBiSourceModel"},
			"91050300": {"DeleteBiSource"},
			"91050400": {"SetBiSourceStatus"},
		} {
			for _, method := range methods {
				if !slices.Contains(menus[menuID], "/bi.admin.v1.BiSourceService/"+method) {
					t.Errorf("%s 菜单 %s 缺少 %s 授权", driver, menuID, method)
				}
			}
		}
	}
}
