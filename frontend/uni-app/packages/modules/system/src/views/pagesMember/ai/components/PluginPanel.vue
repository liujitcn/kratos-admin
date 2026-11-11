<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from '@liujitcn/kratos-uni-app-core'
import { defAiPluginService } from '../../../../api/base/v1/ai_plugin'
import type {
  AiPlugin,
  AiPluginConnection,
  AiPluginSelection,
  AiPluginTool,
} from '../../../../rpc/base/v1/ai_plugin'
import { pluginSelectionIssue } from '../workspace'
import { Terminal } from '../../../../rpc/base/v1/ai_tool'

/** 插件面板沿用当前会话的工具能力配置。 */
const props = defineProps<{
  active: boolean
  terminal: Terminal
  picker?: boolean
  topPadding?: string
  selected: AiPluginSelection[]
  saving?: boolean
}>()
const emit = defineEmits<{ save: [selection: AiPluginSelection[]]; close: [] }>()
const { t } = useI18n()
const plugins = ref<AiPlugin[]>([])
const connections = ref<AiPluginConnection[]>([])
const tools = ref<Record<string, AiPluginTool[]>>({})
const draft = ref<AiPluginSelection[]>([])
const keyword = ref('')
const connectedOnly = ref(false)
const loading = ref(false)
const errorText = ref('')
const expanded = ref('')
const visiblePlugins = computed(() =>
  plugins.value.filter(
    (plugin) =>
      (!connectedOnly.value ||
        connections.value.some((connection) => connection.plugin_id === plugin.id)) &&
      `${plugin.name} ${plugin.description}`.toLowerCase().includes(keyword.value.toLowerCase()),
  ),
)
const missingSelections = computed(() =>
  draft.value.filter(
    (selection) =>
      !plugins.value.some(
        (plugin) =>
          plugin.id === selection.plugin_id &&
          (!selection.connection_id ||
            connections.value.some((connection) => connection.id === selection.connection_id)),
      ),
  ),
)

watch(
  () => [props.active, props.terminal] as const,
  ([active]) => {
    if (active) {
      draft.value = props.selected.map((selection) => ({ ...selection }))
      void load()
    }
  },
  { immediate: true },
)

/** 一次加载目录、可见账号与工具状态，失败保留错误供重试。 */
async function load() {
  if (loading.value) return
  const terminal = props.terminal
  loading.value = true
  errorText.value = ''
  try {
    const [catalog, accounts] = await Promise.all([
      defAiPluginService.ListAiPlugin({ keyword: '', terminal }),
      defAiPluginService.ListAiPluginConnection({ plugin_id: '' }),
    ])
    const snapshots = await Promise.all(
      catalog.plugins.map(async (plugin) => ({
        id: plugin.id,
        tools: (await defAiPluginService.ListAiPluginTool({ plugin_id: plugin.id, terminal })).tools,
      })),
    )
    if (terminal !== props.terminal) return
    plugins.value = catalog.plugins
    connections.value = accounts.connections
    tools.value = Object.fromEntries(snapshots.map((snapshot) => [snapshot.id, snapshot.tools]))
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : t('system.ai.workspace.load_failed')
  } finally {
    loading.value = false
    if (props.active && terminal !== props.terminal) void load()
  }
}

function pluginConnections(plugin: AiPlugin) {
  return connections.value.filter((connection) => connection.plugin_id === plugin.id)
}

function isSelected(pluginId: string, connectionId = '') {
  return draft.value.some(
    (selection) => selection.plugin_id === pluginId && selection.connection_id === connectionId,
  )
}

function issue(plugin: AiPlugin, connection?: AiPluginConnection) {
  const reason = pluginSelectionIssue(plugin, tools.value[plugin.id] || [], connection,
    props.terminal === Terminal.TERMINAL_APP ? 'app' : 'admin')
  return [
    'not_assembled',
    'disabled',
    'terminal_unavailable',
    'connection_unavailable',
    'no_tools',
  ].includes(reason)
    ? t(`system.ai.workspace.${reason}`)
    : reason
}

/** 只调整草稿中的确定账号；选中的失效条目仍允许取消。 */
function toggle(plugin: AiPlugin, connection?: AiPluginConnection) {
  if (props.saving) return
  const connectionId = connection?.id || ''
  if (isSelected(plugin.id, connectionId)) {
    draft.value = draft.value.filter(
      (selection) => selection.plugin_id !== plugin.id || selection.connection_id !== connectionId,
    )
  } else if (!issue(plugin, connection)) {
    draft.value.push({ plugin_id: plugin.id, connection_id: connectionId })
  }
}
</script>

