import { Button, Input, Picker, ScrollView, Text, View } from '@tarojs/components'
import { forwardRef, useCallback, useEffect, useImperativeHandle, useRef, useState } from 'react'
import { t, useI18n, useThemeStore } from '@liujitcn/kratos-taro-app-core'
import { UniIcon } from '@liujitcn/kratos-taro-app-ui'
import { defAiAutomationService } from '../../../../api/base/v1/ai_automation'
import {
  defAiFileService,
  isAiWorkspaceTextFile,
  openAiWorkspaceFile,
} from '../../../../api/base/v1/ai_file'
import { defAiPluginService } from '../../../../api/base/v1/ai_plugin'
import { AiFileView, type AiFile } from '../../../../rpc/base/v1/ai_file'
import { Terminal } from '../../../../rpc/base/v1/ai_tool'
import type { AiAutomation, AiAutomationRun } from '../../../../rpc/base/v1/ai_automation'
import type { AiPlugin, AiPluginConnection, AiPluginTool } from '../../../../rpc/base/v1/ai_plugin'
import type { AiKnowledgeOption } from '../../../../rpc/base/v1/ai_knowledge'
import type { AiProviderModelOption } from '../../../../rpc/base/v1/ai_provider'
import { resolveModelDisplayName } from '../modelDisplay'
import {
  type AiWorkspaceHandle,
  type AiWorkspaceView,
  formatWorkspaceFileSize,
  formatWorkspaceKnowledge,
  formatWorkspaceSchedule,
  formatWorkspaceTime,
  workspacePluginIssue,
  workspaceAutomationStatus,
  workspaceTimestamp,
} from '../workspace'
import DetailsSheet from './DetailsSheet'

/** 只读工作区的父页面回调。 */
type WorkspacePanelProps = {
  view: AiWorkspaceView
  modelProviders: AiProviderModelOption[]
  knowledgeOptions: AiKnowledgeOption[]
  onResult: (run: AiAutomationRun) => void
  onPending: (count: number) => void
  onTitle: (key: string) => void
}
/** 查询、分页与只读详情状态；不包含管理写操作表单。 */
type WorkspaceState = {
  keyword: string
  status: string
  fileView: AiFileView
  pluginTab: string
  scope: string
  pluginFilter: string
  page: number
  loading: boolean
  error: string
  tasks: AiAutomation[]
  runs: AiAutomationRun[]
  files: AiFile[]
  plugins: AiPlugin[]
  connections: AiPluginConnection[]
  tools: Record<string, AiPluginTool[]>
  totals: Record<number, number>
  recordsTask?: AiAutomation
  run?: AiAutomationRun
  task?: AiAutomation
  file?: AiFile
  plugin?: AiPlugin
  connection?: AiPluginConnection
  tool?: AiPluginTool
  fileText: string
  previewLoading: boolean
  expandedPluginTool: string
}

/** 工作区按钮复用统一主题与尺寸。 */
function Action({
  children,
  onClick,
  disabled = false,
  primary = false,
}: {
  children: React.ReactNode
  onClick: () => void
  disabled?: boolean
  primary?: boolean
}) {
  return (
    <Button
      className={`ai-workspace-button${primary ? ' is-primary' : ''}`}
      disabled={disabled || undefined}
      onClick={onClick}
    >
      {children}
    </Button>
  )
}
/** 只读补充信息。 */
function Meta({ children }: { children: React.ReactNode }) {
  return <View className='ai-workspace-meta'>{children}</View>
}
/** 按服务端实际状态显示标签，未知值不映射为成功。 */
function statusLabel(value: string) {
  const key = `system.ai.workspace.${value}`
  const label = t(key)
  return label === key ? value : label
}
/** 参数始终作为文本排版。 */
function schemaText(value: string) {
  try {
    return JSON.stringify(JSON.parse(value || '{}'), null, 2)
  } catch {
    return value
  }
}

