<template>
  <main class="agent-chat-panel" :class="{ 'is-empty': isEmptyState, 'has-shortcuts': hasShortcutPanel }">
    <template v-if="isEmptyState">
      <div class="agent-chat-empty">
        <div class="agent-chat-empty__title">{{ welcomeTitle }}</div>
        <div class="agent-chat-empty__desc">可直接提问，也可以上传附件一起分析。</div>
        <section v-if="hasShortcutPanel" class="agent-shortcuts">
          <div class="agent-shortcuts__head">
            <div>
              <div class="agent-shortcuts__eyebrow">快捷操作</div>
              <div class="agent-shortcuts__title">从常用工作流开始</div>
            </div>
            <div class="agent-shortcuts__actions">
              <span class="agent-shortcuts__count">{{ shortcutBatchLabel }}</span>
              <button
                v-if="canRefreshShortcutBatch"
                class="agent-shortcuts__refresh"
                type="button"
                :disabled="sending"
                @click="refreshShortcutBatch"
              >
                <span>换一批</span>
                <el-icon><Refresh /></el-icon>
              </button>
            </div>
          </div>
          <div v-if="loadingShortcuts" class="agent-shortcuts__loading">正在加载快捷入口...</div>
          <div v-else class="agent-shortcuts__grid">
            <button
              v-for="shortcut in shortcutVisibleList"
              :key="shortcut.key"
              class="agent-shortcut"
              type="button"
              :disabled="sending"
              @click="handleShortcutClick(shortcut)"
            >
              <span class="agent-shortcut__icon">
                <el-icon><component :is="resolveShortcutIcon(shortcut)" /></el-icon>
              </span>
              <span class="agent-shortcut__content">
                <span class="agent-shortcut__title">{{ shortcut.title }}</span>
                <span class="agent-shortcut__meta">{{ resolveShortcutMeta(shortcut) }}</span>
              </span>
            </button>
          </div>
        </section>
        <div class="agent-chat-empty__sender">
          <XSender :key="senderKey" :sending="sending" @submit="handleSubmit" />
        </div>
      </div>
    </template>

    <template v-else>
      <div class="agent-chat-content">
        <BubbleList class="agent-message-list" :list="bubbleList" max-height="100%" :auto-scroll="true">
          <template #content="{ item }">
            <div class="agent-message-body" :class="{ 'is-user': item.role === 'user' }">
              <div
                v-if="item.role !== 'user' && (item.replySourceTag || item.model || item.fallback || resolveMessageReplyTime(item))"
                class="agent-message-meta"
              >
                <span
                  v-if="item.replySourceTag"
                  class="agent-message-meta__tag"
                  :class="resolveTagClass(item.replySourceTag.tone)"
                >
                  {{ item.replySourceTag.text }}
                </span>
                <span v-if="item.model" class="agent-message-meta__model">{{ item.model }}</span>
                <span v-if="resolveMessageReplyTime(item)" class="agent-message-meta__time">{{ resolveMessageReplyTime(item) }}</span>
              </div>
              <el-collapse
                v-if="item.role !== 'user' && resolveVisibleTools(item).length"
                class="agent-message-tools"
                :model-value="resolveActiveToolKeys(item)"
                @update:model-value="value => handleToolCollapseUpdate(item, value)"
              >
                <el-collapse-item
                  v-for="(tool, toolIndex) in resolveVisibleTools(item)"
                  :key="resolveToolKey(tool, toolIndex)"
                  :name="resolveToolKey(tool, toolIndex)"
                >
                  <template #title>
                    <span class="agent-message-tool-title">
                      <span class="agent-message-tool-title__main">
                        <span class="agent-message-tool-title__icon">
                          <el-icon><Link /></el-icon>
                        </span>
                        <span class="agent-message-tool-title__text">
                          <span class="agent-message-tool-title__label">{{ resolveToolTitle(tool) }}</span>
                          <span v-if="resolveToolName(tool)" class="agent-message-tool-title__name">{{ resolveToolName(tool) }}</span>
                        </span>
                      </span>
                      <span class="agent-message-tool-title__status" :class="resolveToolStatusClass(tool.status)">
                        {{ resolveToolStatusText(tool.status) }}
                        <el-icon v-if="!isToolFailed(tool.status)"><Check /></el-icon>
                      </span>
                      <button
                        class="agent-message-tool-copy"
                        type="button"
                        aria-label="复制请求报文"
                        @click.stop="handleCopyToolRequest(tool)"
                      >
                        <el-icon><CopyDocument /></el-icon>
                      </button>
                    </span>
                  </template>
                  <div class="agent-message-tool-payload">
                    <section class="agent-message-tool-payload__section">
                      <div class="agent-message-tool-payload__label">REQUEST</div>
                      <pre>{{ formatToolRequest(tool) }}</pre>
                    </section>
                    <section class="agent-message-tool-payload__section">
                      <div class="agent-message-tool-payload__label">RESPONSE</div>
                      <pre>{{ formatToolResponse(tool) }}</pre>
                    </section>
                  </div>
                </el-collapse-item>
              </el-collapse>
              <div
                class="agent-message-content"
                :class="{
                  'is-thinking': item.progressState === 'streaming',
                  'is-failed-ai': isAIFailedMessage(item),
                  'is-user': item.role === 'user'
                }"
              >
                <div v-if="isAIFailedMessage(item)" class="agent-message-error">
                  <div class="agent-message-error__title">服务器异常，建议稍后重试</div>
                  <div class="agent-message-error__content">{{ resolveAIErrorMessage(item) }}</div>
                  <el-collapse v-if="item.fallback_reason" class="agent-message-error__detail" accordion>
                    <el-collapse-item title="错误详情" :name="String(item.id)">
                      <pre>{{ item.fallback_reason }}</pre>
                    </el-collapse-item>
                  </el-collapse>
                </div>
                <template v-else-if="item.role !== 'user'">
                  <AiMarkdown :content="item.content" :streaming="item.progressState === 'streaming'" />
                  <el-collapse v-if="item.fallback_reason" class="agent-message-error__detail" accordion>
                    <el-collapse-item title="错误详情" :name="String(item.id)">
                      <pre>{{ item.fallback_reason }}</pre>
                    </el-collapse-item>
                  </el-collapse>
                </template>
                <span v-else>{{ item.content }}</span>
                <span v-if="item.progressState === 'streaming'" class="agent-thinking-dots"> <i></i><i></i><i></i> </span>
              </div>
              <div v-if="item.attachments?.length" class="agent-message-attachments">
                <Attachments :items="buildMessageAttachmentItems(item.attachments)" overflow="wrap" :hide-upload="true" />
              </div>
            </div>
          </template>
          <template #footer="{ item }">
            <div
              v-message-footer-width
              class="agent-message-footer"
              :class="{ 'is-user': item.role === 'user', 'is-editing': isEditingMessage(item) }"
            >
              <div class="agent-message-actions" :class="{ 'is-user': item.role === 'user' }">
                <template v-for="action in resolveMessageActions(item)" :key="action.key">
                  <el-popconfirm
                    v-if="shouldConfirmMessageAction(action.key)"
                    :title="resolveActionConfirmTitle(action.key, item)"
                    confirm-button-text="确定"
                    cancel-button-text="取消"
                    width="220"
                    @confirm="handleMessageAction(action.key, item)"
                  >
                    <template #reference>
                      <button
                        class="agent-message-action"
                        type="button"
                        :disabled="sending || item.progressState === 'streaming'"
                        :aria-label="action.label"
                      >
                        <el-icon><component :is="action.icon" /></el-icon>
                      </button>
                    </template>
                  </el-popconfirm>
                  <el-tooltip v-else :content="action.label" placement="top">
                    <button
                      class="agent-message-action"
                      type="button"
                      :disabled="sending || item.progressState === 'streaming'"
                      :aria-label="action.label"
                      @click="handleMessageAction(action.key, item)"
                    >
                      <el-icon><component :is="action.icon" /></el-icon>
                    </button>
                  </el-tooltip>
                </template>
                <el-popover
                  v-if="item.role !== 'user' && hasAIUsage(item)"
                  popper-class="agent-message-runtime-popover"
                  placement="top"
                  trigger="hover"
                  width="260"
                >
                  <template #reference>
                    <button
                      class="agent-message-action agent-message-runtime-trigger"
                      type="button"
                      aria-label="查看运行明细"
                      :disabled="item.progressState === 'streaming'"
                    >
                      <el-icon><DataAnalysis /></el-icon>
                      <span v-if="resolveTokenTotal(item) > 0">{{ formatCompactNumber(resolveTokenTotal(item)) }}</span>
                      <span v-if="item.duration_ms > 0">{{ formatDurationMs(item.duration_ms) }}</span>
                    </button>
                  </template>
                  <div class="agent-runtime-detail">
                    <div class="agent-runtime-detail__title">运行明细</div>
                    <div v-if="resolveTokenTotal(item) > 0" class="agent-runtime-detail__section">
                      <div class="agent-runtime-detail__section-title">Token</div>
                      <div class="agent-runtime-detail__row">
                        <span>输入</span>
                        <strong>{{ formatNumber(item.token?.input) }}</strong>
                      </div>
                      <div class="agent-runtime-detail__row">
                        <span>输出</span>
                        <strong>{{ formatNumber(item.token?.output) }}</strong>
                      </div>
                      <div v-if="(item.token?.cache ?? 0) > 0" class="agent-runtime-detail__row">
                        <span>缓存读取</span>
                        <strong>{{ formatNumber(item.token?.cache) }}</strong>
                      </div>
                      <div class="agent-runtime-detail__row is-total">
                        <span>总计</span>
                        <strong>{{ formatNumber(item.token?.total) }}</strong>
                      </div>
                    </div>
                    <div v-if="item.first_token_ms > 0 || item.duration_ms > 0" class="agent-runtime-detail__section">
                      <div class="agent-runtime-detail__section-title">耗时</div>
                      <div v-if="item.first_token_ms > 0" class="agent-runtime-detail__row">
                        <span>首 Token</span>
                        <strong>{{ formatDurationMs(item.first_token_ms) }}</strong>
                      </div>
                      <div v-if="item.duration_ms > 0" class="agent-runtime-detail__row is-total">
                        <span>总耗时</span>
                        <strong>{{ formatDurationMs(item.duration_ms) }}</strong>
                      </div>
                    </div>
                  </div>
                </el-popover>
              </div>
              <div v-if="isEditingMessage(item)" class="agent-message-edit">
                <el-input
                  v-model="editingContent"
                  class="agent-message-edit__input"
                  type="textarea"
                  resize="none"
                  :autosize="{ minRows: 3, maxRows: 8 }"
                  :disabled="sending"
                  @keydown.ctrl.enter.prevent="submitMessageEdit(item)"
                  @keydown.meta.enter.prevent="submitMessageEdit(item)"
                />
                <div class="agent-message-edit__footer">
                  <el-button size="small" :disabled="sending" @click="cancelMessageEdit">取消</el-button>
                  <el-button size="small" type="primary" :loading="sending" @click="submitMessageEdit(item)">发送</el-button>
                </div>
              </div>
            </div>
          </template>
        </BubbleList>
      </div>

      <div class="agent-composer-dock">
        <section v-if="hasShortcutPanel" class="agent-shortcuts agent-shortcuts--compact">
          <div class="agent-shortcuts__head">
            <div>
              <div class="agent-shortcuts__eyebrow">快捷操作</div>
              <div class="agent-shortcuts__title">继续调用常用流程</div>
            </div>
            <div class="agent-shortcuts__actions">
              <span class="agent-shortcuts__count">{{ shortcutBatchLabel }}</span>
              <button
                v-if="canRefreshShortcutBatch"
                class="agent-shortcuts__refresh"
                type="button"
                :disabled="sending"
                @click="refreshShortcutBatch"
              >
                <span>换一批</span>
                <el-icon><Refresh /></el-icon>
              </button>
            </div>
          </div>
          <div v-if="loadingShortcuts" class="agent-shortcuts__loading">正在加载快捷入口...</div>
          <div v-else class="agent-shortcuts__grid">
            <button
              v-for="shortcut in shortcutVisibleList"
              :key="shortcut.key"
              class="agent-shortcut"
              type="button"
              :disabled="sending"
              @click="handleShortcutClick(shortcut)"
            >
              <span class="agent-shortcut__icon">
                <el-icon><component :is="resolveShortcutIcon(shortcut)" /></el-icon>
              </span>
              <span class="agent-shortcut__content">
                <span class="agent-shortcut__title">{{ shortcut.title }}</span>
                <span class="agent-shortcut__meta">{{ resolveShortcutMeta(shortcut) }}</span>
              </span>
            </button>
          </div>
        </section>
        <XSender :key="senderKey" :sending="sending" @submit="handleSubmit" />
      </div>
    </template>
  </main>
