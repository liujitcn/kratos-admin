<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :request-api="requestTable" />

    <el-drawer
      v-model="detail.visible"
      :title="t('system.ai.session.detail.title')"
      size="min(920px, calc(100vw - 32px))"
      destroy-on-close
    >
      <div v-if="detail.row" class="ai-session-detail">
        <div class="ai-session-detail__head">
          <div class="ai-session-detail__title">{{ detail.row.title }}</div>
          <div v-if="detail.row.summary" class="ai-session-detail__summary">{{ detail.row.summary }}</div>
          <div class="ai-session-detail__meta">
            <el-tag size="small" effect="plain">{{ terminalLabel(detail.row.terminal) }}</el-tag>
            <span>{{ t("system.ai.session.field.user") }}：{{ displayUser(detail.row) }}</span>
            <span>{{ t("system.ai.session.field.message_count") }}：{{ detail.row.message_count }}</span>
            <span>{{ t("system.ai.session.field.total_tokens") }}：{{ detail.row.total_tokens }}</span>
            <span>{{ t("common.field.updated_at") }}：{{ detail.row.updated_at }}</span>
          </div>
        </div>

        <div v-loading="detail.loading" class="ai-session-detail__chat">
          <div v-if="!detail.list.length" class="ai-session-detail__empty">
            {{ t("system.ai.session.message.empty") }}
          </div>
          <BubbleList v-else class="ai-session-detail__bubbles" :list="detailBubbles" max-height="52vh" :auto-scroll="false">
            <template #content="{ item }">
              <div class="ai-session-detail__bubble" :class="{ 'is-user': item.role === 'user' }">
                <div
                  v-if="item.role !== 'user'"
                  class="ai-session-detail__message-meta"
                >
                  <el-tag :type="statusTagType(item.status)" size="small" effect="plain">{{ statusLabel(item.status) }}</el-tag>
                  <span v-if="item.model">{{ t("system.ai.session.field.model") }}：{{ item.model }}</span>
                  <span>{{ t("system.ai.session.field.input_tokens") }}：{{ item.inputTokens }}</span>
                  <span>{{ t("system.ai.session.field.output_tokens") }}：{{ item.outputTokens }}</span>
                  <span>{{ t("system.ai.session.field.total_tokens") }}：{{ item.totalTokens }}</span>
                  <span>{{ t("common.field.created_at") }}：{{ item.createdAtText }}</span>
                </div>
                <div class="ai-session-detail__content" :class="{ 'is-user': item.role === 'user' }">
                  <AiMarkdown v-if="item.role !== 'user'" :content="item.content" />
                  <span v-else>{{ item.content || t("system.ai.session.detail.no_content") }}</span>
                </div>
                <div v-if="item.attachments?.length" class="ai-session-detail__attachments">
                  <Attachments :items="buildMessageAttachmentItems(item.attachments)" overflow="wrap" :hide-upload="true" />
                </div>
              </div>
            </template>
          </BubbleList>
        </div>

        <el-pagination
          v-model:current-page="detail.pageNum"
          :page-size="detail.pageSize"
          :total="detail.total"
          layout="total, prev, pager, next"
          background
          class="ai-session-detail__pagination"
          @current-change="loadDetailMessages"
        />
      </div>
    </el-drawer>

    <ProDialog
      v-model="usage.visible"
      :title="usage.row ? t('system.ai.session.usage.title') + ' - ' + usage.row.title : t('system.ai.session.usage.title')"
      width="min(1320px, calc(100vw - 32px))"
      :show-footer="false"
    >
      <div v-loading="usage.loading" class="ai-session-usage">
        <el-table :data="usage.list" row-key="id" border size="default">
          <el-table-column prop="model" :label="t('system.ai.session.field.model')" min-width="160" />
          <el-table-column prop="input_tokens" :label="t('system.ai.session.field.input_tokens')" width="110" align="right" />
          <el-table-column prop="output_tokens" :label="t('system.ai.session.field.output_tokens')" width="110" align="right" />
          <el-table-column prop="cache_tokens" :label="t('system.ai.session.field.cache_tokens')" width="110" align="right" />
          <el-table-column prop="total_tokens" :label="t('system.ai.session.field.total_tokens')" width="110" align="right" />
          <el-table-column prop="first_token_ms" :label="t('system.ai.session.field.first_token_ms')" width="120" align="right" />
          <el-table-column prop="duration_ms" :label="t('system.ai.session.field.duration_ms')" width="100" align="right" />
          <el-table-column :label="t('system.ai.session.field.status')" width="90" align="center">
            <template #default="{ row }">
              <el-tag :type="statusTagType(row.status)" size="small" effect="plain">{{ statusLabel(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="created_at" :label="t('common.field.created_at')" width="185" align="center" />
        </el-table>
      </div>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { DataAnalysis, View } from "@element-plus/icons-vue";
import { Attachments, BubbleList } from "vue-element-plus-x";
import type { FilesCardProps } from "vue-element-plus-x/types/FilesCard";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import type { ColumnProps, EnumProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defAiSessionService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/ai_session";
import type { AiSessionMessageRecord, AiSessionRecord, PageAiSessionRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ai_session";
import type { AiAttachment } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import { AiMessageStatus } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import { Terminal } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_tool";
import AiMarkdown from "../chat/components/AiMarkdown.vue";
import { buildAIAttachmentFileCard } from "../chat/attachment";
import type { ChatMessageItem } from "../chat/types";

defineOptions({ name: "AiSession", inheritAttrs: false });

const PAGE_SIZE = 20;
const MAX_USAGE_MESSAGES = 2000;
const USAGE_PAGE_SIZE = 100;

/** 会话详情气泡展示项，在聊天气泡基础上补充管理端展示信息。 */
interface DetailBubbleItem extends ChatMessageItem {
  /** 创建时间文本。 */
  createdAtText?: string;
  /** 输入 Token 数。 */
  inputTokens?: number;
  /** 输出 Token 数。 */
  outputTokens?: number;
  /** 总 Token 数。 */
  totalTokens?: number;
}

const { BUTTONS } = useAuthButtons();
const table = ref<ProTableInstance>();
const detail = reactive({
  visible: false,
  row: null as AiSessionRecord | null,
  list: [] as AiSessionMessageRecord[],
  loading: false,
  pageNum: 1,
  pageSize: PAGE_SIZE,
  total: 0
});
const usage = reactive({
  visible: false,
  row: null as AiSessionRecord | null,
  list: [] as AiSessionMessageRecord[],
  loading: false
});

/** 终端枚举选项。 */
const terminalEnum = computed<EnumProps[]>(() => [
  { label: t("system.ai.session.terminal.admin"), value: Terminal.TERMINAL_ADMIN, tagType: "primary" },
  { label: t("system.ai.session.terminal.app"), value: Terminal.TERMINAL_APP, tagType: "success" }
]);

const columns = computed<ColumnProps[]>(() => [
  { prop: "id", label: t("system.ai.session.field.id"), width: 90, align: "center" },
  { prop: "title", label: t("system.ai.session.field.title"), minWidth: 200, showOverflowTooltip: true, search: { el: "input" } },
  {
    prop: "user",
    label: t("system.ai.session.field.user"),
    minWidth: 140,
    showOverflowTooltip: true,
    render: scope => displayUser(scope.row as AiSessionRecord)
  },
  {
    prop: "terminal",
    label: t("system.ai.session.field.terminal"),
    width: 110,
    align: "center",
    enum: terminalEnum,
    isFilterEnum: true,
    tag: true,
    search: { el: "select", props: { clearable: true } }
  },
  { prop: "message_count", label: t("system.ai.session.field.message_count"), width: 100, align: "right" },
  { prop: "input_tokens", label: t("system.ai.session.field.input_tokens"), width: 110, align: "right" },
  { prop: "output_tokens", label: t("system.ai.session.field.output_tokens"), width: 110, align: "right" },
  { prop: "total_tokens", label: t("system.ai.session.field.total_tokens"), width: 110, align: "right" },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 170, align: "center" },
  {
    prop: "operation",
    label: t("common.field.operation"),
    cellType: "actions",
    fixed: "right",
    width: 170,
    actions: [
      { label: t("common.action.detail"), link: true, icon: View, hidden: () => !BUTTONS.value["base:ai-session:detail"], onClick: scope => openDetail(scope.row as AiSessionRecord) },
      { label: t("system.ai.session.action.usage"), link: true, icon: DataAnalysis, hidden: () => !BUTTONS.value["base:ai-session:usage"], onClick: scope => openUsage(scope.row as AiSessionRecord) }
    ]
  }
]);

/** 详情抽屉内按聊天窗口结构拆分的提问与回复气泡。 */
const detailBubbles = computed<DetailBubbleItem[]>(() =>
  detail.list.flatMap(message => {
    const attachments = (message.attachments ?? []).map(item => ({
      id: "",
      name: item.name,
      size: item.size,
      url: item.url,
      mime_type: item.mime_type
    })) as AiAttachment[];
    const question: DetailBubbleItem = {
      key: `question-${message.id}`,
      role: "user",
      kind: "text",
      content: String(message.input_content ?? ""),
      placement: "end",
      attachments: attachments
    } as DetailBubbleItem;
    const answer: DetailBubbleItem = {
      key: `answer-${message.id}`,
      role: "ai",
      kind: "text",
      content: String(message.output_content ?? ""),
      placement: "start",
      model: message.model,
      status: message.status,
      createdAtText: message.created_at,
      inputTokens: message.input_tokens,
      outputTokens: message.output_tokens,
      totalTokens: message.total_tokens
    } as DetailBubbleItem;
    return [question, answer];
  })
);

/** 请求AI会话表格数据。 */
async function requestTable(params: Record<string, unknown>) {
  const request = buildPageRequest<PageAiSessionRequest>(params as unknown as PageAiSessionRequest);
  if (!request.terminal) delete request.terminal;
  const data = await defAiSessionService.PageAiSession(request);
  return { data: { list: data.sessions ?? [], total: data.total } };
}

/** 展示会话所属用户，昵称缺失时回退账号，用户已删除时标注未知用户。 */
function displayUser(row: AiSessionRecord): string {
  return row.nick_name || row.user_name || t("system.ai.session.unknown_user", { id: row.user_id ?? 0 });
}

/** 转换终端枚举为展示文案。 */
function terminalLabel(terminal?: Terminal): string {
  return terminal === Terminal.TERMINAL_APP ? t("system.ai.session.terminal.app") : t("system.ai.session.terminal.admin");
}

/** 转换消息状态为展示文案。 */
function statusLabel(status?: AiMessageStatus): string {
  switch (status) {
    case AiMessageStatus.AI_MESSAGE_STATUS_SUCCESS:
      return t("system.ai.session.status.success");
    case AiMessageStatus.AI_MESSAGE_STATUS_FAILED:
      return t("system.ai.session.status.failed");
    case AiMessageStatus.AI_MESSAGE_STATUS_GENERATING:
      return t("system.ai.session.status.generating");
    default:
      return "";
  }
}

/** 转换消息状态为标签类型。 */
function statusTagType(status?: AiMessageStatus): "success" | "danger" | "warning" | "info" {
  switch (status) {
    case AiMessageStatus.AI_MESSAGE_STATUS_SUCCESS:
      return "success";
    case AiMessageStatus.AI_MESSAGE_STATUS_FAILED:
      return "danger";
    case AiMessageStatus.AI_MESSAGE_STATUS_GENERATING:
      return "warning";
    default:
      return "info";
  }
}

/** 构建消息附件卡片，与聊天窗口一致。 */
function buildMessageAttachmentItems(attachments: AiAttachment[]): FilesCardProps[] {
  return attachments.map(attachment => buildAIAttachmentFileCard(attachment, { maxWidth: "240px" }));
}

/** 打开会话详情抽屉。 */
async function openDetail(row: AiSessionRecord) {
  detail.row = row;
  detail.visible = true;
  detail.pageNum = 1;
  detail.total = 0;
  detail.list = [];
  await loadDetailMessages();
}

/** 加载会话详情消息列表。 */
async function loadDetailMessages() {
  const row = detail.row;
  if (!row) return;
  detail.loading = true;
  try {
    const response = await defAiSessionService.PageAiSessionMessage({ id: row.id, page_num: detail.pageNum, page_size: detail.pageSize });
    detail.list = response.messages ?? [];
    detail.total = response.total;
  } finally {
    detail.loading = false;
  }
}

/** 打开会话用量统计弹窗。 */
async function openUsage(row: AiSessionRecord) {
  usage.row = row;
  usage.visible = true;
  usage.list = [];
  await loadUsageMessages();
}

/** 加载当前会话全部消息。 */
async function loadUsageMessages() {
  const row = usage.row;
  if (!row) return;
  usage.loading = true;
  try {
    let pageNum = 1;
    while (pageNum * USAGE_PAGE_SIZE <= MAX_USAGE_MESSAGES) {
      const response = await defAiSessionService.PageAiSessionMessage({ id: row.id, page_num: pageNum, page_size: USAGE_PAGE_SIZE });
      const messages = response.messages ?? [];
      usage.list.push(...messages);
      if (messages.length < USAGE_PAGE_SIZE || usage.list.length >= response.total) break;
      pageNum++;
    }
  } finally {
    usage.loading = false;
  }
}
</script>

<style scoped>
.ai-session-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.ai-session-detail__head {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.ai-session-detail__title {
  font-size: 16px;
  font-weight: 600;
}

.ai-session-detail__summary {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}

.ai-session-detail__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.ai-session-detail__chat {
  min-height: 120px;
}

.ai-session-detail__empty {
  padding: 32px 0;
  color: var(--el-text-color-secondary);
  text-align: center;
}

.ai-session-detail__bubbles {
  width: 100%;
}

.ai-session-detail__bubble {
  min-width: 0;
}

.ai-session-detail__message-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-bottom: 4px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.ai-session-detail__content {
  font-size: 13px;
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.ai-session-detail__content.is-user {
  white-space: pre-wrap;
  word-break: break-word;
}

.ai-session-detail__attachments {
  margin-top: 8px;
}

.ai-session-detail__pagination {
  justify-content: flex-end;
}

.ai-session-usage {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 用量表格单元格统一不换行，避免时间等列折行。 */
.ai-session-usage :deep(.el-table .cell) {
  white-space: nowrap;
}
</style>
