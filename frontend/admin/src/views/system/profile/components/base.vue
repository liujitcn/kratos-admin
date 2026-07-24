<template>
  <el-card class="base-card" shadow="never">
    <template #header>
      <div class="panel-header">
        <div>
          <h3>账号信息</h3>
          <p>维护头像、昵称和基础账号资料。</p>
        </div>
        <el-button type="primary" plain @click="openAccountDialog">修改基本资料</el-button>
      </div>
    </template>

    <div class="base-layout">
      <div class="avatar-panel">
        <div class="avatar-shell">
          <el-avatar :src="avatarSrc" :size="116" @error="handleAvatarError" />
          <el-button class="avatar-trigger" circle type="primary" :icon="Camera" @click="triggerFileUpload" />
          <input ref="fileInputRef" type="file" class="hidden-input" accept="image/*" @change="handleFileChange" />
        </div>
        <div class="avatar-copy">
          <strong>头像</strong>
          <p>点击右下角可更换头像</p>
        </div>
      </div>

      <div class="detail-grid">
        <div class="detail-item">
          <span class="detail-label">登录账号</span>
          <span class="detail-value">{{ profile.user_name || "--" }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">昵称</span>
          <span class="detail-value">{{ profile.nick_name || "--" }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">性别</span>
          <span class="detail-value">{{ genderText }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">绑定手机</span>
          <span class="detail-value">{{ profile.phone || "未绑定" }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">所属角色</span>
          <span class="detail-value">{{ profile.role_name || "--" }}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">所属部门</span>
          <span class="detail-value">{{ profile.dept_name || "--" }}</span>
        </div>
        <div class="detail-item detail-item--wide">
          <span class="detail-label">创建时间</span>
          <span class="detail-value">{{ profile.created_at || "--" }}</span>
        </div>
      </div>
    </div>

    <ProDialog v-model="accountDialogVisible" title="修改基本资料" :width="520" @closed="handleDialogClosed">
      <ProForm ref="accountFormRef" :model="accountForm" :fields="accountFormFields" label-width="96px" />
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="accountDialogVisible = false">取消</el-button>
          <el-button type="primary" :loading="submitLoading" @click="handleSubmitAccount">保存</el-button>
        </div>
      </template>
    </ProDialog>
  </el-card>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { defAuthService } from "@/api/system/auth";
import { defFileService } from "@/api/base/file";
import ProForm from "@/components/ProForm/index.vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import type { UserProfileForm } from "@/rpc/system/admin/v1/auth";
import { ElMessage } from "element-plus";
import { Camera } from "@element-plus/icons-vue";
import defaultAvatar from "@/assets/images/avatar.png";

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
const accountForm = reactive<Pick<UserProfileForm, "nick_name" | "gender">>({
  nick_name: "",
  gender: 3
});

const accountFormFields: ProFormField[] = [
  { prop: "nick_name", label: "昵称", component: "input", props: { placeholder: "请输入昵称" } },
  { prop: "gender", label: "性别", component: "dict", props: { code: "base_user_gender" } }
];

/** 根据资料中的性别值输出展示文案。 */
const genderText = computed(() => {
  if (props.profile.gender === 1) return "男";
  if (props.profile.gender === 2) return "女";
  return "保密";
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
  accountDialogVisible.value = true;
}

/** 处理头像文件上传并同步到个人资料。 */
async function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement;
  const file = target.files?.[0];
  if (!file) return;

  try {
    const uploadResult = await defFileService.UploadFile(file, "avatar");
    await defAuthService.UpdateUserProfile({
      user_profile: {
        ...props.profile,
        avatar: uploadResult.url
      }
    });
    ElMessage.success("头像更新成功");
    emit("refreshed");
  } catch (_error) {
    ElMessage.error("头像上传失败");
  } finally {
    target.value = "";
  }
}

/** 提交基本资料更新。 */
async function handleSubmitAccount() {
  if (!(await accountFormRef.value?.validate())) return;

  submitLoading.value = true;
  try {
    await defAuthService.UpdateUserProfile({
      user_profile: {
        ...props.profile,
        nick_name: accountForm.nick_name,
        gender: accountForm.gender
      }
    });
    ElMessage.success("基本资料已更新");
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
}
</script>

<style scoped lang="scss">
.base-card {
  border: 1px solid #ebeef5;
  border-radius: 12px;
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
  border-radius: 12px;
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
  border-radius: 10px;
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
