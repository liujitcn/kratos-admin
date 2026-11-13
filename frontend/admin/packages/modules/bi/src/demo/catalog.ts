import type { AdminComponentApi, AdminComponentDoc } from "@liujitcn/kratos-admin-core";
import componentApi from "./component-api.json";

const chartExample = () => import("./ChartExample.vue");
const dashboardExample = () => import("./DashboardExample.vue");
const docs: Omit<AdminComponentDoc, "owner" | "group" | "api">[] = [
  {
    id: "BiChart",
    label: "BI 图表",
    file: "BiChart.vue",
    description: "根据图表配置与行对象渲染图表，处理主题、尺寸、空态和点击事件。",
    usage:
      "chart 使用 BiChartSchema，rows 为按字段名索引的对象数组，theme 由 useBiTheme() 提供。encoding 必须引用 rows 中的字段；datasetId 是配置中的标识，实际数据直接通过 rows 传入。组件不会发起业务查询。示例可切换图型、查看空态并接收点击事件。以下代码位于 BI 模块 src/demo。",
    code: `<script setup lang="ts">
import BiChart from "../components/BiChart.vue";
import { useBiTheme } from "../env";
import type { BiChartSchema } from "../bi-core";

const theme = useBiTheme();
const rows = [{ region: "华东", amount: 224 }, { region: "华南", amount: 268 }, { region: "华北", amount: 188 }];
const chart: BiChartSchema = {
  id: "service-sales", datasetId: "local-sales", type: "bar",
  encoding: { dimension: "region", measure: "amount" },
  aggregation: "none", style: { showLabel: true }
};
</script>

<template>
  <div style="height: 300px">
    <BiChart :chart="chart" :rows="rows" :theme="theme" empty-text="暂无数据" />
  </div>
</template>`,
    example: chartExample,
    related: ["BiCardVisualization", "ECharts"]
  },
  {
    id: "BiCardVisualization",
    label: "卡片可视化",
    file: "BiCardVisualization.vue",
    description: "把卡片查询结果映射为图表或指标展示，支持工作台缩略图使用的紧凑模式。",
    usage:
      "card 使用 BiCard，result 使用 BiQueryResult：columns 定义编码和类型，rows 每项的 values 按列顺序排列且为字符串。chart_type=metric 显示无分组指标，其他图型使用 chart_json 中的字段映射。compact 缩小指标字号并精简图表。组件不会请求数据；演示数据定义在本模块 src/demo/fixtures.ts。",
    code: `<script setup lang="ts">
import BiCardVisualization from "../components/BiCardVisualization.vue";
import { demoCard, demoResult, demoMetricCard, demoMetricResult } from "./fixtures";
</script>

<template>
  <div style="height: 300px">
    <BiCardVisualization :card="demoCard" :result="demoResult" />
  </div>
  <BiCardVisualization :card="demoMetricCard" :result="demoMetricResult" compact />
</template>`,
    example: chartExample,
    related: ["BiChart", "BiDataCard", "BiWorkbenchPreview"]
  },
  {
    id: "BiDataCard",
    label: "BI 数据卡片",
    file: "BiDataCard.vue",
    description: "组合卡片标题、分类、周期筛选、图表和来源说明，提供操作与底部插槽。",
    usage:
      "传 previewResult 可直接展示本地查询结果，并跳过真实取数；未传时按已保存 card.id 调用 GetBiCardData。id=0 草稿不会查询，refresh_seconds=0 关闭自动刷新。rangeDays 需配合 showPeriodFilter 与 update:rangeDays，由父页面控制。示例始终传 previewResult，不改变任何卡片记录。",
    code: `<script setup lang="ts">
import BiDataCard from "../components/BiDataCard.vue";
import { demoCard, demoResult } from "./fixtures";
</script>

<template>
  <BiDataCard :card="demoCard" :preview-result="demoResult" source-name="本地服务数据">
    <template #actions><el-tag effect="plain">本地预览</el-tag></template>
    <template #footer>数据保存在当前页面内存中</template>
  </BiDataCard>
</template>`,
    example: chartExample,
    related: ["BiCardVisualization", "BiDashboard", "BiCardFormDialog"]
  },
  {
    id: "BiDashboard",
    label: "仪表盘展示",
    file: "BiDashboard.vue",
    description: "按 schemaVersion=3 的仪表盘发布快照排列卡片，读取外部传入的结果。",
    usage:
      "schema 使用 DashboardLayout，cards 保存卡片快照及 12 列栅格位置；results 是 Map，键必须为布局项 id，而非 card_id。卡片展示复用 BiDataCard 的 previewResult，不在组件内查询。父页面负责加载发布版本和查询结果，示例使用完整的本地快照与结果。",
    code: `<script setup lang="ts">
import BiDashboard from "../components/BiDashboard.vue";
import { demoLayout, demoResults } from "./fixtures";

// demoLayout.schemaVersion === 3
// demoResults 的键与 demoLayout.cards[].id 一致。
</script>

<template>
  <BiDashboard :schema="demoLayout" :results="demoResults" />
</template>`,
    example: dashboardExample,
    related: ["BiDataCard", "BiDesigner", "BiWorkbenchPreview"]
  },
  {
    id: "BiDesigner",
    label: "仪表盘设计器",
    file: "BiDesigner.vue",
    description: "从卡片列表添加引用，通过拖动、方向按钮和尺寸设置编辑仪表盘布局。",
    usage:
      "v-model 使用 DashboardLayout，cards 为可选 BiCard 数组；edit-card 与 create-card 交给父页面打开业务编辑器，布局保存也由父页面完成。组件内部的卡片预览会按已保存卡片编号取数，当前没有注入本地结果的入口。此示例只提供 id=0 的未保存草稿，演示添加、尺寸和移除，图表区域保留真实空态。",
    code: `<script setup lang="ts">
import { ref } from "vue";
import BiDesigner from "../components/BiDesigner.vue";
import type { DashboardLayout } from "../bi-dashboard-schema";
import { demoCard } from "./fixtures";

const layout = ref<DashboardLayout>({ schemaVersion: 3, name: "本地布局草稿", cards: [] });
// demoCard.id 为 0，refresh_seconds 为 0，不触发真实取数。
const cards = [demoCard];
</script>

<template>
  <BiDesigner v-model="layout" :cards="cards" />
</template>`,
    example: dashboardExample,
    related: ["BiDashboard", "BiDataCard", "BiCardFormDialog"]
  },
  {
    id: "BiCardLibrary",
    label: "数据卡片库",
    file: "BiCardLibrary.vue",
    description: "按来源和分类浏览真实数据卡片，组合收藏、预览、编辑及加入仪表盘。",
    usage:
      "该业务容器没有 props 或对外事件，挂载后直接加载语义数据源、工作台收藏和卡片分页。收藏、删除、编辑、加入仪表盘均会调用实际服务，也依赖路由上下文；这里仅展示 BI 模块内的接入方式。卡片的纯展示可使用 BiDataCard。",
    code: `<script setup lang="ts">
import BiCardLibrary from "../components/BiCardLibrary.vue";
</script>

<template>
  <!-- 在 BI 卡片业务页挂载，会加载当前账号可见的实际数据。 -->
  <BiCardLibrary />
</template>`,
    related: ["BiDataCard", "BiCardFormDialog", "BiDesigner"]
  },
  {
    id: "BiCardChartEditor",
    label: "卡片图表编辑",
    file: "BiCardChartEditor.vue",
    description: "编辑卡片图型、维度和指标映射，以及图例、标签、堆叠等样式。",
    usage:
      "v-model 使用 CardChartConfig，v-model:chart-type 为图型字符串；query 的维度、指标与 result.columns 一起决定可选字段和图型。编辑只发出受控状态，不查询、不保存。无分组查询才启用指标卡和仪表图，散点图需要两个数值字段，桑基图等需要两个分组维度。示例同步调用可视化组件预览。",
    code: `<script setup lang="ts">
import { computed, ref } from "vue";
import BiCardChartEditor from "../components/BiCardChartEditor.vue";
import BiCardVisualization from "../components/BiCardVisualization.vue";
import type { CardChartConfig } from "../card-schema";
import { demoCard, demoChartConfig, demoQuery, demoResult } from "./fixtures";

const chartType = ref("bar");
const config = ref<CardChartConfig>({ encoding: { ...demoChartConfig.encoding }, style: { ...demoChartConfig.style } });
const card = computed(() => ({ ...demoCard, chart_type: chartType.value, chart_json: JSON.stringify(config.value) }));
</script>

<template>
  <BiCardChartEditor v-model="config" v-model:chart-type="chartType" :query="demoQuery" :result="demoResult" />
  <div style="height: 300px"><BiCardVisualization :card="card" :result="demoResult" /></div>
</template>`,
    example: chartExample,
    related: ["BiCardVisualization", "BiCardFormDialog"]
  },
  {
    id: "BiCardFormDialog",
    label: "数据卡片表单",
    file: "BiCardFormDialog.vue",
    description: "基于已有智能问数结果创建或编辑卡片，组合图型编辑、预览和实际保存。",
    usage:
      "v-model 控制打开；sources 为 BiSourceDescriptor 数组，编辑时传 card，新建时传 initialQuery、initialResult、initialTitle、initialChartType。保存必须关联成功问数的 query_record_id，不能用本地伪造结果代替。预览和确认会调用服务，saved 返回实际保存的卡片，因此文档页仅展示接入代码。",
    code: `<BiCardFormDialog
  v-model="editorOpen"
  :sources="sources"
  :card="editingCard"
  :initial-query="query"
  :initial-result="result"
  initial-title="区域服务概览"
  initial-chart-type="bar"
  @saved="reloadCards"
/>

<!-- BI 模块内部从 ../components/BiCardFormDialog.vue 导入。
query/result 来自成功的智能问数，result.query_record_id 为实际查询记录编号。 -->`,
    related: ["BiCardChartEditor", "BiDataCard", "FormDialog"]
  },
  {
    id: "BiSourceModelDialog",
    label: "数据源模型弹窗",
    file: "BiSourceModelDialog.vue",
    description: "读取数据库结构，通过表关系画布配置语义模型、维度与指标。",
    usage:
      "v-model 控制弹窗，source 为已有 BiSource 记录。打开后按 source.id 读取真实数据表，保存会更新语义模型，预览会执行查询；saved 返回更新后的数据源。它依赖数据库结构和数据源权限，不适合用虚构数据表在文档页挂载。",
    code: `<BiSourceModelDialog
  v-model="modelOpen"
  :source="editingSource"
  @saved="reloadSources"
/>

<!-- BI 模块内部从 ../components/BiSourceModelDialog.vue 导入。
editingSource 为已保存的 BiSource，关闭和保存状态由业务页面维护。 -->`,
    related: ["BiCardFormDialog"]
  },
  {
    id: "BiDashboardMenuDialog",
    label: "仪表盘菜单配置",
    file: "BiDashboardMenuDialog.vue",
    description: "为仪表盘创建或更新管理端菜单入口，配置父目录、标题、图标和译文。",
    usage:
      "v-model 控制打开，dashboard 为已保存的 BiDashboard。打开会加载真实语言和菜单目录，确认会创建或更新菜单，并设置仪表盘路由及数据接口权限。需要相应菜单管理权限，文档页仅展示集成方式。",
    code: `<BiDashboardMenuDialog v-model="menuOpen" :dashboard="selectedDashboard" />

<!-- BI 模块内部从 ../components/BiDashboardMenuDialog.vue 导入。
selectedDashboard 为实际仪表盘，确认会写入管理端菜单。 -->`,
    related: ["BiDashboard", "DynamicI18nEditor", "SelectIcon", "FormDialog"]
  },
  {
    id: "BiWorkbenchPreview",
    label: "工作台预览",
    file: "BiWorkbenchPreview.vue",
    description: "当工作台仪表盘入口进入可视区域后，按发布版本加载前两张卡片的缩略图。",
    usage:
      "dashboard 使用 BiDashboard，包含 id 与 published_version。组件自行使用 IntersectionObserver 延迟请求 GetBiDashboardData，之后解析发布快照并通过紧凑可视化渲染；没有本地结果注入入口，所以此处提供集成说明。纯本地仪表盘预览可使用 BiDashboard。",
    code: `<BiWorkbenchPreview :dashboard="dashboard" />

<!-- BI 模块内部从 ../components/BiWorkbenchPreview.vue 导入。
dashboard 为服务返回的已发布仪表盘，进入可视区域后读取该发布版本数据。 -->`,
    related: ["BiDashboard", "BiCardVisualization"]
  }
];

const apis: Record<string, AdminComponentApi> = componentApi;

/** BI 模块提供的组件说明与本地示例。 */
export const biComponentDocs: AdminComponentDoc[] = docs.map(doc => ({
  ...doc,
  group: "数据分析",
  owner: "BI",
  api: apis[doc.file]
}));
