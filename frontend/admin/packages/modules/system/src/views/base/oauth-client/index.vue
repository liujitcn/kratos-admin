<!-- 开放授权客户端管理 -->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestOauthClientTable"
    />

    <FormDialog
      v-model="dialog.visible"
      ref="formDialogRef"
      :title="t(dialog.titleKey)"
      width="min(1200px, calc(100vw - 32px))"
      label-width="150px"
      :model="formData"
      :fields="formFields"
      :rules="rules"
      @confirm="handleSubmit"
      @close="handleCloseDialog"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { copyText } from "@liujitcn/kratos-admin-core/security";
import { defOauthClientService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/oauth_client";
import type { BaseApi } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_api";
import {
  OauthClientCryptoType,
  type OauthClient,
  type OauthClientForm,
  type PageOauthClientRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/oauth_client";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";

/** 开放授权客户端表单状态，新增时租户保持未选择。 */
type OauthClientFormState = Omit<OauthClientForm, "tenant_id"> & {
  /** 绑定租户ID。 */
  tenant_id?: number;
};

defineOptions({
  name: "OauthClient",
  inheritAttrs: false
});

const { BUTTONS } = useAuthButtons();
const { isDefaultTenant, tenantColumns, tenantFormField, loadTenantOptions } = useTenantScope();
const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const apiOptions = ref<BaseApi[]>([]);

const dialog = reactive({
  titleKey: "common.action.create",
  visible: false
});

const formData = reactive<OauthClientFormState>({
  id: 0,
  tenant_id: undefined,
  client_name: "",
  crypto_type: OauthClientCryptoType.OAUTH_CLIENT_CRYPTO_TYPE_SM4,
  ip_whitelist: "",
  api: [],
  status: Status.STATUS_ENABLE
});

onMounted(() => {
  void loadTenantOptions();
});

const cryptoOptions = computed<ProFormOption[]>(() => [
  { label: "SM4-GCM", value: OauthClientCryptoType.OAUTH_CLIENT_CRYPTO_TYPE_SM4 },
  { label: "AES-GCM", value: OauthClientCryptoType.OAUTH_CLIENT_CRYPTO_TYPE_AES }
]);

const statusOptions = computed<ProFormOption[]>(() => [
  { label: t("common.status.enabled"), value: Status.STATUS_ENABLE },
  { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }
]);

const transferOptions = computed<ProFormOption[]>(() =>
  apiOptions.value.map(item => ({
    value: item.operation,
    label: item.operation
  }))
);

const rules = computed(() => ({
  tenant_id: [
    {
      required: isDefaultTenant.value,
      message: t("common.validation.required_select", { field: t("common.field.tenant") }),
      trigger: "change"
    }
  ],
  client_name: [
    {
      required: true,
      message: t("common.validation.required_input", { field: t("system.base.oauth_client.field.client_name") }),
      trigger: "blur"
    },
    {
      max: 100,
      message: t("common.validation.max_length", { field: t("system.base.oauth_client.field.client_name"), max: 100 }),
      trigger: "blur"
    }
  ],
  crypto_type: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("system.base.oauth_client.field.crypto_type") }),
      trigger: "change"
    }
  ],
  status: [
    { required: true, message: t("common.validation.required_select", { field: t("common.field.status") }), trigger: "change" }
  ]
}));

const formFields = computed<ProFormField[]>(() => [
  tenantFormField({ label: t("common.field.tenant"), disabledOnEdit: true, props: { placeholder: t("common.placeholder.select") } }),
  {
    prop: "client_name",
    label: t("system.base.oauth_client.field.client_name"),
    component: "input"
  },
  {
    prop: "crypto_type",
    label: t("system.base.oauth_client.field.crypto_type"),
    component: "select",
    options: cryptoOptions.value
  },
  {
    prop: "ip_whitelist",
    label: t("system.base.oauth_client.field.ip_whitelist"),
    component: "textarea",
    props: { rows: 2 },
    colSpan: 24
  },
  {
    prop: "api",
    label: t("system.base.oauth_client.field.api"),
    component: "transfer",
    options: transferOptions.value,
    props: {
      filterable: true,
      titles: [t("system.base.oauth_client.value.available_api"), t("system.base.oauth_client.value.selected_api")],
      style: { width: "100%" }
    },
    colSpan: 24
  },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value }
]);

const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  ...tenantColumns({ label: t("common.field.tenant"), minWidth: 130, order: 1 }),
  { prop: "client_id", label: t("system.base.oauth_client.field.client_id"), minWidth: 220 },
  { prop: "client_name", label: t("system.base.oauth_client.field.client_name"), minWidth: 160, search: { el: "input" } },
  {
    prop: "crypto_type",
    label: t("system.base.oauth_client.field.crypto_type"),
    width: 110,
    render: scope => formatCrypto((scope.row as OauthClient).crypto_type)
  },
  {
    prop: "ip_whitelist",
    label: t("system.base.oauth_client.field.ip_whitelist"),
    minWidth: 180,
    render: scope => (scope.row as OauthClient).ip_whitelist || t("common.value.none")
  },
  {
    prop: "api",
    label: t("system.base.oauth_client.field.api"),
    width: 90,
    render: scope => String((scope.row as OauthClient).api?.length ?? 0)
  },
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
      disabled: () => !BUTTONS.value["base:oauth-client:status"],
      beforeChange: scope => handleSetStatus(scope.row as OauthClient)
    }
  },
  {
    prop: "operation",
    label: t("common.field.operation"),
    cellType: "actions",
    actions: [
      {
        label: t("system.base.oauth_client.action.copy_credentials"),
        type: "primary",
        link: true,
        icon: CopyDocument,
        hidden: () => !BUTTONS.value["base:oauth-client:credentials"],
        onClick: scope => handleCopyCredentials(scope.row as OauthClient)
      },
      {
        label: t("common.action.edit"),
        link: true,
        icon: EditPen,
        hidden: () => !BUTTONS.value["base:oauth-client:update"],
        onClick: scope => handleOpenDialog((scope.row as OauthClient).id)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:oauth-client:delete"],
        onClick: scope => handleDelete(scope.row as OauthClient)
      }
    ]
  }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    hidden: () => !BUTTONS.value["base:oauth-client:create"],
    onClick: () => handleOpenDialog()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    hidden: () => !BUTTONS.value["base:oauth-client:delete"],
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as OauthClient[])
  }
]);

