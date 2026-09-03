<template>
  <div class="login-policy-page">
    <el-card class="admin-page-card">
      <ProTable ref="proTable" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    </el-card>

    <FormDialog
      ref="formDialogRef"
      v-model="dialog.visible"
      :title="t(dialog.titleKey)"
      width="980px"
      :model="formData"
      :fields="formFields"
      :rules="formRules"
      @confirm="handleSubmit"
      @close="handleClose"
    >
      <template #rules>
        <div class="rules-editor">
          <div v-for="(rule, index) in formData.rules" :key="rule.id || index" class="rule-row">
            <el-select v-model="rule.restriction_type" class="rule-type">
              <el-option v-for="option in restrictionTypeOptions" :key="String(option.value)" :label="option.label" :value="option.value" />
            </el-select>
            <el-select v-model="rule.restriction_method" class="rule-method">
              <el-option v-for="option in restrictionMethodOptions" :key="String(option.value)" :label="option.label" :value="option.value" />
            </el-select>
            <el-input v-model="rule.restriction_value" class="rule-value" :placeholder="rulePlaceholder(rule.restriction_method)" />
            <el-input v-model="rule.reason" class="rule-reason" :placeholder="t('system.base.login_policy.placeholder.reason')" />
            <el-select v-model="rule.status" class="rule-status">
              <el-option v-for="option in statusOptions" :key="String(option.value)" :label="option.label" :value="option.value" />
            </el-select>
            <el-button link type="danger" :icon="Delete" @click="removeRule(index)" />
          </div>
          <el-button link type="primary" :icon="CirclePlus" @click="addRule">
            {{ t("system.base.login_policy.rules.add") }}
          </el-button>
        </div>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, watch } from "vue";
