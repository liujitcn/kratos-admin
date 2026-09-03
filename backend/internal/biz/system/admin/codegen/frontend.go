package codegen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	adminv1 "github.com/liujitcn/kratos-admin/backend/api/gen/go/system/admin/v1"
	"github.com/liujitcn/kratos-core/errorsx"

	"github.com/liujitcn/go-utils/stringcase"
)

// --- 管理端 API 与页面模板渲染 ---

// renderFrontendAPIFile 渲染前端 API 文件占位内容。
func (c *renderer) renderFrontendAPIFile(table *Table, methods []*Proto) string {
	entity := table.EntityName
	target := ProtoTargetForTable(table)
	snakeEntity := stringcase.ToSnakeCase(entity)
	urlConst := stringcase.UpperSnakeCase(entity) + "_URL"
	statusMethods := methodsByKinds(methods, APIKindStatus)
	hasTree := firstTreeMethodByTrigger(methods, TriggerPageTree) != nil
	optionMethod := currentEntityOptionMethod(table, methods)
	methodNames := protoMethodNameSet(methods)
	_, hasPage := methodNames["Page"+entity]
	_, hasGet := methodNames["Get"+entity]
	_, hasCreate := methodNames["Create"+entity]
	_, hasUpdate := methodNames["Update"+entity]
	_, hasDelete := methodNames["Delete"+entity]
	var typeImports []string
	if hasCreate {
		typeImports = append(typeImports, "  type Create"+entity+"Request")
	}
	if hasDelete {
		typeImports = append(typeImports, "  type Delete"+entity+"Request")
	}
	if hasGet {
		typeImports = append(typeImports, "  type Get"+entity+"Request")
	}
	if hasTree {
		typeImports = append(typeImports, "  type Tree"+entity+"Request", "  type Tree"+entity+"Response")
	} else if hasPage {
		typeImports = append(typeImports, "  type Page"+entity+"Request", "  type Page"+entity+"Response")
	}
	if optionMethod != nil {
		typeImports = append(typeImports, "  type "+optionMethod.MethodName+"Request")
	}
	for _, statusMethod := range statusMethods {
		typeImports = append(typeImports, "  type "+statusMethod.MethodName+"Request")
	}
	if hasGet || hasCreate || hasUpdate {
		typeImports = append(typeImports, "  type "+entity+"Form")
	}
	typeImports = append(typeImports, "  type "+entity+"Service")
	if hasUpdate {
		typeImports = append(typeImports, "  type Update"+entity+"Request")
	}

	emptyImport := ""
	if hasCreate || hasUpdate || hasDelete || len(statusMethods) > 0 {
		emptyImport = "import type { Empty } from \"" + target.FrontendPackageName + "/rpc/google/protobuf/empty\";\n"
	}
	optionImport := ""
	if optionMethod != nil {
		optionImport = "import type { " + strings.Join(tsCommonOptionResponseTypes([]*Proto{optionMethod}), ", ") + " } from \"" + target.FrontendPackageName + "/rpc/common/v1/common\";\n"
	}
	var methodsBuilder strings.Builder
	if hasTree {
		methodsBuilder.WriteString(fmt.Sprintf(`
  /** 查询树形列表 */
  Tree%s(request: Tree%sRequest): Promise<Tree%sResponse> {
    return service<Tree%sRequest, Tree%sResponse>({
      url: %s + "/tree",
      method: "get",
      params: request
    });
  }
`, entity, entity, entity, entity, entity, urlConst))
	}
	if optionMethod != nil {
		returnType := "SelectOptionResponse"
		comment := "查询下拉选择"
		if optionMethod.APIKind == APIKindTree {
			returnType = "TreeOptionResponse"
			comment = "查询树形选择"
		}
		methodsBuilder.WriteString(fmt.Sprintf(`
  /** %s */
  %s(request: %sRequest): Promise<%s> {
    return service<%sRequest, %s>({
      url: %s + "/option",
      method: "get",
      params: request
    });
  }
`, comment, optionMethod.MethodName, optionMethod.MethodName, returnType, optionMethod.MethodName, returnType, urlConst))
	}
	coreMethods := fmt.Sprintf(`
  /** 查询分页列表 */
  Page%s(request: Page%sRequest): Promise<Page%sResponse> {
    return service<Page%sRequest, Page%sResponse>({
      url: %s,
      method: "get",
      params: request
    });
  }

  /** 查询详情 */
  Get%s(request: Get%sRequest): Promise<%sForm> {
    return service<Get%sRequest, %sForm>({
      url: %s + "/" + request.id,
      method: "get"
    });
  }

  /** 创建 */
  Create%s(request: Create%sRequest): Promise<Empty> {
    return service<%sForm | undefined, Empty>({
      url: %s,
      method: "post",
      data: request.%s
    });
  }

  /** 更新 */
  Update%s(request: Update%sRequest): Promise<Empty> {
    return service<%sForm | undefined, Empty>({
      url: %s + "/" + request.id,
      method: "put",
      data: request.%s
    });
  }

  /** 删除 */
  Delete%s(request: Delete%sRequest): Promise<Empty> {
    return service<Delete%sRequest, Empty>({
      url: %s + "/" + request.ids,
      method: "delete"
    });
  }
`, entity, entity, entity, entity, entity, urlConst, entity, entity, entity, entity, entity, urlConst, entity, entity, entity, urlConst, snakeEntity, entity, entity, entity, urlConst, snakeEntity, entity, entity, entity, urlConst)
	if hasTree {
		detailMethodIndex := strings.Index(coreMethods, "  /** 查询详情 */")
		if detailMethodIndex >= 0 {
			coreMethods = coreMethods[detailMethodIndex:]
		}
	}
	methodsBuilder.WriteString(coreMethods)
	for _, statusMethod := range statusMethods {
		methodsBuilder.WriteString(fmt.Sprintf(`
  /** 设置状态 */
  %s(request: %sRequest): Promise<Empty> {
    return service<%sRequest, Empty>({
      url: %s + "/" + request.id + "/%s",
      method: "put",
      data: request
    });
  }
`, statusMethod.MethodName, statusMethod.MethodName, statusMethod.MethodName, urlConst, statusResourcePath(table, statusMethod)))
	}
	content := renderTemplate("frontend_api.tmpl", frontendAPITemplateData{
		Entity:       entity,
		BusinessName: table.BusinessName,
		TypeImports:  strings.Join(typeImports, ",\n"),
		RPCImport:    frontendRPCImportPath(c.defaultProtoPath(table)),
		EmptyImport:  emptyImport,
		OptionImport: optionImport,
		URLConst:     urlConst,
		ResourcePath: c.frontendResourcePath(table),
		Methods:      methodsBuilder.String(),
	})
	content = removeTSClassMethods(content, missingCoreMethodNames(table, methods))
	return reorderTSClassMethods(content, entity+"ServiceImpl")
}

// renderExternalTargetFrontendAPIFile 渲染外部目标实体的最小前端 API 文件。
func (c *renderer) renderExternalTargetFrontendAPIFile(table *Table, methods []*Proto) string {
	target := ProtoTargetForTable(table)
	entity := table.EntityName
	urlConst := stringcase.UpperSnakeCase(entity) + "_URL"
	var typeImports []string
	for _, method := range methods {
		typeImports = append(typeImports, "  type "+method.MethodName+"Request")
	}
	typeImports = append(typeImports, "  type "+entity+"Service")
	return renderTemplate("frontend_api.tmpl", frontendAPITemplateData{
		Entity:       entity,
		BusinessName: table.BusinessName,
		TypeImports:  strings.Join(typeImports, ",\n"),
		RPCImport:    frontendRPCImportPath(c.defaultProtoPath(table)),
		OptionImport: "import type { " + strings.Join(tsCommonOptionResponseTypes(methods), ", ") + " } from \"" + target.FrontendPackageName + "/rpc/common/v1/common\";\n",
		URLConst:     urlConst,
		ResourcePath: c.frontendResourcePath(table),
		Methods:      c.renderExternalTargetFrontendAPIMethods(table, methods),
	})
}

// appendExternalTargetFrontendAPIMethods 仅补齐已有前端 API 文件缺失的外部目标选项方法。
func (c *renderer) appendExternalTargetFrontendAPIMethods(content string, table *Table, methods []*Proto) string {
	className := table.EntityName + "ServiceImpl"
	existingBlocks, _, _, ok := tsClassMethodBlocks(content, className)
	if !ok {
		return content
	}
	existingMethodNames := make(map[string]struct{}, len(existingBlocks))
	for _, block := range existingBlocks {
		existingMethodNames[block.Name] = struct{}{}
	}
	missingMethods := make([]*Proto, 0, len(methods))
	for _, method := range methods {
		if _, exists := existingMethodNames[method.MethodName]; !exists {
			missingMethods = append(missingMethods, method)
		}
	}
	if len(missingMethods) == 0 {
		return content
	}
	typeNames := make([]string, 0, len(missingMethods))
	for _, method := range missingMethods {
		typeNames = append(typeNames, method.MethodName+"Request")
	}
	content = ensureTSNamedTypeNames(content, frontendRPCImportPath(c.defaultProtoPath(table)), typeNames)
	content = ensureTSCommonOptionImport(content, missingMethods, ProtoTargetForTable(table).FrontendPackageName+"/rpc/common/v1/common")
	return mergeGeneratedTSClassMethods(content, c.renderExternalTargetFrontendAPIFile(table, missingMethods), className)
}

// renderExternalTargetFrontendAPIMethods 渲染外部目标实体选项 API 方法。
func (c *renderer) renderExternalTargetFrontendAPIMethods(table *Table, methods []*Proto) string {
	urlConst := stringcase.UpperSnakeCase(table.EntityName) + "_URL"
	var builder strings.Builder
	for _, method := range methods {
		returnType := "SelectOptionResponse"
		comment := "查询下拉选择"
		if method.APIKind == APIKindTree {
			returnType = "TreeOptionResponse"
			comment = "查询树形选择"
		}
		builder.WriteString(fmt.Sprintf(`
  /** %s */
  %s(request: %sRequest): Promise<%s> {
    return service<%sRequest, %s>({
      url: %s + "/option",
      method: "get",
      params: request
    });
  }
`, comment, method.MethodName, method.MethodName, returnType, method.MethodName, returnType, urlConst))
	}
	return builder.String()
}

