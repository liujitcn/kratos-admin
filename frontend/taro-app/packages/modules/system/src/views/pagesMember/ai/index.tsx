import { Button, ScrollView, Text, View } from '@tarojs/components'
import Taro, { getCurrentPages, useLoad } from '@tarojs/taro'
import { useEffect, useMemo, useRef, useState } from 'react'
import {
  formatSrc,
  getCurrentLocale,
  navigateAppRoute,
  t,
  uploadFile,
  useI18n,
} from '@liujitcn/kratos-taro-app-core'
import { UniIcon } from '@liujitcn/kratos-taro-app-ui'
import { defAiMessageService, StreamAiMessageByChunkedRequest } from '../../../api/base/ai_message'
import { defAiSessionService } from '../../../api/base/ai_session'
import { defAiToolService } from '../../../api/base/ai_tool'
import type { AiAttachment, AiMessage, AiSession } from '../../../rpc/base/v1/ai_session'
import type { AiShortcut, AiToolCall } from '../../../rpc/base/v1/ai_tool'
import { AiMessageStatus, Terminal } from '../../../rpc/base/v1/enum'
import Composer from './components/Composer'
import SessionDrawer from './components/SessionDrawer'
import WelcomePanel from './components/WelcomePanel'
import {
  type AiStreamEvent,
  type AiStreamPayload,
  createAiEventStreamTextParser,
  parseAiEventStreamText,
  readAiEventStream,
} from './stream'
import './index.scss'

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

type AttachmentUpload = { path: string; name: string; size: number }
type StreamTask = { abort: () => void; aborted: boolean; finished: boolean; success?: boolean }
type MessageMap = Record<string, ChatMessageItem[]>

const AI_TERMINAL = Terminal.TERMINAL_APP
const LOCAL_USER_MESSAGE_PREFIX = 'ai-user-local'
const PENDING_MESSAGE_ID = 'pending'
const MAX_ATTACHMENT_COUNT = 6
const STARTER_PROMPT_PAGE_SIZE = 4