import { CirclePlus, Delete, EditPen } from "@element-plus/icons-vue";
import { ElButton, ElMessage, ElMessageBox, ElTag } from "element-plus";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { PASSWORD_CRYPTO_SCENE, encryptPassword } from "@liujitcn/kratos-admin-core/security";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseLoginPolicyService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_login_policy";
import { defBaseTenantService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_tenant";
import { defBaseUserService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_user";
import type { SelectOptionResponse_Option } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import type {
  BaseLoginPolicy,
  BaseLoginPolicyForm,
  BaseLoginPolicyRule,
  PageBaseLoginPolicyRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_login_policy";
import {
  BaseLoginPolicyRestrictionMethod,
  BaseLoginPolicyRestrictionType,
  BaseLoginPolicyScopeType
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_login_policy";

/** 登录策略表单状态，初始化密码仅在提交时转换为一次性密码密文。 */
interface BaseLoginPolicyFormState extends Omit<BaseLoginPolicyForm, "initial_password"> {
  /** 初始化密码明文只保留在当前表单，留空表示不修改已有值。 */
  initial_password: string;
}

defineOptions({ name: "BaseLoginPolicy", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({ visible: false, titleKey: "common.action.create" });
const tenantOptions = ref<SelectOptionResponse_Option[]>([]);
const userOptions = ref<SelectOptionResponse_Option[]>([]);
const formData = reactive<BaseLoginPolicyFormState>(defaultForm());

const scopeTypeOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.login_policy.scope.global"), value: BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_GLOBAL },
  { label: t("system.base.login_policy.scope.tenant"), value: BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_TENANT },
  { label: t("system.base.login_policy.scope.user"), value: BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_USER }
]);
const restrictionTypeOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.login_policy.restriction.blacklist"), value: BaseLoginPolicyRestrictionType.BASE_LOGIN_POLICY_RESTRICTION_TYPE_BLACKLIST },
  { label: t("system.base.login_policy.restriction.whitelist"), value: BaseLoginPolicyRestrictionType.BASE_LOGIN_POLICY_RESTRICTION_TYPE_WHITELIST }
]);
const restrictionMethodOptions = computed<ProFormOption[]>(() => [
  { label: t("system.base.login_policy.method.ip"), value: BaseLoginPolicyRestrictionMethod.BASE_LOGIN_POLICY_RESTRICTION_METHOD_IP },
  { label: t("system.base.login_policy.method.mac"), value: BaseLoginPolicyRestrictionMethod.BASE_LOGIN_POLICY_RESTRICTION_METHOD_MAC },
  { label: t("system.base.login_policy.method.region"), value: BaseLoginPolicyRestrictionMethod.BASE_LOGIN_POLICY_RESTRICTION_METHOD_REGION },
  { label: t("system.base.login_policy.method.time"), value: BaseLoginPolicyRestrictionMethod.BASE_LOGIN_POLICY_RESTRICTION_METHOD_TIME },
  { label: t("system.base.login_policy.method.device"), value: BaseLoginPolicyRestrictionMethod.BASE_LOGIN_POLICY_RESTRICTION_METHOD_DEVICE }
]);
const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);

const formFields = computed<ProFormField[]>(() => [
  { prop: "scope_type", label: t("system.base.login_policy.field.scope_type"), component: "select", options: scopeTypeOptions.value },
  {
    prop: "tenant_id",
    label: t("system.base.login_policy.field.tenant"),
    component: "select",
    options: tenantOptions.value,
    visible: model => model.scope_type !== BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_GLOBAL,
    props: { filterable: true, clearable: true }
  },
  {
    prop: "user_id",
    label: t("system.base.login_policy.field.user"),
    component: "select",
    options: userOptions.value,
    visible: model => model.scope_type === BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_USER,
    props: { filterable: true, clearable: true }
  },
  {
    prop: "max_failed_attempts",
    label: t("system.base.login_policy.field.max_failed_attempts"),
    component: "input-number",
    props: { min: 1, max: 1000, precision: 0 }
  },
  {
    prop: "lock_duration_minutes",
    label: t("system.base.login_policy.field.lock_duration_minutes"),
    component: "input-number",
    props: { min: 1, max: 1440, precision: 0 }
  },
  {
    prop: "password_min_length",
    label: t("system.base.login_policy.field.password_min_length"),
    component: "input-number",
    props: { min: 1, max: 128, precision: 0 }
  },
  {
    prop: "password_history_count",
    label: t("system.base.login_policy.field.password_history_count"),
    component: "input-number",
    props: { min: 0, max: 100, precision: 0 }
  },
  {
    prop: "password_min_complexity_classes",
    label: t("system.base.login_policy.field.password_min_complexity_classes"),
    component: "input-number",
    props: { min: 1, max: 4, precision: 0 }
  },
  {
    prop: "password_max_age_days",
    label: t("system.base.login_policy.field.password_max_age_days"),
    component: "input-number",
    props: { min: 0, max: 3650, precision: 0 }
  },
  {
    prop: "initial_password",
    label: t("system.base.login_policy.field.initial_password"),
    component: "password",
    props: { placeholder: t("system.base.login_policy.placeholder.initial_password"), showPassword: true }
  },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value },
  { prop: "rules", label: t("system.base.login_policy.field.rules"), component: "slot", slotName: "rules", colSpan: 24 }
]);

const formRules = computed(() => ({
  scope_type: [{ required: true, message: t("system.base.login_policy.validation.scope_type"), trigger: "change" }],
  tenant_id: [{ validator: validateTenant, trigger: "change" }],
  user_id: [{ validator: validateUser, trigger: "change" }],
  max_failed_attempts: [{ required: true, message: t("system.base.login_policy.validation.max_failed_attempts"), trigger: "change" }],
  lock_duration_minutes: [{ required: true, message: t("system.base.login_policy.validation.lock_duration_minutes"), trigger: "change" }],
  password_min_length: [{ required: true, message: t("system.base.login_policy.validation.password_min_length"), trigger: "change" }],
  password_history_count: [{ required: true, message: t("system.base.login_policy.validation.password_history_count"), trigger: "change" }],
  password_min_complexity_classes: [
    { required: true, message: t("system.base.login_policy.validation.password_min_complexity_classes"), trigger: "change" }
  ],
  password_max_age_days: [{ required: true, message: t("system.base.login_policy.validation.password_max_age_days"), trigger: "change" }]
}));

