<!--
  Demo 页面二：纯表格（标准 Table 基准页，ProTable 全量能力展示）。

  本页演示要点（对照 core/src/components/ProTable/interface/index.ts 阅读）：
  - 列类型：type: "selection" 多选列（radio 单选在“左树右表”页演示）；
  - 预置单元格：cellType: "money" 金额列、"actions" 操作列；
  - 标签列：tag: true + enum 的 tagType，按枚举值渲染 el-tag；
  - 多级表头：_children 嵌套出“任务情况”分组表头；
  - 自定义渲染：render 渲染进度条单元格、headerRender 渲染带提示的表头；
  - 搜索区：演示 input / input-number / select / select-v2 / tree-select / cascader /
    date-picker / time-picker / time-select / radio / switch / slider 十二种控件
    （tenant-select 需要租户接口授权，静态页不演示，用法与表单内一致）；
    部分搜索项绑定 isShow: false 的隐藏列，说明“搜索字段不必是可见列”；
  - 组件级：toolButton 数组形态、border、headerActions 批量操作、requestApi 静态分页模拟。

  requestApi 真实写法（本页用本地过滤模拟）：
    const data = await defXxxService.PageXxx({ ...buildPageRequest(params), ...筛选 });
    return { data: { list: data.list ?? [], total: data.total } };
-->
<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      border
      :columns="columns"
      :header-actions="headerActions"
      :tool-button="['refresh', 'setting', 'search']"
      :request-api="requestProjectTable"
      :request-auto="true"
    />

    <DemoFormDialog ref="formDialogRef" />
    <DemoDetailDrawer v-model="detailVisible" :detail="detailModel" />
  </div>
</template>

<script setup lang="ts">
import { computed, h, ref } from "vue";
import { ElMessage, ElMessageBox, ElProgress, ElTooltip } from "element-plus";
import { CirclePlus, Delete, EditPen, QuestionFilled, View } from "@element-plus/icons-vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@/components/ProTable/interface";
import ProTable from "@/components/ProTable";
import { t } from "@/locales";

import DemoFormDialog from "../components/DemoFormDialog.vue";
import DemoDetailDrawer from "../components/DemoDetailDrawer.vue";
import type { DemoDetailModel, DemoProject } from "../demo-data";
import { STATUS_DISABLE, STATUS_ENABLE, demoLevelOptions, demoProjects, demoRegionTree } from "../demo-data";

defineOptions({
  name: "DemoTable",
  inheritAttrs: false
});

const proTable = ref<ProTableInstance>();
const formDialogRef = ref<InstanceType<typeof DemoFormDialog>>();
const detailVisible = ref(false);
const detailModel = ref<DemoDetailModel | null>(null);

/** 项目本地数据副本：删除操作在这份副本上进行，真实项目调用 RPC 后刷新表格。 */
const projectList = ref<DemoProject[]>([...demoProjects]);

/** 状态枚举选项：tagType 决定标签颜色，同时供搜索下拉复用。 */
const statusTagOptions = [
  { label: t("common.status.enabled"), value: STATUS_ENABLE, tagType: "success" },
  { label: t("common.status.disabled"), value: STATUS_DISABLE, tagType: "info" }
];

/** select-v2 大数据下拉的静态选项（isShow: false，仅出现在搜索区）。 */
const tagSearchOptions = Array.from({ length: 50 }, (_, index) => ({ label: `标签 ${index + 1}`, value: `tag-${index + 1}` }));

/** tree-select 搜索项的静态团队树（isShow: false，仅出现在搜索区）。 */
const ownerGroupOptions = demoRegionTree;

/** 把级联编码翻译成“省 / 市”文本，用于 region 列单元格展示。 */
function formatRegion(region: string[]): string {
  const names: string[] = [];
  for (const code of region) {
    for (const province of demoRegionTree) {
      if (province.value === code) names.push(province.label);
      const city = province.children.find(item => item.value === code);
      if (city) names.push(city.label);
    }
  }
  return names.join(" / ");
}

