<template>
  <template v-if="enabled">
    <slot name="trigger" :open="openDialog">
      <el-tooltip :content="t('system.base.i18n.field.i18ns')" placement="top" :show-after="250">
        <el-button
          :icon="Languages"
          :disabled="disabled"
          :aria-label="t('system.base.i18n.field.i18ns')"
          aria-haspopup="dialog"
          :aria-expanded="visible"
          @click="openDialog"
        />
      </el-tooltip>
    </slot>
    <ProDialog
      v-model="visible"
      :title="t('system.base.i18n.field.i18ns')"
      width="min(800px, calc(100vw - 32px))"
      append-to-body
      destroy-on-close
      :confirm-loading="translating"
      :show-footer="!readonly"
      @confirm="confirmChanges"
      @close="closeDialog"
    >
      <div class="i18n-editor">
        <div v-if="!readonly" class="i18n-editor__toolbar">
          <el-button
            type="primary"
            :loading="translating && !translatingLocale"
            :disabled="!source || translating"
            @click="translate()"
          >
            {{ t("system.base.i18n.action.batch_translate") }}
          </el-button>
        </div>
        <el-form label-position="right" label-width="100px" @submit.prevent>
          <el-form-item
            v-for="item in draftValues"
            :key="item.locale"
            :label="getLanguageLabel(item.locale)"
            :for="`${inputId}-${item.locale}`"
          >
            <div class="i18n-editor__control">
              <el-input
                :id="`${inputId}-${item.locale}`"
                :model-value="item.text || source"
                :maxlength="maxlength"
                :type="multiline ? 'textarea' : 'text'"
                :rows="multiline ? 5 : undefined"
                :disabled="translating || readonly"
                show-word-limit
                @update:model-value="value => (item.text = value)"
              />
              <el-button
                v-if="!readonly"
                link
                type="primary"
                :loading="translatingLocale === item.locale"
                :disabled="!source || translating"
                @click="translate(item.locale)"
              >
                {{ t("system.base.i18n.action.translate") }}
              </el-button>
            </div>
          </el-form-item>
        </el-form>
      </div>
    </ProDialog>
  </template>
</template>

<script setup lang="ts">
import { computed, ref, useId, watch } from "vue";
import { Languages } from "@lucide/vue";
import { t } from "@liujitcn/kratos-admin-core";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import { defBaseI18nService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_i18n";
import { getEditableLanguageOptions, getLanguageLabel, type DynamicI18nValue } from "./dynamicI18n";

/** 字段旁国际化入口及本地草稿编辑弹窗。 */
interface DynamicI18nEditorProps {
  modelValue: DynamicI18nValue[];
  source?: string;
  maxlength?: number;
  multiline?: boolean;
  disabled?: boolean;
  readonly?: boolean;
}
const props = withDefaults(defineProps<DynamicI18nEditorProps>(), {
  source: "",
  maxlength: 100,
  multiline: false,
  disabled: false,
  readonly: false
});
const emit = defineEmits<{ "update:modelValue": [value: DynamicI18nValue[]] }>();
const enabled = computed(() => getEditableLanguageOptions().length > 0);
const inputId = useId();
const visible = ref(false);
const draftValues = ref<DynamicI18nValue[]>([]);
const translating = ref(false);
const translatingLocale = ref("");
let requestVersion = 0;

/** 打开编辑窗口，未填写译文使用原值展示，不提前写入主表单。 */
function openDialog() {
  draftValues.value = getEditableLanguageOptions().map(({ value }) => ({
    locale: value,
    id: props.modelValue.find(item => item.locale === value)?.id ?? 0,
    text: props.modelValue.find(item => item.locale === value)?.text ?? ""
  }));
  visible.value = true;
}

/** 将用户确认的译文交给主表单，实际保存由主表单负责。 */
function confirmChanges() {
  if (translating.value) return;
  const editable = new Set(draftValues.value.map(item => item.locale));
  emit("update:modelValue", [
    ...props.modelValue.filter(item => !editable.has(item.locale)),
    ...draftValues.value.map(item => ({ ...item }))
  ]);
  visible.value = false;
}

/** 关闭时丢弃草稿并使尚未返回的翻译结果失效。 */
function closeDialog() {
  requestVersion++;
  translating.value = false;
  translatingLocale.value = "";
  draftValues.value = [];
}

/** 用当前原文翻译指定语言或全部目标语言，成功结果直接替换草稿。 */
async function translate(locale?: string) {
  if (!props.source || translating.value) return;
  const version = ++requestVersion;
  const source = props.source;
  translating.value = true;
  translatingLocale.value = locale ?? "";
  try {
    const response = await defBaseI18nService.DraftBaseI18n({ source, locale });
    if (version !== requestVersion || source !== props.source || !visible.value) return;
    const targets = draftValues.value.filter(item => !locale || item.locale === locale);
    const results = new Map(response.i18ns.filter(item => item.i18n).map(item => [item.locale, item.i18n]));
    let success = 0;
    for (const item of targets) {
      const text = results.get(item.locale);
      if (text === undefined) continue;
      item.text = text;
      success++;
    }
    const failed = targets.length - success;
    if (failed) ElMessage.warning(t("system.base.i18n.message.batch_translate_partial", { success, failed }));
    else ElMessage.success(t("system.base.i18n.message.batch_translate_success", { count: success }));
  } catch {
    if (version === requestVersion) ElMessage.error(t("system.base.i18n.message.batch_translate_failed"));
  } finally {
    if (version === requestVersion) {
      translating.value = false;
      translatingLocale.value = "";
    }
  }
}

watch(enabled, value => {
  if (!value) visible.value = false;
});
watch(
  () => props.source,
  () => {
    requestVersion++;
    translating.value = false;
    translatingLocale.value = "";
  }
);
</script>

<style scoped>
.i18n-editor {
  display: grid;
  gap: 16px;
}
.i18n-editor__toolbar {
  display: flex;
  justify-content: flex-end;
}
.i18n-editor__control {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: flex-start;
  gap: 12px;
}
.i18n-editor__control :deep(.el-input),
.i18n-editor__control :deep(.el-textarea) {
  flex: 1;
  min-width: 0;
}
</style>