// renderFrontendPageFile 渲染前端页面内容。
func (c *renderer) renderFrontendPageFile(table *Table, columns []*CodeGenColumn, methods []*Proto, paths *adminv1.CodeGenOutputPaths) string {
	// 页面中的查询、列表与表单共用字段排序，Proto 仍使用调用方保留的数据库字段顺序。
	columns = slices.Clone(columns)
	slices.SortStableFunc(columns, func(left *CodeGenColumn, right *CodeGenColumn) int {
		if left.Sort < right.Sort {
			return -1
		}
		if left.Sort > right.Sort {
			return 1
		}
		return 0
	})
	entity := table.EntityName
	target := ProtoTargetForTable(table)
	pluralEntity := pluralize(entity)
	snakeEntity := stringcase.ToSnakeCase(entity)
	frontendAPIImport := target.FrontendPackageName + "/api/" + table.BusinessModule + "/" + snakeEntity
	frontendRPCImport := frontendRPCImportPath(paths.GetProtoFilePath())
	listField := stringcase.ToSnakeCase(pluralEntity)
	hasTenantOption := hasTenantQueryOption(columns) || hasTenantListColumn(columns)
	statusTypeImport := renderFrontendStatusTypeImports(columns, methods)
	formStateType := renderFrontendFormStateType(table, columns)
	formDataType := entity + "Form"
	if formStateType != "" {
		formDataType = entity + "FormState"
	}
	proFormTypeImport := "ProFormField"
	for _, column := range columns {
		hasTypedOptions := false
		for _, scope := range frontendOptionScopes(column) {
			if scope.option.SourceType == OptionSourceStatic || scope.option.SourceType == OptionSourceTable {
				hasTypedOptions = true
				break
			}
		}
		if statusNeedsFrontendOptions(column) || hasTypedOptions {
			proFormTypeImport += ", ProFormOption"
			break
		}
	}
	tenantImports := ""
	tenantState := ""
	if hasTenantOption {
		tenantImports = "import { useUserStore } from \"@liujitcn/kratos-admin-core/stores/runtime\";\nimport { DEFAULT_TENANT_CODE, requestTenantOptions } from \"@liujitcn/kratos-admin-core/tenant\";"
		tenantState = `const userStore = useUserStore();
/** 当前登录账号是否默认租户。 */
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);`
	}
	if tenantState != "" {
		tenantState += "\n"
	}
	tenantState += `/** 树形页面行内新增时锁定继承的上下文。 */
const treeCreateState = reactive({
  lockTenant: false,
  lockParent: false
});`
	script := fmt.Sprintf(`<script setup lang="ts">
import { computed, reactive, ref } from "vue";
%s
import type { FormRules } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { %s } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { t } from "@liujitcn/kratos-admin-core";
import { def%sService } from "%s";
%s
%s
import type { Page%sRequest, %s, %sForm%s } from "%s";
%s
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";

defineOptions({
  name: "%s",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const localeKeyPrefix = "%s";
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
%s
%s

const dialog = reactive({
  visible: false,
  editing: false
});
const dialogTitle = computed(() => t(dialog.editing ? "common.action.edit_resource" : "common.action.create_resource", { resource: t(localeKeyPrefix + ".resource") }));

const formData = reactive<%s>({
%s
});

const rules = computed<FormRules>(() => ({
%s
}));
%s
/** %s表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
%s
]);

/** %s表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
%s
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: __CODEGEN_OPERATION_WIDTH__,
    fixed: "right",
    cellType: "actions",
    actions: [
__CODEGEN_TREE_CREATE_ACTION__
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["%s:update"],
__CODEGEN_TREE_EDIT_PARAMS__
        onClick: __CODEGEN_EDIT_HANDLER__
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["%s:delete"],
        onClick: scope => handleDelete(scope.row as %s)
      }
    ]
  }
]);

/** %s顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["%s:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["%s:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as %s[])
  }
]);

%s
/**
 * 请求%s列表，并适配 ProTable 固定列表字段。
 */
async function request%sTable(params: Page%sRequest) {
  const requestParams = buildPageRequest(params);
  const data = await def%sService.Page%s(requestParams);
  const list = data.%s ?? [];
  return { data: { ...data, list } };
}

/**
 * 刷新%s表格。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 重置%s表单。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
%s
}

/**
 * 打开%s弹窗。
 */
async function handleOpenDialog(__CODEGEN_OPEN_DIALOG_PARAMETERS__) {
  resetForm();
%s  dialog.editing = Boolean(id);
  dialog.visible = true;
__CODEGEN_OPEN_DIALOG_DATA__
}

/**
 * 提交%s表单。
 */
function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;

    const payload = JSON.parse(JSON.stringify(formData)) as %sForm;
__CODEGEN_PASSWORD_UPDATE__
    const request = payload.id
      ? def%sService.Update%s({ id: payload.id, %s: payload })
      : def%sService.Create%s({ %s: payload });
    request.then(() => {
      ElMessage.success(t(payload.id ? "common.message.update_success" : "common.message.create_success", { resource: t(localeKeyPrefix + ".resource") }));
      handleCloseDialog();
      refreshTable();
    });
  });
}
%s
/**
 * 删除%s，兼容单项删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | %s | %s[]) {
  const rowList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as %s[])
    : selected && typeof selected === "object"
      ? [selected as %s]
      : [];
  const ids = (
    rowList.length ? rowList.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>)
  ).join(",");
  if (!ids) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }

  const confirmMessage = t(rowList.length === 1 ? "common.dialog.delete_single" : "common.dialog.delete_selected", { resource: t(localeKeyPrefix + ".resource") });
  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      def%sService.Delete%s({ ids }).then(() => {
        ElMessage.success(t("common.message.delete_success", { resource: t(localeKeyPrefix + ".resource") }));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("common.dialog.cancel_delete", { resource: t(localeKeyPrefix + ".resource") }));
    }
  );
}

/**
 * 关闭%s弹窗。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}
</script>
			`, renderFrontendDateImport(columns), proFormTypeImport, entity, frontendAPIImport, c.renderFrontendOptionImports(table, columns, methods), tenantImports, entity, entity, entity, statusTypeImport, frontendRPCImport, c.renderFrontendEnumImports(table, columns), entity, FrontendLocaleKeyPrefix(table), formStateType, tenantState, formDataType, c.renderFrontendFormDefaults(columns), c.renderFrontendRules(table, columns), c.renderFrontendStatusOptions(table, columns)+c.renderFrontendOptionState(table, columns, methods), table.BusinessName, c.renderFrontendFormFields(table, columns), table.BusinessName, c.renderFrontendColumns(table, columns, methods), PermissionPrefix(table), entity, PermissionPrefix(table), entity, table.BusinessName, PermissionPrefix(table), PermissionPrefix(table), entity, table.BusinessName, entity, entity, entity, entity, listField, table.BusinessName, table.BusinessName, c.renderFrontendResetForm(columns), table.BusinessName, c.renderFrontendLoadOptionsCall(columns, methods), table.BusinessName, entity, entity, entity, snakeEntity, entity, entity, snakeEntity, c.renderFrontendStatusHandlers(table, columns, methods), table.BusinessName, entity, entity, entity, entity, entity, entity, table.BusinessName)
	if passwordColumn := c.findFrontendPasswordColumn(columns); passwordColumn != nil {
		script = strings.Replace(
			script,
			`import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";`,
			`import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import PasswordStrength from "@liujitcn/kratos-admin-core/components/PasswordStrength/index.vue";`,
			1,
		)
	}
	script = strings.ReplaceAll(script, "__CODEGEN_OPERATION_WIDTH__", c.renderFrontendOperationWidth(table))
	script = strings.ReplaceAll(script, "__CODEGEN_TREE_CREATE_ACTION__", c.renderFrontendTreeCreateAction(table, columns))
	script = strings.ReplaceAll(script, "__CODEGEN_TREE_EDIT_PARAMS__", c.renderFrontendTreeEditParams(table))
	script = strings.ReplaceAll(script, "__CODEGEN_EDIT_HANDLER__", c.renderFrontendEditHandler(table))
	script = strings.ReplaceAll(script, "__CODEGEN_OPEN_DIALOG_PARAMETERS__", c.renderFrontendOpenDialogParameters(table, columns))
	script = strings.ReplaceAll(script, "__CODEGEN_OPEN_DIALOG_DATA__", c.renderFrontendOpenDialogData(table, columns))
	script = strings.ReplaceAll(script, "__CODEGEN_PASSWORD_UPDATE__", c.renderFrontendPasswordUpdate(columns))
	script = c.renderFrontendTreeParentField(script, table)
	script = c.reorderFrontendPageMethods(script)
	scriptLines := strings.Split(script, "\n")
	for i := range scriptLines {
		scriptLines[i] = strings.TrimRight(scriptLines[i], " \t")
	}
	script = strings.Join(scriptLines, "\n")
	script = strings.TrimRight(script, " \t\r\n") + "\n"
	content := renderTemplate("frontend_page.tmpl", frontendPageTemplateData{Entity: entity, BusinessName: table.BusinessName, HasTenantOption: hasTenantOption, Script: script})
	passwordSlot := ""
	if passwordColumn := c.findFrontendPasswordColumn(columns); passwordColumn != nil {
		passwordSlot = fmt.Sprintf(`      <template #passwordStrength>
        <PasswordStrength :password="formData.%s" />
      </template>
`, passwordColumn.Name)
	}
	content = strings.Replace(content, "__CODEGEN_PASSWORD_STRENGTH_SLOT__", passwordSlot, 1)
	return c.applyFrontendPageType(content, table, methods, frontendAPIImport)
}

// renderFrontendOperationWidth 返回操作列宽度。
func (c *renderer) renderFrontendOperationWidth(table *Table) string {
	if isTreePageType(table.PageType) {
		return "220"
	}
	return "150"
}

// renderFrontendTreeCreateAction 渲染树形表格的行内新增子节点操作。
func (c *renderer) renderFrontendTreeCreateAction(table *Table, columns []*CodeGenColumn) string {
	if !isTreePageType(table.PageType) {
		return ""
	}
	tenantColumn := FindColumnByName(columns, "tenant_id")
	if tenantColumn != nil && generatedFormIncludesColumn(tenantColumn) {
		return fmt.Sprintf(`      {
        label: t("common.action.create"),
        type: "primary",
        link: true,
        icon: CirclePlus,
        hidden: () => !BUTTONS.value["%s:create"],
        params: scope => ({
          parentId: (scope.row as %s).id,
          tenantId: (scope.row as unknown as Record<string, unknown>)["tenant_id"] as number | undefined
        }),
        onClick: (_, params) =>
          handleOpenDialog(params?.parentId as number | undefined, undefined, params?.tenantId as number | undefined)
      },
`, PermissionPrefix(table), table.EntityName)
	}
	return fmt.Sprintf(`      {
        label: t("common.action.create"),
        type: "primary",
        link: true,
        icon: CirclePlus,
        hidden: () => !BUTTONS.value["%s:create"],
        params: scope => ({ parentId: (scope.row as %s).id }),
        onClick: (_, params) => handleOpenDialog(params?.parentId as number | undefined)
      },
`, PermissionPrefix(table), table.EntityName)
}

// renderFrontendTreeEditParams 渲染树形表格编辑操作所需的父节点参数。
func (c *renderer) renderFrontendTreeEditParams(table *Table) string {
	if !isTreePageType(table.PageType) {
		return ""
	}
	parentColumn := DefaultString(table.ParentColumn, "parent_id")
	return fmt.Sprintf(`        params: scope => ({
          parentId: (scope.row as %s)[%q] as number | undefined,
          entityId: (scope.row as %s).id
        }),
`, table.EntityName, parentColumn, table.EntityName)
}

// renderFrontendEditHandler 返回普通页面或树形页面的编辑操作处理器。
func (c *renderer) renderFrontendEditHandler(table *Table) string {
	if !isTreePageType(table.PageType) {
		return fmt.Sprintf("scope => handleOpenDialog((scope.row as %s).id)", table.EntityName)
	}
	return "(_, params) => handleOpenDialog(params?.parentId as number | undefined, params?.entityId as number | undefined)"
}

// renderFrontendOpenDialogParameters 返回页面弹窗方法的参数定义。
func (c *renderer) renderFrontendOpenDialogParameters(table *Table, columns []*CodeGenColumn) string {
	if isTreePageType(table.PageType) {
		if tenantColumn := FindColumnByName(columns, "tenant_id"); tenantColumn != nil && generatedFormIncludesColumn(tenantColumn) {
			return "parentId?: number, id?: number, tenantId?: number"
		}
		return "parentId?: number, id?: number"
	}
	return "id?: number"
}

// renderFrontendOpenDialogData 渲染普通页面或树形页面的弹窗数据处理逻辑。
func (c *renderer) renderFrontendOpenDialogData(table *Table, columns []*CodeGenColumn) string {
	entity := table.EntityName
	passwordReset := renderFrontendPasswordReset(columns)
	if !isTreePageType(table.PageType) {
		return fmt.Sprintf(`  if (!id) return;

  const data = await def%sService.Get%s({ id });
  Object.assign(formData, data);
%s`, entity, entity, passwordReset)
	}
	parentColumn := DefaultString(table.ParentColumn, "parent_id")
	content := fmt.Sprintf(`  if (id) {
    const data = await def%sService.Get%s({ id });
    Object.assign(formData, data);
%s
    return;
  }

  formData[%q] = parentId ?? 0;
  treeCreateState.lockParent = parentId !== undefined;`, entity, entity, passwordReset, parentColumn)
	if tenantColumn := FindColumnByName(columns, "tenant_id"); tenantColumn != nil && generatedFormIncludesColumn(tenantColumn) {
		content += `
  if (tenantId !== undefined) {
    formData["tenant_id"] = tenantId;
  }
  treeCreateState.lockTenant = parentId !== undefined;`
	}
	return content
}