/**
 * 表格列配置：搜索项 order 从大到小排列，
 * 未配置 order 的搜索项按列定义顺序排在后面。
 */
const columns = computed<ColumnProps[]>(() => [
  // 多选列：配合 headerActions 的批量删除（selectedList）使用
  { type: "selection", width: 55 },
  { prop: "name", label: t("core.demo.table.field.name"), minWidth: 160, search: { el: "input", order: 8 } },
  {
    // 标签列演示：tag: true 后按 enum 里的 tagType 渲染 el-tag
    prop: "level",
    label: t("core.demo.table.field.level"),
    width: 90,
    align: "center",
    tag: true,
    enum: demoLevelOptions,
    search: { el: "radio", enum: demoLevelOptions, order: 6 }
  },
  { prop: "owner", label: t("core.demo.table.field.owner"), minWidth: 100 },
  {
    prop: "status",
    label: t("common.field.status"),
    minWidth: 90,
    align: "center",
    tag: true,
    enum: statusTagOptions,
    search: { el: "select", enum: statusTagOptions, order: 7 }
  },
  {
    // 金额列演示：cellType: "money"，suffix 拼在金额后
    prop: "budget",
    label: t("core.demo.table.field.budget"),
    minWidth: 130,
    align: "right",
    cellType: "money",
    moneyProps: { prefix: "¥", suffix: " 元" }
  },
  {
    // 多级表头：_children 嵌套的列渲染在“任务情况”分组下
    prop: "task_group",
    label: t("core.demo.table.group.task"),
    align: "center",
    _children: [
      {
        prop: "task_total",
        label: t("core.demo.table.field.task_total"),
        minWidth: 110,
        search: { el: "input-number", props: { min: 0, max: 999 }, order: 5 }
      },
      {
        // 自定义单元格渲染：render 返回 VNode，这里用 el-progress 画进度条
        prop: "progress",
        label: t("core.demo.table.field.progress"),
        minWidth: 150,
        search: { el: "slider", props: { min: 0, max: 100, range: true }, order: 4 },
        headerRender: () =>
          // 自定义表头渲染：列名后追加提示图标
          h("span", { style: "display:inline-flex;align-items:center;gap:4px" }, [
            t("core.demo.table.field.progress"),
            h(ElTooltip, { content: t("core.demo.table.tooltip.progress"), placement: "top" }, () =>
              h(QuestionFilled, { style: "width:14px;height:14px;color:var(--el-text-color-secondary)" })
            )
          ]),
        render: scope =>
          h(ElProgress, {
            percentage: Number((scope.row as DemoProject).progress) || 0,
            strokeWidth: 8,
            textInside: true
          })
      }
    ]
  },
  {
    // 普通自定义渲染：把级联编码格式化为“省 / 市”文本
    prop: "region",
    label: t("core.demo.table.field.region"),
    minWidth: 160,
    search: {
      el: "cascader",
      props: { options: demoRegionTree, clearable: true, placeholder: t("common.placeholder.select") },
      order: 3
    },
    render: scope => formatRegion((scope.row as DemoProject).region)
  },
  {
    prop: "created_at",
    label: t("common.field.created_at"),
    minWidth: 170,
    align: "center",
    search: {
      el: "date-picker",
      props: { type: "daterange", valueFormat: "YYYY-MM-DD", startPlaceholder: t("common.placeholder.start_date"), endPlaceholder: t("common.placeholder.end_date") },
      order: 2
    }
  },
  // ── 以下列 isShow: false，只出现在搜索区，演示“搜索字段不必是可见列” ──
  {
    // select-v2：虚拟列表下拉，适合大数据量选项
    prop: "tags",
    label: t("core.demo.table.field.tags"),
    isShow: false,
    search: { el: "select-v2", enum: tagSearchOptions, props: { clearable: true, filterable: true }, order: 1 }
  },
  {
    // tree-select：树形下拉搜索
    prop: "owner_group",
    label: t("core.demo.table.field.owner_group"),
    isShow: false,
    search: { el: "tree-select", enum: ownerGroupOptions, props: { clearable: true }, order: 1 }
  },
  {
    // time-picker：时间范围
    prop: "updated_at",
    label: t("common.field.updated_at"),
    isShow: false,
    search: { el: "time-picker", props: { valueFormat: "HH:mm:ss" }, order: 1 }
  },
  {
    // time-select：单时间下拉
    prop: "start_time",
    label: t("common.field.start_time"),
    isShow: false,
    search: { el: "time-select", props: { start: "08:00", end: "22:00", step: "00:30" }, order: 1 }
  },
  {
    // switch：布尔开关搜索，defaultValue 演示搜索默认值
    prop: "enabled",
    label: t("core.demo.table.field.enabled"),
    isShow: false,
    search: { el: "switch", defaultValue: undefined, order: 1 }
  },
  {
    // 操作列：详情 / 编辑 / 删除
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
        onClick: scope => handleOpenDetail(scope.row as DemoProject)
      },
      {
        label: t("common.action.edit"),
        type: "primary",
        link: true,
        icon: EditPen,
        onClick: scope => handleEdit(scope.row as DemoProject)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        onClick: scope => handleDelete(scope.row as DemoProject)
      }
    ]
  }
]);

