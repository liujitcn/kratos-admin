<template>
  <el-container class="bi-assistant-workbench">
    <el-header class="bi-assistant-workbench__topbar">
      <div class="bi-assistant-workbench__brand">
        <el-icon><DataAnalysis /></el-icon>
        <div>
          <h1>AI 助手 · BI 智能问数工作台</h1>
          <span>{{ t("bi.workbench.title") }}</span>
        </div>
      </div>
      <el-select v-model="activeSessionID" class="bi-assistant-workbench__session" filterable :loading="loadingSessions" :placeholder="t('system.ai.chat.value.new_conversation')" @change="selectSession">
        <el-option v-for="session in visibleSessions" :key="session.id" :label="session.title" :value="session.id" />
      </el-select>
      <el-tooltip :content="t('system.ai.chat.action.new_conversation')" placement="bottom">
        <el-button circle :icon="Plus" :disabled="currentSessionSending" @click="createSession" />
      </el-tooltip>
      <el-input v-model="sessionKeyword" class="bi-assistant-workbench__search" clearable :prefix-icon="Search" :placeholder="t('system.ai.chat.placeholder.search_conversation')" />
      <span class="bi-assistant-workbench__spacer" />
      <el-tooltip content="工具箱" placement="bottom">
        <el-button circle :icon="Tools" @click="ElMessage.info('文件、自动化和插件仍可从 AI 助手入口打开')" />
      </el-tooltip>
    </el-header>

    <el-main class="bi-assistant-workbench__body">
      <el-aside class="bi-assistant-workbench__zone" :class="{ 'is-collapsed': collapsed.left }">
        <el-header class="bi-assistant-workbench__zone-head">
          <el-button text circle :icon="collapsed.left ? ArrowRight : ArrowLeft" @click="toggleZone('left')" />
          <span v-if="!collapsed.left" class="bi-assistant-workbench__zone-title">左侧工作台</span>
          <span v-if="!collapsed.left" class="bi-assistant-workbench__spacer" />
          <el-button v-if="!collapsed.left" text :icon="Plus" @click="openCardPicker('left')">{{ t("bi.card.add_short") }}</el-button>
        </el-header>
        <div v-if="!collapsed.left" class="bi-assistant-workbench__zone-body" @dragover.prevent @drop.prevent="dropCard('left')">
          <BiAssistantCardZone zone="left" :cards="leftCards" :refresh-keys="refreshKeys" @drag-start="startCardDrag" @drag-end="dragged = undefined" @drop="dropCard" @refresh="refreshCard" @quote="quoteCard" @remove="removeCard" @export="exportCard" />
        </div>
        <span v-else class="bi-assistant-workbench__vertical-label">左</span>
      </el-aside>

      <el-main class="bi-assistant-workbench__chat">
        <ChatPanel
          :active-session="activeSession"
          :messages="currentMessages"
          :sending="currentSessionSending"
          :loading-shortcuts="loadingShortcuts"
          :shortcuts="starterShortcuts"
          :model-providers="modelProviders"
          :provider-id="selectedProviderId"
          :model-name="selectedModelName"
          :knowledge-options="knowledgeOptions"
          :knowledge-ids="selectedKnowledgeIds"
          :draft="drafts[activeSessionID]"
          @submit="submitMessage"
          @draft-change="updateDraft"
          @model-change="changeModel"
          @knowledge-change="changeKnowledge"
          @message-action="handleMessageAction"
          @message-edit="editMessage"
          @approval-decision="decideApproval"
          @bi-card-saved="handleBiCardSaved"
        >
          <template #flow-blocks="{ message, activeFlowMessageId }">
            <AiFlowBlocks v-if="message.blocks?.length" :message="message" :active-flow-message-id="activeFlowMessageId" :renderers="flowBlockRenderers" @flow-action="handleFlowAction" />
          </template>
        </ChatPanel>
      </el-main>

      <el-aside class="bi-assistant-workbench__zone" :class="{ 'is-collapsed': collapsed.right }">
        <el-header class="bi-assistant-workbench__zone-head">
          <span v-if="!collapsed.right" class="bi-assistant-workbench__zone-title">右侧工作台</span>
          <span v-if="!collapsed.right" class="bi-assistant-workbench__spacer" />
          <el-button v-if="!collapsed.right" text :icon="Plus" @click="openCardPicker('right')">{{ t("bi.card.add_short") }}</el-button>
          <el-button text circle :icon="collapsed.right ? ArrowLeft : ArrowRight" @click="toggleZone('right')" />
        </el-header>
        <div v-if="!collapsed.right" class="bi-assistant-workbench__zone-body" @dragover.prevent @drop.prevent="dropCard('right')">
          <BiAssistantCardZone zone="right" :cards="rightCards" :refresh-keys="refreshKeys" @drag-start="startCardDrag" @drag-end="dragged = undefined" @drop="dropCard" @refresh="refreshCard" @quote="quoteCard" @remove="removeCard" @export="exportCard" />
        </div>
        <span v-else class="bi-assistant-workbench__vertical-label">右</span>
      </el-aside>
    </el-main>

    <el-dialog v-model="cardPicker.open" :title="t('bi.card.library')" width="min(720px, 92vw)" destroy-on-close>
      <div class="bi-assistant-workbench__picker-head">
        <span>{{ t("bi.card.library_hint") }}</span>
        <el-input v-model="cardPicker.keyword" clearable :prefix-icon="Search" :placeholder="t('bi.card.search')" />
      </div>
      <div v-loading="cardPicker.loading" class="bi-assistant-workbench__picker-list">
        <el-button v-for="card in pickerCards" :key="card.id" class="bi-assistant-workbench__picker-item" text @click="addCard(card)">
          <span class="bi-assistant-workbench__picker-icon"><el-icon><DataAnalysis /></el-icon></span>
          <span class="bi-assistant-workbench__picker-text"><strong>{{ card.name }}</strong><small>{{ card.source_code }} · {{ card.chart_type }}</small></span>
          <el-icon><Plus /></el-icon>
        </el-button>
        <el-empty v-if="!cardPicker.loading && !pickerCards.length" :description="t('bi.card.empty')" />
      </div>
    </el-dialog>
  </el-container>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { ElMessage } from "element-plus";