const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  {
    prop: "scope_type",
    label: t("system.base.login_policy.field.scope_type"),
    width: 110,
    search: { el: "select", enum: scopeTypeOptions.value },
    render: scope => scopeTypeLabel((scope.row as BaseLoginPolicy).scope_type)
  },
  {
    prop: "target",
    label: t("system.base.login_policy.field.target"),
    minWidth: 160,
    render: scope => targetLabel(scope.row as BaseLoginPolicy)
  },
  {
    prop: "rules",
    label: t("system.base.login_policy.field.rules"),
    minWidth: 320,
    render: scope => renderRules(scope.row as BaseLoginPolicy)
  },
  { prop: "max_failed_attempts", label: t("system.base.login_policy.field.max_failed_attempts"), width: 110, align: "right" },
  { prop: "lock_duration_minutes", label: t("system.base.login_policy.field.lock_duration_minutes"), width: 110, align: "right" },
  { prop: "password_min_length", label: t("system.base.login_policy.field.password_min_length"), width: 110, align: "right" },
  { prop: "password_history_count", label: t("system.base.login_policy.field.password_history_count"), width: 110, align: "right" },
  {
    prop: "password_min_complexity_classes",
    label: t("system.base.login_policy.field.password_min_complexity_classes"),
    width: 130,
    align: "right"
  },
  { prop: "password_max_age_days", label: t("system.base.login_policy.field.password_max_age_days"), width: 110, align: "right" },
  {
    prop: "status",
    label: t("common.field.status"),
    width: 110,
    search: { el: "select", enum: statusOptions.value },
    cellType: "status",
    statusProps: {
      activeValue: Status.STATUS_ENABLE,
      inactiveValue: Status.STATUS_DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: () => !BUTTONS.value["base:login-policy:status"],
      beforeChange: scope => handleSetStatus(scope.row as BaseLoginPolicy)
    }
  },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.edit"),
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:login-policy:update"],
        onClick: scope => openDialog((scope.row as BaseLoginPolicy).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:login-policy:delete"],
        onClick: scope => handleDelete(scope.row as BaseLoginPolicy)
      }
    ]
  }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "primary",
    icon: CirclePlus,
    hidden: !BUTTONS.value["base:login-policy:create"],
    onClick: () => openDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: !BUTTONS.value["base:login-policy:delete"],
    disabled: scope => !scope.isSelected,
    onClick: scope => handleDelete(scope.selectedListIds as number[])
  }
]);

watch(
  () => formData.scope_type,
  value => {
    if (value === BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_GLOBAL) {
      formData.tenant_id = 0;
      formData.user_id = 0;
      userOptions.value = [];
      return;
    }
    if (value === BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_TENANT) {
      formData.user_id = 0;
      userOptions.value = [];
    }
  }
);

watch(
  () => formData.tenant_id,
  async tenantId => {
    if (formData.scope_type !== BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_USER || !tenantId) {
      userOptions.value = [];
      return;
    }
    const response = await defBaseUserService.OptionBaseUser({ keyword: "", tenant_id: tenantId });
    userOptions.value = response.list ?? [];
  }
);

/** 创建默认登录策略表单。 */
function defaultForm(): BaseLoginPolicyFormState {
  return {
    id: 0,
    scope_type: BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_GLOBAL,
    tenant_id: 0,
    user_id: 0,
    max_failed_attempts: 5,
    lock_duration_minutes: 15,
    password_min_length: 8,
    password_history_count: 3,
    password_min_complexity_classes: 3,
    password_max_age_days: 90,
    initial_password: "",
    status: Status.STATUS_ENABLE,
    rules: []
  };
}

