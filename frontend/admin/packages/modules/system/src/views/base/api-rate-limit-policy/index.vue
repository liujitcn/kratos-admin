<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="dialogRef"
      :title="t(dialog.titleKey)"
      width="min(1600px, calc(100vw - 32px))"
      label-position="left"
      :model="form"
      :fields="fields"
      :rules="rules"
      @confirm="submit"
      @close="resetForm"
    >
      <template #apiTransfer>
        <ApiTransfer
          v-model="form.operations"
          :options="apiOptions"
          :titles="[t('system.base.menu.value.available_api'), t('system.base.menu.value.selected_api')]"
          :panel-height="420"
        />
      </template>
      <template #parameters>
        <div v-if="form.rule_type === BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET" class="parameter-grid">
          <div class="parameter-item">
            <span class="parameter-label">
              <span>{{ t("system.base.rate_limit_rule.field.tokens_per_second") }}</span>
              <el-tooltip :content="t('system.base.rate_limit_rule.tooltip.tokens_per_second')" placement="top" effect="light">
                <el-icon class="parameter-help" tabindex="0"><QuestionFilled /></el-icon>
              </el-tooltip>
            </span>
            <el-input-number v-model="form.params.tokens_per_second" :min="0" :max="1000000" :precision="3" :step="1" controls-position="right" />
          </div>
          <div class="parameter-item">
            <span class="parameter-label">
              <span>{{ t("system.base.rate_limit_rule.field.burst") }}</span>
              <el-tooltip :content="t('system.base.rate_limit_rule.tooltip.burst')" placement="top" effect="light">
                <el-icon class="parameter-help" tabindex="0"><QuestionFilled /></el-icon>
              </el-tooltip>
            </span>
            <el-input-number v-model="form.params.burst" :min="0" :max="1000000" :precision="0" controls-position="right" />
          </div>
        </div>
        <div v-else-if="windowedAlgorithms.includes(form.rule_type)" class="parameter-grid">
          <div class="parameter-item">
            <span class="parameter-label">
              <span>{{ t("system.base.rate_limit_rule.field.limit") }}</span>
              <el-tooltip :content="t('system.base.rate_limit_rule.tooltip.limit')" placement="top" effect="light">
                <el-icon class="parameter-help" tabindex="0"><QuestionFilled /></el-icon>
              </el-tooltip>
            </span>
            <el-input-number v-model="form.params.limit" :min="1" :max="form.rule_type === BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG ? 10000 : 1000000" :precision="0" controls-position="right" />
          </div>
          <div class="parameter-item">
            <span class="parameter-label">
              <span>{{ t("system.base.rate_limit_rule.field.window_seconds") }}</span>
              <el-tooltip :content="t('system.base.rate_limit_rule.tooltip.window_seconds')" placement="top" effect="light">
                <el-icon class="parameter-help" tabindex="0"><QuestionFilled /></el-icon>
              </el-tooltip>
            </span>
            <el-input-number v-model="form.params.window_seconds" :min="1" :max="86400" :precision="0" controls-position="right" />
          </div>
        </div>
        <div v-else-if="form.rule_type === BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_LEAKY_BUCKET" class="parameter-grid">
          <div class="parameter-item">
            <span class="parameter-label">
              <span>{{ t("system.base.rate_limit_rule.field.leak_rate_per_second") }}</span>
              <el-tooltip :content="t('system.base.rate_limit_rule.tooltip.leak_rate_per_second')" placement="top" effect="light">
                <el-icon class="parameter-help" tabindex="0"><QuestionFilled /></el-icon>
              </el-tooltip>
            </span>
            <el-input-number v-model="form.params.leak_rate_per_second" :min="0.001" :max="1000000" :precision="3" :step="1" controls-position="right" />
          </div>
          <div class="parameter-item">
            <span class="parameter-label">
              <span>{{ t("system.base.rate_limit_rule.field.capacity") }}</span>
              <el-tooltip :content="t('system.base.rate_limit_rule.tooltip.capacity')" placement="top" effect="light">
                <el-icon class="parameter-help" tabindex="0"><QuestionFilled /></el-icon>
              </el-tooltip>
            </span>
            <el-input-number v-model="form.params.capacity" :min="1" :max="1000000" :precision="0" controls-position="right" />
          </div>
        </div>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import type { FormRules } from "element-plus";
