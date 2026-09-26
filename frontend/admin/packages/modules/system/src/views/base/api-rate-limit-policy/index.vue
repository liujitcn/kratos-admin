<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="dialogRef"
      :title="t(dialog.titleKey)"
      width="min(1280px, calc(100vw - 32px))"
      label-position="left"
      :model="form"
      :fields="fields"
      :rules="rules"
      @confirm="submit"
      @close="resetForm"
    >
      <template #parameters>
        <div class="parameter-grid">
          <div class="parameter-item">
            <span>{{ t("system.base.rate_limit_rule.field.tokens_per_second") }}</span>
            <el-input-number v-model="form.params.tokens_per_second" :min="0" :max="1000000" :precision="3" :step="1" controls-position="right" />
          </div>
          <div class="parameter-item">
            <span>{{ t("system.base.rate_limit_rule.field.burst") }}</span>
            <el-input-number v-model="form.params.burst" :min="0" :max="1000000" :precision="0" controls-position="right" />
          </div>
        </div>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import type { FormRules } from "element-plus";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseApiService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_api";
import { defBaseApiRateLimitPolicyService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_api_rate_limit_policy";
import { defBaseRateLimitRuleService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_rate_limit_rule";
import type { BaseApi } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api";
import type {
  BaseApiRateLimitPolicy,
  BaseApiRateLimitPolicyForm,
  PageBaseApiRateLimitPolicyRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api_rate_limit_policy";
import { BaseApiRateLimitDimension } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api_rate_limit_policy";
import type { BaseRateLimitRuleOption } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_rate_limit_rule";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

/** 令牌桶参数快照。 */
interface RateLimitParams {
  /** 每秒生成的令牌数。 */
  tokens_per_second: number | undefined;
  /** 令牌桶容量。 */
  burst: number | undefined;
}

/** 接口限流策略表单状态。 */
interface PolicyFormState extends BaseApiRateLimitPolicyForm {
  /** 当前编辑的限流参数。 */
  params: RateLimitParams;
}

defineOptions({ name: "BaseApiRateLimitPolicy", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const table = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const apiCatalog = ref<BaseApi[]>([]);
const ruleCatalog = ref<BaseRateLimitRuleOption[]>([]);
const apiOptions = ref<ProFormOption[]>([]);
const ruleOptions = ref<ProFormOption[]>([]);
const dialog = reactive({ visible: false, titleKey: "common.action.create_resource" });
const form = reactive<PolicyFormState>(defaultForm());
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
  {
    prop: "operations",
    label: t("system.base.api_rate_limit_policy.field.api"),
    component: "transfer",
    options: apiOptions.value,
    props: {
      class: "policy-api-transfer",
      style: {
        width: "100%"
      },
      filterable: true,
      titles: [t("system.base.menu.value.available_api"), t("system.base.menu.value.selected_api")]
    },
    colSpan: 24
  },
  { prop: "dimension", label: t("system.base.api_rate_limit_policy.field.dimension"), component: "select", options: dimensionOptions.value },
  { prop: "rule_id", label: t("system.base.api_rate_limit_policy.field.rule"), component: "select", options: ruleOptions.value, props: { filterable: true, onChange: handleRuleChange } },
  { prop: "parameters", label: t("system.base.rate_limit_rule.field.parameters"), component: "slot", slotName: "parameters", colSpan: 24 },
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
  {
    prop: "apis",
    label: t("system.base.api_rate_limit_policy.field.api"),
    minWidth: 360,
    render: scope => (scope.row as BaseApiRateLimitPolicy).apis.map(api => `${api.service_desc || api.service_name} · ${api.method} ${api.path} · ${api.operation}`).join("\n")
  },
  { prop: "dimension", label: t("system.base.api_rate_limit_policy.field.dimension"), width: 150, render: scope => dimensionLabel((scope.row as BaseApiRateLimitPolicy).dimension) },
  { prop: "rule_name", label: t("system.base.api_rate_limit_policy.field.rule"), minWidth: 160 },
  { prop: "rule_params", label: t("system.base.rate_limit_rule.field.parameters"), minWidth: 230, render: scope => paramsSummary((scope.row as BaseApiRateLimitPolicy).rule_params) },
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
      if (data) Object.assign(form, data, { params: parseParams(data.rule_params) });
      dialog.titleKey = id !== undefined ? "common.action.edit_resource" : "common.action.create_resource";
    }
  });
}

/** 加载可配置接口和限流规则。 */
async function loadOptions() {
  const [apis, rules] = await Promise.all([
    defBaseApiService.OptionBaseApi({ include_public: true }),
    defBaseRateLimitRuleService.OptionBaseRateLimitRule({ keyword: "" })
  ]);
  apiCatalog.value = apis.base_apis ?? [];
  apiOptions.value = apiCatalog.value.map(api => ({
    label: `${api.service_desc || api.service_name}/${api.desc || `${api.method} ${api.path}`}`,
    value: api.operation
  }));
  ruleCatalog.value = rules.list ?? [];
  ruleOptions.value = ruleCatalog.value.map(rule => ({ label: `${rule.label} (${rule.code})`, value: rule.id, disabled: rule.disabled }));
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
  form.params = selected ? parseParams(selected.default_params) : { tokens_per_second: undefined, burst: undefined };
}

/** 校验并保存接口策略参数快照。 */
async function submit() {
  if (
    typeof form.params.tokens_per_second !== "number" ||
    form.params.tokens_per_second < 0.001 ||
    typeof form.params.burst !== "number" ||
    form.params.burst < 1
  ) {
    ElMessage.warning(t("system.base.rate_limit_rule.validation.parameters"));
    return;
  }
  form.rule_params = JSON.stringify(form.params);
  const valid = await dialogRef.value?.validate();
  if (!valid) return;
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

/** 解析令牌桶参数 JSON。 */
function parseParams(raw: string): RateLimitParams {
  try {
    const value = JSON.parse(raw) as Partial<RateLimitParams>;
    const tokensPerSecond = Number(value.tokens_per_second);
    const burst = Number(value.burst);
    return {
      tokens_per_second: Number.isFinite(tokensPerSecond) ? tokensPerSecond : 0,
      burst: Number.isFinite(burst) ? burst : 0
    };
  } catch {
    return { tokens_per_second: 0, burst: 0 };
  }
}

/** 格式化策略参数摘要。 */
function paramsSummary(raw: string) {
  const params = parseParams(raw);
  return t("system.base.rate_limit_rule.message.parameters_summary", { rate: params.tokens_per_second ?? 0, burst: params.burst ?? 0 });
}

/** 返回限流维度的显示名称。 */
function dimensionLabel(dimension: BaseApiRateLimitDimension) {
  const option = dimensionOptions.value.find(item => item.value === dimension);
  return option?.label ?? String(dimension);
}

/** 返回接口策略表单初始值。 */
function defaultForm(): PolicyFormState {
  return {
    id: 0,
    operations: [],
    dimension: BaseApiRateLimitDimension.BASE_API_RATE_LIMIT_DIMENSION_GLOBAL,
    rule_id: 0,
    rule_params: "{}",
    status: Status.STATUS_ENABLE,
    remark: "",
    params: { tokens_per_second: undefined, burst: undefined }
  };
}
</script>

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

.parameter-item > span {
  flex: none;
  color: var(--el-text-color-regular);
  font-size: 12px;
}

.parameter-item :deep(.el-input-number) {
  flex: 1;
  width: 0;
  min-width: 0;
}

.policy-api-transfer {
  width: 100%;
}

.policy-api-transfer :deep(.el-transfer-panel) {
  width: min(520px, calc((100% - 112px) / 2));
  min-width: 0;
}

.policy-api-transfer :deep(.el-transfer__buttons) {
  padding: 0 16px;
}

.policy-api-transfer :deep(.el-transfer-panel__body),
.policy-api-transfer :deep(.el-transfer-panel__list) {
  height: min(420px, 50vh);
}

.policy-api-transfer :deep(.el-transfer-panel__item) {
  height: auto;
  min-height: 30px;
  line-height: 1.4;
  padding-top: 6px;
  padding-bottom: 6px;
}

.policy-api-transfer :deep(.el-transfer-panel__item .el-checkbox__label) {
  height: auto;
  line-height: 1.4;
  white-space: normal;
  overflow: visible;
  text-overflow: clip;
  overflow-wrap: anywhere;
}

@media (max-width: 760px) {
  .parameter-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .policy-api-transfer {
    width: 100%;
  }

  .policy-api-transfer :deep(.el-transfer-panel) {
    width: calc((100% - 80px) / 2);
  }

  .policy-api-transfer :deep(.el-transfer__buttons) {
    padding: 0 6px;
  }
}
</style>
