<template>
  <ProDialog
    :model-value="passwordChangeState.required"
    :title="t('system.profile.password.title')"
    width="min(560px, calc(100vw - 32px))"
    top="8vh"
    class="forced-password-dialog"
    append-to-body
    :show-footer="false"
    :show-close="false"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
  >
    <el-alert
      :title="passwordChangeState.message || t('system.profile.password.description')"
      type="warning"
      :closable="false"
      show-icon
      class="forced-password-alert"
    />
    <ChangePasswordForm v-if="passwordChangeState.required" />
  </ProDialog>
</template>

<script setup lang="ts">
import { t } from "@liujitcn/kratos-admin-core";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import { passwordChangeState } from "@liujitcn/kratos-admin-core/request";
import ChangePasswordForm from "./ChangePasswordForm.vue";
</script>

<style scoped>
.forced-password-alert {
  margin-bottom: 16px;
}

:deep(.forced-password-dialog .change-password-form__footer) {
  margin-top: 12px;
}
</style>
