<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { defAiMessageService, StreamAiMessageByChunkedRequest } from '../../../api/base/ai_message'
import { defAiSessionService } from '../../../api/base/ai_session'
import { defAiToolService } from '../../../api/base/ai_tool'
import type { AiMessage } from '../../../rpc/base/v1/ai_session'
import type { AiAttachment, AiSession } from '../../../rpc/base/v1/ai_session'
import type { AiShortcut, AiToolCall } from '../../../rpc/base/v1/ai_tool'
import { AiMessageStatus, Terminal } from '../../../rpc/base/v1/enum'
import { uploadFile } from '@liujitcn/kratos-uni-app-core/utils/file'
import { formatSrc } from '@liujitcn/kratos-uni-app-core/utils/index'
import Composer from './components/Composer.vue'
import SessionDrawer from './components/SessionDrawer.vue'
import WelcomePanel from './components/WelcomePanel.vue'
import { navigateAppRoute, useI18n } from '@liujitcn/kratos-uni-app-core'
import {
  type AiStreamEvent,
  type AiStreamPayload,
  createAiEventStreamTextParser,
  parseAiEventStreamText,
  readAiEventStream,
} from './stream'

type ChatRole = 'user' | 'ai'

type ChatMessageItem = AiMessage & {
  key: string
  messageID: string
  role: ChatRole
  content: string
  status: AiMessageStatus
  tools: AiToolCall[]
  model: string
  replySource: string
  fallback: boolean
  fallbackReason: string
  tokenTotal: number
  firstTokenMs: number
  durationMs: number
  localOnly?: boolean
  streamKey?: string
}

type AttachmentUpload = {
  path: string
  name: string
  size: number
}

type StreamTask = {
  abort: () => void
  aborted: boolean
  finished: boolean
  success?: boolean
}

const AI_TERMINAL = Terminal.TERMINAL_APP
const LOCAL_USER_MESSAGE_PREFIX = 'ai-user-local'
const PENDING_MESSAGE_ID = 'pending'
const MAX_ATTACHMENT_COUNT = 6
const STARTER_PROMPT_PAGE_SIZE = 4
const { locale, t } = useI18n()

const windowInfo = uni.getWindowInfo()
let safeAreaTop =
  windowInfo.safeArea?.top || windowInfo.statusBarHeight || windowInfo.safeAreaInsets?.top || 0
// #ifdef MP-WEIXIN
safeAreaTop = safeAreaTop || 44
// #endif
const safeAreaBottom = Math.max(windowInfo.safeAreaInsets?.bottom || 0, 9)
const windowHeight = windowInfo.windowHeight || windowInfo.screenHeight || 667
const composerBottom = `${safeAreaBottom + 48}px`
const drawerTopPadding = `${safeAreaTop + 12}px`

const showSessionDrawer = ref(false)
const activeSessionID = ref('')
const inputText = ref('')
const isRecording = ref(false)
const starterPromptGroupIndex = ref(0)
const sessionKeyword = ref('')
const loadingSessions = ref(false)
const loadingShortcuts = ref(false)
const loadingSessionID = ref('')
const uploadingAttachment = ref(false)
const sendingSessionMap = ref<Record<string, boolean>>({})
const chatBottomAnchor = ref('')
const sessions = ref<AiSession[]>([])
const messages = ref<Record<string, ChatMessageItem[]>>({})
const selectedAttachments = ref<AiAttachment[]>([])
const runningStreamTaskMap = new Map<string, StreamTask>()
const pendingDeltaMap = new Map<string, AiStreamPayload>()
let pendingDeltaTimer = 0

const createStarterShortcuts = (): AiShortcut[] => [
  {
    key: 'summarize',
    title: t('system.ai.prompt.summary'),
    prompt: t('system.ai.prompt.summaryContent'),
    action: undefined,
    required_tools: [],
    sort: 1,
    group: t('system.ai.textAssistant'),
  },
  {
    key: 'rewrite',
    title: t('system.ai.prompt.optimize'),
    prompt: t('system.ai.prompt.optimizeContent'),
    action: undefined,
    required_tools: [],
    sort: 2,
    group: t('system.ai.textAssistant'),
  },
  {
    key: 'plan',
    title: t('system.ai.prompt.plan'),
    prompt: t('system.ai.prompt.planContent'),
    action: undefined,
    required_tools: [],
    sort: 3,
    group: t('system.ai.efficiencyAssistant'),
  },
  {
    key: 'ideas',
    title: t('system.ai.prompt.idea'),
    prompt: t('system.ai.prompt.ideaContent'),
    action: undefined,
    required_tools: [],
    sort: 4,
    group: t('system.ai.efficiencyAssistant'),
  },
]
const starterShortcuts = ref<AiShortcut[]>(createStarterShortcuts())

const filteredSessions = computed(() => {
  const keyword = sessionKeyword.value.trim()
  if (!keyword) {
    return sessions.value
  }
  return sessions.value.filter(
    (item) => item.title.includes(keyword) || item.summary.includes(keyword),
  )
})

