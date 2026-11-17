<template>
  <template v-for="renderer in renderedBlocks" :key="renderer.key">
    <component
      :is="renderer.component"
      :message="renderer.message"
      :active-flow-message-id="activeFlowMessageId"
      @flow-action="forwardFlowAction"
    />
  </template>
</template>

<script setup lang="ts">
import { computed, type Component } from "vue";
import type { AiAction } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_message";
import type { AdminAiFlowBlockRenderer } from "../../ai";
import type { AIFlowBlock, ChatMessageItem } from "../../views/ai/chat/types";

const props = defineProps<{
  /** 当前助手消息及其结构化流程块。 */
  message: ChatMessageItem;
  /** 当前会话中最新可交互流程消息的编号。 */
  activeFlowMessageId: string;
  /** 由模块组合根提供的流程块渲染器。 */
  renderers: AdminAiFlowBlockRenderer[];
}>();

const emit = defineEmits<{
  /** 将业务组件触发的流程动作交回聊天页面。 */
  "flow-action": [action: AiAction, label?: string];
}>();

const renderedBlocks = computed(() =>
  props.renderers.flatMap(renderer => {
    const blocks = selectBlocks(props.message.blocks ?? [], renderer);
    return blocks.length
      ? [{ key: renderer.key, component: renderer.component as Component, message: { ...props.message, blocks } }]
      : [];
  })
);

/** 按渲染器声明筛选流程块，未声明类型的旧组件继续接收全部块。 */
function selectBlocks(blocks: AIFlowBlock[], renderer: AdminAiFlowBlockRenderer) {
  if (!renderer.types?.length) return blocks;
  const types = new Set(renderer.types);
  return blocks.filter(block => types.has(block.type));
}

/** 统一转发所有业务流程组件的动作事件。 */
function forwardFlowAction(action: AiAction, label?: string) {
  emit("flow-action", action, label);
}
</script>
