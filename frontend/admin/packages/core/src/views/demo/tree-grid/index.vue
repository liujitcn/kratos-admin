<!--
  Demo 页面三：树形表格（行内树 + 静态 data 直灌）。

  与“左树右表”页的区别：
  - 左树右表 = 左侧独立的树控件过滤右侧平铺表格；
  - 本页 = 表格行自身带层级（children 嵌套），点行首箭头逐级展开。

  本页演示要点：
  - ProTable 传静态 data（不传 requestApi 即不走请求流），pagination 关闭后
    processTableData 会原样返回 data，树形结构不会被分页截断；
  - tree-props / default-expand-all / indent / expand-row-keys 等属性通过
    v-bind="$attrs" 直接透传给 el-table（见 ProTable/index.vue 表格主体）；
  - 状态列改用 tag: true + enum 展示（区别于状态开关列和纯标签列）；
  - 操作列 params 按行动态生成，演示“新增子级”按钮把父行 ID 带进表单。

  参照真实页面：system/base/dept/index.vue、system/base/menu/index.vue。
-->
<template>
  <div class="table-box">
    <ProTable
      :key="expandAll ? 'expand' : 'collapse'"
      ref="proTable"
      row-key="id"
      :indent="20"
      :columns="columns"
      :header-actions="headerActions"
      :data="catalogList"
      :pagination="false"
      :default-expand-all="expandAll"
      :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
    />

    <DemoFormDialog ref="formDialogRef" @submit="handleFormSubmit" />
    <DemoDetailDrawer v-model="detailVisible" :detail="detailModel" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, Plus, View } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable";
import { t } from "@/locales";

import DemoFormDialog from "../components/DemoFormDialog.vue";
import DemoDetailDrawer from "../components/DemoDetailDrawer.vue";
import type { DemoCatalog, DemoDetailModel } from "../demo-data";
import { STATUS_DISABLE, STATUS_ENABLE, demoCatalogTree } from "../demo-data";

defineOptions({
  name: "DemoTreeGrid",
  inheritAttrs: false
});

const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof DemoFormDialog>>();
const detailVisible = ref(false);
const detailModel = ref<DemoDetailModel | null>(null);

/**
 * 目录树本地数据副本：新增子级/删除在这份副本上递归操作。
 * 真实项目由后端返回树形列表，前端只负责展示。
 */
const catalogList = ref<DemoCatalog[]>(structuredClone(demoCatalogTree));

/** 是否默认展开全部节点：工具按钮切换，透传给 el-table 的 default-expand-all。 */
const expandAll = ref(false);

/** 状态枚举选项：tag 形态展示（success/info 两色）。 */
const statusTagOptions = [
  { label: t("common.status.enabled"), value: STATUS_ENABLE, tagType: "success" },
  { label: t("common.status.disabled"), value: STATUS_DISABLE, tagType: "info" }
];

/**
 * 表格列配置：首列 name 即树形展开列（el-table 会自动在该列前加展开箭头）。
 */
const columns = computed<ColumnProps[]>(() => [
  { prop: "name", label: t("core.demo.grid.field.name"), minWidth: 260 },
  { prop: "code", label: t("core.demo.grid.field.code"), minWidth: 160 },
  { prop: "owner", label: t("core.demo.table.field.owner"), minWidth: 100 },
  {
    // 状态列：tag: true + enum，按枚举 tagType 渲染标签
    prop: "status",
    label: t("common.field.status"),
    minWidth: 90,
    align: "center",
    tag: true,
    enum: statusTagOptions
  },
  { prop: "doc_count", label: t("core.demo.grid.field.doc_count"), minWidth: 100, align: "right" },
  {
    // 操作列：params 动态携带父行 ID，演示“新增子级”
    prop: "operation",
    label: t("common.field.operation"),
    align: "center",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.detail"),
        type: "primary",
        link: true,
        icon: View,
        onClick: scope => handleOpenDetail(scope.row as DemoCatalog)
      },
      {
        label: t("core.demo.grid.action.create_child"),
        type: "success",
        link: true,
        icon: Plus,
        params: scope => ({ parentId: scope.row.id }),
        onClick: (scope, params) => handleCreateChild((scope.row as DemoCatalog).id ?? (params?.parentId as number))
      },
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        onClick: scope => handleEdit(scope.row as DemoCatalog)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        onClick: scope => handleDelete(scope.row as DemoCatalog)
      }
    ]
  }
]);

/** 顶部按钮：新增根级目录 + 展开状态切换。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    onClick: () => handleCreateChild(undefined)
  },
  {
    // default-expand-all 非响应式，切换后通过 :key 重建表格使其生效
    label: expandAll.value ? t("core.demo.grid.action.collapse_all") : t("core.demo.grid.action.expand_all"),
    type: "primary",
    onClick: () => {
      expandAll.value = !expandAll.value;
    }
  }
]);

/** 在静态树中定位节点：返回节点对象与其父容器数组。 */
function findNode(nodes: DemoCatalog[], id: number, parent: DemoCatalog[] | null): { node: DemoCatalog; siblings: DemoCatalog[] } | undefined {
  for (const node of nodes) {
    if (node.id === id) return { node, siblings: parent ?? nodes };
    const found = node.children ? findNode(node.children, id, node.children) : undefined;
    if (found) return found;
  }
  return undefined;
}

