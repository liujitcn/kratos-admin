<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="dialogRef"
      :title="t(dialog.titleKey)"
      width="1180px"
      top="3vh"
      label-width="150px"
      destroy-on-close
      :model="form"
      :fields="fields"
      :rules="rules"
      @confirm="submit"
      @close="resetForm"
    >
      <template #field_rows>
        <div class="field-table">
          <div class="field-table__summary">
            {{ t("system.base.redact_output_policy.message.configured_fields", { count: configuredFieldCount }) }}
          </div>
          <el-table :data="form.field_rows" row-key="value" border height="500">
            <el-table-column :label="t('system.base.redact_output_policy.field.response_field')" min-width="280">
              <template #default="{ row }">
                <div class="field-cell">
                  <div class="field-cell__title">{{ row.field_path }}</div>
                  <div class="field-cell__meta">{{ row.message_ref }}<span v-if="row.description"> · {{ row.description }}</span></div>
                </div>
              </template>
            </el-table-column>
            <el-table-column :label="t('system.base.redact_output_policy.field.mode')" width="170">
              <template #default="{ row }">
                <el-select v-model="row.mode" class="table-control" size="small" @change="handleModeChange(row)">
                  <el-option v-for="option in modeOptions" :key="String(option.value)" :label="option.label" :value="option.value" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column :label="t('system.base.redact_output_policy.field.rule')" width="250">
              <template #default="{ row }">
                <el-select
                  v-model="row.rule_id"
                  class="table-control"
                  size="small"
                  clearable
                  :disabled="row.mode !== BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE"
                  @change="handleRuleChange(row)"
                >
                  <el-option v-for="option in ruleOptions" :key="String(option.value)" :label="option.label" :value="option.value" :disabled="option.disabled" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column :label="t('system.base.redact_output_policy.field.parameters')" min-width="460">
              <template #default="{ row }">
                <div v-if="isRuleRow(row)" class="parameter-grid">
                  <template v-if="row.rule_type === 'MASK'">
                    <ParameterNumber v-model="row.params.keep_first" :label="t('system.base.redact_rule.parameter.keep_first')" />
                    <ParameterNumber v-model="row.params.keep_last" :label="t('system.base.redact_rule.parameter.keep_last')" />
                    <ParameterText v-model="row.params.mask_char" :label="t('system.base.redact_rule.parameter.mask_char')" />
                  </template>
                  <template v-else-if="row.rule_type === 'EMAIL'">
                    <ParameterNumber v-model="row.params.keep_local_first" :label="t('system.base.redact_rule.parameter.keep_local_first')" />
                    <div class="parameter-item"><span>{{ t("system.base.redact_rule.parameter.mask_domain") }}</span><el-switch v-model="row.params.mask_domain" /></div>
                    <ParameterText v-model="row.params.mask_char" :label="t('system.base.redact_rule.parameter.mask_char')" />
                  </template>
                  <template v-else-if="row.rule_type === 'REGEX'">
                    <ParameterText v-model="row.params.pattern" :label="t('system.base.redact_rule.parameter.pattern')" />
                    <ParameterText v-model="row.params.replacement" :label="t('system.base.redact_rule.parameter.replacement')" />
                  </template>
                  <template v-else-if="row.rule_type === 'TRUNCATE'">
                    <ParameterNumber v-model="row.params.length" :label="t('system.base.redact_rule.parameter.length')" />
                    <ParameterText v-model="row.params.suffix" :label="t('system.base.redact_rule.parameter.suffix')" />
                  </template>
                  <template v-else-if="row.rule_type === 'HASH'">
                    <div class="parameter-item"><span>{{ t("system.base.redact_rule.parameter.algo") }}</span><el-select v-model="row.params.algo" class="parameter-control" size="small"><el-option label="MD5" value="MD5" /><el-option label="SHA1" value="SHA1" /><el-option label="SHA256" value="SHA256" /></el-select></div>
                  </template>
                  <template v-else-if="row.rule_type === 'IP'">
                    <ParameterNumber v-model="row.params.keep_octets" :label="t('system.base.redact_rule.parameter.keep_octets')" />
                    <ParameterText v-model="row.params.mask_char" :label="t('system.base.redact_rule.parameter.mask_char')" />
                  </template>
                  <template v-else-if="row.rule_type === 'URL'">
                    <div class="parameter-item"><span>{{ t("system.base.redact_rule.parameter.mask_query") }}</span><el-switch v-model="row.params.mask_query" /></div>
                    <ParameterText v-model="row.params.mask_char" :label="t('system.base.redact_rule.parameter.mask_char')" />
                  </template>
                  <template v-else-if="row.rule_type === 'FIXED_LENGTH'">
                    <ParameterText v-model="row.params.char" :label="t('system.base.redact_rule.parameter.char')" />
                  </template>
                </div>
                <span v-else class="empty-value">--</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, reactive, ref } from "vue";
