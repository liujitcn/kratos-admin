<!-- 用户管理 -->
<template>
  <div class="main-box">
    <TreeFilter
      :key="`dept-filter-${selectedTenantId ?? 0}`"
      label="name"
      :title="t('system.base.user.title.department_list')"
      :request-api="requestDeptTreeFilter"
      :show-all="false"
      :default-value="deptFilterValue"
      @change="changeTreeFilter"
    />

    <div class="table-box">
      <ProTable
        ref="proTable"
        :key="`user-table-${isDefaultTenant ? 'default-tenant' : 'current'}`"
        row-key="id"
        :columns="columns"
        :header-actions="headerActions"
        :request-api="requestBaseUserTable"
        :init-param="initParam"
      />
    </div>

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.editing ? 'system.base.user.action.edit' : 'system.base.user.action.create')"
      width="800px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    >
      <template #passwordStrength>
        <PasswordStrength :password="formData.pwd" />
      </template>
    </FormDialog>

    <FormDialog
      v-model="resetPwdDialog.visible"
      ref="resetPwdFormDialogRef"
      :title="t('system.base.user.title.reset_password', { name: resetPwdTargetName })"
      width="520px"
      :model="resetPwdForm"
      :fields="resetPwdFields"
      :rules="resetPwdRules"
      @confirm="handleConfirmResetPassword"
      @close="handleCloseResetPasswordDialog"
    >
      <template #resetPwdStrength>
        <PasswordStrength :password="resetPwdForm.pwd" />
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { useDebounceFn } from "@vueuse/core";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, RefreshLeft } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import PasswordStrength from "@liujitcn/kratos-admin-core/components/PasswordStrength/index.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import TreeFilter from "@liujitcn/kratos-admin-core/components/TreeFilter/index.vue";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { defBaseUserService } from "@liujitcn/kratos-admin-system/api/system/base_user";
import type {
  BaseUser,
  BaseUserForm,
  PageBaseUserRequest,
  ResetBaseUserPasswordRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_user";
import { defBaseDeptService } from "@liujitcn/kratos-admin-system/api/system/base_dept";
import { defBaseRoleService } from "@liujitcn/kratos-admin-system/api/system/base_role";
import { defBasePostService } from "@liujitcn/kratos-admin-system/api/system/base_post";
import { defBaseTenantService } from "@liujitcn/kratos-admin-system/api/system/base_tenant";
import type { SelectOptionResponse_Option, TreeOptionResponse_Option } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { PASSWORD_CRYPTO_SCENE, encryptPassword, validatePasswordStrengthValue } from "@liujitcn/kratos-admin-core/security";
import { DEFAULT_TENANT_CODE, requestTenantOptions } from "@liujitcn/kratos-admin-core/tenant";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import { t } from "@liujitcn/kratos-admin-core";

/** 用户表单状态，前端保留明文密码并在提交前加密。 */
interface BaseUserFormState extends Omit<BaseUserForm, "dept_id" | "post_id" | "pwd" | "role_id" | "tenant_id"> {
  /** 租户ID，默认租户新增时必须由管理员显式选择。 */
  tenant_id?: number;
  /** 部门ID，未选择时保持空白。 */
  dept_id?: number;
  /** 岗位ID，未选择时保持空白。 */
  post_id?: number;
  /** 角色ID，未选择时保持空白。 */
  role_id?: number;
  /** 密码明文只保留在前端表单中，提交前转换为密码密文。 */
  pwd: string;
}

/** 重置用户密码表单状态，前端保留明文密码并在提交前加密。 */
interface ResetBaseUserPasswordFormState extends Omit<ResetBaseUserPasswordRequest, "pwd"> {
  /** 密码明文只保留在前端表单中，提交前转换为密码密文。 */
  pwd: string;
}

defineOptions({
  name: "BaseUser",
  inheritAttrs: false
});

/** 用户管理左侧部门筛选树节点。 */
type DeptFilterNode = {
  id: string;
  name: string;
  children?: DeptFilterNode[];
};

const { BUTTONS } = useAuthButtons();
const userStore = useUserStore();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const resetPwdFormDialogRef = ref<InstanceType<typeof FormDialog>>();

const initParam = reactive({
  dept_id: undefined as number | undefined
});
const selectedTenantId = ref<number | undefined>();
const deptFilterValue = ref("");

const dialog = reactive({
  visible: false,
  editing: false
});
const resetPwdDialog = reactive({
  visible: false
});

const formData = reactive<BaseUserFormState>({
  /** 用户ID */
  id: 0,
  /** 租户ID */
  tenant_id: undefined,
  /** 用户账号 */
  user_name: "",
  /** 用户编号 */
  user_code: "",
  /** 用户昵称 */
  nick_name: "",
  /** 角色ID */
  role_id: undefined,
  /** 部门ID */
  dept_id: undefined,
  /** 岗位ID */
  post_id: undefined,
  /** 手机号 */
  phone: "",
  /** 密码 */
  pwd: "",
  /** 性别 */
  gender: 3,
  /** 头像 */
  avatar: "",
  /** 用户状态 */
  status: Status.STATUS_ENABLE,
  /** 备注名 */
  remark: ""
});
const resetPwdForm = reactive<ResetBaseUserPasswordFormState>({
  id: 0,
  pwd: ""
});
const resetPwdTargetName = ref("");

const rules = computed(() => ({
  tenant_id: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("system.base.user.field.tenant") }),
      trigger: "change"
    }
  ],
  user_name: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.user.field.user_name") }),
      trigger: "blur"
    },
    {
      max: 50,
      message: t("common.validation.max_length", { field: t("system.base.user.field.user_name"), max: 50 }),
      trigger: "blur"
    }
  ],
  user_code: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.user.field.user_code") }),
      trigger: "blur"
    },
    {
      max: 30,
      message: t("common.validation.max_length", { field: t("system.base.user.field.user_code"), max: 30 }),
      trigger: "blur"
    }
  ],
  nick_name: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.user.field.nick_name") }),
      trigger: "blur"
    },
    {
      max: 30,
      message: t("common.validation.max_length", { field: t("system.base.user.field.nick_name"), max: 30 }),
      trigger: "blur"
    }
  ],
  dept_id: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("system.base.user.field.department") }),
      trigger: "change"
    }
  ],
  role_id: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("system.base.user.field.role") }),
      trigger: "change"
    }
  ],
  phone: [
    { max: 20, message: t("common.validation.max_length", { field: t("common.field.phone"), max: 20 }), trigger: "blur" },
    {
      pattern: /^1[3-9]\d{9}$/,
      message: t("common.validation.phone_invalid"),
      trigger: "blur"
    }
  ],
  pwd: [
    { required: true, message: t("core.password.required"), trigger: "blur" },
    { validator: validatePasswordField, trigger: "blur" }
  ],
  status: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("common.field.status") }),
      trigger: "change"
    }
  ],
  remark: [
    {
      max: 500,
      message: t("common.validation.max_length", { field: t("common.field.remark"), max: 500 }),
      trigger: "blur"
    }
  ]
}));
const resetPwdRules = computed(() => ({
  pwd: [
    { required: true, message: t("system.base.user.placeholder.new_password"), trigger: "blur" },
    { validator: validatePasswordField, trigger: "blur" }
  ]
}));

