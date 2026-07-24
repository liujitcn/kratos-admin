<template>
  <el-dropdown ref="dropdownRef" trigger="click" placement="bottom-end" popper-class="header-avatar-dropdown">
    <div class="avatar">
      <img :src="avatarSrc" alt="avatar" @error="handleAvatarError" />
    </div>
    <template #dropdown>
      <div class="user-panel">
        <div class="user-panel__summary">
          <div class="user-panel__avatar">
            <img :src="avatarSrc" alt="avatar" @error="handleAvatarError" />
          </div>
          <div class="user-panel__identity">
            <div class="user-panel__name">{{ displayName }}</div>
            <div class="user-panel__meta">{{ profileSummary }}</div>
          </div>
        </div>
        <div class="user-panel__actions">
          <button class="action-btn action-btn--primary" type="button" @click.stop="handleGoToProfile">
            <el-icon><User /></el-icon>
            <span>个人中心</span>
          </button>
          <button class="action-btn action-btn--danger" type="button" @click="logout">
            <el-icon><SwitchButton /></el-icon>
            <span>退出登录</span>
          </button>
        </div>
      </div>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { LOGIN_URL } from "@/config";
import { useRouter } from "vue-router";
import { useUserStore } from "@/stores/modules/user";
import { ElMessageBox, ElMessage } from "element-plus";
import type { DropdownInstance } from "element-plus";
import defaultAvatar from "@/assets/images/avatar.png";

const router = useRouter();
const userStore = useUserStore();
const dropdownRef = ref<DropdownInstance>();
const avatarSrc = ref(defaultAvatar);
const displayName = computed(() => userStore.userInfo.nick_name || userStore.userInfo.user_name || "未设置");
const roleName = computed(() => userStore.userInfo.role_name || "未分配角色");
const deptName = computed(() => userStore.userInfo.dept_name || "未分配部门");
const profileSummary = computed(() => `${roleName.value} / ${deptName.value}`);

/**
 * 同步头部头像展示，优先使用用户头像，为空时回退默认头像。
 *
 * @param avatar 用户头像地址
 */
const syncAvatarSrc = (avatar?: string) => {
  // 用户未上传头像时，统一回退到本地默认头像。
  avatarSrc.value = avatar || defaultAvatar;
};

watch(
  () => userStore.userInfo.avatar,
  avatar => {
    syncAvatarSrc(avatar);
  },
  { immediate: true }
);

/** 退出登录并清理当前登录态。 */
const logout = () => {
  ElMessageBox.confirm("您是否确认退出登录?", "温馨提示", {
    confirmButtonText: "确定",
    cancelButtonText: "取消",
    type: "warning"
  }).then(async () => {
    // 1.执行退出登录接口并清理本地状态
    await userStore.logout();

    // 2.重定向到登录页
    router.replace(LOGIN_URL);
    ElMessage.success("退出登录成功！");
  });
};

/** 头像加载失败时兜底显示默认头像，避免出现破图。 */
const handleAvatarError = () => {
  avatarSrc.value = defaultAvatar;
};

/**
 * 跳转到个人中心页。
 * 头像面板统一收敛到个人中心入口，进入后默认展示账号信息模块。
 */
const goToProfile = async () => {
  await router.push("/profile");
};

/** 关闭头像下拉后跳转个人中心，避免自定义浮层吞掉点击行为。 */
const handleGoToProfile = async () => {
  dropdownRef.value?.handleClose();
  await goToProfile();
};
</script>

<style scoped lang="scss">
.avatar {
  width: 40px;
  height: 40px;
  overflow: hidden;
  cursor: pointer;
  border: 2px solid rgb(255 255 255 / 85%);
  border-radius: 50%;
  box-shadow: 0 8px 18px rgb(15 23 42 / 12%);
  img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}
.user-panel {
  width: 280px;
  padding: 16px;
  background: linear-gradient(180deg, #ffffff 0%, #f7f9fc 100%);
  border: 1px solid rgb(15 23 42 / 8%);
  border-radius: 12px;
  box-shadow: 0 16px 40px rgb(15 23 42 / 14%);
  &__summary {
    display: flex;
    align-items: center;
    padding-bottom: 14px;
    border-bottom: 1px solid rgb(15 23 42 / 8%);
  }
  &__avatar {
    flex-shrink: 0;
    width: 48px;
    height: 48px;
    overflow: hidden;
    background: #edf2f7;
    border-radius: 50%;
    img {
      display: block;
      width: 100%;
      height: 100%;
      object-fit: cover;
    }
  }
  &__identity {
    min-width: 0;
    margin-left: 12px;
  }
  &__name {
    font-size: 16px;
    font-weight: 600;
    line-height: 22px;
    color: #1f2937;
  }
  &__meta {
    margin-top: 3px;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: 13px;
    line-height: 18px;
    color: #6b7280;
    white-space: nowrap;
  }
  &__actions {
    display: flex;
    flex-direction: column;
    gap: 10px;
    margin-top: 14px;
  }
}
.action-btn {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 38px;
  cursor: pointer;
  border: none;
  border-radius: 10px;
  transition:
    transform 0.2s ease,
    box-shadow 0.2s ease,
    background-color 0.2s ease;

  // 悬浮时仅做轻微上浮，避免头部浮层交互显得过重。
  &:hover {
    transform: translateY(-1px);
  }
  &--primary {
    color: #1d4ed8;
    background: rgb(37 99 235 / 10%);
    &:hover {
      background: rgb(37 99 235 / 14%);
      box-shadow: 0 8px 18px rgb(37 99 235 / 16%);
    }
  }
  &--danger {
    color: #dc2626;
    background: rgb(220 38 38 / 8%);
    &:hover {
      background: rgb(220 38 38 / 12%);
      box-shadow: 0 8px 18px rgb(220 38 38 / 14%);
    }
  }
}
:deep(.header-avatar-dropdown) {
  padding: 0;
  background: transparent;
  border: none;
  box-shadow: none;
}
</style>
