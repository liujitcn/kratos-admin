<template>
  <aside class="agent-session-panel">
    <el-input
      :model-value="keyword"
      placeholder="搜索对话"
      clearable
      :prefix-icon="Search"
      @update:model-value="handleKeywordUpdate"
    />
    <div class="agent-divider"></div>
    <div class="agent-panel-header">
      <div class="agent-panel-title">最近对话</div>
      <div class="agent-panel-actions">
        <el-tooltip content="收起会话栏" placement="top">
          <button class="agent-panel-icon" type="button" aria-label="收起会话栏" @click="$emit('toggleCollapse')">
            <el-icon><DArrowLeft /></el-icon>
          </button>
        </el-tooltip>
        <button class="agent-panel-create" type="button" aria-label="新建会话" @click="handleCreateSession">
          <el-icon><Plus /></el-icon>
          <span>新建</span>
        </button>
      </div>
    </div>
    <ul class="agent-session-list">
      <li
        v-for="item in sessions"
        :key="item.id"
        class="agent-session-row"
        :class="{ 'is-active': item.id === active }"
        @click="handleSessionChange(item)"
      >
        <button class="agent-session-item" type="button" :title="item.title" :aria-current="item.id === active ? 'page' : undefined">
          <div class="agent-session-main">
            <div class="agent-session-name">{{ item.title }}</div>
            <div class="agent-session-meta">{{ item.summary }}</div>
          </div>
        </button>
        <el-dropdown
          class="agent-session-menu"
          trigger="click"
          placement="bottom-end"
          @click.stop
          @command="command => handleMenuCommand(command, item)"
        >
          <button class="agent-session-more" type="button" aria-label="会话操作">
            <el-icon><MoreFilled /></el-icon>
          </button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="rename" :icon="EditPen">重命名</el-dropdown-item>
              <el-dropdown-item command="delete" :icon="Delete" divided>删除</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </li>
    </ul>
  </aside>
</template>

<script setup lang="ts" name="SessionPanel">
import { DArrowLeft, Delete, EditPen, MoreFilled, Plus, Search } from "@element-plus/icons-vue";
import type { AiSession } from "@/rpc/base/v1/ai_session";

/** 会话列表支持的菜单操作。 */
type SessionAction = "rename" | "delete";

const props = defineProps<{
  /** 当前活动会话编号。 */
  active: string;
  /** 会话搜索关键词。 */
  keyword: string;
  /** 过滤后的会话列表。 */
  sessions: AiSession[];
}>();

const emit = defineEmits<{
  /** 更新当前活动会话。 */
  "update:active": [value: string];
  /** 更新搜索关键词。 */
  "update:keyword": [value: string];
  /** 通知父组件会话已切换。 */
  change: [item: AiSession];
  /** 会话操作菜单。 */
  action: [payload: { action: SessionAction; item: AiSession }];
  /** 创建新的会话。 */
  create: [];
  /** 收起会话栏。 */
  toggleCollapse: [];
}>();

/** 同步搜索关键词。 */
function handleKeywordUpdate(value: string) {
  emit("update:keyword", value);
}

/** 同步当前会话，并通知父组件加载对应消息。 */
function handleSessionChange(item: AiSession) {
  emit("update:active", item.id);
  emit("change", item);
}

/** 透传当前会话菜单操作，后续由父组件接入真实重命名和删除。 */
function handleMenuCommand(command: string | number | object, item: AiSession) {
  const action = String(command ?? "") as SessionAction;
  if (action !== "rename" && action !== "delete") return;
  emit("action", { action, item });
}

/** 通知父组件创建新的会话。 */
function handleCreateSession() {
  emit("create");
}
</script>

<style scoped lang="scss">
.agent-session-panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  padding: 16px 0;
  overflow: hidden;
  background: var(--admin-page-card-bg);
  border-right: 1px solid var(--admin-page-divider-strong);
  :deep(.el-input__wrapper) {
    padding: 7px 12px;
    background: var(--el-fill-color-light);
    border-radius: var(--admin-page-radius);
    box-shadow: none;
  }
  :deep(.el-input) {
    width: calc(100% - 28px);
    margin: 0 14px;
  }
}
.agent-panel-icon {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: var(--el-fill-color-light);
  border: 0;
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    background-color 0.2s ease;
  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}
.agent-panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 14px;
  margin: 16px 0 10px;
}
.agent-panel-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}
.agent-panel-actions {
  display: inline-flex;
  gap: 6px;
  align-items: center;
}
.agent-panel-create {
  display: inline-flex;
  gap: 4px;
  align-items: center;
  height: 26px;
  padding: 0 9px;
  font-size: 12px;
  font-weight: 600;
  color: var(--el-color-primary);
  cursor: pointer;
  background: var(--el-color-primary-light-9);
  border: 0;
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    background-color 0.2s ease;
  &:hover {
    color: var(--el-color-primary-dark-2);
    background: var(--el-color-primary-light-8);
  }
}
.agent-divider {
  height: 1px;
  margin: 16px 0 0;
  background: var(--el-border-color-lighter);
}
.agent-session-list {
  flex: 1;
  min-width: 0;
  padding: 0;
  margin: 0;
  overflow: auto;
  list-style: none;
}
.agent-session-row {
  position: relative;
  display: flex;
  align-items: stretch;
  min-width: 0;
  transition: background-color 0.2s ease;
  &::after {
    position: absolute;
    right: 14px;
    bottom: 0;
    left: 32px;
    height: 1px;
    content: "";
    background: var(--el-border-color-lighter);
  }
  &:hover {
    background: var(--el-fill-color-extra-light);
  }
  &.is-active {
    background: var(--el-color-primary-light-9);
    box-shadow: inset 3px 0 0 var(--el-color-primary);
  }
  &.is-active .agent-session-name {
    color: var(--el-color-primary);
  }
  &.is-active .agent-session-meta {
    color: var(--admin-page-text-primary);
  }
  &.is-active .agent-session-more,
  &:hover .agent-session-more {
    opacity: 1;
  }
}
.agent-session-item {
  display: flex;
  flex: 1;
  gap: 8px;
  align-items: center;
  min-width: 0;
  min-height: 66px;
  padding: 11px 14px 11px 32px;
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 0;
}
.agent-session-main {
  flex: 1;
  min-width: 0;
}
.agent-session-name {
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 14px;
  font-weight: 700;
  line-height: 22px;
  color: var(--admin-page-text-primary);
  white-space: nowrap;
}
.agent-session-meta {
  margin-top: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
  white-space: nowrap;
}
.agent-session-menu {
  position: absolute;
  top: 50%;
  right: 14px;
  z-index: 1;
  transform: translateY(-50%);
}
.agent-session-more {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 50%;
  opacity: 0;
  transition:
    color 0.2s ease,
    background-color 0.2s ease,
    opacity 0.2s ease;
  &:hover {
    color: var(--admin-page-text-primary);
    background: var(--el-fill-color-light);
  }
}

@media screen and (width <= 768px) {
  .agent-session-panel {
    display: none;
  }
}
</style>
