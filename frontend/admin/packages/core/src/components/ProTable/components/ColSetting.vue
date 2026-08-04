<template>
  <!-- 列设置 -->
  <el-drawer v-model="drawerVisible" :title="t('core.table.column_setting')" size="450px">
    <div class="table-main">
      <el-table :data="colSetting" :border="true" row-key="prop" default-expand-all :tree-props="{ children: '_children' }">
        <el-table-column prop="label" align="left" :label="t('core.table.column_name')" />
        <el-table-column v-slot="scope" prop="isShow" align="center" :label="t('core.table.show')">
          <el-switch v-model="scope.row.isShow"></el-switch>
        </el-table-column>
        <el-table-column v-slot="scope" prop="sortable" align="center" :label="t('core.table.sort')">
          <el-switch v-model="scope.row.sortable"></el-switch>
        </el-table-column>
        <template #empty>
          <div class="table-empty">
            <img src="@/assets/images/notData.png" alt="notData" />
            <div>{{ t("core.table.no_configurable_columns") }}</div>
          </div>
        </template>
      </el-table>
    </div>
  </el-drawer>
</template>

<script setup lang="ts" name="ColSetting">
import { ref } from "vue";
import { ColumnProps } from "@/components/ProTable/interface";
import { useLocaleStore } from "@/locales";

defineProps<{ colSetting: ColumnProps[] }>();

const drawerVisible = ref<boolean>(false);
const { t } = useLocaleStore();

const openColSetting = () => {
  drawerVisible.value = true;
};

defineExpose({
  openColSetting
});
</script>

<style scoped lang="scss">
.cursor-move {
  cursor: move;
}
</style>
