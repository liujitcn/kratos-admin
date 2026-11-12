<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from '@liujitcn/kratos-uni-app-core'
import { defAiAutomationService } from '../../../../api/base/v1/ai_automation'
import { defAiPluginService } from '../../../../api/base/v1/ai_plugin'
import type { AiAutomation, AiAutomationRun } from '../../../../rpc/base/v1/ai_automation'
import type {
  AiPlugin,
  AiPluginConnection,
  AiPluginSelection,
} from '../../../../rpc/base/v1/ai_plugin'
import { canStopAutomationRun } from '../workspace'
import { Terminal } from '../../../../rpc/base/v1/ai_tool'

const props = defineProps<{ active: boolean }>()
const emit = defineEmits<{ result: [run: AiAutomationRun]; pending: [count: number] }>()
const { t } = useI18n()
const automations = ref<AiAutomation[]>([])
const runs = ref<AiAutomationRun[]>([])
const plugins = ref<AiPlugin[]>([])
const connections = ref<AiPluginConnection[]>([])
const selected = ref<AiAutomation>()
const selectedRun = ref<AiAutomationRun>()
const filter = ref('')
const keyword = ref('')
const loading = ref(false)
const busy = ref(false)
const errorText = ref('')
const requestKeys = new Map<string, string>()
let refreshTimer: ReturnType<typeof setInterval> | undefined

watch(
  () => props.active,
  (active) => {
    if (refreshTimer) clearInterval(refreshTimer)
    if (!active) return
    void load()
    refreshTimer = setInterval(() => {
      if (
        !errorText.value &&
        !busy.value &&
        runs.value.some((run) => ['queued', 'running', 'waiting_approval'].includes(run.status))
      )
        void load()
    }, 5000)
  },
  { immediate: true },
)
onBeforeUnmount(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})

/** 使用服务端筛选与执行快照刷新，不把接口失败显示成空列表。 */
async function load() {
  if (loading.value) return
  loading.value = true
  errorText.value = ''
  try {
    const [list, history, catalog, accounts] = await Promise.all([
      defAiAutomationService.ListAiAutomation({ status: filter.value, keyword: keyword.value }),
      defAiAutomationService.ListAiAutomationRun({
        automation_id: selected.value?.id || '',
        limit: 50,
      }),
      defAiPluginService.ListAiPlugin({ keyword: '', terminal: Terminal.TERMINAL_ADMIN }),
      defAiPluginService.ListAiPluginConnection({ plugin_id: '' }),
    ])
    automations.value = list.automations
    runs.value = history.runs
    plugins.value = catalog.plugins
    connections.value = accounts.connections
    emit('pending', list.pending_count)
    if (selected.value)
      selected.value = await defAiAutomationService.GetAiAutomation({ id: selected.value.id })
    if (selectedRun.value)
      selectedRun.value = await defAiAutomationService.GetAiAutomationRun({
        id: selectedRun.value.id,
      })
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : t('system.ai.workspace.load_failed')
  } finally {
    loading.value = false
  }
}

function selectFilter(status: string) {
  filter.value = status
  selected.value = undefined
  selectedRun.value = undefined
  void load()
}

async function chooseTask(task?: AiAutomation) {
  if (loading.value || busy.value) return
  selected.value = task
  selectedRun.value = undefined
  await load()
}

async function chooseRun(run: AiAutomationRun) {
  if (loading.value || busy.value) return
  busy.value = true
  try {
    selectedRun.value = await defAiAutomationService.GetAiAutomationRun({ id: run.id })
  } catch (error) {
    errorText.value = error instanceof Error ? error.message : t('system.ai.workspace.load_failed')
  } finally {
    busy.value = false
  }
}

/** 重试同一次发起复用幂等键，收到成功记录后才结束该操作。 */
async function runOnce(task: AiAutomation) {
  if (busy.value || loading.value) return
  busy.value = true
  try {
    const confirmation = await uni.showModal({
      title: t('system.ai.workspace.run_once'),
      content: t('system.ai.workspace.run_confirm'),
    })
    if (!confirmation.confirm) return
    const operation = `${task.id}:${task.revision}`
    if (!requestKeys.has(operation))
      requestKeys.set(operation, `${Date.now()}-${Math.random().toString(36).slice(2)}`)
    selectedRun.value = await defAiAutomationService.RunAiAutomation({
      id: task.id,
      revision: task.revision,
      request_key: requestKeys.get(operation)!,
    })
    requestKeys.delete(operation)
    await load()
  } catch (error) {
    errorText.value =
      error instanceof Error ? error.message : t('system.ai.workspace.action_failed')
  } finally {
    busy.value = false
  }
}

