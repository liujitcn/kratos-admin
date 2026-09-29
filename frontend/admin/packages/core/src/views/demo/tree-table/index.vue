<!--
  Demo 页面一：左树右表（TreeFilter + ProTable 经典布局）。

  布局说明：
  - .main-box 是 core 全局样式（packages/core/src/styles/element.scss）定义的横向布局容器，
    左侧 TreeFilter 自适应宽度，右侧 .table-box 撑满剩余空间；
  - 左树通过 @change 把选中的部门写入 initParam，触发右侧表格重新请求（本页为本地静态过滤）；
  - 参照真实页面：system/base/user/index.vue（部门树 + 用户表）。

  本页演示要点：
  - TreeFilter 传入静态 data（不传 requestApi 即不请求接口）+ 默认插槽自定义节点内容；
  - ProTable 单选列（type: "radio"）、图片列（cellType: "image"）、金额列（cellType: "money"）、
    状态开关列（cellType: "status" + beforeChange 确认）；
  - requestApi 返回 Promise 模拟接口分页，真实项目替换为 RPC + buildPageRequest。
-->
<template>
  <div class="main-box">
    <!-- 左侧部门树：静态数据 + 自定义节点插槽（展示部门人数） -->
    <TreeFilter
      label="name"
      :title="t('core.demo.tree.title.dept_tree')"
      :data="demoDeptTree"
      :default-value="deptFilterValue"
      @change="handleDeptChange"
    >
      <!-- 默认插槽：TreeFilter 以 { row } 透传 el-tree 作用域，row.data 为原始节点数据 -->
      <template #default="scope">
        <span>{{ scope.row.data.name }}</span>
        <el-tag class="dept-count" size="small" effect="plain" round>{{ scope.row.data.count }}</el-tag>
      </template>
    </TreeFilter>

    <div class="table-box">
      <ProTable
        ref="proTable"
        row-key="id"
        :columns="columns"
        :header-actions="headerActions"
        :request-api="requestEmployeeTable"
        :init-param="initParam"
      />

      <!-- 共享全组件表单弹窗：新增/编辑入口 -->
      <DemoFormDialog ref="formDialogRef" />

      <!-- 共享详情抽屉：详情入口 -->
      <DemoDetailDrawer v-model="detailVisible" :detail="detailModel" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Delete, EditPen, View } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable";
import TreeFilter from "@/components/TreeFilter/index.vue";
import type { ProFormOption } from "@/components/ProForm/interface";
import { t } from "@/locales";

import DemoFormDialog from "../components/DemoFormDialog.vue";
import DemoDetailDrawer from "../components/DemoDetailDrawer.vue";
import type { DemoDetailModel, DemoDeptNode, DemoEmployee } from "../demo-data";
import { STATUS_DISABLE, STATUS_ENABLE, demoDeptTree, demoEmployees } from "../demo-data";

defineOptions({
  name: "DemoTreeTable",
  inheritAttrs: false
});

/** 左树选中的部门回显值（“全部”节点 id 为空字符串）。 */
const deptFilterValue = ref("");
/** 右侧表格的联动参数：树节点切换后触发表格重新请求。 */
const initParam = reactive({
  dept_id: undefined as number | undefined
});

const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof DemoFormDialog>>();
const detailVisible = ref(false);
const detailModel = ref<DemoDetailModel | null>(null);

/**
 * 员工本地数据副本：删除/状态切换直接改这份副本。
 * 真实项目中这里不需要副本，直接调用 RPC 后刷新表格即可。
 */
const employeeList = ref<DemoEmployee[]>([...demoEmployees]);

/** 状态枚举选项：搜索下拉与表单回填共用。 */
const statusOptions: ProFormOption[] = [
  { label: t("common.status.enabled"), value: STATUS_ENABLE },
  { label: t("common.status.disabled"), value: STATUS_DISABLE }
];

