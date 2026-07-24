<!-- 📚📚📚 Pro-Table 文档: https://juejin.cn/post/7166068828202336263 -->

<template>
  <!-- 查询表单 -->
  <SearchForm
    v-if="hasSearchColumns && isShowSearch"
    :search="_search"
    :reset="_reset"
    :columns="searchColumns"
    :search-param="searchParam"
    :search-col="searchCol"
  />

  <!-- 表格主体 -->
  <div class="card table-main">
    <!-- 表格头部 操作按钮 -->
    <div v-if="showTableHeader" class="table-header">
      <div class="header-button-lf">
        <slot name="tableHeader" :selected-list="selectedList" :selected-list-ids="selectedListIds" :is-selected="isSelected">
          <el-button
            v-for="action in visibleHeaderActions"
            :key="action.label"
            :type="action.type ?? 'primary'"
            :icon="normalizeActionIcon(action.icon)"
            :disabled="getHeaderActionDisabled(action)"
            @click="handleHeaderActionClick(action)"
          >
            {{ action.label }}
          </el-button>
        </slot>
      </div>
      <div v-if="showToolButtonArea" class="header-button-ri">
        <slot name="toolButton">
          <el-tooltip v-if="showTreeToggleButton" :content="isTreeExpanded ? '折叠全部' : '展开全部'" placement="top">
            <el-button class="tool-button" :icon="isTreeExpanded ? Fold : Expand" circle @click="toggleTreeExpand" />
          </el-tooltip>
          <el-tooltip v-if="showToolButton('refresh')" content="刷新数据" placement="top">
            <el-button class="tool-button" :icon="Refresh" circle @click="handleToolRefresh" />
          </el-tooltip>
          <el-tooltip v-if="showToolButton('setting') && columns.length" content="列设置" placement="top">
            <el-button class="tool-button" :icon="Operation" circle @click="openColSetting" />
          </el-tooltip>
          <el-tooltip
            v-if="showToolButton('search') && searchColumns?.length"
            :content="isShowSearch ? '隐藏搜索' : '显示搜索'"
            placement="top"
          >
            <el-button class="tool-button" :icon="Search" circle @click="isShowSearch = !isShowSearch" />
          </el-tooltip>
        </slot>
      </div>
    </div>
    <!-- 表格主体 -->
    <el-table
      ref="tableRef"
      v-loading="loading"
      v-bind="$attrs"
      :id="uuid"
      :data="processTableData"
      :border="border"
      :row-key="rowKey"
      :expand-row-keys="expandRowKeys"
      @selection-change="selectionChange"
    >
      <!-- 默认插槽 -->
      <slot />
      <template v-for="item in tableColumns" :key="item">
        <!-- selection || radio || index || expand || sort -->
        <el-table-column
          v-if="item.type && columnTypes.includes(item.type)"
          v-bind="item"
          :align="resolveColumnAlign(item)"
          :reserve-selection="item.type == 'selection'"
        >
          <template #default="scope">
            <!-- expand -->
            <template v-if="item.type == 'expand'">
              <component :is="item.render" v-bind="scope" v-if="item.render" />
              <slot v-else :name="item.type" v-bind="scope" />
            </template>
            <!-- radio -->
            <el-radio v-if="item.type == 'radio'" v-model="radio" :value="scope.row[rowKey]">
              <i></i>
            </el-radio>
            <!-- sort -->
            <el-tag v-if="item.type == 'sort'" class="move">
              <el-icon> <DCaret /></el-icon>
            </el-tag>
          </template>
        </el-table-column>
        <!-- other -->
        <TableColumn v-else :column="item" :resolve-align="resolveColumnAlign">
          <template v-for="slot in Object.keys($slots)" #[slot]="scope">
            <slot :name="slot" v-bind="scope" />
          </template>
        </TableColumn>
      </template>
      <!-- 插入表格最后一行之后的插槽 -->
      <template #append>
        <slot name="append" />
      </template>
      <!-- 无数据 -->
      <template #empty>
        <div class="table-empty">
          <slot name="empty">
            <img src="@/assets/images/notData.png" alt="notData" />
            <div>暂无数据</div>
          </slot>
        </div>
      </template>
    </el-table>
    <!-- 分页组件 -->
    <slot name="pagination">
      <Pagination
        v-if="pagination"
        :pageable="pageable"
        :handle-size-change="handleSizeChange"
        :handle-current-change="handleCurrentChange"
      />
    </slot>
  </div>
  <!-- 列设置 -->
  <ColSetting v-if="toolButton" ref="colRef" v-model:col-setting="colSetting" />