const basedDeptOptions = ref<TreeOptionResponse_Option[]>([]);
const baseRoleOptions = ref<SelectOptionResponse_Option[]>([]);
const basePostOptions = ref<SelectOptionResponse_Option[]>([]);
const tenantOptions = ref<SelectOptionResponse_Option[]>([]);
const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);
const resetPwdFields = computed<ProFormField[]>(() => [
  {
    prop: "pwd",
    label: t("system.base.user.field.new_password"),
    component: "password",
    props: { placeholder: t("system.base.user.placeholder.new_password"), showPassword: true }
  },
  {
    prop: "resetPwdStrength",
    label: t("system.base.user.field.password_strength"),
    component: "slot",
    slotName: "resetPwdStrength"
  }
]);

/** 当前登录账号是否默认租户。 */
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);

/** 当前编辑用户是否绑定 super 或 tenant 内置角色。 */
const isProtectedUserRole = computed(() => {
  if (!formData.id || !formData.role_id) return false;
  return baseRoleOptions.value.some(item => item.value === formData.role_id && item.disabled);
});

/** 当前是否正在编辑受状态保护的超级管理员。 */
const isSuperEditUser = computed(() => Boolean(formData.id && formData.user_name === "super"));

/** 用户表单字段配置。 */
const formFields = computed<ProFormField[]>(() => [
  {
    prop: "tenant_id",
    label: t("system.base.user.field.tenant"),
    component: "select",
    props: {
      placeholder: t("system.base.user.placeholder.tenant"),
      filterable: true,
      disabled: Boolean(formData.id),
      onChange: handleFormTenantChange
    },
    visible: () => isDefaultTenant.value,
    options: tenantOptions.value
  },
  {
    prop: "user_name",
    label: t("system.base.user.field.user_name"),
    component: "input",
    props: { placeholder: t("system.base.user.placeholder.user_name"), disabled: Boolean(formData.id) }
  },
  {
    prop: "user_code",
    label: t("system.base.user.field.user_code"),
    component: "input",
    props: { placeholder: t("system.base.user.placeholder.user_code"), disabled: Boolean(formData.id) }
  },
  {
    prop: "nick_name",
    label: t("system.base.user.field.nick_name"),
    component: "input",
    props: { placeholder: t("system.base.user.placeholder.nick_name") }
  },
  {
    prop: "role_id",
    label: t("system.base.user.field.role"),
    component: "select",
    options: baseRoleOptions.value.map(item => ({ label: item.label, value: item.value, disabled: item.disabled })),
    props: { placeholder: t("common.placeholder.select"), disabled: isProtectedUserRole.value }
  },
  {
    prop: "dept_id",
    label: t("system.base.user.field.department"),
    component: "tree-select",
    options: basedDeptOptions.value as unknown as ProFormOption[],
    props: {
      placeholder: t("system.base.user.placeholder.department"),
      filterable: true,
      checkStrictly: true,
      renderAfterExpand: false,
      style: { width: "100%" }
    }
  },
  {
    prop: "post_id",
    label: t("system.base.user.field.post"),
    component: "select",
    options: basePostOptions.value.map(item => ({ label: item.label, value: item.value, disabled: item.disabled })),
    props: { placeholder: t("system.base.user.placeholder.post"), clearable: true, filterable: true }
  },
  {
    prop: "phone",
    label: t("common.field.phone"),
    component: "input",
    props: { placeholder: t("common.validation.phone_required") }
  },
  {
    prop: "pwd",
    label: t("system.base.user.field.password"),
    component: "password",
    props: { placeholder: t("system.base.user.placeholder.password"), showPassword: true },
    visible: model => !model.id
  },
  {
    prop: "passwordStrength",
    label: t("system.base.user.field.password_strength"),
    component: "slot",
    slotName: "passwordStrength",
    visible: model => !model.id
  },
  { prop: "gender", label: t("system.base.user.field.gender"), component: "dict", props: { code: "base_user_gender" } },
  {
    prop: "status",
    label: t("common.field.status"),
    component: "radio-group",
    options: statusOptions.value,
    props: { disabled: isSuperEditUser.value }
  },
  {
    prop: "remark",
    label: t("common.field.remark"),
    component: "textarea",
    props: { placeholder: t("common.placeholder.remark") }
  }
]);

