<template>
  <el-dropdown trigger="click" placement="bottom-end" @command="handleLocaleChange">
    <Languages class="toolBar-icon" :size="20" :title="t('core.header.language')" />
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
import { Languages } from "@lucide/vue";
import { SUPPORTED_LOCALES, type SupportedLocale, useLocaleStore } from "@/locales";

const { locale, setLocale, t } = useLocaleStore();
const localeOptions = computed(() => {
  return SUPPORTED_LOCALES.map(value => ({ value, label: t(`common.language.${value}`) }));
});

/** 切换管理端语言并刷新动态本地化数据。 */
function handleLocaleChange(value: SupportedLocale) {
  void setLocale(value);
}
</script>
