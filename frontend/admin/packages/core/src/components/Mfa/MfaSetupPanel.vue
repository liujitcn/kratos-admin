<template>
  <div v-if="uri" class="mfa-setup-panel">
    <img v-if="qrDataUrl" class="mfa-setup-panel__qr" :src="qrDataUrl" :alt="t('core.login.mfa_setup_qr_alt')" />
    <el-collapse class="mfa-setup-panel__uri">
      <el-collapse-item :title="t('core.login.mfa_show_uri')" name="uri">
        <div class="mfa-setup-panel__value">
          <span class="mfa-setup-panel__text">{{ uri }}</span>
          <el-tooltip :content="t('core.login.mfa_copy_uri')" placement="top">
            <el-button text circle :aria-label="t('core.login.mfa_copy_uri')" @click.stop="copySetupUri">
              <el-icon class="mfa-setup-panel__copy"><CopyDocument /></el-icon>
            </el-button>
          </el-tooltip>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup lang="ts" name="MfaSetupPanel">
import { computed } from "vue";
import qrcode from "qrcode-generator";
import { useLocaleStore } from "@/locales";
import { copyText } from "@/utils/clipboard";

/** MFA TOTP 绑定面板属性。 */
interface MfaSetupPanelProps {
  /** TOTP 绑定地址。 */
  uri: string;
}

const props = defineProps<MfaSetupPanelProps>();
const { t } = useLocaleStore();

const qrDataUrl = computed(() => {
  if (!props.uri) return "";
  const code = qrcode(0, "M");
  code.addData(props.uri);
  code.make();
  return code.createDataURL(4, 8);
});

/** 复制 TOTP 绑定地址。 */
async function copySetupUri() {
  await copyText(props.uri);
  ElMessage.success(t("core.login.mfa_uri_copied"));
}
</script>

<style scoped lang="scss">
.mfa-setup-panel__qr {
  display: block;
  width: 220px;
  height: 220px;
  margin: 12px auto;
  image-rendering: pixelated;
}

.mfa-setup-panel__uri {
  margin-top: 12px;
}

.mfa-setup-panel__value {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  color: var(--el-text-color-regular);
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.mfa-setup-panel__text {
  flex: 1;
  min-width: 0;
  white-space: pre-wrap;
}

.mfa-setup-panel__copy {
  color: var(--el-color-primary);
  font-size: 18px;
}
</style>