async function requestOauthClientTable(params: PageOauthClientRequest) {
  const data = await defOauthClientService.PageOauthClient({
    ...buildPageRequest(params),
    tenant_id: isDefaultTenant.value ? params.tenant_id : undefined
  });
  return { data: { list: data.oauth_clients ?? [], total: data.total } };
}

async function loadApiOptions() {
  const data = await defOauthClientService.OptionOauthClientApi({});
  apiOptions.value = data.base_apis ?? [];
}

async function handleOpenDialog(id?: number) {
  resetForm();
  dialog.titleKey = id ? "common.action.edit" : "common.action.create";
  await formDialogRef.value?.open({
    load: async () => ({
      data: id ? await defOauthClientService.GetOauthClient({ id }) : undefined,
      resources: await Promise.all([loadTenantOptions(), loadApiOptions()])
    }),
    commit: ({ data }) => {
      if (data) Object.assign(formData, data);
    }
  });
}

function resetForm() {
  formDialogRef.value?.resetFields();
  formDialogRef.value?.clearValidate();
  Object.assign(formData, {
    id: 0,
    tenant_id: undefined,
    client_name: "",
    crypto_type: OauthClientCryptoType.OAUTH_CLIENT_CRYPTO_TYPE_SM4,
    ip_whitelist: "",
    api: [],
    status: Status.STATUS_ENABLE
  });
}

function handleCloseDialog() {
  formDialogRef.value?.close();
  resetForm();
}

function handleSubmit() {
  formDialogRef.value?.validate()?.then(valid => {
    if (!valid) return;
    const payload = JSON.parse(JSON.stringify(formData)) as OauthClientForm;
    const request = payload.id
      ? defOauthClientService.UpdateOauthClient({ oauth_client: payload })
      : defOauthClientService.CreateOauthClient({ oauth_client: payload });
    request.then(() => {
      ElMessage.success(
        t(payload.id ? "common.message.update_success" : "common.message.create_success", {
          resource: t("system.base.oauth_client.resource")
        })
      );
      handleCloseDialog();
      proTable.value?.getTableList();
    });
  });
}

async function handleSetStatus(row: OauthClient) {
  const nextStatus = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE;
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action: nextStatus === Status.STATUS_ENABLE ? t("common.action.enable") : t("common.action.disable"),
        resource: t("system.base.oauth_client.resource"),
        field: t("system.base.oauth_client.field.client_name"),
        value: row.client_name
      }),
      t("common.title.warning"),
      { type: "warning", confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel") }
    );
    await defOauthClientService.SetOauthClientStatus({ id: row.id, status: nextStatus });
    ElMessage.success(t("common.message.update_success", { resource: t("system.base.oauth_client.resource") }));
    proTable.value?.getTableList();
    return true;
  } catch {
    return false;
  }
}

function handleDelete(selected?: number | string | Array<number | string> | OauthClient | OauthClient[]) {
  const rowList = Array.isArray(selected)
    ? (selected.filter(item => typeof item === "object") as OauthClient[])
    : selected && typeof selected === "object"
      ? [selected as OauthClient]
      : [];
  const ids = rowList.length
    ? rowList.map(item => item.id)
    : normalizeSelectedIds(selected as number | string | Array<number | string>);
  const value = ids.filter(Boolean).join(",");
  if (!value) {
    ElMessage.warning(t("common.message.select_delete_item"));
    return;
  }
  const resource = rowList.length
    ? `${t("system.base.oauth_client.resource")}：${rowList
        .map(item => item.client_name)
        .filter(Boolean)
        .join("、")}`
    : t("system.base.oauth_client.resource");
  ElMessageBox.confirm(t("common.dialog.delete_selected", { resource }), t("common.title.warning"), {
    type: "warning",
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel")
  })
    .then(() => {
      return defOauthClientService.DeleteOauthClient({ id: value });
    })
    .then(() => {
      ElMessage.success(t("common.message.delete_success", { resource: t("system.base.oauth_client.resource") }));
      proTable.value?.getTableList();
    });
}

async function handleCopyCredentials(row: OauthClient) {
  await ElMessageBox.confirm(
    t("system.base.oauth_client.confirm.rotate_credentials"),
    t("common.title.warning"),
    { type: "warning" }
  );
  const credentials = await defOauthClientService.RotateOauthClientCredentials({ id: row.id });
  const content = [
    `${t("system.base.oauth_client.field.client_id")}：${credentials.client_id}`,
    `${t("system.base.oauth_client.field.client_secret")}：${credentials.client_secret}`,
    `${t("system.base.oauth_client.field.crypto_type")}：${formatCrypto(credentials.crypto_type)}`,
    `${t("system.base.oauth_client.field.crypto_key")}：${credentials.crypto_key}`
  ].join("\n");
  await copyText(content);
  ElMessage.success(t("core.clipboard.success"));
}

function formatCrypto(value: OauthClientCryptoType) {
  return cryptoOptions.value.find(item => item.value === value)?.label ?? String(value);
}
</script>