<template>
  <view
    class="plugin-panel"
    :class="{ 'is-picker': picker }"
    :style="picker ? { paddingTop: topPadding } : undefined"
  >
    <view v-if="picker" class="panel-row">
      <text class="panel-title">{{ t('system.ai.workspace.plugins') }}</text>
      <button size="mini" @tap="emit('close')">{{ t('system.ai.workspace.close') }}</button>
    </view>
    <view class="panel-hint">{{ t('system.ai.workspace.desktop_configuration') }}</view>
    <view class="panel-row">
      <button size="mini" :class="{ 'is-active': !connectedOnly }" @tap="connectedOnly = false">
        {{ t('system.ai.workspace.catalog') }}
      </button>
      <button size="mini" :class="{ 'is-active': connectedOnly }" @tap="connectedOnly = true">
        {{ t('system.ai.workspace.accounts') }}
      </button>
    </view>
    <input
      v-model="keyword"
      class="panel-search"
      :placeholder="t('system.ai.workspace.search_plugins')"
    />
    <view v-if="keyword" class="panel-link" @tap="keyword = ''">{{
      t('system.ai.workspace.clear')
    }}</view>
    <view v-if="loading" class="panel-hint">{{ t('system.ai.workspace.loading') }}</view>
    <view v-if="errorText" class="panel-error">
      {{ errorText }}
      <button size="mini" @tap="load">{{ t('system.ai.workspace.retry') }}</button>
    </view>
    <scroll-view class="panel-content" scroll-y>
      <view
        v-for="selection in missingSelections"
        :key="`${selection.plugin_id}:${selection.connection_id}`"
        class="panel-card"
      >
        <text
          >{{ t('system.ai.workspace.selection_unavailable') }} · {{ selection.plugin_id }} /
          {{ selection.connection_id }}</text
        >
        <button
          v-if="picker"
          size="mini"
          :disabled="saving"
          @tap="draft = draft.filter((item) => item !== selection)"
        >
          {{ t('system.ai.workspace.remove') }}
        </button>
      </view>
      <view v-for="plugin in visiblePlugins" :key="plugin.id" class="panel-card">
        <view class="panel-title" @tap="expanded = expanded === plugin.id ? '' : plugin.id"
          >{{ plugin.name }} · {{ plugin.source }}</view
        >
        <view class="panel-hint">{{ plugin.description }}</view>
        <view v-if="plugin.unavailable_reason" class="panel-error">{{
          plugin.unavailable_reason
        }}</view>
        <template v-if="plugin.source !== 'MCP'">
          <view class="panel-hint">{{ issue(plugin) || t('system.ai.workspace.available') }}</view>
          <button
            v-if="picker"
            size="mini"
            :disabled="saving || (!isSelected(plugin.id) && Boolean(issue(plugin)))"
            @tap="toggle(plugin)"
          >
            {{
              t(isSelected(plugin.id) ? 'system.ai.workspace.remove' : 'system.ai.workspace.select')
            }}
          </button>
        </template>
        <view
          v-for="connection in pluginConnections(plugin)"
          :key="connection.id"
          class="account-row"
        >
          <view
            >{{ connection.label }} ·
            {{
              t(
                connection.sharing === 'tenant'
                  ? 'system.ai.workspace.shared'
                  : 'system.ai.workspace.personal',
              )
            }}</view
          >
          <view class="panel-hint">{{
            issue(plugin, connection) || t('system.ai.workspace.available')
          }}</view>
          <view v-if="connection.scopes" class="panel-hint">{{ connection.scopes }}</view>
          <button
            v-if="picker"
            size="mini"
            :disabled="
              saving ||
              (!isSelected(plugin.id, connection.id) && Boolean(issue(plugin, connection)))
            "
            @tap="toggle(plugin, connection)"
          >
            {{
              t(
                isSelected(plugin.id, connection.id)
                  ? 'system.ai.workspace.remove'
                  : 'system.ai.workspace.select',
              )
            }}
          </button>
        </view>
        <view
          v-if="plugin.source === 'MCP' && !pluginConnections(plugin).length"
          class="panel-hint"
          >{{ t('system.ai.workspace.connection_unavailable') }}</view
        >
        <button size="mini" @tap="expanded = expanded === plugin.id ? '' : plugin.id">
          {{ t('system.ai.workspace.tools') }}
        </button>
        <view v-if="expanded === plugin.id">
          <view v-for="tool in tools[plugin.id] || []" :key="tool.id" class="account-row">
            <view
              >{{ tool.source_name }} ·
              {{
                t(
                  tool.effect === 'read'
                    ? 'system.ai.workspace.read_only'
                    : 'system.ai.workspace.needs_confirmation',
                )
              }}</view
            >
            <view class="panel-hint">{{ tool.description }}</view>
            <view class="panel-hint">{{
              tool.unavailable_reason ||
              t(
                tool.effective_enabled
                  ? 'system.ai.workspace.available'
                  : 'system.ai.workspace.no_tools',
              )
            }}</view>
          </view>
        </view>
      </view>
      <view v-if="!loading && !errorText && !visiblePlugins.length" class="panel-hint">{{
        t('system.ai.workspace.no_plugins')
      }}</view>
    </scroll-view>
    <view class="panel-row">
      <button size="mini" @tap="load">{{ t('system.ai.workspace.refresh') }}</button>
      <template v-if="picker">
        <button size="mini" :disabled="saving" @tap="draft = []">
          {{ t('system.ai.workspace.clear_selection') }}
        </button>
        <button
          size="mini"
          class="is-active"
          :loading="saving"
          :disabled="saving || loading || Boolean(errorText)"
          @tap="emit('save', draft)"
        >
          {{ t('system.ai.workspace.save_selection') }}
        </button>
      </template>
    </view>
  </view>
</template>

<style lang="scss" scoped>
@use '../workspace.scss';
.plugin-panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  padding: 20rpx;
  box-sizing: border-box;
}
.is-picker {
  position: absolute;
  inset: 0;
  z-index: 1002;
  background: var(--kratos-color-surface-solid);
  padding-top: 100rpx;
  padding-bottom: calc(140rpx + env(safe-area-inset-bottom));
}
.account-row {
  margin-top: 16rpx;
  padding-top: 16rpx;
  border-top: 1rpx solid var(--kratos-color-border);
}
</style>
