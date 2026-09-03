<template>
  <el-card class="base-card" shadow="never">
    <template #header>
      <div class="panel-header">
        <div>
          <h3>{{ t("system.profile.account.title") }}</h3>
          <p>{{ t("system.profile.account.description") }}</p>
        </div>
        <el-button type="primary" plain @click="openAccountDialog">{{ t("system.profile.account.action.edit") }}</el-button>
      </div>
    </template>

    <div class="base-layout">
      <div class="avatar-panel">
        <div class="avatar-shell">
          <el-avatar :src="avatarSrc" :size="116" @error="handleAvatarError" />
          <el-button class="avatar-trigger" circle type="primary" @click="triggerFileUpload">
            <el-icon><Camera /></el-icon>
          </el-button>
          <input ref="fileInputRef" type="file" class="hidden-input" accept="image/*" @change="handleFileChange" />
        </div>
        <div class="avatar-copy">
          <strong>{{ t("system.profile.account.field.avatar") }}</strong>
          <p>{{ t("system.profile.account.message.avatar_hint") }}</p>
        </div>
      </div>

      <div class="detail-grid">
        <div class="detail-item">
          <span class="detail-label">{{ t("system.profile.account.field.user_name") }}</span>
          <span class="detail-value">{{ profile.user_name || "--" }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ t("system.profile.account.field.nick_name") }}</span>
          <span class="detail-value">{{ profile.nick_name || "--" }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ t("system.profile.account.field.gender") }}</span>
          <span class="detail-value">{{ genderText }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ t("system.profile.account.field.phone") }}</span>
          <span class="detail-value">{{ profile.phone || t("system.profile.value.unbound") }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ t("system.profile.account.field.email") }}</span>
          <span class="detail-value">{{ profile.email || t("system.profile.value.unbound") }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ t("system.profile.account.field.id_type") }}</span>
          <span class="detail-value">{{ idTypeText }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ t("system.profile.account.field.id_code") }}</span>
          <span class="detail-value">{{ profile.id_code || t("system.profile.value.unbound") }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ t("common.field.role") }}</span>
          <span class="detail-value">{{ profile.role_name || "--" }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">{{ t("common.field.department") }}</span>
          <span class="detail-value">{{ profile.dept_name || "--" }}</span>
        </div>
        <div class="detail-item detail-item--wide">
          <span class="detail-label">{{ t("common.field.created_at") }}</span>
          <span class="detail-value">{{ profile.created_at || "--" }}</span>
        </div>
      </div>
    </div>

    <ProDialog
      v-model="accountDialogVisible"
      :title="t('system.profile.account.action.edit')"
      :width="520"
      @closed="handleDialogClosed"
    >
      <ProForm ref="accountFormRef" :model="accountForm" :fields="accountFormFields" :rules="accountFormRules" />
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="accountDialogVisible = false">{{ t("common.action.cancel") }}</el-button>
          <el-button type="primary" :loading="submitLoading" @click="handleSubmitAccount">
            {{ t("common.action.save") }}
          </el-button>
        </div>
      </template>
    </ProDialog>
  </el-card>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import { defProfileAuthService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/auth";
import type { UserProfileForm } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/auth";
import { BaseUserIDType } from "@liujitcn/kratos-admin-system/rpc/system/common/v1/common";
import { defFileService } from "@liujitcn/kratos-admin-core/api/base/v1/file";
import defaultAvatar from "@liujitcn/kratos-admin-core/assets/images/avatar.png";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import ProForm from "@liujitcn/kratos-admin-core/components/ProForm/index.vue";
import type { ProFormField, ProFormInstance } from "@liujitcn/kratos-admin-core/components/ProForm/interface";

/** 个人中心基础资料组件属性。 */
interface ProfileBaseProps {
  /** 当前用户资料。 */
  profile: UserProfileForm;
}

const props = defineProps<ProfileBaseProps>();

const emit = defineEmits<{
  refreshed: [];
}>();

const fileInputRef = ref<HTMLInputElement | null>(null);
const accountFormRef = ref<ProFormInstance>();
const accountDialogVisible = ref(false);
const submitLoading = ref(false);
const avatarSrc = ref(defaultAvatar);
const accountForm = reactive<Pick<UserProfileForm, "nick_name" | "gender" | "email" | "id_type" | "id_code">>({
  nick_name: "",
  gender: 3,
  email: "",
  id_type: BaseUserIDType.BASE_USER_ID_TYPE_UNSPECIFIED,
  id_code: ""
});

const accountFormFields = computed<ProFormField[]>(() => [
  {
    prop: "nick_name",
    label: t("system.profile.account.field.nick_name"),
    component: "input",
    props: { placeholder: t("system.profile.account.placeholder.nick_name") }
  },
  {
    prop: "gender",
    label: t("system.profile.account.field.gender"),
    component: "dict",
    props: { code: "base_user_gender" }
  },
  {
    prop: "email",
    label: t("system.profile.account.field.email"),
    component: "input",
    props: { placeholder: t("system.base.user.placeholder.email") }
  },
  {
    prop: "id_type",
    label: t("system.profile.account.field.id_type"),
    component: "select",
    options: idTypeOptions.value,
    props: { placeholder: t("common.placeholder.select") }
  },
  {
    prop: "id_code",
    label: t("system.profile.account.field.id_code"),
    component: "input",
    props: { placeholder: t("system.base.user.placeholder.id_code") }
  }
]);

const idTypeOptions = computed(() => [
  { label: t("system.base.user.id_type.unspecified"), value: BaseUserIDType.BASE_USER_ID_TYPE_UNSPECIFIED },
  { label: t("system.base.user.id_type.id_card"), value: BaseUserIDType.BASE_USER_ID_TYPE_ID_CARD },
  { label: t("system.base.user.id_type.passport"), value: BaseUserIDType.BASE_USER_ID_TYPE_PASSPORT },
  { label: t("system.base.user.id_type.hk_macao_permit"), value: BaseUserIDType.BASE_USER_ID_TYPE_HK_MACAO_PERMIT },
  { label: t("system.base.user.id_type.taiwan_permit"), value: BaseUserIDType.BASE_USER_ID_TYPE_TAIWAN_PERMIT },
  {
    label: t("system.base.user.id_type.foreign_permanent_residence"),
    value: BaseUserIDType.BASE_USER_ID_TYPE_FOREIGN_PERMANENT_RESIDENCE
  },
  { label: t("system.base.user.id_type.other"), value: BaseUserIDType.BASE_USER_ID_TYPE_OTHER }
]);

const accountFormRules = computed(() => ({
  email: [{ type: "email", message: t("system.base.user.validation.email_invalid"), trigger: "blur" }],
  id_code: [
    {
      pattern: /^[A-Za-z0-9][A-Za-z0-9-]{0,63}$/,
      message: t("system.base.user.validation.id_code_invalid"),
      trigger: "blur"
    }
  ]
}));

/** 根据资料中的性别值输出展示文案。 */
const genderText = computed(() => {
  if (props.profile.gender === 1) return t("common.value.male");
  if (props.profile.gender === 2) return t("common.value.female");
  return t("system.profile.value.private");
});

/** 根据证件类型输出展示文案。 */
const idTypeText = computed(() => {
  return idTypeOptions.value.find(item => item.value === props.profile.id_type)?.label ?? t("system.profile.value.unbound");
});

/**
 * 同步个人中心头像展示，优先使用用户头像，为空时回退默认头像。
 *
 * @param avatar 用户头像地址
 */
function syncAvatarSrc(avatar?: string) {
  // 个人中心与头部头像保持一致，统一使用本地默认头像兜底。
  avatarSrc.value = avatar || defaultAvatar;
}

watch(
  () => props.profile,
  profile => {
    accountForm.nick_name = profile.nick_name;
    accountForm.gender = profile.gender;
    accountForm.email = profile.email;
    accountForm.id_type = profile.id_type;
    accountForm.id_code = profile.id_code;
    syncAvatarSrc(profile.avatar);
  },
  { immediate: true, deep: true }
);

/** 头像加载失败时回退默认头像，避免出现空白或破图。 */
function handleAvatarError() {
  avatarSrc.value = defaultAvatar;
  return false;
}

/** 触发头像文件选择。 */
function triggerFileUpload() {
  fileInputRef.value?.click();
}

/** 打开基本资料编辑弹窗，并回填当前资料。 */
function openAccountDialog() {
  accountForm.nick_name = props.profile.nick_name;
  accountForm.gender = props.profile.gender;
  accountForm.email = props.profile.email;
  accountForm.id_type = props.profile.id_type;
  accountForm.id_code = props.profile.id_code;
  accountDialogVisible.value = true;
}

/** 处理头像文件上传并同步到个人资料。 */
async function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  try {
    const uploadResult = await defFileService.UploadFile(file, "avatar");
    await defProfileAuthService.UpdateUserProfile({
      user_profile: {
        ...props.profile,
        avatar: uploadResult.url
      }
    });
    ElMessage.success(t("system.profile.account.message.avatar_updated"));
    emit("refreshed");
  } catch (_error) {
    ElMessage.error(t("system.profile.account.message.avatar_upload_failed"));
  } finally {
    target.value = "";
  }
}

/** 提交基本资料更新。 */
async function handleSubmitAccount() {
  if (!(await accountFormRef.value?.validate())) return;

  submitLoading.value = true;
  try {
    await defProfileAuthService.UpdateUserProfile({
      user_profile: {
        ...props.profile,
        nick_name: accountForm.nick_name,
        gender: accountForm.gender,
        email: accountForm.email,
        id_type: accountForm.id_type,
        id_code: accountForm.id_code
      }
    });
    ElMessage.success(t("system.profile.account.message.updated"));
    accountDialogVisible.value = false;
    emit("refreshed");
  } finally {
    submitLoading.value = false;
  }
}

/** 弹窗关闭后清理校验状态，并恢复表单值。 */
function handleDialogClosed() {
  accountFormRef.value?.clearValidate();
  accountForm.nick_name = props.profile.nick_name;
  accountForm.gender = props.profile.gender;
  accountForm.email = props.profile.email;
  accountForm.id_type = props.profile.id_type;
  accountForm.id_code = props.profile.id_code;
}
</script>

<style scoped lang="scss">
.base-card {
  border: 1px solid #ebeef5;
  border-radius: var(--admin-page-radius);
}
:deep(.base-card .el-card__header) {
  padding: 18px 20px;
  border-bottom: 1px solid #f0f2f5;
}
:deep(.base-card .el-card__body) {
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
.base-layout {
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}
.avatar-panel {
  padding: 24px 20px;
  text-align: center;
  background: #f8fafc;
  border: 1px solid #eef2f6;
  border-radius: var(--admin-page-radius);
}
.avatar-shell {
  position: relative;
  display: inline-flex;
}
.avatar-trigger {
  position: absolute;
  right: 6px;
  bottom: 4px;
}
.hidden-input {
  display: none;
}
.avatar-copy {
  margin-top: 16px;
}
.avatar-copy strong {
  display: block;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}
.avatar-copy p {
  margin: 6px 0 0;
  font-size: 12px;
  color: #909399;
}
.detail-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.detail-item {
  padding: 16px;
  background: #ffffff;
  border: 1px solid #f0f2f5;
  border-radius: var(--admin-page-radius);
}
.detail-item--wide {
  grid-column: 1 / -1;
}
.detail-label {
  display: block;
  margin-bottom: 8px;
  font-size: 12px;
  color: #909399;
}
.detail-value {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  word-break: break-all;
}
.dialog-footer {
  display: flex;
  justify-content: flex-end;
}

@media screen and (width <= 992px) {
  .base-layout,
  .detail-grid {
    grid-template-columns: 1fr;
  }
  .detail-item--wide {
    grid-column: auto;
  }
}

@media screen and (width <= 640px) {
  .panel-header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
