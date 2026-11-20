<template>
  <div v-if="total > 0" class="docs-pagination">
    <div class="docs-pagination__summary" role="status">
      <span>共 {{ total }} 项 · 第 {{ page }} / {{ Math.max(1, Math.ceil(total / pageSize)) }} 页</span>
    </div>
    <el-pagination
      :current-page="page"
      :page-size="pageSize"
      :total="total"
      :pager-count="5"
      background
      layout="prev, pager, next"
      @current-change="emit('update:page', $event)"
    />
  </div>
</template>

<script setup lang="ts">
/** 组件 API 文档分页参数。 */
interface DocsPaginationProps {
  total: number;
  page: number;
  pageSize: number;
}

defineProps<DocsPaginationProps>();
const emit = defineEmits<{ "update:page": [page: number] }>();
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
    font-size: var(--el-font-size-base);
    color: var(--admin-page-text-secondary);
  }
}
</style>