/** 停止请求只影响当前记录，界面始终展示服务端实际终态。 */
async function stopRun(run: AiAutomationRun) {
  if (busy.value || !canStopAutomationRun(run)) return
  busy.value = true
  try {
    const confirmation = await uni.showModal({
      title: t('system.ai.workspace.stop_run'),
      content: t('system.ai.workspace.stop_confirm'),
    })
    if (!confirmation.confirm) return
    selectedRun.value = await defAiAutomationService.StopAiAutomationRun({ id: run.id })
    await load()
  } catch (error) {
    errorText.value =
      error instanceof Error ? error.message : t('system.ai.workspace.action_failed')
  } finally {
    busy.value = false
  }
}

function formatTime(value?: { seconds: number; nanos: number }) {
  return value?.seconds ? new Date(value.seconds * 1000).toLocaleString() : '—'
}

function selectionLabel(selection: AiPluginSelection) {
  const plugin = plugins.value.find((item) => item.id === selection.plugin_id)
  const connection = connections.value.find((item) => item.id === selection.connection_id)
  return `${plugin?.name || selection.plugin_id}${selection.connection_id ? ` · ${connection?.label || selection.connection_id}` : ''}`
}
</script>

<template>
  <view class="automation-panel">
    <view class="panel-hint">{{ t('system.ai.workspace.desktop_configuration') }}</view>
    <view class="panel-row">
      <button
        v-for="status in ['', 'active', 'paused', 'attention', 'archived']"
        :key="status"
        size="mini"
        :class="{ 'is-active': filter === status }"
        :disabled="loading"
        @tap="selectFilter(status)"
      >
        {{ t(`system.ai.workspace.${status || 'all'}`) }}
      </button>
    </view>
    <input
      v-model="keyword"
      class="panel-search"
      :placeholder="t('system.ai.workspace.search_tasks')"
      confirm-type="search"
      @confirm="load"
    />
    <view class="panel-row"
      ><button size="mini" :disabled="loading" @tap="load">
        {{ t('system.ai.workspace.refresh') }}</button
      ><button v-if="selected || selectedRun" size="mini" @tap="chooseTask()">
        {{ t('system.ai.workspace.back') }}
      </button></view
    >
    <view v-if="loading" class="panel-hint">{{ t('system.ai.workspace.loading') }}</view>
    <view v-if="errorText" class="panel-error"
      >{{ errorText
      }}<button size="mini" @tap="load">{{ t('system.ai.workspace.retry') }}</button></view
    >
    <scroll-view class="panel-content" scroll-y>
      <view v-if="selectedRun" class="panel-card">
        <view class="panel-title"
          >{{ selectedRun.configuration?.name }} ·
          {{ t(`system.ai.workspace.${selectedRun.status}`) }}</view
        >
        <view class="panel-hint"
          >{{ t('system.ai.workspace.frozen_result') }} · {{ selectedRun.id }}</view
        >
        <view class="panel-hint">{{ selectedRun.configuration?.prompt }}</view>
        <view class="panel-hint"
          >{{ selectedRun.configuration?.model_name }} ·
          {{
            selectedRun.configuration?.schedule?.time ||
            selectedRun.configuration?.schedule?.once_at
          }}
          · {{ selectedRun.configuration?.schedule?.timezone }}</view
        >
        <view
          v-for="selection in selectedRun.configuration?.plugins || []"
          :key="`${selection.plugin_id}:${selection.connection_id}`"
          class="panel-hint"
          >{{ selectionLabel(selection) }}</view
        >
        <view class="panel-hint"
          >{{ formatTime(selectedRun.started_at || selectedRun.created_at) }} →
          {{ formatTime(selectedRun.finished_at) }}</view
        >
        <view class="panel-hint">{{ selectedRun.summary }}</view>
        <view v-if="selectedRun.error_code" class="panel-error">{{ selectedRun.error_code }}</view>
        <view
          v-if="selectedRun.result_unknown || selectedRun.status === 'unknown'"
          class="panel-error"
          >{{ t('system.ai.workspace.unknown_result') }}</view
        >
        <view v-if="selectedRun.cancel_requested" class="panel-hint">{{
          t('system.ai.workspace.stop_requested')
        }}</view>
        <view class="panel-row">
          <button
            v-if="selectedRun.session_id && !selectedRun.session_deleted"
            size="mini"
            @tap="emit('result', selectedRun)"
          >
            {{
              t(
                selectedRun.status === 'waiting_approval'
                  ? 'system.ai.workspace.review'
                  : 'system.ai.workspace.view_result',
              )
            }}
          </button>
          <button
            v-if="canStopAutomationRun(selectedRun)"
            size="mini"
            :disabled="busy"
            @tap="stopRun(selectedRun)"
          >
            {{ t('system.ai.workspace.stop_run') }}
          </button>
        </view>
        <view v-if="selectedRun.session_deleted" class="panel-hint">{{
          t('system.ai.workspace.result_deleted')
        }}</view>
      </view>
      <template v-else>
        <view v-for="task in selected ? [selected] : automations" :key="task.id" class="panel-card">
          <view class="panel-title" @tap="chooseTask(task)"
            >{{ task.configuration?.name }} · {{ t(`system.ai.workspace.${task.status}`) }}</view
          >
          <view class="panel-hint">{{ task.configuration?.prompt }}</view>
          <view class="panel-hint"
            >{{ t(`system.ai.workspace.${task.configuration?.schedule?.frequency || 'once'}`) }} ·
            {{ task.configuration?.schedule?.time || task.configuration?.schedule?.once_at }} ·
            {{ task.configuration?.schedule?.timezone }}
            <text v-if="task.configuration?.schedule?.weekdays.length">
              · {{ task.configuration.schedule.weekdays.join(', ') }}</text
            ></view
          >
          <view class="panel-hint">{{ task.configuration?.model_name }}</view>
          <view
            v-for="selection in task.configuration?.plugins || []"
            :key="`${selection.plugin_id}:${selection.connection_id}`"
            class="panel-hint"
            >{{ selectionLabel(selection) }}</view
          >
          <view class="panel-hint"
            >{{ t('system.ai.workspace.next_run') }} · {{ formatTime(task.next_fire_at) }}</view
          >
          <view v-if="task.last_run" class="panel-link" @tap="chooseRun(task.last_run)"
            >{{ t('system.ai.workspace.last_run') }} ·
            {{ t(`system.ai.workspace.${task.last_run.status}`) }}</view
          >
          <view v-if="task.problem" class="panel-error">{{ task.problem }}</view>
          <view class="panel-row">
            <button size="mini" @tap="chooseTask(task)">
              {{ t('system.ai.workspace.details') }}
            </button>
            <button
              v-if="task.status !== 'archived'"
              size="mini"
              :disabled="busy || Boolean(task.active_run_id)"
              @tap="runOnce(task)"
            >
              {{ t('system.ai.workspace.run_once') }}
            </button>
          </view>
        </view>
        <view
          v-if="!loading && !errorText && !selected && !automations.length"
          class="panel-hint"
          >{{ t('system.ai.workspace.no_tasks') }}</view
        >
      </template>
      <view class="panel-title">{{ t('system.ai.workspace.history') }}</view>
      <view v-for="run in runs" :key="run.id" class="panel-card" @tap="chooseRun(run)">
        <view>{{ run.configuration?.name }} · {{ t(`system.ai.workspace.${run.status}`) }}</view>
        <view class="panel-hint">{{ formatTime(run.created_at) }}</view>
        <view class="panel-hint">{{ run.summary }}</view>
      </view>
      <view v-if="!loading && !errorText && !runs.length" class="panel-hint">{{
        t('system.ai.workspace.no_runs')
      }}</view>
    </scroll-view>
  </view>
</template>

<style lang="scss" scoped>
@use '../workspace.scss';
.automation-panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  padding: 20rpx 20rpx calc(160rpx + env(safe-area-inset-bottom));
  box-sizing: border-box;
}
</style>