</template>

<script setup lang="ts" name="ProTable">
import { ref, watch, provide, onMounted, unref, computed, nextTick, useAttrs, useSlots, isProxy, markRaw, toRaw } from "vue";
import { ElTable } from "element-plus";
import { useTable } from "@/hooks/useTable";
import { useSelection } from "@/hooks/useSelection";
import { BreakPoint } from "@/components/Grid/interface";
import { ColumnProps, EnumProps, HeaderActionProps, HeaderActionScope, TypeProps } from "@/components/ProTable/interface";
import { Expand, Fold, Refresh, Operation, Search } from "@element-plus/icons-vue";
import { generateUUID, handleProp } from "@/utils";
import { resolveTableColumnAlign } from "@/utils/proTable";
import SearchForm from "@/components/SearchForm/index.vue";
import Pagination from "./components/Pagination.vue";
import ColSetting from "./components/ColSetting.vue";
import TableColumn from "./components/TableColumn.vue";
import Sortable from "sortablejs";

/** ProTable 组件属性。 */
export interface ProTableProps {
  columns: ColumnProps[]; // 列配置项  ==> 必传
  headerActions?: HeaderActionProps[]; // 顶部操作按钮配置 ==> 非必传
  data?: any[]; // 静态 table data 数据，若存在则不会使用 requestApi 返回的 data ==> 非必传
  requestApi?: (params: any) => Promise<any>; // 请求表格数据的 api ==> 非必传
  requestAuto?: boolean; // 是否自动执行请求 api ==> 非必传（默认为true）
  requestError?: (params: any) => void; // 表格 api 请求错误监听 ==> 非必传
  dataCallback?: (data: any) => any; // 返回数据的回调函数，可以对数据进行处理 ==> 非必传
  title?: string; // 表格标题 ==> 非必传
  pagination?: boolean; // 是否需要分页组件 ==> 非必传（默认为true）
  initParam?: any; // 初始化请求参数 ==> 非必传（默认为{}）
  border?: boolean; // 是否带有纵向边框 ==> 非必传（默认为true）
  toolButton?: ("refresh" | "setting" | "search")[] | boolean; // 是否显示表格功能按钮 ==> 非必传（默认为true）
  rowKey?: string; // 行数据的 Key，用来优化 Table 的渲染，当表格数据多选时，所指定的 id ==> 非必传（默认为 id）
  restoreSelectedRowKeys?: Array<string | number>; // 数据加载后需要恢复勾选的行 Key，仅补充选择状态，不会清空当前选择 ==> 非必传
  searchCol?: number | Record<BreakPoint, number>; // 表格搜索项 每列占比配置 ==> 非必传 { xs: 1, sm: 2, md: 2, lg: 3, xl: 4 }
}

// 接受父组件参数，配置默认值
const props = withDefaults(defineProps<ProTableProps>(), {
  columns: () => [],
  headerActions: () => [],
  requestAuto: true,
  pagination: true,
  initParam: {},
  border: true,
  toolButton: true,
  rowKey: "id",
  restoreSelectedRowKeys: () => [],
  searchCol: () => ({ xs: 1, sm: 2, md: 2, lg: 3, xl: 4 })
});

// table 实例
const tableRef = ref<InstanceType<typeof ElTable>>();
const attrs = useAttrs();
const slots = useSlots();

// 生成组件唯一id
const uuid = ref("id-" + generateUUID());

// column 列类型
const columnTypes: TypeProps[] = ["selection", "radio", "index", "expand", "sort"];

// 是否显示搜索模块
const isShowSearch = ref(true);
const isTreeExpanded = ref(false);
const expandRowKeys = ref<string[]>([]);

// 判断当前表格是否为树形表格
const treeProps = computed(() => {
  return (attrs.treeProps ?? attrs["tree-props"]) as Record<string, any> | undefined;
});

const showTreeToggleButton = computed(() => !!treeProps.value && !attrs.lazy);

/**
 * 透传给 Element Plus 前移除图标组件上的响应式代理，避免 Vue 对组件对象发出性能告警。
 */