const currentMessages = computed(() => messages.value[activeSessionID.value] ?? [])
const hasMessages = computed(() => currentMessages.value.length > 0)
const currentSessionSending = computed(() => isSessionSending(activeSessionID.value))
const starterPromptPageCount = computed(() => {
  return Math.max(1, Math.ceil(starterShortcuts.value.length / STARTER_PROMPT_PAGE_SIZE))
})
const canRefreshStarterPrompts = computed(
  () => starterShortcuts.value.length > STARTER_PROMPT_PAGE_SIZE,
)
const starterPrompts = computed(() => {
  const pageIndex = starterPromptGroupIndex.value % starterPromptPageCount.value
  const start = pageIndex * STARTER_PROMPT_PAGE_SIZE
  return starterShortcuts.value.slice(start, start + STARTER_PROMPT_PAGE_SIZE)
})
const aiGreetingPeriod = computed(() => {
  const hour = new Date().getHours()
  if (hour < 11) {
    return t('system.ai.period.morning')
  }
  if (hour < 14) {
    return t('system.ai.period.noon')
  }
  if (hour < 18) {
    return t('system.ai.period.afternoon')
  }
  return t('system.ai.period.evening')
})
const aiGreetingMessage = computed(() =>
  t('system.ai.greeting', { period: aiGreetingPeriod.value }),
)
const composerPlaceholder = computed(() => {
  if (isRecording.value) {
    return t('system.ai.recording')
  }
  if (uploadingAttachment.value) {
    return t('system.ai.attachmentUploading')
  }
  return hasMessages.value
    ? t('system.ai.placeholder.continue')
    : t('system.ai.placeholder.default')
})
const isSubmitDisabled = computed(
  () =>
    uploadingAttachment.value ||
    currentSessionSending.value ||
    isRecording.value ||
    (!inputText.value.trim() && selectedAttachments.value.length === 0),
)

onLoad(() => {
  void loadAiShortcuts()
  void ensureSessionsLoaded()
})

watch(locale, () => {
  starterShortcuts.value = createStarterShortcuts()
  void loadAiShortcuts()
})

onBeforeUnmount(() => {
  cancelAllStreamTasks()
  clearPendingDelta()
  activeSessionID.value = ''
})

const toggleSessionDrawer = () => {
  showSessionDrawer.value = !showSessionDrawer.value
}

const selectSession = (sessionID: string) => {
  activeSessionID.value = sessionID
  showSessionDrawer.value = false
  if (!messages.value[sessionID]?.length || !isSessionSending(sessionID)) {
    void loadMessages(sessionID)
    return
  }
  scrollChatToBottom()
}

const createSession = async () => {
  try {
    const sessionID = await createRemoteSession()
    if (!sessionID) {
      return
    }
    activeSessionID.value = sessionID
    messages.value[sessionID] = []
    sessionKeyword.value = ''
    showSessionDrawer.value = false
  } catch (error) {
    showError(error, t('system.ai.createSessionFailed'))
  }
}

const deleteSession = async (sessionID: string) => {
  const session = sessions.value.find((item) => item.id === sessionID)
  const result = await uni.showModal({
    title: t('system.ai.deleteSession'),
    content: t('system.ai.deleteSessionConfirm', {
      title: session?.title || t('system.ai.currentSession'),
    }),
    confirmText: t('common.action.delete'),
    confirmColor: '#cf4444',
  })
  if (!result.confirm) {
    return
  }

  try {
    await defAiSessionService.DeleteAiSession({ id: sessionID })
    sessions.value = sessions.value.filter((item) => item.id !== sessionID)
    delete messages.value[sessionID]
    if (activeSessionID.value === sessionID) {
      activeSessionID.value = ''
      await ensureActiveSession()
    }
  } catch (error) {
    showError(error, t('system.ai.deleteSessionFailed'))
  }
}

const handleSessionAction = (session: AiSession) => {
  uni.showActionSheet({
    itemList: [t('system.ai.deleteSession')],
    success: ({ tapIndex }) => {
      if (tapIndex === 0) {
        void deleteSession(session.id)
      }
    },
  })
}

const copyMessage = (item: ChatMessageItem) => {
  uni.setClipboardData({
    data: item.content,
    success: () => uni.showToast({ icon: 'none', title: t('system.ai.copySuccess') }),
  })
}

const deleteMessage = async (item: ChatMessageItem) => {
  const sessionID = activeSessionID.value
  if (!sessionID) {
    return
  }
  try {
    if (!item.localOnly) {
      await defAiMessageService.DeleteAiMessage({
        session_id: sessionID,
        message_id: item.messageID,
      })
    }
    messages.value[sessionID] = (messages.value[sessionID] ?? []).filter(
      (message) => message.messageID !== item.messageID,
    )
  } catch (error) {
    showError(error, t('system.ai.deleteMessageFailed'))
  }
}

const regenerateMessage = async (item: ChatMessageItem) => {
  if (item.role !== 'ai' || item.localOnly || currentSessionSending.value) {
    return
  }
  setSessionSending(activeSessionID.value, true)
  try {
    const response = await defAiMessageService.RegenerateAiMessage({
      session_id: activeSessionID.value,
      message_id: item.messageID,
    })
    messages.value[activeSessionID.value] = normalizeMessageList(response.messages)
    if (response.session) {
      upsertSession(normalizeSession(response.session))
    }
  } catch (error) {
    showError(error, t('system.ai.regenerateFailed'))
  } finally {
    setSessionSending(activeSessionID.value, false)
  }
}

const handleMessageAction = (item: ChatMessageItem) => {
  const itemList =
    item.role === 'ai'
      ? [t('system.ai.action.copy'), t('common.action.delete'), t('system.ai.action.regenerate')]
      : [t('system.ai.action.copy'), t('common.action.delete')]
  uni.showActionSheet({
    itemList,
    success: ({ tapIndex }) => {
      if (tapIndex === 0) {
        copyMessage(item)
      } else if (tapIndex === 1) {
        void deleteMessage(item)
      } else {
        void regenerateMessage(item)
      }
    },
  })
}