</template>

<script setup lang="ts" name="ChatPanel">
import { computed, defineAsyncComponent, h, nextTick, ref } from "vue";
import type { Component, ObjectDirective } from "vue";
import { Attachments, BubbleList } from "vue-element-plus-x";
import type { FilesCardProps } from "vue-element-plus-x/types/FilesCard";
import { Check, CopyDocument, DataAnalysis, Delete, EditPen, Link, Memo, PieChart, Refresh, User } from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import {
  type AiAttachment,
  type AiSession
} from "@/rpc/base/v1/ai_session";
import type { AiShortcut, AiToolCall } from "@/rpc/base/v1/ai_tool";
import type { AiAction } from "@/rpc/base/v1/ai_message";
import { AiMessageStatus } from "@/rpc/common/v1/enum";
import XSender from "./XSender.vue";

// AI Markdown 渲染器依赖较重，仅在真正出现助手消息时再加载。
const AiMarkdown = defineAsyncComponent(() => import("./AiMarkdown.vue"));
import { buildAIAttachmentFileCard } from "../attachment";
import type { ChatMessageAction, ChatMessageEditPayload, ChatMessageItem, ReplySourceTag, SubmitPayload } from "../types";

/** 消息操作按钮配置。 */
type MessageActionOption = {
  /** 操作类型。 */
  key: ChatMessageAction;
  /** 悬浮提示和无障碍文案。 */
  label: string;
  /** 操作图标组件。 */
  icon: Component;
};

