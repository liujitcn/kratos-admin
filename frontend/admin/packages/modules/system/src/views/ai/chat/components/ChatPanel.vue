<template>
  <el-main class="agent-chat-panel" :class="{ 'is-empty': isEmptyState }">
    <header v-if="activeSession?.title && !isEmptyState" class="agent-chat-heading">
      <div class="agent-reading-column">
        <h2>{{ activeSession.title }}</h2>
        <p v-if="activeSession.summary && activeSession.summary !== activeSession.title">{{ activeSession.summary }}</p>
      </div>
    </header>

    <div ref="scrollRef" class="agent-chat-content" :aria-busy="historyLoading || undefined" @scroll.passive="onScroll">
      <div ref="readingRef" class="agent-reading-column agent-transcript">
        <section v-if="activeSession?.read_only" class="agent-result-banner">
          <span><el-icon><Lock /></el-icon>{{ t("system.ai.workspace.frozen_result") }}</span>
          <el-button :loading="continuingResult" size="small" @click="emit('continueResult')">
            {{ t("system.ai.workspace.continue_discussion") }}
          </el-button>
        </section>

        <section v-if="branchOrigin" class="agent-branch-note">
          <GitBranch :size="17" aria-hidden="true" />
          <div>
            <strong>{{ t("system.ai.chat.branch.origin", { title: branchOrigin.title || t("system.ai.chat.value.default_conversation_title") }) }}</strong>
            <p v-if="branchOrigin.excerpt">{{ branchOrigin.excerpt }}</p>
          </div>
        </section>

        <section v-if="historyLoading && !messages.length" class="agent-history-loading" role="status">
          <p class="agent-status-line"><el-icon class="is-loading"><Loading /></el-icon>{{ t("system.ai.chat.history.loading") }}</p>
          <el-skeleton animated :rows="3" />
        </section>
        <section v-if="historyError" class="agent-callout is-danger" role="alert">
          <el-icon><Warning /></el-icon>
          <div>
            <strong>{{ t("system.ai.chat.history.failed") }}</strong>
            <p>{{ historyError }}</p>
            <el-button size="small" @click="emit('retryHistory')">{{ t("system.ai.chat.history.retry") }}</el-button>
          </div>
        </section>

        <section v-if="isEmptyState" class="agent-chat-empty">
          <el-icon class="agent-chat-empty__icon"><ChatDotRound /></el-icon>
          <h2>{{ branchOrigin ? t("system.ai.chat.branch.continue") : welcomeTitle }}</h2>
          <p>{{ t("system.ai.chat.welcome_description") }}</p>
          <div v-if="shortcutVisibleList.length" class="agent-empty-shortcuts">
            <el-button
              v-for="shortcut in shortcutVisibleList"
              :key="shortcut.key"
              size="small"
              :disabled="sending || (!!shortcut.action?.type && Boolean(blockedReason))"
              @click="shortcut.action?.type ? handleShortcutClick(shortcut) : fillDraft(shortcut.prompt || shortcut.title)"
            >
              {{ shortcut.title || shortcut.prompt }}
            </el-button>
          </div>
        </section>

        <ol v-else-if="messages.length" class="agent-message-list" :aria-label="t('system.ai.chat.title.assistant')">
          <li
            v-for="item in visibleMessages"
            :key="item.key"
            class="agent-message"
            :class="{ 'is-user': item.role === 'user' }"
            :data-message-id="item.id"
          >
            <template v-if="item.role === 'user'">
              <div v-if="!isEditingMessage(item)" class="agent-user-message">
                <div v-if="item.content" class="agent-user-message__text">{{ item.content }}</div>
                <AiAttachments v-if="item.attachments?.length" :files="resolveUserAttachments(item)" />
              </div>
              <div v-if="item.sendError" class="agent-send-error" role="alert">
                <el-icon><Warning /></el-icon>
                <span :title="item.sendError">{{ t("system.ai.chat.status.send_failed") }}</span>
                <el-button type="danger" link :disabled="sending || Boolean(blockedReason)" @click="handleMessageAction('retry', item)">
                  {{ t("system.ai.chat.action.resend") }}
                </el-button>
              </div>
              <div v-if="isEditingMessage(item)" class="agent-message-edit">
                <textarea
                  v-model="editingContent"
                  class="agent-message-edit__input"
                  rows="2"
                  :aria-label="t('common.action.edit')"
                  :disabled="sending"
                  @keydown.ctrl.enter="handleEditKey($event, item)"
                  @keydown.meta.enter="handleEditKey($event, item)"
                ></textarea>
                <div class="agent-inline-actions">
                  <el-button size="small" :disabled="sending" @click="cancelMessageEdit">{{ t("common.action.cancel") }}</el-button>
                  <el-button size="small" type="primary" :disabled="sending || Boolean(blockedReason)" @click="submitMessageEdit(item)">
                    {{ t("system.ai.chat.action.save_and_regenerate") }}
                  </el-button>
                </div>
              </div>
            </template>

            <template v-else>
              <details v-if="hasProcess(item)" class="agent-message-activity">
                <summary>
                  <el-icon :class="{ 'is-loading': hasRunningActivity(item) }">
                    <Loading v-if="hasRunningActivity(item)" />
                    <Warning v-else-if="hasFailedTool(item)" class="is-warning" />
                    <Check v-else />
                  </el-icon>
                  {{ resolveActivitySummary(item) }}
                </summary>
                <div class="agent-process-list">
                  <div v-for="activity in resolveVisibleActivities(item)" :key="activity.id" class="agent-step">
                    <span>{{ resolveActivityTitle(activity) }}</span>
                    <span :class="{ 'is-warning': activity.status === 'failed' }">{{ activityStatus(activity.status) }}</span>
                  </div>
                  <details v-for="(tool, index) in resolveVisibleTools(item)" :key="tool.call_id || index" class="agent-tool">
                    <summary>
                      <span>{{ resolveToolTitle(tool) }}</span>
                      <span class="agent-tool__status" :class="{ 'is-warning': isToolFailed(tool.status) }">{{ resolveToolStatusText(tool.status) }}</span>
                    </summary>
                    <slot name="tool-view" :message="item" :tool="tool">
                      <div class="agent-tool-payload">
                        <section>
                          <div class="agent-tool-payload__head">
                            <strong>{{ t("system.ai.chat.field.request_payload") }}</strong>
                            <el-button link :aria-label="t('system.ai.chat.action.copy_request')" @click="handleCopyToolRequest(tool)">
                              <el-icon><CopyDocument /></el-icon>
                            </el-button>
                          </div>
                          <pre>{{ formatToolPayload(tool.input) }}</pre>
                        </section>
                        <section>
                          <strong>{{ t("system.ai.chat.field.response_payload") }}</strong>
                          <pre>{{ formatToolPayload(tool.output) }}</pre>
                        </section>
                      </div>
                    </slot>
                  </details>
                </div>
              </details>

              <div
                v-if="item.progressState === 'streaming' && !hasReply(item) && !item.recovery && !hasRunningActivity(item)"
                class="agent-status-line"
                role="status"
              >
                <el-icon class="is-loading"><Loading /></el-icon>{{ t("system.ai.chat.message.thinking") }}
              </div>

              <slot v-if="hasReply(item)" name="reply" :message="item">
                <AiMarkdown :content="item.content" :streaming="item.progressState === 'streaming'" @click.capture="openCitation($event, item)" />
              </slot>

              <template v-if="knowledgeSources(item).length">
                <details class="agent-sources">
                  <summary><el-icon><Collection /></el-icon>{{ t("system.ai.chat.sources.title", { count: knowledgeSources(item).length }) }}</summary>
                  <ol>
                    <li v-for="(source, index) in knowledgeSources(item)" :key="index" :data-source-index="index + 1" tabindex="-1">
                      <strong>{{ source.title }}</strong>
                      <span v-if="source.collection">{{ source.collection }}</span>
                      <blockquote v-if="source.excerpt">{{ source.excerpt }}</blockquote>
                    </li>
                  </ol>
                </details>
              </template>

              <section v-if="item.recovery" class="agent-callout is-warning" role="status">
                <el-icon :class="{ 'is-loading': item.recovery === 'reconnecting' }">
                  <Loading v-if="item.recovery === 'reconnecting'" /><Warning v-else />
                </el-icon>
                <div>
                  <strong>{{ t(item.recovery === "reconnecting" ? "system.ai.chat.recovery.reconnecting" : "system.ai.chat.recovery.disconnected") }}</strong>
                  <p>{{ t("system.ai.chat.recovery.description") }}</p>
                  <el-button v-if="item.recovery === 'disconnected'" size="small" @click="emit('recoverReply', item)">
                    {{ t("system.ai.chat.recovery.retry") }}
                  </el-button>
                </div>
              </section>

              <section v-if="isAIFailedMessage(item)" class="agent-callout" :class="isPermissionFailure(item) ? 'is-warning' : 'is-danger'" role="alert">
                <el-icon><Lock v-if="isPermissionFailure(item)" /><Warning v-else /></el-icon>
                <div>
                  <strong>{{ resolveFailureTitle(item) }}</strong>
                  <p>{{ resolveAIErrorMessage(item) }}</p>
                  <details v-if="item.fallback_reason" class="agent-diagnostics">
                    <summary>{{ t("system.ai.chat.title.error_detail") }}</summary>
                    <pre>{{ resolvePersistedMessage(item.fallback_reason) }}</pre>
                  </details>
                  <el-button
                    v-if="canRegenerate(item) && !isPermissionFailure(item) && replyErrorCode(item) !== 'TOOL_OUTCOME_UNKNOWN'"
                    type="primary"
                    size="small"
                    :disabled="sending || Boolean(blockedReason)"
                    @click="handleMessageAction('retry', item)"
                  >
                    {{ t("system.ai.chat.action.regenerate") }}
                  </el-button>
                </div>
              </section>

              <section
                v-if="item.status === AiMessageStatus.AI_MESSAGE_STATUS_WAITING_APPROVAL && item.output_content?.approval"
                class="agent-callout is-warning"
                :aria-label="t('system.ai.chat.approval.title')"
              >
                <ShieldCheck :size="18" aria-hidden="true" />
                <div>
                  <strong>{{ t(approvalExpired ? "system.ai.chat.approval.expired_title" : "system.ai.chat.approval.title") }}</strong>
                  <p>{{ t(approvalExpired ? "system.ai.chat.approval.expired_pending" : "system.ai.chat.approval.description") }}</p>
                  <template v-if="!approvalExpired">
                    <details v-for="(call, index) in item.output_content.approval.calls" :key="index" class="agent-diagnostics">
                      <summary>{{ call.title || call.name }}</summary>
                      <pre>{{ formatToolPayload(call.arguments) }}</pre>
                    </details>
                    <p class="agent-muted">
                      {{ t("system.ai.chat.approval.expires_at", { time: formatDateTime(approvalDeadline) }) }}
                    </p>
                    <div class="agent-inline-actions">
                      <el-button size="small" :disabled="sending" @click="emit('approvalDecision', item, false)">
                        {{ t("system.ai.chat.approval.reject") }}
                      </el-button>
                      <el-button size="small" type="danger" :loading="sending" @click="emit('approvalDecision', item, true)">
                        {{ t("system.ai.chat.approval.confirm") }}
                      </el-button>
                    </div>
                  </template>
                  <el-button v-else size="small" :loading="historyLoading" @click="emit('retryHistory')">
                    {{ t("system.ai.chat.history.retry") }}
                  </el-button>
                </div>
              </section>
              <section v-else-if="item.progressState !== 'streaming' && approvalOutcome(item)" class="agent-callout" :class="approvalOutcomeClass(item)" role="status">
                <el-icon><Check v-if="approvalOutcomeClass(item) === 'is-success'" /><Warning v-else /></el-icon>
                <div>
                  <strong>{{ resolveApprovalTitle(item) }}</strong>
                  <p v-if="approvalOutcome(item)?.state === 'expired'">{{ t("system.ai.chat.approval.expired") }}</p>
                  <p v-else-if="approvalOutcome(item)?.state === 'rejected'">{{ t("system.ai.chat.approval.rejected") }}</p>
                  <el-button
                    v-if="approvalOutcome(item)?.state === 'expired' && approvalOutcome(item)?.retryable === true && canRegenerate(item)"
                    size="small"
                    :disabled="sending || Boolean(blockedReason)"
                    @click="handleMessageAction('retry', item)"
                  >
                    {{ t("system.ai.chat.approval.restart") }}
                  </el-button>
                </div>
              </section>

              <div v-if="hasFailedTool(item) && !isAIFailedMessage(item) && item.progressState !== 'streaming'" class="agent-callout is-warning" role="status">
                <el-icon><Warning /></el-icon>
                <div>
                  <strong>{{ t("system.ai.chat.status.tool_failed") }}</strong>
                  <p>{{ resolveVisibleTools(item).filter(tool => isToolFailed(tool.status)).map(resolveToolTitle).join(" · ") }}</p>
                  <slot name="tool-retry" :message="item" :tools="resolveVisibleTools(item).filter(tool => isToolFailed(tool.status))" />
                </div>
              </div>

              <details v-if="item.fallback && !isAIFailedMessage(item)" class="agent-diagnostics">
                <summary>{{ t("system.ai.chat.source.fallback") }}</summary>
                <p v-if="item.model">{{ resolveModelDisplayName(modelProviders, item.model) }}</p>
                <pre v-if="item.fallback_reason">{{ resolvePersistedMessage(item.fallback_reason) }}</pre>
              </details>

              <slot name="flow-blocks" :message="item" :active-flow-message-id="activeFlowMessageID" />
              <AiAttachments
                :files="resolveGeneratedAttachments(item)"
                :issues="item.blocks?.filter(block => block.type === 'generated_file_issue')"
                :can-reexport="!activeSession?.read_only && !sending && !blockedReason"
                generated
                @reexport="requestFileExport"
              />
            </template>

            <div v-if="!isEditingMessage(item) && (resolveMessageActions(item).length || hasAIUsage(item))" class="agent-message-actions">
              <template v-for="action in resolveMessageActions(item)" :key="action.key">
                <el-popconfirm
                  v-if="shouldConfirmMessageAction(action.key)"
                  :title="resolveActionConfirmTitle(action.key, item)"
                  :confirm-button-text="t('common.action.confirm')"
                  :cancel-button-text="t('common.action.cancel')"
                  width="220"
                  @confirm="handleMessageAction(action.key, item)"
                >
                  <template #reference>
                    <el-button class="agent-message-action" text :disabled="sending" :aria-label="action.label">
                      <el-icon><component :is="action.icon" /></el-icon>
                    </el-button>
                  </template>
                </el-popconfirm>
                <el-tooltip v-else :content="action.label" placement="top">
                  <el-button class="agent-message-action" text :disabled="sending && action.key !== 'copy'" :aria-label="action.label" @click="handleMessageAction(action.key, item)">
                    <el-icon><component :is="action.icon" /></el-icon>
                  </el-button>
                </el-tooltip>
              </template>
              <details v-if="hasAIUsage(item)" class="agent-runtime">
                <summary><el-icon><InfoFilled /></el-icon>{{ t("system.ai.chat.action.view_runtime") }}</summary>
                <dl>
                  <div v-if="item.model">
                    <dt>{{ t("system.ai.session.field.model") }}</dt>
                    <dd>{{ resolveModelDisplayName(modelProviders, item.model) }}</dd>
                  </div>
                  <div v-if="resolveMessageReplyTime(item)">
                    <dt>{{ t("common.field.created_at") }}</dt><dd>{{ resolveMessageReplyTime(item) }}</dd>
                  </div>
                  <template v-if="item.token?.total">
                    <div><dt>{{ t("system.ai.chat.field.input_tokens") }}</dt><dd>{{ formatNumber(item.token.input) }}</dd></div>
                    <div><dt>{{ t("system.ai.chat.field.output_tokens") }}</dt><dd>{{ formatNumber(item.token.output) }}</dd></div>
                    <div v-if="item.token.cache"><dt>{{ t("system.ai.chat.field.cache_tokens") }}</dt><dd>{{ formatNumber(item.token.cache) }}</dd></div>
                    <div><dt>{{ t("system.ai.chat.field.total_tokens") }}</dt><dd>{{ formatNumber(item.token.total) }}</dd></div>
                  </template>
                  <div v-if="item.first_token_ms"><dt>{{ t("system.ai.chat.field.first_token") }}</dt><dd>{{ formatDurationMs(item.first_token_ms) }}</dd></div>
                  <div v-if="item.duration_ms"><dt>{{ t("system.ai.chat.field.total_duration") }}</dt><dd>{{ formatDurationMs(item.duration_ms) }}</dd></div>
                </dl>
              </details>
            </div>
          </li>
        </ol>
      </div>
    </div>

    <div v-if="messages.length && !followLatest" class="agent-jump">
      <el-button round size="small" @click="scrollToLatest()">
        <el-icon><ArrowDown /></el-icon>{{ t("system.ai.chat.action.latest") }}
      </el-button>
    </div>
    <div v-if="!activeSession?.read_only" ref="composerDockRef" class="agent-reading-column agent-composer-dock">
      <ChatComposer
        ref="composerRef"
        :blocked-reason="blockedReason"
        :key="senderKey"
        :draft="draft"
        @draft-change="emit('draftChange', $event)"
        :sending="sending"
        :model-providers="modelProviders"
        :provider-id="providerId"
        :model-name="modelName"
        :knowledge-options="knowledgeOptions"
        :knowledge-ids="knowledgeIds"
        @submit="handleSubmit"
        @model-change="handleModelChange"
        @knowledge-change="handleKnowledgeChange"
      >
        <template v-if="hasShortcutPanel" #before-attachment>
          <el-popover
            v-model:visible="shortcutPopoverOpen"
            :reference-el="composerDockRef ?? undefined"
            :width="Math.min(composerDockWidth || 420, 420)"
            :offset="8"
            :show-arrow="false"
            :persistent="false"
            :disabled="sending || Boolean(blockedReason)"
            :aria-label="t('system.ai.chat.shortcut.label')"
            placement="top-start"
            trigger="click"
            role="dialog"
            popper-class="agent-shortcut-popper"
            @before-enter="updateComposerDockBounds"
            @after-enter="focusShortcutPopover"
          >
            <template #reference>
              <el-button
                ref="shortcutTriggerRef"
                class="admin-custom-button agent-shortcut-trigger"
                native-type="button"
                aria-haspopup="dialog"
                :aria-expanded="shortcutPopoverOpen"
                :disabled="sending || Boolean(blockedReason)"
              >
                <Zap :size="14" aria-hidden="true" />
                {{ t("system.ai.chat.shortcut.label") }}
                <ChevronUp :size="14" :class="{ 'is-open': shortcutPopoverOpen }" aria-hidden="true" />
              </el-button>
            </template>
            <div
              ref="shortcutPopoverRef"
              class="agent-shortcut-picker"
              :style="{ maxHeight: `${Math.max(120, composerDockTop - 16)}px` }"
              tabindex="-1"
              @keydown.esc.stop.prevent="dismissShortcutPopover"
            >
              <div class="agent-shortcut-picker__head">
                <strong>{{ t("system.ai.chat.shortcut.label") }}</strong>
                <span class="agent-shortcut-picker__count" aria-live="polite">{{ shortcutBatchLabel }}</span>
                <el-button
                  v-if="canRefreshShortcutBatch"
                  class="admin-custom-button agent-shortcuts__refresh"
                  native-type="button"
                  :disabled="sending || Boolean(blockedReason)"
                  @click="refreshShortcutBatch"
                >
                  <el-icon><Refresh /></el-icon>
                  <span>{{ t("system.ai.chat.action.refresh_shortcuts") }}</span>
                </el-button>
              </div>
              <div v-if="loadingShortcuts" class="agent-shortcuts__loading" role="status">
                {{ t("system.ai.chat.message.loading_shortcuts") }}
              </div>
              <div v-else class="agent-shortcut-picker__list">
                <el-button
                  v-for="shortcut in shortcutVisibleList"
                  :key="shortcut.key"
                  class="admin-custom-button agent-shortcut"
                  native-type="button"
                  :disabled="sending || Boolean(blockedReason)"
                  @click="handleShortcutClick(shortcut)"
                >
                  <span class="agent-shortcut__title">{{ shortcut.title || shortcut.prompt }}</span>
                  <span class="agent-shortcut__meta">{{ resolveShortcutMeta(shortcut) }}</span>
                </el-button>
              </div>
            </div>
          </el-popover>
        </template>
      </ChatComposer>
    </div>
  </el-main>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, ref, watch } from "vue";
