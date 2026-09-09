<template>
  <el-card class="login-log-card" shadow="never">
    <template #header>
      <div class="login-log-header">
        <div>
          <h3>{{ t("system.profile.login_log.title") }}</h3>
          <p>{{ t("system.profile.login_log.description") }}</p>
        </div>
        <el-button :loading="loading" @click="refresh">
          <el-icon><Refresh /></el-icon>
          {{ t("system.profile.login_log.action.refresh") }}
        </el-button>
      </div>
    </template>
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestTable" />
  </el-card>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseLoginLogService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_login_log";
import { BaseLogResult } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_log";
import { BaseLoginLogType } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_login_log";
import type { BaseLoginLog } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_login_log";

defineOptions({ name: "ProfileLoginLog", inheritAttrs: false });

const proTable = ref<ProTableInstance>();
const loading = ref(false);
const loginTypeOptions = computed<Array<[number, string]>>(() => [
  [BaseLoginLogType.BASE_LOGIN_LOG_TYPE_PASSWORD, t("system.base.log.login_type.password")],
  [BaseLoginLogType.BASE_LOGIN_LOG_TYPE_OAUTH, t("system.base.log.login_type.oauth")],
  [BaseLoginLogType.BASE_LOGIN_LOG_TYPE_MFA, t("system.base.log.login_type.mfa")],
  [BaseLoginLogType.BASE_LOGIN_LOG_TYPE_TOKEN_REFRESH, t("system.base.log.login_type.token_refresh")],
  [BaseLoginLogType.BASE_LOGIN_LOG_TYPE_LOGOUT, t("system.base.log.login_type.logout")]
]);
const resultOptions = computed<Array<[number, string]>>(() => [
  [BaseLogResult.BASE_LOG_RESULT_SUCCESS, t("system.base.log.result.success")],
  [BaseLogResult.BASE_LOG_RESULT_FAILURE, t("system.base.log.result.failure")],
  [BaseLogResult.BASE_LOG_RESULT_ERROR, t("system.base.log.result.error")]
]);
const columns = computed<ColumnProps[]>(() => [
  { prop: "occurred_at", label: t("system.profile.login_log.field.occurred_at"), minWidth: 180 },
  {
    prop: "login_type",
    label: t("system.profile.login_log.field.login_type"),
    minWidth: 130,
    render: scope => enumLabel(loginTypeOptions.value, (scope.row as BaseLoginLog).login_type)
  },
  {
    prop: "result",
    label: t("system.profile.login_log.field.result"),
    minWidth: 100,
    render: scope => enumLabel(resultOptions.value, (scope.row as BaseLoginLog).result)
  },
  { prop: "client_ip", label: t("system.profile.login_log.field.client_ip"), minWidth: 140 },
  { prop: "device_id", label: t("system.profile.login_log.field.device_id"), minWidth: 160 },
  { prop: "reason", label: t("system.profile.login_log.field.reason"), minWidth: 180 }
]);

/** 查询当前用户的登录记录。 */
async function requestTable(params: Record<string, unknown>) {
  const response = await defBaseLoginLogService.PageCurrentUserLoginLog(
    buildPageRequest({ page_num: Number(params.page_num ?? 1), page_size: Number(params.page_size ?? 10) })
  );
  return { data: { list: response.base_login_logs ?? [], total: response.total } };
}

/** 刷新当前用户的登录记录。 */
async function refresh() {
  loading.value = true;
  try {
    await proTable.value?.getTableList();
  } finally {
    loading.value = false;
  }
}

/** 输出枚举显示文本。 */
function enumLabel(options: Array<[number, string]>, value: number) {
  return options.find(option => option[0] === value)?.[1] ?? String(value);
}
</script>

<style scoped lang="scss">
.login-log-card {
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--admin-page-radius);
}
:deep(.login-log-card .el-card__header) {
  padding: 18px 20px;
  border-bottom: 1px solid var(--el-border-color-light);
}
:deep(.login-log-card .el-card__body) {
  padding: 20px;
}
.login-log-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.login-log-header h3 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 18px;
}
.login-log-header p {
  margin: 6px 0 0;
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
@media screen and (width <= 640px) {
  .login-log-header {
    align-items: flex-start;
    flex-direction: column;
  }
}
</style>