/** 用户表格列配置。 */
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55, selectable: row => !isProtectedManagementUser(row as BaseUser) },
  ...(isDefaultTenant.value
    ? ([
        {
          prop: "tenant_id",
          label: t("system.base.user.field.tenant_short"),
          minWidth: 140,
          showOverflowTooltip: true,
          search: { el: "select", key: "tenant_id", props: { filterable: true }, order: 1 },
          enum: requestTenantOptions
        }
      ] satisfies ColumnProps[])
    : []),
  { prop: "user_name", label: t("system.base.user.field.user_name"), minWidth: 140, search: { el: "input" } },
  { prop: "user_code", label: t("system.base.user.field.user_code"), minWidth: 140, search: { el: "input" } },
  { prop: "nick_name", label: t("system.base.user.field.nick_name_short"), minWidth: 100, search: { el: "input" } },
  { prop: "role_id", label: t("system.base.user.field.role"), minWidth: 140, enum: requestRoleOptions },
  {
    prop: "dept_id",
    label: t("system.base.user.field.department_short"),
    minWidth: 180,
    showOverflowTooltip: true,
    enum: requestDeptOptions
  },
  { prop: "post_id", label: t("system.base.user.field.post_short"), minWidth: 120, enum: requestPostOptions },
  { prop: "phone", label: t("common.field.phone"), minWidth: 130, search: { el: "input" } },
  {
    prop: "gender",
    label: t("system.base.user.field.gender"),
    minWidth: 90,
    align: "center",
    dictCode: "base_user_gender",
    search: { el: "select" }
  },
  {
    prop: "status",
    label: t("common.field.status"),
    minWidth: 100,
    search: { el: "select" },
    cellType: "status",
    statusProps: {
      activeValue: Status.STATUS_ENABLE,
      inactiveValue: Status.STATUS_DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      disabled: scope => isProtectedManagementUser(scope.row as BaseUser) || !BUTTONS.value["base:user:status"],
      beforeChange: scope => handleBeforeSetStatus(scope.row as BaseUser)
    }
  },
  { prop: "remark", label: t("common.field.remark"), minWidth: 160 },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.action"),
    width: 260,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("system.base.user.action.reset_password"),
        type: "primary",
        link: true,
        icon: RefreshLeft,
        hidden: scope => isProtectedManagementUser(scope.row as BaseUser) || !BUTTONS.value["base:user:pwd"],
        onClick: scope => handleResetPassword(scope.row as BaseUser)
      },
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        hidden: scope => isProtectedManagementUser(scope.row as BaseUser) || !BUTTONS.value["base:user:update"],
        params: scope => ({ userId: scope.row.id }),
        onClick: (scope, params) => handleOpenDialog((params?.userId as number | undefined) ?? (scope.row as BaseUser).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: scope => isProtectedManagementUser(scope.row as BaseUser) || !BUTTONS.value["base:user:delete"],
        onClick: scope => handleDelete(scope.row as BaseUser)
      }
    ]
  }
]);