import type { Component } from "vue";
import { useElementBounding, useResizeObserver } from "@vueuse/core";
import {
  ArrowDown,
  ChatDotRound,
  Check,
  Collection,
  CopyDocument,
  Delete,
  EditPen,
  InfoFilled,
  Loading,
  Lock,
  Refresh,
  Warning
} from "@element-plus/icons-vue";
import { ChevronUp, GitBranch, ShieldCheck, Volume2, Zap } from "@lucide/vue";
import { ElMessage } from "element-plus";
import type { ButtonInstance } from "element-plus";
import { getCurrentLocale, t } from "@liujitcn/kratos-admin-core";
import { formatDateTime } from "@liujitcn/kratos-admin-core/format";
import type { AiAction } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_message";
import { AiMessageStatus, type AiSession } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import type { AiProviderModelOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_provider";
import type { AiKnowledgeOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_knowledge";
import type { AiShortcut, AiToolCall } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_tool";
import { resolveModelDisplayName } from "@liujitcn/kratos-admin-system/components/AiModelSelector/modelDisplay";
import { resolvePersistedMessage } from "../../../../utils/persisted-message";
import { isAIGeneratedAttachment } from "../attachment";
import { resolveTimestamp } from "../message";
import type { AiActivityItem, ChatMessageAction, ChatMessageEditPayload, ChatMessageItem, SubmitPayload } from "../types";
import ChatComposer from "./ChatComposer.vue";
import AiAttachments from "./AiAttachments.vue";

const AiMarkdown = defineAsyncComponent(() => import("./AiMarkdown.vue"));

/** 消息操作按钮配置。 */
type MessageActionOption = { key: ChatMessageAction; label: string; icon: Component };

/** 来自当前回复的知识库来源。 */
type KnowledgeSource = { title: string; collection?: string; excerpt?: string };

const props = defineProps<{
  /** 当前活动会话。 */
  activeSession?: AiSession;
  /** 当前会话的消息列表。 */
  messages: ChatMessageItem[];
  /** 当前会话是否正在提交或生成消息。 */
  sending: boolean;
  /** 当前会话历史记录是否正在加载。 */
  historyLoading?: boolean;
  /** 最近一次历史读取错误。 */
  historyError?: string;
  /** 是否正在将只读结果复制成讨论。 */
  continuingResult?: boolean;
  /** 快捷指令与模型配置是否正在读取。 */
  loadingShortcuts?: boolean;
  /** 当前终端可用快捷指令。 */
  shortcuts?: AiShortcut[];
  /** 当前启用的供应商和模型。 */
  modelProviders: AiProviderModelOption[];
  /** 当前选择的供应商编号。 */
  providerId: number;
  /** 当前选择的模型名称。 */
  modelName: string;
  /** 当前可选的知识库。 */
  knowledgeOptions: AiKnowledgeOption[];
  /** 当前选择的知识库编号列表。 */
  knowledgeIds: number[];
  /** 按会话保存的输入与附件。 */
  draft?: SubmitPayload;
}>();

const emit = defineEmits<{
  /** 提交聊天消息。 */
  submit: [payload: SubmitPayload];
  draftChange: [value: SubmitPayload];
  continueResult: [];
  retryHistory: [];
  recoverReply: [item: ChatMessageItem];
  modelChange: [providerId: number, modelName: string];
  knowledgeChange: [knowledgeIds: number[]];
  messageAction: [payload: { action: ChatMessageAction; item: ChatMessageItem }];
  messageEdit: [payload: ChatMessageEditPayload];
  approvalDecision: [item: ChatMessageItem, approved: boolean];
}>();

defineExpose({ fillDraft });

const isEmptyState = computed(() => !props.messages.length && !props.historyLoading && !props.historyError && !props.activeSession?.read_only);
const composerRef = ref<InstanceType<typeof ChatComposer>>();
const senderKey = computed(() => props.activeSession?.id || "empty-session");
const scrollRef = ref<HTMLElement>();
const readingRef = ref<HTMLElement>();
const followLatest = ref(true);
const composerDockRef = ref<HTMLElement>();
const { top: composerDockTop, width: composerDockWidth, update: updateComposerDockBounds } = useElementBounding(composerDockRef);
const shortcutTriggerRef = ref<ButtonInstance>();
const shortcutPopoverRef = ref<HTMLElement>();
const shortcutPopoverOpen = ref(false);
const editingMessageKey = ref("");
const editingContent = ref("");

const visibleMessages = computed(() => props.messages.filter(item => item.role !== "user" || item.content || resolveUserAttachments(item).length));
const branchOrigin = computed(() => props.messages.find(item => item.blocks?.some(block => block.type === "branch_origin"))
  ?.blocks?.find(block => block.type === "branch_origin"));
const pendingApproval = computed(() => props.messages.findLast(item => item.role === "ai" && item.status === AiMessageStatus.AI_MESSAGE_STATUS_WAITING_APPROVAL));
const approvalDeadline = computed(() => resolveTimestamp(pendingApproval.value?.output_content?.approval?.expires_at));
const approvalExpired = ref(false);

watch(approvalDeadline, (deadline, _, onCleanup) => {
  approvalExpired.value = deadline > 0 && deadline <= Date.now();
  if (deadline > Date.now()) {
    const timeout = window.setTimeout(() => { approvalExpired.value = true; }, deadline - Date.now());
    onCleanup(() => window.clearTimeout(timeout));
  }
}, { immediate: true });

const blockedReason = computed(() => {
  if (pendingApproval.value) return t(approvalExpired.value ? "system.ai.chat.approval.expired_pending" : "system.ai.chat.approval.pending");
  if (props.messages.some(item => item.recovery === "disconnected")) return t("system.ai.chat.recovery.description");
  if (props.historyError || props.historyLoading) return t("system.ai.chat.history.composer_hold");
  if (props.loadingShortcuts && !props.modelName) return t("system.ai.chat.status.loading");
  if (!props.modelName || !props.providerId) return t("system.ai.chat.model_unavailable");
  return "";
});

const lastEditableUserMessage = computed(() => props.messages.findLast(item => item.role === "user" && !item.localOnly));
const activeFlowMessageID = computed(() => {
  const latest = props.messages.at(-1);
  if (props.activeSession?.read_only || props.sending || props.historyLoading || props.historyError ||
      latest?.role !== "ai" || latest.status !== AiMessageStatus.AI_MESSAGE_STATUS_SUCCESS) return "";
  return latest.id;
});

watch(senderKey, async () => {
  cancelMessageEdit();
  followLatest.value = true;
  await nextTick();
  scrollToLatest();
});
watch([senderKey, isEmptyState, () => props.sending, () => props.activeSession?.read_only], () => {
  shortcutPopoverOpen.value = false;
});
useResizeObserver([readingRef, scrollRef], () => {
  if (followLatest.value) scrollToLatest();
});

/** 选择用户上传的附件。 */
function resolveUserAttachments(item: ChatMessageItem) {
  return item.attachments?.filter(file => !isAIGeneratedAttachment(file)) ?? [];
}

/** 选择助手生成的附件。 */
function resolveGeneratedAttachments(item: ChatMessageItem) {
  return item.attachments?.filter(isAIGeneratedAttachment) ?? [];
}

/** 通过滚动位置区分跟随最新回复和查看历史。 */
function onScroll() {
  const container = scrollRef.value;
  if (!container?.clientHeight) return;
  followLatest.value = container.scrollHeight - container.scrollTop - container.clientHeight < 80;
}

/** 恢复跟随并滚动到最新消息。 */
function scrollToLatest() {
  followLatest.value = true;
  const container = scrollRef.value;
  if (container) container.scrollTop = container.scrollHeight;
}

const welcomeTitle = computed(() => {
  const hour = new Date().getHours();
  if (hour < 12) return t("system.ai.chat.welcome.morning");
  if (hour < 18) return t("system.ai.chat.welcome.afternoon");
  return t("system.ai.chat.welcome.evening");
});

const SHORTCUT_BATCH_SIZE = 4;
const shortcutBatchIndex = ref(0);
const shortcutAllList = computed(() =>
  [...(props.shortcuts ?? [])]
    .filter(item => Boolean(item?.key && (item.title || item.prompt)))
    .sort((left, right) => Number(left.sort ?? 0) - Number(right.sort ?? 0))
);
const shortcutBatchCount = computed(() => Math.max(1, Math.ceil(shortcutAllList.value.length / SHORTCUT_BATCH_SIZE)));
const shortcutVisibleList = computed(() => {
  const start = (shortcutBatchIndex.value % shortcutBatchCount.value) * SHORTCUT_BATCH_SIZE;
  return shortcutAllList.value.slice(start, start + SHORTCUT_BATCH_SIZE);
});
const hasShortcutPanel = computed(() => !props.activeSession?.read_only && (props.loadingShortcuts || shortcutVisibleList.value.length > 0));
const canRefreshShortcutBatch = computed(() => shortcutAllList.value.length > SHORTCUT_BATCH_SIZE);
const shortcutBatchLabel = computed(() => {
  const total = shortcutAllList.value.length;
  if (props.loadingShortcuts && total === 0) return t("system.ai.chat.status.loading");
  if (total <= SHORTCUT_BATCH_SIZE) return t("system.ai.chat.value.item_count", { count: total });
  const start = (shortcutBatchIndex.value % shortcutBatchCount.value) * SHORTCUT_BATCH_SIZE + 1;
  return [start + "-" + Math.min(start + SHORTCUT_BATCH_SIZE - 1, total), total].join(" / ");
});

/** 在可发送状态下交由页面提交，保持视口跟随新一轮。 */
function handleSubmit(payload: SubmitPayload) {
  if (props.activeSession?.read_only || props.sending || blockedReason.value) return;
  scrollToLatest();
  emit("submit", payload);
}

/** 在当前草稿里预填问题，保留已上传附件。 */
function fillDraft(text: string) {
  if (props.activeSession?.read_only) return;
  emit("draftChange", { text, attachments: props.draft?.attachments ?? [] });
  composerRef.value?.focus();
}

/** 在现有草稿中追加重导出请求，交给用户确认后发送。 */
function requestFileExport(name: string) {
  const request = t("system.ai.chat.file.reexport_draft", { name });
  const existing = props.draft?.text.trimEnd();
  if (existing?.endsWith(request)) return;
  fillDraft(existing ? `${existing}\n\n${request}` : request);
}

/** 转发供应商和模型选择。 */
function handleModelChange(providerId: number, modelName: string) {
  emit("modelChange", providerId, modelName);
}

/** 转发知识库选择。 */
function handleKnowledgeChange(knowledgeIds: number[]) {
  emit("knowledgeChange", knowledgeIds);
}

/** 展开快捷指令时聚焦首个选项。 */
function focusShortcutPopover() {
  const firstShortcut = shortcutPopoverRef.value?.querySelector<HTMLButtonElement>(".agent-shortcut:not(:disabled)");
  (firstShortcut ?? shortcutPopoverRef.value)?.focus({ preventScroll: true });
}

/** 关闭快捷指令并将焦点还给入口。 */
async function dismissShortcutPopover() {
  shortcutPopoverOpen.value = false;
  await nextTick();
  shortcutTriggerRef.value?.ref?.focus({ preventScroll: true });
}

/** 触发已配置的快捷动作，普通提示只预填输入。 */
function handleShortcutClick(shortcut: AiShortcut) {
  shortcutPopoverOpen.value = false;
  const text = shortcut.prompt || shortcut.title;
  if (!shortcut.action?.type) {
    fillDraft(text);
    return;
  }
  const action: AiAction = {
    flow: shortcut.action.flow || "",
    step: shortcut.action.step || "",
    type: shortcut.action.type,
    payload_json: shortcut.action.payload_json || "{}",
    source_message_id: "",
    action_id: "",
    flow_version: 0
  };
  handleSubmit({ text, attachments: [], action });
}

/** 切换下一组快捷指令。 */
function refreshShortcutBatch() {
  shortcutBatchIndex.value = (shortcutBatchIndex.value + 1) % shortcutBatchCount.value;
}

/** 展示快捷入口的分组和工具依赖数量。 */
function resolveShortcutMeta(shortcut: AiShortcut) {
  const group = shortcut.group || t("system.ai.chat.value.general_assistant");
  const count = shortcut.required_tools?.length ?? 0;
  return count ? t("system.ai.chat.value.shortcut_tools", { group, count }) : group;
}

/** 在最新的持久化轮次上提供重新生成。 */
function canRegenerate(item: ChatMessageItem) {
  return !props.activeSession?.read_only && !item.localOnly && props.messages.at(-1)?.key === item.key;
}

/** 根据角色与消息状态提供允许的操作。 */
function resolveMessageActions(item: ChatMessageItem): MessageActionOption[] {
  if (item.progressState === "streaming" || item.status === AiMessageStatus.AI_MESSAGE_STATUS_GENERATING) return [];
  const copy: MessageActionOption = { key: "copy", label: t("system.ai.chat.action.copy"), icon: CopyDocument };
  if (props.activeSession?.read_only) return item.content ? [copy] : [];
  const remove: MessageActionOption = { key: "delete", label: t("common.action.delete"), icon: Delete };
  if (item.status === AiMessageStatus.AI_MESSAGE_STATUS_WAITING_APPROVAL) return [remove];
  const actions: MessageActionOption[] = [];
  if (item.role === "user" && item.key === lastEditableUserMessage.value?.key) {
    actions.push({ key: "edit", label: t("common.action.edit"), icon: EditPen });
  }
  if (item.role === "ai" && item.status === AiMessageStatus.AI_MESSAGE_STATUS_SUCCESS) {
    if (canRegenerate(item) && !blockedReason.value && !approvalOutcome(item) && !hasFailedTool(item)) {
      actions.push({ key: "retry", label: t("system.ai.chat.action.regenerate"), icon: Refresh });
    }
    if (!approvalOutcome(item)) actions.push({ key: "branch", label: t("system.ai.chat.action.create_branch"), icon: GitBranch });
    if (item.content) {
      actions.push({ key: "speak", label: t(item.speaking ? "system.ai.chat.action.stop_speaking" : "system.ai.chat.action.speak"), icon: Volume2 });
    }
  }
  if (item.content) actions.unshift(copy);
  return [...actions, remove];
}

/** 编辑在面板内进行，其他操作由所属页面处理。 */
function handleMessageAction(action: ChatMessageAction, item: ChatMessageItem) {
  if (action === "edit") {
    editingMessageKey.value = item.key;
    editingContent.value = item.content;
    return;
  }
  emit("messageAction", { action, item });
}

/** 判断这条消息是否正在编辑。 */
function isEditingMessage(item: ChatMessageItem) {
  return editingMessageKey.value === item.key;
}

/** 收起编辑框并清理编辑状态。 */
function cancelMessageEdit() {
  editingMessageKey.value = "";
  editingContent.value = "";
}

/** 输入法确认时保留 Enter 的选字行为。 */
function handleEditKey(event: Event, item: ChatMessageItem) {
  if (!(event instanceof KeyboardEvent) || event.isComposing || event.keyCode === 229) return;
  event.preventDefault();
  submitMessageEdit(item);
}

/** 将更新后的问题交给页面重新生成。 */
function submitMessageEdit(item: ChatMessageItem) {
  if (props.sending || blockedReason.value) return;
  if (!editingContent.value.trim()) {
    ElMessage.warning(t("system.ai.chat.message.content_required"));
    return;
  }
  if (editingContent.value === item.content) {
    ElMessage.info(t("system.ai.chat.message.content_unchanged"));
    return;
  }
  emit("messageEdit", { item, content: editingContent.value });
  cancelMessageEdit();
}

/** 对会覆盖历史或删除消息的操作二次确认。 */
function shouldConfirmMessageAction(action: ChatMessageAction) {
  return action === "retry" || action === "delete";
}

/** 返回与消息操作一致的确认提示。 */
function resolveActionConfirmTitle(action: ChatMessageAction, item: ChatMessageItem) {
  if (action === "delete") return t("system.ai.chat.dialog.confirm_delete_message");
  return t(item.role === "user" ? "system.ai.chat.dialog.confirm_resend_message" : "system.ai.chat.dialog.confirm_regenerate_message");
}

/** 错误内容单独展示，断线前收到的局部正文仍作为 Markdown。 */
function hasReply(item: ChatMessageItem) {
  if (approvalOutcome(item)?.state === "expired" || approvalOutcome(item)?.state === "rejected") return false;
  return Boolean(item.content && item.content !== t("system.ai.chat.message.thinking") && (!isAIFailedMessage(item) || item.localOnly));
}

/** 判断该回复是否失败。 */
function isAIFailedMessage(item: ChatMessageItem) {
  return item.role === "ai" && item.status === AiMessageStatus.AI_MESSAGE_STATUS_FAILED;
}

/** 取结构化服务端错误码。 */
function replyErrorCode(item: ChatMessageItem) {
  return item.blocks?.find(block => block.type === "reply_error")?.code;
}

/** 区分权限失败和普通生成错误。 */
function isPermissionFailure(item: ChatMessageItem) {
  return replyErrorCode(item) === "PERMISSION_DENIED";
}

/** 生成与服务端失败类型匹配的标题。 */
function resolveFailureTitle(item: ChatMessageItem) {
  const code = replyErrorCode(item);
  if (code === "PERMISSION_DENIED") return t("system.ai.chat.error.permission");
  if (code === "REPLY_TIMEOUT") return t("system.ai.chat.error.timeout");
  if (code === "TOOL_OUTCOME_UNKNOWN") return t("system.ai.chat.error.tool_unknown");
  return t("system.ai.chat.error.reply");
}

/** 已持久化错误使用原始可读摘要。 */
function resolveAIErrorMessage(item: ChatMessageItem) {
  if (replyErrorCode(item) === "TOOL_OUTCOME_UNKNOWN") return t("system.ai.chat.error.tool_unknown_description");
  return (!item.localOnly && resolvePersistedMessage(item.content)) || t("system.ai.chat.message.reply_failed");
}

/** 返回由服务端审计状态生成的审批结果。 */
function approvalOutcome(item: ChatMessageItem) {
  return item.blocks?.findLast(block => block.type === "approval_outcome");
}

/** 只为有确定证据的完成态使用成功颜色。 */
function approvalOutcomeClass(item: ChatMessageItem) {
  return approvalOutcome(item)?.execution === "succeeded" ? "is-success" : "is-warning";
}

/** 表达审批决定和可核实的实际执行状态。 */
function resolveApprovalTitle(item: ChatMessageItem) {
  const result = approvalOutcome(item);
  if (result?.state === "expired") return t("system.ai.chat.approval.expired_title");
  if (result?.state === "rejected") return t("system.ai.chat.approval.rejected_title");
  if (result?.execution === "succeeded") return t("system.ai.chat.approval.executed");
  if (result?.execution === "failed") return t("system.ai.chat.approval.execution_failed");
  return t("system.ai.chat.approval.execution_unknown");
}

/** 读取本轮回复持久化的检索来源。 */
function knowledgeSources(item: ChatMessageItem): KnowledgeSource[] {
  const sources: unknown = item.blocks?.find(block => block.type === "knowledge_sources")?.sources;
  if (!Array.isArray(sources)) return [];
  return sources.filter((source): source is KnowledgeSource => source && typeof source.title === "string");
}

/** 引用链接只打开同一条回复的已保存来源。 */
function openCitation(event: MouseEvent, item: ChatMessageItem) {
  if (!(event.target instanceof Element) || !(event.currentTarget instanceof Element)) return;
  const href = event.target.closest("a")?.getAttribute("href") ?? "";
  const index = href.match(/^#ai-source-([1-9]\d*)$/)?.[1];
  if (!index) return;
  event.preventDefault();
  event.stopPropagation();
  if (Number(index) > knowledgeSources(item).length) return;
  const source = event.currentTarget.closest(".agent-message")?.querySelector<HTMLElement>('[data-source-index="' + index + '"]');
  const details = source?.closest("details");
  if (!details || !source) return;
  details.open = true;
  source.focus({ preventScroll: true });
  source.scrollIntoView({ block: "nearest" });
}

/** 回复详情同时承载时间、模型和用量。 */
function hasAIUsage(item: ChatMessageItem) {
  return item.role === "ai" &&
    item.progressState !== "streaming" &&
    Boolean(item.model || item.created_at || item.token?.total || item.first_token_ms || item.duration_ms);
}

/** 将毫秒耗时格式化为秒。 */
function formatDurationMs(value: number) {
  return (value / 1000).toFixed(2) + "s";
}

/** 格式化消息创建时间。 */
function resolveMessageReplyTime(item: ChatMessageItem) {
  const timestamp = resolveTimestamp(item.created_at);
  return timestamp ? formatDateTime(timestamp) : "";
}

/** 按当前界面语言渲染数字。 */
function formatNumber(value: number) {
  return new Intl.NumberFormat(getCurrentLocale()).format(value);
}

/** 工具记录与活动归入一个默认折叠的处理过程。 */
function hasProcess(item: ChatMessageItem) {
  return (item.activities?.length ?? 0) > 0 || resolveVisibleTools(item).length > 0;
}

/** 判断执行过程中是否仍有进行中的活动。 */
function hasRunningActivity(item: ChatMessageItem) {
  return !item.recovery && (item.activities ?? []).some(activity => activity.status === "running");
}

/** 保留尚未形成持久记录的工具进度以及其他执行阶段。 */
function resolveVisibleActivities(item: ChatMessageItem) {
  const recordedTools = new Set(resolveVisibleTools(item).map(tool => tool.name));
  return (item.activities ?? []).filter(activity =>
    activity.phase !== "tool" || activity.status === "running" || !recordedTools.has(activity.tool_name || "")
  );
}

/** 只读取已持久化且可展示的函数工具记录。 */
function resolveVisibleTools(item: ChatMessageItem) {
  return (item.tools ?? []).filter(tool => tool.type === "function" && (tool.name || tool.title));
}

/** 判断本轮是否存在工具错误。 */
function hasFailedTool(item: ChatMessageItem) {
  return resolveVisibleTools(item).some(tool => isToolFailed(tool.status)) || (item.activities ?? []).some(
    activity => activity.phase === "tool" && activity.status === "failed"
  );
}

/** 根据运行中的步骤或已完成的工具记录生成摘要。 */
function resolveActivitySummary(item: ChatMessageItem) {
  const current = item.activities?.findLast(activity => activity.status === "running");
  if (current && !item.recovery) return current.phase === "tool"
    ? t("system.ai.chat.status.running_tool", { tool: resolveActivityTitle(current) })
    : resolveActivityTitle(current);
  const tools = resolveVisibleTools(item);
  const failed = tools.filter(tool => isToolFailed(tool.status)).length;
  if (failed) return t("system.ai.chat.status.tool_summary_failed", { done: tools.length - failed, failed });
  if (item.recovery) return t("system.ai.chat.title.activity");
  if (hasFailedTool(item)) return t("system.ai.chat.status.tool_failed");
  if (pendingApproval.value?.key === item.key) return t("system.ai.chat.approval.title");
  if (tools.length) return t("system.ai.chat.status.tool_summary_completed", { count: tools.length });
  return t("system.ai.chat.title.activity");
}

/** 展示本地化步骤名称。 */
function resolveActivityTitle(activity: AiActivityItem) {
  if (activity.phase === "reasoning") return activity.summary || t("system.ai.chat.status.thinking");
  if (activity.phase === "tool") {
    if (activity.tool_name === "web_search") return t("system.ai.chat.value.web_search");
    return activity.tool_title || activity.tool_name || t("system.ai.chat.value.tool");
  }
  return t("system.ai.chat.status.answering");
}

/** 展示工具步骤的运行状态。 */
function activityStatus(status: AiActivityItem["status"]) {
  if (status === "waiting_approval") return t("system.ai.chat.approval.title");
  if (status === "running") return t("system.ai.chat.status.running");
  if (status === "failed") return t("system.ai.chat.status.failed");
  return t("system.ai.chat.status.completed");
}

/** 展示本地化工具名称。 */
function resolveToolTitle(tool: AiToolCall) {
  if (tool.name === "web_search") return t("system.ai.chat.value.web_search");
  return tool.title || tool.name || t("system.ai.chat.value.tool");
}

/** 区分工具失败与已完成。 */
function resolveToolStatusText(status?: string) {
  return t(isToolFailed(status) ? "system.ai.chat.status.failed" : "system.ai.chat.status.completed");
}

/** 判断工具记录是否为失败。 */
function isToolFailed(status?: string) {
  return ["error", "failed", "fail", "timeout"].includes(String(status ?? "").toLowerCase());
}

/** 尽量按 JSON 缩进展示入参和结果。 */
function formatToolPayload(payload?: string) {
  if (!payload) return "{}";
  try {
    return JSON.stringify(JSON.parse(payload), null, 2);
  } catch {
    return payload;
  }
}

/** 复制工具的实际请求参数。 */
async function handleCopyToolRequest(tool: AiToolCall) {
  try {
    await navigator.clipboard.writeText(formatToolPayload(tool.input));
    ElMessage.success(t("system.ai.chat.message.request_copied"));
  } catch {
    ElMessage.error(t("system.ai.chat.message.copy_unsupported"));
  }
}
</script>

<style scoped lang="scss">
.agent-chat-panel {
  --agent-reading-width: 760px;
  --agent-gutter: clamp(14px, 3cqw, 32px);
  --el-main-padding: 0;

  position: relative;
  container-type: inline-size;
  display: flex;
  flex: 1 1 auto;
  flex-direction: column;
  height: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  color: var(--admin-page-text-primary);
  background: var(--admin-page-card-bg);
}

.agent-reading-column {
  width: min(var(--agent-reading-width), calc(100% - var(--agent-gutter) * 2));
  min-width: 0;
  margin-inline: auto;
}

.agent-chat-heading {
  flex: none;
  padding-block: 12px;
  border-bottom: 1px solid var(--admin-page-divider);

  h2,
  p {
    margin: 0;
    overflow-wrap: anywhere;
  }

  h2 {
    font-size: 16px;
    font-weight: 630;
    line-height: 1.4;
  }

  p {
    margin-top: 3px;
    color: var(--admin-page-text-secondary);
    font-size: 12px;
  }
}

.agent-chat-content {
  flex: 1 1 auto;
  min-height: 0;
  overflow: auto;
  overscroll-behavior: contain;
  scrollbar-gutter: stable both-edges;
}

.agent-transcript {
  display: grid;
  gap: 16px;
  align-content: start;
  padding-block: 22px 12px;
}

.is-empty .agent-chat-content {
  display: flex;
  align-items: safe center;
}

.agent-chat-empty {
  padding: 24px 0;
  text-align: center;

  h2 {
    margin: 12px 0 8px;
    font-size: clamp(22px, 4cqw, 32px);
    font-weight: 650;
    line-height: 1.3;
  }

  p {
    margin: 0;
    color: var(--admin-page-text-secondary);
    font-size: 14px;
  }
}

.agent-chat-empty__icon {
  width: 42px;
  height: 42px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: var(--admin-page-radius);
  font-size: 22px;
}

.agent-empty-shortcuts,
.agent-inline-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;

  :deep(.el-button + .el-button) {
    margin: 0;
  }
}

.agent-empty-shortcuts {
  justify-content: center;
  margin-top: 16px;

  :deep(.el-button) {
    max-width: 100%;
    height: auto;
    min-height: 26px;
    white-space: normal;
  }
}

.agent-result-banner,
.agent-branch-note {
  display: flex;
  align-items: center;
  gap: 8px 12px;
  padding: 10px 12px;
  border: 1px solid var(--admin-page-divider);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg-soft);
  font-size: 13px;
}

.agent-result-banner {
  justify-content: space-between;
  flex-wrap: wrap;

  span {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
}

.agent-branch-note {
  > svg {
    flex: none;
    color: var(--el-color-primary);
  }

  p {
    display: -webkit-box;
    margin: 4px 0 0;
    overflow: hidden;
    color: var(--admin-page-text-secondary);
    overflow-wrap: anywhere;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 3;
  }
}

.agent-history-loading {
  padding-block: 6px 18px;
}

.agent-message-list {
  display: grid;
  min-width: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.agent-message {
  display: grid;
  gap: 12px;
  min-width: 0;
  padding: 0 0 12px;
  line-height: 1.55;
  overflow-wrap: anywhere;
}

.agent-message.is-user {
  justify-items: end;
  gap: 4px;
  padding-bottom: 0;
}

.agent-message:not(.is-user) + .agent-message.is-user {
  margin-top: 22px;
}

.agent-message.is-user + .agent-message:not(.is-user) {
  margin-top: 21px;
}

.agent-user-message {
  display: grid;
  gap: 8px;
  min-width: 0;
  width: fit-content;
  max-width: min(84%, 640px);
  padding: 10px 14px;
  color: var(--admin-page-text-primary);
  background: var(--el-color-primary-light-9);
  border-radius: 10px 10px 3px 10px;
  font-size: 14px;
  line-height: 24px;
}

.agent-user-message__text {
  white-space: pre-wrap;
}

.agent-send-error,
.agent-status-line {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  font-size: 13px;
}

.agent-send-error {
  color: var(--el-color-danger);
}

.agent-status-line {
  margin-bottom: 10px;
  color: var(--admin-page-text-secondary);
}

.agent-message > .agent-status-line {
  margin-bottom: 0;
}

.agent-message-edit {
  display: grid;
  gap: 10px;
  width: min(100%, 520px);
}

.agent-message-edit__input {
  box-sizing: border-box;
  width: 100%;
  min-height: 62px;
  max-height: 210px;
  padding: 8px 10px;
  resize: vertical;
  field-sizing: content;
  border: 1px solid var(--admin-page-divider);
  border-radius: var(--admin-page-radius);
  background: var(--el-bg-color);
  color: var(--admin-page-text-primary);
  font: inherit;
  line-height: 1.5;

  &:focus {
    border-color: var(--el-color-primary);
    outline: 1px solid var(--el-color-primary);
  }
}

.agent-message-edit .agent-inline-actions {
  justify-content: flex-end;
}

.agent-callout {
  --agent-callout-accent: var(--el-color-warning);
  --agent-callout-bg: var(--el-color-warning-light-9);

  display: flex;
  align-items: flex-start;
  gap: 9px;
  min-width: 0;
  padding: 12px;
  border: 1px solid color-mix(in srgb, var(--agent-callout-accent) 38%, var(--admin-page-divider));
  border-radius: var(--admin-page-radius);
  background: var(--agent-callout-bg);
  font-size: 13px;
  overflow-wrap: anywhere;

  > :is(svg, .el-icon) {
    flex: none;
    margin-top: 2px;
    color: var(--agent-callout-accent);
  }

  > div {
    min-width: 0;
  }

  p {
    margin: 4px 0 0;
  }

  :deep(.el-button) {
    margin-top: 8px;
  }
}

.agent-callout.is-danger {
  --agent-callout-accent: var(--el-color-danger);
  --agent-callout-bg: var(--el-color-danger-light-9);
}

.agent-callout.is-success {
  --agent-callout-accent: var(--el-color-success);
  --agent-callout-bg: var(--el-color-success-light-9);
}

.agent-muted,
.is-warning {
  color: var(--admin-page-text-secondary);
}

.el-icon.is-warning,
.agent-tool__status.is-warning,
.agent-step > .is-warning {
  color: var(--el-color-warning);
}

.agent-message-activity,
.agent-diagnostics,
.agent-sources,
.agent-runtime {
  min-width: 0;
  max-width: 100%;
  color: var(--admin-page-text-secondary);
  font-size: 13px;

  > summary {
    width: fit-content;
    max-width: 100%;
    padding: 2px 5px;
    margin-inline-start: -5px;
    border-radius: var(--admin-page-radius);
    cursor: pointer;
  }

  > summary:hover {
    color: var(--admin-page-text-primary);
    background: var(--admin-page-card-bg-soft);
  }

  > summary > .el-icon {
    margin-inline: 3px;
    vertical-align: -2px;
  }

  p {
    margin: 5px 0;
  }
}

.agent-process-list,
.agent-sources > ol {
  padding: 6px 10px;
  margin: 5px 0 0;
  border: 1px solid var(--admin-page-divider);
  border-radius: var(--admin-page-radius);
}

.agent-sources > ol {
  padding-inline-start: 26px;

  li + li {
    margin-top: 9px;
  }

  strong {
    color: var(--admin-page-text-primary);
  }

  span {
    display: block;
  }

  blockquote {
    margin: 4px 0 0;
    padding-inline-start: 8px;
    border-inline-start: 2px solid var(--admin-page-divider-strong);
  }
}

.agent-step,
.agent-tool > summary {
  position: relative;
  min-width: 0;
  padding: 6px 76px 6px 4px;
  border-bottom: 1px solid var(--admin-page-divider);
}

.agent-step > :last-child,
.agent-tool__status {
  position: absolute;
  inset-inline-end: 4px;
  inset-block-start: 6px;
  color: var(--admin-page-text-secondary);
  font-size: 12px;
}

.agent-tool > summary {
  cursor: pointer;
}

.agent-tool:last-child > summary,
.agent-step:last-child {
  border-bottom: 0;
}

.agent-tool-payload {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(200px, 100%), 1fr));
  align-items: start;
  gap: 8px;
  padding: 8px 0;

  section {
    min-width: 0;
  }
}

.agent-tool-payload__head {
  display: flex;
  min-height: 20px;
  align-items: center;
  justify-content: space-between;
}

.agent-message pre {
  max-height: 320px;
  padding: 8px;
  margin: 5px 0;
  overflow: auto;
  border: 1px solid var(--admin-page-divider);
  border-radius: var(--admin-page-radius);
  color: var(--admin-page-text-primary);
  background: var(--admin-page-card-bg-muted);
  font-size: 12px;
  white-space: pre;
}

.agent-callout .agent-diagnostics {
  margin-top: 8px;
  margin-inline-start: 5px;
}

.agent-message-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 3px;
  margin: -5px 0 0 -6px;

  :deep(.el-button + .el-button) {
    margin: 0;
  }
}