import { CirclePlus, Delete, EditPen, QuestionFilled } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import ApiTransfer from "@liujitcn/kratos-admin-system/components/api/ApiTransfer.vue";
import { loadApiTransferCatalog, type ApiTransferOption } from "@liujitcn/kratos-admin-system/components/api/apiTransfer";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseApiRateLimitPolicyService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_api_rate_limit_policy";
import { defBaseRateLimitRuleService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_rate_limit_rule";
import type { BaseApi } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api";
import { BaseRateLimitAlgorithm } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_rate_limit_rule";
import type {
  BaseApiRateLimitPolicy,
  BaseApiRateLimitPolicyForm,
  PageBaseApiRateLimitPolicyRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api_rate_limit_policy";
import { BaseApiRateLimitDimension } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api_rate_limit_policy";
import type { BaseRateLimitRuleOption } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_rate_limit_rule";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { defaultRateLimitParams, parseRateLimitParams, rateLimitAlgorithmLabelKey, rateLimitSummaryKey, rateLimitSummaryParams, serializeRateLimitParams, type RateLimitParams } from "../../../utils/rateLimit";

/** 接口限流策略表单状态。 */
interface PolicyFormState extends Omit<BaseApiRateLimitPolicyForm, "rule_id"> {
  /** 当前选择的限流规则 ID。 */
  rule_id: number | undefined;
  /** 当前选择算法及其参数快照。 */
  rule_type: BaseRateLimitAlgorithm;
  params: RateLimitParams;
}

defineOptions({ name: "BaseApiRateLimitPolicy", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const table = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const ruleCatalog = ref<BaseRateLimitRuleOption[]>([]);
const apiOptions = ref<ApiTransferOption[]>([]);
const ruleOptions = ref<ProFormOption[]>([]);
const dialog = reactive({ visible: false, titleKey: "common.action.create_resource" });
const form = reactive<PolicyFormState>(defaultForm());
const windowedAlgorithms = [
  BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_FIXED_WINDOW,
  BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_COUNTER,
  BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG
];
const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);
const dimensionOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.api_rate_limit_policy.dimension.global"), value: BaseApiRateLimitDimension.BASE_API_RATE_LIMIT_DIMENSION_GLOBAL },
  { label: t("system.base.api_rate_limit_policy.dimension.ip"), value: BaseApiRateLimitDimension.BASE_API_RATE_LIMIT_DIMENSION_IP },
  { label: t("system.base.api_rate_limit_policy.dimension.tenant"), value: BaseApiRateLimitDimension.BASE_API_RATE_LIMIT_DIMENSION_TENANT },
  { label: t("system.base.api_rate_limit_policy.dimension.user"), value: BaseApiRateLimitDimension.BASE_API_RATE_LIMIT_DIMENSION_USER },
  { label: t("system.base.api_rate_limit_policy.dimension.oauth_client"), value: BaseApiRateLimitDimension.BASE_API_RATE_LIMIT_DIMENSION_OAUTH_CLIENT }
]);
const fields = computed<ProFormField[]>(() => [
  { prop: "dimension", label: t("system.base.api_rate_limit_policy.field.dimension"), labelTooltip: t("system.base.api_rate_limit_policy.tooltip.dimension"), component: "select", options: dimensionOptions.value },
  { prop: "rule_id", label: t("system.base.api_rate_limit_policy.field.rule"), labelTooltip: t("system.base.api_rate_limit_policy.tooltip.rule"), component: "select", options: ruleOptions.value, props: { filterable: true, placeholder: t("common.placeholder.select"), onChange: handleRuleChange } },
  { prop: "parameters", label: t("system.base.rate_limit_rule.field.parameters"), labelTooltip: t("system.base.api_rate_limit_policy.tooltip.parameters"), component: "slot", slotName: "parameters", colSpan: 24 },
  {
    prop: "operations",
    label: t("system.base.api_rate_limit_policy.field.api"),
    labelTooltip: t("system.base.api_rate_limit_policy.tooltip.apis"),
    component: "slot",
    slotName: "apiTransfer",
    colSpan: 24
  },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value },
  { prop: "remark", label: t("common.field.remark"), component: "textarea" }
]);
const rules = computed<FormRules>(() => ({
  operations: [{ required: true, type: "array", min: 1, message: t("system.base.api_rate_limit_policy.validation.api"), trigger: "change" }],
  dimension: [{ required: true, message: t("system.base.api_rate_limit_policy.validation.dimension"), trigger: "change" }],
  rule_id: [{ required: true, message: t("system.base.api_rate_limit_policy.validation.rule"), trigger: "change" }]
}));
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "dimension", label: t("system.base.api_rate_limit_policy.field.dimension"), width: 150, render: scope => dimensionLabel((scope.row as BaseApiRateLimitPolicy).dimension) },
  { prop: "rule_name", label: t("system.base.api_rate_limit_policy.field.rule"), minWidth: 160 },
  { prop: "rule_type", label: t("system.base.rate_limit_rule.field.rule_type"), width: 190, render: scope => algorithmLabel((scope.row as BaseApiRateLimitPolicy).rule_type) },
  { prop: "rule_params", label: t("system.base.rate_limit_rule.field.parameters"), minWidth: 230, render: scope => paramsSummary((scope.row as BaseApiRateLimitPolicy).rule_type, (scope.row as BaseApiRateLimitPolicy).rule_params) },
  {
    prop: "apis",
    label: t("system.base.api_rate_limit_policy.field.api"),
    minWidth: 360,
    showOverflowTooltip: { popperClass: "api-rate-limit-api-tooltip" },
    tooltipFormatter: ({ row }) =>
      h(
        "div",
        { class: "api-rate-limit-api-tooltip__content" },
        (row as BaseApiRateLimitPolicy).apis.map(api => h("div", { class: "api-rate-limit-api-tooltip__item" }, formatApiLabel(api)))
      ),
    render: scope => (scope.row as BaseApiRateLimitPolicy).apis.map(formatApiLabel).join(" · ")
  },
  {
    prop: "status",
    label: t("common.field.status"),
    width: 100,
    cellType: "status",
    statusProps: {
      activeValue: Status.STATUS_ENABLE,
      inactiveValue: Status.STATUS_DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:api-rate-limit-policy:status"],
      beforeChange: scope => changeStatus(scope.row as BaseApiRateLimitPolicy)
    }
  },
  {
    prop: "actions",
    label: t("common.field.operation"),
    cellType: "actions",
    actions: [
      { label: t("common.action.edit"), type: "primary", link: true, icon: EditPen, hidden: () => !BUTTONS.value["base:api-rate-limit-policy:update"], onClick: scope => openDialog((scope.row as BaseApiRateLimitPolicy).id) },
      { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !BUTTONS.value["base:api-rate-limit-policy:delete"], onClick: scope => deleteItems(scope.row as BaseApiRateLimitPolicy) }
    ]
  }
]);
const headerActions = computed<HeaderActionProps[]>(() => [
  { label: t("common.action.create"), type: "success", icon: CirclePlus, hidden: () => !BUTTONS.value["base:api-rate-limit-policy:create"], onClick: () => openDialog() },
  { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: () => !BUTTONS.value["base:api-rate-limit-policy:delete"], disabled: scope => !scope.selectedList.length, onClick: scope => deleteItems(scope.selectedList as BaseApiRateLimitPolicy[]) }
]);

