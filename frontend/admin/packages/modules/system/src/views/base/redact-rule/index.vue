<template>
  <div class="table-box page-box">
    <ProTable ref="table" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog v-model="dialog.visible" ref="dialogRef" :title="t(dialog.titleKey)" width="620px" :model="form" :fields="fields" :rules="rules" @confirm="submit" @close="resetForm" />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseRedactRuleService } from "@liujitcn/kratos-admin-system/api/system/base_redact_rule";
import type { BaseRedactRule, BaseRedactRuleForm, PageBaseRedactRuleRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_rule";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineOptions({ name: "BaseRedactRule", inheritAttrs: false });
const { BUTTONS } = useAuthButtons();
const table = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ visible: false, titleKey: "common.action.create_resource" });
const form = reactive<BaseRedactRuleForm>({ id: 0, code: "", name: "", rule_type: "MASK", rule: '{"mask":{"keep_first":3,"keep_last":4,"mask_char":"*"}}', status: Status.STATUS_ENABLE, version: 1, remark: "" });
const statusOptions = computed<ProFormOption[]>(() => [{ label: t("common.status.enabled"), value: Status.STATUS_ENABLE }, { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }]);
const fields = computed<ProFormField[]>(() => [{ prop: "code", label: t("system.base.redact_policy.field.code"), component: "input" }, { prop: "name", label: t("system.base.redact_policy.field.name"), component: "input" }, { prop: "rule_type", label: t("system.base.redact_policy.field.rule_type"), component: "input" }, { prop: "rule", label: t("system.base.redact_policy.field.rule"), component: "textarea", props: { rows: 8 } }, { prop: "version", label: t("system.base.redact_policy.field.version"), component: "input-number", props: { min: 1, precision: 0, style: { width: "100%" } } }, { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value }, { prop: "remark", label: t("common.field.remark"), component: "textarea" }]);
const rules = computed(() => ({ code: [{ required: true, message: t("system.base.redact_policy.validation.code"), trigger: "blur" }], name: [{ required: true, message: t("system.base.redact_policy.validation.name"), trigger: "blur" }], rule_type: [{ required: true, message: t("system.base.redact_policy.validation.rule_type"), trigger: "blur" }], rule: [{ required: true, message: t("system.base.redact_policy.validation.rule"), trigger: "blur" }] }));
const columns = computed<ColumnProps[]>(() => [{ type: "selection", width: 55 }, { prop: "code", label: t("system.base.redact_policy.field.code"), minWidth: 160, search: { el: "input" } }, { prop: "name", label: t("system.base.redact_policy.field.name"), minWidth: 160, search: { el: "input" } }, { prop: "rule_type", label: t("system.base.redact_policy.field.rule_type"), width: 120, search: { el: "input" } }, { prop: "rule", label: t("system.base.redact_policy.field.rule"), minWidth: 240, showOverflowTooltip: true }, { prop: "version", label: t("system.base.redact_policy.field.version"), width: 80 }, { prop: "status", label: t("common.field.status"), width: 100, cellType: "status", statusProps: { activeValue: Status.STATUS_ENABLE, inactiveValue: Status.STATUS_DISABLE, activeText: t("common.status.enabled"), inactiveText: t("common.status.disabled"), disabled: () => !BUTTONS.value["base:redact-rule:status"], beforeChange: scope => changeStatus(scope.row as BaseRedactRule) } }, { prop: "operation", label: t("common.field.operation"), width: 140, fixed: "right", cellType: "actions", actions: [{ label: t("common.action.edit"), type: "primary", link: true, icon: EditPen, hidden: () => !BUTTONS.value["base:redact-rule:update"], onClick: scope => openDialog((scope.row as BaseRedactRule).id) }, { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !BUTTONS.value["base:redact-rule:delete"], onClick: scope => deleteItems(scope.row as BaseRedactRule) }] }]);
const headerActions = computed<HeaderActionProps[]>(() => [{ label: t("common.action.create"), type: "success", icon: CirclePlus, hidden: () => !BUTTONS.value["base:redact-rule:create"], onClick: () => openDialog() }, { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: () => !BUTTONS.value["base:redact-rule:delete"], disabled: scope => !scope.selectedList.length, onClick: scope => deleteItems(scope.selectedList as BaseRedactRule[]) }]);

/** 请求规则分页列表。 */
async function requestTable(params: PageBaseRedactRuleRequest) { const data = await defBaseRedactRuleService.PageBaseRedactRule(buildPageRequest(params)); return { data: { list: data.base_redact_rules ?? [], total: data.total } }; }
/** 打开规则编辑弹窗。 */
async function openDialog(id?: number) { resetForm(); dialog.titleKey = id ? "common.action.edit_resource" : "common.action.create_resource"; dialog.visible = true; if (id) Object.assign(form, await defBaseRedactRuleService.GetBaseRedactRule({ id })); }
/** 重置规则表单。 */
function resetForm() { dialog.visible = false; dialogRef.value?.resetFields(); Object.assign(form, { id: 0, code: "", name: "", rule_type: "MASK", rule: '{"mask":{"keep_first":3,"keep_last":4,"mask_char":"*"}}', status: Status.STATUS_ENABLE, version: 1, remark: "" }); }
/** 提交规则表单。 */
function submit() { dialogRef.value?.validate()?.then(async valid => { if (!valid) return; const payload = JSON.parse(JSON.stringify(form)) as BaseRedactRuleForm; if (payload.id) await defBaseRedactRuleService.UpdateBaseRedactRule({ base_redact_rule: payload }); else await defBaseRedactRuleService.CreateBaseRedactRule({ base_redact_rule: payload }); ElMessage.success(t(payload.id ? "common.message.update_success" : "common.message.create_success", { resource: t("system.base.redact_policy.tabs.rule") })); resetForm(); table.value?.getTableList(); }); }
/** 切换规则状态。 */
async function changeStatus(row: BaseRedactRule) { const next = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE; try { await ElMessageBox.confirm(t("common.dialog.status_change", { action: t(next === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled"), resource: t("system.base.redact_policy.tabs.rule"), field: t("system.base.redact_policy.field.code"), value: row.code })); await defBaseRedactRuleService.SetBaseRedactRuleStatus({ id: row.id, status: next }); table.value?.getTableList(); return true; } catch { return false; } }
/** 删除规则模板。 */
function deleteItems(selected?: BaseRedactRule | BaseRedactRule[] | number | string | Array<number | string>) { const items = Array.isArray(selected) ? selected.filter((item): item is BaseRedactRule => typeof item === "object") : selected && typeof selected === "object" ? [selected] : []; const ids = items.length ? items.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>); if (!ids.length) { ElMessage.warning(t("common.message.select_delete_item")); return; } ElMessageBox.confirm(t("common.dialog.delete_selected", { resource: t("system.base.redact_policy.tabs.rule") }), t("common.title.warning"), { type: "warning" }).then(async () => { await defBaseRedactRuleService.DeleteBaseRedactRule({ id: ids.join(",") }); ElMessage.success(t("common.message.delete_success", { resource: t("system.base.redact_policy.tabs.rule") })); table.value?.getTableList(); }); }
</script>

<style scoped lang="scss">
.page-box { padding: 20px; }
</style>
