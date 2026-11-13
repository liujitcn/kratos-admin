<template>
  <div v-if="active" class="security-example">
    <template v-if="componentId === 'SecretInput'">
      <SecretInput v-model="password" placeholder="输入示例密钥" />
      <p class="example-tip">点击眼睛切换本地输入的明文与掩码，不读取账号密钥。</p>
    </template>
    <template v-else-if="componentId === 'PasswordStrength'">
      <el-input v-model="password" type="password" show-password placeholder="输入示例密码" />
      <PasswordStrength :password="password" tip="建议使用至少 8 位字符，组合大小写字母、数字和符号。" />
    </template>
    <template v-else-if="componentId === 'MfaSetupPanel'">
      <MfaSetupPanel :uri="setupUri" />
      <p class="example-tip">此二维码使用公开演示密钥，仅用于查看绑定面板。</p>
    </template>
    <template v-else>
      <el-button type="primary" @click="visible = true">查看示例恢复码</el-button>
      <p class="example-tip" role="status">{{ saved ? "已确认保存演示恢复码。" : "可查看、复制并确认保存恢复码。" }}</p>
      <MfaRecoveryCodesDialog v-model="visible" :codes="codes" @confirm="saved = true" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { onDeactivated, ref } from "vue";
import SecretInput from "@/components/ProForm/components/SecretInput.vue";
import PasswordStrength from "@/components/PasswordStrength/index.vue";
import MfaSetupPanel from "@/components/Mfa/MfaSetupPanel.vue";
import MfaRecoveryCodesDialog from "@/components/Mfa/MfaRecoveryCodesDialog.vue";
import { useDemoActive } from "./useDemoActive";

defineProps<{ componentId: string }>();
const active = useDemoActive();
const password = ref("Demo-2026!");
const visible = ref(false);
const saved = ref(false);
const setupUri = "otpauth://totp/Demo:component-docs?secret=JBSWY3DPEHPK3PXP&issuer=Demo";
const codes = ["DEMO-0001-ABCD", "DEMO-0002-EFGH", "DEMO-0003-IJKL", "DEMO-0004-MNOP"];
onDeactivated(() => {
  visible.value = false;
});
</script>

<style scoped lang="scss">
.security-example {
  max-width: 560px;
}
.example-tip {
  margin: 16px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
</style>