/** 创建默认限制规则。 */
function defaultRule(): BaseLoginPolicyRule {
  return {
    id: 0,
    policy_id: 0,
    restriction_type: BaseLoginPolicyRestrictionType.BASE_LOGIN_POLICY_RESTRICTION_TYPE_BLACKLIST,
    restriction_method: BaseLoginPolicyRestrictionMethod.BASE_LOGIN_POLICY_RESTRICTION_METHOD_IP,
    restriction_value: "",
    reason: "",
    status: Status.STATUS_ENABLE
  };
}

/** 请求登录策略表格数据。 */
async function requestTable(params: Record<string, unknown>) {
  const data = await defBaseLoginPolicyService.PageBaseLoginPolicy(
    buildPageRequest<PageBaseLoginPolicyRequest>(params as unknown as PageBaseLoginPolicyRequest)
  );
  return { data: { list: data.base_login_policies ?? [], total: data.total } };
}

/** 打开登录策略表单。 */
async function openDialog(id?: number) {
  Object.assign(formData, defaultForm());
  dialog.titleKey = id ? "common.action.edit" : "common.action.create";
  await loadTenantOptions();
  if (id) {
    const detail = await defBaseLoginPolicyService.GetBaseLoginPolicy({ id });
    Object.assign(formData, detail, { initial_password: "", rules: detail.rules ?? [] });
    if (formData.scope_type === BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_USER && formData.tenant_id) {
      await loadUserOptions(formData.tenant_id);
    }
  }
  dialog.visible = true;
}

/** 加载租户选项。 */
async function loadTenantOptions() {
  const response = await defBaseTenantService.OptionBaseTenant({ keyword: "" });
  tenantOptions.value = response.list ?? [];
}

/** 加载指定租户的用户选项。 */
async function loadUserOptions(tenantId: number) {
  const response = await defBaseUserService.OptionBaseUser({ keyword: "", tenant_id: tenantId });
  userOptions.value = response.list ?? [];
}

/** 提交登录策略表单。 */
async function handleSubmit() {
  const valid = await formDialogRef.value?.validate();
  if (!valid) return;

  const baseLoginPolicy: BaseLoginPolicyForm = {
    ...formData,
    initial_password: formData.initial_password
      ? await encryptPassword(formData.initial_password, PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_CONFIGURE_PASSWORD_POLICY)
      : undefined
  };
  if (formData.id) await defBaseLoginPolicyService.UpdateBaseLoginPolicy({ base_login_policy: baseLoginPolicy });
  else await defBaseLoginPolicyService.CreateBaseLoginPolicy({ base_login_policy: baseLoginPolicy });
  ElMessage.success(t("common.message.operation_success"));
  dialog.visible = false;
  proTable.value?.getTableList();
}

/** 删除登录策略。 */
async function handleDelete(value: BaseLoginPolicy | BaseLoginPolicy[] | number | number[]) {
  const ids = normalizeSelectedIds(value as Parameters<typeof normalizeSelectedIds>[0]);
  await ElMessageBox.confirm(t("common.confirm.delete"), t("common.title.warning"), { type: "warning" });
  await defBaseLoginPolicyService.DeleteBaseLoginPolicy({ id: ids.join(",") });
  ElMessage.success(t("common.message.operation_success"));
  proTable.value?.getTableList();
}

/** 设置登录策略状态。 */
async function handleSetStatus(row: BaseLoginPolicy) {
  const status = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  await defBaseLoginPolicyService.SetBaseLoginPolicyStatus({ id: row.id, status });
  return true;
}

/** 关闭登录策略表单。 */
function handleClose() {
  Object.assign(formData, defaultForm());
  userOptions.value = [];
  formDialogRef.value?.resetFields();
}

/** 新增限制规则。 */
function addRule() {
  formData.rules.push(defaultRule());
}