/** AI 助手页面。 */
export default function AiPage() {
  const { locale } = useI18n()
  const windowInfo = Taro.getWindowInfo() as ReturnType<typeof Taro.getWindowInfo> & {
    safeAreaInsets?: { top?: number; bottom?: number }
  }
  let safeAreaTop = windowInfo.safeArea?.top || windowInfo.statusBarHeight || windowInfo.safeAreaInsets?.top || 0
  if (process.env.TARO_ENV === 'weapp') safeAreaTop ||= 44
  // 部分微信基础库只返回 safeArea 坐标，需要用屏幕高度反推底部安全区。
  const measuredSafeAreaBottom =
    windowInfo.safeAreaInsets?.bottom ??
    (windowInfo.screenHeight && windowInfo.safeArea?.bottom
      ? Math.max(windowInfo.screenHeight - windowInfo.safeArea.bottom, 0)
      : 0)
  const safeAreaBottom = Math.max(measuredSafeAreaBottom, 9)
  const composerBottom = `${safeAreaBottom + 48}px`
  const drawerTopPadding = `${safeAreaTop + 12}px`

  const [showSessionDrawer, setShowSessionDrawer] = useState(false)
  const [activeSessionID, setActiveSessionIDState] = useState('')
  const [inputText, setInputText] = useState('')
  const [isRecording, setIsRecording] = useState(false)
  const [starterPromptGroupIndex, setStarterPromptGroupIndex] = useState(0)
  const [sessionKeyword, setSessionKeyword] = useState('')
  const [loadingSessions, setLoadingSessionsState] = useState(false)
  const [loadingShortcuts, setLoadingShortcutsState] = useState(false)
  const [loadingSessionID, setLoadingSessionIDState] = useState('')
  const [uploadingAttachment, setUploadingAttachment] = useState(false)
  const [sendingSessionMap, setSendingSessionMap] = useState<Record<string, boolean>>({})
  const [chatBottomAnchor, setChatBottomAnchor] = useState('')
  const [sessions, setSessionsState] = useState<AiSession[]>([])
  const [messages, setMessagesState] = useState<MessageMap>({})
  const [selectedAttachments, setSelectedAttachments] = useState<AiAttachment[]>([])
  const [starterShortcuts, setStarterShortcuts] = useState<AiShortcut[]>(createDefaultShortcuts)

  const activeSessionIDRef = useRef('')
  const sessionsRef = useRef<AiSession[]>([])
  const messagesRef = useRef<MessageMap>({})
  const sendingSessionMapRef = useRef<Record<string, boolean>>({})
  const loadingSessionsRef = useRef(false)
  const loadingShortcutsRef = useRef(false)
  const loadingSessionIDRef = useRef('')
  const runningStreamTaskMap = useRef(new Map<string, StreamTask>())
  const pendingDeltaMap = useRef(new Map<string, AiStreamPayload>())
  const pendingDeltaTimer = useRef<ReturnType<typeof setTimeout>>()
  const scrollTimer = useRef<ReturnType<typeof setTimeout>>()

  const filteredSessions = useMemo(() => {
    const keyword = sessionKeyword.trim()
    if (!keyword) return sessions
    return sessions.filter((item) => item.title.includes(keyword) || item.summary.includes(keyword))
  }, [sessionKeyword, sessions])
  const currentMessages = messages[activeSessionID] ?? []
  const hasMessages = currentMessages.length > 0
  const currentSessionSending = Boolean(activeSessionID && sendingSessionMap[activeSessionID])
  const starterPromptPageCount = Math.max(1, Math.ceil(starterShortcuts.length / STARTER_PROMPT_PAGE_SIZE))
  const canRefreshStarterPrompts = starterShortcuts.length > STARTER_PROMPT_PAGE_SIZE
  const starterPromptStart = (starterPromptGroupIndex % starterPromptPageCount) * STARTER_PROMPT_PAGE_SIZE
  const starterPrompts = starterShortcuts.slice(starterPromptStart, starterPromptStart + STARTER_PROMPT_PAGE_SIZE)
  const hour = new Date().getHours()
  const greetingPeriod = hour < 11
    ? t('system.ai.period.morning')
    : hour < 14
      ? t('system.ai.period.noon')
      : hour < 18
        ? t('system.ai.period.afternoon')
        : t('system.ai.period.evening')
  const aiGreetingMessage = t('system.ai.greeting', { period: greetingPeriod })
  const composerPlaceholder = isRecording
    ? t('system.ai.recording')
    : uploadingAttachment
      ? t('system.ai.attachmentUploading')
      : hasMessages
        ? t('system.ai.placeholder.continue')
        : t('system.ai.placeholder.default')
  const isSubmitDisabled =
    uploadingAttachment ||
    currentSessionSending ||
    isRecording ||
    (!inputText.trim() && selectedAttachments.length === 0)

  const setActiveSessionID = (sessionID: string) => {
    activeSessionIDRef.current = sessionID
    setActiveSessionIDState(sessionID)
  }

  const setSessions = (next: AiSession[] | ((current: AiSession[]) => AiSession[])) => {
    const resolved = typeof next === 'function' ? next(sessionsRef.current) : next
    sessionsRef.current = resolved
    setSessionsState(resolved)
  }

  const setMessages = (next: MessageMap | ((current: MessageMap) => MessageMap)) => {
    const resolved = typeof next === 'function' ? next(messagesRef.current) : next
    messagesRef.current = resolved
    setMessagesState(resolved)
  }

  const updateSessionMessages = (
    sessionID: string,
    next: ChatMessageItem[] | ((current: ChatMessageItem[]) => ChatMessageItem[]),
  ) => {
    const current = messagesRef.current[sessionID] ?? []
    const resolved = typeof next === 'function' ? next(current) : next
    setMessages({ ...messagesRef.current, [sessionID]: resolved })
  }

  const setLoadingSessions = (loading: boolean) => {
    loadingSessionsRef.current = loading
    setLoadingSessionsState(loading)
  }

  const setLoadingShortcuts = (loading: boolean) => {
    loadingShortcutsRef.current = loading
    setLoadingShortcutsState(loading)
  }

  const setLoadingSessionID = (sessionID: string) => {
    loadingSessionIDRef.current = sessionID
    setLoadingSessionIDState(sessionID)
  }

  const setSessionSending = (sessionID: string, sending: boolean) => {
    if (!sessionID) return
    const next = { ...sendingSessionMapRef.current, [sessionID]: sending }
    sendingSessionMapRef.current = next
    setSendingSessionMap(next)
  }

  const isSessionSending = (sessionID: string) => Boolean(sessionID && sendingSessionMapRef.current[sessionID])

  const scrollChatToBottom = () => {
    setChatBottomAnchor('')
    if (scrollTimer.current) clearTimeout(scrollTimer.current)
    scrollTimer.current = setTimeout(() => setChatBottomAnchor('chat-bottom'), 0)
  }

  const upsertSession = (session: AiSession) => {
    setSessions([session, ...sessionsRef.current.filter((item) => item.id !== session.id)])
  }

  const flushAiDelta = () => {
    if (pendingDeltaTimer.current) clearTimeout(pendingDeltaTimer.current)
    pendingDeltaTimer.current = undefined
    if (!pendingDeltaMap.current.size) return
    const payloadList = Array.from(pendingDeltaMap.current.values())
    pendingDeltaMap.current.clear()
    for (const payload of payloadList) {
      const sessionID = payload.session_id
      if (!sessionID || !messagesRef.current[sessionID]) continue
      updateSessionMessages(sessionID, (current) =>
        appendStreamingDelta(ensureStreamingMessage(current, payload), payload),
      )
      scrollChatToBottom()
    }
  }

  const queueAiDelta = (payload: AiStreamPayload) => {
    const { session_id: sessionID, message_id: messageID } = payload
    if (!sessionID || !messageID || !messagesRef.current[sessionID]) return
    const key = buildStreamMessageKey(sessionID, messageID)
    const cachedPayload = pendingDeltaMap.current.get(key)
    pendingDeltaMap.current.set(key, {
      ...payload,
      delta: `${cachedPayload?.delta ?? ''}${payload.delta ?? ''}`,
    })
    if (pendingDeltaTimer.current) return
    pendingDeltaTimer.current = setTimeout(flushAiDelta, 32)
  }

  const handleAiFinish = (payload: AiStreamPayload, task?: StreamTask) => {
    const sessionID = payload.session_id
    if (!sessionID) return
    if (task) task.finished = true
    flushAiDelta()
    const nextMessages = normalizeMessageList(payload.messages)
    if (task) task.success = hasSuccessfulAiMessages(nextMessages)
    const current = messagesRef.current[sessionID] ?? []
    const streamKey = payload.message_id ? buildStreamMessageKey(sessionID, payload.message_id) : ''
    const hasLocalStreamingMessages = current.some((item) => item.localOnly && item.streamKey === streamKey)
    updateSessionMessages(
      sessionID,
      nextMessages.length || !hasLocalStreamingMessages
        ? replacePendingMessages(current, nextMessages, payload)
        : current,
    )
    scrollChatToBottom()
    if (payload.session) upsertSession(normalizeSession(payload.session))
  }

  const handleAiError = (payload: AiStreamPayload, task?: StreamTask) => {
    const sessionID = payload.session_id
    if (!sessionID) return
    if (task) {
      task.finished = true
      task.success = false
    }
    flushAiDelta()
    const nextMessages = normalizeMessageList(payload.messages)
    updateSessionMessages(sessionID, (current) =>
      nextMessages.length
        ? replacePendingMessages(current, nextMessages, payload)
        : markStreamingError(ensureStreamingMessage(current, payload), payload),
    )
    scrollChatToBottom()
  }

  const handleAiStreamEvent = (event: AiStreamEvent, task?: StreamTask) => {
    if (event.event === 'delta') {
      if (event.payload.delta) queueAiDelta(event.payload)
      return
    }
    if (event.event === 'finish') {
      handleAiFinish(event.payload, task)
      return
    }
    handleAiError(event.payload, task)
  }

  const createRemoteSession = async () => {
    const response = await defAiSessionService.CreateAiSession({
      title: t('system.ai.newSession'),
      terminal: AI_TERMINAL,
    })
    const session = response.session ? normalizeSession(response.session) : undefined
    if (!session) return ''
    upsertSession(session)
    return session.id
  }

  const ensureActiveSession = async () => {
    if (activeSessionIDRef.current) return activeSessionIDRef.current
    if (sessionsRef.current.length > 0) {
      setActiveSessionID(sessionsRef.current[0].id)
      return sessionsRef.current[0].id
    }
    const sessionID = await createRemoteSession()
    if (sessionID) {
      setActiveSessionID(sessionID)
      updateSessionMessages(sessionID, [])
    }
    return sessionID
  }

  const loadMessages = async (sessionID: string) => {
    if (!sessionID) return
    setLoadingSessionID(sessionID)
    try {
      const response = await defAiSessionService.ListAiMessage({ session_id: sessionID })
      if (loadingSessionIDRef.current !== sessionID) return
      updateSessionMessages(sessionID, normalizeMessageList(response.messages))
      if (activeSessionIDRef.current === sessionID) scrollChatToBottom()
    } catch (error) {
      if (loadingSessionIDRef.current === sessionID) updateSessionMessages(sessionID, [])
      showError(error, t('system.ai.loadMessagesFailed'))
    } finally {
      if (loadingSessionIDRef.current === sessionID) setLoadingSessionID('')
    }
  }

  const ensureSessionsLoaded = async () => {
    if (loadingSessionsRef.current || sessionsRef.current.length > 0) return
    setLoadingSessions(true)
    try {
      const response = await defAiSessionService.ListAiSession({ terminal: AI_TERMINAL })
      setSessions(normalizeSessionList(response.sessions))
      const sessionID = await ensureActiveSession()
      if (sessionID) await loadMessages(sessionID)
    } catch (error) {
      showError(error, t('system.ai.loadSessionsFailed'))
    } finally {
      setLoadingSessions(false)
    }
  }

  const loadAiShortcuts = async () => {
    if (loadingShortcutsRef.current) return
    setLoadingShortcuts(true)
    try {
      const response = await defAiToolService.ListAiShortcut({ terminal: AI_TERMINAL })
      const shortcuts = normalizeStarterShortcuts(response.shortcuts).filter((item) => !item.action)
      if (shortcuts.length) {
        setStarterShortcuts(shortcuts)
        setStarterPromptGroupIndex(0)
      }
    } catch (error) {
      showError(error, t('system.ai.loadShortcutsFailed'))
    } finally {
      setLoadingShortcuts(false)
    }
  }

  const runAiTask = async (
    sessionID: string,
    payload: { text: string; attachments: AiAttachment[] },
  ) => {
    let task: StreamTask | undefined
    const request = { session_id: sessionID, content: payload.text, attachments: payload.attachments, action: undefined }
    try {
      let handledByStream = false
      if (process.env.TARO_ENV === 'weapp') {
        const parser = createAiEventStreamTextParser((event) => handleAiStreamEvent(event, task))
        const chunkedTask = StreamAiMessageByChunkedRequest(request, { onChunk: (text) => parser.push(text) })
        task = {
          aborted: false,
          finished: false,
          abort() {
            if (task) task.aborted = true
            chunkedTask.abort()
          },
        }
        runningStreamTaskMap.current.set(sessionID, task)
        handledByStream = true
        await chunkedTask.promise
        parser.flush()
        if (!task.finished && !task.aborted) throw new Error(t('system.ai.responseIncomplete'))
      }
      if (
        process.env.TARO_ENV === 'h5' &&
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
            if (task) task.aborted = true
            controller.abort()
          },
        }
        runningStreamTaskMap.current.set(sessionID, task)
        const response = await defAiMessageService.StreamAiMessage(request, { signal: controller.signal })
        if (!response.body) throw new Error(t('system.ai.streamEmpty'))
        await readAiEventStream(response.body, (event) => handleAiStreamEvent(event, task), controller.signal)
        if (!task.finished && !task.aborted) throw new Error(t('system.ai.responseIncomplete'))
        handledByStream = true
      }
      if (!handledByStream) {
        const response = await defAiMessageService.SendAiMessage(request)
        const nextMessages = normalizeNonStreamMessages(response)
        if (!nextMessages.length) throw new Error(t('system.ai.responseEmpty'))
        updateSessionMessages(sessionID, (current) => replacePendingMessages(current, nextMessages))
        scrollChatToBottom()
        if (response.session) upsertSession(normalizeSession(response.session))
        return hasSuccessfulAiMessages(nextMessages)
      }
      return Boolean(task?.success)
    } catch (error) {
      if (task?.aborted) return false
      updateSessionMessages(sessionID, markThinkingMessageFailed)
      scrollChatToBottom()
      showError(error, t('system.ai.requestFailed'))
      return false
    } finally {
      if (task && runningStreamTaskMap.current.get(sessionID) === task) {
        runningStreamTaskMap.current.delete(sessionID)
      }
      setSessionSending(sessionID, false)
    }
  }

  const sendAiPayload = async (payload: { text: string; attachments: AiAttachment[] }) => {
    const sessionID = await ensureActiveSession()
    if (!sessionID || isSessionSending(sessionID)) return false
    updateSessionMessages(sessionID, (current) =>
      sortMessages([...current, createLocalUserMessage(payload), createThinkingMessage({ sessionID })]),
    )
    scrollChatToBottom()
    setSessionSending(sessionID, true)
    return runAiTask(sessionID, payload)
  }

  useLoad(() => {
    void loadAiShortcuts()
    void ensureSessionsLoaded()
  })

  useEffect(() => {
    void Taro.setNavigationBarTitle({ title: t('system.ai.chatTitle') })
    setStarterShortcuts(createDefaultShortcuts())
    setStarterPromptGroupIndex(0)
  }, [locale])

  useEffect(() => {
    const tasks = runningStreamTaskMap.current
    const deltaMap = pendingDeltaMap.current
    return () => {
      tasks.forEach((task) => {
        task.finished = true
        task.abort()
      })
      tasks.clear()
      deltaMap.clear()
      if (pendingDeltaTimer.current) clearTimeout(pendingDeltaTimer.current)
      if (scrollTimer.current) clearTimeout(scrollTimer.current)
      activeSessionIDRef.current = ''
    }
  }, [])

  const selectSession = (sessionID: string) => {
    setActiveSessionID(sessionID)
    setShowSessionDrawer(false)
    if (!messagesRef.current[sessionID]?.length || !isSessionSending(sessionID)) {
      void loadMessages(sessionID)
      return
    }
    scrollChatToBottom()
  }

  const createSession = async () => {
    try {
      const sessionID = await createRemoteSession()
      if (!sessionID) return
      setActiveSessionID(sessionID)
      updateSessionMessages(sessionID, [])
      setSessionKeyword('')
      setShowSessionDrawer(false)
    } catch (error) {
      showError(error, t('system.ai.createSessionFailed'))
    }
  }

  const deleteSession = async (sessionID: string) => {
    const session = sessionsRef.current.find((item) => item.id === sessionID)
    const result = await Taro.showModal({
      title: t('system.ai.deleteSession'),
      content: t('system.ai.deleteSessionConfirm', {
        title: session?.title || t('system.ai.currentSession'),
      }),
      confirmText: t('common.action.delete'),
      confirmColor: '#cf4444',
    })
    if (!result.confirm) return
    try {
      await defAiSessionService.DeleteAiSession({ id: sessionID })
      setSessions((current) => current.filter((item) => item.id !== sessionID))
      const nextMessages = { ...messagesRef.current }
      delete nextMessages[sessionID]
      setMessages(nextMessages)
      if (activeSessionIDRef.current === sessionID) {
        setActiveSessionID('')
        await ensureActiveSession()
      }
    } catch (error) {
      showError(error, t('system.ai.deleteSessionFailed'))
    }
  }

  const handleSessionAction = async (session: AiSession) => {
    try {
      const result = await Taro.showActionSheet({ itemList: [t('system.ai.deleteSession')] })
      if (result.tapIndex === 0) await deleteSession(session.id)
    } catch {
      // 用户取消操作无需提示。
    }
  }

  const copyMessage = async (item: ChatMessageItem) => {
    await Taro.setClipboardData({ data: item.content })
    await Taro.showToast({ icon: 'none', title: t('system.ai.copySuccess') })
  }

  const deleteMessage = async (item: ChatMessageItem) => {
    const sessionID = activeSessionIDRef.current
    if (!sessionID) return
    try {
      if (!item.localOnly) {
        await defAiMessageService.DeleteAiMessage({ session_id: sessionID, message_id: item.messageID })
      }
      updateSessionMessages(sessionID, (current) => current.filter((message) => message.messageID !== item.messageID))
    } catch (error) {
      showError(error, t('system.ai.deleteMessageFailed'))
    }
  }

  const regenerateMessage = async (item: ChatMessageItem) => {
    const sessionID = activeSessionIDRef.current
    if (item.role !== 'ai' || item.localOnly || isSessionSending(sessionID)) return
    setSessionSending(sessionID, true)
    try {
      const response = await defAiMessageService.RegenerateAiMessage({ session_id: sessionID, message_id: item.messageID })
      updateSessionMessages(sessionID, normalizeMessageList(response.messages))
      if (response.session) upsertSession(normalizeSession(response.session))
    } catch (error) {
      showError(error, t('system.ai.regenerateFailed'))
    } finally {
      setSessionSending(sessionID, false)
    }
  }

  const handleMessageAction = async (item: ChatMessageItem) => {
    try {
      const itemList = item.role === 'ai'
        ? [t('system.ai.action.copy'), t('common.action.delete'), t('system.ai.action.regenerate')]
        : [t('system.ai.action.copy'), t('common.action.delete')]
      const { tapIndex } = await Taro.showActionSheet({ itemList })
      if (tapIndex === 0) await copyMessage(item)
      else if (tapIndex === 1) await deleteMessage(item)
      else await regenerateMessage(item)
    } catch {
      // 用户取消操作无需提示。
    }
  }

  const navigateBack = () => {
    if (getCurrentPages().length > 1) {
      void Taro.navigateBack()
      return
    }
    navigateAppRoute('app/home')
  }

  const handleSend = async () => {
    if (isSubmitDisabled) return
    const text = inputText.trim() || t('system.ai.attachmentAnswer')
    const attachments = [...selectedAttachments]
    setInputText('')
    setSelectedAttachments([])
    await sendAiPayload({ text, attachments })
  }

  const handleAttachment = async () => {
    if (uploadingAttachment || currentSessionSending) return
    if (selectedAttachments.length >= MAX_ATTACHMENT_COUNT) {
      await Taro.showToast({
        icon: 'none',
        title: t('system.ai.attachmentLimit', { count: MAX_ATTACHMENT_COUNT }),
      })
      return
    }
    try {
      const result = await Taro.chooseImage({
        count: MAX_ATTACHMENT_COUNT - selectedAttachments.length,
        sourceType: ['album', 'camera'],
      })
      const files: AttachmentUpload[] = result.tempFilePaths.map((path, index) => ({
        path,
        name:
          result.tempFiles[index]?.originalFileObj?.name ||
          t('system.ai.imageName', { index: index + 1 }),
        size: Number(result.tempFiles[index]?.size || 0),
      }))
      setUploadingAttachment(true)
      const uploaded = await Promise.all(files.map((file) => uploadFile('ai', file.path)))
      const attachments = uploaded.map<AiAttachment>((file, index) => ({
        id: file.url || `${file.name}-${index}`,
        name: files[index]?.name || file.name,
        size: files[index]?.size || 0,
        url: file.url,
        mime_type: 'image/*',
      }))
      setSelectedAttachments((current) => [...current, ...attachments].slice(0, MAX_ATTACHMENT_COUNT))
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error)
      if (!/cancel/i.test(message)) showError(error, t('system.ai.uploadFailed'))
    } finally {
      setUploadingAttachment(false)
    }
  }

  const previewAttachment = (attachment: AiAttachment, attachments: AiAttachment[]) => {
    const current = formatSrc(attachment.url)
    const urls = attachments.map((item) => formatSrc(item.url)).filter(Boolean)
    if (current) void Taro.previewImage({ current, urls: urls.length ? urls : [current] })
  }

  return (
    <View className='ai-page'>
      <View className={`ai-navbar${process.env.TARO_ENV === 'weapp' ? ' is-weapp' : ''}`}>
        <Button className='nav-back-button' hoverClass='none' onClick={navigateBack}>
          <UniIcon type='left' size={24} color='#111' />
        </Button>
        <View className='ai-navbar__title'>{t('system.ai.chatTitle')}</View>
        <Button className='nav-menu-button' hoverClass='none' onClick={() => setShowSessionDrawer((open) => !open)}>
          <UniIcon type='bars' size={24} color='#111' />
        </Button>
      </View>
      <ScrollView
        className='ai-body'
        scrollY
        scrollWithAnimation
        scrollIntoView={chatBottomAnchor}
        showScrollbar={false}
      >
        {!hasMessages ? (
          <WelcomePanel
            greetingMessage={aiGreetingMessage}
            loading={loadingSessions || loadingShortcuts}
            shortcuts={starterPrompts}
            canRefresh={canRefreshStarterPrompts}
            onRefresh={() => setStarterPromptGroupIndex((index) => (index + 1) % starterPromptPageCount)}
            onShortcutTap={(shortcut) => {
              if (!currentSessionSending && !loadingSessions) {
                void sendAiPayload({ text: shortcut.prompt || shortcut.title, attachments: [] })
              }
            }}
          />
        ) : (
          <View className='chat-list'>
            {currentMessages.map((item) => (
              <View
                id={item.key}
                key={item.key}
                className={`message-row ${item.role === 'user' ? 'is-user' : 'is-ai'}`}
              >
                <View
                  className={`bubble ${item.role === 'ai' ? 'ai-bubble' : 'user-bubble'}${item.status === AiMessageStatus.GENERATING_AMS ? ' is-streaming' : ''}`}
                  onLongPress={() => void handleMessageAction(item)}
                >
                  {item.role === 'ai' && item.model ? (
                    <View className='reply-meta'><Text className='reply-tag'>{t('system.ai.modelReply')}</Text><Text className='reply-model'>{item.model}</Text></View>
                  ) : null}
                  <View className='bubble-content'>{item.content}</View>
                  {item.attachments.length ? (
                    <View className='attachment-list'>
                      {item.attachments.map((attachment) => (
                        <View
                          key={attachment.id || attachment.url || attachment.name}
                          className='attachment-card'
                          onClick={() => previewAttachment(attachment, item.attachments)}
                        >
                          <View className='attachment-icon'>{isImageAttachment(attachment) ? t('system.ai.imageAttachment') : t('system.ai.attachment')}</View>
                          <View className='attachment-info'>
                            <View className='attachment-name'>{attachment.name}</View>
                            <View className='attachment-meta'>{formatAttachmentMeta(attachment)}</View>
                          </View>
                        </View>
                      ))}
                    </View>
                  ) : null}
                  {item.tools.length ? <View className='tool-row'>{t('system.ai.toolCalled', { tools: formatTools(item.tools) })}</View> : null}
                </View>
              </View>
            ))}
            <View id='chat-bottom' className='chat-bottom' />
          </View>
        )}
        {loadingSessionID ? <View className='loading-session'>{t('system.ai.loadMessages')}</View> : null}
      </ScrollView>
      <Composer
        value={inputText}
        attachments={selectedAttachments}
        placeholder={composerPlaceholder}
        bottom={composerBottom}
        recording={isRecording}
        sending={currentSessionSending}
        disabled={isSubmitDisabled}
        onChange={setInputText}
        onAttach={() => void handleAttachment()}
        onRecord={() => {
          setIsRecording((recording) => {
            void Taro.showToast({
              icon: 'none',
              title: !recording ? t('system.ai.recognizing') : t('system.ai.speechStopped'),
            })
            return !recording
          })
        }}
        onSend={() => void handleSend()}
        onRemoveAttachment={(attachment) => setSelectedAttachments((current) => current.filter((item) => item !== attachment))}
      />
      <SessionDrawer
        open={showSessionDrawer}
        topPadding={drawerTopPadding}
        keyword={sessionKeyword}
        loading={loadingSessions}
        sessions={filteredSessions}
        activeSessionId={activeSessionID}
        onClose={() => setShowSessionDrawer(false)}
        onCreate={() => void createSession()}
        onSelect={selectSession}
        onAction={(session) => void handleSessionAction(session)}
        onKeywordChange={setSessionKeyword}
      />
    </View>
  )
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
  return Array.isArray(list) ? list.map(normalizeSession).filter((item) => item.id) : []
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
  if (!Array.isArray(list)) return []
  return sortMessages(list.filter(Boolean).flatMap((item) => [mapMessageItem(item, 'user'), mapMessageItem(item, 'ai')]))
}