/** 消息底部操作栏宽度同步状态。 */
type MessageFooterWidthState = {
  /** 监听上方内容卡片宽度变化。 */
  observer: ResizeObserver;
  /** 当前 footer 对应的内容卡片元素。 */
  contentEl: HTMLElement;
};

/** 工具折叠面板的 Element Plus 受控值类型。 */
type ToolCollapseValue = string | number | Array<string | number>;

const messageFooterWidthStateMap = new WeakMap<HTMLElement, MessageFooterWidthState>();
const activeToolKeyMap = ref<Record<string, string[]>>({});
const editingMessageKey = ref("");
const editingContent = ref("");
/** 助手消息最小阅读宽度，保证底部操作和运行信息不互相遮挡。 */
const AI_MESSAGE_MIN_WIDTH = 360;
/** 快捷入口每批展示数量，和移动端保持一致，避免大量入口挤压输入区。 */
const SHORTCUT_BATCH_SIZE = 4;
const shortcutBatchIndex = ref(0);

/** 朗读图标，按产品示意图绘制为喇叭声波。 */
const SpeakActionIcon = defineAsyncComponent(() =>
  Promise.resolve({
    name: "SpeakActionIcon",
    render() {
      return h(
        "svg",
        {
          class: "agent-message-action__custom-icon",
          xmlns: "http://www.w3.org/2000/svg",
          viewBox: "80 160 864 704",
          fill: "none",
          "aria-hidden": "true"
        },
        [
          h("path", {
            d: "M128 400h144L512 224v576L272 624H128z",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linejoin": "round"
          }),
          h("path", {
            d: "M640 352a224 224 0 0 1 0 320",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round"
          }),
          h("path", {
            d: "M768 224a384 384 0 0 1 0 576",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round"
          })
        ]
      );
    }
  })
);

/** 分支图标，按产品示意图绘制为向上分叉箭头。 */
const BranchActionIcon = defineAsyncComponent(() =>
  Promise.resolve({
    name: "BranchActionIcon",
    render() {
      return h(
        "svg",
        {
          class: "agent-message-action__custom-icon",
          xmlns: "http://www.w3.org/2000/svg",
          viewBox: "96 160 832 704",
          fill: "none",
          "aria-hidden": "true"
        },
        [
          h("path", {
            d: "M256 832V192",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round"
          }),
          h("path", {
            d: "M256 192 128 320M256 192l128 128",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round",
            "stroke-linejoin": "round"
          }),
          h("path", {
            d: "M384 640c256 0 384-176 384-448",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round"
          }),
          h("path", {
            d: "M768 192 640 320M768 192l128 128",
            stroke: "currentColor",
            "stroke-width": "64",
            "stroke-linecap": "round",
            "stroke-linejoin": "round"
          })
        ]
      );
    }
  })
);

const props = defineProps<{
  /** 当前活动会话，用于输入框按会话重建。 */
  activeSession?: AiSession;
  /** 当前会话消息列表。 */
  messages: ChatMessageItem[];
  /** 消息发送加载状态。 */
  sending: boolean;
  /** 空态快捷入口加载状态。 */
  loadingShortcuts?: boolean;
  /** 当前终端可用快捷入口。 */
  shortcuts?: AiShortcut[];
}>();

const emit = defineEmits<{
  /** 提交输入框内容。 */
  submit: [payload: SubmitPayload];
  /** 触发消息级操作。 */
  messageAction: [payload: { action: ChatMessageAction; item: ChatMessageItem }];
  /** 提交当前用户消息的文本编辑。 */
  messageEdit: [payload: ChatMessageEditPayload];
}>();

const isEmptyState = computed(() => props.messages.length === 0);

const senderKey = computed(() => props.activeSession?.id || "empty-session");

const bubbleList = computed<ChatMessageItem[]>(() =>
  (props.messages ?? []).map(item => ({
    ...item,
    key: String(item.key ?? item.id ?? `${item.role}-${item.created_at?.seconds ?? 0}`),
    content: String(item.content ?? ""),
    placement: item.placement ?? "start"
  }))
);

const lastEditableUserMessageKey = computed(() => {
  for (let index = props.messages.length - 1; index >= 0; index--) {
    const item = props.messages[index];
    if (item.role === "user" && !item.localOnly) {
      return resolveMessageEditKey(item);
    }
  }
  return "";
});

const welcomeTitle = computed(() => {
  const hour = new Date().getHours();
  if (hour < 12) return "上午好，我是通用 AI 助手";
  if (hour < 18) return "下午好，我是通用 AI 助手";
  return "晚上好，我是通用 AI 助手";
});

const shortcutAllList = computed(() =>
  [...(props.shortcuts ?? [])]
    .filter(item => Boolean(item?.key && (item.title || item.prompt)))
    .sort((left, right) => Number(left.sort ?? 0) - Number(right.sort ?? 0))
);

const shortcutBatchCount = computed(() => Math.max(1, Math.ceil(shortcutAllList.value.length / SHORTCUT_BATCH_SIZE)));

const shortcutVisibleList = computed(() => {
  const pageIndex = shortcutBatchIndex.value % shortcutBatchCount.value;
  const start = pageIndex * SHORTCUT_BATCH_SIZE;
  return shortcutAllList.value.slice(start, start + SHORTCUT_BATCH_SIZE);
});

const hasShortcutPanel = computed(() => props.loadingShortcuts || shortcutVisibleList.value.length > 0);

const canRefreshShortcutBatch = computed(() => shortcutAllList.value.length > SHORTCUT_BATCH_SIZE);

const shortcutBatchLabel = computed(() => {
  const total = shortcutAllList.value.length;
  if (props.loadingShortcuts && total === 0) return "加载中";
  if (total <= SHORTCUT_BATCH_SIZE) return `${total} 项`;
  const pageIndex = shortcutBatchIndex.value % shortcutBatchCount.value;
  const start = pageIndex * SHORTCUT_BATCH_SIZE + 1;
  const end = Math.min(start + SHORTCUT_BATCH_SIZE - 1, total);
  return `${start}-${end} / ${total}`;
});

/** 查找 footer 所属气泡里的消息内容元素。 */
function findMessageContentElement(el: HTMLElement) {
  const wrapper = el.closest(".elx-bubble__content-wrapper");
  return wrapper?.querySelector<HTMLElement>(".agent-message-body") ?? wrapper?.querySelector<HTMLElement>(".elx-bubble__content") ?? null;
}

/** 把消息 footer 宽度同步为上方内容宽度，避免底部按钮越出正文边界。 */
function syncMessageFooterWidth(el: HTMLElement) {
  const contentEl = findMessageContentElement(el);
  if (!contentEl) return;

  const isAIMessage = !el.classList.contains("is-user");
  const width = Math.max(contentEl.getBoundingClientRect().width, isAIMessage ? AI_MESSAGE_MIN_WIDTH : 0);
  if (width <= 0) return;

  el.style.setProperty("--agent-message-footer-width", `${Math.ceil(width)}px`);
}

