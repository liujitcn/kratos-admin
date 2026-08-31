<template>
  <div v-if="lockScreenStore.setupVisible" class="lock-setup-mask" role="presentation">
    <section class="lock-setup-dialog" role="dialog" aria-modal="true" :aria-label="t('core.layout.lock_screen')">
      <header class="lock-dialog-header">
        <h2>{{ t("core.layout.lock_screen") }}</h2>
        <button class="lock-icon-button" type="button" :aria-label="t('common.action.close')" @click="closeSetup">
          <el-icon><Close /></el-icon>
        </button>
      </header>
      <div class="lock-dialog-body">
        <div class="lock-avatar lock-avatar--setup">
          <img :src="avatarSrc" :alt="t('core.layout.avatar')" @error="handleAvatarError" />
        </div>
        <div class="lock-user-name">{{ displayName }}</div>
        <form class="lock-form" @submit.prevent="handleLock">
          <el-input
            v-model="setupPassword"
            class="lock-input"
            type="password"
            size="large"
            show-password
            autocomplete="new-password"
            :placeholder="t('core.layout.lock_screen_password_placeholder')"
            :aria-label="t('core.layout.lock_screen_password_placeholder')"
          />
          <p v-if="setupError" class="lock-form-error">{{ setupError }}</p>
          <el-button class="lock-submit" type="primary" size="large" native-type="submit" :loading="settingLock">
            {{ t("core.layout.lock_screen_action") }}
          </el-button>
        </form>
      </div>
    </section>
  </div>

  <div v-if="lockScreenStore.isLocked" class="lock-screen" role="dialog" aria-modal="true">
    <div class="lock-screen__trigger-wrap">
      <button
        v-if="!lockScreenStore.unlockVisible"
        class="lock-screen__trigger"
        type="button"
        @click="openUnlock"
      >
        <el-icon :size="26"><Lock /></el-icon>
        <span>{{ t("core.layout.lock_screen_unlock_hint") }}</span>
      </button>
      <section v-else class="unlock-panel" :aria-label="t('core.layout.lock_screen')">
        <div class="lock-avatar lock-avatar--unlock">
          <img :src="avatarSrc" :alt="t('core.layout.avatar')" @error="handleAvatarError" />
        </div>
        <div class="lock-user-name">{{ displayName }}</div>
        <form class="lock-form" @submit.prevent="handleUnlock">
          <el-input
            ref="unlockInputRef"
            v-model="unlockPassword"
            class="lock-input"
            type="password"
            size="large"
            show-password
            autofocus
            autocomplete="current-password"
            :placeholder="t('core.layout.lock_screen_password_placeholder')"
            :aria-label="t('core.layout.lock_screen_password_placeholder')"
          />
          <p v-if="lockScreenStore.unlockError" class="lock-form-error">{{ t("core.layout.lock_screen_password_invalid") }}</p>
          <el-button class="lock-submit" type="primary" size="large" native-type="submit" :loading="unlocking">
            {{ t("core.layout.lock_screen_enter") }}
          </el-button>
        </form>
        <button class="lock-text-button" type="button" :disabled="loggingOut" @click="returnToLogin">
          <el-icon><SwitchButton /></el-icon>
          <span>{{ t("core.layout.lock_screen_return_login") }}</span>
        </button>
        <button class="lock-text-button" type="button" @click="closeUnlock">
          <el-icon><ArrowLeft /></el-icon>
          <span>{{ t("common.action.back") }}</span>
        </button>
      </section>
    </div>
    <div class="lock-screen__clock" aria-live="polite">
      <time class="lock-screen__time">{{ timeText }}</time>
      <time class="lock-screen__date">{{ dateText }}</time>
    </div>
  </div>
</template>

<script setup lang="ts" name="LockScreen">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { LOGIN_URL } from "@/config";
import defaultAvatar from "@/assets/images/avatar.png";
import { useLocaleStore } from "@/locales";
import { useLockScreenStore } from "@/stores/modules/lockScreen";
import { useUserStore } from "@/stores/modules/user";

const router = useRouter();
const userStore = useUserStore();
const lockScreenStore = useLockScreenStore();
const { locale, t } = useLocaleStore();
const avatarSrc = ref(defaultAvatar);
const setupPassword = ref("");
const unlockPassword = ref("");
const setupError = ref("");
const settingLock = ref(false);
const unlocking = ref(false);
const loggingOut = ref(false);
const now = ref(new Date());
const unlockInputRef = ref<{ focus?: () => void }>();
let clockTimer: number | undefined;

const displayName = computed(() => userStore.userInfo.nick_name || userStore.userInfo.user_name || t("core.layout.not_set"));
const timeText = computed(() => {
  return new Intl.DateTimeFormat(locale.value, {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false
  }).format(now.value);
});
const dateText = computed(() => {
  return new Intl.DateTimeFormat(locale.value, {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    weekday: "long"
  }).format(now.value);
});

watch(
  () => userStore.userInfo.avatar,
  avatar => {
    avatarSrc.value = avatar || defaultAvatar;
  },
  { immediate: true }
);

watch(
  () => lockScreenStore.setupVisible,
  visible => {
    if (!visible) return;
    setupPassword.value = "";
    setupError.value = "";
  }
);

watch(
  () => lockScreenStore.isLocked,
  locked => {
    if (locked) {
      unlockPassword.value = "";
      nextTick(() => unlockInputRef.value?.focus?.());
    }
  }
);

onMounted(() => {
  clockTimer = window.setInterval(() => {
    now.value = new Date();
  }, 1000);
});

onBeforeUnmount(() => {
  if (clockTimer) window.clearInterval(clockTimer);
});

