<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog v-model="dialog.visible" ref="dialogRef" :title="t(dialog.titleKey)" width="560px" :model="form" :fields="fields" :rules="rules" @confirm="submit" @close="resetForm" />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, EnumProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseApiService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_api";
import { defBaseRedactFieldService } from "@liujitcn/kratos-admin-system/api/system/base_redact_field";
import { defBaseRedactPolicyService } from "@liujitcn/kratos-admin-system/api/system/base_redact_policy";
import { defBaseRedactRuleService } from "@liujitcn/kratos-admin-system/api/system/base_redact_rule";
import type { BaseApi } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api";
import type { BaseRedactPolicy, BaseRedactPolicyForm, PageBaseRedactPolicyRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_policy";
import { BaseRedactOutputPolicyMode } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_policy";
import { BaseRedactDirection } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_common";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

defineOptions({ name: "BaseRedactPolicy", inheritAttrs: false });
const { BUTTONS } = useAuthButtons();
const table = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const apiOptions = ref<ProFormOption[]>([]);
const fieldOptions = ref<ProFormOption[]>([]);
const ruleOptions = ref<ProFormOption[]>([]);
const dialog = reactive({ visible: false, titleKey: "common.action.create_resource" });
const statusOptions = computed<ProFormOption[]>(() => [{ label: t("common.status.enabled"), value: Status.STATUS_ENABLE }, { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }]);
const directionOptions = computed<ProFormOption[]>(() => [{ label: t("system.base.redact_policy.direction.response"), value: BaseRedactDirection.BASE_REDACT_DIRECTION_RESPONSE }]);
const modeOptions = computed<ProFormOption[]>(() => [{ label: t("system.base.redact_policy.mode.rule"), value: BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE }, { label: t("system.base.redact_policy.mode.hide"), value: BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_HIDE }, { label: t("system.base.redact_policy.mode.full"), value: BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_FULL }]);
const form = reactive<BaseRedactPolicyForm>({ id: 0, field_id: 0, rule_id: 0, scene_code: "*", operation: "", direction: BaseRedactDirection.BASE_REDACT_DIRECTION_RESPONSE, mode: BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE, priority: 0, status: Status.STATUS_ENABLE, remark: "" });
const fields = computed<ProFormField[]>(() => [
  { prop: "operation", label: t("system.base.redact_policy.field.operation"), component: "select", props: { filterable: true }, options: apiOptions.value },
  { prop: "direction", label: t("system.base.redact_policy.field.direction"), component: "select", options: directionOptions.value },
  { prop: "field_id", label: t("system.base.redact_policy.field.field"), component: "select", props: { filterable: true }, options: fieldOptions.value },
  { prop: "rule_id", label: t("system.base.redact_policy.field.rule"), component: "select", props: { filterable: true, clearable: true }, options: ruleOptions.value, visible: () => form.mode === BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE },
  { prop: "scene_code", label: t("system.base.redact_policy.field.scene_code"), component: "input", props: { disabled: true } },
  { prop: "mode", label: t("system.base.redact_policy.field.mode"), component: "select", options: modeOptions.value },
  { prop: "priority", label: t("system.base.redact_policy.field.priority"), component: "input-number", props: { min: 0, precision: 0, style: { width: "100%" } } },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value },
  { prop: "remark", label: t("common.field.remark"), component: "textarea" }
]);
const rules = computed(() => ({ operation: [{ required: true, message: t("system.base.redact_policy.validation.operation"), trigger: "change" }], direction: [{ required: true, message: t("system.base.redact_policy.validation.direction"), trigger: "change" }], field_id: [{ required: true, message: t("system.base.redact_policy.validation.field"), trigger: "change" }], scene_code: [{ required: true, message: t("system.base.redact_policy.validation.scene"), trigger: "blur" }] }));
const modeEnums = computed<EnumProps[]>(() => modeOptions.value);
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "operation", label: t("system.base.redact_policy.field.operation"), minWidth: 300, search: { el: "input" } },
  { prop: "direction", label: t("system.base.redact_policy.field.direction"), width: 110, enum: directionOptions.value, isFilterEnum: true, search: { el: "select", enum: directionOptions.value } },
  { prop: "message_ref", label: t("system.base.redact_policy.field.message_ref"), minWidth: 220, search: { el: "input" } },
  { prop: "field_path", label: t("system.base.redact_policy.field.field_path"), minWidth: 130, search: { el: "input" } },
  { prop: "scene_code", label: t("system.base.redact_policy.field.scene_code"), width: 110, search: { el: "input" } },
  { prop: "mode", label: t("system.base.redact_policy.field.mode"), width: 110, enum: modeEnums.value, isFilterEnum: true, search: { el: "select", enum: modeEnums.value } },
  { prop: "rule_code", label: t("system.base.redact_policy.field.rule"), width: 150 },
  { prop: "status", label: t("common.field.status"), width: 100, cellType: "status", statusProps: { activeValue: Status.STATUS_ENABLE, inactiveValue: Status.STATUS_DISABLE, activeText: t("common.status.enabled"), inactiveText: t("common.status.disabled"), disabled: () => !BUTTONS.value["base:redact-policy:status"], beforeChange: scope => changeStatus(scope.row as BaseRedactPolicy) } },
  { prop: "actions", label: t("common.field.operation"), width: 140, fixed: "right", cellType: "actions", actions: [{ label: t("common.action.edit"), type: "primary", link: true, icon: EditPen, hidden: () => !BUTTONS.value["base:redact-policy:update"], onClick: scope => openDialog((scope.row as BaseRedactPolicy).id) }, { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !BUTTONS.value["base:redact-policy:delete"], onClick: scope => deleteItems(scope.row as BaseRedactPolicy) }] }
]);
const headerActions = computed<HeaderActionProps[]>(() => [{ label: t("common.action.create"), type: "success", icon: CirclePlus, hidden: () => !BUTTONS.value["base:redact-policy:create"], onClick: () => openDialog() }, { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: () => !BUTTONS.value["base:redact-policy:delete"], disabled: scope => !scope.selectedList.length, onClick: scope => deleteItems(scope.selectedList as BaseRedactPolicy[]) }]);