const navigateBack = () => {
  const pages = getCurrentPages()
  if (pages.length > 1) {
    uni.navigateBack()
    return
  }
  navigateAppRoute('app/home')
}

const handleSend = async () => {
  if (isSubmitDisabled.value) {
    return
  }
  const text = inputText.value.trim() || t('system.ai.attachmentAnswer')
  inputText.value = ''
  const attachments = [...selectedAttachments.value]
  selectedAttachments.value = []
  await sendAiPayload({ text, attachments })
}

const handleStarterPrompt = async (shortcut: AiShortcut) => {
  if (currentSessionSending.value || loadingSessions.value) {
    return
  }
  await sendAiPayload({
    text: shortcut.prompt || shortcut.title,
    attachments: [],
  })
}

const refreshStarterPrompts = () => {
  if (!canRefreshStarterPrompts.value) {
    return
  }
  starterPromptGroupIndex.value = (starterPromptGroupIndex.value + 1) % starterPromptPageCount.value
}

const handleToggleRecord = () => {
  isRecording.value = !isRecording.value
  uni.showToast({
    icon: 'none',
    title: isRecording.value ? t('system.ai.recognizing') : t('system.ai.speechStopped'),
  })
}

const handleAttachment = () => {
  if (uploadingAttachment.value || currentSessionSending.value) {
    return
  }
  if (selectedAttachments.value.length >= MAX_ATTACHMENT_COUNT) {
    uni.showToast({
      icon: 'none',
      title: t('system.ai.attachmentLimit', { count: MAX_ATTACHMENT_COUNT }),
    })
    return
  }

  uni.chooseImage({
    count: MAX_ATTACHMENT_COUNT - selectedAttachments.value.length,
    sourceType: ['album', 'camera'],
    success: async (result) => {
      const paths = Array.isArray(result.tempFilePaths)
        ? result.tempFilePaths
        : [result.tempFilePaths]
      const tempFiles = Array.isArray(result.tempFiles)
        ? result.tempFiles
        : result.tempFiles
          ? [result.tempFiles]
          : []
      const files: AttachmentUpload[] = paths.map((path: string, index: number) => ({
        path,
        name:
          (tempFiles[index] as { name?: string } | undefined)?.name ||
          t('system.ai.imageName', { index: index + 1 }),
        size: Number((tempFiles[index] as { size?: number } | undefined)?.size || 0),
      }))
      uploadingAttachment.value = true
      try {
        const uploaded = await Promise.all(files.map((file) => uploadFile('ai', file.path)))
        const attachments = uploaded.map<AiAttachment>((file, index) => ({
          id: file.url || `${file.name}-${index}`,
          name: files[index]?.name || file.name,
          size: files[index]?.size || 0,
          url: file.url,
          mime_type: 'image/*',
        }))
        selectedAttachments.value = [...selectedAttachments.value, ...attachments].slice(
          0,
          MAX_ATTACHMENT_COUNT,
        )
      } catch (error) {
        showError(error, t('system.ai.uploadFailed'))
      } finally {
        uploadingAttachment.value = false
      }
    },
  })
}

const removeSelectedAttachment = (attachment: AiAttachment) => {
  selectedAttachments.value = selectedAttachments.value.filter((item) => item !== attachment)
}

const previewAttachment = (attachment: AiAttachment, attachments: AiAttachment[]) => {
  const current = formatSrc(attachment.url)
  const urls = attachments.map((item) => formatSrc(item.url)).filter(Boolean)
  if (!current) {
    return
  }
  uni.previewImage({ current, urls: urls.length ? urls : [current] })
}

/** 加载当前终端可用的通用 AI 快捷入口。 */
async function loadAiShortcuts() {
  if (loadingShortcuts.value) {
    return
  }
  loadingShortcuts.value = true
  try {
    const response = await defAiToolService.ListAiShortcut({
      terminal: AI_TERMINAL,
    })
    const shortcuts = normalizeStarterShortcuts(response.shortcuts).filter((item) => !item.action)
    if (shortcuts.length) {
      starterShortcuts.value = shortcuts
      starterPromptGroupIndex.value = 0
    }
  } catch (error) {
    showError(error, t('system.ai.loadShortcutsFailed'))
  } finally {
    loadingShortcuts.value = false
  }
}

async function sendAiPayload(payload: { text: string; attachments: AiAttachment[] }) {
  const sessionID = await ensureActiveSession()
  if (!sessionID || isSessionSending(sessionID)) {
    return false
  }

  const localUserMessage = createLocalUserMessage(payload)
  const thinkingMessage = createThinkingMessage({ sessionID })
  messages.value[sessionID] = sortMessages([
    ...(messages.value[sessionID] ?? []),
    localUserMessage,
    thinkingMessage,
  ])
  scrollChatToBottom()
  setSessionSending(sessionID, true)
  return runAiTask(sessionID, payload)
}

