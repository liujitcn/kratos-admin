<template>
  <el-container class="bi-assistant-workbench">
    <header class="bi-assistant-workbench__topbar" @click="handleTopbarClick">
      <div class="bi-assistant-workbench__session-wrap">
        <button
          class="bi-assistant-workbench__session-switch"
          :class="{ 'is-open': sessionMenuOpen }"
          type="button"
          @click.stop="toggleSessionMenu()"
        >
          <span>{{ activeSessionTitle }}</span>
          <el-icon><ArrowDown /></el-icon>
        </button>
        <div v-if="sessionMenuOpen" class="bi-assistant-workbench__session-menu" @click.stop>
          <div class="bi-assistant-workbench__session-search">
            <el-icon><Search /></el-icon>
            <input v-model="sessionKeyword" type="text" placeholder="搜索会话" @keydown.esc="closeSessionMenu" />
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
                  <span class="bi-assistant-workbench__session-meta">{{ session.summary || "AI 对话" }}</span>
                </span>
                <el-dropdown
                  trigger="click"
                  placement="bottom-end"
                  @click.stop
                  @command="command => handleSessionMenuCommand(command, session)"
                >
                  <button class="bi-assistant-workbench__session-more" type="button" title="会话操作" @click.stop>
                    <el-icon><MoreFilled /></el-icon>
                  </button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="rename" :icon="EditPen">重命名</el-dropdown-item>
                      <el-dropdown-item command="pin" :icon="Star">
                        {{ pinnedSessionIDs.includes(session.id) ? "取消置顶" : "置顶" }}
                      </el-dropdown-item>
                      <el-dropdown-item command="delete" :icon="Delete" divided>删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
            </template>
            <div v-if="!sessionGroups.some(group => group.items.length)" class="bi-assistant-workbench__session-empty">暂无匹配会话</div>
          </div>
          <div class="bi-assistant-workbench__session-foot">
            <button class="bi-assistant-workbench__plain-button" type="button" @click="createSessionFromMenu">
              <el-icon><Plus /></el-icon>
              新建会话
            </button>
          </div>
        </div>
      </div>

      <button class="bi-assistant-workbench__icon-button" type="button" title="新建会话" @click.stop="createSession">
        <el-icon><Plus /></el-icon>
      </button>

      <div
        class="bi-assistant-workbench__topbar-search"
        :class="{ open: topSearchOpen }"
        title="搜索会话"
        @click.stop="openTopSearch"
      >
        <el-icon><Search /></el-icon>
        <input
          v-if="topSearchOpen"
          ref="topSearchInput"
          v-model="sessionKeyword"
          type="text"
          placeholder="搜索会话..."
          @keydown.esc="closeTopSearch"
        />
        <button v-if="topSearchOpen" class="bi-assistant-workbench__search-clear" type="button" title="关闭" @click.stop="closeTopSearch">
          <el-icon><Close /></el-icon>
        </button>
      </div>

      <span class="bi-assistant-workbench__spacer" />

      <button class="bi-assistant-workbench__icon-button" type="button" title="工具箱" @click.stop="toggleToolbox">
        <el-icon><Grid /></el-icon>
        <span class="bi-assistant-workbench__dot" />
      </button>
      <span class="bi-assistant-workbench__topbar-sep" />
      <div class="bi-assistant-workbench__avatar" title="寇豆码">寇</div>

      <div v-if="toolboxOpen" class="bi-assistant-workbench__toolbox" @click.stop>
        <div class="bi-assistant-workbench__toolbox-title">工作台工具</div>
        <div class="bi-assistant-workbench__toolbox-grid">
          <button type="button" @click="openToolMenu('files')">
            <span class="bi-assistant-workbench__toolbox-icon"><el-icon><Document /></el-icon></span>
            <span class="bi-assistant-workbench__toolbox-badge">2</span>
            文件
          </button>
          <button type="button" @click="openToolMenu('auto')">
            <span class="bi-assistant-workbench__toolbox-icon is-warning"><el-icon><Timer /></el-icon></span>
            <span class="bi-assistant-workbench__toolbox-badge is-warning-dot" />
            自动化
          </button>
          <button type="button" @click="openToolMenu('plugins')">
            <span class="bi-assistant-workbench__toolbox-icon is-plugin"><el-icon><Grid /></el-icon></span>
            <span class="bi-assistant-workbench__toolbox-badge">5</span>
            插件
          </button>
        </div>
      </div>

      <div v-if="toolMenuKind" class="bi-assistant-workbench__menu-pop" @click.stop>
        <button v-for="item in toolMenuItems" :key="item.label" type="button" @click="openAssistantTool(item.action)">
          <span>{{ item.label }}</span><small>{{ item.actionLabel }}</small>
        </button>
      </div>
    </header>

    <el-main class="bi-assistant-workbench__body">
      <el-aside class="bi-assistant-workbench__zone" :class="{ 'is-collapsed': collapsed.left }">
        <el-header class="bi-assistant-workbench__zone-head">
          <el-button text circle :icon="collapsed.left ? ArrowRight : ArrowLeft" @click="toggleZone('left')" />
          <span v-if="!collapsed.left" class="bi-assistant-workbench__zone-title">左侧工作台</span>
          <span v-if="!collapsed.left" class="bi-assistant-workbench__spacer" />
          <el-button v-if="!collapsed.left" text :icon="Plus" @click="openCardPicker('left')">{{ t("bi.card.add_short") }}</el-button>
        </el-header>
        <div v-if="!collapsed.left" class="bi-assistant-workbench__zone-body" @dragover.prevent @drop.prevent="dropCard('left')">
          <BiAssistantCardZone zone="left" :cards="leftCards" :refresh-keys="refreshKeys" @drag-start="startCardDrag" @drag-end="dragged = undefined" @drop="dropCard" @refresh="refreshCard" @quote="quoteCard" @remove="removeCard" @export="exportCard" @preview="previewCard" />
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
          <BiAssistantCardZone zone="right" :cards="rightCards" :refresh-keys="refreshKeys" @drag-start="startCardDrag" @drag-end="dragged = undefined" @drop="dropCard" @refresh="refreshCard" @quote="quoteCard" @remove="removeCard" @export="exportCard" @preview="previewCard" />
        </div>
        <span v-else class="bi-assistant-workbench__vertical-label">右</span>
      </el-aside>
    </el-main>

    <el-dialog v-model="cardPicker.open" class="bi-assistant-workbench__picker-dialog" :title="t('bi.card.library')" width="min(880px, calc(100vw - 64px))" destroy-on-close>
      <div class="bi-assistant-workbench__picker-subrow">
        <span>{{ t("bi.card.library_hint") }}</span>
        <span class="bi-assistant-workbench__spacer" />
        <span class="bi-assistant-workbench__picker-label">添加到</span>
        <div class="bi-assistant-workbench__zone-picker">
          <button type="button" :class="{ active: cardPicker.zone === 'left' }" @click="cardPicker.zone = 'left'">左侧</button>
          <button type="button" :class="{ active: cardPicker.zone === 'right' }" @click="cardPicker.zone = 'right'">右侧</button>
        </div>
      </div>
      <div class="bi-assistant-workbench__picker-head">
        <span>选择卡片，按当前会话口径固定到工作台</span>
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
    <el-dialog
      v-model="cardPreview.open"
      class="bi-assistant-workbench__preview-dialog"
      :title="cardPreview.card?.name || '卡片详情'"
      width="min(780px, calc(100vw - 64px))"
      destroy-on-close
    >
      <BiDataCard v-if="cardPreview.card" :card="cardPreview.card" :show-category="false" />
    </el-dialog>
  </el-container>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  Close,
  CopyDocument,
  DataAnalysis,
  Delete,
  Document,
  Download,
  EditPen,
  Grid,
  MoreFilled,
  Plus,
  Refresh,
  Search,
  Star,
  Timer
} from "@element-plus/icons-vue";
import { navigateTo } from "@liujitcn/kratos-admin-core/navigation";
import { t } from "@liujitcn/kratos-admin-core";
import { useRouter } from "vue-router";
import { defAiKnowledgeService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_knowledge";
import { defAiMessageService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_message";
import { defAiModelService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_provider";
import { defAiSessionService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_session";
import { defAiToolService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_tool";
import { AiMessageStatus } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import type { AiSession } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
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
const router = useRouter();
const sessionKeyword = ref("");
const sessionMenuOpen = ref(false);
const topSearchOpen = ref(false);
const toolboxOpen = ref(false);
const toolMenuKind = ref<"files" | "auto" | "plugins">();
const topSearchInput = ref<HTMLInputElement>();
const pinnedSessionIDs = ref<string[]>(readPinnedSessionIDs());
const leftCards = ref<BiCard[]>([]);
const rightCards = ref<BiCard[]>([]);
const allCards = ref<BiCard[]>([]);
const refreshKeys = reactive<Record<number, number>>({});
const dragged = ref<CardDrag>();
const collapsed = reactive({ left: false, right: false });
const cardPicker = reactive({ open: false, loading: false, allLoaded: false, keyword: "", zone: "right" as Zone });
const cardPreview = reactive<{ open: boolean; card?: BiCard }>({ open: false });
const workbench = ref<BiWorkbench>({ dashboard_ids: [], favorite_ids: [], dashboards: [] });
// shortcut: 左右分区暂存在浏览器本地，跨设备同步时再扩展 BiWorkbench 协议。
const zoneStorageKey = "kratos-admin:bi-assistant-workbench:zones";

const visibleSessions = computed(() => {
  const keyword = sessionKeyword.value.trim().toLowerCase();
  const filtered = keyword ? sessions.value.filter(item => (item.title + " " + item.summary).toLowerCase().includes(keyword)) : sessions.value;
  return [...filtered].sort((left, right) => {
    const pinDiff = Number(pinnedSessionIDs.value.includes(right.id)) - Number(pinnedSessionIDs.value.includes(left.id));
    return pinDiff || sessionTime(right).getTime() - sessionTime(left).getTime();
  });
});
const activeSessionTitle = computed(() => sessions.value.find(item => item.id === activeSessionID.value)?.title || "新会话");
const sessionGroups = computed(() => {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const yesterday = today - 86_400_000;
  const groups = [
    { label: "今天", items: [] as AiSession[] },
    { label: "昨天", items: [] as AiSession[] },
    { label: "更早", items: [] as AiSession[] }
  ];
  visibleSessions.value.forEach(item => {
    const time = sessionTime(item).getTime();
    const target = time >= today ? groups[0] : time >= yesterday ? groups[1] : groups[2];
    target.items.push(item);
  });
  return groups;
});
const toolMenuItems = computed(() => {
  const menus = {
    files: [
      { label: "营收月报_2026-11.xlsx", actionLabel: "打开", action: "files" as const },
      { label: "设备巡检记录_11-15.pdf", actionLabel: "打开", action: "files" as const },
      { label: "活跃用户明细_11-16.csv", actionLabel: "打开", action: "files" as const },
      { label: "+ 上传文件", actionLabel: "上传", action: "files" as const }
    ],
    auto: [
      { label: "周报自动生成 · 待确认", actionLabel: "查看", action: "auto" as const },
      { label: "异常数据巡检 · 运行中", actionLabel: "查看", action: "auto" as const },
      { label: "每日报表推送 · 已启用", actionLabel: "配置", action: "auto" as const },
      { label: "+ 新建自动化", actionLabel: "新建", action: "auto" as const }
    ],
    plugins: [
      { label: "BI 问数 · 已启用", actionLabel: "配置", action: "plugins" as const },
      { label: "工单同步 · 已启用", actionLabel: "配置", action: "plugins" as const },
      { label: "Doris 查询 · 已启用", actionLabel: "配置", action: "plugins" as const },
      { label: "数据大屏推送 · 已停用", actionLabel: "启用", action: "plugins" as const },
      { label: "导出中心 · 已启用", actionLabel: "配置", action: "plugins" as const }
    ]
  };
  return toolMenuKind.value ? menus[toolMenuKind.value] : [];
});
const pickerCards = computed(() => {
  const used = new Set([...leftCards.value, ...rightCards.value].map(item => item.id));
  const keyword = cardPicker.keyword.trim().toLowerCase();
  return allCards.value.filter(item => !used.has(item.id) && (!keyword || (item.name + " " + item.source_code).toLowerCase().includes(keyword)));
});

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
  toolMenuKind.value = undefined;
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
    toolMenuKind.value = undefined;
  }
}

/** 关闭会话菜单。 */
function closeSessionMenu() {
  sessionMenuOpen.value = false;
}

/** 打开顶部会话搜索并聚焦输入框。 */
async function openTopSearch() {
  topSearchOpen.value = true;
  sessionMenuOpen.value = true;
  toolboxOpen.value = false;
  toolMenuKind.value = undefined;
  await nextTick();
  topSearchInput.value?.focus();
}

/** 关闭顶部会话搜索并清空过滤条件。 */
function closeTopSearch() {
  topSearchOpen.value = false;
  sessionKeyword.value = "";
  sessionMenuOpen.value = false;
}

/** 切换工具箱面板。 */
function toggleToolbox() {
  toolboxOpen.value = !toolboxOpen.value;
  sessionMenuOpen.value = false;
  toolMenuKind.value = undefined;
}

/** 打开工具箱二级菜单。 */
function openToolMenu(kind: "files" | "auto" | "plugins") {
  toolboxOpen.value = false;
  toolMenuKind.value = kind;
}

/** 将工具箱操作跳转到现有 AI 助手功能面板。 */
async function openAssistantTool(kind: "files" | "auto" | "plugins") {
  toolMenuKind.value = undefined;
  await navigateTo(router, "/ai/chat", { view: kind === "files" ? "files" : kind === "auto" ? "automations" : "plugins" });
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
      const result = await ElMessageBox.prompt("请输入新的会话名称", "重命名会话", {
        inputValue: session.title,
        confirmButtonText: "确定",
        cancelButtonText: "取消"
      });
      const response = await defAiSessionService.UpdateAiSession({ id: session.id, title: result.value.trim() });
      if (response.session) upsertSession(normalizeSession(response.session));
      return;
    }
    if (String(command) !== "delete") return;
    await ElMessageBox.confirm(`是否删除会话「${session.title}」？`, "删除会话", {
      confirmButtonText: "删除",
      cancelButtonText: "取消",
      type: "warning"
    });
    await defAiSessionService.DeleteAiSession({ id: session.id });
    sessions.value = sessions.value.filter(item => item.id !== session.id);
    if (activeSessionID.value === session.id) {
      activeSessionID.value = sessions.value[0]?.id || "";
      if (activeSessionID.value) await selectSession(activeSessionID.value);
    }
  } catch (cause) {
    if (cause === "cancel" || cause === "close") return;
    ElMessage.error(cause instanceof Error ? cause.message : "会话操作失败");
  }
}

/** 从会话弹层创建新会话。 */
async function createSessionFromMenu() {
  closeSessionMenu();
  await createSession();
}

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

/** 打开卡片放大查看弹窗。 */
function previewCard(card: BiCard) {
  cardPreview.card = card;
  cardPreview.open = true;
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
  document.addEventListener("click", closeTopbarPopovers);
  try {
    await Promise.all([loadCards(), loadChat()]);
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
  display: flex;
  flex: none;
  align-items: center;
  gap: 6px;
  box-sizing: border-box;
  min-height: 42px;
  padding: 4px 10px;
  border-bottom: 1px solid var(--admin-page-divider-strong);
}
.bi-assistant-workbench__topbar-sep {
  width: 1px;
  height: 18px;
  flex: none;
  background: var(--el-border-color-lighter);
}
.bi-assistant-workbench__session-wrap {
  position: relative;
  display: inline-flex;
  flex: none;
}
.bi-assistant-workbench__session-switch {
  display: inline-flex;
  max-width: 260px;
  height: 32px;
  align-items: center;
  gap: 6px;
  padding: 0 8px;
  color: var(--admin-page-text-primary);
  font-size: 13px;
  font-weight: 500;
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
  width: 32px;
  height: 32px;
  flex: none;
  align-items: center;
  justify-content: center;
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
.bi-assistant-workbench__topbar-search {
  display: flex;
  width: 32px;
  height: 32px;
  flex: none;
  align-items: center;
  gap: 6px;
  padding: 0;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  overflow: hidden;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 4px;
  transition: width 0.2s ease, padding 0.2s ease, border-color 0.2s ease;
}
.bi-assistant-workbench__topbar-search > .el-icon {
  flex: none;
  margin-left: 8px;
}
.bi-assistant-workbench__topbar-search:hover {
  color: var(--admin-page-text-primary);
  background: var(--el-fill-color-light);
}
.bi-assistant-workbench__topbar-search.open {
  width: min(220px, 100%);
  padding-right: 6px;
  cursor: text;
  background: var(--el-bg-color-overlay);
  border-color: var(--el-color-primary-light-7);
  box-shadow: 0 1px 4px rgb(15 23 42 / 6%);
}
.bi-assistant-workbench__topbar-search input {
  min-width: 0;
  flex: 1;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 13px;
  background: transparent;
  border: 0;
  outline: 0;
}
.bi-assistant-workbench__search-clear {
  display: inline-flex;
  width: 16px;
  height: 16px;
  flex: none;
  align-items: center;
  justify-content: center;
  color: var(--admin-page-text-placeholder);
  background: transparent;
  border: 0;
  border-radius: 3px;
}
.bi-assistant-workbench__search-clear:hover {
  background: var(--el-fill-color);
}
.bi-assistant-workbench__spacer {
  flex: 1;
  min-width: 0;
}
.bi-assistant-workbench__avatar {
  display: flex;
  width: 30px;
  height: 30px;
  flex: none;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  user-select: none;
  cursor: default;
  background: linear-gradient(135deg, var(--el-color-primary), var(--el-color-info));
  border-radius: 50%;
}
.bi-assistant-workbench__session-menu,
.bi-assistant-workbench__toolbox,
.bi-assistant-workbench__menu-pop {
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
  width: min(320px, calc(100vw - 32px));
  max-height: 440px;
  flex-direction: column;
  overflow: hidden;
  border-radius: var(--admin-page-radius);
}
.bi-assistant-workbench__session-search {
  display: flex;
  height: 30px;
  align-items: center;
  gap: 6px;
  margin: 10px 10px 6px;
  padding: 0 10px;
  color: var(--admin-page-text-placeholder);
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}
.bi-assistant-workbench__session-search input {
  min-width: 0;
  flex: 1;
  font-size: 13px;
  background: transparent;
  border: 0;
  outline: 0;
}
.bi-assistant-workbench__session-list {
  flex: 1;
  padding: 0 6px 6px;
  overflow-y: auto;
}
.bi-assistant-workbench__group-label {
  padding: 10px 6px 3px;
  color: var(--admin-page-text-placeholder);
  font-size: 11px;
}
.bi-assistant-workbench__session-item {
  display: flex;
  width: 100%;
  min-width: 0;
  align-items: center;
  gap: 6px;
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
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}
.bi-assistant-workbench__session-title,
.bi-assistant-workbench__session-meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bi-assistant-workbench__session-title {
  color: var(--admin-page-text-primary);
  font-size: 13px;
  font-weight: 500;
}
.bi-assistant-workbench__session-item.is-active .bi-assistant-workbench__session-title {
  color: var(--el-color-primary);
}
.bi-assistant-workbench__session-meta {
  color: var(--admin-page-text-placeholder);
  font-size: 11px;
}
.bi-assistant-workbench__session-more {
  display: inline-flex;
  width: 20px;
  height: 20px;
  flex: none;
  align-items: center;
  justify-content: center;
  color: var(--admin-page-text-placeholder);
  opacity: 0;
  background: transparent;
  border: 0;
  border-radius: 4px;
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
  color: var(--admin-page-text-placeholder);
  font-size: 12px;
  text-align: center;
}
.bi-assistant-workbench__session-foot {
  padding: 8px;
  border-top: 1px solid var(--admin-page-divider);
}
.bi-assistant-workbench__plain-button {
  display: inline-flex;
  width: 100%;
  height: 30px;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: var(--admin-page-text-regular);
  background: #fff;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
}
.bi-assistant-workbench__plain-button:hover {
  color: var(--el-color-primary);
  border-color: var(--el-color-primary-light-7);
  background: var(--el-color-primary-light-9);
}
.bi-assistant-workbench__toolbox {
  top: 100%;
  right: 10px;
  box-sizing: border-box;
  width: min(288px, calc(100% - 20px));
  padding: 12px;
  border-radius: 8px;
}
.bi-assistant-workbench__toolbox-title {
  margin: 0 2px 8px;
  color: var(--admin-page-text-placeholder);
  font-size: 12px;
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
  align-items: center;
  gap: 6px;
  padding: 12px 4px 10px;
  color: var(--admin-page-text-regular);
  font-size: 12px;
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
  width: 34px;
  height: 34px;
  align-items: center;
  justify-content: center;
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
  color: #fff;
  font-size: 10px;
  line-height: 16px;
  text-align: center;
  background: var(--el-color-danger);
  border-radius: 8px;
}
.bi-assistant-workbench__toolbox-badge.is-warning-dot {
  width: 8px;
  min-width: 8px;
  padding: 0;
  background: var(--el-color-warning);
}
.bi-assistant-workbench__menu-pop {
  top: 100%;
  right: 10px;
  min-width: 220px;
  padding: 4px;
  border-radius: 6px;
}
.bi-assistant-workbench__menu-pop button {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  color: var(--admin-page-text-regular);
  font-size: 13px;
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: 4px;
}
.bi-assistant-workbench__menu-pop button:hover {
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
}
.bi-assistant-workbench__menu-pop small {
  flex: none;
  color: var(--admin-page-text-placeholder);
  font-size: 11px;
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
.bi-assistant-workbench__picker-dialog :deep(.el-dialog) {
  overflow: hidden;
  border-radius: var(--admin-page-radius);
}
.bi-assistant-workbench__picker-dialog :deep(.el-dialog__header) {
  padding: 16px 20px 4px;
  margin-right: 0;
}
.bi-assistant-workbench__picker-dialog :deep(.el-dialog__title) {
  color: var(--admin-page-text-primary);
  font-size: 16px;
  font-weight: 600;
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
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  color: var(--admin-page-text-placeholder);
  font-size: 12px;
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
  color: var(--el-text-color-regular);
  font-size: 12px;
  white-space: nowrap;
  background: #fff;
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
  color: #fff;
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
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
  .bi-assistant-workbench__session-switch {
    max-width: min(260px, 24vw);
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
  .bi-assistant-workbench__session-switch {
    max-width: min(220px, 34vw);
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