/** 追加计数器：保证静态演示新增的节点 ID 唯一。 */
let nextId = 900;

/** 当前待挂载的父节点 ID，undefined 表示新增根级目录。 */
const pendingParentId = ref<number>();

/**
 * 新增目录：记录待挂载父级后打开表单弹窗，
 * 提交回调 handleFormSubmit 里真正把节点插入本地树。
 */
function handleCreateChild(parentId?: number) {
  pendingParentId.value = parentId;
  formDialogRef.value?.open();
}

/**
 * 表单提交回调：把新目录插入本地树（根级或父级 children 末尾）。
 * 真实项目调用 CreateXxx 成功后重新拉取树形列表。
 */
function handleFormSubmit(data: Record<string, any>) {
  const node: DemoCatalog = {
    id: ++nextId,
    name: String(data.name || t("core.demo.tree.new_dir", { id: nextId })),
    code: `demo-${nextId}`,
    owner: data.visibility === "private" ? "--" : "Demo",
    status: data.enabled ? STATUS_ENABLE : STATUS_DISABLE,
    doc_count: Number(data.count) || 0
  };
  const parent = pendingParentId.value ? findNode(catalogList.value, pendingParentId.value, null)?.node : undefined;
  if (parent) {
    parent.children = [...(parent.children ?? []), node];
  } else {
    catalogList.value = [...catalogList.value, node];
  }
  pendingParentId.value = undefined;
}

/** 打开编辑弹窗：回填行数据。 */
function handleEdit(row: DemoCatalog) {
  formDialogRef.value?.open({ name: row.name, count: row.doc_count });
}

/**
 * 删除目录：存在子级时阻止删除并提示（与真实页面的引用检查行为一致），
 * 否则确认后从本地树中递归移除。
 */
function handleDelete(row: DemoCatalog) {
  if (row.children?.length) {
    ElMessage.warning(t("core.demo.grid.message.has_children"));
    return;
  }
  ElMessageBox.confirm(
    `${t("common.dialog.delete_single", { resource: t("core.demo.resource.doc_catalog") })}\n${t("common.dialog.resource_field", {
      field: t("core.demo.grid.field.name"),
      value: row.name
    })}`,
    t("common.title.warning"),
    { confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel"), type: "warning" }
  ).then(
    () => {
      const remove = (nodes: DemoCatalog[]): DemoCatalog[] =>
        nodes.filter(node => {
          if (node.children?.length) node.children = remove(node.children);
          return node.id !== row.id;
        });
      catalogList.value = remove([...catalogList.value]);
      ElMessage.success(t("common.message.delete_success", { resource: t("core.demo.resource.doc_catalog") }));
    },
    () => ElMessage.info(t("common.dialog.cancel_delete", { resource: t("core.demo.resource.doc_catalog") }))
  );
}

/** 组装详情模型并打开抽屉：附加“层级路径”字段演示树形数据溯源。 */
function handleOpenDetail(row: DemoCatalog) {
  const path: string[] = [];
  const walk = (nodes: DemoCatalog[], trail: string[]) => {
    for (const node of nodes) {
      const next = [...trail, node.name];
      if (node.id === row.id) path.push(...next);
      if (node.children?.length) walk(node.children, next);
    }
  };
  walk(catalogList.value, []);
  detailModel.value = {
    title: `${t("core.demo.resource.doc_catalog")} · ${row.name}`,
    sections: [
      {
        title: t("core.demo.detail.section.basic"),
        fields: [
          { label: t("core.demo.grid.field.name"), value: row.name },
          { label: t("core.demo.grid.field.code"), value: row.code },
          { label: t("core.demo.table.field.owner"), value: row.owner },
          { label: t("common.field.status"), value: row.status === STATUS_ENABLE ? t("common.status.enabled") : t("common.status.disabled"), tag: true, tagType: row.status === STATUS_ENABLE ? "success" : "info" },
          { label: t("core.demo.grid.field.doc_count"), value: String(row.doc_count) },
          { label: t("core.demo.grid.field.path"), value: path.join(" / ") || row.name }
        ]
      }
    ],
    steps: [
      { title: t("core.demo.detail.step.create"), description: "--" },
      { title: t("core.demo.detail.step.review"), description: "--" },
      { title: t("core.demo.detail.step.publish"), description: "--" }
    ],
    timeline: [
      { title: t("core.demo.detail.timeline.created"), timestamp: "2025-05-06 10:00:00", type: "primary" },
      { title: t("core.demo.detail.timeline.updated"), timestamp: "2026-01-02 15:40:00", type: "success" }
    ],
    raw: { ...row }
  };
  detailVisible.value = true;
}
</script>