/** 表格列配置：演示 radio 单选列、图片列、金额列、状态开关列与搜索项。 */
const columns = computed<ColumnProps[]>(() => [
  // type: "radio" 单选列；多选用 "selection"，序号/展开/拖拽排序分别对应 index/expand/sort
  { type: "radio", width: 55 },
  { prop: "name", label: t("core.demo.tree.field.name"), minWidth: 120, search: { el: "input", order: 4 } },
  { prop: "email", label: t("core.demo.tree.field.email"), minWidth: 180 },
  { prop: "phone", label: t("core.demo.tree.field.phone"), minWidth: 130, search: { el: "input", order: 3 } },
  {
    // 图片列：cellType: "image"，avatar 为 data URL 也能正常预览
    prop: "avatar",
    label: t("core.demo.form.field.avatar"),
    width: 90,
    cellType: "image",
    imageProps: { width: 40, height: 40, previewWidth: 120, previewHeight: 120 }
  },
  {
    // 金额列：cellType: "money"，prefix 拼在金额前
    prop: "salary",
    label: t("core.demo.tree.field.salary"),
    minWidth: 120,
    align: "right",
    cellType: "money",
    moneyProps: { prefix: "¥" }
  },
  {
    // 状态开关列：beforeChange 里确认后本地切换，返回 false 则回弹
    prop: "status",
    label: t("common.field.status"),
    minWidth: 100,
    search: { el: "select", enum: statusOptions, order: 2 },
    cellType: "status",
    statusProps: {
      activeValue: STATUS_ENABLE,
      inactiveValue: STATUS_DISABLE,
      activeText: t("common.status.enabled"),
      inactiveText: t("common.status.disabled"),
      beforeChange: scope => handleBeforeSetStatus(scope.row as DemoEmployee)
    }
  },
  {
    prop: "entry_date",
    label: t("core.demo.tree.field.entry_date"),
    minWidth: 120,
    align: "center",
    search: { el: "date-picker", props: { type: "daterange", valueFormat: "YYYY-MM-DD", startPlaceholder: t("common.placeholder.start_date"), endPlaceholder: t("common.placeholder.end_date") }, order: 1 }
  },
  {
    // 操作列：cellType: "actions" + actions 数组，label/icon/type/link 决定按钮外观
    prop: "operation",
    label: t("common.field.operation"),
    align: "center",
    cellType: "actions",
    fixed: "right",
    actions: [
      {
        label: t("common.action.detail"),
        type: "primary",
        link: true,
        icon: View,
        onClick: scope => handleOpenDetail(scope.row as DemoEmployee)
      },
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        // params 演示：支持按行动态生成，onClick 第二个参数接收
        params: scope => ({ employeeId: scope.row.id }),
        onClick: (scope, params) => handleEdit((scope.row as DemoEmployee).id ?? (params?.employeeId as number))
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        onClick: scope => handleDelete(scope.row as DemoEmployee)
      }
    ]
  }
]);

/** 顶部按钮：真实项目中用 useAuthButtons 的 BUTTONS 控制可见性，Demo 无权限位直接展示。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    onClick: () => formDialogRef.value?.open()
  }
]);

/**
 * 切换左侧部门树：写入 initParam 并回到第一页，
 * initParam 变化会自动触发 ProTable 重新请求。
 */
function handleDeptChange(value: string) {
  deptFilterValue.value = value ?? "";
  initParam.dept_id = value ? Number(value) : undefined;
  if (proTable.value) {
    proTable.value.pageable.page_num = 1;
  }
}

/**
 * 静态模拟分页请求：本地完成部门过滤 + 搜索 + 分页。
 * 真实项目写法对照 system/base/user/index.vue 的 requestBaseUserTable：
 * 先 buildPageRequest(params) 再调用 RPC，最后返回 { data: { list, total } }。
 */
async function requestEmployeeTable(params: Record<string, any>) {
  // 模拟网络延迟，让 loading 效果可见
  await new Promise(resolve => setTimeout(resolve, 200));
  const pagination = params.pagination ?? {};
  const page = Number(pagination.page_num ?? 1);
  const size = Number(pagination.page_size ?? 10);
  let list = [...employeeList.value];
  // 左树联动：initParam 里的字段会合并进请求参数
  if (initParam.dept_id) {
    list = list.filter(item => item.dept_id === initParam.dept_id);
  }
  // 搜索区参数与列 prop 同名，直接从 params 解构
  if (params.name) list = list.filter(item => item.name.includes(params.name));
  if (params.phone) list = list.filter(item => item.phone.includes(params.phone));
  if (params.status !== undefined && params.status !== null && params.status !== "") {
    list = list.filter(item => item.status === params.status);
  }
  if (Array.isArray(params.entry_date) && params.entry_date.length === 2) {
    const [start, end] = params.entry_date;
    list = list.filter(item => item.entry_date >= start && item.entry_date <= end);
  }
  return { data: { list: list.slice((page - 1) * size, page * size), total: list.length } };
}

