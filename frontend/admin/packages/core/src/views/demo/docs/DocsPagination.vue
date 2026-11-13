<template>
  <div v-if="total > 0" class="docs-pagination">
    <div class="docs-pagination__summary" role="status">
      <span>共 {{ total }} 项 · 第 {{ page }} / {{ Math.max(1, Math.ceil(total / pageSize)) }} 页</span>
      <el-select
        v-if="sizes.length"
        :model-value="pageSize"
        size="small"
        aria-label="每页组件数量"
        @change="emit('update:pageSize', $event)"
      >
        <el-option v-for="size in sizes" :key="size" :value="size" :label="`${size} 个 / 页`" />
      </el-select>
    </div>
    <el-pagination
      :current-page="page"
      :page-size="pageSize"
      :total="total"
      :pager-count="5"
      size="small"
      background
      layout="prev, pager, next"
      @current-change="emit('update:page', $event)"
    />
  </div>
</template>

<script setup lang="ts">
/** 文档分页参数，总览可切换每页数量，API 使用固定数量。 */
interface DocsPaginationProps {
  total: number;
  page: number;
  pageSize: number;
  sizes?: number[];
}

withDefaults(defineProps<DocsPaginationProps>(), { sizes: () => [] });
const emit = defineEmits<{ "update:page": [page: number]; "update:pageSize": [size: number] }>();
</script>

<style scoped lang="scss">
.docs-pagination {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  padding-top: 16px;
  margin-top: 20px;
  border-top: 1px solid var(--admin-page-divider);

  &__summary {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
    font-size: 12px;
    color: var(--admin-page-text-secondary);

    .el-select {
      width: 94px;
    }
  }
}
</style>