/** 绑定内容尺寸监听，覆盖流式内容、Markdown 渲染和窗口变化后的重新对齐。 */
function bindMessageFooterWidth(el: HTMLElement) {
  const contentEl = findMessageContentElement(el);
  const existingState = messageFooterWidthStateMap.get(el);
  if (!contentEl) {
    existingState?.observer.disconnect();
    messageFooterWidthStateMap.delete(el);
    el.style.removeProperty("--agent-message-footer-width");
    return;
  }

  if (existingState?.contentEl === contentEl) {
    syncMessageFooterWidth(el);
    return;
  }

  existingState?.observer.disconnect();
  const observer = new ResizeObserver(() => syncMessageFooterWidth(el));
  observer.observe(contentEl);
  messageFooterWidthStateMap.set(el, { observer, contentEl });
  syncMessageFooterWidth(el);
}

const vMessageFooterWidth: ObjectDirective<HTMLElement> = {
  mounted(el) {
    void nextTick(() => bindMessageFooterWidth(el));
  },
  updated(el) {
    void nextTick(() => bindMessageFooterWidth(el));
  },
  beforeUnmount(el) {
    messageFooterWidthStateMap.get(el)?.observer.disconnect();
    messageFooterWidthStateMap.delete(el);
  }
};

/** 读取输入框内容并提交给父组件。 */
function handleSubmit(payload: SubmitPayload) {
  emit("submit", payload);
}

/** 点击快捷入口时提交对应入口动作。 */
function handleShortcutClick(shortcut: AiShortcut) {
  emit("submit", {
    text: shortcut.prompt || shortcut.title,
    attachments: [],
    action: buildShortcutAction(shortcut)
  });
}

/** 轮换下一批快捷入口，保持大量入口也能被逐批访问。 */
function refreshShortcutBatch() {
  if (!canRefreshShortcutBatch.value) return;
  shortcutBatchIndex.value = (shortcutBatchIndex.value + 1) % shortcutBatchCount.value;
}

/** 根据快捷入口的通用类型给出稳定的轻量图标。 */
function resolveShortcutIcon(shortcut: AiShortcut) {
  const text = `${shortcut.group || ""} ${shortcut.title || ""} ${shortcut.key || ""}`.toLowerCase();
  if (text.includes("user") || text.includes("用户")) return User;
  if (text.includes("report") || text.includes("统计") || text.includes("分析")) return PieChart;
  return Memo;
}

/** 展示快捷入口的分组和工具依赖摘要，帮助运营判断动作上下文。 */
function resolveShortcutMeta(shortcut: AiShortcut) {
  const group = shortcut.group || "通用助手";
  const toolCount = shortcut.required_tools?.length ?? 0;
  return toolCount > 0 ? `${group} · ${toolCount} 个工具` : group;
}

/** 生成快捷入口对应的流程动作。 */
function buildShortcutAction(shortcut: AiShortcut) {
  if (!shortcut.action?.type) return undefined;
  return buildFlowAction(shortcut.action, parseActionPayload(shortcut.action.payload_json));
}

/** 构造发送给后端的流程动作，统一补齐 payload JSON 默认值。 */
function buildFlowAction(action?: Partial<AiAction>, payload?: Record<string, unknown>): AiAction | undefined {
  if (!action?.type) return undefined;
  return {
    flow: action.flow || "",
    step: action.step || "",
    type: action.type,
    payload_json: JSON.stringify(payload || {}),
    source_message_id: action.source_message_id || "",
    action_id: action.action_id || "",
    flow_version: Number(action.flow_version || 0)
  };
}

/** 解析流程动作负载，兼容空字符串和异常 JSON。 */
function parseActionPayload(payload?: string) {
  try {
    return JSON.parse(payload || "{}") as Record<string, unknown>;
  } catch {
    return {};
  }
}

/** 根据消息角色返回可用操作。 */
function resolveMessageActions(item: ChatMessageItem) {
  if (item.progressState === "streaming" || item.status === AiMessageStatus.GENERATING_AAMS) return [];

  const copyAction: MessageActionOption = { key: "copy" as const, label: "复制", icon: CopyDocument };
  const deleteAction: MessageActionOption = { key: "delete" as const, label: "删除", icon: Delete };
  const editAction: MessageActionOption = { key: "edit" as const, label: "编辑", icon: EditPen };
  if (item.role === "user") {
    const actions = isLastEditableUserMessage(item) ? [editAction, copyAction, deleteAction] : [copyAction, deleteAction];
    if (item.status === AiMessageStatus.FAILED_AAMS) {
      return item.localOnly
        ? [{ key: "retry" as const, label: "重新发送", icon: Refresh }, copyAction, deleteAction]
        : [{ key: "retry" as const, label: "重新发送", icon: Refresh }, ...actions];
    }
    return item.localOnly ? [copyAction, deleteAction] : actions;
  }

  const actions: MessageActionOption[] = [{ key: "retry" as const, label: "重新生成", icon: Refresh }];
  if (item.status === AiMessageStatus.SUCCESS_AAMS) {
    actions.push({ key: "branch" as const, label: "从此处创建分支会话", icon: BranchActionIcon });
    actions.push({ key: "speak" as const, label: item.speaking ? "停止朗读" : "朗读", icon: SpeakActionIcon });
  }
  return [
    ...actions,
    copyAction,
    deleteAction
  ];
}

/** 判断是否为需要渲染错误卡片的助手失败消息。 */
function isAIFailedMessage(item: ChatMessageItem) {
  return item.role !== "user" && item.status === AiMessageStatus.FAILED_AAMS;
}

/** 返回助手错误摘要，优先展示服务端可读错误。 */
function resolveAIErrorMessage(item: ChatMessageItem) {
  const content = String(item.content ?? "").trim();
  return content || "这次回复没有成功返回，你可以直接重试刚才的问题。";
}

/** 判断助手消息是否存在最终用量信息。 */
function hasAIUsage(item: ChatMessageItem) {
  return resolveTokenTotal(item) > 0 || item.first_token_ms > 0 || item.duration_ms > 0;
}

/** 返回当前气泡总 token 数。 */
function resolveTokenTotal(item: ChatMessageItem) {
  return Number(item.token?.total ?? 0);
}

/** 将毫秒耗时格式化为秒。 */
function formatDurationMs(value?: number) {
  const ms = Number(value ?? 0);
  if (ms <= 0) return "0s";
  return `${(ms / 1000).toFixed(2)}s`;
}

/** 返回助手消息回复时间，优先展示后端消息创建时间。 */
function resolveMessageReplyTime(item: ChatMessageItem) {
  const timestamp = resolveMessageCreatedAt(item);
  if (!timestamp) return "";
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false
  }).format(new Date(timestamp));
}

/** 解析消息创建时间戳，兼容 protobuf Timestamp 结构。 */
function resolveMessageCreatedAt(item: ChatMessageItem) {
  const seconds = Number(item.created_at?.seconds ?? 0);
  const nanos = Number(item.created_at?.nanos ?? 0);
  if (seconds <= 0 && nanos <= 0) return 0;
  return seconds * 1000 + Math.floor(nanos / 1_000_000);
}