/** 请求接口限流策略列表。 */
async function requestTable(params: PageBaseApiRateLimitPolicyRequest) {
  const data = await defBaseApiRateLimitPolicyService.PageBaseApiRateLimitPolicy(buildPageRequest(params));
  return { data: { list: data.base_api_rate_limit_policies ?? [], total: data.total } };
}

/** 打开新增或编辑接口限流策略弹窗。 */
async function openDialog(id?: number) {
  await loadOptions();
  await dialogRef.value?.open({
    load: () => (id !== undefined ? defBaseApiRateLimitPolicyService.GetBaseApiRateLimitPolicy({ id }) : undefined),
    commit: data => {
      resetForm();
      if (data) {
        Object.assign(form, data);
        handleRuleChange(data.rule_id);
        form.params = parseRateLimitParams(form.rule_type, data.rule_params);
      }
      dialog.titleKey = id !== undefined ? "common.action.edit_resource" : "common.action.create_resource";
    }
  });
}

/** 加载可配置接口和限流规则。 */
async function loadOptions() {
  const [apiCatalog, rules] = await Promise.all([
    loadApiTransferCatalog({ include_public: true }),
    defBaseRateLimitRuleService.OptionBaseRateLimitRule({ keyword: "" })
  ]);
  apiOptions.value = apiCatalog.options;
  ruleCatalog.value = rules.list ?? [];
  ruleOptions.value = ruleCatalog.value.map(rule => ({ label: `${rule.label} (${rule.code}) · ${algorithmLabel(rule.rule_type)}`, value: rule.id, disabled: rule.disabled }));
}

/** 重置接口限流策略表单。 */
function resetForm() {
  dialog.visible = false;
  dialogRef.value?.resetFields();
  Object.assign(form, defaultForm());
}

/** 根据所选模板加载默认参数。 */
function handleRuleChange(ruleId?: number) {
  const selected = ruleCatalog.value.find(rule => rule.id === Number(ruleId));
  form.rule_type = selected?.rule_type ?? BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_UNSPECIFIED;
  form.params = selected ? parseRateLimitParams(form.rule_type, selected.default_params) : defaultRateLimitParams(form.rule_type);
}

