<template>
  <div class="password-strength">
    <div class="password-strength__header">
      <span>{{ t("core.password.strength") }}</span>
      <strong :class="`password-strength__label password-strength__label--${strength.level}`">
        {{ t(`core.password.strength.${strength.level}`) }}
      </strong>
    </div>
    <div class="password-strength__bars">
      <span
        v-for="segment in segments"
        :key="segment"
        class="password-strength__bar"
        :class="{
          'password-strength__bar--active': segment <= strength.strengthScore,
          [`password-strength__bar--${strength.level}`]: segment <= strength.strengthScore
        }"
      />
    </div>
    <p class="password-strength__tip">{{ displayedTip }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { defAuthService } from "@/api/system/admin/v1/auth";
import type { CurrentPasswordPolicy } from "@/rpc/system/admin/v1/auth";
import { getPasswordStrength } from "@/utils/passwordStrength";
import { useLocaleStore } from "@/locales";

const { t } = useLocaleStore();

/** 密码强度组件属性。 */
interface PasswordStrengthProps {
  /** 当前密码值。 */
  password?: string;
  /** 底部提示文案。 */
  tip?: string;
  /** 已知的密码策略；传入后直接用于摘要展示。 */
  policy?: CurrentPasswordPolicy;
  /** 是否加载当前登录用户生效的密码策略。 */
  loadCurrentPolicy?: boolean;
}

const props = withDefaults(defineProps<PasswordStrengthProps>(), {
  password: "",
  tip: "",
  policy: undefined,
  loadCurrentPolicy: false
});

const currentPolicy = ref<CurrentPasswordPolicy>();

/** 强度条固定为三段，保持所有页面一致。 */
const segments = [1, 2, 3];

/** 根据输入密码实时输出强度结果。 */
const strength = computed(() => getPasswordStrength(props.password));

/** 将生效策略压缩为用户可直接理解的一句话。 */
const displayedTip = computed(() => {
  if (props.tip) return props.tip;
  const policy = props.policy ?? currentPolicy.value;
  if (!policy) return t("core.password.tip");

  const requirements = [
    t("core.password.policy.min_length", { count: policy.min_length }),
    t("core.password.policy.complexity", { count: policy.min_complexity_classes })
  ];
  if (policy.history_count > 0) {
    requirements.push(t("core.password.policy.history", { count: policy.history_count }));
  }
  requirements.push(
    policy.max_age_days > 0
      ? t("core.password.policy.max_age", { count: policy.max_age_days })
      : t("core.password.policy.no_expiry")
  );
  return requirements.join(t("core.password.policy.separator"));
});

/** 按需加载当前登录用户实际生效的密码策略。 */
onMounted(async () => {
  if (!props.loadCurrentPolicy || props.policy) return;
  try {
    currentPolicy.value = await defAuthService.GetCurrentPasswordPolicy({});
  } catch (_error) {
    currentPolicy.value = undefined;
  }
});
</script>

<style scoped lang="scss">
.password-strength {
  padding: 14px 16px;
  background: #fafbfd;
  border: 1px solid #ebeef5;
  border-radius: var(--admin-page-radius);
}
.password-strength__header {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
}
.password-strength__header span {
  font-size: 13px;
  color: #606266;
}
.password-strength__label {
  font-size: 13px;
  font-weight: 600;
  transition: color 0.2s ease;
}
.password-strength__label--empty {
  color: #909399;
}
.password-strength__label--low {
  color: #f56c6c;
}
.password-strength__label--medium {
  color: #e6a23c;
}
.password-strength__label--high {
  color: #67c23a;
}
.password-strength__bars {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin-top: 10px;
}
.password-strength__bar {
  height: 8px;
  background: #ebeef5;
  border-radius: 999px;
  transform: scaleX(0.92);
  transform-origin: left center;
  transition:
    background-color 0.25s ease,
    transform 0.25s ease,
    box-shadow 0.25s ease;
}
.password-strength__bar:nth-child(1) {
  transition-delay: 0s;
}
.password-strength__bar:nth-child(2) {
  transition-delay: 0.08s;
}
.password-strength__bar:nth-child(3) {
  transition-delay: 0.16s;
}
.password-strength__bar--active {
  box-shadow: 0 0 0 1px rgb(255 255 255 / 18%) inset;
  transform: scaleX(1);
}
.password-strength__bar--low {
  background: #f56c6c;
}
.password-strength__bar--medium {
  background: #e6a23c;
}
.password-strength__bar--high {
  background: #67c23a;
}
.password-strength__tip {
  margin: 10px 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: #909399;
}
</style>