/** 用户顶部按钮配置。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:user:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:user:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as BaseUser[])
  }
]);

/**
 * 递归转换部门树筛选数据，适配 TreeFilter 组件的 id/name 字段。
 */
function transformDeptFilterNodes(options: TreeOptionResponse_Option[] = []): DeptFilterNode[] {
  return options.map(option => ({
    id: String(option.value),
    name: option.label,
    children: transformDeptFilterNodes(option.children ?? [])
  }));
}

/**
 * 将部门树选项转换为完整路径标签，便于扁平列表关联字段按部门 ID 映射展示。
 */
function transformDeptOptionPaths(options: TreeOptionResponse_Option[] = [], parentPath = ""): TreeOptionResponse_Option[] {
  return options.map(option => {
    const label = option.label || "";
    const fullPath = [parentPath, label].filter(Boolean).join("/");
    return {
      ...option,
      label: fullPath,
      children: transformDeptOptionPaths(option.children ?? [], fullPath)
    };
  });
}

/**
 * 请求部门树筛选数据。
 */
async function requestDeptTreeFilter() {
  const response = await defBaseDeptService.OptionBaseDept({ tenant_id: selectedTenantId.value });
  return {
    data: transformDeptFilterNodes(response.list ?? [])
  };
}

/**
 * 切换部门树筛选时同步更新表格初始化参数。
 */
function changeTreeFilter(value: string) {
  deptFilterValue.value = value ?? "";
  initParam.dept_id = value ? Number(value) : undefined;
  if (proTable.value) {
    proTable.value.pageable.page_num = 1;
  }
}

/**
 * 请求用户分页列表，并统一处理分页参数。
 */
async function requestBaseUserTable(params: PageBaseUserRequest) {
  const tenantId = isDefaultTenant.value ? params.tenant_id : undefined;
  if (tenantId !== selectedTenantId.value) {
    selectedTenantId.value = tenantId;
    initParam.dept_id = undefined;
    deptFilterValue.value = "";
    await proTable.value?.refreshEnums();
  }
  const data = await defBaseUserService.PageBaseUser({
    ...buildPageRequest(params),
    tenant_id: tenantId,
    dept_id: initParam.dept_id
  });
  return { data: { list: data.base_users ?? [], total: data.total } };
}

/** 读取当前列表关联选项使用的租户。 */
function getOptionTenantId() {
  return isDefaultTenant.value ? selectedTenantId.value : undefined;
}

/** 请求角色关联选项。 */
async function requestRoleOptions() {
  const response = await defBaseRoleService.OptionBaseRole({ tenant_id: getOptionTenantId() });
  return { data: response.list ?? [] };
}