const normalizeActionIcon = (icon: unknown): any => {
  if (!icon || (typeof icon !== "object" && typeof icon !== "function")) return icon;
  const rawIcon = isProxy(icon) ? toRaw(icon) : icon;
  return typeof rawIcon === "object" ? markRaw(rawIcon) : rawIcon;
};

// 控制 ToolButton 显示
const showToolButton = (key: "refresh" | "setting" | "search") => {
  return Array.isArray(props.toolButton) ? props.toolButton.includes(key) : props.toolButton;
};

const showToolButtonArea = computed(() => Boolean(props.toolButton || slots.toolButton));
const hasRefreshListener = computed(() => Boolean(attrs.onRefresh));

// 单选值
const radio = ref("");

// 表格多选 Hooks
const { selectionChange, selectedList, selectedListIds, isSelected } = useSelection(props.rowKey);

/**
 * 构建顶部按钮回调上下文，统一透出勾选状态给业务层。
 */
const headerActionScope = computed<HeaderActionScope>(() => ({
  selectedList: selectedList.value,
  selectedListIds: selectedListIds.value,
  isSelected: isSelected.value
}));

// 表格操作 Hooks
const { loading, tableData, pageable, searchParam, searchInitParam, getTableList, search, reset, handleSizeChange, handleCurrentChange } =
  useTable(props.requestApi, props.initParam, props.pagination, props.dataCallback, props.requestError);

// 清空选中数据列表
const clearSelection = () => tableRef.value!.clearSelection();

/** 返回当前表格分页总数，供父组件读取统计数量。 */
const getTotal = () => Number(pageable.value.total || 0);

// 初始化表格数据 && 拖拽排序
onMounted(() => {
  if (tableColumns.value.some(item => item.type === "sort")) {
    nextTick(() => {
      dragSort();
    });
  }
  props.requestAuto && getTableList();
  props.data && (pageable.value.total = props.data.length);
});

// 处理表格数据
const processTableData = computed(() => {
  if (!props.data) return tableData.value;
  if (!props.pagination) return props.data;
  return props.data.slice(
    (pageable.value.page_num - 1) * pageable.value.page_size,
    pageable.value.page_size * pageable.value.page_num
  );
});

/** 解析 ProTable 列对齐方式，随当前页数据类型变化。 */
const resolveColumnAlign = (column: ColumnProps) => resolveTableColumnAlign(column, processTableData.value);

/** 数据加载完成后按传入行 Key 恢复多选状态，用于页面重建后的选择态续接。 */
function restoreSelectedRows() {
  if (!tableRef.value || !props.restoreSelectedRowKeys.length) return;
  const selectedKeySet = new Set(props.restoreSelectedRowKeys.map(key => String(key)));
  processTableData.value.forEach(row => {
    if (selectedKeySet.has(String(row[props.rowKey]))) tableRef.value?.toggleRowSelection(row, true);
  });
  // 保留勾选状态时同步最新行对象，避免刷新后顶部操作继续读取旧数据。
  selectionChange(tableRef.value.getSelectionRows());
}

watch(
  [processTableData, () => props.restoreSelectedRowKeys, () => props.rowKey],
  () => {
    void nextTick(restoreSelectedRows);
  },
  { deep: true, immediate: true }
);

// 树形表格默认折叠，并在数据变化时根据当前展开状态同步展开项
watch(
  [processTableData, treeProps, () => props.rowKey, isTreeExpanded],
  () => {
    if (!treeProps.value) {
      expandRowKeys.value = [];
      return;
    }
    expandRowKeys.value = isTreeExpanded.value
      ? collectTreeRowKeys(processTableData.value, props.rowKey, treeProps.value.children ?? "children")
      : [];
  },
  { deep: true, immediate: true }
);

// 监听页面 initParam 改化，重新获取表格数据
watch(() => props.initParam, getTableList, { deep: true });

// 接收 columns 并设置为响应式
const tableColumns = ref<ColumnProps[]>(props.columns as ColumnProps[]);

// 扁平化 columns
const flatColumns = computed(() => flatColumnsFunc(tableColumns.value as any) as ColumnProps[]);

// 定义 enumMap 存储 enum 值（避免异步请求无法格式化单元格内容 || 无法填充搜索下拉选择）
const enumMap = ref(new Map<string, { [key: string]: any }[]>());

