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
import { type SupportedLocale, useLocaleStore } from "@/locales";

const { locale, setLocale, languageOptions, t } = useLocaleStore();
const localeOptions = computed(() => {
  return languageOptions.value.map(item => ({ value: item.language_code, label: item.language_name }));
});

/** 切换管理端语言并刷新动态本地化数据。 */
function handleLocaleChange(value: SupportedLocale) {
  void setLocale(value);
}
</script>