/** 请求部门关联选项。 */
async function requestDeptOptions() {
  const response = await defBaseDeptService.OptionBaseDept({ tenant_id: getOptionTenantId() });
  return { data: transformDeptOptionPaths(response.list ?? []) };
}

/** 请求岗位关联选项。 */
async function requestPostOptions() {
  const response = await defBasePostService.OptionBasePost({ tenant_id: getOptionTenantId() });
  return { data: response.list ?? [] };
}

/**
 * 刷新用户表格数据。
 */
function refreshTable() {
  proTable.value?.getTableList();
}

/**
 * 加载用户表单依赖的角色和部门选项。
 */
async function loadFormOptions() {
  // 默认租户必须先选择目标租户，避免角色和部门选项跨租户混用。
  if (isDefaultTenant.value && !formData.tenant_id) {
    baseRoleOptions.value = [];
    basePostOptions.value = [];
    basedDeptOptions.value = [];
    return;
  }
  const tenantId = isDefaultTenant.value ? formData.tenant_id : undefined;
  const [optionBaseRoleResponse, optionBaseDeptResponse, optionBasePostResponse] = await Promise.all([
    defBaseRoleService.OptionBaseRole({ tenant_id: tenantId }),
    defBaseDeptService.OptionBaseDept({ tenant_id: tenantId }),
    defBasePostService.OptionBasePost({ tenant_id: tenantId })
  ]);
  baseRoleOptions.value = optionBaseRoleResponse.list || [];
  basePostOptions.value = optionBasePostResponse.list || [];
  basedDeptOptions.value = optionBaseDeptResponse.list || [];
}

/**
 * 加载租户下拉选项。
 */
async function loadTenantOptions() {
  if (!isDefaultTenant.value || tenantOptions.value.length) return;
  const response = await defBaseTenantService.OptionBaseTenant({ keyword: "" });
  tenantOptions.value = response.list ?? [];
}

/**
 * 切换用户表单租户时，清空角色和部门并重新加载选项。
 */
async function handleFormTenantChange() {
  formData.role_id = undefined;
  formData.dept_id = undefined;
  formData.post_id = undefined;
  await loadFormOptions();
}

/**
 * 打开用户弹窗，并加载角色和部门下拉选项。
 */
async function handleOpenDialog(id?: number) {
  resetForm();
  await loadTenantOptions();
  dialog.editing = Boolean(id);
  dialog.visible = true;
  if (!id) {
    await loadFormOptions();
    return;
  }

  defBaseUserService.GetBaseUser({ id }).then(async data => {
    Object.assign(formData, data);
    await loadFormOptions();
  });
}

/**
 * 关闭用户弹窗并恢复默认表单值。
 */
function handleCloseDialog() {
  dialog.visible = false;
  resetForm();
}

/**
 * 重置用户表单，避免新增与编辑之间互相污染。
 */
function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  formData.id = 0;
  formData.tenant_id = undefined;
  formData.user_name = "";
  formData.user_code = "";
  formData.nick_name = "";
  formData.role_id = undefined;
  formData.dept_id = undefined;
  formData.post_id = undefined;
  formData.phone = "";
  formData.pwd = "";
  formData.gender = 3;
  formData.avatar = "";
  formData.status = Status.STATUS_ENABLE;
  formData.remark = "";
  baseRoleOptions.value = [];
  basePostOptions.value = [];
  basedDeptOptions.value = [];
}

/**
 * 打开重置密码弹窗，并回填当前操作用户。
 */
function handleResetPassword(row: BaseUser) {
  resetPwdFormDialogRef.value?.resetFields();
  resetPwdFormDialogRef.value?.clearValidate();
  resetPwdForm.id = row.id;
  resetPwdForm.pwd = "";
  resetPwdTargetName.value = row.nick_name || row.user_name || `ID:${row.id}`;
  resetPwdDialog.visible = true;
}

/**
 * 关闭重置密码弹窗并恢复默认表单值。
 */
function handleCloseResetPasswordDialog() {
  resetPwdDialog.visible = false;
  resetPwdFormDialogRef.value?.resetFields();
  resetPwdFormDialogRef.value?.clearValidate();
  resetPwdForm.id = 0;
  resetPwdForm.pwd = "";
  resetPwdTargetName.value = "";
}

/**
 * 确认重置用户密码，并复用统一密码强度校验。
 */