async function runAiTask(
  sessionID: string,
  payload: { text: string; attachments: AiAttachment[] },
) {
  let task: StreamTask | undefined
  const request = {
    session_id: sessionID,
    content: payload.text,
    attachments: payload.attachments,
    action: undefined,
  }
  try {
    let handledByStream = false

    // #ifdef MP-WEIXIN
    const parser = createAiEventStreamTextParser((event) => handleAiStreamEvent(event, task))
    const chunkedTask = StreamAiMessageByChunkedRequest(request, {
      onChunk: (chunkText) => parser.push(chunkText),
    })
    task = {
      aborted: false,
      finished: false,
      abort() {
        task!.aborted = true
        chunkedTask.abort()
      },
    }
    runningStreamTaskMap.set(sessionID, task)
    handledByStream = true
    await chunkedTask.promise
    parser.flush()
    if (!task.finished && !task.aborted) {
      throw new Error(t('system.ai.responseIncomplete'))
    }
    // #endif

    // #ifdef H5
    if (
      !handledByStream &&
      typeof fetch === 'function' &&
      typeof ReadableStream !== 'undefined' &&
      typeof AbortController !== 'undefined'
    ) {
      const controller = new AbortController()
      task = {
        aborted: false,
        finished: false,
        abort() {
          task!.aborted = true
          controller.abort()
        },
      }
      runningStreamTaskMap.set(sessionID, task)
      const response = await defAiMessageService.StreamAiMessage(request, {
        signal: controller.signal,
      })
      if (!response.body) {
        throw new Error(t('system.ai.streamEmpty'))
      }
      await readAiEventStream(
        response.body,
        (event) => handleAiStreamEvent(event, task),
        controller.signal,
      )
      if (!task.finished && !task.aborted) {
        throw new Error(t('system.ai.responseIncomplete'))
      }
      handledByStream = true
    }
    // #endif

    if (!handledByStream) {
      const response = await defAiMessageService.SendAiMessage(request)
      const nextMessages = normalizeNonStreamMessages(response)
      if (!nextMessages.length) {
        throw new Error(t('system.ai.responseEmpty'))
      }
      const success = hasSuccessfulAiMessages(nextMessages)
      messages.value[sessionID] = replacePendingMessages(
        messages.value[sessionID] ?? [],
        nextMessages,
      )
      scrollChatToBottom()
      if (response.session) {
        upsertSession(normalizeSession(response.session))
      }
      return success
    }
    return Boolean(task?.success)
  } catch (error) {
    if (task?.aborted) {
      return false
    }
    messages.value[sessionID] = markThinkingMessageFailed(messages.value[sessionID] ?? [])
    scrollChatToBottom()
    showError(error, t('system.ai.requestFailed'))
  } finally {
    if (task && runningStreamTaskMap.get(sessionID) === task) {
      runningStreamTaskMap.delete(sessionID)
    }
    setSessionSending(sessionID, false)
  }
}

function handleAiStreamEvent(event: AiStreamEvent, task?: StreamTask) {
  if (event.event === 'delta') {
    handleAiDelta(event.payload)
    return
  }
  if (event.event === 'finish') {
    handleAiFinish(event.payload, task)
    return
  }
  handleAiError(event.payload, task)
}

function handleAiDelta(payload: AiStreamPayload) {
  if (!payload.delta) {
    return
  }
  queueAiDelta(payload)
}

function handleAiFinish(payload: AiStreamPayload, task?: StreamTask) {
  const sessionID = payload.session_id
  if (!sessionID) {
    return
  }
  if (task) {
    task.finished = true
  }
  flushAiDelta()
  const nextMessages = normalizeMessageList(payload.messages)
  if (task) {
    task.success = hasSuccessfulAiMessages(nextMessages)
  }
  const current = messages.value[sessionID] ?? []
  const streamKey = payload.message_id ? buildStreamMessageKey(sessionID, payload.message_id) : ''
  const hasLocalStreamingMessages = current.some(
    (item) => item.localOnly && item.streamKey === streamKey,
  )
  messages.value[sessionID] =
    nextMessages.length || !hasLocalStreamingMessages
      ? replacePendingMessages(current, nextMessages, payload)
      : current
  scrollChatToBottom()
  if (payload.session) {
    upsertSession(normalizeSession(payload.session))
  }
}

function handleAiError(payload: AiStreamPayload, task?: StreamTask) {
  const sessionID = payload.session_id
  if (!sessionID) {
    return
  }
  if (task) {
    task.finished = true
    task.success = false
  }
  flushAiDelta()
  const nextMessages = normalizeMessageList(payload.messages)
  if (nextMessages.length) {
    messages.value[sessionID] = replacePendingMessages(
      messages.value[sessionID] ?? [],
      nextMessages,
      payload,
    )
    scrollChatToBottom()
    return
  }
  messages.value[sessionID] = markStreamingError(
    ensureStreamingMessage(messages.value[sessionID] ?? [], payload),
    payload,
  )
  scrollChatToBottom()
}

/** 合并同一时刻的流式分片，降低移动端频繁渲染压力。 */
function queueAiDelta(payload: AiStreamPayload) {
  const sessionID = payload.session_id
  const messageID = payload.message_id
  if (!sessionID || !messageID || !messages.value[sessionID]) {
    return
  }

  const key = buildStreamMessageKey(sessionID, messageID)
  const cachedPayload = pendingDeltaMap.get(key)
  pendingDeltaMap.set(key, {
    ...payload,
    delta: `${cachedPayload?.delta ?? ''}${payload.delta ?? ''}`,
  })

  if (pendingDeltaTimer) {
    return
  }
  pendingDeltaTimer = setTimeout(() => {
    pendingDeltaTimer = 0
    flushAiDelta()
  }, 32) as unknown as number
}