/** 关闭设置锁屏密码界面。 */
const closeSetup = () => {
  lockScreenStore.closeSetup();
};

/** 设置锁屏密码并进入锁屏界面。 */
const handleLock = async () => {
  if (!setupPassword.value) {
    setupError.value = t("core.layout.lock_screen_password_required");
    return;
  }

  settingLock.value = true;
  setupError.value = "";
  try {
    await lockScreenStore.lock(setupPassword.value);
  } finally {
    settingLock.value = false;
  }
};

/** 显示锁屏密码输入界面。 */
const openUnlock = () => {
  lockScreenStore.openUnlock();
  nextTick(() => unlockInputRef.value?.focus?.());
};

/** 校验锁屏密码并返回当前管理页面。 */
const handleUnlock = async () => {
  if (!unlockPassword.value) {
    lockScreenStore.unlockError = true;
    return;
  }

  unlocking.value = true;
  try {
    await lockScreenStore.unlock(unlockPassword.value);
    if (!lockScreenStore.isLocked) unlockPassword.value = "";
  } finally {
    unlocking.value = false;
  }
};

/** 返回登录页并清理当前登录态。 */
const returnToLogin = async () => {
  if (loggingOut.value) return;

  loggingOut.value = true;
  try {
    await userStore.logout();
    await router.replace(LOGIN_URL);
  } finally {
    loggingOut.value = false;
  }
};

/** 锁屏首页不再显示密码输入。 */
const closeUnlock = () => {
  lockScreenStore.closeUnlock();
  unlockPassword.value = "";
};

/** 头像加载失败时回退到本地默认头像。 */
const handleAvatarError = () => {
  avatarSrc.value = defaultAvatar;
};
</script>

<style scoped lang="scss">
.lock-setup-mask {
  position: fixed;
  inset: 0;
  z-index: 4100;
  display: grid;
  place-items: center;
  padding: 20px;
  background: rgb(0 0 0 / 60%);
}
.lock-setup-dialog {
  width: min(720px, 100%);
  overflow: hidden;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color);
  border-radius: 16px;
  box-shadow: 0 24px 80px rgb(0 0 0 / 28%);
}
.lock-dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 26px 32px;
  border-bottom: 1px solid var(--el-border-color-light);
  h2 {
    margin: 0;
    font-size: 28px;
    line-height: 1.25;
  }
}
.lock-icon-button {
  display: inline-grid;
  place-items: center;
  width: 40px;
  height: 40px;
  padding: 0;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 50%;
  &:hover {
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
  }
}
.lock-dialog-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 36px 32px 44px;
}
.lock-avatar {
  overflow: hidden;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  border-radius: 50%;
  img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  &--setup {
    width: 168px;
    height: 168px;
  }
  &--unlock {
    width: 132px;
    height: 132px;
  }
}
.lock-user-name {
  margin-top: 20px;
  font-size: 24px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--el-text-color-primary);
}
.lock-form {
  width: min(100%, 560px);
  margin-top: 34px;
}
.lock-input {
  :deep(.el-input__wrapper) {
    min-height: 58px;
    padding: 0 18px;
    background: transparent;
    border: 1px solid var(--el-border-color);
    border-radius: 12px;
    box-shadow: none;
  }
  :deep(.el-input__inner) {
    font-size: 18px;
  }
}
.lock-form-error {
  margin: 10px 2px 0;
  font-size: 13px;
  line-height: 1.4;
  color: var(--el-color-danger);
}
.lock-submit {
  width: 100%;
  min-height: 56px;
  margin-top: 28px;
  border-radius: 10px;
}
.lock-screen {
  position: fixed;
  inset: 0;
  z-index: 4000;
  display: flex;
  justify-content: center;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-page);
}
.lock-screen__trigger-wrap {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  width: 100%;
  padding-top: clamp(28px, 7vh, 72px);
}
.lock-screen__trigger {
  display: inline-flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
  padding: 10px 18px;
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 10px;
  &:hover,
  &:focus-visible {
    color: var(--el-text-color-primary);
    outline: none;
    background: var(--el-fill-color-light);
  }
}
.unlock-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: min(520px, calc(100vw - 32px));
  padding: 24px;
}
.unlock-panel .lock-form {
  margin-top: 28px;
}
.lock-text-button {
  display: inline-flex;
  gap: 7px;
  align-items: center;
  padding: 4px 8px;
  margin-top: 24px;
  font-size: 15px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 6px;
  &:hover,
  &:focus-visible {
    color: var(--el-text-color-primary);
    outline: none;
    background: var(--el-fill-color-light);
  }
  &:disabled {
    cursor: wait;
    opacity: 0.6;
  }
}
.lock-screen__clock {
  position: absolute;
  right: 24px;
  bottom: clamp(22px, 6vh, 64px);
  left: 24px;
  display: flex;
  flex-direction: column;
  align-items: center;
  pointer-events: none;
}
.lock-screen__time {
  font-size: clamp(44px, 8vw, 88px);
  font-weight: 300;
  line-height: 1;
}
.lock-screen__date {
  margin-top: 14px;
  font-size: clamp(17px, 2.4vw, 26px);
  line-height: 1.35;
  color: var(--el-text-color-secondary);
}

@media (width <= 600px) {
  .lock-dialog-header {
    padding: 20px 22px;
    h2 {
      font-size: 24px;
    }
  }
  .lock-dialog-body {
    padding: 28px 20px 32px;
  }
  .lock-avatar--setup {
    width: 132px;
    height: 132px;
  }
  .lock-screen__clock {
    bottom: 28px;
  }
}
</style>