function hasSuccessfulAiMessages(list: ChatMessageItem[]) {
  return list.some((item) => item.status === AiMessageStatus.SUCCESS_AMS)
}

function mapMessageItem(message: AiMessage, role: ChatRole): ChatMessageItem {
  const inputContent = { kind: message.input_content?.kind || 'text', content: message.input_content?.content ?? '' }
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
  const message = mapMessageItem({
    id: `${LOCAL_USER_MESSAGE_PREFIX}-${now}`,
    input_content: { kind: 'text', content: payload.text },
    output_content: undefined,
    attachments: payload.attachments,
    created_at: { seconds: Math.floor(now / 1000), nanos: (now % 1000) * 1_000_000 },
    status: AiMessageStatus.GENERATING_AMS,
    token: { input: 0, output: 0, cache: 0, total: 0 },
    tools: [],
    first_token_ms: 0,
    duration_ms: 0,
  }, 'user')
  message.localOnly = true
  message.status = AiMessageStatus.GENERATING_AMS
  return message
}

function createThinkingMessage(options?: { sessionID?: string; messageID?: string }) {
  const now = Date.now()
  const streamKey = options?.sessionID
    ? buildStreamMessageKey(options.sessionID, options.messageID || PENDING_MESSAGE_ID)
    : undefined
  const message = mapMessageItem({
    id: streamKey || `ai-thinking-${now}`,
    input_content: undefined,
    output_content: {
      kind: 'text', content: t('system.ai.thinking'), reply_source: '', model: '', fallback: false,
      fallback_reason: '', flow: '', step: '', blocks_json: '',
    },
    attachments: [],
    created_at: { seconds: Math.floor(now / 1000), nanos: (now % 1000) * 1_000_000 },
    status: AiMessageStatus.GENERATING_AMS,
    token: { input: 0, output: 0, cache: 0, total: 0 },
    tools: [],
    first_token_ms: 0,
    duration_ms: 0,
  }, 'ai')
  message.localOnly = true
  message.streamKey = streamKey
  return message
}

