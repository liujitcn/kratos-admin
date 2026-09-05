<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :request-api="requestTable" />
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
      <template #rule>
        <div class="rule-editor">
          <div class="rule-description">{{ t(templateDescriptionKey) }}</div>
          <div v-if="form.rule_type === 'MASK'" class="parameter-grid">
            <div class="parameter-item">
              <span>{{ t("system.base.redact_rule.parameter.keep_first") }}</span>
              <el-input-number v-model="form.params.keep_first" :min="0" :precision="0" controls-position="right" />
            </div>
            <div class="parameter-item">
              <span>{{ t("system.base.redact_rule.parameter.keep_last") }}</span>
              <el-input-number v-model="form.params.keep_last" :min="0" :precision="0" controls-position="right" />
            </div>
            <div class="parameter-item">
              <span>{{ t("system.base.redact_rule.parameter.mask_char") }}</span>
              <el-input v-model="form.params.mask_char" maxlength="8" />
            </div>
          </div>
          <div v-else-if="form.rule_type === 'EMAIL'" class="parameter-grid">
            <div class="parameter-item">
              <span>{{ t("system.base.redact_rule.parameter.keep_local_first") }}</span>
              <el-input-number v-model="form.params.keep_local_first" :min="0" :precision="0" controls-position="right" />
            </div>
            <div class="parameter-item">
              <span>{{ t("system.base.redact_rule.parameter.mask_domain") }}</span>
              <el-switch v-model="form.params.mask_domain" />
            </div>
            <div class="parameter-item">
              <span>{{ t("system.base.redact_rule.parameter.mask_char") }}</span>
              <el-input v-model="form.params.mask_char" maxlength="8" />
            </div>
          </div>
          <div v-else-if="form.rule_type === 'REGEX'" class="parameter-grid parameter-grid--single">
            <div class="parameter-item parameter-item--wide">
              <span>{{ t("system.base.redact_rule.parameter.pattern") }}</span>
              <el-input v-model="form.params.pattern" />
            </div>
            <div class="parameter-item parameter-item--wide">
              <span>{{ t("system.base.redact_rule.parameter.replacement") }}</span>
              <el-input v-model="form.params.replacement" />
            </div>
          </div>
          <div v-else-if="form.rule_type === 'TRUNCATE'" class="parameter-grid">
            <div class="parameter-item">
              <span>{{ t("system.base.redact_rule.parameter.length") }}</span>
              <el-input-number v-model="form.params.length" :min="0" :precision="0" controls-position="right" />
            </div>
            <div class="parameter-item parameter-item--wide">
              <span>{{ t("system.base.redact_rule.parameter.suffix") }}</span>
              <el-input v-model="form.params.suffix" maxlength="32" />
            </div>
          </div>
          <div v-else-if="form.rule_type === 'HASH'" class="parameter-grid parameter-grid--single">
            <div class="parameter-item parameter-item--wide">
              <span>{{ t("system.base.redact_rule.parameter.algo") }}</span>
              <el-select v-model="form.params.algo" class="parameter-control">
                <el-option label="MD5" value="MD5" />
                <el-option label="SHA1" value="SHA1" />
                <el-option label="SHA256" value="SHA256" />
              </el-select>
            </div>
          </div>
          <div v-else-if="form.rule_type === 'IP'" class="parameter-grid">
            <div class="parameter-item">
              <span>{{ t("system.base.redact_rule.parameter.keep_octets") }}</span>
              <el-input-number v-model="form.params.keep_octets" :min="0" :precision="0" controls-position="right" />
            </div>
            <div class="parameter-item">
              <span>{{ t("system.base.redact_rule.parameter.mask_char") }}</span>
              <el-input v-model="form.params.mask_char" maxlength="8" />
            </div>
          </div>
          <div v-else-if="form.rule_type === 'URL'" class="parameter-grid">
            <div class="parameter-item">
              <span>{{ t("system.base.redact_rule.parameter.mask_query") }}</span>
              <el-switch v-model="form.params.mask_query" />
            </div>
            <div class="parameter-item">
              <span>{{ t("system.base.redact_rule.parameter.mask_char") }}</span>
              <el-input v-model="form.params.mask_char" maxlength="8" />
            </div>
          </div>
          <div v-else-if="form.rule_type === 'FIXED_LENGTH'" class="parameter-grid parameter-grid--single">
            <div class="parameter-item parameter-item--wide">
              <span>{{ t("system.base.redact_rule.parameter.char") }}</span>
              <el-input v-model="form.params.char" maxlength="8" />
            </div>
          </div>
          <div v-else class="rule-empty">{{ t("system.base.redact_rule.message.no_parameters") }}</div>
        </div>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { EditPen } from "@element-plus/icons-vue";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseRedactRuleService } from "@liujitcn/kratos-admin-system/api/system/base_redact_rule";