// renderFrontendPasswordUpdate 渲染编辑态删除密码字段的语句，避免空值覆盖原密码。
func (c *renderer) renderFrontendPasswordUpdate(columns []*CodeGenColumn) string {
	lines := make([]string, 0)
	for _, column := range columns {
		if column == nil || column.FormComponent != "password" {
			continue
		}
		lines = append(lines, fmt.Sprintf("    if (payload.id) delete (payload as unknown as Record<string, unknown>)[%q];", column.Name))
	}
	return strings.Join(lines, "\n")
}

// renderFrontendTreeParentField 为树形页面的父节点字段增加编辑态禁用配置。
func (c *renderer) renderFrontendTreeParentField(content string, table *Table) string {
	if !isTreePageType(table.PageType) {
		return content
	}
	parentColumn := DefaultString(table.ParentColumn, "parent_id")
	formFieldsStart := strings.Index(content, "const formFields = computed<ProFormField[]>(() => [")
	if formFieldsStart < 0 {
		return content
	}
	formFieldsEndOffset := strings.Index(content[formFieldsStart:], "\n]);")
	if formFieldsEndOffset < 0 {
		return content
	}
	formFieldsEnd := formFieldsStart + formFieldsEndOffset
	fieldStartOffset := strings.Index(content[formFieldsStart:formFieldsEnd], fmt.Sprintf(`  { prop: %q,`, parentColumn))
	if fieldStartOffset < 0 {
		return content
	}
	fieldStart := formFieldsStart + fieldStartOffset
	fieldEndOffset := strings.IndexByte(content[fieldStart:], '\n')
	fieldEnd := len(content)
	if fieldEndOffset >= 0 {
		fieldEnd = fieldStart + fieldEndOffset
	}
	field := content[fieldStart:fieldEnd]
	if strings.Contains(field, "treeCreateState.lockParent") {
		return content
	}
	if strings.Contains(field, "props: {") {
		field = strings.Replace(field, "props: {", "props: { disabled: Boolean(formData.id) || treeCreateState.lockParent, ", 1)
	} else {
		field = strings.TrimSpace(field)
		field = strings.TrimSuffix(field, "}")
		field += `, props: { disabled: Boolean(formData.id) || treeCreateState.lockParent } }`
	}
	return content[:fieldStart] + field + content[fieldEnd:]
}

// reorderFrontendPageMethods 统一生成页面的主流程方法顺序，保持列表、选项与弹窗逻辑按阅读顺序排列。
func (c *renderer) reorderFrontendPageMethods(content string) string {
	originalContent := content
	methodNames := []string{"loadFormOptions", "handleOpenDialog", "handleCloseDialog", "resetForm", "handleSubmit", "handleDelete"}
	blocks := make(map[string]string, len(methodNames))
	for _, methodName := range methodNames {
		var block string
		var ok bool
		content, block, ok = removeFrontendFunctionBlock(content, methodName)
		if ok {
			blocks[methodName] = block
		} else if methodName != "loadFormOptions" {
			return originalContent
		}
	}

	if len(blocks) == 0 {
		return content
	}

	refreshEnd := frontendFunctionEndWithLineBreak(content, "refreshTable")
	if refreshEnd < 0 {
		return originalContent
	}
	if block, ok := blocks["loadFormOptions"]; ok {
		content = content[:refreshEnd] + block + content[refreshEnd:]
	}

	methodSequence := []string{"handleOpenDialog", "handleCloseDialog", "resetForm", "handleSubmit"}
	sequence := strings.Builder{}
	for _, methodName := range methodSequence {
		if block, ok := blocks[methodName]; ok {
			sequence.WriteString(block)
		}
	}
	statusIndex := strings.Index(content, "\n/**\n * 切换")
	if statusIndex < 0 {
		statusIndex = strings.Index(content, "</script>")
	}
	if statusIndex < 0 {
		return originalContent
	}
	content = content[:statusIndex] + sequence.String() + content[statusIndex:]

	if block, ok := blocks["handleDelete"]; ok {
		scriptEnd := strings.Index(content, "</script>")
		if scriptEnd >= 0 {
			content = content[:scriptEnd] + block + content[scriptEnd:]
		}
	}
	return content
}

// applyFrontendPageType 根据页面类型补充树表格或左树右表结构。
func (c *renderer) applyFrontendPageType(content string, table *Table, methods []*Proto, frontendAPIImport string) string {
	switch table.PageType {
	case PageTypeTree, PageTypeTreeLazy:
		return c.applyFrontendTreePage(content, table, methods)
	case PageTypeLeftTree:
		return c.applyFrontendLeftTreePage(content, table, methods, frontendAPIImport)
	default:
		return content
	}
}

// applyFrontendTreePage 将普通分页模板调整为树形表格模板。
func (c *renderer) applyFrontendTreePage(content string, table *Table, methods []*Proto) string {
	treeMethod := firstTreeMethodByTrigger(methods, TriggerPageTree)
	if treeMethod == nil {
		return content
	}
	pluralEntity := pluralize(table.EntityName)
	content = strings.ReplaceAll(content, "Page"+table.EntityName+"Request", "Tree"+table.EntityName+"Request")
	content = strings.Replace(content, `import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";`, `import { normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";`, 1)
	requestMarker := fmt.Sprintf(`:request-api="request%sTable"`, table.EntityName)
	treeProps := requestMarker + `
      :pagination="false"
      :indent="20"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"`
	if table.PageType == PageTypeTreeLazy {
		treeProps = requestMarker + `
      :pagination="false"
      :indent="20"
      :lazy="true"
      :load="load` + table.EntityName + `Children"
      :tree-props="{ children: 'children', hasChildren: 'has_children' }"`
	}
	content = strings.Replace(content, requestMarker, treeProps, 1)

	functionName := "request" + table.EntityName + "Table"
	start := strings.Index(content, "async function "+functionName+"(")
	end := findTSFunctionEndIndex(content, start)
	if start < 0 || end < 0 {
		return content
	}
	requestFunction := fmt.Sprintf(`async function %s(params: Tree%sRequest) {
  const data = await def%sService.%s(%s);
  return { data: data.%s ?? [] };
}`, functionName, table.EntityName, table.EntityName, treeMethod.MethodName, treePageRequestExpression(table, treeMethod), stringcase.ToSnakeCase(pluralEntity))
	if table.PageType == PageTypeTreeLazy {
		requestFunction += fmt.Sprintf(`

/** 懒加载%s树形表格的子节点。 */
async function load%sChildren(row: %s, _treeNode: unknown, resolve: (data: %s[]) => void) {
  try {
    const data = await def%sService.%s({ %q: row.id, lazy: true } as Parameters<typeof def%sService.%s>[0]);
    resolve(data.%s ?? []);
  } catch {
    resolve([]);
  }
}`, table.BusinessName, table.EntityName, table.EntityName, table.EntityName, table.EntityName, treeMethod.MethodName, DefaultString(treeMethod.ParentColumn, DefaultString(table.ParentColumn, "parent_id")), table.EntityName, treeMethod.MethodName, stringcase.ToSnakeCase(pluralEntity))
	}
	return content[:start] + requestFunction + content[end+1:]
}

// applyFrontendLeftTreePage 将普通分页模板调整为左树右表模板。
func (c *renderer) applyFrontendLeftTreePage(content string, table *Table, methods []*Proto, frontendAPIImport string) string {
	config := LeftTreeConfigFromTable(table)
	filterColumn := DefaultString(config.FilterColumn, "parent_id")
	requestMarker := fmt.Sprintf(`:request-api="request%sTable"`, table.EntityName)
	content = strings.Replace(content, requestMarker, requestMarker+` :init-param="initParam"`, 1)
	treeFilterProps := ""
	if config.Lazy {
		treeFilterProps = fmt.Sprintf(`      :lazy="true"
      parent-key="%s"
      :show-all="false"`, DefaultString(config.ParentColumn, "parent_id"))
	}
	content = strings.Replace(content, `  <div class="table-box">`, fmt.Sprintf(`  <div class="main-box">
    <TreeFilter
      label="name"
      :title="t('%s.title.left_tree')"
      :request-api="request%sTreeFilter"
      :default-value="treeFilterValue"
%s
      @change="changeTreeFilter"
    />

    <div class="table-box">`, FrontendLocaleKeyPrefix(table), table.EntityName, treeFilterProps), 1)
	closingIndex := strings.LastIndex(content, "  </div>\n</template>")
	if closingIndex >= 0 {
		content = content[:closingIndex] + "    </div>\n  </div>\n</template>" + content[closingIndex+len("  </div>\n</template>"):]
	}
	content = strings.Replace(content, `import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";`, `import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import TreeFilter from "@liujitcn/kratos-admin-core/components/TreeFilter/index.vue";`, 1)

	treeMethod := firstTreeMethodByTrigger(methods, TriggerLeftTree)
	serviceName := ""
	requestCall := "return { data: [] };"
	requestSignature := "()"
	if treeMethod != nil {
		serviceName = "def" + treeMethod.TargetEntityName + "Service"
		parentColumn := DefaultString(treeMethod.ParentColumn, DefaultString(config.ParentColumn, "parent_id"))
		if config.Lazy {
			requestSignature = "(params?: Record<string, unknown>)"
			requestCall = fmt.Sprintf("const data = await %s.%s({ %q: params?.[%q] ?? 0, lazy: true } as Parameters<typeof %s.%s>[0]);\n  return { data: transform%sTreeNodes(data.list ?? []) };", serviceName, treeMethod.MethodName, parentColumn, parentColumn, serviceName, treeMethod.MethodName, table.EntityName)
		} else {
			requestCall = fmt.Sprintf("const data = await %s.%s({} as Parameters<typeof %s.%s>[0]);\n  return { data: transform%sTreeNodes(data.list ?? []) };", serviceName, treeMethod.MethodName, serviceName, treeMethod.MethodName, table.EntityName)
		}
		if treeMethod.TargetEntityName != table.EntityName {
			importLine := fmt.Sprintf(`import { %s } from %q;`, serviceName, frontendAPIImportPathForMethod(treeMethod))
			// 左树与表单字段可能依赖同一个外部服务，避免重复导入同名服务实例。
			if !strings.Contains(content, importLine) {
				apiImport := fmt.Sprintf(`import { def%sService } from "%s";`, table.EntityName, frontendAPIImport)
				content = strings.Replace(content, apiImport, apiImport+"\n"+importLine, 1)
			}
		}
	}

	refMarker := fmt.Sprintf("const formDialogRef = ref<InstanceType<typeof FormDialog>>();")
	state := fmt.Sprintf(`%s

type %sTreeOption = {
  value: number;
  label: string;
  children?: %sTreeOption[];
  has_children?: boolean;
};

type %sFilterNode = {
  id: string;
  name: string;
  has_children?: boolean;
  children?: %sFilterNode[];
};

const initParam = reactive({
  %s: undefined as number | undefined
});
const treeFilterValue = ref("");`, refMarker, table.EntityName, table.EntityName, table.EntityName, table.EntityName, filterColumn)
	content = strings.Replace(content, refMarker, state, 1)

	requestComment := fmt.Sprintf("/**\n * 请求%s列表", table.BusinessName)
	helper := fmt.Sprintf(`/** 转换左侧树筛选节点。 */
function transform%sTreeNodes(options: %sTreeOption[] = []): %sFilterNode[] {
  return options.map(option => ({
    id: String(option.value),
    name: option.label,
    has_children: option.has_children,
    children: transform%sTreeNodes(option.children ?? [])
  }));
}

/** 请求左侧树筛选数据。 */
async function request%sTreeFilter%s {
  %s
}

/** 切换左侧树筛选条件。 */
function changeTreeFilter(value: string) {
  treeFilterValue.value = value ?? "";
  initParam.%s = value ? Number(value) : undefined;
}

`, table.EntityName, table.EntityName, table.EntityName, table.EntityName, table.EntityName, requestSignature, requestCall, filterColumn)
	content = strings.Replace(content, requestComment, helper+requestComment, 1)
	openDialogMarker := `  resetForm();`
	openDialogReplacement := fmt.Sprintf(`  resetForm();
  // 新增时继承当前左树节点，保证记录归入正在查看的分组。
  if (!id && initParam.%s !== undefined) {
    formData.%s = initParam.%s;
  }
`, filterColumn, filterColumn, filterColumn)
	return strings.Replace(content, openDialogMarker, openDialogReplacement, 1)
}

