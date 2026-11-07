<template>
  <div v-if="content && version">
    <p class="agreement-version">{{ t("core.login.admin_agreement_version", { version }) }}</p>
    <RichTextPreview :model-value="content" />
  </div>
  <el-empty v-else :description="t('core.login.admin_agreement_unavailable')" />
</template>

<script setup lang="ts">
import { computed } from "vue";
import RichTextPreview from "@/components/RichTextPreview/index.vue";
import { useConfigStore } from "@/stores/modules/config";
import { useLocaleStore } from "@/locales";
import type { AdminAgreementKind } from "../agreement";

/** 管理端协议正文属性。 */
interface AdminAgreementContentProps {
  protocol: AdminAgreementKind;
}

const props = defineProps<AdminAgreementContentProps>();
const configStore = useConfigStore();
const { t } = useLocaleStore();
const content = computed(() => configStore.agreements[props.protocol]);
const version = computed(() =>
  props.protocol === "service" ? configStore.agreements.serviceVersion : configStore.agreements.privacyVersion
);
</script>

<style scoped lang="scss">
.agreement-version {
  margin: 0 0 16px;
  color: var(--el-text-color-secondary);
}
</style>
