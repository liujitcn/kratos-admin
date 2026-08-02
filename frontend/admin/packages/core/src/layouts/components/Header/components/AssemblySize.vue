<template>
  <el-tooltip effect="dark" :content="t('core.header.sizeSelect')" placement="bottom" :show-after="200">
    <el-dropdown trigger="click" @command="setAssemblySize">
      <i :class="'iconfont icon-contentright'" class="toolBar-icon"></i>
      <template #dropdown>
        <el-dropdown-menu>
          <el-dropdown-item
            v-for="item in assemblySizeList"
            :key="item.value"
            :command="item.value"
            :disabled="assemblySize === item.value"
          >
            {{ item.label }}
          </el-dropdown-item>
        </el-dropdown-menu>
      </template>
    </el-dropdown>
  </el-tooltip>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useGlobalStore } from "@/stores/modules/global";
import { AssemblySizeType } from "@/stores/interface";
import { useLocaleStore } from "@/locales";

const globalStore = useGlobalStore();
const assemblySize = computed(() => globalStore.assemblySize);
const { t } = useLocaleStore();

const assemblySizeList = computed(() => [
  { label: t("core.header.size.default"), value: "default" },
  { label: t("core.header.size.large"), value: "large" },
  { label: t("core.header.size.small"), value: "small" }
]);

const setAssemblySize = (item: AssemblySizeType) => {
  if (assemblySize.value === item) return;
  globalStore.setGlobalState("assemblySize", item);
};
</script>