/** 顶部按钮：新增打开全组件表单弹窗，批量删除要求先勾选行。 */
const headerActions = computed<HeaderActionProps[]>(() => [
  {
    label: t("common.action.create"),
    type: "success",
    icon: CirclePlus,
    onClick: () => formDialogRef.value?.open()
  },
  {
    label: t("common.action.delete"),
    type: "danger",
    icon: Delete,
    // disabled 回调拿到 HeaderActionScope，未勾选时置灰
    disabled: scope => !scope.selectedList.length,
    onClick: scope => handleDelete(scope.selectedList as DemoProject[])
  }
]);

/**
 * 静态模拟分页请求：本地实现搜索 + 分页。
 * 演示了常见搜索参数的取值形态：区间日期为数组、级联为数组、
 * 范围滑杆为数组、开关为布尔，真实项目直接透传给后端即可。
 */
async function requestProjectTable(params: Record<string, any>) {
  await new Promise(resolve => setTimeout(resolve, 200));
  const pagination = params.pagination ?? {};
  const page = Number(pagination.page_num ?? 1);
  const size = Number(pagination.page_size ?? 10);
  let list = [...projectList.value];
  if (params.name) list = list.filter(item => item.name.includes(params.name));
  if (params.level) list = list.filter(item => item.level === params.level);
  if (params.status !== undefined && params.status !== null && params.status !== "") {
    list = list.filter(item => item.status === params.status);
  }
  if (params.task_total !== undefined && params.task_total !== null && params.task_total !== "") {
    list = list.filter(item => item.task_total >= Number(params.task_total));
  }
  // 级联搜索：选中到市则精确匹配，仅选省则按首段匹配
  if (Array.isArray(params.region) && params.region.length) {
    list = list.filter(
      item => item.region.slice(0, params.region.length).join("/") === params.region.join("/")
    );
  }
  if (Array.isArray(params.progress) && params.progress.length === 2) {
    const [min, max] = params.progress;
    list = list.filter(item => item.progress >= min && item.progress <= max);
  }
  if (Array.isArray(params.created_at) && params.created_at.length === 2) {
    const [start, end] = params.created_at;
    list = list.filter(item => item.created_at.slice(0, 10) >= start && item.created_at.slice(0, 10) <= end);
  }
  return { data: { list: list.slice((page - 1) * size, page * size), total: list.length } };
}

