<template>
  <div class="assistant-navigation" :class="{ 'is-collapsed': collapsed, 'is-mobile': mobile }">
    <SessionPanel
      v-if="!mobile && !collapsed"
      :active="active"
      :keyword="keyword"
      :sessions="sessions"
      @update:keyword="emit('update:keyword', $event)"
      @change="selectSession"
      @action="emit('action', $event)"
      @create="emit('create')"
      @toggle-collapse="emit('toggle')"
    >
      <template #session-meta="{ item }"
        ><span v-if="item.automation_id">{{ t("system.ai.workspace.automations") }} · </span>{{ item.summary }}</template
      >
      <template #navigation>
        <nav class="assistant-nav" :aria-label="t('system.ai.workspace.navigation')">
          <button
            v-for="item in functions"
            :key="item.key"
            :class="{ 'is-active': view === item.key }"
            type="button"
            @click="emit('view', item.key)"
          >
            <el-icon><component :is="item.icon" /></el-icon><span>{{ item.label }}</span>
            <el-badge v-if="item.key === 'automations' && pending" :value="pending" />
          </button>
        </nav>
      </template>
    </SessionPanel>
    <template v-else-if="!mobile">
      <el-tooltip :content="t('system.ai.chat.action.expand_sessions')" placement="right">
        <button
          class="assistant-icon"
          type="button"
          :aria-label="t('system.ai.chat.action.expand_sessions')"
          @click="emit('toggle')"
        >
          <el-icon><Expand /></el-icon>
        </button>
      </el-tooltip>
      <nav class="assistant-nav" :aria-label="t('system.ai.workspace.navigation')">
        <el-tooltip
          v-for="item in functions"
          :key="item.key"
          :content="item.key === 'automations' ? `${item.label} (${pending})` : item.label"
          placement="right"
        >
          <button
            type="button"
            :class="{ 'is-active': view === item.key }"
            :aria-label="item.key === 'automations' ? `${item.label} (${pending})` : item.label"
            @click="emit('view', item.key)"
          >
            <el-icon><component :is="item.icon" /></el-icon><i v-if="item.key === 'automations' && pending" class="pending-dot" />
          </button>
        </el-tooltip>
      </nav>
      <div class="assistant-nav-divider" />
      <el-tooltip :content="t('system.ai.chat.action.new_conversation')" placement="right"
        ><button
          class="assistant-icon"
          type="button"
          :aria-label="t('system.ai.chat.action.new_conversation')"
          @click="emit('create')"
        >
          <el-icon><Plus /></el-icon></button
      ></el-tooltip>
      <el-tooltip :content="t('system.ai.chat.title.recent_conversations')" placement="right"
        ><button
          class="assistant-icon"
          type="button"
          :aria-label="t('system.ai.chat.title.recent_conversations')"
          @click="historyOpen = true"
        >
          <el-icon><RefreshLeft /></el-icon></button
      ></el-tooltip>
    </template>
    <template v-else>
      <el-dropdown trigger="click" @command="changeView">
        <button
          class="assistant-mobile-trigger"
          type="button"
          :aria-label="`${t('system.ai.workspace.functions')} · ${currentLabel} (${pending})`"
        >
          {{ t("system.ai.workspace.functions") }} · {{ currentLabel }} <el-badge v-if="pending" :value="pending" /><el-icon
            ><ArrowDown
          /></el-icon>
        </button>
        <template #dropdown
          ><el-dropdown-menu>
            <el-dropdown-item v-for="item in functions" :key="item.key" :command="item.key" :icon="item.icon"
              >{{ item.label }}<el-badge v-if="item.key === 'automations' && pending" :value="pending"
            /></el-dropdown-item> </el-dropdown-menu
        ></template>
      </el-dropdown>
      <button
        class="assistant-icon"
        type="button"
        :aria-label="t('system.ai.chat.title.recent_conversations')"
        @click="historyOpen = true"
      >
        <el-icon><RefreshLeft /></el-icon>
      </button>
      <button
        class="assistant-icon"
        type="button"
        :aria-label="t('system.ai.chat.action.new_conversation')"
        @click="emit('create')"
      >
        <el-icon><Plus /></el-icon>
      </button>
    </template>
    <ProDialog
      v-model="historyOpen"
      :title="t('system.ai.chat.title.recent_conversations')"
      width="min(420px, 95vw)"
      :show-footer="false"
    >
      <SessionPanel
        class="assistant-history-panel"
        :active="active"
        :keyword="keyword"
        :sessions="sessions"
        :editable="!mobile"
        hide-collapse
        @update:keyword="emit('update:keyword', $event)"
        @change="selectSession"
        @action="emit('action', $event)"
        @create="
          historyOpen = false;
          emit('create');
        "
      />
    </ProDialog>
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from "vue";
import { Files } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import SessionPanel from "@liujitcn/kratos-admin-system/components/AiSessionPanel/index.vue";
import type { AiSession } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import { isAssistantView, type AssistantView } from "./state";
import type { SessionAction } from "../types";
const props = defineProps<{
  active: string;
  keyword: string;
  sessions: AiSession[];
  collapsed: boolean;
  mobile: boolean;
  view: AssistantView;
  pending: number;
}>();
const emit = defineEmits<{
  "update:keyword": [value: string];
  view: [value: AssistantView];
  change: [value: AiSession];
  action: [value: { action: SessionAction; item: AiSession }];
  create: [];
  toggle: [];
}>();
const historyOpen = ref(false);
const functions = computed(() => [
  { key: "chat" as const, label: t("system.ai.workspace.chat"), icon: ChatDotRound },
  { key: "files" as const, label: t("system.ai.workspace.files"), icon: Files },
  { key: "automations" as const, label: t("system.ai.workspace.automations"), icon: Timer },
  { key: "plugins" as const, label: t("system.ai.workspace.plugins"), icon: Grid }
]);
const currentLabel = computed(() => functions.value.find(item => item.key === props.view)?.label);
/** 下拉命令只接受既有助手功能。 */
function changeView(value: string | number | object) {
  if (typeof value === "string" && isAssistantView(value)) emit("view", value);
}
/** 历史选择返回当前对话并关闭弹层。 */
function selectSession(item: AiSession) {
  historyOpen.value = false;
  emit("view", "chat");
  emit("change", item);
}
</script>
<style scoped lang="scss">
.assistant-history-panel {
  display: flex !important;
  min-height: 300px;
  max-height: 65vh;
}
.assistant-navigation {
  min-width: 0;
  min-height: 0;
  border-right: 1px solid var(--admin-page-divider-strong);
}
.assistant-navigation :deep(.agent-session-panel) {
  height: 100%;
  box-sizing: border-box;
  border-right: 0;
}
.assistant-nav {
  display: grid;
  gap: 4px;
  padding: 12px 10px 0;
}
.assistant-nav button,
.assistant-icon,
.assistant-mobile-trigger {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  padding: 8px 12px;
  border: 0;
  border-radius: var(--admin-page-radius);
  color: var(--admin-page-text-regular);
  background: transparent;
  cursor: pointer;
}
.assistant-nav button span {
  flex: 1;
  text-align: left;
}
.assistant-nav button:hover,
.assistant-icon:hover {
  background: var(--el-fill-color-light);
}
.assistant-nav button.is-active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.assistant-nav .el-icon,
.assistant-icon .el-icon {
  font-size: 18px;
  flex-shrink: 0;
}
.is-collapsed:not(.is-mobile) {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding-top: 14px;
}
.is-collapsed .assistant-nav {
  width: 100%;
  box-sizing: border-box;
  padding: 10px 4px 0;
}
.is-collapsed .assistant-nav button {
  position: relative;
  justify-content: center;
  padding: 8px 0;
  width: 100%;
}
.is-collapsed .assistant-icon {
  width: 36px;
  padding: 8px;
  justify-content: center;
}
.pending-dot {
  position: absolute;
  right: 3px;
  top: 6px;
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--el-color-warning);
}
.assistant-nav-divider {
  width: 28px;
  height: 1px;
  background: var(--el-border-color);
  margin: 12px 0 6px;
}
.is-mobile {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  border-right: 0;
  border-bottom: 1px solid var(--el-border-color);
}
.is-mobile .el-dropdown {
  flex: 1;
  min-width: 0;
}
.is-mobile .assistant-icon,
.assistant-mobile-trigger {
  min-width: 44px;
  min-height: 44px;
}
.assistant-mobile-trigger {
  background: var(--el-fill-color-light);
  max-width: 100%;
}
</style>