/** 请求策略分页列表。 */
async function requestTable(params: PageBaseRedactPolicyRequest) { const data = await defBaseRedactPolicyService.PageBaseRedactPolicy(buildPageRequest(params)); return { data: { list: data.base_redact_policies ?? [], total: data.total } }; }
/** 加载接口和规则选项。 */
async function loadOptions() { const [apisResult, rulesResult] = await Promise.all([defBaseApiService.OptionBaseApi({ include_public: true }), defBaseRedactRuleService.OptionBaseRedactRule({ keyword: "" })]); apiOptions.value = (apisResult.base_apis ?? []).map((item: BaseApi) => ({ label: `${item.service_name} / ${item.operation} (${item.method} ${item.path})`, value: item.operation })); ruleOptions.value = rulesResult.list ?? []; await loadFieldOptions(); }
/** 按当前接口和方向加载可选字段。 */
async function loadFieldOptions() { if (!form.operation) { fieldOptions.value = []; form.field_id = 0; return; } const result = await defBaseRedactFieldService.OptionBaseRedactField({ keyword: "", operation: form.operation, direction: form.direction }); fieldOptions.value = result.list ?? []; if (!fieldOptions.value.some(item => item.value === form.field_id)) form.field_id = 0; }
/** 打开策略编辑弹窗。 */
async function openDialog(id?: number) { resetForm(); await loadOptions(); dialog.titleKey = id ? "common.action.edit_resource" : "common.action.create_resource"; dialog.visible = true; if (id) Object.assign(form, await defBaseRedactPolicyService.GetBaseRedactPolicy({ id })); }
/** 重置策略表单。 */
function resetForm() { dialog.visible = false; dialogRef.value?.resetFields(); Object.assign(form, { id: 0, field_id: 0, rule_id: 0, scene_code: "*", operation: "", direction: BaseRedactDirection.BASE_REDACT_DIRECTION_RESPONSE, mode: BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE, priority: 0, status: Status.STATUS_ENABLE, remark: "" }); }
/** 提交策略表单。 */
function submit() { dialogRef.value?.validate()?.then(async valid => { if (!valid) return; const payload = JSON.parse(JSON.stringify(form)) as BaseRedactPolicyForm; if (payload.mode !== BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE) payload.rule_id = 0; if (payload.id) await defBaseRedactPolicyService.UpdateBaseRedactPolicy({ base_redact_policy: payload }); else await defBaseRedactPolicyService.CreateBaseRedactPolicy({ base_redact_policy: payload }); ElMessage.success(t(payload.id ? "common.message.update_success" : "common.message.create_success", { resource: t("system.base.redact_policy.title") })); resetForm(); table.value?.getTableList(); }); }
/** 切换策略状态。 */
async function changeStatus(row: BaseRedactPolicy) { const next = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE; try { await ElMessageBox.confirm(t("common.dialog.status_change", { action: t(next === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled"), resource: t("system.base.redact_policy.title"), field: t("system.base.redact_policy.field.message_ref"), value: row.message_ref })); await defBaseRedactPolicyService.SetBaseRedactPolicyStatus({ id: row.id, status: next }); table.value?.getTableList(); return true; } catch { return false; } }
/** 删除策略绑定。 */
function deleteItems(selected?: BaseRedactPolicy | BaseRedactPolicy[] | number | string | Array<number | string>) { const items = Array.isArray(selected) ? selected.filter((item): item is BaseRedactPolicy => typeof item === "object") : selected && typeof selected === "object" ? [selected] : []; const ids = items.length ? items.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>); if (!ids.length) { ElMessage.warning(t("common.message.select_delete_item")); return; } ElMessageBox.confirm(t("common.dialog.delete_selected", { resource: t("system.base.redact_policy.title") }), t("common.title.warning"), { type: "warning" }).then(async () => { await defBaseRedactPolicyService.DeleteBaseRedactPolicy({ id: ids.join(",") }); ElMessage.success(t("common.message.delete_success", { resource: t("system.base.redact_policy.title") })); table.value?.getTableList(); }); }

watch([() => form.operation, () => form.direction], () => { void loadFieldOptions(); });
</script>