/** 打开编辑弹窗：把行数据回填进共享表单。 */
function handleEdit(row: DemoProject) {
  formDialogRef.value?.open({
    name: row.name,
    level: row.level,
    visibility: row.status === STATUS_ENABLE ? "public" : "private",
    count: row.task_total,
    range: [row.created_at, row.created_at]
  });
}

/** 组装详情模型并打开抽屉。 */
function handleOpenDetail(row: DemoProject) {
  const levelOption = demoLevelOptions.find(item => item.value === row.level);
  detailModel.value = {
    title: `${t("core.demo.resource.project")} · ${row.name}`,
    sections: [
      {
        title: t("core.demo.detail.section.basic"),
        fields: [
          { label: t("core.demo.table.field.name"), value: row.name },
          { label: t("core.demo.table.field.level"), value: levelOption?.label ?? row.level, tag: true, tagType: (levelOption?.tagType as "success" | "info" | "warning" | "danger") ?? "info" },
          { label: t("core.demo.table.field.owner"), value: row.owner },
          { label: t("common.field.status"), value: row.status === STATUS_ENABLE ? t("common.status.enabled") : t("common.status.disabled"), tag: true, tagType: row.status === STATUS_ENABLE ? "success" : "info" },
          { label: t("core.demo.table.field.budget"), value: `¥ ${(row.budget / 100).toLocaleString()}` },
          { label: t("core.demo.table.field.region"), value: formatRegion(row.region) },
          { label: t("common.field.created_at"), value: row.created_at }
        ]
      },
      {
        title: t("core.demo.table.group.task"),
        fields: [
          { label: t("core.demo.table.field.task_total"), value: String(row.task_total) },
          { label: t("core.demo.table.field.progress"), value: `${row.progress}%` }
        ]
      }
    ],
    steps: [
      { title: t("core.demo.detail.step.create"), description: row.created_at.slice(0, 10) },
      { title: t("core.demo.detail.step.review"), description: "--" },
      { title: t("core.demo.detail.step.publish"), description: "--" },
      { title: t("core.demo.detail.step.done"), description: "--" }
    ],
    timeline: [
      { title: t("core.demo.detail.timeline.created"), timestamp: row.created_at, type: "primary" },
      { title: t("core.demo.detail.timeline.updated"), timestamp: "2026-02-10 09:00:00", type: "success" },
      {
        title: row.status === STATUS_ENABLE ? t("core.demo.detail.timeline.enabled") : t("core.demo.detail.timeline.disabled"),
        timestamp: "2026-04-01 14:20:00",
        type: row.status === STATUS_ENABLE ? "success" : "danger"
      }
    ],
    raw: { ...row }
  };
  detailVisible.value = true;
}

/**
 * 删除项目：兼容“单行对象 / 勾选数组”两种入参（与真实页面写法一致），
 * 确认后从本地副本移除并刷新表格。
 */
function handleDelete(selected: DemoProject | DemoProject[]) {
  const rows = Array.isArray(selected) ? selected : [selected];
  const confirmMessage =
    rows.length === 1
      ? `${t("common.dialog.delete_single", { resource: t("core.demo.resource.project") })}\n${t("common.dialog.resource_field", {
          field: t("core.demo.table.field.name"),
          value: rows[0].name
        })}`
      : t("common.dialog.delete_batch", { count: rows.length, unit: "", resource: t("core.demo.resource.project") });
  ElMessageBox.confirm(confirmMessage, t("common.title.warning"), {
    confirmButtonText: t("common.action.confirm"),
    cancelButtonText: t("common.action.cancel"),
    type: "warning"
  }).then(
    () => {
      const ids = new Set(rows.map(row => row.id));
      projectList.value = projectList.value.filter(item => !ids.has(item.id));
      ElMessage.success(t("common.message.delete_success", { resource: t("core.demo.resource.project") }));
      proTable.value?.getTableList();
    },
    () => ElMessage.info(t("common.dialog.cancel_delete", { resource: t("core.demo.resource.project") }))
  );
}
</script>