// defaultProtoPath 获取默认 Proto 文件路径。
func (c *renderer) defaultProtoPath(table *Table) string {
	if table.ProtoFilePath != "" {
		return table.ProtoFilePath
	}
	return ProtoFilePath(ProtoTargetForTable(table).Directory, table.EntityName)
}

// frontendResourcePath 获取前端接口资源路径。
func (c *renderer) frontendResourcePath(table *Table) string {
	resourcePath := strings.Trim(table.ModulePath, "/")
	snakeEntity := stringcase.ToSnakeCase(table.EntityName)
	if resourcePath == "" {
		return snakeEntity
	}
	entityResourcePath := resourcePathByEntity(table.EntityName)
	if resourcePath == entityResourcePath {
		return resourcePath
	}
	modulePrefix := strings.ReplaceAll(resourcePath, "/", "_") + "_"
	if strings.HasPrefix(snakeEntity, modulePrefix) {
		snakeEntity = strings.TrimPrefix(snakeEntity, modulePrefix)
	}
	return resourcePath + "/" + snakeEntity
}

// renderFrontendColumns 渲染完整表格列配置。
func (c *renderer) renderFrontendColumns(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	list := make([]string, 0, len(columns))
	statusColumnCount := len(statusColumns(columns))
	hasVisibleTenantColumn := hasTenantListColumn(columns)
	if hasTenantQueryOption(columns) || hasVisibleTenantColumn {
		tenantVisibility := ""
		tenantSearch := ""
		if !hasVisibleTenantColumn {
			// 查询字段不在列表展示时保留搜索项，但禁止列设置将其切换到表格中。
			tenantVisibility = "        isShow: false,\n        isSetting: false,\n"
		}
		if hasTenantQueryOption(columns) {
			tenantSearch = `        search: { el: "select", key: "tenant_id", props: { filterable: true }, order: 1 },
`
		}
		list = append(list, fmt.Sprintf(`  ...(isDefaultTenant.value
	    ? ([{
%s        prop: "tenant_id",
        label: t(localeKeyPrefix + ".field.tenant_id"),
        minWidth: 140,
        align: "left",
        showOverflowTooltip: true,
%s        enum: requestTenantOptions
      }] satisfies ColumnProps[])
    : [])`, tenantVisibility, tenantSearch))
	}
	treeParentColumn := ""
	treeLabelColumn := ""
	if isTreePageType(table.PageType) {
		treeParentColumn = DefaultString(table.ParentColumn, "parent_id")
		treeLabelColumn = table.TreeLabelColumn
		if treeLabelColumn == "" {
			_, treeLabelColumn, _ = EntityOptionColumns(table, columns)
		}
	}
	appendColumn := func(column *CodeGenColumn, align string, hidden bool) {
		if hidden {
			// 查询专用字段不参与列表单元格渲染，避免误用列表组件或其选项依赖。
			list = append(list, fmt.Sprintf(`  { isShow: false, isSetting: false, prop: "%s", label: t(%q)%s }`, column.Name, frontendFieldLocaleKey(table, column), frontendSearchConfig(column)))
			return
		}
		align = c.resolveFrontendColumnAlign(column, align)
		content := renderFrontendColumn(FrontendLocaleKeyPrefix(table), PermissionPrefix(table), table.EntityName, column, findStatusMethodForColumn(column, methods), statusColumnCount, align)
		list = append(list, content)
	}
	// 树表格将树显示字段固定为首个数据列，确保 Element Plus 的缩进落在业务名称上。
	if treeLabelColumn != "" {
		for _, column := range columns {
			if hasVisibleTenantColumn && column.Name == "tenant_id" {
				continue
			}
			if column.Name != treeLabelColumn || column.Name == "deleted_at" || !generatedListIncludesColumn(column) {
				continue
			}
			appendColumn(column, "left", false)
			break
		}
	}
	for _, column := range columns {
		if hasVisibleTenantColumn && column.Name == "tenant_id" {
			continue
		}
		if column.Name == treeLabelColumn || column.Name == "deleted_at" {
			continue
		}
		if isTreePageType(table.PageType) && column.Name == treeParentColumn && treeParentColumn != treeLabelColumn {
			continue
		}
		if !generatedListIncludesColumn(column) || column.IsPrimary == 1 {
			continue
		}
		appendColumn(column, "", false)
	}
	// 查询配置独立于列表配置，查询专用字段以隐藏列形式保留搜索项。
	for _, column := range columns {
		if column.Name == "tenant_id" || column.Name == "deleted_at" {
			continue
		}
		if !generatedQueryIncludesColumn(column) || generatedListIncludesColumn(column) && column.IsPrimary != 1 {
			continue
		}
		appendColumn(column, "", true)
	}
	return strings.Join(list, ",\n") + ","
}

// resolveFrontendColumnAlign 返回代码生成页面列的默认对齐方式。
func (c *renderer) resolveFrontendColumnAlign(column *CodeGenColumn, align string) string {
	if align != "" {
		return align
	}
	if column == nil {
		return "left"
	}
	if column.ListComponent == "money" {
		return "right"
	}
	if column.IsStatusField == 1 || column.ListComponent == "image" {
		return "center"
	}
	if generatedListIncludesColumn(column) && isSelectComponent(column.ListComponent) {
		switch column.ListOption.SourceType {
		case OptionSourceTable:
			return "left"
		case OptionSourceDict, OptionSourceStatic:
			return "center"
		}
	}
	if DefaultString(column.TsType, InferTSType(column.DbType)) == "number" {
		return "right"
	}
	return "left"
}

