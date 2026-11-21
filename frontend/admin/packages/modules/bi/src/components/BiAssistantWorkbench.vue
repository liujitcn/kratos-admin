<template>
  <el-container class="bi-assistant-workbench">
    <el-header
      class="bi-assistant-workbench__topbar"
      style="--el-header-height: auto; --el-header-padding: 0"
      @click="handleTopbarClick"
    >
      <el-button class="bi-assistant-workbench__icon-button" :title="t('bi.workbench.toolbox')" @click.stop="toggleToolbox">
        <el-icon><Grid /></el-icon>
        <span class="bi-assistant-workbench__dot" />
      </el-button>

      <div class="bi-assistant-workbench__session-wrap">
        <el-button
          class="bi-assistant-workbench__session-switch"
          :class="{ 'is-open': sessionMenuOpen }"
          @click.stop="toggleSessionMenu()"
        >
          <span>{{ activeSessionTitle }}</span>
          <el-icon><ArrowDown /></el-icon>
        </el-button>
        <div v-if="sessionMenuOpen" class="bi-assistant-workbench__session-menu" @click.stop>
          <div class="bi-assistant-workbench__session-search">
            <el-icon><Search /></el-icon>
            <el-input
              v-model="sessionKeyword"
              :placeholder="t('system.ai.chat.placeholder.search_conversation')"
              @keydown.esc="closeSessionMenu"
            />
          </div>
          <div class="bi-assistant-workbench__session-list">
            <template v-for="group in sessionGroups" :key="group.label">
              <div v-if="group.items.length" class="bi-assistant-workbench__group-label">{{ group.label }}</div>
              <div
                v-for="session in group.items"
                :key="session.id"
                class="bi-assistant-workbench__session-item"
                :class="{ 'is-active': session.id === activeSessionID }"
                role="button"
                tabindex="0"
                @click="selectSessionFromMenu(session)"
                @keydown.enter="selectSessionFromMenu(session)"
              >
                <span class="bi-assistant-workbench__session-text">
                  <span class="bi-assistant-workbench__session-title">{{ session.title }}</span>
                  <span class="bi-assistant-workbench__session-meta">{{
                    session.summary || t("system.ai.chat.value.default_conversation_title")
                  }}</span>
                </span>
                <el-dropdown
                  trigger="click"
                  placement="bottom-end"
                  @click.stop
                  @command="command => handleSessionMenuCommand(command, session)"
                >
                  <el-button
                    class="bi-assistant-workbench__session-more"
                    :title="t('system.ai.chat.action.conversation_menu')"
                    @click.stop
                  >
                    <el-icon><MoreFilled /></el-icon>
                  </el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="rename" :icon="EditPen">{{
                        t("system.ai.chat.action.rename")
                      }}</el-dropdown-item>
                      <el-dropdown-item command="pin" :icon="Star">
                        {{
                          pinnedSessionIDs.includes(session.id)
                            ? t("system.ai.chat.action.unpin")
                            : t("system.ai.chat.action.pin")
                        }}
                      </el-dropdown-item>
                      <el-dropdown-item command="delete" :icon="Delete" divided>{{ t("common.action.delete") }}</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </template>
            <div v-if="!sessionGroups.some(group => group.items.length)" class="bi-assistant-workbench__session-empty">
              {{ t("system.ai.chat.empty.no_match") }}
            </div>
          </div>
          <div class="bi-assistant-workbench__session-foot">
            <el-button class="bi-assistant-workbench__plain-button" @click="createSessionFromMenu">
              <el-icon><Plus /></el-icon>
              {{ t("system.ai.chat.action.new_conversation") }}
            </el-button>
          </div>
        </div>
      </div>

      <el-button
        class="bi-assistant-workbench__icon-button"
        :title="t('system.ai.chat.action.new_conversation')"
        @click.stop="createSession"
      >
        <el-icon><Plus /></el-icon>
      </el-button>

      <span class="bi-assistant-workbench__spacer" />

      <div v-if="toolboxOpen" class="bi-assistant-workbench__toolbox" @click.stop>
        <div class="bi-assistant-workbench__toolbox-title">{{ t("bi.workbench.toolbox_title") }}</div>
        <div class="bi-assistant-workbench__toolbox-grid">
          <el-button @click="openWorkbenchTool('files')">
            <span class="bi-assistant-workbench__toolbox-icon"
              ><el-icon><Document /></el-icon
            ></span>
            <span v-if="fileCount" class="bi-assistant-workbench__toolbox-badge">{{ fileCount }}</span>
            {{ t("bi.workbench.tool.files") }}
          </el-button>
          <el-button @click="openWorkbenchTool('automations')">
            <span class="bi-assistant-workbench__toolbox-icon is-warning"
              ><el-icon><Timer /></el-icon
            ></span>
            <span v-if="pendingAutomationCount" class="bi-assistant-workbench__toolbox-badge">{{ pendingAutomationCount }}</span>
            <span v-else-if="automationTotal" class="bi-assistant-workbench__toolbox-badge is-warning">{{
              automationTotal
            }}</span>
            <span v-else class="bi-assistant-workbench__toolbox-badge is-warning-dot" />
            {{ t("bi.workbench.tool.automations") }}
          </el-button>
          <el-button @click="openWorkbenchTool('plugins')">
            <span class="bi-assistant-workbench__toolbox-icon is-plugin"
              ><el-icon><Grid /></el-icon
            ></span>
            <span v-if="pluginCount" class="bi-assistant-workbench__toolbox-badge">{{ pluginCount }}</span>
            {{ t("bi.workbench.tool.plugins") }}
          </el-button>
        </div>
      </div>
    </el-header>

    <el-main
      class="bi-assistant-workbench__body"
      :class="{ 'is-left-collapsed': collapsed.left, 'is-right-collapsed': collapsed.right }"
    >
      <el-aside class="bi-assistant-workbench__zone is-left" :class="{ 'is-collapsed': collapsed.left }">
        <el-header class="bi-assistant-workbench__zone-head">
          <el-button
            text
            circle
            :icon="collapsed.left ? DArrowRight : DArrowLeft"
            :title="collapsed.left ? t('bi.workbench.side.expand_left') : t('bi.workbench.side.collapse_left')"
            @click="toggleZone('left')"
          />
          <span v-if="!collapsed.left" class="bi-assistant-workbench__spacer" />
          <el-button v-if="!collapsed.left" text type="primary" :icon="Plus" @click="openCardPicker('left')">{{
            t("bi.card.add_short")
          }}</el-button>
        </el-header>
        <div v-if="!collapsed.left" class="bi-assistant-workbench__zone-body" @dragover.prevent @drop.prevent="dropCard('left')">
          <BiAssistantCardZone
            zone="left"
            :cards="leftCards"
            :refresh-keys="refreshKeys"
            @drag-start="startCardDrag"
            @drag-end="dragged = undefined"
            @drop="dropCard"
            @refresh="refreshCard"
            @quote="quoteCard"
            @remove="removeCard"
            @preview="previewCard"
          />
        </div>
        <span v-else class="bi-assistant-workbench__vertical-label">{{ t("bi.workbench.side.left") }}</span>
      </el-aside>

      <el-main class="bi-assistant-workbench__chat">
        <ChatPanel
          v-show="workbenchView === 'chat'"
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
            <AiFlowBlocks
              v-if="message.blocks?.length"
              :message="message"
              :active-flow-message-id="activeFlowMessageId"
              :renderers="flowBlockRenderers"
              @flow-action="handleFlowAction"
            />
          </template>
        </ChatPanel>
      </el-main>

      <el-aside class="bi-assistant-workbench__zone is-right" :class="{ 'is-collapsed': collapsed.right }">
        <el-header class="bi-assistant-workbench__zone-head">
          <el-button v-if="!collapsed.right" text type="primary" :icon="Plus" @click="openCardPicker('right')">{{
            t("bi.card.add_short")
          }}</el-button>
          <span v-if="!collapsed.right" class="bi-assistant-workbench__spacer" />
          <el-button
            text
            circle
            :icon="collapsed.right ? DArrowLeft : DArrowRight"
            :title="collapsed.right ? t('bi.workbench.side.expand_right') : t('bi.workbench.side.collapse_right')"
            @click="toggleZone('right')"
          />
        </el-header>
        <div
          v-if="!collapsed.right"
          class="bi-assistant-workbench__zone-body"
          @dragover.prevent
          @drop.prevent="dropCard('right')"
        >
          <BiAssistantCardZone
            zone="right"
            :cards="rightCards"
            :refresh-keys="refreshKeys"
            @drag-start="startCardDrag"
            @drag-end="dragged = undefined"
            @drop="dropCard"
            @refresh="refreshCard"
            @quote="quoteCard"
            @remove="removeCard"
            @preview="previewCard"
          />
        </div>
        <span v-else class="bi-assistant-workbench__vertical-label">{{ t("bi.workbench.side.right") }}</span>
      </el-aside>

      <!-- 紧凑布局浮层遮罩：点击收起已展开的卡片区 -->
      <div v-if="zoneScrimVisible" class="bi-assistant-workbench__zone-scrim" @click="collapseAllZones" />

      <!-- 工作台工具全屏覆盖层：打开时覆盖左右工作台与对话区，不再跳转 AI 对话页 -->
      <div v-show="workbenchView !== 'chat'" class="bi-assistant-workbench__tool-view">
        <el-header class="bi-assistant-workbench__tool-head" style="--el-header-height: auto; --el-header-padding: 0">
          <span class="bi-assistant-workbench__tool-title">{{ workbenchViewTitle }}</span>
          <span class="bi-assistant-workbench__spacer" />
          <el-button text :icon="Close" @click="workbenchView = 'chat'">{{ t("bi.workbench.action.back_to_chat") }}</el-button>
        </el-header>
        <div class="bi-assistant-workbench__tool-body">
          <!-- v-if 按需挂载：避免表格在隐藏状态渲染导致列宽塌缩，切换时自动重新加载 -->
          <AutomationPanel
            v-if="workbenchView === 'automations'"
            :active="true"
            :mobile="mobile"
            :model-providers="modelProviders"
            :knowledge-options="knowledgeOptions"
            :plugins="plugins"
            :connections="connections"
            :provider-id="selectedProviderId"
            :model-name="selectedModelName"
            :knowledge-ids="selectedKnowledgeIds"
            :selected-plugins="activeSession?.plugins || []"
            @pending="pendingAutomationCount = $event"
            @result="openResultSession"
            @session="openDiscussion"
          />
          <FilePanel v-if="workbenchView === 'files'" />
          <PluginPanel
            v-if="workbenchView === 'plugins'"
            :plugins="plugins"
            :connections="connections"
            :mobile="mobile"
            :loading="pluginsLoading"
            :error="pluginsError"
            :can-manage-tools="pluginCapabilities.can_manage_tools"
            :can-manage-catalog="pluginCapabilities.can_manage_catalog"
            :can-create-private="pluginCapabilities.can_create_private"
            @refresh="refreshPluginTools"
            @select="addSessionPlugin"
          />
        </div>
      </div>
    </el-main>

    <el-dialog
      v-model="cardPicker.open"
      class="bi-assistant-workbench__picker-dialog"
      :title="t('bi.workbench.card.library_title')"
      width="min(1440px, calc(100vw - 48px))"
      destroy-on-close
    >
      <div class="bi-assistant-workbench__picker-subrow">
        <el-input v-model="cardPicker.keyword" clearable :prefix-icon="Search" :placeholder="t('bi.card.search')" />
        <div class="bi-assistant-workbench__picker-target">
          <span class="bi-assistant-workbench__picker-label">{{ t("bi.workbench.card.add_to") }}</span>
          <div class="bi-assistant-workbench__zone-picker">
            <el-button :class="{ active: cardPicker.zone === 'left' }" @click="cardPicker.zone = 'left'">{{
              t("bi.workbench.side.left")
            }}</el-button>
            <el-button :class="{ active: cardPicker.zone === 'right' }" @click="cardPicker.zone = 'right'">{{
              t("bi.workbench.side.right")
            }}</el-button>
          </div>
        </div>
      </div>
      <div v-loading="cardPicker.loading" class="bi-assistant-workbench__picker-list">
        <div
          v-for="card in pickerVisibleCards"
          :key="card.id"
          class="bi-assistant-workbench__picker-item"
          :class="{ 'is-added': isCardAdded(card) }"
          :aria-disabled="isCardAdded(card)"
          role="button"
          tabindex="0"
          @click="addCard(card)"
          @keydown.enter.prevent="addCard(card)"
          @keydown.space.prevent="addCard(card)"
        >
          <BiDataCard :card="card" :show-category="false" compact />
        </div>
        <el-empty v-if="!cardPicker.loading && !pickerCards.length" :description="t('bi.card.empty')" />
      </div>
      <el-pagination
        v-if="pickerCards.length > pickerPageSize"
        v-model:current-page="cardPicker.page"
        class="bi-assistant-workbench__picker-pagination"
        :page-size="pickerPageSize"
        :total="pickerCards.length"
        layout="prev, pager, next"
      />
    </el-dialog>
    <el-dialog
      v-model="cardPreview.open"
      class="bi-assistant-workbench__preview-dialog"
      :title="cardPreview.card?.name || t('bi.card.detail')"
      width="min(780px, calc(100vw - 64px))"
      destroy-on-close
    >
      <BiDataCard v-if="cardPreview.card" :card="cardPreview.card" :show-category="false" />
    </el-dialog>
  </el-container>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowDown,
  Close,
  CopyDocument,
  DArrowLeft,
  DArrowRight,
  Delete,
  Document,
  EditPen,
  Grid,
  MoreFilled,
  Plus,
  Refresh,
  Search,
  Star,
  Timer
} from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import { useMediaQuery } from "@vueuse/core";
import { defAiFileService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_file";
import { AiFileView } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_file";
import { defAiAutomationService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_automation";
import { defAiKnowledgeService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_knowledge";
import { defAiMessageService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_message";
import { defAiModelService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_provider";
import { defAiPluginService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_plugin";
import { defAiSessionService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_session";
import { defAiToolService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_tool";
import { AiMessageStatus } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import type { AiSession } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import { Terminal } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_tool";
import type { AiAction } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_message";
import type { AiPlugin, AiPluginConnection, AiPluginSelection } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_plugin";
import { pluginSelectionKey } from "@liujitcn/kratos-admin-system/components/ai/workspace/state";
import { getAdminAiFlowBlockRenderers } from "@liujitcn/kratos-admin-system";
import AiFlowBlocks from "@liujitcn/kratos-admin-system/components/ai/AiFlowBlocks.vue";
import ChatPanel from "@liujitcn/kratos-admin-system/components/ai/ChatPanel.vue";
import AutomationPanel from "@liujitcn/kratos-admin-system/components/ai/workspace/AutomationPanel.vue";
import FilePanel from "@liujitcn/kratos-admin-system/components/ai/workspace/FilePanel.vue";
import PluginPanel from "@liujitcn/kratos-admin-system/components/ai/workspace/PluginPanel.vue";
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
import type {
  ChatMessageAction,
  ChatMessageEditPayload,
  ChatMessageItem,
  SubmitPayload
} from "@liujitcn/kratos-admin-system/components/ai/chat";
import {
  resolveModelSelection,
  saveModelSelection
} from "@liujitcn/kratos-admin-system/components/AiModelSelector/modelSelection";
import type { BiCard, BiWorkbench } from "../rpc/bi/admin/v1/bi_card";
import { defBiCardService } from "../api/bi/admin/v1/bi_card";
import BiDataCard from "./BiDataCard.vue";
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
const mobile = useMediaQuery("(max-width: 768px)");
/** 紧凑布局：小屏下卡片区默认收起、展开时以浮层覆盖聊天区 */
const compact = useMediaQuery("(max-width: 1280px)");
const sessionKeyword = ref("");
const sessionMenuOpen = ref(false);
const toolboxOpen = ref(false);
const workbenchView = ref<"chat" | "files" | "automations" | "plugins">("chat");
const pendingAutomationCount = ref(0);
/** 工具箱「自动化」角标：本人非归档任务总数。 */
const automationTotal = ref(0);
/** 工具箱「插件」角标：当前身份可见插件数量。 */
const pluginCount = ref(0);
/** 工具箱「文件」角标：AI 文件总数（全部视图）。 */
const fileCount = ref(0);

/** 查询自动化与插件统计，供工具箱角标展示。 */
async function loadToolCounts() {
  try {
    const response = await defAiAutomationService.CountAiAutomation({});
    automationTotal.value = response.total;
    pendingAutomationCount.value = response.pending_count;
  } catch {
    // 角标仅做提示，查询失败时保持现有数值。
  }
  try {
    const response = await defAiPluginService.CountAiPlugin({});
    pluginCount.value = response.total;
  } catch {
    // 角标仅做提示，查询失败时保持现有数值。
  }
}

/** 插件面板刷新后同步目录与角标统计。 */
async function refreshPluginTools() {
  await Promise.all([loadPlugins(), loadToolCounts()]);
}

/** 查询 AI 文件总数，供工具箱角标展示。 */
async function loadFileCount() {
  try {
    const response = await defAiFileService.CountAiFile({ view: AiFileView.AI_FILE_VIEW_ALL, keyword: "" });
    fileCount.value = response.total;
  } catch {
    // 角标仅做提示，查询失败时保持现有数值。
  }
}
const plugins = ref<AiPlugin[]>([]),
  connections = ref<AiPluginConnection[]>([]);
const pluginCapabilities = ref({ can_manage_tools: false, can_manage_catalog: false, can_create_private: false });
const pluginsLoading = ref(false),
  pluginsError = ref("");
const savingPlugins = ref(false),
  pluginSelection = ref<AiPluginSelection[]>([]);
const pinnedSessionIDs = ref<string[]>(readPinnedSessionIDs());
const leftCards = ref<BiCard[]>([]);
const rightCards = ref<BiCard[]>([]);
const allCards = ref<BiCard[]>([]);
const refreshKeys = reactive<Record<number, number>>({});
const dragged = ref<CardDrag>();
const collapsed = reactive({ left: false, right: false });
const pickerPageSize = 12;
const cardPicker = reactive({ open: false, loading: false, allLoaded: false, keyword: "", page: 1, zone: "right" as Zone });
const cardPreview = reactive<{ open: boolean; card?: BiCard }>({ open: false });
const workbench = ref<BiWorkbench>({ dashboard_ids: [], favorite_ids: [], dashboards: [] });
// shortcut: 左右分区暂存在浏览器本地，跨设备同步时再扩展 BiWorkbench 协议。
const zoneStorageKey = "kratos-admin:bi-assistant-workbench:zones";

const visibleSessions = computed(() => {
  const keyword = sessionKeyword.value.trim().toLowerCase();
  const filtered = keyword
    ? sessions.value.filter(item => (item.title + " " + item.summary).toLowerCase().includes(keyword))
    : sessions.value;
  return [...filtered].sort((left, right) => {
    const pinDiff = Number(pinnedSessionIDs.value.includes(right.id)) - Number(pinnedSessionIDs.value.includes(left.id));
    return pinDiff || sessionTime(right).getTime() - sessionTime(left).getTime();
  });
});
const activeSessionTitle = computed(
  () => sessions.value.find(item => item.id === activeSessionID.value)?.title || t("system.ai.chat.action.new_conversation")
);
const sessionGroups = computed(() => {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const yesterday = today - 86_400_000;
  const groups = [
    { label: t("system.ai.chat.group.today"), items: [] as AiSession[] },
    { label: t("system.ai.chat.group.yesterday"), items: [] as AiSession[] },
    { label: t("system.ai.chat.group.earlier"), items: [] as AiSession[] }
  ];
  visibleSessions.value.forEach(item => {
    const time = sessionTime(item).getTime();
    const target = time >= today ? groups[0] : time >= yesterday ? groups[1] : groups[2];
    target.items.push(item);
  });
  return groups;
});
const workbenchViewTitle = computed(() => {
  const titles = {
    files: t("bi.workbench.tool.files"),
    automations: t("bi.workbench.tool.automations"),
    plugins: t("bi.workbench.tool.plugins")
  } as const;
  return workbenchView.value === "chat" ? "" : titles[workbenchView.value];
});
const pickerCards = computed(() => {
  const keyword = cardPicker.keyword.trim().toLowerCase();
  return allCards.value.filter(
    item => !keyword || (item.name + " " + item.source_code + " " + item.chart_type).toLowerCase().includes(keyword)
  );
});
const pickerVisibleCards = computed(() => {
  const start = (cardPicker.page - 1) * pickerPageSize;
  return pickerCards.value.slice(start, start + pickerPageSize);
});

/** 返回卡片在当前工作台中的加入状态。 */
function isCardAdded(card: BiCard) {
  return leftCards.value.some(item => item.id === card.id) || rightCards.value.some(item => item.id === card.id);
}

/** 读取工作台会话置顶状态。 */
function readPinnedSessionIDs() {
  try {
    const value = JSON.parse(localStorage.getItem("kratos-admin:bi-assistant:pinned-sessions") || "[]");
    return Array.isArray(value) ? value.filter(item => typeof item === "string") : [];
  } catch {
    return [];
  }
}

/** 返回会话更新时间，兼容 proto Timestamp 的字符串和数字字段。 */
function sessionTime(session: AiSession) {
  const seconds = Number(session.updated_at?.seconds ?? 0);
  const nanos = Number(session.updated_at?.nanos ?? 0);
  return new Date(seconds * 1000 + nanos / 1_000_000);
}

/** 持久化工作台会话置顶状态。 */
function persistPinnedSessionIDs() {
  localStorage.setItem("kratos-admin:bi-assistant:pinned-sessions", JSON.stringify(pinnedSessionIDs.value));
}

/** 关闭工作台顶部的所有弹层。 */
function closeTopbarPopovers() {
  sessionMenuOpen.value = false;
  toolboxOpen.value = false;
}

/** 处理顶部空白区域点击，关闭已打开的弹层。 */
function handleTopbarClick() {
  closeTopbarPopovers();
}

/** 切换会话菜单。 */
function toggleSessionMenu(force?: boolean) {
  sessionMenuOpen.value = force ?? !sessionMenuOpen.value;
  if (sessionMenuOpen.value) {
    toolboxOpen.value = false;
  }
}

/** 关闭会话菜单。 */
function closeSessionMenu() {
  sessionMenuOpen.value = false;
}

/** 切换工具箱面板。 */
function toggleToolbox() {
  toolboxOpen.value = !toolboxOpen.value;
  sessionMenuOpen.value = false;
}

/** 打开工作台工具面板，内容直接内嵌展示，不再跳转 AI 对话页。 */
function openWorkbenchTool(view: "files" | "automations" | "plugins") {
  toolboxOpen.value = false;
  sessionMenuOpen.value = false;
  // 紧凑布局下先收起浮层卡片，避免遮挡工具面板
  if (compact.value) collapseAllZones();
  if (view === "files") void loadFileCount();
  workbenchView.value = view;
}

/** 自动化按管理员终端读取目录；插件目录同样固定管理端。 */
const pluginTerminal = computed(() => Terminal.TERMINAL_ADMIN);

/** 按终端读取插件目录、连接账号与当前账号的能力开关。 */
async function loadPlugins() {
  if (pluginsLoading.value) return;
  pluginsLoading.value = true;
  pluginsError.value = "";
  plugins.value = [];
  try {
    const [catalog, accounts] = await Promise.all([
      defAiPluginService.ListAiPlugin({ keyword: "", terminal: pluginTerminal.value }),
      defAiPluginService.ListAiPluginConnection({ plugin_id: "" })
    ]);
    plugins.value = catalog.plugins ?? [];
    connections.value = accounts.connections ?? [];
    pluginCapabilities.value = {
      can_manage_tools: catalog.can_manage_tools,
      can_manage_catalog: catalog.can_manage_catalog,
      can_create_private: catalog.can_create_private
    };
  } catch (error) {
    pluginsError.value = error instanceof Error ? error.message : t("bi.query.error");
  } finally {
    pluginsLoading.value = false;
  }
}

/** 保存显式插件选择，空数组表示关闭所有插件。 */
async function savePluginSelection() {
  if (pluginsError.value || pluginsLoading.value || activeSession.value?.read_only) return;
  savingPlugins.value = true;
  try {
    const id = await ensureSession();
    if (!id) return;
    const response = await defAiSessionService.SetAiSessionPlugins({ id, plugins: pluginSelection.value });
    if (response.session) upsertSession(normalizeSession(response.session));
  } finally {
    savingPlugins.value = false;
  }
}

/** 从插件目录选用确定账号并回到对话视图。 */
async function addSessionPlugin(selection: AiPluginSelection) {
  if (activeSession.value?.read_only) {
    ElMessage.warning(t("system.ai.chat.message.readonly_plugin_change"));
    return;
  }
  pluginSelection.value = [...(activeSession.value?.plugins ?? [])];
  if (!pluginSelection.value.some(item => pluginSelectionKey(item) === pluginSelectionKey(selection)))
    pluginSelection.value.push(selection);
  await savePluginSelection();
  workbenchView.value = "chat";
}

/** 打开自动化执行结果会话并回到对话视图。 */
async function openResultSession(id: string) {
  workbenchView.value = "chat";
  activeSessionID.value = id;
  await selectSession(id);
}

/** 从执行结果继续讨论时切换到服务端新建的普通会话。 */
function openDiscussion(session: AiSession) {
  const normalized = normalizeSession(session);
  upsertSession(normalized);
  void openResultSession(normalized.id);
}

/** 从顶部菜单切换会话并加载消息。 */
async function selectSessionFromMenu(session: AiSession) {
  closeSessionMenu();
  activeSessionID.value = session.id;
  await selectSession(session.id);
}

/** 处理会话重命名、置顶和删除。 */
async function handleSessionMenuCommand(command: string | number | object, session: AiSession) {
  try {
    if (String(command) === "pin") {
      pinnedSessionIDs.value = pinnedSessionIDs.value.includes(session.id)
        ? pinnedSessionIDs.value.filter(id => id !== session.id)
        : [session.id, ...pinnedSessionIDs.value];
      persistPinnedSessionIDs();
      return;
    }
    if (String(command) === "rename") {
      const result = await ElMessageBox.prompt(
        t("system.ai.chat.placeholder.conversation_name"),
        t("system.ai.chat.dialog.rename_title"),
        {
          inputValue: session.title,
          confirmButtonText: t("common.action.confirm"),
          cancelButtonText: t("common.action.cancel")
        }
      );
      const response = await defAiSessionService.UpdateAiSession({ id: session.id, title: result.value.trim() });
      if (response.session) upsertSession(normalizeSession(response.session));
      return;
    }
    if (String(command) !== "delete") return;
    await ElMessageBox.confirm(
      t("system.ai.chat.dialog.delete_description", { title: session.title }),
      t("system.ai.chat.dialog.delete_title"),
      {
        confirmButtonText: t("common.action.delete"),
        cancelButtonText: t("common.action.cancel"),
        type: "warning"
      }
    );
    await defAiSessionService.DeleteAiSession({ id: session.id });
    sessions.value = sessions.value.filter(item => item.id !== session.id);
    if (activeSessionID.value === session.id) {
      activeSessionID.value = sessions.value[0]?.id || "";
      if (activeSessionID.value) await selectSession(activeSessionID.value);
    }
  } catch (cause) {
    if (cause === "cancel" || cause === "close") return;
    ElMessage.error(cause instanceof Error ? cause.message : t("system.ai.chat.message.operation_failed"));
  }
}

/** 从会话弹层创建新会话。 */
async function createSessionFromMenu() {
  closeSessionMenu();
  await createSession();
}

/** 紧凑布局下是否有卡片区处于浮层展开态（用于显示遮罩） */
const zoneScrimVisible = computed(() => compact.value && (!collapsed.left || !collapsed.right) && workbenchView.value === "chat");

/** 收起两侧卡片区（遮罩点击 / 进入紧凑布局时调用） */
function collapseAllZones() {
  collapsed.left = true;
  collapsed.right = true;
}

function toggleZone(zone: Zone) {
  const willOpen = collapsed[zone];
  // 紧凑布局：展开一侧时先收起另一侧，避免两个浮层叠加遮挡
  if (willOpen && compact.value) collapsed[zone === "left" ? "right" : "left"] = true;
  collapsed[zone] = !collapsed[zone];
}

/** 仅跨越断点时重置卡片区布局；小屏内 resize 不打断用户已展开的浮层 */
watch(
  compact,
  isCompact => {
    if (isCompact) collapseAllZones();
    else {
      collapsed.left = false;
      collapsed.right = false;
    }
  },
  { immediate: true }
);
watch(
  () => cardPicker.keyword,
  () => {
    cardPicker.page = 1;
  }
);

async function loadCards() {
  const [config, page] = await Promise.all([
    defBiCardService.GetBiWorkbench({}),
    defBiCardService.PageBiCard({
      name: "",
      source_code: "",
      category: "favorite",
      search_source_codes: [],
      page_num: 1,
      page_size: 100
    })
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
  localStorage.setItem(
    zoneStorageKey,
    JSON.stringify({
      left: leftCards.value.map(item => item.id),
      right: rightCards.value.map(item => item.id)
    } satisfies StoredZoneLayout)
  );
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

/** 打开卡片放大查看弹窗。 */
function previewCard(card: BiCard) {
  cardPreview.card = card;
  cardPreview.open = true;
}

function quoteCard(card: BiCard) {
  if (!activeSessionID.value) return;
  drafts.value[activeSessionID.value] = {
    ...(drafts.value[activeSessionID.value] || { text: "", attachments: [] }),
    text: t("bi.workbench.card.quote_prefix", { name: card.name }) + " "
  };
  ElMessage.success(t("bi.workbench.card.quote_success"));
}

async function openCardPicker(zone: Zone) {
  cardPicker.zone = zone;
  cardPicker.keyword = "";
  cardPicker.page = 1;
  cardPicker.open = true;
  if (cardPicker.allLoaded) return;
  cardPicker.loading = true;
  try {
    const page = await defBiCardService.PageBiCard({
      name: "",
      source_code: "",
      category: "",
      search_source_codes: [],
      page_num: 1,
      page_size: 100
    });
    allCards.value = page.cards;
    cardPicker.allLoaded = true;
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    cardPicker.loading = false;
  }
}

async function addCard(card: BiCard) {
  if (isCardAdded(card)) return;
  const target = cardPicker.zone === "left" ? leftCards : rightCards;
  target.value.push(card);
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
  closeTopbarPopovers();
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
    if (!window.speechSynthesis) return ElMessage.warning(t("system.ai.chat.message.speech_unsupported"));
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
      title: t("system.ai.chat.value.branch_title", { title: payload.item.content.slice(0, 18) }),
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
  document.addEventListener("click", closeTopbarPopovers);
  try {
    await Promise.all([loadCards(), loadChat(), loadPlugins(), loadFileCount(), loadToolCounts()]);
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  }
});

onBeforeUnmount(() => document.removeEventListener("click", closeTopbarPopovers));
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
  position: relative;
  box-sizing: border-box;
  display: flex;
  flex: none;
  gap: 6px;
  align-items: center;
  min-height: 42px;
  padding: 4px 10px;
  border-bottom: 1px solid var(--admin-page-divider-strong);
}
.bi-assistant-workbench__session-wrap {
  position: relative;
  display: inline-flex;
  flex: none;
}
.bi-assistant-workbench__session-switch {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  max-width: 260px;
  height: 32px;
  padding: 0 8px;
  font-size: 13px;
  font-weight: 500;
  color: var(--admin-page-text-primary);
  text-align: left;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
}
.bi-assistant-workbench__session-switch:hover,
.bi-assistant-workbench__session-switch.is-open {
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-7);
}
.bi-assistant-workbench__session-switch > span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bi-assistant-workbench__session-switch .el-icon {
  flex: none;
  color: var(--admin-page-text-placeholder);
  transition: transform 0.2s;
}
.bi-assistant-workbench__session-switch.is-open .el-icon {
  color: var(--el-color-primary);
  transform: rotate(180deg);
}
.bi-assistant-workbench__icon-button {
  position: relative;
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  color: var(--admin-page-text-secondary);
  background: transparent;
  border: 0;
  border-radius: 4px;
}
.bi-assistant-workbench__icon-button:hover {
  color: var(--admin-page-text-primary);
  background: var(--el-fill-color-light);
}
.bi-assistant-workbench__dot {
  position: absolute;
  top: 5px;
  right: 5px;
  width: 7px;
  height: 7px;
  background: var(--el-color-danger);
  border-radius: 50%;
}
.bi-assistant-workbench__spacer {
  flex: 1;
  min-width: 0;
}
.bi-assistant-workbench__session-menu,
.bi-assistant-workbench__toolbox {
  position: absolute;
  z-index: 30;
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-lighter);
  box-shadow: 0 8px 24px rgb(15 23 42 / 14%);
}
.bi-assistant-workbench__session-menu {
  top: 40px;
  left: 0;
  display: flex;
  flex-direction: column;
  width: min(320px, calc(100vw - 32px));
  max-height: 440px;
  overflow: hidden;
  border-radius: var(--admin-page-radius);
}
.bi-assistant-workbench__session-search {
  display: flex;
  gap: 6px;
  align-items: center;
  height: 30px;
  padding: 0 10px;
  margin: 10px 10px 6px;
  color: var(--admin-page-text-placeholder);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}
.bi-assistant-workbench__session-search .el-input {
  flex: 1;
  min-width: 0;
}
.bi-assistant-workbench__session-search :deep(.el-input__wrapper) {
  padding: 0;
  background: transparent;
  box-shadow: none;
}
.bi-assistant-workbench__session-search input {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  outline: 0;
  background: transparent;
  border: 0;
}
.bi-assistant-workbench__session-list {
  flex: 1;
  padding: 0 6px 6px;
  overflow-y: auto;
}
.bi-assistant-workbench__group-label {
  padding: 10px 6px 3px;
  font-size: 11px;
  color: var(--admin-page-text-placeholder);
}
.bi-assistant-workbench__session-item {
  display: flex;
  gap: 6px;
  align-items: center;
  width: 100%;
  min-width: 0;
  padding: 8px 8px 8px 10px;
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--admin-page-radius);
}
.bi-assistant-workbench__session-item:hover,
.bi-assistant-workbench__session-item.is-active {
  background: var(--el-color-primary-light-9);
}
.bi-assistant-workbench__session-text {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.bi-assistant-workbench__session-title,
.bi-assistant-workbench__session-meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bi-assistant-workbench__session-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--admin-page-text-primary);
}
.bi-assistant-workbench__session-item.is-active .bi-assistant-workbench__session-title {
  color: var(--el-color-primary);
}
.bi-assistant-workbench__session-meta {
  font-size: 11px;
  color: var(--admin-page-text-placeholder);
}
.bi-assistant-workbench__session-more {
  display: inline-flex;
  flex: none;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  color: var(--admin-page-text-placeholder);
  background: transparent;
  border: 0;
  border-radius: 4px;
  opacity: 0;
}
.bi-assistant-workbench__session-item:hover .bi-assistant-workbench__session-more,
.bi-assistant-workbench__session-more:focus-visible {
  opacity: 1;
}
.bi-assistant-workbench__session-more:hover {
  color: var(--admin-page-text-regular);
  background: var(--el-fill-color);
}
.bi-assistant-workbench__session-empty {
  padding: 28px 10px;
  font-size: 12px;
  color: var(--admin-page-text-placeholder);
  text-align: center;
}
.bi-assistant-workbench__session-foot {
  padding: 8px;
  border-top: 1px solid var(--admin-page-divider);
}
.bi-assistant-workbench__plain-button {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 30px;
  padding: 0;
  color: var(--admin-page-text-regular);
  background: #ffffff;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}
.bi-assistant-workbench__plain-button:hover {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-7);
}
.bi-assistant-workbench__toolbox {
  top: 100%;
  left: 0;
  box-sizing: border-box;
  width: min(288px, calc(100% - 20px));
  padding: 12px;
  border-radius: 8px;
}
.bi-assistant-workbench__toolbox-title {
  margin: 0 2px 8px;
  font-size: 12px;
  color: var(--admin-page-text-placeholder);
}
.bi-assistant-workbench__toolbox-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
}
.bi-assistant-workbench__toolbox-grid button {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
  height: auto;
  padding: 12px 4px 10px;
  font-size: 12px;
  font-weight: 400;
  color: var(--admin-page-text-regular);
  background: transparent;
  border: 0;
  border-radius: 6px;
}
.bi-assistant-workbench__toolbox-grid button:hover {
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
}
.bi-assistant-workbench__toolbox-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-radius: 8px;
}
.bi-assistant-workbench__toolbox-icon.is-warning {
  color: var(--el-color-warning);
  background: rgb(230 162 60 / 12%);
}
.bi-assistant-workbench__toolbox-icon.is-plugin {
  color: #13a8a0;
  background: rgb(0 150 136 / 10%);
}
.bi-assistant-workbench__toolbox-badge {
  position: absolute;
  top: 6px;
  right: 8px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  font-size: 10px;
  line-height: 16px;
  color: #ffffff;
  text-align: center;
  background: var(--el-color-danger);
  border-radius: 8px;
}
.bi-assistant-workbench__toolbox-badge.is-warning {
  background: var(--el-color-warning);
}
.bi-assistant-workbench__toolbox-badge.is-warning-dot {
  width: 8px;
  min-width: 8px;
  padding: 0;
  background: var(--el-color-warning);
}
.bi-assistant-workbench__body {
  /* 左右工作台列宽：沿用项目弹性区间，上限对齐原型 360px；折叠态 44px */
  --zone-left-w: minmax(248px, 360px);
  --zone-right-w: minmax(248px, 360px);

  position: relative;
  display: grid;
  flex: 1;
  grid-template-columns: var(--zone-left-w) minmax(460px, 1fr) var(--zone-right-w);
  min-height: 0;
  padding: 0;
}
.bi-assistant-workbench__body.is-left-collapsed {
  --zone-left-w: 44px;
}
.bi-assistant-workbench__body.is-right-collapsed {
  --zone-right-w: 44px;
}
.bi-assistant-workbench__zone {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  padding: 0;
  background: var(--el-fill-color-light);
}
.bi-assistant-workbench__zone.is-left {
  border-right: 1px solid var(--admin-page-divider-strong);
}
.bi-assistant-workbench__zone.is-right {
  border-left: 1px solid var(--admin-page-divider-strong);
}
.bi-assistant-workbench__zone.is-collapsed {
  align-items: center;
  width: 44px;
  background: var(--admin-page-card-bg);
}
.bi-assistant-workbench__zone-head {
  display: flex;
  flex: none;
  gap: 6px;
  align-items: center;
  height: 48px;
  min-height: 48px;
  padding: 0 10px;
  background: var(--admin-page-card-bg);
  border-bottom: 1px solid var(--admin-page-divider);
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
  padding: 0;
  background: var(--admin-page-card-bg);
}
.bi-assistant-workbench__chat :deep(.agent-chat-panel) {
  height: 100%;
}
.bi-assistant-workbench__tool-view {
  position: absolute;
  inset: 0;
  z-index: 5;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--admin-page-card-bg);
}
.bi-assistant-workbench__tool-head {
  display: flex;
  flex: none;
  gap: 6px;
  align-items: center;
  min-height: 48px;
  padding: 0 10px;
  border-bottom: 1px solid var(--admin-page-divider);
}
.bi-assistant-workbench__tool-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.bi-assistant-workbench__tool-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.bi-assistant-workbench__tool-body > * {
  height: 100%;
}
.bi-assistant-workbench__cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 100%;
}
.bi-assistant-workbench__card {
  flex: none;
  min-width: 0;
  cursor: grab;
}
.bi-assistant-workbench__card:active {
  cursor: grabbing;
}
.bi-assistant-workbench__card-actions {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: nowrap;
  width: max-content;
  min-width: max-content;
  gap: 0;
  align-items: center;
  justify-content: flex-start;
  margin-left: auto;
  white-space: nowrap;
}
.bi-assistant-workbench__card-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
.bi-assistant-workbench__card-actions :deep(.el-tooltip__trigger) {
  flex: 0 0 auto;
}
.bi-assistant-workbench__card-foot {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.bi-assistant-workbench__empty {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: center;
  justify-content: center;
  min-height: 160px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  border: 1px dashed var(--el-border-color);
  border-radius: var(--admin-page-radius);
}
.bi-assistant-workbench__vertical-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  letter-spacing: 2px;
  writing-mode: vertical-rl;
}
.bi-assistant-workbench__picker-dialog :deep(.el-dialog) {
  overflow: hidden;
  border-radius: var(--admin-page-radius);
}
.bi-assistant-workbench__picker-dialog :deep(.el-dialog__header) {
  padding: 16px 20px 4px;
  margin-right: 0;
}
.bi-assistant-workbench__picker-dialog :deep(.el-dialog__title) {
  font-size: 16px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}
.bi-assistant-workbench__picker-dialog :deep(.el-dialog__body) {
  max-height: 72vh;
  padding: 0 20px 24px;
  overflow-y: auto;
}
.bi-assistant-workbench__preview-dialog :deep(.el-dialog) {
  overflow: hidden;
  border-radius: var(--admin-page-radius);
}
.bi-assistant-workbench__preview-dialog :deep(.el-dialog__header) {
  padding: 16px 20px 4px;
  margin-right: 0;
}
.bi-assistant-workbench__preview-dialog :deep(.el-dialog__body) {
  padding: 8px 20px 24px;
}
.bi-assistant-workbench__picker-subrow {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--admin-page-text-placeholder);
}
.bi-assistant-workbench__picker-target {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}
.bi-assistant-workbench__picker-label {
  color: var(--admin-page-text-secondary);
  white-space: nowrap;
}
.bi-assistant-workbench__zone-picker {
  display: inline-flex;
}
.bi-assistant-workbench__zone-picker button {
  height: 28px;
  padding: 0 14px;
  font-size: 12px;
  font-weight: 400;
  color: var(--el-text-color-regular);
  white-space: nowrap;
  background: #ffffff;
  border: 1px solid var(--el-border-color);
}
.bi-assistant-workbench__zone-picker button:first-child {
  border-radius: 4px 0 0 4px;
}
.bi-assistant-workbench__zone-picker button:last-child {
  margin-left: -1px;
  border-radius: 0 4px 4px 0;
}
.bi-assistant-workbench__zone-picker button.active {
  color: #ffffff;
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
}
.bi-assistant-workbench__picker-subrow .el-input {
  max-width: 280px;
}
.bi-assistant-workbench__picker-list {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  max-height: 64vh;
  overflow: auto;
}
.bi-assistant-workbench__picker-pagination {
  justify-content: flex-end;
  margin-top: 16px;
}
.bi-assistant-workbench__picker-item {
  display: flex;
  flex-direction: column;
  min-width: 0;
  width: 100%;
  min-height: 0;
  height: auto;
  padding: 0;
  margin: 0;
  color: var(--el-text-color-primary);
  text-align: left;
  cursor: pointer;
  border: 0;
  border-radius: var(--admin-page-radius);
}
.bi-assistant-workbench__picker-item:hover {
  background: transparent;
}
.bi-assistant-workbench__picker-item.is-added,
.bi-assistant-workbench__picker-item.is-added:hover {
  color: var(--el-text-color-secondary);
  cursor: not-allowed;
  opacity: 0.62;
}
.bi-assistant-workbench__picker-item :deep(.bi-data-card) {
  width: 100%;
  height: 100%;
}
.bi-assistant-workbench__picker-item :deep(.bi-data-card__native-actions) {
  display: none;
}
.bi-assistant-workbench__picker-item :deep(.bi-data-card__header) {
  height: 44px;
  padding: 4px 10px;
}
.bi-assistant-workbench__picker-item :deep(.bi-data-card__chart) {
  height: 112px;
  padding: 8px 10px;
}
@media (max-width: 1100px) {
  .bi-assistant-workbench__picker-list {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
@media (max-width: 720px) {
  .bi-assistant-workbench__picker-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 480px) {
  .bi-assistant-workbench__picker-list {
    grid-template-columns: 1fr;
  }
}

/* ---------- 紧凑布局：卡片区默认收起，展开时浮层覆盖聊天区（对齐原型） ---------- */
@media (width <= 1280px) {
  .bi-assistant-workbench__session-switch {
    max-width: min(260px, 34vw);
  }
  /* 两侧始终按收起态 44px 占位，聊天区占满剩余宽度，不被展开的卡片挤压 */
  .bi-assistant-workbench__body {
    --zone-left-w: 44px;
    --zone-right-w: 44px;

    grid-template-columns: var(--zone-left-w) minmax(0, 1fr) var(--zone-right-w);
  }
  /* 展开态脱离文档流，浮在聊天区之上 */
  .bi-assistant-workbench__zone:not(.is-collapsed) {
    position: absolute;
    z-index: 10;
    top: 0;
    bottom: 0;
    width: min(360px, 86vw);
    box-shadow: 0 10px 36px rgb(15 23 42 / 18%);
  }
  .bi-assistant-workbench__zone.is-left:not(.is-collapsed) {
    left: 0;
    border-right: 1px solid var(--admin-page-divider-strong);
  }
  .bi-assistant-workbench__zone.is-right:not(.is-collapsed) {
    right: 0;
    border-left: 1px solid var(--admin-page-divider-strong);
  }
  .bi-assistant-workbench__zone-scrim {
    position: absolute;
    z-index: 9;
    inset: 0;
    background: rgb(15 23 42 / 16%);
  }
}

@media (width <= 620px) {
  .bi-assistant-workbench__topbar {
    flex-wrap: wrap;
  }
  .bi-assistant-workbench__session-switch {
    max-width: min(220px, 52vw);
  }
}
</style>
