<template>
  <el-input
    :model-value="modelValue"
    :type="visible ? 'text' : 'password'"
    autocomplete="new-password"
    v-bind="inputAttrs"
    @update:model-value="onInput"
  >
    <template #suffix>
      <el-icon class="secret-input__toggle" @click="toggle">
        <Loading v-if="loading" />
        <View v-else-if="visible" />
        <Hide v-else />
      </el-icon>
    </template>
  </el-input>
</template>

<script setup lang="ts">
import { computed, ref, useAttrs } from "vue";
import { ElMessage } from "element-plus";
import { Hide, Loading, View } from "@element-plus/icons-vue";
import type { ProFormSecretConfig } from "@/components/ProForm/interface";
import { revealSecret } from "@/utils/secretCrypto";
import { t } from "@/locales";

/** SecretInput 组件属性。 */
interface SecretInputProps {
  /** 字段值。 */
  modelValue?: string;
  /** 查看配置：声明后眼睛点击时按需拉取明文。 */
  secret?: boolean | ProFormSecretConfig;
  /** 查看的资源标识。 */
  resource?: string;
  /** 查看的字段名。 */
  fieldName?: string;
  /** 记录ID。 */
  recordId?: number;
  /** 是否已配置密钥。 */
  configured?: boolean;
}

const props = defineProps<SecretInputProps>();

const emit = defineEmits<{
  (event: "update:modelValue", value: string): void;
}>();

/** 明文是否可见。 */
const visible = ref(false);
/** 明文是否来自远端查看。 */
const revealed = ref(false);
/** 查看请求进行中。 */
const loading = ref(false);

/** 解析查看配置。 */
const secretConfig = computed<ProFormSecretConfig | null>(() =>
  typeof props.secret === "object" ? props.secret : null
);

// 透传调用方的 el-input 参数（maxlength、placeholder 等）。
const attrs = useAttrs();

/** 过滤掉原生密码框专属参数后透传其余输入参数。 */
const inputAttrs = computed(() => {
  const { showPassword: _ignored, ...rest } = { ...(attrs as Record<string, unknown>) };
  return rest;
});

/** 判断当前是否可以查看远端明文。 */
function canReveal() {
  return Boolean(
    secretConfig.value && props.resource && props.fieldName && (props.recordId ?? 0) > 0 && props.configured
  );
}

/** 切换明文可见状态：查看态隐藏时清空回填的明文，保持"留空表示保留"语义。 */
async function toggle() {
  if (loading.value) return;
  if (visible.value) {
    visible.value = false;
    if (revealed.value) {
      revealed.value = false;
      emit("update:modelValue", "");
    }
    return;
  }
  if (props.modelValue) {
    visible.value = true;
    return;
  }
  if (!canReveal()) return;
  loading.value = true;
  try {
    const text = await revealSecret(props.resource!, (props.recordId ?? 0), props.fieldName!);
    emit("update:modelValue", text);
    revealed.value = true;
    visible.value = true;
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : t("core.password.reveal_failed"));
  } finally {
    loading.value = false;
  }
}

/** 输入回调：本地输入的值不再视为远端明文。 */
function onInput(value: string) {
  revealed.value = false;
  emit("update:modelValue", value);
}
</script>

<style scoped lang="scss">
.secret-input__toggle {
  cursor: pointer;
}
</style>