// renderFrontendFormDefaults 渲染表单默认值。
func (c *renderer) renderFrontendFormDefaults(columns []*CodeGenColumn) string {
	lines := make([]string, 0, len(columns))
	for _, column := range columns {
		if !generatedFormIncludesColumn(column) && column.IsPrimary != 1 {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %s: %s", column.Name, frontendDefaultValue(column)))
	}
	return strings.Join(lines, ",\n")
}

// renderFrontendResetForm 渲染表单重置语句。
func (c *renderer) renderFrontendResetForm(columns []*CodeGenColumn) string {
	lines := make([]string, 0, len(columns))
	for _, column := range columns {
		if !generatedFormIncludesColumn(column) && column.IsPrimary != 1 {
			continue
		}
		lines = append(lines, fmt.Sprintf("  formData.%s = %s;", column.Name, frontendDefaultValue(column)))
	}
	lines = append(lines, "  treeCreateState.lockTenant = false;", "  treeCreateState.lockParent = false;")
	return strings.Join(lines, "\n")
}

// renderFrontendRules 渲染表单校验规则。
func (c *renderer) renderFrontendRules(table *Table, columns []*CodeGenColumn) string {
	lines := make([]string, 0)
	for _, column := range columns {
		if !generatedFormIncludesColumn(column) {
			continue
		}
		isString := DefaultString(column.TsType, InferTSType(column.DbType)) == "string"
		required := generatedFormRequired(column)
		if !required && (!isString || column.DbLength <= 0) {
			continue
		}
		trigger := "blur"
		if column.IsStatusField == 1 && column.StatusForm == 1 || column.FormComponent == "switch" || isSelectComponent(column.FormComponent) {
			trigger = "change"
		}
		fieldKey := frontendFieldLocaleKey(table, column)
		rules := fmt.Sprintf("{ required: true, message: t(\"common.validation.required_input\", { field: t(%q) }), trigger: %q }", fieldKey, trigger)
		if isString && column.DbLength > 0 {
			maxRule := fmt.Sprintf("{ max: %d, message: t(\"common.validation.max_length\", { field: t(%q), max: %d }), trigger: %q }", column.DbLength, fieldKey, column.DbLength, trigger)
			if required {
				rules += ", " + maxRule
			} else {
				rules = maxRule
			}
		}
		lines = append(lines, fmt.Sprintf("  %s: [%s]", column.Name, rules))
	}
	return strings.Join(lines, ",\n")
}

// renderFrontendStatusOptions 渲染状态选项。
func (c *renderer) renderFrontendStatusOptions(_ *Table, columns []*CodeGenColumn) string {
	statusColumnList := statusColumns(columns)
	var builder strings.Builder
	for _, column := range statusColumnList {
		if !statusNeedsFrontendOptions(column) {
			continue
		}
		builder.WriteString(fmt.Sprintf(`const %s = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: %s },
  { label: t("common.status.disabled"), value: %s }
]);
`, statusOptionsVariable(column, len(statusColumnList)), statusValueExpression(column, column.StatusEnabledValue, "1"), statusValueExpression(column, column.StatusDisabledValue, "2")))
	}
	return builder.String()
}

// renderFrontendFormFields 渲染 ProForm 字段配置。
func (c *renderer) renderFrontendFormFields(table *Table, columns []*CodeGenColumn) string {
	fields := make([]string, 0, len(columns))
	for _, column := range columns {
		if !generatedFormIncludesColumn(column) || column.IsPrimary == 1 {
			continue
		}
		fields = append(fields, c.renderFrontendFormField(table, column))
		if column.FormComponent == "password" {
			fields = append(fields, "  { prop: \"passwordStrength\", label: t(localeKeyPrefix + \".field.password_strength\"), component: \"slot\", slotName: \"passwordStrength\", visible: model => !model.id }")
		}
	}
	return strings.Join(fields, ",\n")
}

// findFrontendPasswordColumn 返回生成表单中第一个密码字段。
func (c *renderer) findFrontendPasswordColumn(columns []*CodeGenColumn) *CodeGenColumn {
	for _, column := range columns {
		if generatedFormIncludesColumn(column) && column.FormComponent == "password" {
			return column
		}
	}
	return nil
}

// renderFrontendFormField 渲染单个 ProForm 字段。
func (c *renderer) renderFrontendFormField(table *Table, column *CodeGenColumn) string {
	component := DefaultString(column.FormComponent, "input")
	label := fmt.Sprintf("t(%q)", frontendFieldLocaleKey(table, column))
	disabled := frontendFormFieldDisabledPrefix(column)
	if component == "password" {
		return fmt.Sprintf(`  { prop: "%s", label: %s, component: "password", props: { placeholder: t("common.validation.required_input", { field: %s }), showPassword: true }, visible: model => !model.id }`, column.Name, label, label)
	}
	if component == "switch" {
		// 开关提交值由表单范围的独立配置决定。
		return fmt.Sprintf(`  { prop: "%s", label: %s, component: "switch", props: { %sactiveValue: %s, inactiveValue: %s } }`, column.Name, label, disabled, statusValueExpression(column, column.FormOption.ActiveValue, "1"), statusValueExpression(column, column.FormOption.InactiveValue, "2"))
	}
	option := column.FormOption
	if option.Kind != "" && isSelectComponent(component) {
		props := fmt.Sprintf(`props: { %splaceholder: t("common.validation.required_select", { field: %s }), filterable: true, style: { width: "100%%" } }`, disabled, label)
		if option.Kind == APIKindTree && option.Lazy {
			props = fmt.Sprintf(`props: { %slazy: true, load: %s, placeholder: t("common.validation.required_select", { field: %s }), filterable: true, style: { width: "100%%" } }`, disabled, frontendOptionLoaderVar(column, "form"), label)
		}
		// 多选树形选择使用复选框，并允许选择任意层级节点。
		if isFormTreeMultiple(column) {
			lazyProps := ""
			if option.Kind == APIKindTree && option.Lazy {
				lazyProps = fmt.Sprintf("lazy: true, load: %s, ", frontendOptionLoaderVar(column, "form"))
			}
			props = fmt.Sprintf(`props: { %s%smultiple: true, showCheckbox: true, checkStrictly: true, nodeKey: "value", placeholder: t("common.validation.required_select", { field: %s }), filterable: true, style: { width: "100%%" } }`, disabled, lazyProps, label)
		}
		switch option.SourceType {
		case OptionSourceDict:
			if option.SourceValue != "" {
				if component == "radio-group" {
					return fmt.Sprintf(`  { prop: "%s", label: %s, component: "dict", props: { %scode: %q, codeType: %q, type: "radio" } }`, column.Name, label, disabled, option.SourceValue, frontendDictValueType(column))
				}
				if component == "checkbox-group" {
					return fmt.Sprintf(`  { prop: "%s", label: %s, component: "dict", props: { %scode: %q, codeType: %q, type: "checkbox" } }`, column.Name, label, disabled, option.SourceValue, frontendDictValueType(column))
				}
				return fmt.Sprintf(`  { prop: "%s", label: %s, component: "dict", props: { %scode: %q, codeType: %q } }`, column.Name, label, disabled, option.SourceValue, frontendDictValueType(column))
			}
		case OptionSourceStatic:
			return fmt.Sprintf(`  { prop: "%s", label: %s, component: "%s", options: %sOptions.value, %s }`, column.Name, label, component, frontendOptionVar(column, "form"), props)
		case OptionSourceTable:
			return fmt.Sprintf(`  { prop: "%s", label: %s, component: "%s", options: %sOptions.value, %s }`, column.Name, label, component, frontendOptionVar(column, "form"), props)
		}
	}
	if component == "input-number" {
		return fmt.Sprintf(`  { prop: "%s", label: %s, component: "input-number", props: { %smin: 0, precision: %d, controlsPosition: "right", style: { width: "100%%" } } }`, column.Name, label, disabled, column.DbScale)
	}
	if component == "date-picker" {
		dbType := strings.ToLower(DefaultString(column.ColumnType, column.DbType))
		if strings.Contains(dbType, "datetime") || strings.Contains(dbType, "timestamp") {
			return fmt.Sprintf(`  { prop: "%s", label: %s, component: "date-picker", props: { %stype: "datetime", valueFormat: "YYYY-MM-DD HH:mm:ss", placeholder: t("common.validation.required_select", { field: %s }), style: { width: "100%%" } } }`, column.Name, label, disabled, label)
		}
		return fmt.Sprintf(`  { prop: "%s", label: %s, component: "date-picker", props: { %stype: "date", valueFormat: "YYYY-MM-DD", placeholder: t("common.validation.required_select", { field: %s }), style: { width: "100%%" } } }`, column.Name, label, disabled, label)
	}
	return fmt.Sprintf(`  { prop: "%s", label: %s, component: "%s", props: { %splaceholder: t("common.validation.required_input", { field: %s }) } }`, column.Name, label, component, disabled, label)
}

// renderFrontendOptionState 渲染静态选项、数据表选项状态与加载方法。
func (c *renderer) renderFrontendOptionState(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	var builder strings.Builder
	hasTableSource := false
	hasLazyTree := false
	hasTreeOptionPaths := false
	for _, column := range columns {
		for _, scope := range frontendOptionScopes(column) {
			if scope.option.SourceType != OptionSourceStatic && scope.option.SourceType != OptionSourceTable {
				continue
			}
			switch scope.option.SourceType {
			case OptionSourceStatic:
				builder.WriteString(fmt.Sprintf("const %sOptions = computed<ProFormOption[]>(() => %s);\n", frontendOptionVar(column, scope.name), renderFrontendStaticOptions(table, column, scope.option)))
			case OptionSourceTable:
				hasTableSource = true
				hasLazyTree = hasLazyTree || scope.name == "form" && scope.option.Kind == APIKindTree && scope.option.Lazy
				hasTreeOptionPaths = hasTreeOptionPaths || scope.option.Kind == APIKindTree
				builder.WriteString(fmt.Sprintf("const %sOptions = ref<ProFormOption[]>([]);\n", frontendOptionVar(column, scope.name)))
			}
		}
	}
	if builder.Len() == 0 {
		return ""
	}
	if !hasTableSource {
		return builder.String()
	}
	if hasTreeOptionPaths {
		builder.WriteString(`
/**
 * 将树形选项转换为完整路径标签。
 */
function transformTreeOptionPaths(options: ProFormOption[] = [], parentPath = ""): ProFormOption[] {
  return options.map(option => {
    const fullPath = [parentPath, option.label].filter(Boolean).join("/");
    return {
      ...option,
      label: fullPath,
      children: transformTreeOptionPaths(option.children ?? [], fullPath)
    };
  });
}
`)
	}
	if hasLazyTree {
		builder.WriteString(`
type GeneratedTreeOption = ProFormOption & {
  has_children?: boolean;
  isLeaf?: boolean;
};

/** 将树形接口返回的子节点转换为 Element Plus 懒加载节点。 */
function normalizeLazyTreeOptions(options: GeneratedTreeOption[] = [], parentPath = ""): ProFormOption[] {
  return options.map(option => {
    const fullPath = [parentPath, option.label].filter(Boolean).join("/");
    return {
      ...option,
      label: fullPath,
      isLeaf: !option.has_children
    };
  });
}
`)
		for _, column := range columns {
			for _, scope := range frontendOptionScopes(column) {
				if scope.name != "form" || scope.option.SourceType != OptionSourceTable || scope.option.Kind != APIKindTree || !scope.option.Lazy {
					continue
				}
				method := findOptionMethodForConfig(scope.option, methods)
				if method == nil {
					continue
				}
				variable := frontendOptionVar(column, scope.name)
				serviceName := "def" + method.TargetEntityName + "Service"
				parentColumn := DefaultString(method.ParentColumn, "parent_id")
				builder.WriteString(fmt.Sprintf(`
/** 懒加载%s树形选项的子节点。 */
async function %s(node: { level: number; value?: string | number; data?: { value?: string | number; label?: string } }, resolve: (data: ProFormOption[]) => void) {
  const parentId = node.level === 0 ? 0 : Number(node.data?.value ?? node.value ?? 0);
  const response = await %s.%s({ %q: parentId, lazy: true } as Parameters<typeof %s.%s>[0]);
  resolve(normalizeLazyTreeOptions((response.list ?? []) as GeneratedTreeOption[], String(node.data?.label ?? "")));
}
`, variable, frontendOptionLoaderVar(column, scope.name), serviceName, method.MethodName, parentColumn, serviceName, method.MethodName))
			}
		}
	}
	builder.WriteString("\n/** 加载表单选择项。 */\nasync function loadFormOptions() {\n")
	for _, column := range columns {
		for _, scope := range frontendOptionScopes(column) {
			if scope.option.SourceType != OptionSourceTable {
				continue
			}
			method := findOptionMethodForConfig(scope.option, methods)
			if method == nil {
				continue
			}
			variable := frontendOptionVar(column, scope.name)
			serviceName := "def" + method.TargetEntityName + "Service"
			request := "{}"
			if scope.option.Kind == APIKindTree {
				lazy := scope.name == "form" && scope.option.Lazy
				request = fmt.Sprintf("{ %q: 0, lazy: %t }", DefaultString(method.ParentColumn, "parent_id"), lazy)
			}
			builder.WriteString(fmt.Sprintf("  const %sResponse = await %s.%s(%s as Parameters<typeof %s.%s>[0]);\n", variable, serviceName, method.MethodName, request, serviceName, method.MethodName))
			if column.Name == DefaultString(method.ParentColumn, "parent_id") && (column.FormComponent == "tree-select" || column.ListComponent == "tree-select" || column.QueryComponent == "tree-select") {
				if scope.name == "form" && scope.option.Kind == APIKindTree && scope.option.Lazy {
					builder.WriteString(fmt.Sprintf("  %sOptions.value = [{ label: t(\"common.value.top_level\"), value: 0 }, ...normalizeLazyTreeOptions((%sResponse.list ?? []) as GeneratedTreeOption[]).filter(option => Number(option.value) !== 0)];\n", variable, variable))
				} else if scope.option.Kind == APIKindTree {
					builder.WriteString(fmt.Sprintf("  %sOptions.value = [{ label: t(\"common.value.top_level\"), value: 0 }, ...transformTreeOptionPaths((%sResponse.list ?? []) as ProFormOption[]).filter(option => Number(option.value) !== 0)];\n", variable, variable))
				} else {
					builder.WriteString(fmt.Sprintf("  %sOptions.value = [{ label: t(\"common.value.top_level\"), value: 0 }, ...((%sResponse.list ?? []) as ProFormOption[]).filter(option => Number(option.value) !== 0)];\n", variable, variable))
				}
				continue
			}
			if scope.name == "form" && scope.option.Kind == APIKindTree && scope.option.Lazy {
				builder.WriteString(fmt.Sprintf("  %sOptions.value = normalizeLazyTreeOptions((%sResponse.list ?? []) as GeneratedTreeOption[]);\n", variable, variable))
			} else if scope.option.Kind == APIKindTree {
				builder.WriteString(fmt.Sprintf("  %sOptions.value = transformTreeOptionPaths((%sResponse.list ?? []) as ProFormOption[]);\n", variable, variable))
			} else {
				builder.WriteString(fmt.Sprintf("  %sOptions.value = (%sResponse.list ?? []) as ProFormOption[];\n", variable, variable))
			}
		}
	}
	builder.WriteString("}\n\nvoid loadFormOptions();\n")
	return builder.String()
}

// renderFrontendOptionImports 渲染字段选项依赖的外部 API import。
func (c *renderer) renderFrontendOptionImports(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	imports := make([]string, 0)
	seen := make(map[string]struct{})
	for _, column := range columns {
		for _, scope := range frontendOptionScopes(column) {
			if scope.option.SourceType != OptionSourceTable {
				continue
			}
			method := findOptionMethodForConfig(scope.option, methods)
			if method == nil || method.TargetEntityName == table.EntityName {
				continue
			}
			if _, ok := seen[method.TargetEntityName]; ok {
				continue
			}
			seen[method.TargetEntityName] = struct{}{}
			imports = append(imports, fmt.Sprintf("import { def%sService } from %q;", method.TargetEntityName, frontendAPIImportPathForMethod(method)))
		}
	}
	return strings.Join(imports, "\n")
}

// renderFrontendEnumImports 渲染状态枚举导入。
func (c *renderer) renderFrontendEnumImports(table *Table, columns []*CodeGenColumn) string {
	enumNames := make([]string, 0)
	seen := make(map[string]struct{})
	for _, column := range statusColumns(columns) {
		if !statusUsesEnumImport(column) {
			continue
		}
		if _, ok := seen[column.StatusEnumName]; ok {
			continue
		}
		seen[column.StatusEnumName] = struct{}{}
		enumNames = append(enumNames, column.StatusEnumName)
	}
	if len(enumNames) == 0 {
		return ""
	}
	return fmt.Sprintf("import { %s } from %q;", strings.Join(enumNames, ", "), ProtoTargetForTable(table).FrontendPackageName+"/rpc/common/v1/enum")
}

// renderFrontendLoadOptionsCall 渲染打开弹窗时加载选项调用。
func (c *renderer) renderFrontendLoadOptionsCall(columns []*CodeGenColumn, _ []*Proto) string {
	for _, column := range columns {
		for _, scope := range frontendOptionScopes(column) {
			if scope.option.SourceType == OptionSourceTable {
				return "  await loadFormOptions();\n"
			}
		}
	}
	return ""
}

// renderFrontendStatusHandlers 渲染全部状态切换处理方法。
func (c *renderer) renderFrontendStatusHandlers(table *Table, columns []*CodeGenColumn, methods []*Proto) string {
	statusColumnList := statusColumns(columns)
	var builder strings.Builder
	for _, column := range statusColumnList {
		method := findStatusMethodForColumn(column, methods)
		if column.StatusSwitch != 1 || method == nil {
			continue
		}
		builder.WriteString(c.renderFrontendStatusHandler(table, column, method, statusHandlerName(column, len(statusColumnList))))
	}
	return builder.String()
}

// renderFrontendStatusHandler 渲染单个状态切换处理方法。
func (c *renderer) renderFrontendStatusHandler(table *Table, column *CodeGenColumn, statusMethod *Proto, handlerName string) string {
	if column == nil || column.StatusSwitch != 1 || statusMethod == nil {
		return ""
	}
	enabled := statusValueExpression(column, column.StatusEnabledValue, "1")
	disabled := statusValueExpression(column, column.StatusDisabledValue, "2")
	return fmt.Sprintf(`
/**
 * 切换%s状态前先确认并调用后端状态接口。
 */
async function %s(row: %s) {
  const currentStatus = (row as unknown as Record<string, unknown>)["%s"];
  const nextStatus = currentStatus === %s ? %s : %s;
  const text = t(nextStatus === %s ? "common.status.enabled" : "common.status.disabled");
  try {
    await ElMessageBox.confirm(t("common.dialog.status_change", { action: text, resource: t(localeKeyPrefix + ".resource") }), t("common.title.notice"), {
      confirmButtonText: t("common.action.confirm"),
      cancelButtonText: t("common.action.cancel"),
      type: "warning"
    });
    await def%sService.%s({ id: row.id, status: nextStatus as %sRequest["status"] });
    ElMessage.success(t("common.message.status_success", { action: text }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}
	`, DefaultString(column.Comment, column.Name), handlerName, table.EntityName, column.Name, enabled, disabled, enabled, enabled, table.EntityName, statusMethod.MethodName, statusMethod.MethodName)
}

// removeFrontendFunctionBlock 移除并返回指定 TypeScript 函数及其中文注释。
func removeFrontendFunctionBlock(content string, functionName string) (string, string, bool) {
	functionStart := strings.Index(content, "async function "+functionName+"(")
	plainStart := strings.Index(content, "function "+functionName+"(")
	if functionStart < 0 || plainStart >= 0 && plainStart < functionStart {
		functionStart = plainStart
	}
	if functionStart < 0 {
		return content, "", false
	}

	lineStart := strings.LastIndex(content[:functionStart], "\n") + 1
	blockStart := strings.LastIndex(content[:lineStart], "/**")
	if blockStart < 0 {
		blockStart = lineStart
	} else {
		blockStart = strings.LastIndex(content[:blockStart], "\n") + 1
	}
	functionEnd := findTSFunctionEndIndex(content, functionStart)
	if functionEnd < 0 {
		return content, "", false
	}
	blockEnd := functionEnd + 1
	if blockEnd < len(content) && content[blockEnd] == '\n' {
		blockEnd++
	}
	return content[:blockStart] + content[blockEnd:], content[blockStart:blockEnd], true
}

// frontendFunctionEndWithLineBreak 返回指定 TypeScript 函数末尾的下一个行首位置。
func frontendFunctionEndWithLineBreak(content string, functionName string) int {
	functionStart := strings.Index(content, "async function "+functionName+"(")
	plainStart := strings.Index(content, "function "+functionName+"(")
	if functionStart < 0 || plainStart >= 0 && plainStart < functionStart {
		functionStart = plainStart
	}
	if functionStart < 0 {
		return -1
	}
	functionEnd := findTSFunctionEndIndex(content, functionStart)
	if functionEnd < 0 {
		return -1
	}
	if functionEnd+1 < len(content) && content[functionEnd+1] == '\n' {
		return functionEnd + 2
	}
	return functionEnd + 1
}

// findTSFunctionEndIndex 查找 TypeScript 函数体结束位置。
func findTSFunctionEndIndex(content string, functionStart int) int {
	if functionStart < 0 {
		return -1
	}
	openIndex := strings.Index(content[functionStart:], "{")
	if openIndex < 0 {
		return -1
	}
	openIndex += functionStart
	depth := 0
	for index := openIndex; index < len(content); index++ {
		switch content[index] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index
			}
		}
	}
	return -1
}