/** 校验并保存接口策略参数快照。 */
async function submit() {
  const valid = await dialogRef.value?.validate();
  if (!valid || form.rule_id === undefined) return;

  const params = serializeRateLimitParams(form.rule_type, form.params);
  if (!params) {
    ElMessage.warning(t("system.base.rate_limit_rule.validation.parameters"));
    return;
  }
  form.rule_params = JSON.stringify(params);
  const payload: BaseApiRateLimitPolicyForm = {
    id: form.id,
    operations: form.operations,
    dimension: form.dimension,
    rule_id: form.rule_id,
    rule_params: form.rule_params,
    status: form.status,
    remark: form.remark
  };
  if (form.id) await defBaseApiRateLimitPolicyService.UpdateBaseApiRateLimitPolicy({ base_api_rate_limit_policy: payload });
  else await defBaseApiRateLimitPolicyService.CreateBaseApiRateLimitPolicy({ base_api_rate_limit_policy: payload });
  ElMessage.success(t(form.id ? "common.message.update_success" : "common.message.create_success", { resource: t("system.base.api_rate_limit_policy.title") }));
  resetForm();
  table.value?.getTableList();
}

/** 确认并切换接口策略状态。 */
async function changeStatus(row: BaseApiRateLimitPolicy) {
  const next = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action: t(next === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled"),
        resource: t("system.base.api_rate_limit_policy.title"),
        field: t("system.base.api_rate_limit_policy.field.api"),
        value: row.apis.map(api => api.operation).join(", ")
      }),
      t("common.title.warning"),
      { type: "warning" }
    );
    await defBaseApiRateLimitPolicyService.SetBaseApiRateLimitPolicyStatus({ id: row.id, status: next });
    table.value?.getTableList();
    return true;
  } catch {
    return false;
  }
}

/** 删除选中的接口限流策略。 */
async function deleteItems(selected?: BaseApiRateLimitPolicy | BaseApiRateLimitPolicy[] | number | string | Array<number | string>) {
  const items = Array.isArray(selected) ? selected.filter((item): item is BaseApiRateLimitPolicy => typeof item === "object") : selected && typeof selected === "object" ? [selected] : [];
  const ids = items.length ? items.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>);
  if (!ids.length) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }
  try {
    await ElMessageBox.confirm(t("common.dialog.delete_selected", { resource: t("system.base.api_rate_limit_policy.title") }), t("common.title.warning"), { type: "warning" });
    await defBaseApiRateLimitPolicyService.DeleteBaseApiRateLimitPolicy({ id: ids.join(",") });
    ElMessage.success(t("common.message.delete_success", { resource: t("system.base.api_rate_limit_policy.title") }));
    table.value?.getTableList();
  } catch {
    return;
  }
}

/** 格式化策略参数摘要。 */
function paramsSummary(algorithm: BaseRateLimitAlgorithm, raw: string) {
  return t(rateLimitSummaryKey(algorithm), rateLimitSummaryParams(algorithm, raw));
}

/** 返回限流算法显示名称。 */
function algorithmLabel(algorithm: BaseRateLimitAlgorithm) { return t(rateLimitAlgorithmLabelKey(algorithm)); }

/** 返回限流维度的显示名称。 */
function dimensionLabel(dimension: BaseApiRateLimitDimension) {
  const option = dimensionOptions.value.find(item => item.value === dimension);
  return option?.label ?? String(dimension);
}

/** 格式化限流策略表格中的接口信息。 */
function formatApiLabel(api: Pick<BaseApi, "service_desc" | "service_name" | "method" | "path" | "operation">) {
  return `${api.service_desc || api.service_name} · ${api.method} ${api.path} · ${api.operation}`;
}

/** 返回接口策略表单初始值。 */
function defaultForm(): PolicyFormState {
  const rule_type = BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_UNSPECIFIED;
  return {
    id: 0,
    operations: [],
    dimension: BaseApiRateLimitDimension.BASE_API_RATE_LIMIT_DIMENSION_GLOBAL,
    rule_id: undefined,
    rule_params: "{}",
    rule_type,
    status: Status.STATUS_ENABLE,
    remark: "",
    params: defaultRateLimitParams(rule_type)
  };
}
</script>

<style lang="scss">
.api-rate-limit-api-tooltip {
  max-width: min(960px, calc(100vw - 48px));
}
.api-rate-limit-api-tooltip__content {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: min(420px, calc(100vh - 32px));
  overflow-y: auto;
}
.api-rate-limit-api-tooltip__item {
  overflow-wrap: anywhere;
  white-space: normal;
}
</style>

<style scoped>
.parameter-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  width: 100%;
}

.parameter-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.parameter-label {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 4px;
  color: var(--el-text-color-regular);
  font-size: 12px;
}

.parameter-help {
  color: var(--el-text-color-placeholder);
  cursor: help;
}

.parameter-item :deep(.el-input-number) {
  flex: 1;
  width: 0;
  min-width: 0;
}
@media (width <= 900px) {
  .parameter-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