function flushAiDelta() {
  if (pendingDeltaTimer) {
    clearTimeout(pendingDeltaTimer)
    pendingDeltaTimer = 0
  }
  if (!pendingDeltaMap.size) {
    return
  }
  const payloadList = Array.from(pendingDeltaMap.values())
  pendingDeltaMap.clear()
  for (const payload of payloadList) {
    const sessionID = payload.session_id
    if (!sessionID || !messages.value[sessionID]) {
      continue
    }
    messages.value[sessionID] = appendStreamingDelta(
      ensureStreamingMessage(messages.value[sessionID] ?? [], payload),
      payload,
    )
    scrollChatToBottom()
  }
}

function clearPendingDelta() {
  if (pendingDeltaTimer) {
    clearTimeout(pendingDeltaTimer)
    pendingDeltaTimer = 0
  }
  pendingDeltaMap.clear()
}

function normalizeNonStreamMessages(response: unknown) {
  const jsonResponse = response as { messages?: AiMessage[] }
  if (Array.isArray(jsonResponse?.messages)) {
    return normalizeMessageList(jsonResponse.messages)
  }

  const events = parseAiEventStreamText(response)
  const finishEvent = [...events].reverse().find((item) => item.event === 'finish')
  if (finishEvent) {
    return normalizeMessageList(finishEvent.payload.messages)
  }
  const errorEvent = [...events].reverse().find((item) => item.event === 'error')
  if (errorEvent) {
    throw new Error(t('system.ai.requestFailed'))
  }
  return []
}

function replacePendingMessages(
  current: ChatMessageItem[],
  nextMessages: ChatMessageItem[],
  payload?: AiStreamPayload,
) {
  const sessionID = payload?.session_id ?? ''
  const streamKey = payload?.message_id ? buildStreamMessageKey(sessionID, payload.message_id) : ''
  const pendingStreamKey = sessionID ? buildPendingStreamMessageKey(sessionID) : ''
  const stableMessages = current.filter((item) => {
    if (!item.localOnly) {
      return true
    }
    if (payload?.message_id && item.role === 'user') {
      return !nextMessages.some(
        (message) => message.role === 'user' && message.messageID === payload.message_id,
      )
    }
    if (!streamKey) {
      return false
    }
    return item.streamKey !== streamKey && item.streamKey !== pendingStreamKey
  })
  const messageMap = new Map<string, ChatMessageItem>()
  for (const item of stableMessages) {
    messageMap.set(item.key, item)
  }
  for (const item of nextMessages) {
    messageMap.set(item.key, item)
  }
  return sortMessages(Array.from(messageMap.values()))
}

function buildStreamMessageKey(sessionID: string, messageID: string) {
  return `${sessionID}:${messageID}`
}

function buildPendingStreamMessageKey(sessionID: string) {
  return buildStreamMessageKey(sessionID, PENDING_MESSAGE_ID)
}

async function ensureSessionsLoaded() {
  if (loadingSessions.value || sessions.value.length > 0) {
    return
  }
  loadingSessions.value = true
  try {
    const response = await defAiSessionService.ListAiSession({
      terminal: AI_TERMINAL,
    })
    sessions.value = normalizeSessionList(response.sessions)
    const sessionID = await ensureActiveSession()
    if (sessionID) {
      await loadMessages(sessionID)
    }
  } catch (error) {
    showError(error, t('system.ai.loadSessionsFailed'))
  } finally {
    loadingSessions.value = false
  }
}

async function ensureActiveSession() {
  if (activeSessionID.value) {
    return activeSessionID.value
  }
  if (sessions.value.length > 0) {
    activeSessionID.value = sessions.value[0].id
    return activeSessionID.value
  }
  const sessionID = await createRemoteSession()
  if (sessionID) {
    activeSessionID.value = sessionID
    messages.value[sessionID] = []
  }
  return sessionID
}

async function createRemoteSession() {
  const response = await defAiSessionService.CreateAiSession({
    title: t('system.ai.newSession'),
    terminal: AI_TERMINAL,
  })
  const session = response.session ? normalizeSession(response.session) : undefined
  if (!session) {
    return ''
  }
  upsertSession(session)
  return session.id
}

async function loadMessages(sessionID: string) {
  if (!sessionID) {
    return
  }

  loadingSessionID.value = sessionID
  try {
    const response = await defAiSessionService.ListAiMessage({ session_id: sessionID })
    if (loadingSessionID.value !== sessionID) {
      return
    }
    messages.value[sessionID] = normalizeMessageList(response.messages)
    if (activeSessionID.value === sessionID) {
      scrollChatToBottom()
    }
  } catch (error) {
    if (loadingSessionID.value === sessionID) {
      messages.value[sessionID] = []
    }
    showError(error, t('system.ai.loadMessagesFailed'))
  } finally {
    if (loadingSessionID.value === sessionID) {
      loadingSessionID.value = ''
    }
  }
}

function normalizeSession(session?: Partial<AiSession> | null): AiSession {
  return {
    id: String(session?.id ?? ''),
    title: String(session?.title ?? t('system.ai.newSession')),
    summary: String(session?.summary ?? ''),
    updated_at: session?.updated_at,
    terminal: Number(session?.terminal ?? AI_TERMINAL),
  }
}

function normalizeSessionList(list?: AiSession[] | null) {
  if (!Array.isArray(list)) {
    return []
  }
  return list.map((item) => normalizeSession(item)).filter((item) => item.id)
}

function normalizeStarterShortcuts(list?: AiShortcut[] | null) {
  return [...(list || [])]
    .filter((item) => Boolean(item?.key && (item.title || item.prompt)))
    .map((item) => ({
      ...item,
      title: item.title || item.prompt,
      prompt: item.prompt || item.title,
      required_tools: Array.isArray(item.required_tools) ? item.required_tools : [],
      sort: Number(item.sort || 0),
    }))
    .sort((left, right) => left.sort - right.sort)
}

