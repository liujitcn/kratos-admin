<template>
  <ProDialog
    :model-value="modelValue"
    :title="t('core.login.mfa_recovery_codes_title')"
    width="420px"
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="mfa-recovery-dialog__codes">
      <span>{{ codesText }}</span>
      <el-tooltip :content="t('core.login.mfa_copy_recovery_codes')" placement="top">
        <el-button text circle :aria-label="t('core.login.mfa_copy_recovery_codes')" @click="copyRecoveryCodes">
          <el-icon class="mfa-recovery-dialog__copy"><CopyDocument /></el-icon>
        </el-button>
      </el-tooltip>
    </div>
    <p class="mfa-recovery-dialog__warning">{{ t("core.login.mfa_recovery_codes_warning") }}</p>
    <template #footer>
      <el-button type="primary" @click="handleConfirm">{{ t("core.login.mfa_recovery_codes_confirm") }}</el-button>
    </template>
  </ProDialog>
</template>

<script setup lang="ts" name="MfaRecoveryCodesDialog">
import { computed } from "vue";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import { useLocaleStore } from "@/locales";
import { copyText } from "@/utils/clipboard";

/** MFA 恢复码弹窗属性。 */
interface MfaRecoveryCodesDialogProps {
  /** 弹窗显示状态。 */
  modelValue: boolean;
  /** 本次生成的一次性恢复码。 */
  codes: string[];
}

const props = withDefaults(defineProps<MfaRecoveryCodesDialogProps>(), {
  modelValue: false,
  codes: () => []
});

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
  confirm: [];
}>();

const { t } = useLocaleStore();
const codesText = computed(() => props.codes.join("\n"));

/** 复制当前显示的恢复码。 */
async function copyRecoveryCodes() {
  await copyText(codesText.value);
  ElMessage.success(t("core.login.mfa_recovery_codes_copied"));
}

/** 关闭弹窗并通知调用方完成绑定流程。 */
function handleConfirm() {
  emit("update:modelValue", false);
  emit("confirm");
}
</script>

<style scoped lang="scss">
.mfa-recovery-dialog__codes {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  color: var(--el-text-color-regular);
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.mfa-recovery-dialog__codes span {
  flex: 1;
  min-width: 0;
  white-space: pre-wrap;
}

.mfa-recovery-dialog__copy {
  color: var(--el-color-primary);
  font-size: 18px;
}

.mfa-recovery-dialog__warning {
  margin: 8px 0 0;
  color: var(--el-color-warning);
  font-size: 13px;
  line-height: 1.6;
}
</style>