// validateCodeGenOutputPathLayout 校验输出路径仍位于生成代码可编译和可加载的目录中。
func validateCodeGenOutputPathLayout(target ProtoTarget, paths *adminv1.CodeGenOutputPaths) error {
	protoDirectory := filepath.ToSlash(filepath.Join(ProtoRootPath, target.Directory))
	if filepath.Dir(paths.GetProtoFilePath()) != protoDirectory || filepath.Ext(paths.GetProtoFilePath()) != ".proto" {
		return errorsx.InvalidArgument("Proto文件必须位于所选Proto目录且使用.proto扩展名")
	}
	if filepath.Dir(paths.GetBackendBizFilePath()) != target.BackendBizDirectory || filepath.Ext(paths.GetBackendBizFilePath()) != ".go" {
		return errorsx.InvalidArgument("后端Biz文件必须位于所选Proto目录对应的服务目录且使用.go扩展名")
	}
	if filepath.Dir(paths.GetBackendServiceFilePath()) != target.BackendModuleDirectory || filepath.Ext(paths.GetBackendServiceFilePath()) != ".go" {
		return errorsx.InvalidArgument("后端Service文件必须位于所选Proto目录对应的服务目录且使用.go扩展名")
	}
	if filepath.Dir(paths.GetFrontendApiFilePath()) != target.FrontendAPIDirectory || filepath.Ext(paths.GetFrontendApiFilePath()) != ".ts" {
		return errorsx.InvalidArgument("前端API文件必须位于所选业务模块目录且使用.ts扩展名")
	}
	if !strings.HasPrefix(paths.GetFrontendPageFilePath(), target.FrontendPageDirectory+"/") || !strings.HasSuffix(paths.GetFrontendPageFilePath(), "/index.vue") {
		return errorsx.InvalidArgument("前端页面文件必须位于所选业务模块目录且文件名为index.vue")
	}
	return nil
}

// frontendRPCImportPath 根据 Proto 文件路径推导前端生成类型导入路径。
func frontendRPCImportPath(protoPath string) string {
	relativePath := strings.TrimPrefix(protoPath, "backend/api/proto/")
	target, ok := ProtoTargetForProtoPath(protoPath)
	if !ok {
		return ""
	}
	return target.FrontendPackageName + "/rpc/" + strings.TrimSuffix(relativePath, filepath.Ext(relativePath))
}

// frontendAPIImportPathForMethod 根据接口所属 Proto 目标推导前端 API 导入路径。
func frontendAPIImportPathForMethod(method *Proto) string {
	protoDirectory := strings.TrimPrefix(filepath.ToSlash(filepath.Dir(method.ProtoFilePath)), ProtoRootPath+"/")
	module := strings.TrimSuffix(strings.TrimSuffix(protoDirectory, "/v1"), "/admin")
	target, _ := ProtoTargetForBusinessModule(module)
	return target.FrontendPackageName + "/api/" + module + "/" + stringcase.ToSnakeCase(method.TargetEntityName)
}

// repoRoot 返回仓库根目录。
func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	// 后端命令通常在 backend 目录执行，这里回到仓库根目录。
	if filepath.Base(wd) == "backend" {
		return filepath.Dir(wd)
	}
	return wd
}

// pluralize 生成项目常用复数方法名。
func pluralize(value string) string {
	if strings.HasSuffix(value, "s") {
		return value + "es"
	}
	if strings.HasSuffix(value, "y") {
		return strings.TrimSuffix(value, "y") + "ies"
	}
	return value + "s"
}

// findStatusColumn 根据字段名查找状态字段。
func findStatusColumn(columns []*CodeGenColumn, columnName string) *CodeGenColumn {
	if columnName != "" {
		for _, column := range columns {
			if column.Name == columnName && column.IsStatusField == 1 {
				return column
			}
		}
	}
	statusColumnList := statusColumns(columns)
	if len(statusColumnList) == 0 {
		return nil
	}
	return statusColumnList[0]
}

// statusProtoType 返回状态字段在 Proto 中的类型。
func statusProtoType(column *CodeGenColumn) string {
	if column == nil {
		return "int32"
	}
	return DefaultString(column.ProtoType, InferProtoType(column.DbType))
}

// statusResourcePath 返回状态接口对应的 HTTP 子资源路径。
func statusResourcePath(table *Table, method *Proto) string {
	if method.MethodName == "Set"+table.EntityName+"Status" || method.Name == "" {
		return "status"
	}
	return "status/" + strings.ReplaceAll(method.Name, "_", "-")
}

// renderFrontendDateImport 仅在列表包含日期展示列时引入前端格式化依赖。
func renderFrontendDateImport(columns []*CodeGenColumn) string {
	for _, column := range columns {
		if generatedListIncludesColumn(column) && column.IsPrimary != 1 && column.Name != "deleted_at" && column.ListComponent == "date" &&
			(column.IsStatusField != 1 || column.StatusTableColumn != 1 && column.StatusSearch != 1) {
			return `import dayjs from "dayjs";`
		}
	}
	return ""
}

// renderFrontendColumn 渲染前端表格列配置。
func renderFrontendColumn(localePrefix string, permissionPrefix string, entityName string, column *CodeGenColumn, statusMethod *Proto, statusColumnCount int, align string) string {
	alignConfig := ""
	label := fmt.Sprintf("t(%q)", localePrefix+".field."+stringcase.ToSnakeCase(column.Name))
	if align != "" {
		alignConfig = fmt.Sprintf(`, align: %q`, align)
	}
	if column.IsStatusField == 1 && (column.StatusTableColumn == 1 || column.StatusSearch == 1) {
		search := ""
		if column.StatusSearch == 1 {
			search = ", search: { el: \"select\" }"
		}
		dictConfig := ""
		if statusUsesDictionary(column) {
			dictConfig = fmt.Sprintf(", dictCode: %q, dictValueType: %q", column.StatusDictCode, frontendDictValueType(column))
		}
		if column.StatusSwitch == 1 && statusMethod != nil {
			return fmt.Sprintf(`  {
    prop: "%s",
    label: %s%s,
    width: 100%s%s,
    cellType: "status",
    statusProps: {
      activeValue: %s,
      inactiveValue: %s,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value[%q],
      beforeChange: scope => %s(scope.row as %s)
    }
  }`, column.Name, label, alignConfig, dictConfig, search, statusValueExpression(column, column.StatusEnabledValue, "1"), statusValueExpression(column, column.StatusDisabledValue, "2"), statusPermissionPath(permissionPrefix, column.Name, statusColumnCount), statusHandlerName(column, statusColumnCount), entityName)
		}
		if statusUsesDictionary(column) {
			return fmt.Sprintf(`  { prop: "%s", label: %s%s%s%s }`, column.Name, label, alignConfig, dictConfig, search)
		}
		return fmt.Sprintf(`  { prop: "%s", label: %s%s, enum: %s.value%s }`, column.Name, label, alignConfig, statusOptionsVariable(column, statusColumnCount), search)
	}
	optionConfig := frontendColumnOptionConfig(column)
	searchConfig := frontendSearchConfig(column)
	if column.ListComponent == "image" {
		return fmt.Sprintf(`  { prop: "%s", label: %s%s, cellType: "image"%s%s }`, column.Name, label, alignConfig, optionConfig, searchConfig)
	}
	if column.ListComponent == "date" {
		dbType := strings.ToLower(DefaultString(column.ColumnType, column.DbType))
		format := "YYYY-MM-DD"
		if strings.Contains(dbType, "datetime") || strings.Contains(dbType, "timestamp") {
			format = "YYYY-MM-DD HH:mm:ss"
		} else if strings.Contains(dbType, "time") {
			format = "HH:mm:ss"
		}
		return fmt.Sprintf(`  { prop: "%s", label: %s%s, render: scope => scope.row.%s ? dayjs(scope.row.%s).format(%q) : "--"%s%s }`, column.Name, label, alignConfig, column.Name, column.Name, format, optionConfig, searchConfig)
	}
	return fmt.Sprintf(`  { prop: "%s", label: %s%s%s%s }`, column.Name, label, alignConfig, optionConfig, searchConfig)
}