function ensureStreamingMessage(current: ChatMessageItem[], payload: AiStreamPayload) {
  const sessionID = payload.session_id
  const messageID = payload.message_id
  if (!sessionID || !messageID) return current
  const streamKey = buildStreamMessageKey(sessionID, messageID)
  if (current.some((item) => item.streamKey === streamKey)) return current
  const pendingStreamKey = buildPendingStreamMessageKey(sessionID)
  const next = current.map((item) =>
    item.streamKey === pendingStreamKey
      ? { ...item, id: messageID, messageID, key: `${messageID}:ai`, streamKey }
      : item,
  )
  return next.some((item) => item.streamKey === streamKey)
    ? next
    : sortMessages([...next, createThinkingMessage({ sessionID, messageID })])
}

function appendStreamingDelta(current: ChatMessageItem[], payload: AiStreamPayload) {
  if (!payload.delta) return current
  const streamKey = buildStreamMessageKey(payload.session_id, payload.message_id)
  return current.map((item) => {
    if (item.streamKey !== streamKey || item.role === 'user') return item
    const baseContent = item.content === t('system.ai.thinking') ? '' : item.content
    return { ...item, content: `${baseContent}${payload.delta}`, status: AiMessageStatus.GENERATING_AMS }
  })
}

function markThinkingMessageFailed(current: ChatMessageItem[]) {
  return current.map((item) =>
    item.localOnly
      ? {
          ...item,
          status: AiMessageStatus.FAILED_AMS,
          content: item.role === 'ai' ? t('system.ai.failedResponse') : item.content,
        }
      : item,
  )
}

