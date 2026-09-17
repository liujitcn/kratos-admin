package biz

import (
	"testing"

	_mapper "github.com/liujitcn/go-utils/mapper"
	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
)

// TestBuildBaseMenuOptionIncludesButtonPermissions 验证角色权限树包含按钮权限且不包含外链。
func TestBuildBaseMenuOptionIncludesButtonPermissions(t *testing.T) {
	baseMenuCase := &BaseMenuCase{
		routerMapper: _mapper.NewCopierMapper[adminv1.RouteItem, models.BaseMenu](),
	}
	options := baseMenuCase.buildBaseMenuOption(
		[]*models.BaseMenu{
			{ID: 100, ParentID: 0, Type: int32(adminv1.BaseMenuType_BASE_MENU_TYPE_FOLDER)},
			{ID: 101, ParentID: 100, Type: int32(adminv1.BaseMenuType_BASE_MENU_TYPE_MENU)},
			{ID: 102, ParentID: 101, Type: int32(adminv1.BaseMenuType_BASE_MENU_TYPE_BUTTON), Path: "base:role:menus"},
			{ID: 103, ParentID: 100, Type: int32(adminv1.BaseMenuType_BASE_MENU_TYPE_EXT_LINK), Path: "https://example.com"},
		},
		0,
		false,
		map[int64]struct{}{},
		map[int64]string{100: "系统管理", 101: "角色管理", 102: "分配角色权限"},
	)

	if len(options) != 1 || options[0].Value != 100 {
		t.Fatalf("根权限树错误: %+v", options)
	}
	if len(options[0].Children) != 1 || options[0].Children[0].Value != 101 {
		t.Fatalf("页面权限树错误: %+v", options[0].Children)
	}
	children := options[0].Children[0].Children
	if len(children) != 1 || children[0].Value != 102 || children[0].Label != "分配角色权限" {
		t.Fatalf("按钮权限未出现在权限树中: %+v", children)
	}
}
