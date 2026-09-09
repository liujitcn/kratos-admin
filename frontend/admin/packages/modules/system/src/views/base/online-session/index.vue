<template>
  <div class="online-session-page">
    <el-card class="admin-page-card">
      <ProTable
        :key="isDefaultTenant ? 'default-tenant' : 'current-tenant'"
        ref="proTable"
        row-key="session_id"
        :columns="columns"
        :request-api="requestTable"
      />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import type { ColumnProps, EnumProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { DEFAULT_TENANT_CODE } from "@liujitcn/kratos-admin-core/tenant";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseTenantService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_tenant";
import { defBaseSessionService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_session";
import type { BaseSession, PageOnlineBaseSessionsRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_session";

defineOptions({ name: "BaseOnlineSession", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const userStore = useUserStore();
const isDefaultTenant = computed(() => userStore.userInfo.tenant_code === DEFAULT_TENANT_CODE);

const columns = computed<ColumnProps[]>(() => [
  {
    prop: "tenant_code",
    label: t("common.field.tenant"),
    minWidth: 130,
    search: isDefaultTenant.value ? { el: "select", props: { filterable: true }, order: 1 } : undefined,
    enum: requestSessionTenantOptions
  },
  {
    prop: "user_name",
    label: t("system.base.online_session.field.user_name"),
    minWidth: 140,
    search: { el: "input", key: "keyword", order: 2 }
  },
  { prop: "client_ip", label: t("system.base.online_session.field.client_ip"), minWidth: 140 },
  { prop: "device", label: t("system.base.online_session.field.device"), minWidth: 180 },
  { prop: "user_agent", label: t("system.base.online_session.field.user_agent"), minWidth: 220 },
  { prop: "issued_at", label: t("system.base.online_session.field.issued_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 100,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("system.base.online_session.action.revoke"),
        type: "danger",
        link: true,
        icon: SwitchButton,
        hidden: () => !BUTTONS.value["base:online-session:revoke"],
        onClick: scope => revokeSession(scope.row as BaseSession)
      }
    ]
  }
]);

/** 将租户名称映射到会话使用的租户编码，供下拉筛选与表格展示复用。 */
async function requestSessionTenantOptions() {
  if (!isDefaultTenant.value) {
    return { data: [{ value: userStore.userInfo.tenant_code, label: userStore.userInfo.tenant_name }] };
  }
  const options: EnumProps[] = [];
  const pageSize = 100;
  for (let pageNum = 1; ; pageNum++) {
    const response = await defBaseTenantService.PageBaseTenant({ code: "", name: "", page_num: pageNum, page_size: pageSize });
    const tenants = response.base_tenants ?? [];
    options.push(...tenants.map(tenant => ({ value: tenant.code, label: tenant.name })));
    if (options.length >= response.total || tenants.length === 0) break;
  }
  return { data: options };
}

/** 查询在线用户会话列表。 */
async function requestTable(params: Record<string, unknown>) {
  const response = await defBaseSessionService.PageOnlineBaseSessions(
    buildPageRequest<PageOnlineBaseSessionsRequest>({
      ...(params as unknown as PageOnlineBaseSessionsRequest),
      tenant_code: isDefaultTenant.value ? String(params.tenant_code ?? "") : ""
    })
  );
  return { data: { list: response.sessions ?? [], total: response.total } };
}

/** 下线指定在线用户会话。 */
async function revokeSession(row: BaseSession) {
  await ElMessageBox.confirm(
    t("system.base.online_session.confirm.revoke", { user_name: row.user_name }),
    t("common.title.warning"),
    { type: "warning" }
  );
  await defBaseSessionService.RevokeBaseSession({ session_id: row.session_id });
  ElMessage.success(t("system.base.online_session.message.revoked"));
  proTable.value?.getTableList();
}
</script>

<style scoped lang="scss">
.online-session-page {
  min-width: 0;
}
</style>