function normalizeMessageList(list?: AiMessage[] | null) {
  if (!Array.isArray(list)) {
    return []
  }

  return sortMessages(
    list
      .filter(Boolean)
      .flatMap((item) => [mapMessageItem(item, 'user'), mapMessageItem(item, 'ai')]),
  )
}

function hasSuccessfulAiMessages(list: ChatMessageItem[]) {
  return list.some((item) => item.status === AiMessageStatus.SUCCESS_AMS)
}

function mapMessageItem(message: AiMessage, role: ChatRole): ChatMessageItem {
  const inputContent = {
    kind: message.input_content?.kind || 'text',
    content: message.input_content?.content ?? '',
  }
  const outputContent = {
    kind: message.output_content?.kind || 'text',
    content: message.output_content?.content ?? '',
    reply_source: message.output_content?.reply_source ?? '',
    model: message.output_content?.model ?? '',
    fallback: Boolean(message.output_content?.fallback),
    fallback_reason: message.output_content?.fallback_reason ?? '',
    flow: message.output_content?.flow ?? '',
    step: message.output_content?.step ?? '',
    blocks_json: message.output_content?.blocks_json ?? '',
  }
  const status = Number(message.status ?? AiMessageStatus.SUCCESS_AMS)
  return {
    ...message,
    key: `${message.id}:${role}`,
    messageID: message.id,
    role,
    content: role === 'user' ? inputContent.content : outputContent.content,
    input_content: inputContent,
    output_content: outputContent,
    attachments: Array.isArray(message.attachments) ? message.attachments : [],
    status,
    token: {
      input: Number(message.token?.input ?? 0),
      output: Number(message.token?.output ?? 0),
      cache: Number(message.token?.cache ?? 0),
      total: Number(message.token?.total ?? 0),
    },
    tools: Array.isArray(message.tools) ? message.tools : [],
    model: role === 'ai' ? outputContent.model : '',
    replySource: role === 'ai' ? outputContent.reply_source : '',
    fallback: role === 'ai' && outputContent.fallback,
    fallbackReason: role === 'ai' ? outputContent.fallback_reason : '',
    tokenTotal: Number(message.token?.total ?? 0),
    firstTokenMs: Number(message.first_token_ms ?? 0),
    durationMs: Number(message.duration_ms ?? 0),
  }
}

function createLocalUserMessage(payload: { text: string; attachments: AiAttachment[] }) {
  const now = Date.now()
  const message = mapMessageItem(
    {
      id: `${LOCAL_USER_MESSAGE_PREFIX}-${now}`,
      input_content: { kind: 'text', content: payload.text },
      output_content: undefined,
      attachments: payload.attachments,
      created_at: {
        seconds: Math.floor(now / 1000),
        nanos: (now % 1000) * 1_000_000,
      },
      status: AiMessageStatus.GENERATING_AMS,
      token: { input: 0, output: 0, cache: 0, total: 0 },
      tools: [],
      first_token_ms: 0,
      duration_ms: 0,
    },
    'user',
  )
  message.localOnly = true
  message.status = AiMessageStatus.GENERATING_AMS
  return message
}

function createThinkingMessage(options?: { sessionID?: string; messageID?: string }) {
  const now = Date.now()
  const streamKey = options?.sessionID
    ? buildStreamMessageKey(options.sessionID, options.messageID || PENDING_MESSAGE_ID)
    : undefined
  const message = mapMessageItem(
    {
      id: streamKey || `ai-thinking-${now}`,
      input_content: undefined,
      output_content: {
        kind: 'text',
        content: t('system.ai.thinking'),
        reply_source: '',
        model: '',
        fallback: false,
        fallback_reason: '',
        flow: '',
        step: '',
        blocks_json: '',
      },
      attachments: [],
      created_at: {
        seconds: Math.floor(now / 1000),
        nanos: (now % 1000) * 1_000_000,
      },
      status: AiMessageStatus.GENERATING_AMS,
      token: { input: 0, output: 0, cache: 0, total: 0 },
      tools: [],
      first_token_ms: 0,
      duration_ms: 0,
    },
    'ai',
  )
  message.localOnly = true
  message.streamKey = streamKey
  return message
}

function ensureStreamingMessage(current: ChatMessageItem[], payload: AiStreamPayload) {
  const sessionID = payload.session_id
  const messageID = payload.message_id
  if (!sessionID || !messageID) {
    return current
  }

  const streamKey = buildStreamMessageKey(sessionID, messageID)
  if (current.some((item) => item.streamKey === streamKey)) {
    return current
  }

  const pendingStreamKey = buildPendingStreamMessageKey(sessionID)
  const next = current.map((item) =>
    item.streamKey === pendingStreamKey
      ? { ...item, id: messageID, messageID, key: `${messageID}:ai`, streamKey }
      : item,
  )
  if (next.some((item) => item.streamKey === streamKey)) {
    return next
  }

  return sortMessages([...next, createThinkingMessage({ sessionID, messageID })])
}

