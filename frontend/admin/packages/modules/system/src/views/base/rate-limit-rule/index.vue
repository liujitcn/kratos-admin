<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="dialogRef"
      :title="t(dialog.titleKey)"
      width="720px"
      :model="form"
      :fields="fields"
      :rules="rules"
      @confirm="submit"
      @close="resetForm"
    >
      <template #parameters>
        <div v-if="form.rule_type === BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET" class="parameter-grid">
          <div class="parameter-item">
            <span class="parameter-label">
              <span>{{ t("system.base.rate_limit_rule.field.tokens_per_second") }}</span>
              <el-tooltip :content="t('system.base.rate_limit_rule.tooltip.tokens_per_second')" placement="top" effect="light">
                <el-icon class="parameter-help" tabindex="0"><QuestionFilled /></el-icon>
              </el-tooltip>
            </span>
            <el-input-number v-model="form.params.tokens_per_second" :min="0.001" :max="1000000" :precision="3" :step="1" controls-position="right" />
          </div>
          <div class="parameter-item">
            <span class="parameter-label">
              <span>{{ t("system.base.rate_limit_rule.field.burst") }}</span>
              <el-tooltip :content="t('system.base.rate_limit_rule.tooltip.burst')" placement="top" effect="light">
                <el-icon class="parameter-help" tabindex="0"><QuestionFilled /></el-icon>
              </el-tooltip>
            </span>
            <el-input-number v-model="form.params.burst" :min="1" :max="1000000" :precision="0" controls-position="right" />
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
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import type { FormRules } from "element-plus";
import { CirclePlus, Delete, EditPen, QuestionFilled } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseRateLimitRuleService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_rate_limit_rule";
import { BaseRateLimitAlgorithm } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_rate_limit_rule";
import type { BaseRateLimitRule, BaseRateLimitRuleForm, PageBaseRateLimitRuleRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_rate_limit_rule";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { defaultRateLimitParams, parseRateLimitParams, rateLimitAlgorithmLabelKey, rateLimitAlgorithmTooltipKey, rateLimitSummaryKey, rateLimitSummaryParams, serializeRateLimitParams, type RateLimitParams } from "../../../utils/rateLimit";

/** 限流规则表单状态。 */
interface RuleFormState extends BaseRateLimitRuleForm {
  /** 当前算法对应的可编辑参数。 */
  params: RateLimitParams;
}

defineOptions({ name: "BaseRateLimitRule", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const table = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ visible: false, titleKey: "common.action.create_resource" });
const form = reactive<RuleFormState>(defaultForm());
const windowedAlgorithms = [
  BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_FIXED_WINDOW,
  BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_COUNTER,
  BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG
];
const algorithmOptions = computed<ProFormOption[]>(() => [
  BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET,
  BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_FIXED_WINDOW,
  BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_COUNTER,
  BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_SLIDING_WINDOW_LOG,
  BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_LEAKY_BUCKET
].map(value => ({ label: t(rateLimitAlgorithmLabelKey(value)), value })));
const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);
/** 当前算法说明提示，未选择算法时为空避免渲染空 tooltip。 */
const ruleTypeTooltip = computed(() => {
  const key = rateLimitAlgorithmTooltipKey(form.rule_type);
  return key ? t(key) : undefined;
});
const fields = computed<ProFormField[]>(() => [
  { prop: "code", label: t("system.base.rate_limit_rule.field.code"), component: "input", props: { disabled: Boolean(form.id) } },
  { prop: "name", label: t("system.base.rate_limit_rule.field.name"), component: "input" },
  { prop: "rule_type", label: t("system.base.rate_limit_rule.field.rule_type"), labelTooltip: ruleTypeTooltip.value, component: "select", options: algorithmOptions.value, props: { disabled: Boolean(form.id), onChange: handleAlgorithmChange } },
  { prop: "parameters", label: t("system.base.rate_limit_rule.field.parameters"), labelTooltip: t("system.base.rate_limit_rule.tooltip.parameters"), component: "slot", slotName: "parameters", colSpan: 24 },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value },
  { prop: "remark", label: t("common.field.remark"), component: "textarea" }
]);
const rules = computed<FormRules>(() => ({
  code: [{ required: true, message: t("system.base.rate_limit_rule.validation.code"), trigger: "blur" }],
  name: [{ required: true, message: t("system.base.rate_limit_rule.validation.name"), trigger: "blur" }]
}));
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  { prop: "code", label: t("system.base.rate_limit_rule.field.code"), minWidth: 190, search: { el: "input" } },
  { prop: "name", label: t("system.base.rate_limit_rule.field.name"), minWidth: 190, search: { el: "input" } },
  { prop: "rule_type", label: t("system.base.rate_limit_rule.field.rule_type"), width: 190, render: scope => algorithmLabel((scope.row as BaseRateLimitRule).rule_type) },
  { prop: "default_params", label: t("system.base.rate_limit_rule.field.parameters"), minWidth: 260, render: scope => paramsSummary((scope.row as BaseRateLimitRule).rule_type, (scope.row as BaseRateLimitRule).default_params) },
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
      disabled: () => !BUTTONS.value["base:rate-limit-rule:status"],
      beforeChange: scope => changeStatus(scope.row as BaseRateLimitRule)
    }
  },
  {
    prop: "actions",
    label: t("common.field.operation"),
    cellType: "actions",
    actions: [
      { label: t("common.action.edit"), type: "primary", link: true, icon: EditPen, hidden: () => !BUTTONS.value["base:rate-limit-rule:update"], onClick: scope => openDialog((scope.row as BaseRateLimitRule).id) },
      { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !BUTTONS.value["base:rate-limit-rule:delete"], onClick: scope => deleteItems(scope.row as BaseRateLimitRule) }
    ]
  }
]);
const headerActions = computed<HeaderActionProps[]>(() => [
  { label: t("common.action.create"), type: "success", icon: CirclePlus, hidden: () => !BUTTONS.value["base:rate-limit-rule:create"], onClick: () => openDialog() },
  { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: () => !BUTTONS.value["base:rate-limit-rule:delete"], disabled: scope => !scope.selectedList.length, onClick: scope => deleteItems(scope.selectedList as BaseRateLimitRule[]) }
]);

