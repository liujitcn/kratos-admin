<!--
  Demo 共享详情抽屉：演示管理端标准的“详情展示”形态。

  组成结构（均为 Element Plus 原生组件，core 未封装 ProDescriptions）：
  - el-drawer（size 62%）承载数据，关闭时通过 update:modelValue 同步父组件；
  - el-tabs 分页签：基本信息 / 流程进度 / 操作时间线 / 原始数据；
  - el-descriptions 描述列表（参照 system/base/api/index.vue 的详情抽屉写法）；
  - el-steps / el-timeline / el-collapse 展示流程、时间线与 JSON 原文。

  各表格页把行数据组装成 DemoDetailModel 后传给本组件即可复用。
-->
<template>
  <el-drawer
    :model-value="modelValue"
    :title="detail?.title ?? t('common.action.detail')"
    size="62%"
    @update:model-value="value => emit('update:modelValue', value)"
  >
    <el-tabs v-if="detail">
      <!-- 页签一：基本信息，多组 el-descriptions 描述列表 -->
      <el-tab-pane :label="t('core.demo.detail.tab.basic')">
        <div v-for="section in detail.sections" :key="section.title" class="detail-section">
          <el-divider content-position="left">{{ section.title }}</el-divider>
          <el-descriptions :column="2" border>
            <el-descriptions-item v-for="field in section.fields" :key="field.label" :label="field.label">
              <el-tag v-if="field.tag" :type="field.tagType ?? 'info'" effect="plain">{{ field.value }}</el-tag>
              <template v-else>{{ field.value }}</template>
            </el-descriptions-item>
          </el-descriptions>
        </div>
      </el-tab-pane>

      <!-- 页签二：流程进度，el-steps 展示节点流转 -->
      <el-tab-pane :label="t('core.demo.detail.tab.progress')">
        <el-steps :active="detail.steps.length - 1" align-center finish-status="success">
          <el-step v-for="(step, index) in detail.steps" :key="step.title" :title="step.title" :description="`T+${index}`" />
        </el-steps>
      </el-tab-pane>

      <!-- 页签三：操作时间线，el-timeline 按时间倒序展示 -->
      <el-tab-pane :label="t('core.demo.detail.tab.timeline')">
        <el-timeline>
          <el-timeline-item
            v-for="item in detail.timeline"
            :key="item.timestamp"
            :timestamp="item.timestamp"
            :type="item.type"
          >
            {{ item.title }}
          </el-timeline-item>
        </el-timeline>
      </el-tab-pane>

      <!-- 页签四：原始数据，el-collapse 折叠面板内展示格式化 JSON -->
      <el-tab-pane :label="t('core.demo.detail.tab.raw')">
        <el-collapse :model-value="rawPanelName">
          <el-collapse-item :title="t('core.demo.detail.raw.title')" :name="rawPanelName">
            <pre class="detail-raw">{{ JSON.stringify(detail.raw, null, 2) }}</pre>
          </el-collapse-item>
        </el-collapse>
      </el-tab-pane>
    </el-tabs>
  </el-drawer>
</template>

<script setup lang="ts">
import { t } from "@/locales";
import type { DemoDetailModel } from "../demo-data";

defineOptions({
  name: "DemoDetailDrawer"
});

/** 组件入参：modelValue 控制显隐，detail 为页面组装好的详情模型。 */
defineProps<{
  /** 抽屉显隐，v-model 双向绑定。 */
  modelValue: boolean;
  /** 详情模型，为空时抽屉仅显示标题占位。 */
  detail: DemoDetailModel | null;
}>();

const emit = defineEmits<{
  "update:modelValue": [value: boolean];
}>();

/** 原始数据折叠面板默认展开的名称。 */
const rawPanelName = "raw";
</script>

<style scoped>
/* 分组之间留出间距，描述列表与分组标题紧凑排列 */
.detail-section + .detail-section {
  margin-top: 16px;
}

/* JSON 原文使用等宽字体并限制纵向滚动，适配暗色主题文字变量 */
.detail-raw {
  max-height: 420px;
  margin: 0;
  overflow: auto;
  font-family: var(--el-font-family-monospace, monospace);
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-regular);
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