/** 更新枚举缓存时替换 Map 引用，确保搜索下拉能响应异步枚举结果。 */
const setEnumMapValue = (prop: string | undefined, data: { [key: string]: any }[]) => {
  if (!prop) return;
  const nextEnumMap = new Map(enumMap.value);
  nextEnumMap.set(prop, data);
  enumMap.value = nextEnumMap;
};

/**
 * 根据状态开关配置自动生成搜索下拉枚举，避免状态列只配置开关后搜索框无选项。
 */
const buildStatusEnum = (column: ColumnProps): EnumProps[] => {
  if (column.enum || column.dictCode || column.cellType !== "status" || column.search?.el !== "select" || !column.statusProps) return [];
  return [
    { label: column.statusProps.activeText ?? "启用", value: column.statusProps.activeValue },
    { label: column.statusProps.inactiveText ?? "禁用", value: column.statusProps.inactiveValue }
  ];
};

/** 初始化指定列的枚举缓存，支持静态枚举和异步枚举。 */
const setEnumMap = async ({ prop, enum: enumValue }: ColumnProps) => {
  if (!prop || !enumValue) return;

  // 如果当前 enumMap 存在相同的值 return
  if (enumMap.value.has(prop) && (typeof enumValue === "function" || enumMap.value.get(prop) === enumValue)) return;

  // 当前 enum 为静态数据，则直接存储到 enumMap
  if (typeof enumValue !== "function") {
    setEnumMapValue(prop, unref(enumValue!));
    return;
  }

  // 为了防止接口执行慢，而存储慢，导致重复请求，所以预先存储为[]，接口返回后再二次存储
  setEnumMapValue(prop, []);

  // 当前 enum 为后台数据需要请求数据，则调用该请求接口，并存储到 enumMap
  try {
    const response = await enumValue();
    const data = Array.isArray(response) ? response : (response?.data ?? response?.list ?? []);
    setEnumMapValue(prop, data);
  } catch {
    // 请求层负责展示接口错误；删除占位缓存，允许列配置再次同步时重试。
    const nextEnumMap = new Map(enumMap.value);
    nextEnumMap.delete(prop);
    enumMap.value = nextEnumMap;
  }
};

/**
 * 同步列枚举配置，状态开关列会自动补齐搜索枚举。
 */
const syncColumnEnumMap = async (column: ColumnProps) => {
  const statusEnum = buildStatusEnum(column);
  if (statusEnum.length && column.prop) {
    setEnumMapValue(column.prop, statusEnum);
    return;
  }
  await setEnumMap(column);
};

// 注入 enumMap
provide("enumMap", enumMap);

// 扁平化 columns 的方法
const flatColumnsFunc = (columns: ColumnProps[], flatArr: ColumnProps[] = []) => {
  columns.forEach(col => {
    if (col._children?.length) flatArr.push(...flatColumnsFunc(col._children));
    flatArr.push(col);

    // column 添加默认 isShow && isSetting && isFilterEnum 属性值
    col.isShow = col.isShow ?? true;
    col.isSetting = col.isSetting ?? true;
    col.isFilterEnum = col.isFilterEnum ?? true;
  });
  return flatArr.filter(item => !item._children?.length);
};

// 监听扁平列配置并同步枚举，避免在 computed 中执行异步副作用导致搜索下拉不刷新。
watch(
  flatColumns,
  columns => {
    columns.forEach(column => {
      void syncColumnEnumMap(column);
    });
  },
  { deep: true, immediate: true }
);

// 过滤需要搜索的配置项 && 排序
const searchColumns = computed(() => {
  return flatColumns.value
    ?.filter(item => item.search?.el || item.search?.render)
    .sort((a, b) => a.search!.order! - b.search!.order!);
});

// 设置 搜索表单默认排序 && 搜索表单项的默认值
searchColumns.value?.forEach((column, index) => {
  column.search!.order = column.search?.order ?? index + 2;
  const key = column.search?.key ?? handleProp(column.prop!);
  const defaultValue = column.search?.defaultValue;
  if (defaultValue !== undefined && defaultValue !== null) {
    searchParam.value[key] = defaultValue;
    searchInitParam.value[key] = defaultValue;
  }
});

// 列设置 ==> 需要过滤掉不需要设置的列
const colRef = ref();
const colSetting = (tableColumns.value as ColumnProps[]).filter(item => {
  const { type, prop, isSetting } = item;
  return !columnTypes.includes(type!) && prop !== "operation" && isSetting;
});
/** 打开列设置弹窗。 */
const openColSetting = () => colRef.value.openColSetting();