import { ArrowLeft, ArrowRight, CopyDocument, DataAnalysis, Delete, Download, Plus, Refresh, Search, Tools } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import { defAiKnowledgeService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_knowledge";
import { defAiMessageService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_message";
import { defAiModelService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_provider";
import { defAiSessionService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_session";
import { defAiToolService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_tool";
import { AiMessageStatus } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import { Terminal } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_tool";
import type { AiAction } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_message";
import { getAdminAiFlowBlockRenderers } from "@liujitcn/kratos-admin-system";
import AiFlowBlocks from "@liujitcn/kratos-admin-system/components/ai/AiFlowBlocks.vue";
import ChatPanel from "@liujitcn/kratos-admin-system/components/ai/ChatPanel.vue";
import {
  createLocalUserMessage,
  createThinkingMessage,
  normalizeMessageList,
  normalizeSession,
  normalizeSessionList,
  replacePendingMessages,
  sortMessages,
  useAiChatState,
  useAiChatStream
} from "@liujitcn/kratos-admin-system/components/ai/chat";
import type { ChatMessageAction, ChatMessageEditPayload, ChatMessageItem, SubmitPayload } from "@liujitcn/kratos-admin-system/components/ai/chat";
import { resolveModelSelection, saveModelSelection } from "@liujitcn/kratos-admin-system/components/AiModelSelector/modelSelection";
import type { BiCard, BiWorkbench } from "../rpc/bi/admin/v1/bi_card";
import { defBiCardService } from "../api/bi/admin/v1/bi_card";
import BiAssistantCardZone from "./BiAssistantCardZone.vue";

type Zone = "left" | "right";
type CardDrag = { zone: Zone; index: number } | undefined;
type StoredZoneLayout = { left: number[]; right: number[] };

const state = useAiChatState();
const {
  drafts,
  activeSessionID,
  loadingSessions,
  loadingShortcuts,
  sessions,
  starterShortcuts,
  modelProviders,
  selectedProviderId,
  selectedModelName,
  knowledgeOptions,
  selectedKnowledgeIds,
  messages,
  activeSession,
  currentMessages,
  currentSessionSending,
  isSessionSending,
  setSessionSending,
  upsertSession
} = state;
const flowBlockRenderers = getAdminAiFlowBlockRenderers();
const aiStream = useAiChatStream({
  messages,
  setSessionSending,
  onSession: session => upsertSession(normalizeSession(session)),
  onError: error => ElMessage.error(error.message)
});
const sessionKeyword = ref("");
const leftCards = ref<BiCard[]>([]);
const rightCards = ref<BiCard[]>([]);
const allCards = ref<BiCard[]>([]);
const refreshKeys = reactive<Record<number, number>>({});
const dragged = ref<CardDrag>();
const collapsed = reactive({ left: false, right: false });
const cardPicker = reactive({ open: false, loading: false, allLoaded: false, keyword: "", zone: "right" as Zone });
const workbench = ref<BiWorkbench>({ dashboard_ids: [], favorite_ids: [], dashboards: [] });
// shortcut: 左右分区暂存在浏览器本地，跨设备同步时再扩展 BiWorkbench 协议。
const zoneStorageKey = "kratos-admin:bi-assistant-workbench:zones";

const visibleSessions = computed(() => {
  const keyword = sessionKeyword.value.trim().toLowerCase();
  return keyword ? sessions.value.filter(item => (item.title + " " + item.summary).toLowerCase().includes(keyword)) : sessions.value;
});
const pickerCards = computed(() => {
  const used = new Set([...leftCards.value, ...rightCards.value].map(item => item.id));
  const keyword = cardPicker.keyword.trim().toLowerCase();
  return allCards.value.filter(item => !used.has(item.id) && (!keyword || (item.name + " " + item.source_code).toLowerCase().includes(keyword)));
});

function toggleZone(zone: Zone) {
  collapsed[zone] = !collapsed[zone];
}

async function loadCards() {
  const [config, page] = await Promise.all([
    defBiCardService.GetBiWorkbench({}),
    defBiCardService.PageBiCard({ name: "", source_code: "", category: "favorite", search_source_codes: [], page_num: 1, page_size: 100 })
  ]);
  workbench.value = config;
  allCards.value = page.cards;
  const byID = new Map(page.cards.map(item => [item.id, item]));
  const ordered = config.favorite_ids.flatMap(id => (byID.has(id) ? [byID.get(id)!] : []));
  const saved = readZoneLayout(ordered);
  leftCards.value = saved.left;
  rightCards.value = saved.right;
}

function readZoneLayout(cards: BiCard[]): { left: BiCard[]; right: BiCard[] } {
  try {
    const saved = JSON.parse(localStorage.getItem(zoneStorageKey) || "null") as StoredZoneLayout | null;
    if (saved && Array.isArray(saved.left) && Array.isArray(saved.right)) {
      const byID = new Map(cards.map(item => [item.id, item]));
      const left = saved.left.flatMap(id => (byID.has(id) ? [byID.get(id)!] : []));
      const right = saved.right.flatMap(id => (byID.has(id) ? [byID.get(id)!] : []));
      const used = new Set([...left, ...right].map(item => item.id));
      return { left, right: [...right, ...cards.filter(item => !used.has(item.id))] };
    }
  } catch {
    // 本地布局损坏时回退到服务端收藏顺序。
  }
  return { left: cards.filter((_, index) => index % 2 === 0), right: cards.filter((_, index) => index % 2 === 1) };
}

function persistZoneLayout() {
  localStorage.setItem(zoneStorageKey, JSON.stringify({ left: leftCards.value.map(item => item.id), right: rightCards.value.map(item => item.id) } satisfies StoredZoneLayout));
}

async function saveFavorites() {
  persistZoneLayout();
  const latest = await defBiCardService.GetBiWorkbench({});
  const ids = [...leftCards.value, ...rightCards.value].map(item => item.id);
  workbench.value = await defBiCardService.SaveBiWorkbench({ dashboard_ids: latest.dashboard_ids, favorite_ids: ids });
}

function startCardDrag(value: { zone: Zone; index: number }) {
  dragged.value = value;
}

async function dropCard(targetZone: Zone, targetIndex?: number) {
  const source = dragged.value;
  dragged.value = undefined;
  if (!source) return;
  const sourceCards = source.zone === "left" ? leftCards.value : rightCards.value;
  const targetCards = targetZone === "left" ? leftCards.value : rightCards.value;
  const [card] = sourceCards.splice(source.index, 1);
  if (!card) return;
  targetCards.splice(targetIndex === undefined ? targetCards.length : Math.min(targetIndex, targetCards.length), 0, card);
  try {
    await saveFavorites();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  }
}

function removeCard(card: BiCard) {
  leftCards.value = leftCards.value.filter(item => item.id !== card.id);
  rightCards.value = rightCards.value.filter(item => item.id !== card.id);
  void saveFavorites().catch(cause => ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error")));
}

function refreshCard(card: BiCard) {
  refreshKeys[card.id] = (refreshKeys[card.id] || 0) + 1;
}

function quoteCard(card: BiCard) {
  if (!activeSessionID.value) return;
  drafts.value[activeSessionID.value] = {
    ...(drafts.value[activeSessionID.value] || { text: "", attachments: [] }),
    text: "[引用卡片：" + card.name + "] "
  };
  ElMessage.success("已在输入框引用该卡片");
}

async function exportCard(card: BiCard) {
  try {
    const result = await defBiCardService.GetBiCardData({ id: card.id, range_days: 0 });
    const header = (result.columns || []).map(item => item.name || item.code);
    const rows = (result.rows || []).map(row => row.values || []);
    const csv = [header, ...rows].map(row => row.map(value => '"' + String(value || "").replaceAll('"', '""') + '"').join(",")).join("\n");
    const url = URL.createObjectURL(new Blob(["\ufeff" + csv], { type: "text/csv;charset=utf-8" }));
    const link = document.createElement("a");
    link.href = url;
    link.download = card.name + ".csv";
    link.click();
    URL.revokeObjectURL(url);
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  }
}

async function openCardPicker(zone: Zone) {
  cardPicker.zone = zone;
  cardPicker.keyword = "";
  cardPicker.open = true;
  if (cardPicker.allLoaded) return;
  cardPicker.loading = true;
  try {
    const page = await defBiCardService.PageBiCard({ name: "", source_code: "", category: "", search_source_codes: [], page_num: 1, page_size: 100 });
    allCards.value = page.cards;
    cardPicker.allLoaded = true;
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    cardPicker.loading = false;
  }
}

async function addCard(card: BiCard) {
  const target = cardPicker.zone === "left" ? leftCards : rightCards;
  if (!target.value.some(item => item.id === card.id)) target.value.push(card);
  cardPicker.open = false;
  try {
    await saveFavorites();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  }
}

async function handleBiCardSaved(value: unknown) {
  if (!value || typeof value !== "object" || !("id" in value)) return;
  const card = value as BiCard;
  allCards.value = [card, ...allCards.value.filter(item => item.id !== card.id)];
  if (!rightCards.value.some(item => item.id === card.id)) rightCards.value.push(card);
  try {
    await saveFavorites();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  }
}

async function createSession() {
  const result = await defAiSessionService.CreateAiSession({
    title: t("system.ai.chat.value.new_conversation"),
    terminal: Terminal.TERMINAL_ADMIN,
    plugins: []
  });
  const session = normalizeSession(result.session);
  upsertSession(session);
  activeSessionID.value = session.id;
  messages.value[session.id] = [];
}

async function selectSession(id: string) {
  if (!id) return;
  const result = await defAiSessionService.ListAiMessage({ session_id: id });
  messages.value[id] = normalizeMessageList(result.messages);
}

async function ensureSession() {
  if (activeSessionID.value) return activeSessionID.value;
  if (sessions.value[0]) {
    activeSessionID.value = sessions.value[0].id;
    return activeSessionID.value;
  }
  await createSession();
  return activeSessionID.value;
}

async function loadChat() {
  const [sessionResult, shortcutResult, modelResult, knowledgeResult] = await Promise.all([
    defAiSessionService.ListAiSession({}),
    defAiToolService.ListAiShortcut({ terminal: Terminal.TERMINAL_ADMIN }),
    defAiModelService.ListAiProviderModelOptions({}),
    defAiKnowledgeService.ListAiKnowledgeOptions({})
  ]);
  sessions.value = normalizeSessionList(sessionResult.sessions);
  starterShortcuts.value = shortcutResult.shortcuts || [];
  modelProviders.value = modelResult.providers || [];
  knowledgeOptions.value = knowledgeResult.options || [];
  const selection = resolveModelSelection(modelProviders.value);
  selectedProviderId.value = selection.provider_id;
  selectedModelName.value = selection.model_name;
  const id = await ensureSession();
  if (id && !messages.value[id]?.length) await selectSession(id);
}

async function submitMessage(payload: SubmitPayload) {
  const sessionID = await ensureSession();
  if (!sessionID || isSessionSending(sessionID) || activeSession.value?.read_only) return;
  if ((messages.value[sessionID] || []).some(item => item.status === AiMessageStatus.AI_MESSAGE_STATUS_WAITING_APPROVAL)) {
    ElMessage.warning(t("system.ai.chat.approval.pending"));
    return;
  }
  messages.value[sessionID] = sortMessages([
    ...(messages.value[sessionID] || []),
    createLocalUserMessage(payload),
    createThinkingMessage({ sessionID })
  ]);
  setSessionSending(sessionID, true);
  void aiStream.run(sessionID, payload, {
    providerID: selectedProviderId.value,
    modelName: selectedModelName.value,
    knowledgeBaseIDs: selectedKnowledgeIds.value
  });
}

function handleFlowAction(action: AiAction, label?: string) {
  void submitMessage({ text: label || t("system.ai.chat.action.continue"), attachments: [], action });
}

function updateDraft(value: SubmitPayload) {
  if (activeSessionID.value) drafts.value[activeSessionID.value] = value;
}

function changeModel(providerID: number, modelName: string) {
  selectedProviderId.value = providerID;
  selectedModelName.value = modelName;
  saveModelSelection({ provider_id: providerID, model_name: modelName });
}

function changeKnowledge(ids: number[]) {
  selectedKnowledgeIds.value = ids;
}

async function handleMessageAction(payload: { action: ChatMessageAction; item: ChatMessageItem }) {
  const id = activeSessionID.value;
  if (!id) return;
  if (payload.action === "copy") {
    await navigator.clipboard?.writeText(payload.item.content);
    return;
  }
  if (payload.action === "speak") {
    if (!window.speechSynthesis) return ElMessage.warning("当前浏览器不支持朗读");
    window.speechSynthesis.cancel();
    window.speechSynthesis.speak(new SpeechSynthesisUtterance(payload.item.content));
    return;
  }
  if (payload.action === "delete") {
    await defAiMessageService.DeleteAiMessage({ session_id: id, message_id: String(payload.item.id) });
    messages.value[id] = messages.value[id].filter(item => item.id !== payload.item.id);
    return;
  }
  if (payload.action === "branch") {
    const result = await defAiSessionService.CreateAiSessionBranch({
      source_session_id: id,
      anchor_message_id: String(payload.item.id),
      title: "分支：" + payload.item.content.slice(0, 18),
      terminal: Terminal.TERMINAL_ADMIN
    });
    const session = normalizeSession(result.session);
    upsertSession(session);
    activeSessionID.value = session.id;
    messages.value[session.id] = normalizeMessageList(result.messages);
    return;
  }
  if (payload.action === "retry") {
    const result =
      payload.item.role === "user"
        ? await defAiMessageService.RetryAiUserMessage({ session_id: id, message_id: String(payload.item.id) })
        : await defAiMessageService.RegenerateAiMessage({ session_id: id, message_id: String(payload.item.id) });
    messages.value[id] = replacePendingMessages(messages.value[id] || [], normalizeMessageList(result.messages));
    if (result.session) upsertSession(normalizeSession(result.session));
  }
}

async function editMessage(payload: ChatMessageEditPayload) {
  const id = activeSessionID.value;
  if (!id) return;
  const result = await defAiMessageService.UpdateAiMessage({
    session_id: id,
    message_id: String(payload.item.id),
    content: payload.content
  });
  messages.value[id] = replacePendingMessages(messages.value[id] || [], normalizeMessageList(result.messages));
  if (result.session) upsertSession(normalizeSession(result.session));
}

async function decideApproval(item: ChatMessageItem, approved: boolean) {
  const id = activeSessionID.value;
  const approvalID = item.output_content?.approval?.id;
  if (!id || !approvalID) return;
  const result = await defAiMessageService.DecideAiToolApproval({
    session_id: id,
    message_id: String(item.id),
    approval_id: approvalID,
    approved
  });
  messages.value[id] = replacePendingMessages(messages.value[id] || [], normalizeMessageList(result.messages));
  if (result.session) upsertSession(normalizeSession(result.session));
}

onMounted(async () => {
  try {
    await Promise.all([loadCards(), loadChat()]);
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  }
});
</script>

<style scoped>
.bi-assistant-workbench {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 104px);
  min-height: 640px;
  overflow: hidden;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-divider-strong);
  border-radius: var(--admin-page-radius);
}
.bi-assistant-workbench__topbar {
  display: flex;
  flex: none;
  align-items: center;
  gap: 10px;
  min-height: 54px;
  height: auto;
  padding: 8px 14px;
  border-bottom: 1px solid var(--admin-page-divider-strong);
}
.bi-assistant-workbench__brand {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 190px;
}
.bi-assistant-workbench__brand > .el-icon {
  color: var(--el-color-primary);
  font-size: 20px;
}
.bi-assistant-workbench__brand h1 {
  margin: 0;
  color: var(--el-text-color-primary);
  font-size: 15px;
  font-weight: 600;
  line-height: 20px;
}
.bi-assistant-workbench__brand span {
  color: var(--el-text-color-secondary);
  font-size: 11px;
  line-height: 16px;
}
.bi-assistant-workbench__session {
  width: min(260px, 22vw);
}
.bi-assistant-workbench__search {
  width: min(220px, 20vw);
}
.bi-assistant-workbench__spacer {
  flex: 1;
  min-width: 0;
}
.bi-assistant-workbench__body {
  display: grid;
  flex: 1;
  min-height: 0;
  padding: 0;
  grid-template-columns: minmax(248px, 300px) minmax(460px, 1fr) minmax(248px, 300px);
}
.bi-assistant-workbench__zone {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  background: var(--el-fill-color-light);
  padding: 0;
}
.bi-assistant-workbench__zone:first-child {
  border-right: 1px solid var(--admin-page-divider-strong);
}
.bi-assistant-workbench__zone:last-child {
  border-left: 1px solid var(--admin-page-divider-strong);
}
.bi-assistant-workbench__zone.is-collapsed {
  width: 44px;
  align-items: center;
  background: var(--admin-page-card-bg);
}
.bi-assistant-workbench__zone-head {
  display: flex;
  flex: none;
  align-items: center;
  min-height: 48px;
  height: 48px;
  gap: 6px;
  padding: 0 10px;
  background: var(--admin-page-card-bg);
  border-bottom: 1px solid var(--admin-page-divider);
}
.bi-assistant-workbench__zone-title {
  color: var(--el-text-color-primary);
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}
.bi-assistant-workbench__zone-body {
  flex: 1;
  min-height: 0;
  padding: 10px;
  overflow: auto;
}
.bi-assistant-workbench__chat {
  min-width: 0;
  min-height: 0;
  background: var(--admin-page-card-bg);
  padding: 0;
}
.bi-assistant-workbench__chat :deep(.agent-chat-panel) {
  height: 100%;
}
.bi-assistant-workbench__cards {
  display: flex;
  min-height: 100%;
  flex-direction: column;
  gap: 10px;
}
.bi-assistant-workbench__card {
  min-width: 0;
  cursor: grab;
}
.bi-assistant-workbench__card:active {
  cursor: grabbing;
}
.bi-assistant-workbench__card-actions {
  display: flex;
  align-items: center;
  gap: 2px;
}
.bi-assistant-workbench__card-foot {
  color: var(--el-text-color-secondary);
  font-size: 11px;
}
.bi-assistant-workbench__empty {
  display: flex;
  min-height: 160px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  border: 1px dashed var(--el-border-color);
  border-radius: var(--admin-page-radius);
}
.bi-assistant-workbench__vertical-label {
  writing-mode: vertical-rl;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  letter-spacing: 2px;
}
.bi-assistant-workbench__picker-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.bi-assistant-workbench__picker-head .el-input {
  max-width: 260px;
  margin-left: auto;
}
.bi-assistant-workbench__picker-list {
  display: flex;
  max-height: 52vh;
  flex-direction: column;
  gap: 6px;
  overflow: auto;
}
.bi-assistant-workbench__picker-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  height: auto;
  margin: 0;
  padding: 10px;
  color: var(--el-text-color-primary);
  text-align: left;
  background: var(--el-fill-color-lighter);
  border: 1px solid transparent;
  border-radius: var(--admin-page-radius);
  cursor: pointer;
  justify-content: flex-start;
}
.bi-assistant-workbench__picker-item:hover {
  border-color: var(--el-color-primary-light-5);
  background: var(--el-color-primary-light-9);
}
.bi-assistant-workbench__picker-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: var(--admin-page-radius);
}
.bi-assistant-workbench__picker-text {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}
.bi-assistant-workbench__picker-text strong {
  overflow: hidden;
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bi-assistant-workbench__picker-text small {
  color: var(--el-text-color-secondary);
  font-size: 11px;
}
@media (max-width: 1220px) {
  .bi-assistant-workbench__body {
    grid-template-columns: minmax(220px, 260px) minmax(420px, 1fr) minmax(220px, 260px);
  }
  .bi-assistant-workbench__brand {
    min-width: 0;
  }
  .bi-assistant-workbench__brand h1 {
    max-width: 180px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
@media (max-width: 980px) {
  .bi-assistant-workbench {
    height: auto;
    min-height: calc(100vh - 104px);
    overflow: visible;
  }
  .bi-assistant-workbench__topbar {
    flex-wrap: wrap;
  }
  .bi-assistant-workbench__search {
    flex: 1;
    width: auto;
  }
  .bi-assistant-workbench__body {
    grid-template-columns: 1fr 1fr;
    grid-template-rows: minmax(620px, 1fr) auto;
  }
  .bi-assistant-workbench__chat {
    grid-column: 1 / -1;
    grid-row: 1;
    min-height: 620px;
  }
  .bi-assistant-workbench__zone {
    grid-row: 2;
    min-height: 360px;
  }
}
@media (max-width: 620px) {
  .bi-assistant-workbench__body {
    display: flex;
    flex-direction: column;
  }
  .bi-assistant-workbench__chat {
    order: 1;
    min-height: 620px;
  }
  .bi-assistant-workbench__zone {
    order: 2;
    min-height: 300px;
    border: 0 !important;
    border-top: 1px solid var(--admin-page-divider-strong) !important;
  }
  .bi-assistant-workbench__zone.is-collapsed {
    width: auto;
    min-height: 48px;
    flex-direction: row;
  }
  .bi-assistant-workbench__zone.is-collapsed .bi-assistant-workbench__zone-head {
    width: 100%;
  }
  .bi-assistant-workbench__vertical-label {
    display: none;
  }
  .bi-assistant-workbench__session {
    flex: 1;
    width: auto;
  }
  .bi-assistant-workbench__search {
    order: 5;
    flex-basis: 100%;
  }
}
</style>
