<template>
  <el-button
    v-for="block in shortcutActions"
    :key="block.action.action_id"
    class="shortcut-confirm-action"
    type="primary"
    plain
    size="small"
    :disabled="block.disabled || !isFlowActionFromMessage(block.action, activeFlowMessageId)"
    @click="forwardFlowAction(block.action, block.label || t('system.ai.shortcut.action.continue'))"
  >
    {{ block.label || t("system.ai.shortcut.action.continue") }}
  </el-button>
  <template v-for="renderer in renderedBlocks" :key="renderer.key">
    <component
      :is="renderer.component"
      :message="renderer.message"
      :active-flow-message-id="activeFlowMessageId"
      @flow-action="forwardFlowAction"
      @fill-draft="emit('fill-draft', $event)"
      @block-saved="emit('block-saved', $event)"
    />
  </template>
  <section v-for="(block, index) in fallbackBlocks" :key="index" class="ai-flow-fallback">
    <strong v-if="block.title">{{ block.title }}</strong>
    <p v-if="block.desc">{{ block.desc }}</p>
    <el-button
      v-if="block.action?.type && block.label"
      type="primary"
      plain
      size="small"
      :disabled="block.disabled || !isFlowActionFromMessage(block.action, activeFlowMessageId)"
      @click="forwardFlowAction(block.action, block.label)"
    >
      {{ block.label }}
    </el-button>
  </section>
</template>

<script setup lang="ts">
import { computed, type Component } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import type { AiAction } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_message";
import type { AdminAiFlowBlockRenderer } from "../../ai";
import { isFlowActionFromMessage } from "../../views/ai/chat/message";
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
  /** 将业务组件提供的问题写入当前聊天草稿。 */
  "fill-draft": [text: string];
  /** 将业务组件创建的资源交给聊天入口。 */
  "block-saved": [value: unknown];
}>();

const shortcutActions = computed(() =>
  (props.message.blocks ?? []).filter(
    (block): block is AIFlowBlock & { action: AiAction } => block.type === "shortcut_confirm" && Boolean(block.action)
  )
);

const renderedBlocks = computed(() =>
  props.renderers.flatMap(renderer => {
    const blocks = selectBlocks(props.message.blocks ?? [], renderer);
    return blocks.length
      ? [{ key: renderer.key, component: renderer.component as Component, message: { ...props.message, blocks } }]
      : [];
  })
);

const hostBlockTypes = new Set(["shortcut_confirm", "knowledge_sources", "approval_outcome", "reply_error", "branch_origin", "generated_file_issue"]);
const fallbackBlocks = computed(() =>
  (props.message.blocks ?? []).filter(block =>
    !hostBlockTypes.has(block.type) &&
    (typeof block.title === "string" || typeof block.desc === "string") &&
    !props.renderers.some(renderer => !renderer.types?.length || renderer.types.includes(block.type))
  )
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

<style scoped>
.shortcut-confirm-action {
  justify-self: start;
}
.ai-flow-fallback {
  padding: 10px 12px;
  border: 1px solid var(--admin-page-divider);
  border-radius: var(--admin-page-radius);
  font-size: 13px;
}
.ai-flow-fallback p {
  margin: 4px 0 0;
  color: var(--admin-page-text-secondary);
}
</style>
