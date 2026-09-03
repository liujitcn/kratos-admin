<template>
  <div class="profile-password">
    <el-card class="password-card" shadow="never">
      <template #header>
        <div class="panel-header">
          <div>
            <h3>{{ t("system.profile.password.title") }}</h3>
            <p>{{ t("system.profile.password.description") }}</p>
          </div>
          <el-tag type="warning" effect="plain">{{ t("system.profile.password.value.strong_recommended") }}</el-tag>
        </div>
      </template>
      <ChangePasswordForm />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { t } from "@liujitcn/kratos-admin-core";
import ChangePasswordForm from "../../../components/ChangePasswordForm.vue";
</script>

<style scoped lang="scss">
.password-card {
  border: 1px solid #ebeef5;
  border-radius: var(--admin-page-radius);
}
:deep(.password-card .el-card__header) {
  padding: 18px 20px;
  border-bottom: 1px solid #f0f2f5;
}
:deep(.password-card .el-card__body) {
  padding: 20px;
}
.panel-header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
}
.panel-header h3 {
  margin: 0;
  font-size: 18px;
  color: #303133;
}
.panel-header p {
  margin: 6px 0 0;
  font-size: 13px;
  color: #909399;
}

@media screen and (width <= 640px) {
  .panel-header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
