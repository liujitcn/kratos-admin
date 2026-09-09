<template>
  <el-card class="session-card" shadow="never">
    <template #header>
      <div class="session-header">
        <div>
          <h3>{{ t("system.profile.session.title") }}</h3>
          <p>{{ t("system.profile.session.description") }}</p>
        </div>
        <el-button :loading="loading" @click="loadSession">
          <el-icon><Refresh /></el-icon>
          {{ t("system.profile.session.action.refresh") }}
        </el-button>
      </div>
    </template>

    <ProTable ref="proTable" row-key="session_id" :columns="columns" :request-api="requestSessionTable" :pagination="false" />
    <div class="session-actions">
      <el-button type="danger" plain :disabled="!hasSessions" @click="revokeAll">
        <el-icon><SwitchButton /></el-icon>
        {{ t("system.profile.session.action.revoke_all") }}
      </el-button>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { useRouter } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import { LOGIN_URL } from "@liujitcn/kratos-admin-core/config";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import { defBaseSessionService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_session";
import type { BaseSession } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_session";

const loading = ref(false);
const proTable = ref<ProTableInstance>();
const hasSessions = ref(false);
const columns = computed<ColumnProps[]>(() => [
  {
    prop: "current",
    label: t("system.profile.session.field.current"),
    cellType: "status",
    minWidth: 130,
    statusProps: {
      activeValue: true,
      inactiveValue: false,
      activeText: t("system.profile.session.status.current"),
      inactiveText: t("system.profile.session.status.other")
    }
  },
  { prop: "client_ip", label: t("system.profile.session.field.client_ip"), minWidth: 140 },
  { prop: "device", label: t("system.profile.session.field.device"), minWidth: 200 },
  { prop: "issued_at", label: t("system.profile.session.field.issued_at"), minWidth: 180 },
  {
    prop: "expires_in",
    label: t("system.profile.session.field.expires_in"),
    minWidth: 140,
    render: scope => formatExpires((scope.row as BaseSession).expires_in)
  }
]);

/** 查询本人全部有效会话。 */
async function requestSessionTable() {
  const response = await defBaseSessionService.ListCurrentBaseSessions({});
  const sessions = response.sessions ?? [];
  hasSessions.value = sessions.length > 0;
  return { data: sessions };
}
const router = useRouter();
const userStore = useUserStore();

/** 刷新本人会话列表。 */
async function loadSession() {
  loading.value = true;
  try {
    await proTable.value?.getTableList();
  } finally {
    loading.value = false;
  }
}

/** 撤销当前用户全部会话并返回登录页。 */
async function revokeAll() {
  await ElMessageBox.confirm(t("system.profile.session.confirm.revoke_all"), t("system.profile.session.confirm.title"), {
    type: "warning"
  });
  await defBaseSessionService.RevokeAllBaseSessions({});
  userStore.clearAuthData();
  ElMessage.success(t("system.profile.session.message.revoked"));
  hasSessions.value = false;
  await router.replace(LOGIN_URL);
}

/** 将秒数格式化为可读的剩余有效期。 */
function formatExpires(seconds: number) {
  if (!seconds || seconds < 60) return t("system.profile.session.value.less_than_minute");
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  if (hours > 0) return t("system.profile.session.value.hours_minutes", { hours, minutes });
  return t("system.profile.session.value.minutes", { minutes });
}
</script>

<style scoped lang="scss">
.session-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--admin-page-radius);
}
:deep(.session-card .el-card__header) {
  padding: 18px 20px;
  border-bottom: 1px solid var(--el-border-color-light);
}
:deep(.session-card .el-card__body) {
  padding: 20px;
}
.session-header,
.session-status,
.session-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.session-header h3 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 18px;
}
.session-header p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.session-actions {
  justify-content: flex-end;
  margin-top: 20px;
}
@media screen and (width <= 640px) {
  .session-header {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