import type { FormRules } from "element-plus";
import type { ColumnProps, EnumProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseApiService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_api";
import { defBaseRedactRuleService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_redact_rule";
import { defBaseRedactOutputPolicyService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_redact_output_policy";
import type { BaseApi, BaseApiDocSchema } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api";
import type { BaseRedactRule } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_rule";
import type { BaseRedactOutputPolicy, BaseRedactOutputPolicyForm, PageBaseRedactOutputPolicyRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_output_policy";
import { BaseRedactOutputPolicyMode } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_output_policy";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

/** 脱敏规则参数。 */
interface RuleParams {
  keep_first?: number;
  keep_last?: number;
  mask_char?: string;
  keep_local_first?: number;
  mask_domain?: boolean;
  pattern?: string;
  replacement?: string;
  length?: number;
  suffix?: string;
  algo?: string;
  keep_octets?: number;
  mask_query?: boolean;
  char?: string;
}

/** 出库字段表格行。 */
interface OutputFieldRow extends ProFormOption {
  /** Proto 消息名称。 */
  message_ref: string;
  /** Proto 字段路径。 */
  field_path: string;
  /** 字段描述。 */
  description?: string;
  /** 已有策略 ID。 */
  id: number;
  /** 处理模式。 */
  mode: BaseRedactOutputPolicyMode;
  /** 脱敏规则 ID。 */
  rule_id?: number;
  /** 脱敏规则类型。 */
  rule_type: string;
  /** 脱敏规则参数对象。 */
  params: RuleParams;
  /** 脱敏规则参数 JSON。 */
  rule_params: string;
}

/** 出库脱敏表单状态。 */
interface OutputFormState extends BaseRedactOutputPolicyForm {
  /** 当前选择的 API ID。 */
  api_id?: number;
  /** API 返回字段表格。 */
  field_rows: OutputFieldRow[];
}

/** 出库表格行事件使用的最小字段视图。 */
interface OutputTableRow {
  /** 处理模式。 */
  mode?: BaseRedactOutputPolicyMode;
  /** 脱敏规则 ID。 */
  rule_id?: number;
  /** 脱敏规则类型。 */
  rule_type?: string;
  /** 脱敏规则参数对象。 */
  params?: RuleParams;
  /** 脱敏规则参数 JSON。 */
  rule_params?: string;
}

const OUTPUT_SYSTEM_FIELD_NAMES = new Set([
  "id", "created_at", "created_by", "updated_at", "updated_by", "deleted_at", "deleted_by",
  "create_time", "creator", "creator_id", "update_time", "updater", "updater_id", "delete_time", "deleter", "deleter_id"
]);

const ParameterText = defineComponent({
  props: { modelValue: { type: [String, Number], default: "" }, label: { type: String, required: true } },
  emits: ["update:modelValue"],
  setup(props, { emit }) {
    return () => h("div", { class: "parameter-item" }, [h("span", props.label), h("input", { class: "el-input__inner", value: props.modelValue, onInput: (event: Event) => emit("update:modelValue", (event.target as HTMLInputElement).value) })]);
  }
});
const ParameterNumber = defineComponent({
  props: { modelValue: { type: Number, default: 0 }, label: { type: String, required: true } },
  emits: ["update:modelValue"],
  setup(props, { emit }) {
    return () => h("div", { class: "parameter-item" }, [h("span", props.label), h("input", { class: "el-input__inner", type: "number", min: 0, value: props.modelValue, onInput: (event: Event) => emit("update:modelValue", Number((event.target as HTMLInputElement).value)) })]);
  }
});

defineOptions({ name: "BaseRedactOutputPolicy", inheritAttrs: false });
const { BUTTONS } = useAuthButtons();
const table = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const apiCatalog = ref<BaseApi[]>([]);
const ruleOptions = ref<ProFormOption[]>([]);
const ruleCatalog = ref<BaseRedactRule[]>([]);
const dialog = reactive({ visible: false, titleKey: "common.action.create_resource" });
const form = reactive<OutputFormState>(defaultForm());
const statusOptions = computed<ProFormOption[]>(() => [{ label: t("common.status.enabled"), value: Status.STATUS_ENABLE }, { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }]);
const serviceOptions = computed<ProFormOption[]>(() => {
  const services = new Map<string, string>();
  for (const item of apiCatalog.value) {
    if (!isGetApi(item)) continue;
    if (item.service_name && !services.has(item.service_name)) services.set(item.service_name, serviceLabel(item.service_name));
  }
  return [...services.entries()].sort(([left], [right]) => left.localeCompare(right)).map(([value, label]) => ({ label, value }));
});
const interfaceOptions = computed(() => apiCatalog.value.filter(item => item.service_name === form.service_name && isGetApi(item)));
const serviceLabels = computed(() => new Map(apiCatalog.value.map(item => [item.service_name, item.service_desc || item.service_name])));
const operationLabels = computed(() => new Map(apiCatalog.value.map(item => [item.operation, item.desc || item.operation])));
const modeOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.redact_output_policy.mode.rule"), value: BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE },
  { label: t("system.base.redact_output_policy.mode.hide"), value: BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_HIDE },
  { label: t("system.base.redact_output_policy.mode.full"), value: BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_FULL }
]);
const configuredFieldCount = computed(() => form.field_rows.filter(isConfiguredRow).length);
const fields = computed<ProFormField[]>(() => [
  { prop: "service_name", label: t("system.base.redact_output_policy.field.service_name"), component: "select", options: serviceOptions.value, props: { filterable: true, clearable: true, placeholder: t("system.base.redact_output_policy.placeholder.service"), onChange: handleServiceChange } },
  { prop: "api_id", label: t("system.base.redact_output_policy.field.api"), component: "select", options: interfaceOptions.value.map(api => ({ label: apiLabel(api), value: api.id })), props: { filterable: true, clearable: true, disabled: !form.service_name, placeholder: t("system.base.redact_output_policy.placeholder.api"), onChange: handleApiChange } },
  { prop: "field_rows", label: t("system.base.redact_output_policy.field.response_field"), component: "slot", slotName: "field_rows", colSpan: 24, visible: () => form.api_id !== undefined },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value },
  { prop: "remark", label: t("common.field.remark"), component: "textarea" }
]);
const rules = computed<FormRules>(() => ({
  service_name: [{ required: true, message: t("system.base.redact_output_policy.validation.service_name"), trigger: "change" }],
  api_id: [{ required: true, message: t("system.base.redact_output_policy.validation.api"), trigger: "change" }],
  field_rows: [{ validator: (_rule, value: OutputFieldRow[], callback) => isConfiguredRowList(value) ? callback() : callback(new Error(t("system.base.redact_output_policy.validation.configured_fields"))), trigger: "change" }]
}));
const modeEnums = computed<EnumProps[]>(() => modeOptions.value);
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "service_name", label: t("system.base.redact_output_policy.field.service_name"), minWidth: 220, search: { el: "input" }, render: scope => serviceLabels.value.get(scope.row.service_name) ?? scope.row.service_name },
  { prop: "operation", label: t("system.base.redact_output_policy.field.api"), minWidth: 260, search: { el: "input" }, render: scope => operationLabels.value.get(scope.row.operation) ?? scope.row.operation },
  { prop: "message_ref", label: t("system.base.redact_output_policy.field.message_ref"), minWidth: 220 },
  { prop: "field_path", label: t("system.base.redact_output_policy.field.field_path"), minWidth: 160 },
  { prop: "mode", label: t("system.base.redact_output_policy.field.mode"), width: 110, enum: modeEnums.value, isFilterEnum: true },
  { prop: "rule_name", label: t("system.base.redact_output_policy.field.rule"), width: 150 },
  { prop: "status", label: t("common.field.status"), width: 100, cellType: "status", statusProps: { activeValue: Status.STATUS_ENABLE, inactiveValue: Status.STATUS_DISABLE, activeText: t("common.status.enabled"), inactiveText: t("common.status.disabled"), disabled: () => !BUTTONS.value["base:redact-output-policy:status"], beforeChange: scope => changeStatus(scope.row as BaseRedactOutputPolicy) } },
  { prop: "actions", label: t("common.field.operation"), width: 150, fixed: "right", cellType: "actions", actions: [{ label: t("common.action.edit"), type: "primary", link: true, icon: EditPen, hidden: () => !BUTTONS.value["base:redact-output-policy:update"], onClick: scope => openDialog((scope.row as BaseRedactOutputPolicy).id) }, { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !BUTTONS.value["base:redact-output-policy:delete"], onClick: scope => deleteItems(scope.row as BaseRedactOutputPolicy) }] }
]);
const headerActions = computed<HeaderActionProps[]>(() => [{ label: t("common.action.create"), type: "success", icon: CirclePlus, hidden: () => !BUTTONS.value["base:redact-output-policy:create"], onClick: () => openDialog() }, { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: () => !BUTTONS.value["base:redact-output-policy:delete"], disabled: scope => !scope.selectedList.length, onClick: scope => deleteItems(scope.selectedList as BaseRedactOutputPolicy[]) }]);

