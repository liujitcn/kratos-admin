<template>
  <div v-if="searchColumns.length" class="card table-search" :class="{ 'table-search--no-operation': !showOperation }">
    <el-form ref="formRef" :model="searchParam">
      <Grid ref="gridRef" :collapsed="collapsed" :gap="[20, 0]" :cols="searchCol">
        <GridItem v-for="(item, index) in searchColumns" :key="item.prop" v-bind="getResponsive(item)" :index="index">
          <el-form-item>
            <template #label>
              <el-space :size="4">
                <span>{{ `${item.search?.label ?? item.label}` }}</span>
                <el-tooltip v-if="item.search?.tooltip" effect="dark" :content="item.search?.tooltip" placement="top">
                  <i :class="'iconfont icon-yiwen'"></i>
                </el-tooltip>
              </el-space>
              <span>&nbsp;:</span>
            </template>
            <template v-if="resolveSearchSlotName(item)">
              <slot
                :name="resolveSearchSlotName(item)"
                :field="item"
                :model="searchParam"
                :column="item"
                :search-param="searchParam"
                :prop="item.search?.key ?? item.prop"
              />
            </template>
            <SearchFormItem v-else :column="item" :search-param="searchParam" />
          </el-form-item>
        </GridItem>
        <GridItem v-if="showOperation" suffix>
          <div class="operation">
            <el-button type="primary" :icon="Search" @click="search">{{ t("common.action.search") }}</el-button>
            <el-button :icon="Delete" @click="reset">{{ t("common.action.reset") }}</el-button>
            <el-button v-if="showCollapse" type="primary" link class="search-isOpen" @click="collapsed = !collapsed">
              {{ collapsed ? t("common.action.expand") : t("common.action.collapse") }}
              <el-icon class="el-icon--right">
                <component :is="collapsed ? ArrowDown : ArrowUp"></component>
              </el-icon>
            </el-button>
          </div>
        </GridItem>
      </Grid>
    </el-form>
  </div>
</template>
<script setup lang="ts" name="SearchForm">
import { computed, ref, useSlots } from "vue";
import { ColumnProps } from "@/components/ProTable/interface";
import { BreakPoint } from "@/components/Grid/interface";
import { Delete, Search, ArrowDown, ArrowUp } from "@element-plus/icons-vue";
import SearchFormItem from "./components/SearchFormItem.vue";
import Grid from "@/components/Grid/index.vue";
import GridItem from "@/components/Grid/components/GridItem.vue";
import { useLocaleStore } from "@/locales";

const { t } = useLocaleStore();

/** 搜索列配置，补充 ProForm 风格的插槽名称和字段后置位置。 */
type SearchColumn = ColumnProps & {
  /** 自定义搜索项插槽名称，未配置时使用字段 prop。 */
  slotName?: string;
  /** 将当前搜索项插入到指定字段后面。 */
  after?: string;
};

/** ProTable 搜索表单组件属性。 */
interface ProTableProps {
  columns?: SearchColumn[]; // 搜索配置列
  searchParam?: { [key: string]: any }; // 搜索参数
  searchCol: number | Record<BreakPoint, number>;
  search: (params: any) => void; // 搜索方法
  reset: (params: any) => void; // 重置方法
  showOperation?: boolean; // 是否展示搜索、重置等操作按钮
}

// 默认值
const props = withDefaults(defineProps<ProTableProps>(), {
  columns: () => [],
  searchParam: () => ({}),
  showOperation: true
});

const slots = useSlots();

/** 按 after 配置排列搜索项，未配置或找不到目标字段时保持原有顺序。 */
const searchColumns = computed<SearchColumn[]>(() => {
  const sourceColumns = [...(props.columns ?? [])];
  const positionedColumns = sourceColumns.filter(column => {
    const after = column.after;
    return Boolean(after && sourceColumns.some(item => item.prop === after));
  });
  const positionedSet = new Set(positionedColumns);
  const columnsByAnchor = new Map<string, SearchColumn[]>();
  for (const column of positionedColumns) {
    const anchor = column.after!;
    const columns = columnsByAnchor.get(anchor) ?? [];
    columns.push(column);
    columnsByAnchor.set(anchor, columns);
  }

  const result: SearchColumn[] = [];
  for (const column of sourceColumns) {
    if (positionedSet.has(column)) continue;
    result.push(column);
    const columns = column.prop ? columnsByAnchor.get(column.prop) : undefined;
    if (!columns) continue;
    result.push(...columns);
    columnsByAnchor.delete(column.prop!);
  }
  for (const columns of columnsByAnchor.values()) result.push(...columns);
  return result;
});

/** 按 ProForm 的 slotName 优先、字段 prop 回退规则解析搜索项插槽。 */
function resolveSearchSlotName(item: SearchColumn): string | undefined {
  const slotName = item.slotName ?? item.prop;
  return slotName && slots[slotName] ? slotName : undefined;
}

// 获取响应式设置
const getResponsive = (item: ColumnProps) => {
  return {
    span: item.search?.span,
    offset: item.search?.offset ?? 0,
    xs: item.search?.xs,
    sm: item.search?.sm,
    md: item.search?.md,
    lg: item.search?.lg,
    xl: item.search?.xl
  };
};

// 是否默认折叠搜索项
const collapsed = ref(true);

// 获取响应式断点
const gridRef = ref();
const breakPoint = computed<BreakPoint>(() => gridRef.value?.breakPoint);

// 判断是否显示 展开/合并 按钮
const showCollapse = computed(() => {
  let show = false;
  searchColumns.value.reduce((prev, current) => {
    prev +=
      (current.search?.[breakPoint.value]?.span ?? current.search?.span ?? 1) +
      (current.search?.[breakPoint.value]?.offset ?? current.search?.offset ?? 0);
    if (typeof props.searchCol !== "number") {
      if (prev >= props.searchCol[breakPoint.value]) show = true;
    } else {
      if (prev >= props.searchCol) show = true;
    }
    return prev;
  }, 0);
  return show;
});
</script>

<style scoped lang="scss">
.table-search--no-operation {
  :deep(.el-form-item) {
    margin-bottom: 0;
  }
}
</style>