/** 删除限制规则。 */
function removeRule(index: number) {
  formData.rules.splice(index, 1);
}

/** 校验租户作用域目标。 */
function validateTenant(_rule: unknown, value: number, callback: (error?: Error) => void) {
  if (formData.scope_type !== BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_GLOBAL && !value) {
    callback(new Error(t("system.base.login_policy.validation.tenant")));
    return;
  }
  callback();
}

/** 校验用户作用域目标。 */
function validateUser(_rule: unknown, value: number, callback: (error?: Error) => void) {
  if (formData.scope_type === BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_USER && !value) {
    callback(new Error(t("system.base.login_policy.validation.user")));
    return;
  }
  callback();
}

/** 输出作用域名称。 */
function scopeTypeLabel(value: BaseLoginPolicyScopeType) {
  return scopeTypeOptions.value.find(option => option.value === value)?.label ?? String(value);
}

/** 输出策略目标名称。 */
function targetLabel(row: BaseLoginPolicy) {
  if (row.scope_type === BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_GLOBAL) return t("system.base.login_policy.scope.global");
  if (row.scope_type === BaseLoginPolicyScopeType.BASE_LOGIN_POLICY_SCOPE_TYPE_USER) return row.user_name || String(row.user_id);
  return row.tenant_name || String(row.tenant_id);
}

/** 输出限制规则摘要。 */
function renderRules(row: BaseLoginPolicy) {
  const rules = row.rules ?? [];
  if (!rules.length) return h("span", { class: "muted" }, t("system.base.login_policy.rules.empty"));
  return h(
    "div",
    { class: "rule-summary" },
    rules.map((rule, index) =>
      h(
        ElTag,
        { key: `${rule.id || index}`, size: "small", class: "rule-tag" },
        () => `${restrictionTypeLabel(rule.restriction_type)}/${restrictionMethodLabel(rule.restriction_method)}: ${rule.restriction_value}`
      )
    )
  );
}

/** 输出限制类型名称。 */
function restrictionTypeLabel(value: BaseLoginPolicyRestrictionType) {
  return restrictionTypeOptions.value.find(option => option.value === value)?.label ?? String(value);
}

/** 输出限制方式名称。 */
function restrictionMethodLabel(value: BaseLoginPolicyRestrictionMethod) {
  return restrictionMethodOptions.value.find(option => option.value === value)?.label ?? String(value);
}

/** 根据限制方式输出输入提示。 */
function rulePlaceholder(method: BaseLoginPolicyRestrictionMethod) {
  switch (method) {
    case BaseLoginPolicyRestrictionMethod.BASE_LOGIN_POLICY_RESTRICTION_METHOD_IP:
      return t("system.base.login_policy.placeholder.ip");
    case BaseLoginPolicyRestrictionMethod.BASE_LOGIN_POLICY_RESTRICTION_METHOD_TIME:
      return t("system.base.login_policy.placeholder.time");
    case BaseLoginPolicyRestrictionMethod.BASE_LOGIN_POLICY_RESTRICTION_METHOD_REGION:
      return t("system.base.login_policy.placeholder.region");
    default:
      return t("system.base.login_policy.placeholder.value");
  }
}
</script>

<style scoped lang="scss">
.login-policy-page {
  min-width: 0;
}
.rules-editor {
  width: 100%;
}
.rule-row {
  display: grid;
  grid-template-columns: 120px 140px minmax(180px, 1fr) minmax(160px, 1fr) 110px 32px;
  gap: 8px;
  align-items: center;
  margin-bottom: 10px;
}
.rule-summary {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.rule-tag {
  max-width: 100%;
}
.muted {
  color: var(--el-text-color-secondary);
}
@media (max-width: 960px) {
  .rule-row {
    grid-template-columns: 1fr 1fr;
  }
  .rule-value,
  .rule-reason {
    grid-column: span 2;
  }
}
</style>
