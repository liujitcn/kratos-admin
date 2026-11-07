<template>
  <main class="agreement-page">
    <el-card v-loading="loading" class="agreement-card">
      <template #header>
        <div class="agreement-header">
          <h1>{{ t(protocol === "service" ? "core.login.admin_agreement_service" : "core.login.admin_agreement_privacy") }}</h1>
          <el-button @click="router.replace('/login')">{{ t("core.login.admin_agreement_back") }}</el-button>
        </div>
      </template>
      <AdminAgreementContent :protocol="protocol" />
    </el-card>
  </main>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { useConfigStore } from "@/stores/modules/config";
import { useLocaleStore } from "@/locales";
import AdminAgreementContent from "./components/AdminAgreementContent.vue";
import type { AdminAgreementKind } from "./agreement";

/** 无需登录即可阅读的管理端协议页面属性。 */
interface AdminAgreementPageProps {
  protocol: AdminAgreementKind;
}

defineProps<AdminAgreementPageProps>();
const router = useRouter();
const configStore = useConfigStore();
const { t } = useLocaleStore();
const loading = ref(true);

onMounted(async () => {
  try {
    await configStore.loadDisplayConfig();
  } finally {
    loading.value = false;
  }
});
</script>

<style scoped lang="scss">
.agreement-page {
  box-sizing: border-box;
  min-height: 100dvh;
  padding: 24px;
  background: var(--el-bg-color-page);
}

.agreement-card {
  max-width: 960px;
  margin: 0 auto;
}

.agreement-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;

  h1 {
    margin: 0;
    font-size: 22px;
  }
}
</style>