import type { BaseRedactRule, BaseRedactRuleForm, PageBaseRedactRuleRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_rule";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

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

interface RuleFormState extends BaseRedactRuleForm {
  params: RuleParams;
}

defineOptions({ name: "BaseRedactRule", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const table = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ visible: false, titleKey: "common.action.edit_resource" });
const form = reactive<RuleFormState>(defaultForm());
const statusOptions = computed(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);
const templateDescriptionKey = computed(() => `system.base.redact_rule.description.${form.code || "unknown"}`);

const fields = computed<ProFormField[]>(() => [
  { prop: "code", label: t("system.base.redact_rule.field.code"), component: "input", props: { disabled: true } },
  { prop: "name", label: t("system.base.redact_rule.field.name"), component: "input", props: { disabled: true } },
  { prop: "rule_type", label: t("system.base.redact_rule.field.rule_type"), component: "input", props: { disabled: true } },
  { prop: "rule", label: t("system.base.redact_rule.field.parameters"), component: "slot", slotName: "rule", colSpan: 24 },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value },
  { prop: "remark", label: t("common.field.remark"), component: "textarea" }
]);

const rules = computed(() => ({
  rule: [{ required: true, message: t("system.base.redact_rule.validation.parameters"), trigger: "change" }]
}));

const columns = computed<ColumnProps[]>(() => [
  { prop: "code", label: t("system.base.redact_rule.field.code"), minWidth: 180, search: { el: "input" } },
  { prop: "name", label: t("system.base.redact_rule.field.name"), minWidth: 180, search: { el: "input" } },
  { prop: "rule_type", label: t("system.base.redact_rule.field.rule_type"), width: 150, search: { el: "input" } },
  { prop: "rule", label: t("system.base.redact_rule.field.parameters"), minWidth: 260, showOverflowTooltip: true, render: scope => ruleSummary(scope.row as BaseRedactRule) },
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
      disabled: () => !BUTTONS.value["base:redact-rule:status"],
      beforeChange: scope => changeStatus(scope.row as BaseRedactRule)
    }
  },
  {
    prop: "actions",
    label: t("common.field.operation"),
    width: 100,
    fixed: "right",
    cellType: "actions",
    actions: [{ label: t("common.action.edit"), type: "primary", link: true, icon: EditPen, hidden: () => !BUTTONS.value["base:redact-rule:update"], onClick: scope => openDialog((scope.row as BaseRedactRule).id) }]
  }
]);

/** 请求固定规则模板列表。 */
async function requestTable(params: PageBaseRedactRuleRequest) {
  const data = await defBaseRedactRuleService.PageBaseRedactRule(buildPageRequest(params));
  return { data: { list: data.base_redact_rules ?? [], total: data.total } };
}

/** 打开固定规则模板编辑弹窗。 */
async function openDialog(id: number) {
  resetForm();
  const data = await defBaseRedactRuleService.GetBaseRedactRule({ id });
  Object.assign(form, data, { params: parseRuleParams(data.rule_type, data.rule) });
  dialog.visible = true;
}