.is-user .agent-message-actions {
  margin-top: 0;
  margin-inline: 0 -6px;
}

.agent-message-action.el-button {
  width: 29px;
  height: 29px;
  padding: 0;
  color: var(--admin-page-text-secondary);
  font-size: 14px;

  &:hover,
  &:focus-visible {
    color: var(--el-color-primary);
  }

  svg {
    width: 1em;
    height: 1em;
  }
}

.agent-runtime {
  max-width: 100%;
  padding: 3px 4px 0;
  font-size: 12px;

  dl {
    display: grid;
    gap: 4px;
    padding: 8px;
    margin: 3px 0 0;
    border: 1px solid var(--admin-page-divider);
    border-radius: var(--admin-page-radius);
  }

  dl > div {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    gap: 8px;
  }

  dd {
    margin: 0;
    color: var(--admin-page-text-primary);
    overflow-wrap: anywhere;
  }
}

.agent-composer-dock {
  flex: none;
  margin-block: 7px 14px;
}

.agent-jump {
  z-index: 1;
  display: flex;
  height: 0;
  align-items: flex-end;
  justify-content: center;

  .el-button {
    gap: 5px;
    margin-bottom: 2px;
    box-shadow: var(--admin-page-shadow);
  }
}

.agent-shortcuts__refresh {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  height: 26px;
  padding: 0 8px;
  box-sizing: border-box;
  margin: 0;
  font-size: 12px;
  font-family: inherit;
  font-weight: 400;
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
.agent-shortcut {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  gap: 2px;
  align-items: flex-start;
  justify-content: center;
  min-width: 0;
  min-height: 44px;
  padding: 5px 8px;
  margin: 0;
  font: inherit;
  line-height: 1.2;
  color: var(--admin-page-text-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--admin-page-radius);
}
.agent-shortcut:hover,
.agent-shortcut:focus-visible {
  background: var(--el-color-primary-light-9);
}
.agent-shortcut:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.agent-shortcut__title {
  font-size: 13px;
  font-weight: 600;
  line-height: 18px;
  overflow-wrap: anywhere;
  white-space: normal;
}
.agent-shortcut__meta {
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
  overflow-wrap: anywhere;
  white-space: normal;
}
.agent-shortcut-trigger {
  display: inline-flex;
  gap: 5px;
  align-items: center;
  height: 34px;
  padding: 0 8px;
  margin: 0;
  font-size: 12px;
  color: var(--el-color-primary);
  white-space: nowrap;
  cursor: pointer;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius);
  &:hover,
  &[aria-expanded="true"] {
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-5);
  }
  &:disabled {
    cursor: not-allowed;
    opacity: 0.55;
  }
  .is-open {
    transform: rotate(180deg);
  }
}
:global(.agent-shortcut-popper.el-popover) {
  box-sizing: border-box;
  padding: 0;
  border-radius: var(--admin-page-radius);
}
.agent-shortcut-picker {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  padding: 8px;
  &__head {
    display: flex;
    flex: none;
    flex-wrap: wrap;
    gap: 6px 8px;
    align-items: center;
    padding: 0 4px 6px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  &__count {
    margin-inline-end: auto;
    font-size: 12px;
    color: var(--admin-page-text-secondary);
  }
  &__list {
    display: grid;
    grid-auto-rows: max-content;
    gap: 2px;
    min-height: 0;
    max-height: min(360px, 50dvh);
    padding: 4px;
    overflow-y: auto;
  }
}


@container (width <= 500px) {
  .agent-chat-heading { padding-block: 9px; }
  .agent-transcript { padding-top: 15px; }
  .agent-user-message { max-width: 92%; }
  .agent-composer-dock { margin-bottom: 8px; }
}

@media (prefers-reduced-motion: reduce) {
  .is-loading { animation: none; }
}
</style>