/**
 * 解析顶部按钮透传参数，兼容静态对象与函数返回值。
 */
const resolveHeaderActionParams = (params: HeaderActionProps["params"]) => {
  if (!params) return undefined;
  return typeof params === "function" ? params(headerActionScope.value) : params;
};

/**
 * 统一解析顶部按钮的显隐状态。
 */
const getHeaderActionHidden = (action: HeaderActionProps) => {
  return typeof action.hidden === "function" ? action.hidden(headerActionScope.value) : Boolean(action.hidden);
};

/**
 * 统一解析顶部按钮的禁用状态。
 */
const getHeaderActionDisabled = (action: HeaderActionProps) => {
  return typeof action.disabled === "function" ? action.disabled(headerActionScope.value) : Boolean(action.disabled);
};

/**
 * 过滤顶部可见按钮，避免模板层重复执行显隐判断。
 */
const visibleHeaderActions = computed(() => {
  return props.headerActions.filter(action => !getHeaderActionHidden(action));
});

const hasSearchColumns = computed(() => searchColumns.value.length > 0);

const showTableHeader = computed(() => {
  return visibleHeaderActions.value.length > 0 || showToolButtonArea.value || Boolean(slots.tableHeader);
});

/**
 * 执行顶部按钮回调，并透传当前选中数据与附加参数。
 */
const handleHeaderActionClick = (action: HeaderActionProps) => {
  action.onClick(headerActionScope.value, resolveHeaderActionParams(action.params));
};

// 定义 emit 事件
const emit = defineEmits<{
  search: [];
  reset: [];
  refresh: [];
  dragSort: [{ newIndex?: number; oldIndex?: number }];
}>();

/** 执行表格搜索并通知外部监听方。 */
const _search = () => {
  search();
  emit("search");
};

/** 重置表格搜索条件并通知外部监听方。 */
const _reset = () => {
  reset();
  emit("reset");
};

/**
 * 处理表格工具栏刷新操作。
 * 若业务页面监听 refresh 事件，则交由业务层执行自定义刷新；
 * 否则沿用 ProTable 默认的数据请求刷新逻辑。
 */
const handleToolRefresh = () => {
  if (hasRefreshListener.value) {
    emit("refresh");
    return;
  }
  getTableList();
};

// 切换树形表格展开状态
/** 切换树形表格展开或折叠状态。 */
const toggleTreeExpand = () => {
  isTreeExpanded.value = !isTreeExpanded.value;
};

// 递归收集树形表格的所有行 key
/** 递归收集树形表格的所有行 key。 */
const collectTreeRowKeys = (rows: Record<string, any>[], rowKey: string, childrenKey: string) => {
  const keys: string[] = [];

  /** 遍历当前层级并继续处理子节点。 */
  const travel = (list: Record<string, any>[]) => {
    list.forEach(item => {
      const key = item[rowKey];
      if (key !== undefined && key !== null && key !== "") {
        keys.push(String(key));
      }
      const children = item[childrenKey];
      if (Array.isArray(children) && children.length) {
        travel(children);
      }
    });
  };

  travel(rows);
  return keys;
};

// 表格拖拽排序
const dragSort = () => {
  const tbody = document.querySelector(`#${uuid.value} tbody`) as HTMLElement;
  if (!tbody) return;
  Sortable.create(tbody, {
    handle: ".move",
    animation: 300,
    onEnd({ newIndex, oldIndex }) {
      const [removedItem] = processTableData.value.splice(oldIndex!, 1);
      processTableData.value.splice(newIndex!, 0, removedItem);
      emit("dragSort", { newIndex, oldIndex });
    }
  });
};

// 暴露给父组件的参数和方法 (外部需要什么，都可以从这里暴露出去)
defineExpose({
  element: tableRef,
  loading,
  tableData: processTableData,
  radio,
  pageable,
  searchParam,
  searchInitParam,
  isSelected,
  selectedList,
  selectedListIds,

  // 下面为 function
  getTableList,
  search,
  reset,
  handleSizeChange,
  handleCurrentChange,
  clearSelection,
  getTotal,
  enumMap
});
</script>

<style scoped lang="scss">
.tool-button {
  cursor: pointer;
}
.tool-button :deep(.el-icon) {
  cursor: pointer;
}
</style>
