<template>
  <el-dropdown trigger="click" placement="bottom-end" @command="handleLocaleChange">
    <button class="locale-switch" type="button" :title="t('core.header.language')">
      <el-icon><Connection /></el-icon>
      <span>{{ currentLabel }}</span>
    </button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item v-for="item in localeOptions" :key="item.value" :command="item.value" :disabled="item.value === locale">
          {{ item.label }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { SUPPORTED_LOCALES, type SupportedLocale, useLocaleStore } from "@/locales";

const { locale, setLocale, t } = useLocaleStore();
const localeOptions = computed(() => {
  return SUPPORTED_LOCALES.map(value => ({ value, label: t(`common.language.${value}`) }));
});
const currentLabel = computed(() => t(`common.language.${locale.value}`));

/** 切换管理端语言并刷新动态本地化数据。 */
function handleLocaleChange(value: SupportedLocale) {
  void setLocale(value);
}
</script>

<style scoped lang="scss">
.locale-switch {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  height: 32px;
  padding: 0 8px;
  color: inherit;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 4px;

  &:hover {
    background: var(--el-fill-color-light);
  }

  span {
    max-width: 92px;
    overflow: hidden;
    text-overflow: ellipsis;
    font-size: 13px;
    white-space: nowrap;
  }
}
</style>