// frontendColumnOptionConfig 渲染列表或查询选择组件的数据源配置。
func frontendColumnOptionConfig(column *CodeGenColumn) string {
	if !generatedListIncludesColumn(column) || column.IsStatusField == 1 || !isSelectComponent(column.ListComponent) {
		return ""
	}
	option := column.ListOption
	switch option.SourceType {
	case OptionSourceDict:
		if option.SourceValue != "" {
			return fmt.Sprintf(", dictCode: %q, dictValueType: %q", option.SourceValue, frontendDictValueType(column))
		}
	case OptionSourceStatic, OptionSourceTable:
		if option.Kind != "" {
			return ", enum: " + frontendOptionVar(column, "list") + "Options.value"
		}
	}
	return ""
}

// frontendSearchConfig 渲染普通查询字段的搜索配置。
func frontendSearchConfig(column *CodeGenColumn) string {
	if !generatedQueryIncludesColumn(column) {
		return ""
	}
	component := DefaultString(column.QueryComponent, "input")
	if component == "date-picker" && effectiveQueryOperator(column) == "between" {
		return `, search: { el: "date-picker", props: { type: "daterange", editable: false, valueFormat: "YYYY-MM-DD" } }`
	}
	if component == "input-number" {
		return `, search: { el: "input-number" }`
	}
	if component == "select" || component == "tree-select" {
		option := column.QueryOption
		if isTenantQueryOption(column) {
			return `, search: { el: "select", key: "tenant_id", props: { filterable: true }, enum: requestTenantOptions }`
		}
		switch option.SourceType {
		case OptionSourceDict:
			return fmt.Sprintf(`, search: { el: "%s", dictCode: %q, dictValueType: %q }`, component, option.SourceValue, frontendDictValueType(column))
		case OptionSourceStatic, OptionSourceTable:
			if option.Kind != "" {
				return fmt.Sprintf(`, search: { el: "%s", enum: %sOptions.value }`, component, frontendOptionVar(column, "query"))
			}
		}
		return fmt.Sprintf(`, search: { el: "%s" }`, component)
	}
	if component == "date-picker" {
		return `, search: { el: "date-picker" }`
	}
	return `, search: { el: "input" }`
}

// statusOptionsVariable 返回状态字段对应的前端选项变量名。
func statusOptionsVariable(column *CodeGenColumn, statusColumnCount int) string {
	if statusColumnCount <= 1 {
		return "statusOptions"
	}
	return stringcase.ToCamelCase(column.Name) + "StatusOptions"
}

// renderFrontendStaticOptions 将静态选项 JSON 渲染为安全的 TypeScript 字面量。
func renderFrontendStaticOptions(table *Table, column *CodeGenColumn, option CodeGenColumnOptionConfig) string {
	options := parseCodeGenStaticOptions(option)
	if len(options) == 0 {
		return "[]"
	}
	items := make([]string, 0, len(options))
	for _, option := range options {
		value, err := json.Marshal(option.Value)
		if err != nil {
			continue
		}
		items = append(items, fmt.Sprintf("{ label: t(%q), value: %s }", frontendStaticOptionLocaleKey(table, column, option.Value), value))
	}
	return "[" + strings.Join(items, ", ") + "]"
}

// renderFrontendStatusTypeImports 渲染状态切换处理函数依赖的请求类型。
func renderFrontendStatusTypeImports(columns []*CodeGenColumn, methods []*Proto) string {
	typeNames := make([]string, 0)
	for _, column := range statusColumns(columns) {
		if column.StatusSwitch != 1 {
			continue
		}
		method := findStatusMethodForColumn(column, methods)
		if method != nil {
			typeNames = append(typeNames, method.MethodName+"Request")
		}
	}
	if len(typeNames) == 0 {
		return ""
	}
	return ", " + strings.Join(typeNames, ", ")
}

// statusColumns 返回全部状态字段。
func statusColumns(columns []*CodeGenColumn) []*CodeGenColumn {
	list := make([]*CodeGenColumn, 0)
	for _, column := range columns {
		if column.IsStatusField == 1 {
			list = append(list, column)
		}
	}
	return list
}

// findStatusMethodForColumn 查找状态字段对应的接口方法。
func findStatusMethodForColumn(column *CodeGenColumn, methods []*Proto) *Proto {
	var onlyStatusMethod *Proto
	statusMethodCount := 0
	for _, method := range methods {
		if method.APIKind != APIKindStatus {
			continue
		}
		statusMethodCount++
		onlyStatusMethod = method
		if method.Name == column.Name {
			return method
		}
	}
	if statusMethodCount == 1 {
		return onlyStatusMethod
	}
	return nil
}

// statusHandlerName 返回状态字段对应的前端切换处理函数名。
func statusHandlerName(column *CodeGenColumn, statusColumnCount int) string {
	if statusColumnCount <= 1 {
		return "handleBeforeSetStatus"
	}
	return "handleBeforeSet" + stringcase.ToPascalCase(column.Name) + "Status"
}

// statusNeedsFrontendOptions 判断状态字段是否需要生成前端静态选项。
func statusNeedsFrontendOptions(column *CodeGenColumn) bool {
	return column != nil && column.IsStatusField == 1 && !statusUsesDictionary(column) && (column.StatusTableColumn == 1 || column.StatusSearch == 1 || column.StatusForm == 1)
}

// statusUsesDictionary 判断状态选项是否来自字典。
func statusUsesDictionary(column *CodeGenColumn) bool {
	return column != nil && column.IsStatusField == 1 && column.StatusDataType == "dict"
}

// frontendOptionScope 表示字段在单个前端作用域中使用的选项配置。
type frontendOptionScope struct {
	name   string
	option CodeGenColumnOptionConfig
}

// frontendOptionScopes 返回前端实际消费的查询、列表和表单选项配置。
func frontendOptionScopes(column *CodeGenColumn) []frontendOptionScope {
	scopes := make([]frontendOptionScope, 0, 3)
	if frontendQueryUsesOptions(column) && column.QueryOption.Kind != "" && !isTenantQueryOption(column) {
		scopes = append(scopes, frontendOptionScope{name: "query", option: column.QueryOption})
	}
	if generatedListIncludesColumn(column) && column.Name != "tenant_id" && column.IsStatusField != 1 && isSelectComponent(column.ListComponent) && column.ListOption.Kind != "" {
		scopes = append(scopes, frontendOptionScope{name: "list", option: column.ListOption})
	}
	if generatedFormIncludesColumn(column) && isSelectComponent(column.FormComponent) && column.FormOption.Kind != "" {
		scopes = append(scopes, frontendOptionScope{name: "form", option: column.FormOption})
	}
	return scopes
}

// frontendQueryUsesOptions 判断普通查询字段是否使用选择型组件。
func frontendQueryUsesOptions(column *CodeGenColumn) bool {
	if column.IsStatusField == 1 || !generatedQueryIncludesColumn(column) {
		return false
	}
	return column.QueryComponent == "select" || column.QueryComponent == "tree-select"
}

// hasTenantQueryOption 判断当前生成对象是否配置了租户查询选项。
func hasTenantQueryOption(columns []*CodeGenColumn) bool {
	for _, column := range columns {
		if isTenantQueryOption(column) {
			return true
		}
	}
	return false
}

// hasTenantListColumn 判断当前生成对象是否配置了租户列表列。
func hasTenantListColumn(columns []*CodeGenColumn) bool {
	column := FindColumnByName(columns, "tenant_id")
	return column != nil && generatedListIncludesColumn(column)
}

// isTenantQueryOption 判断字段是否使用 base_tenant 数据表作为租户查询选项。
func isTenantQueryOption(column *CodeGenColumn) bool {
	if column == nil || column.Name != "tenant_id" || !generatedQueryIncludesColumn(column) {
		return false
	}
	if column.QueryComponent != "select" && column.QueryComponent != "tree-select" {
		return false
	}
	option := column.QueryOption
	return option.Kind == APIKindOption &&
		option.SourceType == OptionSourceTable &&
		strings.EqualFold(stringcase.ToSnakeCase(option.SourceValue), "base_tenant")
}

// generatedRequestIncludesColumn 判断请求是否需要包含显式查询字段或左树关联字段。
func generatedRequestIncludesColumn(table *Table, column *CodeGenColumn) bool {
	if generatedQueryIncludesColumn(column) {
		return true
	}
	leftTreeConfig := LeftTreeConfigFromTable(table)
	return leftTreeConfig.Enabled && leftTreeConfig.FilterColumn != "" && column.Name == leftTreeConfig.FilterColumn
}

// codeGenRequestColumns 为后端查询模板补入左树隐式关联字段。
func codeGenRequestColumns(table *Table, columns []*CodeGenColumn) []*CodeGenColumn {
	leftTreeConfig := LeftTreeConfigFromTable(table)
	if !leftTreeConfig.Enabled || leftTreeConfig.FilterColumn == "" {
		return columns
	}
	list := make([]*CodeGenColumn, 0, len(columns))
	for _, column := range columns {
		if column.Name != leftTreeConfig.FilterColumn || generatedQueryIncludesColumn(column) {
			list = append(list, column)
			continue
		}
		requestColumn := *column
		requestColumn.IsQuery = 1
		requestColumn.QueryOperator = "eq"
		list = append(list, &requestColumn)
	}
	return list
}

// generatedQueryIncludesColumn 判断查询请求是否包含字段。
func generatedQueryIncludesColumn(column *CodeGenColumn) bool {
	return column.IsQuery == 1 || column.IsStatusField == 1 && column.StatusSearch == 1
}

// generatedListIncludesColumn 判断列表响应是否包含字段。
func generatedListIncludesColumn(column *CodeGenColumn) bool {
	return column.IsList == 1 || column.IsStatusField == 1 && column.StatusTableColumn == 1
}

// generatedFormIncludesColumn 判断新增编辑表单是否包含字段。
func generatedFormIncludesColumn(column *CodeGenColumn) bool {
	return column.IsForm == 1 || column.IsStatusField == 1 && column.StatusForm == 1 || column.Name == "code" && column.IsPrimary != 1
}

// generatedFormRequired 判断生成表单字段是否必须填写。
func generatedFormRequired(column *CodeGenColumn) bool {
	return column.IsRequired == 1 || column.Name == "code"
}

// frontendDictValueType 返回前端字典组件使用的值类型。
func frontendDictValueType(column *CodeGenColumn) string {
	if DefaultString(column.TsType, InferTSType(column.DbType)) == "string" {
		return "string"
	}
	return "number"
}