const statuses = ['', 'active', 'paused', 'attention', 'archived']
const fileViews = [
  { value: AiFileView.AI_FILE_VIEW_ALL, label: 'all_files' },
  { value: AiFileView.AI_FILE_VIEW_UPLOADED, label: 'uploaded_files' },
  { value: AiFileView.AI_FILE_VIEW_GENERATED, label: 'generated_files' },
  { value: AiFileView.AI_FILE_VIEW_TRASHED, label: 'trash' },
]
const pluginTabs = ['catalog', 'accounts', 'tools']
const scopes = ['', 'system', 'tenant', 'private']

/** WorkspacePanel 还原原型的任务、文件、插件三个只读工作区。 */
const WorkspacePanel = forwardRef<AiWorkspaceHandle, WorkspacePanelProps>(function WorkspacePanel(
  { view, modelProviders, knowledgeOptions, onResult, onPending, onTitle },
  ref,
) {
  const { locale } = useI18n()
  const theme = useThemeStore()
  const [state, setState] = useState<WorkspaceState>({
    keyword: '',
    status: '',
    fileView: AiFileView.AI_FILE_VIEW_ALL,
    pluginTab: 'catalog',
    scope: '',
    pluginFilter: '',
    page: 1,
    loading: true,
    error: '',
    tasks: [],
    runs: [],
    files: [],
    plugins: [],
    connections: [],
    tools: {},
    totals: {},
    fileText: '',
    previewLoading: false,
    expandedPluginTool: '',
  })
  const sequence = useRef(0)
  const previewSequence = useRef(0)
  const pluginsLoaded = useRef(false)
  const nextPagePending = useRef(false)
  const patch = useCallback(
    (update: Partial<WorkspaceState> | ((previous: WorkspaceState) => Partial<WorkspaceState>)) =>
      setState((previous) => ({
        ...previous,
        ...(typeof update === 'function' ? update(previous) : update),
      })),
    [],
  )
  const { keyword, status, fileView, page, recordsTask } = state
  const request = useCallback(
    async function <T>(operation: () => Promise<T>, apply: (value: T) => void) {
      const current = ++sequence.current
      patch({ loading: true, error: '' })
      try {
        const response = await operation()
        if (current === sequence.current) apply(response)
      } catch (error) {
        if (current === sequence.current)
          patch({
            error: error instanceof Error ? error.message : t('system.ai.workspace.load_failed'),
          })
      } finally {
        if (current === sequence.current) {
          nextPagePending.current = false
          patch({ loading: false })
        }
      }
    },
    [patch],
  )

  const load = useCallback(async () => {
    if (view === 'automations') {
      if (recordsTask)
        await request(
          () =>
            defAiAutomationService.ListAiAutomationRun({
              automation_id: recordsTask.id,
              limit: 100,
            }),
          (value) => patch({ runs: value.runs || [] }),
        )
      else
        await request(
          async () => {
            const responses = await Promise.all([
              defAiAutomationService.ListAiAutomation({ status, keyword }),
              ...(status
                ? []
                : [defAiAutomationService.ListAiAutomation({ status: 'archived', keyword })]),
            ])
            return {
              responses,
              items: [
                ...new Map(
                  responses
                    .flatMap((value) => value.automations || [])
                    .map((item) => [item.id, item]),
                ).values(),
              ].sort(
                (left, right) =>
                  workspaceTimestamp(right.updated_at) - workspaceTimestamp(left.updated_at),
              ),
            }
          },
          (value) => {
            patch({ tasks: value.items })
            onPending(value.responses[0]?.pending_count || 0)
          },
        )
    } else if (view === 'files') {
      await request(
        async () => {
          const [response, counts] = await Promise.all([
            defAiFileService.PageAiFile({ view: fileView, keyword, page_num: page, page_size: 20 }),
            page === 1
              ? Promise.all(
                  fileViews
                    .filter((item) => item.value !== fileView)
                    .map(async (item) => ({
                      view: item.value,
                      response: await defAiFileService.PageAiFile({
                        view: item.value,
                        keyword,
                        page_num: 1,
                        page_size: 1,
                      }),
                    })),
                )
              : Promise.resolve([]),
          ])
          return { response, counts }
        },
        (value) =>
          patch((previous) => ({
            files:
              page === 1
                ? value.response.files || []
                : [...previous.files, ...(value.response.files || [])],
            totals: {
              ...previous.totals,
              [fileView]: Number(value.response.total || 0),
              ...Object.fromEntries(
                value.counts.map((item) => [item.view, Number(item.response.total || 0)]),
              ),
            },
          })),
      )
    } else if (view === 'plugins') {
      await request(
        async () => {
          const [catalog, accounts] = await Promise.all([
            defAiPluginService.ListAiPlugin({ terminal: Terminal.TERMINAL_APP, keyword: '' }),
            defAiPluginService.ListAiPluginConnection({ plugin_id: '' }),
          ])
          const definitions = catalog.plugins || []
          const snapshots = await Promise.all(
            definitions.map(
              async (plugin) =>
                [
                  plugin.id,
                  (
                    await defAiPluginService.ListAiPluginTool({
                      plugin_id: plugin.id,
                      terminal: Terminal.TERMINAL_APP,
                    })
                  ).tools || [],
                ] as const,
            ),
          )
          return { definitions, accounts: accounts.connections || [], snapshots }
        },
        (value) => {
          pluginsLoaded.current = true
          patch({
            plugins: value.definitions,
            connections: value.accounts,
            tools: Object.fromEntries(value.snapshots),
          })
        },
      )
    }
  }, [view, recordsTask, status, keyword, fileView, page, request, patch, onPending])

  useEffect(() => {
    if (view === 'plugins' && pluginsLoaded.current) return
    const timer = setTimeout(() => void load(), keyword ? 250 : 0)
    return () => {
      clearTimeout(timer)
      sequence.current += 1
    }
  }, [view, keyword, load])
  useEffect(
    () => () => {
      sequence.current += 1
      previewSequence.current += 1
    },
    [],
  )
  useEffect(() => {
    onTitle(`system.ai.workspace.${state.run ? 'run_details' : recordsTask ? 'history' : view}`)
  }, [state.run, recordsTask, view, onTitle])

  const closeDetails = useCallback(() => {
    previewSequence.current += 1
    patch({
      task: undefined,
      file: undefined,
      plugin: undefined,
      connection: undefined,
      tool: undefined,
      fileText: '',
      previewLoading: false,
      expandedPluginTool: '',
    })
  }, [patch])
  const sheetTitle =
    state.task?.configuration?.name ||
    state.file?.file_name ||
    state.plugin?.name ||
    state.connection?.label ||
    state.tool?.source_name ||
    ''
  useImperativeHandle(
    ref,
    () => ({
      goBack() {
        if (sheetTitle) {
          closeDetails()
          return true
        }
        if (state.run) {
          patch({ run: undefined })
          return true
        }
        if (recordsTask) {
          patch({ recordsTask: undefined, runs: [] })
          return true
        }
        return false
      },
    }),
    [sheetTitle, closeDetails, state.run, recordsTask, patch],
  )

  const formatTime = (value: Parameters<typeof formatWorkspaceTime>[0], timezone?: string) =>
    formatWorkspaceTime(value, locale, timezone)
  const pluginName = (id: string) => state.plugins.find((plugin) => plugin.id === id)?.name || id
  const pluginIssue = (plugin: AiPlugin) => {
    const reason = workspacePluginIssue(plugin, state.tools[plugin.id] || [])
    return reason ? statusLabel(reason) : t('system.ai.workspace.auto_available')
  }
  const openRuns = (task: AiAutomation) => {
    closeDetails()
    patch({ recordsTask: task, run: undefined, runs: [], keyword: '' })
  }
  const openPluginTab = (tab: string, id: string) => {
    closeDetails()
    patch({ pluginTab: tab, pluginFilter: id, keyword: '' })
  }
  const loadMore = () => {
    if (
      view === 'files' &&
      !state.loading &&
      !nextPagePending.current &&
      state.files.length < (state.totals[fileView] || 0)
    ) {
      nextPagePending.current = true
      patch({ page: page + 1 })
    }
  }
  const readFile = async (file: AiFile, preview: boolean) => {
    const current = ++previewSequence.current
    patch({ previewLoading: true, error: '' })
    try {
      const text = await openAiWorkspaceFile(file, preview)
      if (current === previewSequence.current) patch({ fileText: text })
    } catch (error) {
      if (current === previewSequence.current)
        patch({
          error: error instanceof Error ? error.message : t('system.ai.workspace.file_unavailable'),
        })
    } finally {
      if (current === previewSequence.current) patch({ previewLoading: false })
    }
  }
  const openFile = (file: AiFile) => {
    closeDetails()
    patch({ file })
    if (file.status !== 'trashed' && isAiWorkspaceTextFile(file)) void readFile(file, true)
  }
  const search = (placeholder: string) => (
    <View className='ai-workspace-search'>
      <UniIcon type='search' size={16} color={theme.colors.aiTextMuted} />
      <Input
        className='ai-workspace-input'
        value={keyword}
        maxlength={100}
        placeholder={placeholder}
        placeholderClass='ai-workspace-placeholder'
        confirmType='search'
        onInput={(event) => patch({ keyword: event.detail.value, page: 1 })}
        onConfirm={() => void load()}
      />
    </View>
  )
  const tag = (value: string) => (
    <Text className={`ai-workspace-tag${value === 'attention' ? ' is-warning' : ''}`}>
      {statusLabel(value)}
    </Text>
  )
  const noItems = (value: boolean, key: string) =>
    !state.loading && !state.error && value ? (
      <View className='ai-workspace-empty'>{t(`system.ai.workspace.${key}`)}</View>
    ) : null
  const visiblePlugins = state.plugins.filter(
    (item) =>
      (!state.scope || item.visibility === state.scope) &&
      `${item.name} ${item.description}`.toLowerCase().includes(keyword.toLowerCase()),
  )
  const visibleConnections = state.connections.filter(
    (item) =>
      (!state.pluginFilter || item.plugin_id === state.pluginFilter) &&
      `${item.label} ${pluginName(item.plugin_id)}`.toLowerCase().includes(keyword.toLowerCase()),
  )
  const visibleTools = Object.values(state.tools)
    .flat()
    .filter(
      (item) =>
        (!state.pluginFilter || item.plugin_id === state.pluginFilter) &&
        `${item.source_name} ${item.description}`.toLowerCase().includes(keyword.toLowerCase()),
    )

  return (
    <View className='ai-workspace-root'>
      <ScrollView
        className='ai-workspace-scroll'
        scrollY
        showScrollbar={false}
        onScrollToLower={loadMore}
      >
        <View className='ai-workspace'>
          {state.error ? (
            <View className='ai-workspace-error'>
              {state.error}
              <Action onClick={() => void load()}>{t('system.ai.workspace.retry')}</Action>
            </View>
          ) : null}
          {state.loading ? <Meta>{t('system.ai.workspace.loading')}</Meta> : null}
          {view === 'automations' ? (
            state.run ? (
              <View className='ai-workspace-card'>
                <View className='ai-workspace-head'>
                  <View className='ai-workspace-title'>{state.run.configuration?.name}</View>
                  {tag(state.run.status)}
                </View>
                <View className='ai-workspace-description'>{state.run.summary}</View>
                {state.run.error_code ? (
                  <View className='ai-workspace-error'>{state.run.error_code}</View>
                ) : null}
                <Meta>
                  {t('system.ai.workspace.time')}：
                  {formatTime(state.run.created_at, state.run.configuration?.schedule?.timezone)}
                </Meta>
                <Meta>
                  {t('system.ai.workspace.model')}：
                  {resolveModelDisplayName(
                    modelProviders,
                    state.run.configuration?.model_name || '',
                  )}
                </Meta>
                <Meta>
                  {t('system.ai.workspace.timezone')}：{state.run.configuration?.schedule?.timezone}
                </Meta>
                <Meta>
                  {t('system.ai.knowledge.title')}：
                  {formatWorkspaceKnowledge(
                    state.run.configuration?.knowledge_base_ids,
                    knowledgeOptions,
                  )}
                </Meta>
                <View className='ai-workspace-title'>
                  {t('system.ai.workspace.frozen_instruction')}
                </View>
                <Text className='ai-workspace-pre' selectable>
                  {state.run.configuration?.prompt}
                </Text>
                {state.run.status === 'waiting_approval' ? (
                  <View className='ai-workspace-hint'>
                    {t('system.ai.workspace.review_on_admin')}
                  </View>
                ) : null}
                {state.run.result_unknown ? (
                  <View className='ai-workspace-hint'>
                    {t('system.ai.workspace.unknown_result')}
                  </View>
                ) : null}
                <View className='ai-workspace-actions'>
                  <Action
                    primary
                    disabled={!state.run.session_id || state.run.session_deleted}
                    onClick={() => onResult(state.run!)}
                  >
                    {t('system.ai.workspace.view_result')}
                  </Action>
                </View>
              </View>
            ) : recordsTask ? (
              <>
                <View className='ai-workspace-head'>
                  <View className='ai-workspace-title'>{recordsTask.configuration?.name}</View>
                  <Action disabled={state.loading} onClick={() => void load()}>
                    {t('system.ai.workspace.refresh')}
                  </Action>
                </View>
                {state.runs.map((run) => (
                  <View className='ai-workspace-card' key={run.id}>
                    <View className='ai-workspace-head'>
                      <View className='ai-workspace-title'>{run.configuration?.name}</View>
                      {tag(run.status)}
                    </View>
                    <Meta>{formatTime(run.created_at, run.configuration?.schedule?.timezone)}</Meta>
                    <View className='ai-workspace-description'>{run.summary}</View>
                    <View className='ai-workspace-actions'>
                      <Action
                        onClick={() =>
                          void request(
                            () => defAiAutomationService.GetAiAutomationRun({ id: run.id }),
                            (value) => patch({ run: value }),
                          )
                        }
                      >
                        {t('system.ai.workspace.view_details')}
                      </Action>
                      <Action
                        primary
                        disabled={!run.session_id || run.session_deleted}
                        onClick={() => onResult(run)}
                      >
                        {t('system.ai.workspace.view_result')}
                      </Action>
                    </View>
                  </View>
                ))}
                {noItems(!state.runs.length, 'no_runs')}
              </>
            ) : (
              <>
                <View className='ai-workspace-tabs'>
                  {statuses.map((value) => (
                    <Button
                      key={value}
                      className={`ai-workspace-chip${status === value ? ' is-active' : ''}`}
                      onClick={() => patch({ status: value })}
                    >
                      {statusLabel(value || 'all')}
                    </Button>
                  ))}
                </View>
                {search(t('system.ai.workspace.search_tasks'))}
                {state.tasks.map((task) => (
                  <View className='ai-workspace-card' key={task.id}>
                    <View className='ai-workspace-head'>
                      <View className='ai-workspace-title'>{task.configuration?.name}</View>
                      {tag(workspaceAutomationStatus(task))}
                    </View>
                    <View className='ai-workspace-description'>{task.configuration?.prompt}</View>
                    <Meta>
                      {formatWorkspaceSchedule(task.configuration?.schedule, t)} ·{' '}
                      {task.configuration?.schedule?.timezone}
                    </Meta>
                    <Meta>
                      {t('system.ai.workspace.next_run')}：
                      {formatTime(task.next_fire_at, task.configuration?.schedule?.timezone)}
                    </Meta>
                    {task.problem ? (
                      <View className='ai-workspace-hint'>{task.problem}</View>
                    ) : task.last_run?.status === 'waiting_approval' ? (
                      <View className='ai-workspace-hint'>
                        {t('system.ai.workspace.review_on_admin')}
                      </View>
                    ) : null}
                    <View className='ai-workspace-actions'>
                      <Action
                        disabled={state.loading}
                        onClick={() =>
                          void request(
                            () => defAiAutomationService.GetAiAutomation({ id: task.id }),
                            (value) => patch({ task: value }),
                          )
                        }
                      >
                        {t('system.ai.workspace.view_details')}
                      </Action>
                      <Action onClick={() => openRuns(task)}>
                        {t('system.ai.workspace.history')}
                      </Action>
                    </View>
                  </View>
                ))}
                {noItems(!state.tasks.length, 'no_tasks')}
              </>
            )
          ) : null}
          {view === 'files' ? (
            <>
              <View className='ai-workspace-tabs'>
                {fileViews.map((item) => (
                  <Button
                    key={item.value}
                    className={`ai-workspace-chip${fileView === item.value ? ' is-active' : ''}`}
                    onClick={() => patch({ fileView: item.value, page: 1 })}
                  >
                    {statusLabel(item.label)}
                    {state.totals[item.value] !== undefined ? ` ${state.totals[item.value]}` : ''}
                  </Button>
                ))}
              </View>
              {search(t('system.ai.workspace.search_files'))}
              {state.files.map((file) => (
                <View
                  key={file.id}
                  className='ai-workspace-card ai-workspace-file'
                  onClick={() => openFile(file)}
                >
                  <UniIcon type='list' size={23} color={theme.colors.aiPrimary} />
                  <View className='ai-workspace-file-copy'>
                    <Text className='ai-workspace-title'>{file.file_name}</Text>
                    <Meta>
                      {formatWorkspaceFileSize(file.size)} ·{' '}
                      {statusLabel(
                        file.source_type === 'upload' ? 'uploaded_files' : 'generated_files',
                      )}
                    </Meta>
                    <Meta>{formatTime(file.created_at)}</Meta>
                  </View>
                  <UniIcon type='right' size={20} color={theme.colors.aiTextMuted} />
                </View>
              ))}
              {noItems(!state.files.length, 'no_files')}
              {state.files.length < (state.totals[fileView] || 0) ? (
                <Action disabled={state.loading} onClick={loadMore}>
                  {t('system.ai.workspace.load_more')}
                </Action>
              ) : null}
            </>
          ) : null}
          {view === 'plugins' ? (
            <>
              <View className='ai-workspace-tabs'>
                {pluginTabs.map((value) => (
                  <Button
                    key={value}
                    className={`ai-workspace-chip${state.pluginTab === value ? ' is-active' : ''}`}
                    onClick={() => patch({ pluginTab: value, pluginFilter: '', keyword: '' })}
                  >
                    {statusLabel(value)}
                  </Button>
                ))}
              </View>
              <Meta>{t('system.ai.workspace.auto_hint')}</Meta>
              {search(
                t(
                  `system.ai.workspace.${state.pluginTab === 'accounts' ? 'search_accounts' : state.pluginTab === 'tools' ? 'search_tools' : 'search_plugins'}`,
                ),
              )}
              {state.pluginTab === 'catalog' ? (
                <>
                  <Picker
                    range={scopes.map((value) => statusLabel(value || 'all'))}
                    value={scopes.indexOf(state.scope)}
                    onChange={(event) => patch({ scope: scopes[Number(event.detail.value)] || '' })}
                  >
                    <View className='ai-workspace-select'>
                      <Text>
                        {t('system.ai.workspace.scope')} · {statusLabel(state.scope || 'all')}
                      </Text>
                      <UniIcon type='down' size={12} color={theme.colors.aiTextMuted} />
                    </View>
                  </Picker>
                  {visiblePlugins.map((plugin) => (
                    <View key={plugin.id} className='ai-workspace-card'>
                      <View className='ai-workspace-head'>
                        <View className='ai-workspace-title'>{plugin.name}</View>
                        <Text className='ai-workspace-tag'>{plugin.source}</Text>
                      </View>
                      <View className='ai-workspace-description'>{plugin.description}</View>
                      <View className='ai-workspace-head'>
                        <Meta>{statusLabel(plugin.visibility)}</Meta>
                        <Text className='ai-workspace-tag'>{pluginIssue(plugin)}</Text>
                      </View>
                      <View className='ai-workspace-actions'>
                        <Action onClick={() => patch({ plugin })}>
                          {t('system.ai.workspace.view_details')}
                        </Action>
                        {plugin.source === 'MCP' ? (
                          <Action onClick={() => openPluginTab('accounts', plugin.id)}>
                            {t('system.ai.workspace.view_accounts')}
                          </Action>
                        ) : null}
                        <Action onClick={() => openPluginTab('tools', plugin.id)}>
                          {t('system.ai.workspace.view_tools')}
                        </Action>
                      </View>
                    </View>
                  ))}
                  {noItems(!visiblePlugins.length, 'no_plugins')}
                </>
              ) : state.pluginTab === 'accounts' ? (
                <>
                  {visibleConnections.map((connection) => (
                    <View key={connection.id} className='ai-workspace-card'>
                      <View className='ai-workspace-head'>
                        <View className='ai-workspace-title'>{connection.label}</View>
                        {tag(`connection_${connection.state}`)}
                      </View>
                      <Meta>
                        {pluginName(connection.plugin_id)} ·{' '}
                        {statusLabel(connection.sharing === 'tenant' ? 'shared' : 'personal')}
                      </Meta>
                      <Meta>
                        {t('system.ai.workspace.authorization_scope')}：{connection.scopes || '—'}
                      </Meta>
                      {connection.last_error ? (
                        <View className='ai-workspace-hint'>{connection.last_error}</View>
                      ) : null}
                      <View className='ai-workspace-actions'>
                        <Action onClick={() => patch({ connection })}>
                          {t('system.ai.workspace.view_details')}
                        </Action>
                      </View>
                    </View>
                  ))}
                  {noItems(!visibleConnections.length, 'no_accounts')}
                </>
              ) : (
                <>
                  {visibleTools.map((tool) => (
                    <View key={tool.id} className='ai-workspace-card'>
                      <View className='ai-workspace-head'>
                        <View className='ai-workspace-title'>{tool.source_name}</View>
                        {tag(tool.effective_enabled ? 'enabled' : 'disabled')}
                      </View>
                      <Meta>
                        {pluginName(tool.plugin_id)} ·{' '}
                        {t(
                          tool.effect === 'read'
                            ? 'system.ai.workspace.read_only'
                            : 'system.ai.workspace.needs_confirmation',
                        )}
                      </Meta>
                      <View className='ai-workspace-description'>{tool.description}</View>
                      <View className='ai-workspace-actions'>
                        <Action onClick={() => patch({ tool })}>
                          {t('system.ai.workspace.parameters_policy')}
                        </Action>
                      </View>
                    </View>
                  ))}
                  {noItems(!visibleTools.length, 'no_tools')}
                </>
              )}
            </>
          ) : null}
        </View>
      </ScrollView>
      {sheetTitle ? (
        <DetailsSheet
          title={sheetTitle}
          onClose={closeDetails}
          footer={
            state.task ? (
              <Action onClick={() => openRuns(state.task!)}>
                {t('system.ai.workspace.history')}
              </Action>
            ) : state.file?.status === 'active' ? (
              <>
                <Action
                  disabled={state.previewLoading}
                  onClick={() => void readFile(state.file!, true)}
                >
                  {t('system.ai.workspace.open_file')}
                </Action>
                <Action
                  primary
                  disabled={state.previewLoading}
                  onClick={() => void readFile(state.file!, false)}
                >
                  {t('common.action.download')}
                </Action>
              </>
            ) : state.plugin ? (
              <>
                {state.plugin.source === 'MCP' ? (
                  <Action onClick={() => openPluginTab('accounts', state.plugin!.id)}>
                    {t('system.ai.workspace.view_accounts')}
                  </Action>
                ) : null}
                <Action onClick={() => openPluginTab('tools', state.plugin!.id)}>
                  {t('system.ai.workspace.view_tools')}
                </Action>
              </>
            ) : undefined
          }
        >
          {state.error ? <View className='ai-workspace-error'>{state.error}</View> : null}
          {state.task ? (
            <>
              <View className='ai-workspace-description'>{state.task.configuration?.prompt}</View>
              <Meta>
                {formatWorkspaceSchedule(state.task.configuration?.schedule, t)} ·{' '}
                {state.task.configuration?.schedule?.timezone}
              </Meta>
              <Meta>
                {t('system.ai.workspace.model')}：
                {resolveModelDisplayName(
                  modelProviders,
                  state.task.configuration?.model_name || '',
                )}
              </Meta>
              <Meta>
                {t('system.ai.knowledge.title')}：
                {formatWorkspaceKnowledge(
                  state.task.configuration?.knowledge_base_ids,
                  knowledgeOptions,
                )}
              </Meta>
              <Meta>
                {t('system.ai.workspace.next_run')}：
                {formatTime(state.task.next_fire_at, state.task.configuration?.schedule?.timezone)}
              </Meta>
            </>
          ) : state.file ? (
            <>
              <Meta>
                {formatWorkspaceFileSize(state.file.size)} ·{' '}
                {statusLabel(
                  state.file.source_type === 'upload' ? 'uploaded_files' : 'generated_files',
                )}{' '}
                · {formatTime(state.file.created_at)}
              </Meta>
              {state.file.status === 'trashed' ? (
                <View className='ai-workspace-hint'>{t('system.ai.workspace.trash_readonly')}</View>
              ) : state.previewLoading ? (
                <Meta>{t('system.ai.workspace.loading')}</Meta>
              ) : state.fileText ? (
                <Text className='ai-workspace-pre' selectable>
                  {state.fileText}
                </Text>
              ) : (
                <Meta>{t('system.ai.workspace.file_preview_hint')}</Meta>
              )}
            </>
          ) : state.plugin ? (
            <>
              <View className='ai-workspace-description'>{state.plugin.description}</View>
              <Meta>
                {state.plugin.source} · {statusLabel(state.plugin.visibility)}
              </Meta>
              <Meta>
                {t('system.ai.workspace.code')}：{state.plugin.code}
              </Meta>
              {state.plugin.endpoint ? (
                <Meta>
                  {t('system.ai.workspace.endpoint')}：{state.plugin.endpoint}
                </Meta>
              ) : null}
              <Meta>
                {t('system.ai.workspace.terminals')}：
                {(state.plugin.installation?.terminals || []).map(statusLabel).join('、')}
              </Meta>
              <Meta>{pluginIssue(state.plugin)}</Meta>
              <View className='ai-workspace-title'>
                {t('system.ai.workspace.tools')} · {(state.tools[state.plugin.id] || []).length}
              </View>
              {(state.tools[state.plugin.id] || []).map((tool) => (
                <View key={tool.id} className='ai-workspace-card'>
                  <View
                    className='ai-workspace-title'
                    onClick={() =>
                      patch({
                        expandedPluginTool: state.expandedPluginTool === tool.id ? '' : tool.id,
                      })
                    }
                  >
                    {tool.source_name} {state.expandedPluginTool === tool.id ? '−' : '+'}
                  </View>
                  {state.expandedPluginTool === tool.id ? (
                    <>
                      <Meta>{tool.description}</Meta>
                      <Meta>
                        {statusLabel(tool.allowed ? 'enabled' : 'disabled')} ·{' '}
                        {statusLabel(tool.effect === 'read' ? 'read_only' : 'needs_confirmation')}
                      </Meta>
                      <Meta>{(tool.prompts || []).join('\n') || '—'}</Meta>
                      <Text className='ai-workspace-pre' selectable>
                        {schemaText(tool.input_schema)}
                      </Text>
                    </>
                  ) : null}
                </View>
              ))}
            </>
          ) : state.connection ? (
            <>
              <Meta>
                {pluginName(state.connection.plugin_id)} ·{' '}
                {statusLabel(`connection_${state.connection.state}`)}
              </Meta>
              <Meta>
                {t('system.ai.workspace.auth_type')}：{state.connection.auth_type}
              </Meta>
              <Meta>
                {t('system.ai.workspace.authorization_scope')}：{state.connection.scopes || '—'}
              </Meta>
              <Meta>
                {statusLabel(state.connection.sharing === 'tenant' ? 'shared' : 'personal')}
              </Meta>
              {state.connection.last_error ? (
                <View className='ai-workspace-hint'>{state.connection.last_error}</View>
              ) : null}
            </>
          ) : state.tool ? (
            <>
              <View className='ai-workspace-description'>{state.tool.description}</View>
              <Meta>
                {t(
                  state.tool.effect === 'read'
                    ? 'system.ai.workspace.read_only'
                    : 'system.ai.workspace.needs_confirmation',
                )}{' '}
                · {statusLabel(state.tool.allowed ? 'enabled' : 'disabled')}
              </Meta>
              <Text className='ai-workspace-pre' selectable>
                {schemaText(state.tool.input_schema)}
              </Text>
              <Meta>{(state.tool.prompts || []).join('\n')}</Meta>
            </>
          ) : null}
        </DetailsSheet>
      ) : null}
    </View>
  )
})
export default WorkspacePanel