/** 请求限流规则列表。 */
async function requestTable(params: PageBaseRateLimitRuleRequest) {
  const data = await defBaseRateLimitRuleService.PageBaseRateLimitRule(buildPageRequest(params));
  return { data: { list: data.base_rate_limit_rules ?? [], total: data.total } };
}

/** 打开新增或编辑限流规则弹窗。 */
async function openDialog(id?: number) {
  await dialogRef.value?.open({
    load: () => (id !== undefined ? defBaseRateLimitRuleService.GetBaseRateLimitRule({ id }) : undefined),
    commit: data => {
      resetForm();
      if (data) Object.assign(form, data, { params: parseRateLimitParams(data.rule_type, data.default_params) });
      dialog.titleKey = id !== undefined ? "common.action.edit_resource" : "common.action.create_resource";
    }
  });
}

/** 重置限流规则表单。 */
function resetForm() {
  dialog.visible = false;
  dialogRef.value?.resetFields();
  Object.assign(form, defaultForm());
}

/** 校验并保存限流规则模板。 */
async function submit() {
  const params = serializeRateLimitParams(form.rule_type, form.params);
  if (!params) {
    ElMessage.warning(t("system.base.rate_limit_rule.validation.parameters"));
    return;
  }
  form.default_params = JSON.stringify(params);
  const valid = await dialogRef.value?.validate();
  if (!valid) return;
  const payload: BaseRateLimitRuleForm = {
    id: form.id,
    code: form.code,
    name: form.name,
    rule_type: form.rule_type,
    default_params: form.default_params,
    status: form.status,
    remark: form.remark
  };
  if (form.id) await defBaseRateLimitRuleService.UpdateBaseRateLimitRule({ base_rate_limit_rule: payload });
  else await defBaseRateLimitRuleService.CreateBaseRateLimitRule({ base_rate_limit_rule: payload });
  ElMessage.success(t(form.id ? "common.message.update_success" : "common.message.create_success", { resource: t("system.base.rate_limit_rule.title") }));
  resetForm();
  table.value?.getTableList();
}

/** 确认并切换限流规则状态。 */
async function changeStatus(row: BaseRateLimitRule) {
  const next = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action: t(next === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled"),
        resource: t("system.base.rate_limit_rule.title"),
        field: t("system.base.rate_limit_rule.field.code"),
        value: row.code
      }),
      t("common.title.warning"),
      { type: "warning" }
    );
    await defBaseRateLimitRuleService.SetBaseRateLimitRuleStatus({ id: row.id, status: next });
    table.value?.getTableList();
    return true;
  } catch {
    return false;
  }
}

/** 删除未被接口策略引用的限流规则。 */
async function deleteItems(selected?: BaseRateLimitRule | BaseRateLimitRule[] | number | string | Array<number | string>) {
  const items = Array.isArray(selected) ? selected.filter((item): item is BaseRateLimitRule => typeof item === "object") : selected && typeof selected === "object" ? [selected] : [];
  const ids = items.length ? items.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>);
  if (!ids.length) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }
  try {
    await ElMessageBox.confirm(t("common.dialog.delete_selected", { resource: t("system.base.rate_limit_rule.title") }), t("common.title.warning"), { type: "warning" });
    await defBaseRateLimitRuleService.DeleteBaseRateLimitRule({ id: ids.join(",") });
    ElMessage.success(t("common.message.delete_success", { resource: t("system.base.rate_limit_rule.title") }));
    table.value?.getTableList();
  } catch {
    return;
  }
}

/** 格式化规则参数摘要。 */
function paramsSummary(algorithm: BaseRateLimitAlgorithm, raw: string) {
  return t(rateLimitSummaryKey(algorithm), rateLimitSummaryParams(algorithm, raw));
}

/** 返回限流算法显示名称。 */
function algorithmLabel(algorithm: BaseRateLimitAlgorithm) { return t(rateLimitAlgorithmLabelKey(algorithm)); }

/** 切换算法时清空其他算法的参数。 */
function handleAlgorithmChange(algorithm: BaseRateLimitAlgorithm) { form.params = defaultRateLimitParams(algorithm); }

/** 返回限流规则表单初始值。 */
function defaultForm(): RuleFormState {
  const rule_type = BaseRateLimitAlgorithm.BASE_RATE_LIMIT_ALGORITHM_TOKEN_BUCKET;
  return { id: 0, code: "", name: "", rule_type, default_params: "", status: Status.STATUS_ENABLE, remark: "", params: defaultRateLimitParams(rule_type) };
}
</script>

<style scoped>
.parameter-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
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
  width: 100%;
  min-width: 0;
}

@media (max-width: 760px) {
  .parameter-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