/** 将较大的数字压缩成适合 footer 展示的短格式。 */
function formatCompactNumber(value?: number) {
  const number = Number(value ?? 0);
  if (number >= 1_000_000) return `${(number / 1_000_000).toFixed(number >= 10_000_000 ? 0 : 1)}M`;
  if (number >= 1_000) return `${(number / 1_000).toFixed(number >= 100_000 ? 0 : 1)}K`;
  return formatNumber(number);
}

/** 格式化明细数字，保持 Token 明细易读。 */
function formatNumber(value?: number) {
  return new Intl.NumberFormat("zh-CN").format(Number(value ?? 0));
}

/** 生成工具标签稳定键。 */
function resolveToolKey(tool: AiToolCall, index?: number) {
  return `${tool.type || "tool"}:${tool.name || tool.title}:${index ?? 0}`;
}

/** 生成消息维度的工具展开状态键，避免不同气泡之间互相影响。 */
function resolveToolCollapseMessageKey(item: ChatMessageItem) {
  return String(item.key ?? item.id ?? "");
}

/** 返回当前消息已展开的工具面板，默认保持全部收起。 */
function resolveActiveToolKeys(item: ChatMessageItem) {
  return activeToolKeyMap.value[resolveToolCollapseMessageKey(item)] ?? [];
}

/** 更新工具面板展开状态，兼容 Element Plus 单值和数组两类返回值。 */
function handleToolCollapseUpdate(item: ChatMessageItem, value: ToolCollapseValue) {
  activeToolKeyMap.value = {
    ...activeToolKeyMap.value,
    [resolveToolCollapseMessageKey(item)]: Array.isArray(value) ? value.map(String) : [String(value)].filter(Boolean)
  };
}

/** 返回需要展示在消息内容区的 function 工具记录，过滤联网搜索等 server 工具。 */
function resolveVisibleTools(item: ChatMessageItem) {
  return (item.tools ?? []).filter(tool => tool.type === "function" && String(tool.name || tool.title).trim());
}

/** 生成工具展示名称。 */
function resolveToolTitle(tool: AiToolCall) {
  return tool.title || tool.name || "工具";
}

/** 返回工具真实名称，用于排障时识别具体 Agent Tool。 */
function resolveToolName(tool: AiToolCall) {
  return String(tool.name ?? "").trim();
}

/** 生成工具状态样式名。 */
function resolveToolStatusClass(status?: string) {
  return isToolFailed(status) ? "is-error" : "is-success";
}

/** 生成工具状态展示文案。 */
function resolveToolStatusText(status?: string) {
  return isToolFailed(status) ? "失败" : "已完成";
}

/** 判断工具调用是否失败，兼容后端历史状态值。 */
function isToolFailed(status?: string) {
  return ["error", "failed", "fail"].includes(String(status ?? "").toLowerCase());
}

/** 判断原始报文是否有实际内容，避免空对象撑出无意义区块。 */
function hasToolPayloadValue(payload?: string) {
  const value = String(payload ?? "").trim();
  if (!value || value === "{}") return false;
  return true;
}