function markStreamingError(current: ChatMessageItem[], payload: AiStreamPayload) {
  const streamKey = buildStreamMessageKey(payload.session_id, payload.message_id)
  return current.map((item) =>
    item.localOnly && item.streamKey === streamKey
      ? { ...item, status: AiMessageStatus.FAILED_AMS, content: t('system.ai.failedResponse') }
      : item,
  )
}

function replacePendingMessages(current: ChatMessageItem[], nextMessages: ChatMessageItem[], payload?: AiStreamPayload) {
  const sessionID = payload?.session_id ?? ''
  const streamKey = payload?.message_id ? buildStreamMessageKey(sessionID, payload.message_id) : ''
  const pendingStreamKey = sessionID ? buildPendingStreamMessageKey(sessionID) : ''
  const stableMessages = current.filter((item) => {
    if (!item.localOnly) return true
    if (payload?.message_id && item.role === 'user') {
      return !nextMessages.some((message) => message.role === 'user' && message.messageID === payload.message_id)
    }
    return streamKey ? item.streamKey !== streamKey && item.streamKey !== pendingStreamKey : false
  })
  const messageMap = new Map<string, ChatMessageItem>()
  for (const item of stableMessages) messageMap.set(item.key, item)
  for (const item of nextMessages) messageMap.set(item.key, item)
  return sortMessages(Array.from(messageMap.values()))
}