/** 请求出库脱敏策略分页列表。 */
async function requestTable(params: PageBaseRedactOutputPolicyRequest) { const data = await defBaseRedactOutputPolicyService.PageBaseRedactOutputPolicy(buildPageRequest(params)); return { data: { list: data.base_redact_output_policies ?? [], total: data.total } }; }
/** 加载 API 选项。 */
async function loadApis() { const data = await defBaseApiService.OptionBaseApi({ include_public: true }); apiCatalog.value = data.base_apis ?? []; }
/** 加载脱敏规则选项。 */
async function loadRules() { const data = await defBaseRedactRuleService.PageBaseRedactRule({ code: "", name: "", rule_type: "", page_num: 1, page_size: 100 }); ruleCatalog.value = data.base_redact_rules ?? []; ruleOptions.value = ruleCatalog.value.map(item => ({ label: `${item.name} (${item.code})`, value: item.id, disabled: item.status !== Status.STATUS_ENABLE })); }
/** 服务变更后清空接口和返回字段。 */
function handleServiceChange(serviceName?: string) { form.service_name = serviceName ?? ""; form.api_id = undefined; form.operation = ""; form.message_ref = ""; form.field_path = ""; form.field_rows = []; }
/** API 变更后重新加载返回字段。 */
async function handleApiChange(apiId?: number) { const api = interfaceOptions.value.find(item => item.id === apiId); form.service_name = api?.service_name ?? form.service_name; form.operation = api?.operation ?? ""; form.field_rows = []; if (apiId !== undefined) await loadResponseFields(apiId); }
/** 加载 API 返回字段并初始化表格行。 */
async function loadResponseFields(apiId: number) { const api = apiCatalog.value.find(item => item.id === apiId); if (!api) return; const doc = await defBaseApiService.GetBaseApiDoc({ id: apiId }); const rows: OutputFieldRow[] = []; for (const response of doc.responses ?? []) { if (response.body) collectFields(response.body, "", "", rows, true); } form.field_rows = [...new Map(rows.map(item => [String(item.value), item])).values()]; }
/** 递归收集可配置的返回叶子字段。 */
function collectFields(schema: BaseApiDocSchema, parentRef: string, parentPath: string, result: OutputFieldRow[], root: boolean) { const schemaRef = normalizeRef(schema.ref); const messageRef = schemaRef || parentRef; const startsNewMessage = Boolean(schemaRef && parentRef && schemaRef !== parentRef); const rootContainer = root || (!parentRef && (schema.name === "body" || schema.name === "body[]")); const path = rootContainer || startsNewMessage ? "" : schema.name ? (parentPath ? `${parentPath}.${schema.name}` : schema.name) : parentPath; if (!schema.children?.length) { const fieldName = path.split(".").pop()?.toLowerCase() ?? ""; if (path && !OUTPUT_SYSTEM_FIELD_NAMES.has(fieldName)) result.push(createFieldRow({ label: `${messageRef}.${path}${schema.description ? ` (${schema.description})` : ""}`, value: `${messageRef}\u0000${path}`, message_ref: messageRef, field_path: path, description: schema.description })); return; } for (const child of schema.children) collectFields(child, messageRef, path, result, false); }
/** 规范化 OpenAPI 引用名称。 */
function normalizeRef(ref?: string) { const parts = (ref ?? "").split("/"); return parts[parts.length - 1] || ""; }
/** 打开新增或编辑弹窗。 */
async function openDialog(id?: number) { resetForm(); await Promise.all([loadApis(), loadRules()]); if (id !== undefined) { const data = await defBaseRedactOutputPolicyService.GetBaseRedactOutputPolicy({ id }); Object.assign(form, data); const api = apiCatalog.value.find(item => item.operation === data.operation); if (api) { form.service_name = api.service_name; form.api_id = api.id; await loadResponseFields(api.id); const row = form.field_rows.find(item => item.message_ref === data.message_ref && item.field_path === data.field_path); if (row) applyPolicy(row, data); } } dialog.titleKey = id !== undefined ? "common.action.edit_resource" : "common.action.create_resource"; dialog.visible = true; }
/** 重置弹窗表单。 */
function resetForm() { dialog.visible = false; dialogRef.value?.resetFields(); Object.assign(form, defaultForm()); }
/** 保存表格中已配置的全部字段。 */
async function submit() { const valid = await dialogRef.value?.validate(); if (!valid || !isConfiguredRowList(form.field_rows)) return; const rows = form.field_rows.filter(isConfiguredRow); const createPolicies = rows.filter(row => !row.id).map(buildPayload); const updatePolicies = rows.filter(row => Boolean(row.id)).map(buildPayload); const requests: Promise<unknown>[] = []; if (createPolicies.length) requests.push(defBaseRedactOutputPolicyService.CreateBaseRedactOutputPolicy({ base_redact_output_policy: createPolicies })); if (updatePolicies.length) requests.push(defBaseRedactOutputPolicyService.UpdateBaseRedactOutputPolicy({ base_redact_output_policy: updatePolicies })); await Promise.all(requests); ElMessage.success(t("system.base.redact_output_policy.message.batch_save_success", { count: rows.length })); resetForm(); table.value?.getTableList(); }
/** 将表格行转换为出库策略请求。 */
function buildPayload(row: OutputFieldRow): BaseRedactOutputPolicyForm { syncRowParams(row); return { id: row.id, operation: form.operation, service_name: form.service_name, message_ref: row.message_ref, field_path: row.field_path, mode: row.mode, rule_id: row.mode === BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE ? row.rule_id ?? 0 : 0, rule_params: row.mode === BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE ? row.rule_params : "{}", status: form.status, remark: form.remark }; }
/** 将已保存的策略回填到对应字段行。 */
function applyPolicy(row: OutputFieldRow, data: BaseRedactOutputPolicyForm) { row.id = data.id; row.mode = data.mode || BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_FULL; row.rule_id = data.rule_id || undefined; row.rule_params = data.rule_params; if (row.rule_id) { const rule = ruleCatalog.value.find(item => item.id === row.rule_id); row.rule_type = rule?.rule_type ?? ""; row.params = parseParams(row.rule_type, row.rule_params); } }
/** 处理行模式变更。 */
function handleModeChange(row: OutputTableRow) { if (row.mode !== BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE) { row.rule_id = undefined; row.rule_type = ""; row.params = {}; row.rule_params = "{}"; } }
/** 处理行规则变更。 */
function handleRuleChange(row: OutputTableRow) { const rule = ruleCatalog.value.find(item => item.id === row.rule_id); row.rule_type = rule?.rule_type ?? ""; row.params = rule ? parseParams(rule.rule_type, rule.rule) : {}; row.rule_params = rule ? rule.rule : "{}"; }
/** 同步单行规则参数。 */
function syncRowParams(row: OutputFieldRow) { const rule = ruleCatalog.value.find(item => item.id === row.rule_id); if (rule) { row.rule_type = rule.rule_type; row.rule_params = JSON.stringify({ [rule.rule_type.toLowerCase()]: row.params }); } }
/** 判断一行是否已经配置处理方式。 */
function isConfiguredRow(row: OutputFieldRow) { return row.mode === BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_HIDE || row.mode === BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_FULL || (row.mode === BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE && Boolean(row.rule_id)); }
/** 判断字段表格是否至少配置了一行。 */
function isConfiguredRowList(rows: OutputFieldRow[] | undefined) { return Boolean(rows?.some(isConfiguredRow)); }
/** 判断字段行是否使用规则模板。 */
function isRuleRow(row: OutputTableRow) { return row.mode === BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_RULE && Boolean(row.rule_id); }
/** 格式化 API 下拉项。 */
function apiLabel(api: BaseApi) { return api.desc && api.desc !== api.operation ? `${api.desc}（${api.operation}）` : api.operation; }
/** 判断接口是否为 GET 请求。 */
function isGetApi(api: BaseApi) { return api.method.toUpperCase() === "GET"; }
/** 格式化服务下拉项。 */
function serviceLabel(serviceName: string) { const service = apiCatalog.value.find(item => item.service_name === serviceName); return service?.service_desc && service.service_desc !== serviceName ? `${service.service_desc}（${serviceName}）` : serviceName; }
/** 解析规则参数。 */
function parseParams(ruleType: string, raw: string): RuleParams { try { const value = JSON.parse(raw) as Record<string, RuleParams>; return value[ruleType.toLowerCase()] ?? defaultParams(ruleType); } catch { return defaultParams(ruleType); } }
/** 返回规则默认参数。 */
function defaultParams(ruleType: string): RuleParams { switch (ruleType) { case "MASK": return { keep_first: 3, keep_last: 4, mask_char: "*" }; case "EMAIL": return { keep_local_first: 2, mask_domain: false, mask_char: "*" }; case "REGEX": return { pattern: "(?s).+", replacement: "[REDACTED]" }; case "TRUNCATE": return { length: 10, suffix: "..." }; case "HASH": return { algo: "SHA256" }; case "IP": return { keep_octets: 2, mask_char: "x" }; case "URL": return { mask_query: true, mask_char: "*" }; case "FIXED_LENGTH": return { char: "X" }; default: return {}; } }
/** 创建空的出库字段行。 */
function createFieldRow(option: Pick<OutputFieldRow, "label" | "value" | "message_ref" | "field_path"> & { description?: string }): OutputFieldRow { return { ...option, id: 0, mode: BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_FULL, rule_id: undefined, rule_type: "", params: {}, rule_params: "{}" }; }
/** 创建默认出库表单。 */
function defaultForm(): OutputFormState { return { id: 0, api_id: undefined, field_rows: [], service_name: "", operation: "", message_ref: "", field_path: "", mode: BaseRedactOutputPolicyMode.BASE_REDACT_OUTPUT_POLICY_MODE_FULL, rule_id: 0, rule_params: "{}", status: Status.STATUS_ENABLE, remark: "" }; }
/** 修改出库策略状态。 */
async function changeStatus(row: BaseRedactOutputPolicy) { const next = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE; try { await ElMessageBox.confirm(t("common.dialog.status_change", { action: t(next === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled"), resource: t("system.base.redact_output_policy.title"), field: t("system.base.redact_output_policy.field.field_path"), value: row.field_path }), t("common.title.warning"), { type: "warning" }); await defBaseRedactOutputPolicyService.SetBaseRedactOutputPolicyStatus({ id: row.id, status: next }); table.value?.getTableList(); return true; } catch { return false; } }
/** 删除选中的出库策略。 */
function deleteItems(selected?: BaseRedactOutputPolicy | BaseRedactOutputPolicy[] | number | string | Array<number | string>) { const items = Array.isArray(selected) ? selected.filter((item): item is BaseRedactOutputPolicy => typeof item === "object") : selected && typeof selected === "object" ? [selected] : []; const ids = items.length ? items.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>); if (!ids.length) { ElMessage.warning(t("common.message.select_delete_item")); return; } ElMessageBox.confirm(t("common.dialog.delete_selected", { resource: t("system.base.redact_output_policy.title") }), t("common.title.warning"), { type: "warning" }).then(async () => { await defBaseRedactOutputPolicyService.DeleteBaseRedactOutputPolicy({ id: ids.join(",") }); ElMessage.success(t("common.message.delete_success", { resource: t("system.base.redact_output_policy.title") })); table.value?.getTableList(); }); }

void loadApis();
</script>

<style scoped lang="scss">
.field-table { width: 100%; }
.field-table__summary { margin-bottom: 8px; color: var(--el-text-color-secondary); line-height: 1.5; }
.field-cell { min-width: 0; line-height: 1.35; }
.field-cell__title { overflow: hidden; color: var(--el-text-color-regular); text-overflow: ellipsis; white-space: nowrap; }
.field-cell__meta { overflow: hidden; color: var(--el-text-color-secondary); font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.table-control { width: 100%; }
.parameter-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; min-width: 420px; }
.parameter-item { display: flex; flex-direction: column; gap: 4px; min-width: 0; color: var(--el-text-color-regular); font-size: 12px; }
.parameter-control { width: 100%; }
.empty-value { color: var(--el-text-color-secondary); }
@media (max-width: 900px) { .parameter-grid { grid-template-columns: minmax(0, 1fr); min-width: 0; } }
</style>