// frontendDefaultValue 返回前端表单默认值表达式。
func frontendDefaultValue(column *CodeGenColumn) string {
	if column.IsPrimary == 1 {
		return "0"
	}
	if column.IsStatusField == 1 {
		defaultValue := DefaultString(column.StatusDefaultValue, column.StatusEnabledValue)
		return statusValueExpression(column, defaultValue, "1")
	}
	if isFormTreeMultiple(column) {
		return "[]"
	}
	// 树形页面的父节点使用 0 表示顶级节点，避免 Proto JSON 省略零值后编辑态变成未选择。
	if column.FormComponent == "tree-select" && (column.Name == "parent_id" || strings.HasPrefix(column.Name, "parent_")) {
		return "0"
	}
	// 关联、字典等选择型字段未选择时不能传递数值零值。
	if isSelectComponent(DefaultString(column.FormComponent, "input")) {
		return "undefined"
	}
	if column.DefaultValue != "" {
		if DefaultString(column.TsType, "string") == "string" {
			return fmt.Sprintf("%q", column.DefaultValue)
		}
		return column.DefaultValue
	}
	switch DefaultString(column.TsType, InferTSType(column.DbType)) {
	case "number":
		return "0"
	case "boolean":
		return "false"
	default:
		return "\"\""
	}
}

// isFormTreeMultiple 判断字段是否使用 JSON 存储的多选树形选择。
func isFormTreeMultiple(column *CodeGenColumn) bool {
	return column != nil && column.FormMultiple && column.FormComponent == "tree-select" && strings.EqualFold(column.DbType, "json")
}

// statusValueExpression 渲染状态值表达式。
func statusValueExpression(column *CodeGenColumn, value string, fallback string) string {
	if value == "" {
		return fallback
	}
	if statusUsesEnumImport(column) && strings.Contains(value, ".") {
		return value
	}
	if statusUsesEnumImport(column) {
		return column.StatusEnumName + "." + value
	}
	if column != nil && DefaultString(column.TsType, "number") == "string" {
		return fmt.Sprintf("%q", value)
	}
	return value
}

// statusUsesEnumImport 判断状态值是否需要从公共枚举导入。
func statusUsesEnumImport(column *CodeGenColumn) bool {
	return column != nil && column.StatusDataType == "enum" && column.StatusEnumName != ""
}

// frontendOptionVar 返回字段指定作用域的选择项状态变量前缀。
func frontendOptionVar(column *CodeGenColumn, scope string) string {
	return stringcase.ToCamelCase(column.Name) + stringcase.ToPascalCase(scope)
}

// frontendOptionLoaderVar 返回字段指定作用域的树形懒加载函数名。
func frontendOptionLoaderVar(column *CodeGenColumn, scope string) string {
	return "load" + stringcase.ToPascalCase(frontendOptionVar(column, scope)) + "TreeOptions"
}

// isSelectComponent 判断组件是否属于选择型控件。
func isSelectComponent(component string) bool {
	return component == "select" || component == "tree-select" || component == "radio-group" || component == "checkbox-group" || component == "dict" || component == "segmented" || component == "transfer"
}

// findOptionMethodForConfig 查找选项配置对应的接口。
func findOptionMethodForConfig(option CodeGenColumnOptionConfig, methods []*Proto) *Proto {
	targetEntity := stringcase.ToPascalCase(option.SourceValue)
	apiKind := APIKindOption
	if option.Kind == APIKindTree {
		apiKind = APIKindTree
	}
	for _, method := range methods {
		if method.TargetEntityName == targetEntity && method.APIKind == apiKind {
			return method
		}
	}
	return nil
}

// isBoolDBType 判断数据库类型是否表示布尔值。
func isBoolDBType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return strings.Contains(lowerType, "bool") || strings.Contains(lowerType, "tinyint(1)")
}

// isNumericDBType 判断数据库类型是否表示数值。
func isNumericDBType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return strings.Contains(lowerType, "int") ||
		strings.Contains(lowerType, "decimal") ||
		strings.Contains(lowerType, "float") ||
		strings.Contains(lowerType, "double")
}

// isDateTimeDBType 判断数据库类型是否表示日期时间。
func isDateTimeDBType(dbType string) bool {
	lowerType := strings.ToLower(dbType)
	return strings.Contains(lowerType, "date") || strings.Contains(lowerType, "time")
}

// dedupeProtoChecks 去重 Proto 检查项。
func dedupeProtoChecks(checks []*ProtoCheck) []*ProtoCheck {
	list := make([]*ProtoCheck, 0, len(checks))
	seen := make(map[string]int, len(checks))
	for _, check := range checks {
		key := check.ProtoFilePath + ":" + check.TargetEntityName + ":" + check.MethodName
		index, ok := seen[key]
		if ok {
			// 同一实体的树形选项接口比普通选项接口提供更多父子节点信息，优先保留树形定义。
			if list[index].APIKind != APIKindTree && check.APIKind == APIKindTree {
				list[index] = check
			}
			continue
		}
		seen[key] = len(list)
		list = append(list, check)
	}
	slices.SortFunc(list, func(a *ProtoCheck, b *ProtoCheck) int {
		if a.ProtoFilePath != b.ProtoFilePath {
			return strings.Compare(a.ProtoFilePath, b.ProtoFilePath)
		}
		if a.TargetEntityName != b.TargetEntityName {
			return strings.Compare(a.TargetEntityName, b.TargetEntityName)
		}
		aPosition := codeGenProtoMethodPosition(a.APIKind, a.TriggerType, a.MethodName)
		bPosition := codeGenProtoMethodPosition(b.APIKind, b.TriggerType, b.MethodName)
		if aPosition != bPosition {
			return aPosition - bPosition
		}
		return strings.Compare(a.MethodName, b.MethodName)
	})
	return list
}

// findSavedProtoMethod 查找已保存的生成选择。
func findSavedProtoMethod(methods []*Proto, check *ProtoCheck) *Proto {
	for _, method := range methods {
		if method.ProtoFilePath != check.ProtoFilePath || method.TargetEntityName != check.TargetEntityName {
			continue
		}
		if method.MethodName == check.MethodName {
			return method
		}
	}
	return nil
}

// FrontendPageComponentPath 根据页面文件路径推导动态路由组件路径。
func FrontendPageComponentPath(path string) string {
	path = filepath.ToSlash(filepath.Clean(path))
	prefix := "frontend/admin/packages/modules/"
	relativePath := strings.TrimPrefix(path, prefix)
	if relativePath == path {
		return ""
	}
	parts := strings.SplitN(relativePath, "/src/views/", 2)
	if len(parts) != 2 || parts[0] == "" || !strings.HasSuffix(parts[1], "/index.vue") && parts[1] != "index.vue" {
		return ""
	}
	return strings.TrimSuffix(parts[0]+"/"+parts[1], "/index.vue")
}

// SafeRepoFilePath 返回仓库内安全文件路径。
func SafeRepoFilePath(path string) (string, error) {
	if path == "" {
		return "", errorsx.InvalidArgument("文件路径不允许为空")
	}
	if filepath.IsAbs(path) {
		return "", errorsx.InvalidArgument("文件路径不允许使用绝对路径")
	}
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", errorsx.InvalidArgument("文件路径不允许跳出仓库目录")
	}

	root, err := filepath.Abs(repoRoot())
	if err != nil {
		return "", err
	}
	fullPath := filepath.Join(root, cleanPath)
	var relPath string
	relPath, err = filepath.Rel(root, fullPath)
	if err != nil {
		return "", err
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", errorsx.InvalidArgument("文件路径不允许跳出仓库目录")
	}
	return fullPath, nil
}

// BackendDir 返回当前仓库的后端目录。
func BackendDir() string {
	return filepath.Join(repoRoot(), "backend")
}

// BoolToInt32 转换布尔数据库值。
func BoolToInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

// DefaultString 返回默认字符串。
func DefaultString(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// PermissionPrefix 返回页面权限前缀。
func PermissionPrefix(table *Table) string {
	if table.PermissionPrefix != "" {
		return table.PermissionPrefix
	}
	return strings.ReplaceAll(strings.Trim(table.ModulePath, "/"), "/", ":") + ":" + stringcase.ToSnakeCase(table.EntityName)
}

// IsOptionProtoMethod 判断方法是否是选择项接口。
func IsOptionProtoMethod(method *Proto) bool {
	return method.APIKind == APIKindOption || method.APIKind == APIKindTree && (method.TriggerType == TriggerEntityOption || method.TriggerType == TriggerFieldOption || method.TriggerType == TriggerLeftTree)
}

// InferTSType 按数据库类型推断 TS 类型。
func InferTSType(dbType string) string {
	protoType := InferProtoType(dbType)
	if protoType == "bool" {
		return "boolean"
	}
	if protoType == "int64" || protoType == "int32" || protoType == "double" {
		return "number"
	}
	return "string"
}

// InferGoType 按数据库类型推断 Go 类型。
func InferGoType(dbType string) string {
	if isBoolDBType(dbType) {
		return "bool"
	}
	lowerType := strings.ToLower(dbType)
	if strings.Contains(lowerType, "bigint") {
		return "int64"
	}
	if isNumericDBType(lowerType) {
		if strings.Contains(lowerType, "decimal") || strings.Contains(lowerType, "float") || strings.Contains(lowerType, "double") {
			return "float64"
		}
		return "int32"
	}
	if isDateTimeDBType(lowerType) {
		return "time.Time"
	}
	return "string"
}

// InferProtoType 按数据库类型推断 Proto 类型。
func InferProtoType(dbType string) string {
	lowerType := strings.ToLower(dbType)
	if isBoolDBType(lowerType) {
		return "bool"
	}
	if strings.Contains(lowerType, "bigint") {
		return "int64"
	}
	if strings.Contains(lowerType, "int") {
		return "int32"
	}
	if strings.Contains(lowerType, "decimal") || strings.Contains(lowerType, "float") || strings.Contains(lowerType, "double") {
		return "double"
	}
	return "string"
}

// renderFrontendPasswordReset 返回编辑态清空密码字段的语句，避免回填后端密文。
func renderFrontendPasswordReset(columns []*CodeGenColumn) string {
	lines := make([]string, 0)
	for _, column := range columns {
		if column == nil || column.FormComponent != "password" {
			continue
		}
		lines = append(lines, fmt.Sprintf("  formData[%q] = \"\";", column.Name))
	}
	return strings.Join(lines, "\n")
}

// treePageRequestExpression 返回树形列表接口的请求参数表达式。
func treePageRequestExpression(table *Table, method *Proto) string {
	if table.PageType != PageTypeTreeLazy {
		return "params"
	}
	parentColumn := DefaultString(method.ParentColumn, DefaultString(table.ParentColumn, "parent_id"))
	return fmt.Sprintf("{ ...params, %q: params.%s ?? 0, lazy: true }", parentColumn, parentColumn)
}

// frontendFieldLocaleKey 返回生成字段的稳定语言键。
func frontendFieldLocaleKey(table *Table, column *CodeGenColumn) string {
	return FrontendLocaleKeyPrefix(table) + ".field." + stringcase.ToSnakeCase(column.Name)
}

// renderFrontendFormStateType 渲染允许选择型字段暂未选择的表单编辑状态类型。
func renderFrontendFormStateType(table *Table, columns []*CodeGenColumn) string {
	optionalFields := make([]string, 0)
	for _, column := range columns {
		if (!generatedFormIncludesColumn(column) && column.IsPrimary != 1) || frontendDefaultValue(column) != "undefined" {
			continue
		}
		optionalFields = append(optionalFields, fmt.Sprintf("%q", column.Name))
	}
	if len(optionalFields) == 0 {
		return ""
	}

	entity := table.EntityName
	fieldUnion := strings.Join(optionalFields, " | ")
	return fmt.Sprintf(`/** %sFormState 表示%s表单编辑状态，选择型字段填写前允许为空。 */
type %sFormState = Omit<%sForm, %s> & Partial<Pick<%sForm, %s>>;
`, entity, table.BusinessName, entity, entity, fieldUnion, entity, fieldUnion)
}

// frontendFormFieldDisabledPrefix 返回租户字段的编辑态禁用配置。
func frontendFormFieldDisabledPrefix(column *CodeGenColumn) string {
	if column == nil || column.Name != "tenant_id" {
		return ""
	}
	return "disabled: Boolean(formData.id) || treeCreateState.lockTenant, "
}
