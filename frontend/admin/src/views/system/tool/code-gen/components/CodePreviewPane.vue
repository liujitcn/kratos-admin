<template>
  <el-table v-loading="loading" :data="files" row-key="path" border height="100%">
    <el-table-column type="expand">
      <template #default="{ row }">
        <pre class="code-preview">{{ row.content || row.message }}</pre>
      </template>
    </el-table-column>
    <el-table-column prop="path" label="文件路径" min-width="320" show-overflow-tooltip />
    <el-table-column prop="action" label="动作" width="100" />
    <el-table-column label="状态" width="100" align="center">
      <template #default="{ row }">
        <el-tag :type="row.exists ? 'info' : 'success'" effect="plain">{{ row.exists ? "已存在" : "待创建" }}</el-tag>
      </template>
    </el-table-column>
    <el-table-column prop="message" label="说明" min-width="220" show-overflow-tooltip />
  </el-table>
</template>

<script setup lang="ts">
import type { CodeGenPreviewFile } from "@/rpc/system/admin/v1/code_gen";

/** 代码预览面板属性。 */
defineProps<{
  files: CodeGenPreviewFile[];
  loading: boolean;
}>();
</script>

<style scoped lang="scss">
.code-preview {
  max-height: 56vh;
  padding: 14px;
  margin: 0;
  overflow: auto;
  font-family: var(--el-font-family-monospace, monospace);
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-primary);
  white-space: pre;
  background: var(--el-fill-color-light);
}
</style>