function appendStreamingDelta(current: ChatMessageItem[], payload: AiStreamPayload) {
  if (!payload.delta) {
    return current
  }
  const streamKey = buildStreamMessageKey(payload.session_id, payload.message_id)
  return current.map((item) => {
    if (item.streamKey !== streamKey || item.role === 'user') {
      return item
    }
    const baseContent = item.content === t('system.ai.thinking') ? '' : item.content
    return {
      ...item,
      content: `${baseContent}${payload.delta}`,
      status: AiMessageStatus.GENERATING_AMS,
    }
  })
}

function markThinkingMessageFailed(current: ChatMessageItem[]) {
  return current.map((item) => {
    if (!item.localOnly) {
      return item
    }
    return {
      ...item,
      status: AiMessageStatus.FAILED_AMS,
      content: item.role === 'ai' ? t('system.ai.failedResponse') : item.content,
    }
  })
}

function markStreamingError(current: ChatMessageItem[], payload: AiStreamPayload) {
  const streamKey = buildStreamMessageKey(payload.session_id, payload.message_id)
  return current.map((item) => {
    if (!item.localOnly || item.streamKey !== streamKey) {
      return item
    }
    return {
      ...item,
      status: AiMessageStatus.FAILED_AMS,
      content: t('system.ai.failedResponse'),
    }
  })
}

function sortMessages(list: ChatMessageItem[]) {
  return [...list].sort((left, right) => {
    const leftTime = resolveTimestamp(left.created_at)
    const rightTime = resolveTimestamp(right.created_at)
    if (leftTime === rightTime) {
      if (left.role !== right.role) {
        return left.role === 'user' ? -1 : 1
      }
      return left.messageID.localeCompare(right.messageID, locale.value, { numeric: true })
    }
    return leftTime - rightTime
  })
}

function upsertSession(session: AiSession) {
  const next = sessions.value.filter((item) => item.id !== session.id)
  next.unshift(session)
  sessions.value = next
}

function setSessionSending(sessionID: string, sending: boolean) {
  if (!sessionID) {
    return
  }
  sendingSessionMap.value = { ...sendingSessionMap.value, [sessionID]: sending }
}

function isSessionSending(sessionID: string) {
  return Boolean(sessionID && sendingSessionMap.value[sessionID])
}

function cancelAllStreamTasks() {
  runningStreamTaskMap.forEach((task) => {
    task.finished = true
    task.abort()
  })
  runningStreamTaskMap.clear()
}

function scrollChatToBottom() {
  void nextTick(() => {
    chatBottomAnchor.value = ''
    void nextTick(() => {
      chatBottomAnchor.value = 'chat-bottom'
    })
  })
}

function resolveTimestamp(timestamp: AiMessage['created_at'] | AiSession['updated_at']) {
  const seconds = Number(timestamp?.seconds || 0)
  const nanos = Number(timestamp?.nanos || 0)
  return seconds * 1000 + Math.floor(nanos / 1_000_000)
}

function isImageAttachment(attachment: AiAttachment) {
  return attachment.mime_type.startsWith('image/')
}

function formatAttachmentMeta(attachment: AiAttachment) {
  if (!attachment.size) {
    return t('system.ai.attachment')
  }
  return `${Math.max(1, Math.round(attachment.size / 1024))} KB`
}

function formatTools(tools: AiToolCall[]) {
  return tools
    .map((item) => item.title || item.name)
    .filter(Boolean)
    .join(' · ')
}

function showError(error: unknown, fallback: string) {
  const message = error instanceof Error ? error.message : fallback
  uni.showToast({ icon: 'none', title: message || fallback })
}
</script>

<template>
  <view class="ai-page">
    <view class="ai-navbar">
      <button class="nav-back-button" hover-class="none" @tap="navigateBack">
        <uni-icons type="left" size="24" color="#111" />
      </button>
      <view class="ai-navbar__title">{{ t('system.ai.chatTitle') }}</view>
      <button class="nav-menu-button" hover-class="none" @tap="toggleSessionDrawer">
        <uni-icons type="bars" size="24" color="#111" />
      </button>
    </view>

    <scroll-view
      class="ai-body"
      scroll-y
      scroll-with-animation
      :scroll-into-view="chatBottomAnchor"
      :show-scrollbar="false"
    >
      <template v-if="!hasMessages">
        <WelcomePanel
          :greeting-message="aiGreetingMessage"
          :loading="loadingSessions || loadingShortcuts"
          :shortcuts="starterPrompts"
          :can-refresh="canRefreshStarterPrompts"
          @refresh="refreshStarterPrompts"
          @shortcut-tap="handleStarterPrompt"
        />
      </template>

      <view v-else class="chat-list">
        <view
          v-for="item in currentMessages"
          :id="item.key"
          :key="item.key"
          class="message-row"
          :class="item.role === 'user' ? 'is-user' : 'is-ai'"
        >
          <view
            class="bubble"
            :class="[
              item.role === 'ai' ? 'ai-bubble' : 'user-bubble',
              item.status === AiMessageStatus.GENERATING_AMS ? 'is-streaming' : '',
            ]"
            @longpress="handleMessageAction(item)"
          >
            <view v-if="item.role === 'ai' && item.model" class="reply-meta">
              <text class="reply-tag">{{ t('system.ai.modelReply') }}</text>
              <text class="reply-model">{{ item.model }}</text>
            </view>
            <view class="bubble-content">{{ item.content }}</view>
            <view v-if="item.attachments.length" class="attachment-list">
              <view
                v-for="attachment in item.attachments"
                :key="attachment.id || attachment.url || attachment.name"
                class="attachment-card"
                @tap="previewAttachment(attachment, item.attachments)"
              >
                <view class="attachment-icon">{{
                  isImageAttachment(attachment)
                    ? t('system.ai.imageAttachment')
                    : t('system.ai.attachment')
                }}</view>
                <view class="attachment-info">
                  <view class="attachment-name">{{ attachment.name }}</view>
                  <view class="attachment-meta">{{ formatAttachmentMeta(attachment) }}</view>
                </view>
              </view>
            </view>
            <view v-if="item.tools.length" class="tool-row">{{
              t('system.ai.toolCalled', { tools: formatTools(item.tools) })
            }}</view>
          </view>
        </view>
        <view id="chat-bottom" class="chat-bottom"></view>
      </view>
      <view v-if="loadingSessionID" class="loading-session">{{ t('system.ai.loadMessages') }}</view>
    </scroll-view>

    <Composer
      v-model="inputText"
      :attachments="selectedAttachments"
      :placeholder="composerPlaceholder"
      :bottom="composerBottom"
      :recording="isRecording"
      :sending="currentSessionSending"
      :disabled="isSubmitDisabled"
      @attach="handleAttachment"
      @record="handleToggleRecord"
      @send="handleSend"
      @remove-attachment="removeSelectedAttachment"
    />

    <SessionDrawer
      :open="showSessionDrawer"
      :top-padding="drawerTopPadding"
      :keyword="sessionKeyword"
      :loading="loadingSessions"
      :sessions="filteredSessions"
      :active-session-id="activeSessionID"
      @close="showSessionDrawer = false"
      @create="createSession"
      @select="selectSession"
      @action="handleSessionAction"
      @update:keyword="sessionKeyword = $event"
    />
  </view>
