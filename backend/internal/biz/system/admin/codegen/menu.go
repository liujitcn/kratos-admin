package codegen

import (
	"encoding/json"
	"strings"

	_const "github.com/liujitcn/kratos-admin/backend/internal/const"
	"github.com/liujitcn/kratos-admin/backend/internal/data/gen/models"
	coreconst "github.com/liujitcn/kratos-core/const"
)

// MenuSpecs 构建页面菜单及页面实际使用的按钮权限定义。
func MenuSpecs(table *Table, columns []*CodeGenColumn, methods []*Proto, resourcePath string, tableComment string, localeState LocaleState) (CodeGenMenuSpec, []CodeGenMenuSpec) {
	listMethodName := "Page" + table.EntityName
	if isTreePageType(table.PageType) {
		listMethodName = "Tree" + table.EntityName
	}
	methodByName := make(map[string]*Proto, len(methods))
	for _, method := range methods {
		if method.TargetEntityName == table.EntityName {
			methodByName[method.MethodName] = method
		}
	}
	pageAPIs := make([]string, 0, 4)
	if listMethod := methodByName[listMethodName]; listMethod != nil {
		pageAPIs = append(pageAPIs, GeneratedRPCPath(table, listMethod))
	}
	for _, method := range methods {
		if IsOptionProtoMethod(method) {
			pageAPIs = append(pageAPIs, GeneratedRPCPath(table, method))
		}
	}
	pageMenu := &models.BaseMenu{
		ParentID:  table.ParentMenuID,
		Type:      _const.BASE_MENU_TYPE_MENU,
		Path:      "/" + strings.Trim(resourcePath, "/"),
		Name:      table.EntityName,
		Component: resourcePath + "/index",
		Meta: marshalJSON(map[string]any{
			"icon":        "Grid",
			"title":       DefaultString(tableComment, table.BusinessName),
			"hidden":      false,
			"keep_alive":  true,
			"always_show": false,
		}),
		API:    marshalJSON(pageAPIs),
		Sort:   100,
		Status: coreconst.STATUS_STATUS_ENABLE,
	}

	permission := PermissionPrefix(table)
	buttonSpecs := make([]CodeGenMenuSpec, 0, 4)
	if createMethod := methodByName["Create"+table.EntityName]; createMethod != nil {
		buttonSpecs = append(buttonSpecs, newButtonMenuSpec(
			permission+":create",
			"新增"+table.BusinessName,
			GeneratedMenuTranslations(table, nil, "create", localeState),
			int32(len(buttonSpecs)+1),
			GeneratedRPCPath(table, createMethod),
		))
	}
	getMethod := methodByName["Get"+table.EntityName]
	updateMethod := methodByName["Update"+table.EntityName]
	if getMethod != nil && updateMethod != nil {
		buttonSpecs = append(buttonSpecs, newButtonMenuSpec(
			permission+":update",
			"编辑"+table.BusinessName,
			GeneratedMenuTranslations(table, nil, "update", localeState),
			int32(len(buttonSpecs)+1),
			GeneratedRPCPath(table, getMethod),
			GeneratedRPCPath(table, updateMethod),
		))
	}
	if deleteMethod := methodByName["Delete"+table.EntityName]; deleteMethod != nil {
		buttonSpecs = append(buttonSpecs, newButtonMenuSpec(
			permission+":delete",
			"删除"+table.BusinessName,
			GeneratedMenuTranslations(table, nil, "delete", localeState),
			int32(len(buttonSpecs)+1),
			GeneratedRPCPath(table, deleteMethod),
		))
	}
	statusColumnList := statusColumns(columns)
	for _, column := range statusColumnList {
		method := findStatusMethodForColumn(column, methods)
		if method == nil {
			continue
		}
		buttonSpecs = append(buttonSpecs, newButtonMenuSpec(
			statusPermissionPath(permission, column.Name, len(statusColumnList)),
			"设置"+DefaultString(column.Comment, column.Name),
			GeneratedMenuTranslations(table, column, "status", localeState),
			int32(len(buttonSpecs)+1),
			GeneratedRPCPath(table, method),
		))
	}
	pageTitle := DefaultString(tableComment, table.BusinessName)
	return CodeGenMenuSpec{
		Menu:         pageMenu,
		SourceTitle:  pageTitle,
		Translations: GeneratedMenuTranslations(table, nil, "", localeState),
	}, buttonSpecs
}

// statusPermissionPath 返回状态字段对应的按钮权限标识。
func statusPermissionPath(permissionPrefix string, columnName string, statusColumnCount int) string {
	suffix := "status"
	if statusColumnCount > 1 {
		suffix += ":" + strings.ReplaceAll(columnName, "_", ":")
	}
	return permissionPrefix + ":" + suffix
}

// ShouldSyncMenus 判断当前生成结果是否具备页面菜单同步条件。
func ShouldSyncMenus(table *Table, methods []*Proto) bool {
	return table.GenSql == 1 && table.GenFrontend == 1 && frontendPageMethodsComplete(table, methods)
}

// GeneratedRPCServicePath 返回实体服务对应的完整 gRPC 权限路径前缀。
func GeneratedRPCServicePath(table *Table, entityName string) string {
	return "/" + ProtoTargetForTable(table).PackageName + "." + entityName + "Service"
}

// GeneratedRPCPath 返回生成方法对应的完整 gRPC 权限路径。
func GeneratedRPCPath(table *Table, method *Proto) string {
	return GeneratedRPCServicePath(table, method.TargetEntityName) + "/" + method.MethodName
}

// newButtonMenuSpec 创建按钮菜单定义。
func newButtonMenuSpec(path string, title string, translations map[string]string, sort int32, apis ...string) CodeGenMenuSpec {
	return CodeGenMenuSpec{
		Menu: &models.BaseMenu{
			Type:   _const.BASE_MENU_TYPE_BUTTON,
			Path:   path,
			Meta:   marshalJSON(map[string]string{"title": title}),
			API:    marshalJSON(apis),
			Sort:   sort,
			Status: coreconst.STATUS_STATUS_ENABLE,
		},
		SourceTitle:  title,
		Translations: translations,
	}
}

// marshalJSON 将模板配置编码为稳定的 JSON 文本。
func marshalJSON(value any) string {
	data, _ := json.Marshal(value)
	return string(data)
}
