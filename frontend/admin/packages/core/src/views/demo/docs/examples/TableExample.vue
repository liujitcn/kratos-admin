<template>
  <div class="component-table-example">
    <p class="example-tip">查询、编辑、状态切换与删除均使用本地数据；右侧列设置按钮会打开真实抽屉。</p>
    <ProTable
      :key="treeMode ? 'tree' : 'list'"
      ref="tableRef"
      row-key="id"
      :columns="columns"
      :header-actions="headerActions"
      :request-api="requestProjectTable"
      :pagination="!treeMode"
      :search-col="2"
      :tool-button="active"
    />
    <div class="example-options">
      <el-switch v-model="treeMode" active-text="树形表格" inactive-text="分页表格" />
      <span>{{ rows.length }} 条本地记录</span>
    </div>
    <FormDialog
      v-if="active"
      ref="dialogRef"
      v-model="visible"
      :title="editingId ? '编辑项目' : '新增项目'"
      :model="model"
      :fields="fields"
      @confirm="saveProject"
      @closed="resetForm"
    />
  </div>
</template>

<script setup lang="ts">
import { onDeactivated, reactive, ref } from "vue";
import ProTable from "@/components/ProTable";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import FormDialog from "@/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormInstance } from "@/components/ProForm/interface";
import avatar from "@/assets/images/avatar.png";
import { useDemoActive } from "./useDemoActive";

defineProps<{ componentId: string }>();

/** 本地项目示例行。 */
interface ProjectRow {
  id: number;
  name: string;
  owner: string;
  status: boolean;
  budget: number;
  image: string;
  children?: ProjectRow[];
}

const active = useDemoActive();
const tableRef = ref<ProTableInstance<ProjectRow>>();
const dialogRef = ref<ProFormInstance>();
const treeMode = ref(false);
const visible = ref(false);
const editingId = ref(0);
const model = reactive({ name: "", owner: "张明", status: true, budget: 10000 });
const rows = ref<ProjectRow[]>(
  Array.from({ length: 23 }, (_, index) => ({
    id: index + 1,
    name: ["智慧校园", "校园支付", "教务服务", "访客管理"][index % 4] + (index < 4 ? "平台" : `项目 ${index + 1}`),
    owner: ["张明", "李华", "王芳"][index % 3],
    status: index % 4 !== 3,
    budget: (index + 1) * 3600,
    image: avatar
  }))
);
const fields: ProFormField[] = [
  {
    prop: "name",
    label: "项目名称",
    component: "input",
    rules: [{ required: true, message: "请输入项目名称", trigger: "blur" }]
  },
  {
    prop: "owner",
    label: "负责人",
    component: "select",
    options: ["张明", "李华", "王芳"].map(value => ({ label: value, value }))
  },
  { prop: "budget", label: "预算", component: "input-number", props: { min: 0, controlsPosition: "right" } },
  { prop: "status", label: "启用状态", component: "switch" }
];
const columns: ColumnProps[] = [
  { type: "selection", width: 48 },
  {
    prop: "image",
    label: "图片",
    width: 72,
    cellType: "image",
    imageProps: { src: scope => (active.value ? scope.row.image : ""), width: 32, height: 32 }
  },
  { prop: "name", label: "项目名称", minWidth: 170, search: { el: "input", props: { placeholder: "输入项目名称" } } },
  {
    prop: "owner",
    label: "负责人",
    minWidth: 100,
    search: { el: "select" },
    enum: ["张明", "李华", "王芳"].map(value => ({ label: value, value }))
  },
  { prop: "budget", label: "预算", minWidth: 110, cellType: "money", moneyProps: { prefix: "¥" } },
  {
    prop: "status",
    label: "状态",
    minWidth: 110,
    cellType: "status",
    statusProps: {
      activeValue: true,
      inactiveValue: false,
      activeText: "启用",
      inactiveText: "停用",
      onChange: (value, scope) => {
        scope.row.status = Boolean(value);
      }
    }
  },
  {
    prop: "operation",
    label: "操作",
    width: 120,
    cellType: "actions",
    actions: [
      { label: "编辑", onClick: scope => openProject(scope.row as ProjectRow) },
      { label: "删除", type: "danger", onClick: scope => removeProjects([scope.row.id]) }
    ]
  }
];
const headerActions: HeaderActionProps[] = [
  { label: "新增项目", onClick: () => openProject() },
  {
    label: "删除选中",
    type: "danger",
    disabled: scope => !scope.isSelected,
    onClick: scope => removeProjects(scope.selectedList.map(row => row.id))
  }
];
onDeactivated(() => {
  visible.value = false;
});

/** 在本地完成条件过滤和分页。 */
async function requestProjectTable(params: Record<string, any>) {
  const filtered = rows.value.filter(
    row => (!params.name || row.name.includes(params.name)) && (!params.owner || row.owner === params.owner)
  );
  if (treeMode.value) {
    return {
      data: [
        {
          id: 1000,
          name: "校园应用",
          owner: "张明",
          status: true,
          budget: filtered.reduce((sum, row) => sum + row.budget, 0),
          image: avatar,
          children: filtered.slice(0, 5)
        }
      ]
    };
  }
  const start = (params.page_num - 1) * params.page_size;
  return { data: { list: filtered.slice(start, start + params.page_size), total: filtered.length } };
}

/** 打开项目编辑表单。 */
function openProject(row?: ProjectRow) {
  editingId.value = row?.id ?? 0;
  Object.assign(
    model,
    row
      ? { name: row.name, owner: row.owner, status: row.status, budget: row.budget }
      : { name: "", owner: "张明", status: true, budget: 10000 }
  );
  visible.value = true;
}

/** 保存本地项目并刷新当前列表。 */
function saveProject() {
  const row = rows.value.find(item => item.id === editingId.value);
  if (row) Object.assign(row, model);
  else rows.value.unshift({ id: Math.max(0, ...rows.value.map(item => item.id)) + 1, ...model, image: avatar });
  visible.value = false;
  void tableRef.value?.getTableList();
}

/** 删除本地选中记录并清理选择态。 */
function removeProjects(ids: number[]) {
  rows.value = rows.value.filter(row => !ids.includes(row.id));
  tableRef.value?.clearSelection();
  void tableRef.value?.getTableList();
}

/** 关闭后重置表单及校验状态。 */
function resetForm() {
  Object.assign(model, { name: "", owner: "张明", status: true, budget: 10000 });
  dialogRef.value?.clearValidate();
}
</script>

<style scoped lang="scss">
.component-table-example {
  min-width: 0;
}
.example-tip {
  margin: 0 0 18px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
.example-options {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: center;
  margin-top: 16px;
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}
.component-table-example :deep(.table-search),
.component-table-example :deep(.table-main) {
  padding: 16px;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border-soft);
  box-shadow: none;
}
.component-table-example :deep(.table-main) {
  height: auto;
}
</style>