function normalizeNonStreamMessages(response: unknown) {
  const jsonResponse = response as { messages?: AiMessage[] }
  if (Array.isArray(jsonResponse?.messages)) return normalizeMessageList(jsonResponse.messages)
  const events = parseAiEventStreamText(response)
  const finishEvent = [...events].reverse().find((item) => item.event === 'finish')
  if (finishEvent) return normalizeMessageList(finishEvent.payload.messages)
  if ([...events].reverse().some((item) => item.event === 'error')) {
    throw new Error(t('system.ai.requestFailed'))
  }
  return []
}

function sortMessages(list: ChatMessageItem[]) {
  return [...list].sort((left, right) => {
    const difference = resolveTimestamp(left.created_at) - resolveTimestamp(right.created_at)
    if (difference) return difference
    if (left.role !== right.role) return left.role === 'user' ? -1 : 1
    return left.messageID.localeCompare(right.messageID, getCurrentLocale(), { numeric: true })
  })
}

function buildStreamMessageKey(sessionID: string, messageID: string) {
  return `${sessionID}:${messageID}`
}

function buildPendingStreamMessageKey(sessionID: string) {
  return buildStreamMessageKey(sessionID, PENDING_MESSAGE_ID)
}

function resolveTimestamp(timestamp: AiMessage['created_at'] | AiSession['updated_at']) {
  return Number(timestamp?.seconds || 0) * 1000 + Math.floor(Number(timestamp?.nanos || 0) / 1_000_000)
}

function isImageAttachment(attachment: AiAttachment) {
  return attachment.mime_type.startsWith('image/')
}

function formatAttachmentMeta(attachment: AiAttachment) {
  return attachment.size
    ? `${Math.max(1, Math.round(attachment.size / 1024))} KB`
    : t('system.ai.attachment')
}

function formatTools(tools: AiToolCall[]) {
  return tools.map((item) => item.title || item.name).filter(Boolean).join(' · ')
}

function showError(error: unknown, fallback: string) {
  const message = error instanceof Error ? error.message : fallback
  void Taro.showToast({ icon: 'none', title: message || fallback })
}

function createDefaultShortcuts(): AiShortcut[] {
  return [
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
}
