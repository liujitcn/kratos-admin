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

    <el-skeleton v-if="loading && !session" :rows="5" animated />
    <el-empty v-else-if="!session" :description="t('system.profile.session.empty')" />
    <div v-else class="session-details">
      <div class="session-status">
        <el-tag type="success" effect="plain">{{ t("system.profile.session.status.active") }}</el-tag>
        <span>{{ session.user_name }} · {{ session.tenant_code }}</span>
      </div>
      <dl class="session-grid">
        <div>
          <dt>{{ t("system.profile.session.field.client_ip") }}</dt>
          <dd>{{ session.client_ip || "-" }}</dd>
        </div>
        <div>
          <dt>{{ t("system.profile.session.field.device") }}</dt>
          <dd>{{ session.device || "-" }}</dd>
        </div>
        <div>
          <dt>{{ t("system.profile.session.field.issued_at") }}</dt>
          <dd>{{ session.issued_at || "-" }}</dd>
        </div>
        <div>
          <dt>{{ t("system.profile.session.field.expires_in") }}</dt>
          <dd>{{ formatExpires(session.expires_in) }}</dd>
        </div>
      </dl>
      <div class="session-actions">
        <el-button type="danger" plain @click="revokeAll">
          <el-icon><SwitchButton /></el-icon>
          {{ t("system.profile.session.action.revoke_all") }}
        </el-button>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import { LOGIN_URL } from "@liujitcn/kratos-admin-core/config";
import { useUserStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import { defBaseSessionService } from "@liujitcn/kratos-admin-system/api/system/base_session";
import type { BaseSession } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_session";

const loading = ref(false);
const session = ref<BaseSession>();
const router = useRouter();
const userStore = useUserStore();

/** 拉取当前会话信息。 */
async function loadSession() {
  loading.value = true;
  try {
    session.value = await defBaseSessionService.GetCurrentBaseSession({});
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
  session.value = undefined;
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

onMounted(loadSession);
</script>

<style scoped lang="scss">
.session-card {
  border: 1px solid var(--el-border-color-light);
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
.session-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin: 24px 0;
}
.session-grid div {
  padding: 14px;
  background: var(--el-fill-color-light);
  border-radius: 6px;
}
.session-grid dt {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.session-grid dd {
  margin: 6px 0 0;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
}
.session-actions {
  justify-content: flex-end;
}
@media screen and (width <= 640px) {
  .session-header {
    align-items: flex-start;
    flex-direction: column;
  }
  .session-grid {
    grid-template-columns: 1fr;
  }
}
</style>