</template>

<style lang="scss">
page {
  height: 100%;
  overflow: hidden;
  background-color: #f6f6f6;
}

.ai-page {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 100%;
  height: 100vh;
  height: 100dvh;
  overflow: hidden;
  color: #333;
  background-color: #f6f6f6;
}

.ai-navbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-sizing: border-box;
  background-color: #fff;
  border-bottom: 1rpx solid #eceef1;

  height: 44px;

  /* #ifdef MP-WEIXIN */
  height: 88px;
  padding-top: 44px;
  /* #endif */
}

/* #ifdef MP-WEIXIN */
.ai-navbar {
  position: relative;
}

.ai-navbar .nav-back-button,
.ai-navbar .nav-menu-button {
  position: absolute;
  bottom: 0;
}

.ai-navbar .nav-back-button {
  left: 0;
}

.ai-navbar .nav-menu-button {
  left: 56rpx;
  right: auto;
}

.ai-navbar__title {
  width: 100%;
  height: 44px;
  line-height: 44px;
}
/* #endif */

.nav-back-button,
.nav-menu-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88rpx;
  height: 44px;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  line-height: normal;
}

.nav-back-button::after,
.nav-menu-button::after {
  border: 0;
}

.ai-navbar__title {
  flex: 1;
  color: #111;
  font-size: 32rpx;
  font-weight: 600;
  text-align: center;
}

.ai-body {
  flex: 1;
  min-height: 0;
  box-sizing: border-box;
  padding: 28rpx 24rpx 10rpx;
}

.chat-list {
  padding-bottom: 24rpx;
}

.message-row {
  display: flex;
  width: 100%;
  margin-bottom: 24rpx;
}

.message-row.is-user {
  justify-content: flex-end;
}

.bubble {
  max-width: 86%;
  padding: 20rpx 24rpx;
  border-radius: 18rpx;
  box-sizing: border-box;
  word-break: break-word;
}

.user-bubble {
  color: #fff;
  background-color: #27ba9b;
  border-bottom-right-radius: 6rpx;
}

.ai-bubble {
  color: #333;
  background-color: #fff;
  border-bottom-left-radius: 6rpx;
  box-shadow: 0 8rpx 24rpx rgba(15, 23, 42, 0.05);
}

.bubble.is-streaming {
  opacity: 0.72;
}

.bubble-content {
  white-space: pre-wrap;
  font-size: 28rpx;
  line-height: 1.65;
}

.reply-meta {
  display: flex;
  align-items: center;
  gap: 12rpx;
  margin-bottom: 8rpx;
  color: #8a8f99;
  font-size: 20rpx;
}

.reply-tag {
  color: #16806d;
}

.attachment-list {
  margin-top: 16rpx;
}

.attachment-card {
  display: flex;
  align-items: center;
  gap: 12rpx;
  max-width: 100%;
  padding: 12rpx;
  margin-top: 8rpx;
  border-radius: 10rpx;
  background-color: rgba(255, 255, 255, 0.72);
  box-sizing: border-box;
}

.attachment-icon {
  flex-shrink: 0;
  width: 48rpx;
  height: 48rpx;
  border-radius: 8rpx;
  color: #16806d;
  font-size: 22rpx;
  line-height: 48rpx;
  text-align: center;
  background-color: #e8f8f4;
}

.attachment-info {
  min-width: 0;
}

.attachment-name,
.attachment-meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.attachment-name {
  max-width: 320rpx;
  font-size: 22rpx;
}

.attachment-meta {
  margin-top: 4rpx;
  color: #8a8f99;
  font-size: 19rpx;
}

.tool-row,
.loading-session {
  color: #8a8f99;
  font-size: 22rpx;
  line-height: 34rpx;
}

.tool-row {
  margin-top: 12rpx;
}

.loading-session {
  padding: 20rpx 0;
  text-align: center;
}

.chat-bottom {
  width: 2rpx;
  height: 2rpx;
}
</style>
