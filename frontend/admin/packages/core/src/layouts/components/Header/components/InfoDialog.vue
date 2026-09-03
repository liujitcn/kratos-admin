<template>
  <ProDialog v-model="dialogVisible" :title="t('core.layout.info')" width="500px" draggable>
    <span>{{ t("core.layout.info_content") }}</span>
    <template #footer>
      <span class="dialog-footer">
        <el-button @click="dialogVisible = false">{{ t("common.action.cancel") }}</el-button>
        <el-button type="primary" @click="dialogVisible = false">{{ t("common.action.confirm") }}</el-button>
      </span>
    </template>
  </ProDialog>
</template>

<script setup lang="ts">
import { useLocaleStore } from "@/locales";
import ProDialog from "@/components/Dialog/ProDialog.vue";

const { t } = useLocaleStore();
import { ref } from "vue";

const dialogVisible = ref(false);
const openDialog = () => {
  dialogVisible.value = true;
};

defineExpose({ openDialog });
</script>
