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
          <el-button class="lock-submit lock-submit--lock" size="large" native-type="submit" :loading="settingLock">
            <el-icon><Lock /></el-icon>
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
    <div class="lock-screen__clock" :class="{ 'lock-screen__clock--compact': lockScreenStore.unlockVisible }" aria-live="polite">
      <div v-if="lockScreenStore.unlockVisible" class="lock-screen__compact-time" :aria-label="`${periodText} ${hourText}:${minuteText}`">
        <time class="lock-screen__compact-time-value">{{ hourText }}:{{ minuteText }}</time>
        <span class="lock-screen__compact-period">{{ periodText }}</span>
      </div>
      <div v-else class="lock-screen__time-grid" :aria-label="`${periodText} ${hourText}:${minuteText}`">
        <div class="lock-screen__time-card">
          <span class="lock-screen__period">{{ periodText }}</span>
          <time class="lock-screen__time-value">{{ hourText }}</time>
        </div>
        <div class="lock-screen__time-card">
          <time class="lock-screen__time-value">{{ minuteText }}</time>
        </div>
      </div>
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

/** 当前小时，使用 24 小时制并补齐两位。 */
const hourText = computed(() => String(now.value.getHours()).padStart(2, "0"));

/** 当前分钟，补齐两位。 */
const minuteText = computed(() => String(now.value.getMinutes()).padStart(2, "0"));

/** 当前时间所属的上午或下午时段。 */
const periodText = computed(() => (now.value.getHours() < 12 ? "AM" : "PM"));

/** 按“日期 星期”格式展示当前本地日期。 */
const dateText = computed(() => {
  const year = now.value.getFullYear();
  const month = String(now.value.getMonth() + 1).padStart(2, "0");
  const day = String(now.value.getDate()).padStart(2, "0");
  const weekday = new Intl.DateTimeFormat(locale.value, { weekday: "long" }).format(now.value);
  return `${year}-${month}-${day} ${weekday}`;
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
  width: min(600px, 100%);
  overflow: hidden;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius);
  box-shadow: 0 24px 80px rgb(0 0 0 / 28%);
}
.lock-dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid var(--el-border-color-light);
  h2 {
    margin: 0;
    font-size: 24px;
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
  padding: 30px 28px 36px;
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
    width: 136px;
    height: 136px;
  }
  &--unlock {
    width: 132px;
    height: 132px;
  }
}
.lock-user-name {
  margin-top: 16px;
  font-size: 22px;
  font-weight: 600;
  line-height: 1.35;
  color: var(--el-text-color-primary);
}
.lock-form {
  width: min(100%, 480px);
  margin-top: 28px;
}
.lock-input {
  :deep(.el-input__wrapper) {
    box-sizing: border-box;
    height: 48px;
    padding: 0 14px;
    background: var(--el-fill-color-blank);
    border: 1px solid var(--el-border-color);
    border-radius: var(--admin-page-radius);
    box-shadow: none;
    transition:
      border-color var(--el-transition-duration),
      box-shadow var(--el-transition-duration);
  }
  :deep(.el-input__wrapper:hover) {
    border-color: var(--el-border-color-hover);
  }
  :deep(.el-input__wrapper.is-focus) {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
  }
  :deep(.el-input__inner) {
    font-size: 16px;
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
  height: 48px;
  margin-top: 20px;
  font-weight: 600;
  border-radius: var(--admin-page-radius);
}
.lock-submit--lock {
  --el-button-text-color: var(--el-text-color-regular);
  --el-button-bg-color: var(--el-fill-color-light);
  --el-button-border-color: transparent;
  --el-button-hover-text-color: var(--el-text-color-primary);
  --el-button-hover-bg-color: var(--el-fill-color);
  --el-button-hover-border-color: var(--el-border-color-light);
  --el-button-active-text-color: var(--el-text-color-primary);
  --el-button-active-bg-color: var(--el-fill-color-dark);
  --el-button-active-border-color: var(--el-border-color);
  --el-button-disabled-text-color: var(--el-text-color-placeholder);
  --el-button-disabled-bg-color: var(--el-fill-color-light);
  --el-button-disabled-border-color: transparent;
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
  position: relative;
  z-index: 1;
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
  border-radius: var(--admin-page-radius);
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
  border-radius: var(--admin-page-radius);
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
  inset: clamp(108px, 15vh, 150px) clamp(20px, 8vw, 180px) clamp(18px, 3vh, 36px);
  display: flex;
  flex-direction: column;
  gap: clamp(18px, 3vh, 30px);
  align-items: center;
  justify-content: center;
  pointer-events: none;
}
.lock-screen__clock--compact {
  inset: auto 24px clamp(18px, 4vh, 36px);
  gap: 10px;
}
.lock-screen__compact-time {
  display: flex;
  gap: 8px;
  align-items: baseline;
  font-variant-numeric: tabular-nums;
  color: var(--el-text-color-primary);
}
.lock-screen__compact-time-value {
  font-size: 30px;
  font-weight: 400;
  line-height: 1;
  letter-spacing: 0;
}
.lock-screen__compact-period {
  font-size: 16px;
  line-height: 1;
}
.lock-screen__time-grid {
  display: grid;
  flex: 1;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: clamp(20px, 4vw, 72px);
  width: min(100%, 1536px);
  min-height: 0;
}
.lock-screen__time-card {
  position: relative;
  display: grid;
  place-items: center;
  min-width: 0;
  overflow: hidden;
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius);
}
.lock-screen__period {
  position: absolute;
  top: clamp(16px, 3vh, 28px);
  left: clamp(16px, 2vw, 28px);
  font-size: 18px;
  font-weight: 600;
  line-height: 1;
  color: var(--el-text-color-regular);
}
.lock-screen__time-value {
  font-size: clamp(112px, 31vh, 272px);
  font-weight: 300;
  font-variant-numeric: tabular-nums;
  line-height: 1;
  letter-spacing: 0;
}
.lock-screen__date {
  flex: none;
  font-size: clamp(20px, 3vh, 30px);
  font-variant-numeric: tabular-nums;
  line-height: 1.35;
  color: var(--el-text-color-primary);
  letter-spacing: 0;
}

@media (width <= 600px) {
  .lock-dialog-header {
    padding: 18px 20px;
    h2 {
      font-size: 22px;
    }
  }
  .lock-dialog-body {
    padding: 24px 20px 28px;
  }
  .lock-avatar--setup {
    width: 112px;
    height: 112px;
  }
  .lock-screen__clock {
    inset: 108px 16px 24px;
    gap: 16px;
  }
  .lock-screen__clock--compact {
    inset: auto 16px 24px;
    gap: 8px;
  }
  .lock-screen__time-grid {
    flex: none;
    gap: 12px;
    height: min(42vh, 220px);
  }
  .lock-screen__period {
    top: 12px;
    left: 12px;
    font-size: 14px;
  }
  .lock-screen__time-value {
    font-size: clamp(72px, 17vh, 120px);
  }
  .lock-screen__date {
    font-size: 18px;
  }
  .lock-screen__compact-time-value {
    font-size: 26px;
  }
  .lock-screen__compact-period {
    font-size: 14px;
  }
}
</style>