/** 格式化工具原始报文，JSON 内容优先缩进展示。 */
function formatToolPayload(payload?: string) {
  const value = String(payload ?? "");
  if (!value) return "{}";
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

/** 格式化工具请求报文，展开态始终保留稳定内容区。 */
function formatToolRequest(tool: AiToolCall) {
  const input = String(tool.input ?? "");
  if (!hasToolPayloadValue(input)) return "{}";
  if (!isJSONLikeText(input) && !hasToolPayloadValue(tool.output)) return "{}";
  return formatToolArguments(input);
}

/** 格式化工具出参，兼容历史结果落在 input 的消息。 */
function formatToolResponse(tool: AiToolCall) {
  if (hasToolPayloadValue(tool.output)) return formatToolPayload(tool.output);
  if (hasToolPayloadValue(tool.input) && !isJSONLikeText(tool.input)) return formatToolPayload(tool.input);
  return "{}";
}

/** 判断文本是否像 JSON，工具请求报文正常应由模型参数 JSON 构成。 */
function isJSONLikeText(payload?: string) {
  const value = String(payload ?? "");
  if (!value) return false;
  const first = value[0];
  return first === "{" || first === "[";
}

/** 格式化工具入参，简单对象按字段行展示，贴近工具调用详情的阅读习惯。 */
function formatToolArguments(payload?: string) {
  const value = String(payload ?? "");
  if (!value) return "";
  try {
    const parsed = JSON.parse(value) as unknown;
    if (!isPlainRecord(parsed)) return JSON.stringify(parsed, null, 2);
    return Object.entries(parsed)
      .map(([key, item]) => `${key} ${formatToolArgumentValue(item)}`)
      .join("\n");
  } catch {
    return value;
  }
}

/** 判断值是否为普通对象，避免把数组按入参表展示。 */
function isPlainRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

/** 格式化单个工具入参值，复杂值保留 JSON 结构。 */
function formatToolArgumentValue(value: unknown) {
  if (isPlainRecord(value) || Array.isArray(value)) return JSON.stringify(value);
  return String(value ?? "");
}

/** 复制工具请求报文，点击时不触发展开收起。 */
async function handleCopyToolRequest(tool: AiToolCall) {
  try {
    await navigator.clipboard.writeText(formatToolRequest(tool));
    ElMessage.success("请求报文已复制");
  } catch {
    ElMessage.error("当前浏览器不支持复制");
  }
}

/** 向父组件透传消息操作，由页面层决定是否重发、复制或移除。 */
function handleMessageAction(action: ChatMessageAction, item: ChatMessageItem) {
  if (action === "edit") {
    startMessageEdit(item);
    return;
  }
  emit("messageAction", { action, item });
}

/** 生成编辑态稳定键，用于只展开当前用户消息下方的编辑框。 */
function resolveMessageEditKey(item: ChatMessageItem) {
  return String(item.key ?? `${item.id}:${item.role}`);
}

/** 判断当前消息是否正在编辑。 */
function isEditingMessage(item: ChatMessageItem) {
  return editingMessageKey.value === resolveMessageEditKey(item);
}

/** 判断用户消息是否为当前会话最后一轮可编辑消息。 */
function isLastEditableUserMessage(item: ChatMessageItem) {
  return item.role === "user" && !item.localOnly && lastEditableUserMessageKey.value === resolveMessageEditKey(item);
}

/** 开始编辑用户消息正文，附件保持原样不进入编辑区。 */
function startMessageEdit(item: ChatMessageItem) {
  if (!isLastEditableUserMessage(item)) return;
  editingMessageKey.value = resolveMessageEditKey(item);
  editingContent.value = String(item.content ?? "");
}

/** 取消当前消息文本编辑。 */
function cancelMessageEdit() {
  editingMessageKey.value = "";
  editingContent.value = "";
}

/** 提交当前消息文本编辑，并交给页面层重新生成同一轮输出。 */
function submitMessageEdit(item: ChatMessageItem) {
  if (props.sending) return;
  const content = editingContent.value;
  if (content === "") {
    ElMessage.warning("消息内容不能为空");
    return;
  }
  if (content === String(item.content ?? "")) {
    ElMessage.info("消息内容没有变化");
    return;
  }
  emit("messageEdit", { item, content });
  cancelMessageEdit();
}

/** 判断消息操作是否需要二次确认。 */
function shouldConfirmMessageAction(action: ChatMessageAction) {
  return action === "retry" || action === "delete";
}

/** 返回消息操作确认文案，避免误删或误覆盖当前气泡。 */
function resolveActionConfirmTitle(action: ChatMessageAction, item: ChatMessageItem) {
  if (action === "delete") return "确认删除当前消息？";
  if (item.role === "user") return "确认重新发送当前消息？";
  return "重新生成会覆盖当前消息";
}

/** 统一回复来源标签配色。 */
function resolveTagClass(tone?: ReplySourceTag["tone"]) {
  return tone ? `is-${tone}` : "";
}

/** 构建消息附件卡片，统一交给 Attachments / FilesCard 处理图片预览。 */
function buildMessageAttachmentItems(attachments: AiAttachment[]): FilesCardProps[] {
  return attachments.map(attachment =>
    buildAIAttachmentFileCard(attachment, {
      maxWidth: "240px"
    })
  );
}

</script>

<style scoped lang="scss">
.agent-chat-panel {
  position: relative;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  min-width: 0;
  height: 100%;
  min-height: 0;
  padding: 20px 0 24px;
  overflow: hidden;
  background: var(--admin-page-card-bg);
}
.agent-chat-panel.is-empty {
  justify-content: center;
  padding: 0;
}
.agent-chat-content {
  display: flex;
  flex: 1;
  width: min(1180px, calc(100% - 72px));
  min-height: 0;
  padding-bottom: 168px;
  margin: 0 auto;
  overflow: hidden;
}
.agent-chat-panel.has-shortcuts .agent-chat-content {
  padding-bottom: 262px;
}
.agent-chat-empty {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: min(1100px, calc(100% - 96px));
  min-height: 0;
  margin: 0 auto;
}
.agent-chat-empty__title {
  font-size: clamp(28px, 4vw, 44px);
  font-weight: 700;
  line-height: 1.25;
  color: var(--admin-page-text-primary);
  text-align: center;
}
.agent-chat-empty__desc {
  margin-top: 14px;
  font-size: 14px;
  line-height: 28px;
  color: var(--admin-page-text-secondary);
  text-align: center;
}
.agent-chat-empty__sender {
  width: min(980px, 100%);
  margin-top: 28px;
}
.agent-shortcuts {
  box-sizing: border-box;
  width: min(760px, 100%);
  padding: 14px;
  margin-top: 26px;
  background:
    linear-gradient(180deg, rgba(64, 158, 255, 0.06), rgba(255, 255, 255, 0)),
    var(--admin-page-card-bg);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius);
  box-shadow: 0 12px 28px rgba(15, 23, 42, 0.06);
}
.agent-shortcuts__head {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
  min-width: 0;
  margin-bottom: 12px;
}
.agent-shortcuts__actions {
  display: inline-flex;
  flex: 0 0 auto;
  gap: 8px;
  align-items: center;
}
.agent-shortcuts__eyebrow {
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  color: var(--el-color-primary);
}
.agent-shortcuts__title {
  margin-top: 2px;
  font-size: 15px;
  font-weight: 700;
  line-height: 22px;
  color: var(--admin-page-text-primary);
}
.agent-shortcuts__count {
  flex: 0 0 auto;
  padding: 2px 8px;
  font-size: 12px;
  line-height: 20px;
  color: var(--admin-page-text-secondary);
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius);
}
.agent-shortcuts__refresh {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  height: 26px;
  padding: 0 8px;
  font-size: 12px;
  line-height: 1;
  color: var(--el-color-primary);
  cursor: pointer;
  background: var(--el-color-primary-light-9);
  border: 1px solid var(--el-color-primary-light-7);
  border-radius: var(--admin-page-radius);
}
.agent-shortcuts__refresh:hover {
  border-color: var(--el-color-primary-light-5);
}
.agent-shortcuts__refresh:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.agent-shortcuts__refresh .el-icon {
  font-size: 13px;
}
.agent-shortcuts__loading {
  box-sizing: border-box;
  min-height: 70px;
  padding: 23px 12px;
  font-size: 13px;
  line-height: 22px;
  color: var(--admin-page-text-secondary);
  text-align: center;
  background: var(--el-fill-color-lighter);
  border-radius: var(--admin-page-radius);
}
.agent-shortcuts__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  min-width: 0;
}
.agent-shortcut {
  box-sizing: border-box;
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr);
  gap: 10px;
  align-items: center;
  min-width: 0;
  min-height: 70px;
  padding: 12px;
  color: var(--admin-page-text-primary);
  text-align: left;
  cursor: pointer;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius);
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    transform 0.18s ease;
}
.agent-shortcut:hover {
  border-color: var(--el-color-primary-light-5);
  box-shadow: 0 8px 18px rgba(64, 158, 255, 0.12);
  transform: translateY(-1px);
}
.agent-shortcut:disabled {
  cursor: not-allowed;
  opacity: 0.55;
  transform: none;
}
.agent-shortcut__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  font-size: 12px;
  font-weight: 700;
  line-height: 1;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border: 1px solid var(--el-color-primary-light-7);
  border-radius: var(--admin-page-radius);
}
.agent-shortcut__content {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.agent-shortcut__title {
  min-width: 0;
  overflow: hidden;
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.agent-shortcut__meta {
  min-width: 0;
  overflow: hidden;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.agent-message-list {
  flex: 1;
  min-height: 0;
  padding: 8px 0 24px;
  overflow: auto;
  :deep(.elx-bubble__content-wrapper),
  :deep(.elx-bubble__content) {
    min-width: 0;
  }
  :deep(.elx-bubble--start .elx-bubble__content-wrapper .elx-bubble__footer) {
    box-sizing: border-box;
    width: fit-content;
    max-width: 100%;
    margin-top: 2px;
  }
  :deep(.elx-bubble__content-wrapper:hover .elx-bubble__footer),
  :deep(.elx-bubble__content-wrapper:focus-within .elx-bubble__footer) {
    pointer-events: auto;
    opacity: 1;
  }
  :deep(.elx-bubble__content-wrapper:hover .agent-message-footer),
  :deep(.elx-bubble__content-wrapper:focus-within .agent-message-footer) {
    pointer-events: auto;
    opacity: 1;
    visibility: visible;
  }
  :deep(.elx-bubble--start .elx-bubble__content-wrapper .elx-bubble__content) {
    box-sizing: border-box;
    width: min(720px, 100%);
    max-width: 100%;
    min-width: min(520px, 100%);
  }
  :deep(.elx-bubble--start .elx-bubble__content-wrapper .elx-bubble__content) {
    padding: 0;
    background: transparent;
    border: 0;
    box-shadow: none;
  }
  :deep(.elx-bubble--end .elx-bubble__content-wrapper .elx-bubble__content) {
    border-radius: var(--admin-page-radius);
  }
  :deep(.elx-bubble--end .elx-bubble__content-wrapper .elx-bubble__content--corner) {
    border-start-start-radius: var(--admin-page-radius);
    border-start-end-radius: var(--admin-page-radius);
  }
  :deep(.elx-bubble-list__boundary-content),
  :deep(.elx-bubble-list__embedded-item) {
    border-radius: var(--admin-page-radius);
  }
}
.agent-message-content {
  box-sizing: border-box;
  width: 100%;
  max-width: 100%;
  min-width: 0;
  line-height: 24px;
  margin: 6px 0 0;
  overflow-wrap: anywhere;
}
.agent-message-content:first-child {
  margin-top: 0;
}
.agent-message-content.is-user {
  width: fit-content;
  min-width: 0;
  white-space: pre-wrap;
}
.agent-message-content.is-thinking {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  :deep(.agent-markdown) {
    flex: 0 1 auto;
    width: auto;
  }
  :deep(.agent-markdown .x-md-core) {
    width: auto;
  }
  :deep(.agent-markdown p) {
    display: inline;
    margin: 0;
  }
}
.agent-message-content.is-failed-ai {
  width: min(720px, 100%);
  max-width: 100%;
}
.agent-message-body {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0;
  width: 100%;
  min-width: min(360px, 100%);
  max-width: 100%;
}
.agent-message-body.is-user {
  width: fit-content;
  min-width: 0;
}
.agent-message-meta {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  width: 100%;
  max-width: 100%;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}
.agent-message-meta__tag {
  padding: 2px 8px;
  background: var(--el-fill-color-light);
  border-radius: var(--admin-page-radius);
}
.agent-message-meta__tag.is-primary {
  color: var(--el-color-primary);
}
.agent-message-meta__tag.is-success {
  color: var(--el-color-success);
}
.agent-message-meta__tag.is-warning {
  color: var(--el-color-warning);
}
.agent-message-meta__tag.is-info {
  color: var(--admin-page-text-secondary);
}
.agent-message-meta__model {
  opacity: 0.85;
}
.agent-message-meta__time {
  opacity: 0.85;
}
.agent-message-error {
  box-sizing: border-box;
  width: 100%;
  max-width: 100%;
  padding: 12px 14px;
  color: var(--admin-page-text-primary);
  background: var(--el-color-danger-light-9);
  border: 1px solid var(--el-color-danger-light-7);
  border-radius: var(--admin-page-radius);
}
.agent-message-error__title {
  font-size: 14px;
  font-weight: 700;
  line-height: 22px;
  color: var(--el-color-danger);
}
.agent-message-error__content {
  margin-top: 6px;
  font-size: 13px;
  line-height: 22px;
  color: var(--admin-page-text-regular);
  word-break: break-word;
}
.agent-message-error__detail {
  margin-top: 8px;
  border: 0;
  :deep(.el-collapse-item__header) {
    height: 28px;
    font-size: 12px;
    color: var(--admin-page-text-secondary);
    background: transparent;
    border: 0;
  }
  :deep(.el-collapse-item__wrap) {
    background: transparent;
    border: 0;
  }
  :deep(.el-collapse-item__content) {
    padding-bottom: 0;
  }
  pre {
    max-height: 180px;
    padding: 8px;
    margin: 0;
    overflow: auto;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
    font-size: 12px;
    line-height: 18px;
    color: var(--admin-page-text-secondary);
    white-space: pre-wrap;
    word-break: break-word;
    background: var(--el-fill-color-blank);
    border: 1px solid var(--el-border-color-light);
    border-radius: var(--admin-page-radius);
  }
}
.agent-thinking-dots {
  display: inline-flex;
  flex: 0 0 auto;
  gap: 4px;
  align-items: center;
  color: var(--admin-page-text-secondary);
  i {
    display: inline-block;
    width: 6px;
    height: 6px;
    background: currentcolor;
    border-radius: 50%;
    animation: thinking-bounce 1.2s infinite ease-in-out;
  }
  i:nth-child(2) {
    animation-delay: 0.15s;
  }
  i:nth-child(3) {
    animation-delay: 0.3s;
  }
}
.agent-message-attachments {
  :deep(.elx-files-card) {
    border-radius: var(--admin-page-radius);
  }
  :deep(.elx-files-card-img),
  :deep(.elx-files-card__image-preview),
  :deep(.elx-files-card-delete-icon) {
    border-radius: var(--admin-page-radius);
  }
}
.agent-message-tools {
  width: 100%;
  min-width: 0;
  max-width: 100%;
  margin-top: 8px;
  border: 0;
  :deep(.el-collapse-item + .el-collapse-item) {
    margin-top: 8px;
  }
  :deep(.el-collapse-item) {
    position: relative;
    overflow: hidden;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: var(--admin-page-radius);
  }
  :deep(.el-collapse-item::after) {
    position: absolute;
    right: 0;
    bottom: 0;
    left: 0;
    z-index: 2;
    height: 1px;
    pointer-events: none;
    content: "";
    background: var(--el-border-color-lighter);
  }
  :deep(.el-collapse-item__header) {
    box-sizing: border-box;
    display: flex;
    min-width: 0;
    min-height: 44px;
    height: auto;
    padding: 0 12px;
    overflow: hidden;
    font-size: 13px;
    line-height: normal;
    color: var(--admin-page-text-regular);
    background: transparent;
    border: 0;
  }
  :deep(.el-collapse-item__arrow) {
    flex: 0 0 auto;
    margin-left: 10px;
    color: var(--admin-page-text-secondary);
  }
  :deep(.el-collapse-item__wrap) {
    overflow: hidden;
    background: transparent;
    border: 0;
  }
  :deep(.el-collapse-item__content) {
    padding: 0;
  }
}
.agent-message-tool-title {
  display: grid;
  grid-template-columns: minmax(0, 1fr) max-content max-content;
  gap: 8px;
  align-items: center;
  width: 100%;
  max-width: 100%;
  min-width: 0;
}
.agent-message-tool-title__main {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  gap: 8px;
  align-items: center;
  min-width: 0;
}
.agent-message-tool-title__text {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}
.agent-message-tool-title__label {
  min-width: 0;
  overflow: hidden;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.agent-message-tool-title__name {
  min-width: 0;
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 11px;
  line-height: 14px;
  color: var(--admin-page-text-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.agent-message-tool-title__icon {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  font-size: 14px;
  color: var(--admin-page-text-secondary);
}
.agent-message-tool-title__status {
  display: inline-flex;
  flex: 0 0 auto;
  gap: 4px;
  align-items: center;
  font-size: 12px;
  line-height: 20px;
  color: var(--el-color-success);
}
.agent-message-tool-title__status.is-error {
  color: var(--el-color-danger);
}
.agent-message-tool-copy {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    background-color 0.2s ease;
  &:hover {
    color: var(--el-color-primary);
    background: var(--el-fill-color-light);
  }
  .el-icon {
    font-size: 15px;
  }
}
.agent-message-tool-payload {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  min-width: 0;
  max-height: 220px;
  padding: 0 0 1px;
  overflow: auto;
  border-top: 1px solid var(--el-border-color-lighter);
}
.agent-message-tool-payload__section {
  box-sizing: border-box;
  min-width: 0;
  padding: 10px 12px;
}
.agent-message-tool-payload__section + .agent-message-tool-payload__section {
  border-top: 1px solid var(--el-border-color-lighter);
}
.agent-message-tool-payload__label {
  margin-bottom: 2px;
  font-size: 11px;
  font-weight: 700;
  line-height: 16px;
  letter-spacing: 0.02em;
  color: var(--admin-page-text-secondary);
}
.agent-message-tool-payload pre {
  box-sizing: border-box;
  padding: 0;
  margin: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 12px;
  line-height: 20px;
  color: var(--admin-page-text-regular);
  white-space: pre-wrap;
  word-break: break-word;
  background: transparent;
  border: 0;
}
.agent-message-footer {
  box-sizing: border-box;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 8px;
  align-items: center;
  width: var(--agent-message-footer-width, fit-content);
  max-width: 100%;
  min-width: min(360px, 100%);
  padding: 0;
  margin-top: 0;
  pointer-events: none;
  opacity: 0;
  transition:
    opacity 0.16s ease,
    visibility 0.16s ease;
  visibility: hidden;
}
.agent-message-footer:hover,
.agent-message-footer:focus-within,
.agent-message-footer.is-editing {
  pointer-events: auto;
  opacity: 1;
  visibility: visible;
}
.agent-message-footer.is-editing {
  grid-template-columns: minmax(0, 1fr);
  row-gap: 6px;
}
.agent-message-footer.is-user {
  grid-template-columns: auto;
  justify-content: flex-end;
  min-width: 0;
  padding: 0;
}
.agent-message-footer.is-user.is-editing {
  grid-template-columns: minmax(0, 1fr);
  justify-content: stretch;
  justify-self: end;
  width: min(380px, 100%);
  min-width: min(320px, 100%);
}
.agent-message-actions {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  justify-content: flex-start;
  min-width: 0;
}
.agent-message-actions.is-user {
  justify-content: flex-end;
}
.agent-message-edit {
  box-sizing: border-box;
  width: 100%;
  min-width: min(320px, 100%);
  padding: 10px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--admin-page-radius);
}
.agent-message-edit__input {
  width: 100%;
  :deep(.el-textarea__inner) {
    min-height: 82px !important;
    padding: 8px 10px;
    font-size: 14px;
    line-height: 24px;
    color: var(--admin-page-text-primary);
    background: var(--admin-page-card-bg);
    border-radius: var(--admin-page-radius);
    box-shadow: 0 0 0 1px var(--el-border-color-light) inset;
  }
}
.agent-message-edit__footer {
  display: flex;
  gap: 8px;
  justify-content: flex-end;
  margin-top: 8px;
}
.agent-message-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    background-color 0.2s ease,
    opacity 0.2s ease;
  &:hover {
    color: var(--el-color-primary);
    background: var(--el-fill-color-light);
  }
  &:disabled {
    cursor: not-allowed;
    opacity: 0.45;
  }
}
.agent-message-action :deep(.el-icon) {
  width: 1em;
  height: 1em;
  font-size: 16px;
  line-height: 1;
}
.agent-message-action :deep(svg) {
  display: block;
  width: 1em;
  height: 1em;
}
.agent-message-action :deep(.agent-message-action__custom-icon) {
  width: 16px;
  height: 16px;
}
.agent-message-action.agent-message-runtime-trigger {
  flex: 0 0 auto;
  gap: 6px;
  width: fit-content;
  min-width: 0;
  max-width: 150px;
  padding: 0 6px;
  overflow: hidden;
  font-size: 12px;
  :deep(.el-icon) {
    flex: 0 0 auto;
    font-size: 15px;
  }
  span {
    max-width: 70px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
.agent-composer-dock {
  position: absolute;
  right: 0;
  bottom: 24px;
  left: 0;
  z-index: 1;
  width: min(760px, calc(100% - 72px));
  padding: 0;
  margin: 0 auto;
  background: var(--admin-page-card-bg);
}
.agent-shortcuts--compact {
  width: 100%;
  padding: 10px;
  margin: 0 0 10px;
  background:
    linear-gradient(180deg, rgba(64, 158, 255, 0.05), rgba(255, 255, 255, 0)),
    var(--admin-page-card-bg);
  box-shadow: 0 10px 24px rgba(15, 23, 42, 0.08);
}
.agent-shortcuts--compact .agent-shortcuts__head {
  align-items: center;
  margin-bottom: 8px;
}
.agent-shortcuts--compact .agent-shortcuts__eyebrow {
  display: none;
}
.agent-shortcuts--compact .agent-shortcuts__title {
  margin-top: 0;
  font-size: 13px;
  line-height: 20px;
}
.agent-shortcuts--compact .agent-shortcuts__actions {
  gap: 6px;
}
.agent-shortcuts--compact .agent-shortcuts__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  min-width: 0;
}
.agent-shortcuts--compact .agent-shortcut {
  grid-template-columns: 28px minmax(0, 1fr);
  min-height: 56px;
  padding: 8px 10px;
}
.agent-shortcuts--compact .agent-shortcut__icon {
  width: 28px;
  height: 28px;
  font-size: 11px;
}
.agent-shortcuts--compact .agent-shortcut__title {
  font-size: 13px;
  line-height: 18px;
}
.agent-shortcuts--compact .agent-shortcut__meta {
  font-size: 11px;
  line-height: 16px;
}

@media screen and (width <= 768px) {
  .agent-chat-panel {
    padding: 0 0 22px;
  }
  .agent-chat-panel.is-empty {
    justify-content: center;
    padding: 24px 0;
  }
  .agent-chat-content,
  .agent-composer-dock {
    width: calc(100% - 36px);
  }
  .agent-chat-empty {
    width: calc(100% - 36px);
  }
  .agent-shortcuts__grid {
    grid-template-columns: minmax(0, 1fr);
  }
  .agent-shortcuts__head {
    align-items: stretch;
  }
  .agent-shortcuts__actions {
    align-items: flex-end;
    flex-direction: column;
    gap: 6px;
  }
  .agent-shortcuts--compact .agent-shortcuts__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .agent-chat-empty__title {
    font-size: 30px;
  }
  .agent-chat-empty__sender {
    width: 100%;
  }
}

@keyframes thinking-bounce {
  0%,
  80%,
  100% {
    opacity: 0.35;
    transform: translateY(0);
  }
  40% {
    opacity: 1;
    transform: translateY(-2px);
  }
}
</style>

<style lang="scss">
.agent-message-runtime-popover {
  .agent-runtime-detail {
    min-width: 0;
    font-size: 12px;
    color: var(--admin-page-text-secondary);
  }
  .agent-runtime-detail__title {
    margin-bottom: 10px;
    font-size: 13px;
    font-weight: 700;
    line-height: 20px;
    color: var(--admin-page-text-primary);
  }
  .agent-runtime-detail__section + .agent-runtime-detail__section {
    padding-top: 10px;
    margin-top: 10px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
  .agent-runtime-detail__section-title {
    margin-bottom: 6px;
    font-weight: 600;
    line-height: 18px;
    color: var(--admin-page-text-regular);
  }
  .agent-runtime-detail__row {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    min-height: 22px;
    line-height: 22px;
  }
  .agent-runtime-detail__row strong {
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    color: var(--admin-page-text-primary);
  }
  .agent-runtime-detail__row.is-total strong {
    color: var(--el-color-primary);
  }
}
</style>
