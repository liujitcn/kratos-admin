<template>
  <div class="card filter">
    <h4 v-if="title" class="title sle">
      {{ title }}
    </h4>
    <div class="search">
      <el-input v-model="filterText" placeholder="输入关键字进行过滤" clearable />
      <el-dropdown trigger="click">
        <el-icon size="20"><More /></el-icon>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item v-if="!multiple" @click="resetSelected">重置选择</el-dropdown-item>
            <el-dropdown-item @click="toggleTreeNodes(true)">展开全部</el-dropdown-item>
            <el-dropdown-item @click="toggleTreeNodes(false)">折叠全部</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
    <el-scrollbar :style="{ height: title ? `calc(100% - 95px)` : `calc(100% - 56px)` }">
      <el-tree
        ref="treeRef"
        :default-expand-all="!lazy"
        :node-key="id"
        :data="multiple || !showAll ? treeData : treeAllData"
        :show-checkbox="multiple"
        :check-strictly="false"
        :current-node-key="!multiple ? selected : undefined"
        :highlight-current="!multiple"
        :expand-on-click-node="false"
        :check-on-click-node="multiple"
        :props="defaultProps"
        :lazy="lazy"
        :load="lazy ? loadTreeNode : undefined"
        :filter-node-method="filterNode"
        :default-checked-keys="multiple ? selected : []"
        @node-click="handleNodeClick"
        @check="handleCheckChange"
      >
        <template #default="scope">
          <span class="el-tree-node__label">
            <slot :row="scope">
              {{ scope.node.label }}
            </slot>
          </span>
        </template>
      </el-tree>
    </el-scrollbar>
  </div>
</template>

<script setup lang="ts" name="TreeFilter">
import { ref, watch, onBeforeMount, nextTick } from "vue";
import { ElTree } from "element-plus";

// 接收父组件参数并设置默认值
interface TreeFilterProps {
  requestApi?: (data?: any) => Promise<any>; // 请求分类数据的 api ==> 非必传
  data?: { [key: string]: any }[]; // 分类数据，如果有分类数据，则不会执行 api 请求 ==> 非必传
  title?: string; // treeFilter 标题 ==> 非必传
  id?: string; // 选择的id ==> 非必传，默认为 “id”
  label?: string; // 显示的label ==> 非必传，默认为 “label”
  multiple?: boolean; // 是否为多选 ==> 非必传，默认为 false
  defaultValue?: any; // 默认选中的值 ==> 非必传
  showAll?: boolean; // 单选模式下是否展示“全部”节点 ==> 非必传，默认为 true
  lazy?: boolean; // 是否按节点懒加载子节点
  parentKey?: string; // 懒加载请求使用的父节点字段
}
const props = withDefaults(defineProps<TreeFilterProps>(), {
  id: "id",
  label: "label",
  multiple: false,
  showAll: true,
  lazy: false,
  parentKey: "parent_id"
});

const defaultProps = {
  children: "children",
  label: props.label
};

const treeRef = ref<InstanceType<typeof ElTree>>();
const treeData = ref<{ [key: string]: any }[]>([]);
const treeAllData = ref<{ [key: string]: any }[]>([]);

const selected = ref();
const setSelected = () => {
  if (props.multiple) selected.value = Array.isArray(props.defaultValue) ? props.defaultValue : [props.defaultValue];
  else selected.value = typeof props.defaultValue === "string" && props.defaultValue ? props.defaultValue : undefined;
};

const normalizeLazyNodes = (nodes: { [key: string]: any }[] = []) =>
  nodes.map(node => ({
    ...node,
    isLeaf: !node.has_children
  }));

const updateTreeData = (data: { [key: string]: any }[]) => {
  treeData.value = props.lazy ? normalizeLazyNodes(data) : data;
  const allNode = { id: "", [props.label]: "全部", isLeaf: true };
  treeAllData.value = props.showAll ? [allNode, ...treeData.value] : treeData.value;
};

onBeforeMount(async () => {
  setSelected();
  if (props.requestApi) {
    const requestParams = props.lazy ? { [props.parentKey]: 0, lazy: true } : undefined;
    const { data } = await props.requestApi(requestParams);
    updateTreeData(data ?? []);
  }
});

const loadTreeNode = async (node: { level: number; data?: { [key: string]: any } }, resolve: (data: { [key: string]: any }[]) => void) => {
  if (!props.requestApi) {
    resolve([]);
    return;
  }
  const parentId = node.level === 0 ? 0 : node.data?.[props.id];
  try {
    const { data } = await props.requestApi({ [props.parentKey]: parentId, lazy: true });
    resolve(normalizeLazyNodes(data ?? []));
  } catch {
    resolve([]);
  }
};

// 使用 nextTick 防止打包后赋值不生效，开发环境是正常的
watch(
  () => props.defaultValue,
  () => nextTick(() => setSelected()),
  { deep: true, immediate: true }
);

watch(
  () => props.data,
  () => {
    if (props.data?.length) {
      updateTreeData(props.data);
    }
  },
  { deep: true, immediate: true }
);

const filterText = ref("");
watch(filterText, val => {
  treeRef.value!.filter(val);
});

// 过滤
const filterNode = (value: string, data: { [key: string]: any }, node: any) => {
  if (!value) return true;
  let parentNode = node.parent,
    labels = [node.label],
    level = 1;
  while (level < node.level) {
    labels = [...labels, parentNode.label];
    parentNode = parentNode.parent;
    level++;
  }
  return labels.some(label => label.indexOf(value) !== -1);
};

// 切换树节点的展开或折叠状态
const toggleTreeNodes = (isExpand: boolean) => {
  let nodes = treeRef.value?.store.nodesMap;
  if (!nodes) return;
  for (const node in nodes) {
    if (nodes.hasOwnProperty(node)) {
      nodes[node].expanded = isExpand;
    }
  }
};

// emit
const emit = defineEmits<{
  change: [value: any];
}>();

// 单选
const handleNodeClick = (data: { [key: string]: any }) => {
  if (props.multiple) return;
  selected.value = data[props.id];
  emit("change", data[props.id]);
};

// 多选
const handleCheckChange = () => {
  emit("change", treeRef.value?.getCheckedKeys());
};

// 重置当前单选树筛选
const resetSelected = () => {
  if (props.multiple) return;
  selected.value = undefined;
  nextTick(() => {
    treeRef.value?.setCurrentKey(null);
  });
  emit("change", "");
};

// 暴露给父组件使用
defineExpose({ treeData, treeAllData, treeRef });
</script>

<style scoped lang="scss">
@use "./index.scss" as *;
</style>