async function handleConfirmResetPassword() {
  resetPwdFormDialogRef.value?.validate()?.then(async valid => {
    if (!valid) return;

    const pwd = await encryptPassword(resetPwdForm.pwd, PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_RESET_BASE_USER_PASSWORD);
    defBaseUserService.ResetBaseUserPassword({ id: resetPwdForm.id, pwd }).then(() => {
      ElMessage.success(t("system.base.user.message.reset_password_success", { name: resetPwdTargetName.value }));
      handleCloseResetPasswordDialog();
    });
  });
}

/**
 * 提交用户表单，使用防抖避免重复提交。
 */
const handleSubmit = useDebounceFn(() => {
  formDialogRef.value?.validate()?.then(async valid => {
    if (!valid) return;

    const submitData = JSON.parse(JSON.stringify(formData)) as BaseUserFormState;
    const baseUser = {
      ...submitData,
      pwd: submitData.id ? undefined : await encryptPassword(submitData.pwd, PASSWORD_CRYPTO_SCENE.PASSWORD_CRYPTO_SCENE_CREATE_BASE_USER)
    } as BaseUserForm;
    const request = submitData.id
      ? defBaseUserService.UpdateBaseUser({ base_user: baseUser })
      : defBaseUserService.CreateBaseUser({ base_user: baseUser });
    request.then(() => {
      ElMessage.success(t(submitData.id ? "system.base.user.message.update_success" : "system.base.user.message.create_success"));
      handleCloseDialog();
      refreshTable();
    });
  });
}, 1000);

/**
 * 在用户状态切换前先完成确认与接口调用，避免首屏渲染触发误操作。
 */
async function handleBeforeSetStatus(row: BaseUser) {
  if (isProtectedManagementUser(row)) {
    ElMessage.warning(t("system.base.user.message.protected_account"));
    return false;
  }

  const nextStatus = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  const action = t(nextStatus === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled");
  const userName = row.nick_name || row.user_name || `ID:${row.id}`;
  try {
    await ElMessageBox.confirm(t("system.base.user.message.confirm_status", { action, name: userName }), t("common.title.notice"), {
      confirmButtonText: t("common.action.confirm"),
      cancelButtonText: t("common.action.cancel"),
      type: "warning"
    });
    await defBaseUserService.SetBaseUserStatus({ id: row.id, status: nextStatus });
    ElMessage.success(t("common.message.status_success", { action }));
    refreshTable();
    return true;
  } catch {
    return false;
  }
}

/**
 * 删除用户，兼容单条删除与批量删除。
 */
function handleDelete(selected?: number | string | Array<number | string> | BaseUser | BaseUser[]) {
  const userList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as BaseUser[])
    : selected && typeof selected === "object"
      ? [selected]
      : [];
  if (userList.some(isProtectedManagementUser)) {
    ElMessage.warning(t("system.base.user.message.protected_account"));
    return;
  }
  const userIds = normalizeSelectedIds(
    userList.length ? userList.map(item => item.id) : (selected as number | string | Array<number | string> | undefined)
  ).join(",");
  if (!userIds) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }

  const confirmMessage = userList.length
    ? userList.length === 1
      ? t("system.base.user.message.confirm_delete_single", {
          name: userList[0].nick_name || userList[0].user_name || `ID:${userList[0].id}`
        })
      : t("system.base.user.message.confirm_delete_batch", { count: userList.length })
    : t("system.base.user.message.confirm_delete_selected");

  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      defBaseUserService.DeleteBaseUser({ id: userIds }).then(() => {
        ElMessage.success(t("system.base.user.message.delete_success"));
        refreshTable();
      });
    },
    () => {
      ElMessage.info(t("system.base.user.message.delete_canceled"));
    }
  );
}

/**
 * 校验密码强度，新增用户和重置密码统一要求达到最高强度。
 *
 * @param _rule 表单规则对象
 * @param value 当前密码值
 * @param callback 校验回调
 */
function validatePasswordField(_rule: unknown, value: string, callback: (error?: Error) => void) {
  if (!value) {
    callback();
    return;
  }
  const result = validatePasswordStrengthValue(value);
  if (!result.valid) {
    callback(new Error(t("core.password.error")));
    return;
  }
  callback();
}

/**
 * 根据后端管理保护标记判断用户是否禁止通过用户管理操作。
 */
function isProtectedManagementUser(row?: BaseUser) {
  return Boolean(row?.is_protected);
}
</script>
