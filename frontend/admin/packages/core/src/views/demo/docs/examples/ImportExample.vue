<template>
  <div v-if="active">
    <el-button type="primary" @click="openImport">打开导入弹窗</el-button>
    <p class="example-tip">下载本地示例模板后选择 Excel 文件，可体验文件校验和覆盖选项。</p>
    <p v-if="result" class="example-tip" role="status">{{ result }}</p>
    <ImportExcel ref="importRef" />
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import ImportExcel from "@/components/ImportExcel/index.vue";
import { excelTemplate } from "./excelTemplate";
import { useDemoActive } from "./useDemoActive";

defineProps<{ componentId: string }>();
const active = useDemoActive();
const importRef = ref<InstanceType<typeof ImportExcel>>();
const result = ref("");

/** 使用本地模板和回调打开实际导入组件。 */
function openImport() {
  result.value = "";
  importRef.value?.acceptParams({
    title: "项目",
    tempApi: async () => excelTemplate,
    importApi: async (form: FormData) => {
      const file = form.get("file") as File;
      result.value = `已选择 ${file.name}，${form.get("isCover") === "true" ? "覆盖" : "保留"}已有数据；示例未提交文件。`;
    }
  });
}
</script>

<style scoped lang="scss">
.example-tip {
  margin: 16px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
</style>