/** 状态切换前确认：真实项目在此调用 SetXxxStatus RPC。 */
async function handleBeforeSetStatus(row: DemoEmployee) {
  const nextStatus = row.status === STATUS_ENABLE ? STATUS_DISABLE : STATUS_ENABLE;
  const action = t(nextStatus === STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled");
  try {
    await ElMessageBox.confirm(
      t("common.dialog.status_change", {
        action,
        resource: t("core.demo.resource.employee"),
        field: t("core.demo.tree.field.name"),
        value: row.name
      }),
      t("common.title.notice"),
      { confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel"), type: "warning" }
    );
    row.status = nextStatus;
    ElMessage.success(t("common.message.status_success", { action }));
    return true;
  } catch {
    return false;
  }
}

/** 打开编辑弹窗：把行数据回填进共享表单（仅回填表单中存在的字段）。 */
function handleEdit(id: number) {
  const row = employeeList.value.find(item => item.id === id);
  if (!row) return;
  formDialogRef.value?.open({
    name: row.name,
    intro: `${row.name}（${t("core.demo.resource.employee")}）`,
    birthday: row.entry_date,
    dept_id: row.dept_id,
    color: "#409eff"
  });
}

/**
 * 组装详情模型并打开抽屉：sections/steps/timeline 均由行数据映射，
 * 标签文案走语言包，记录内容保持原样。
 */
function handleOpenDetail(row: DemoEmployee) {
  detailModel.value = {
    title: `${t("core.demo.resource.employee")} · ${row.name}`,
    sections: [
      {
        title: t("core.demo.detail.section.basic"),
        fields: [
          { label: t("core.demo.tree.field.name"), value: row.name },
          { label: t("common.field.status"), value: row.status === STATUS_ENABLE ? t("common.status.enabled") : t("common.status.disabled"), tag: true, tagType: row.status === STATUS_ENABLE ? "success" : "info" },
          { label: t("core.demo.tree.field.email"), value: row.email },
          { label: t("core.demo.tree.field.phone"), value: row.phone },
          { label: t("core.demo.tree.field.entry_date"), value: row.entry_date }
        ]
      },
      {
        title: t("core.demo.detail.section.work"),
        fields: [
          { label: t("common.field.department"), value: findDeptName(row.dept_id) },
          { label: t("core.demo.tree.field.salary"), value: `¥ ${(row.salary / 100).toLocaleString()}` }
        ]
      }
    ],
    steps: [
      { title: t("core.demo.detail.step.create"), description: row.entry_date },
      { title: t("core.demo.detail.step.review"), description: "--" },
      { title: t("core.demo.detail.step.publish"), description: "--" }
    ],
    timeline: [
      { title: t("core.demo.detail.timeline.created"), timestamp: row.entry_date, type: "primary" },
      { title: t("core.demo.detail.timeline.updated"), timestamp: "2026-01-15 10:30:00", type: "success" },
      {
        title: row.status === STATUS_ENABLE ? t("core.demo.detail.timeline.enabled") : t("core.demo.detail.timeline.disabled"),
        timestamp: "2026-03-20 16:00:00",
        type: row.status === STATUS_ENABLE ? "success" : "warning"
      }
    ],
    raw: { ...row }
  };
  detailVisible.value = true;
}

/** 删除员工：确认后从本地副本移除并刷新表格。 */
function handleDelete(row: DemoEmployee) {
  ElMessageBox.confirm(
    `${t("common.dialog.delete_single", { resource: t("core.demo.resource.employee") })}\n${t("common.dialog.resource_field", {
      field: t("core.demo.tree.field.name"),
      value: row.name
    })}`,
    t("common.title.warning"),
    { confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel"), type: "warning" }
  ).then(
    () => {
      employeeList.value = employeeList.value.filter(item => item.id !== row.id);
      ElMessage.success(t("common.message.delete_success", { resource: t("core.demo.resource.employee") }));
      proTable.value?.getTableList();
    },
    () => ElMessage.info(t("common.dialog.cancel_delete", { resource: t("core.demo.resource.employee") }))
  );
}

/** 根据部门 ID 在静态树中查找部门名称（两级即可覆盖 Demo 数据）。 */
function findDeptName(deptId: number): string {
  for (const dept of demoDeptTree as DemoDeptNode[]) {
    if (Number(dept.id) === deptId) return dept.name;
    for (const child of dept.children ?? []) {
      if (Number(child.id) === deptId) return `${dept.name} / ${child.name}`;
    }
  }
  return "--";
}
</script>

<style scoped>
/* 树节点人数徽标：右浮并使用主题间距变量，兼容暗色模式 */
.dept-count {
  float: right;
  margin-left: 8px;
}
</style>