/** 重置规则模板编辑表单。 */
function resetForm() {
  dialog.visible = false;
  dialogRef.value?.resetFields();
  Object.assign(form, defaultForm());
}

/** 提交固定规则模板参数。 */
async function submit() {
  const valid = await dialogRef.value?.validate();
  if (!valid) return;
  syncRule();
  const payload: BaseRedactRuleForm = {
    id: form.id,
    code: form.code,
    name: form.name,
    rule_type: form.rule_type,
    rule: form.rule,
    status: form.status,
    remark: form.remark
  };
  await defBaseRedactRuleService.UpdateBaseRedactRule({ base_redact_rule: payload });
  ElMessage.success(t("common.message.update_success", { resource: t("system.base.redact_rule.title") }));
  resetForm();
  table.value?.getTableList();
}

/** 切换固定规则模板状态。 */
async function changeStatus(row: BaseRedactRule) {
  const next = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action: t(next === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled"),
        resource: t("system.base.redact_rule.title"),
        field: t("system.base.redact_rule.field.code"),
        value: row.code
      }),
      t("common.title.warning"),
      { type: "warning" }
    );
    await defBaseRedactRuleService.SetBaseRedactRuleStatus({ id: row.id, status: next });
    table.value?.getTableList();
    return true;
  } catch {
    return false;
  }
}

/** 将参数表单同步为运行时规则 JSON。 */
function syncRule() {
  form.params = { ...defaultRuleParams(form.rule_type), ...form.params };
  form.rule = JSON.stringify({ [form.rule_type.toLowerCase()]: form.params }, null, 2);
}

/** 解析规则 JSON 中当前类型的参数。 */
function parseRuleParams(ruleType: string, rawRule: string): RuleParams {
  try {
    const value = JSON.parse(rawRule) as Record<string, unknown>;
    const params = value[ruleType.toLowerCase()];
    if (params && typeof params === "object" && !Array.isArray(params)) return params as RuleParams;
  } catch {
    return defaultRuleParams(ruleType);
  }
  return defaultRuleParams(ruleType);
}

/** 返回固定规则模板的默认参数。 */
function defaultRuleParams(ruleType: string): RuleParams {
  switch (ruleType) {
    case "MASK":
      return { keep_first: 3, keep_last: 4, mask_char: "*" };
    case "EMAIL":
      return { keep_local_first: 2, mask_domain: false, mask_char: "*" };
    case "REGEX":
      return { pattern: "(?s).+", replacement: "[REDACTED]" };
    case "TRUNCATE":
      return { length: 10, suffix: "..." };
    case "HASH":
      return { algo: "SHA256" };
    case "IP":
      return { keep_octets: 2, mask_char: "x" };
    case "URL":
      return { mask_query: true, mask_char: "*" };
    case "FIXED_LENGTH":
      return { char: "X" };
    default:
      return {};
  }
}

/** 返回规则参数的紧凑展示文本。 */
function ruleSummary(row: BaseRedactRule) {
  return JSON.stringify(parseRuleParams(row.rule_type, row.rule));
}

/** 返回规则模板表单的初始值。 */
function defaultForm(): RuleFormState {
  return { id: 0, code: "", name: "", rule_type: "MASK", rule: "", status: Status.STATUS_ENABLE, remark: "", params: defaultRuleParams("MASK") };
}
</script>

<style scoped lang="scss">
.rule-editor {
  width: 100%;
}

.rule-description {
  margin-bottom: 12px;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}

.parameter-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.parameter-grid--single {
  grid-template-columns: minmax(0, 1fr);
}

.parameter-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  color: var(--el-text-color-regular);
}

.parameter-item--wide {
  min-width: 0;
}

.parameter-control {
  width: 100%;
}

.rule-empty {
  color: var(--el-text-color-secondary);
}

@media (max-width: 900px) {
  .parameter-grid {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
