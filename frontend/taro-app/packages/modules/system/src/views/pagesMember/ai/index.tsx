import { Button, ScrollView, Text, View } from '@tarojs/components'
import Taro, { getCurrentPages, useLoad } from '@tarojs/taro'
import { useEffect, useMemo, useRef, useState } from 'react'
import {
  formatSrc,
  navigateAppRoute,
  uploadFile,
} from '@liujitcn/kratos-taro-app-core'
import { ArrowLeft, Category } from '@liujitcn/kratos-taro-app-ui'
import { defAiMessageService, StreamAiMessageByChunkedRequest } from '../../../api/base/ai_message'
import { defAiSessionService } from '../../../api/base/ai_session'
import { defAiToolService } from '../../../api/base/ai_tool'
import type { AiAttachment, AiMessage, AiSession } from '../../../rpc/base/v1/ai_session'
import type { AiShortcut, AiToolCall } from '../../../rpc/base/v1/ai_tool'
import { AiMessageStatus, Terminal } from '../../../rpc/common/v1/enum'
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
const THINKING_MESSAGE_CONTENT = '正在回复'
const LOCAL_USER_MESSAGE_PREFIX = 'ai-user-local'
const PENDING_MESSAGE_ID = 'pending'
const MAX_ATTACHMENT_COUNT = 6
const STARTER_PROMPT_PAGE_SIZE = 4

const DEFAULT_SHORTCUTS: AiShortcut[] = [
  { key: 'summarize', title: '帮我总结一段内容', prompt: '请帮我总结以下内容', action: undefined, required_tools: [], sort: 1, group: '文本助手' },
  { key: 'rewrite', title: '帮我优化一段文字', prompt: '请帮我优化以下文字', action: undefined, required_tools: [], sort: 2, group: '文本助手' },
  { key: 'plan', title: '帮我制定一个计划', prompt: '请帮我制定一个清晰的执行计划', action: undefined, required_tools: [], sort: 3, group: '效率助手' },
  { key: 'ideas', title: '给我一些灵感', prompt: '请围绕这个主题给我一些新想法', action: undefined, required_tools: [], sort: 4, group: '效率助手' },
]

/** AI 助手页面。 */
export default function AiPage() {
  const windowInfo = Taro.getWindowInfo() as ReturnType<typeof Taro.getWindowInfo> & {
    safeAreaInsets?: { top?: number; bottom?: number }
  }
  let safeAreaTop = windowInfo.safeArea?.top || windowInfo.statusBarHeight || windowInfo.safeAreaInsets?.top || 0
  if (process.env.TARO_ENV === 'weapp') safeAreaTop ||= 44
  const safeAreaBottom = Math.max(windowInfo.safeAreaInsets?.bottom || 0, 9)
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
  const [starterShortcuts, setStarterShortcuts] = useState<AiShortcut[]>(DEFAULT_SHORTCUTS)

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
  const greetingPeriod = hour < 11 ? '上午' : hour < 14 ? '中午' : hour < 18 ? '下午' : '晚上'
  const aiGreetingMessage = `您好，${greetingPeriod}好！今天有什么需要我协助的吗？`
  const composerPlaceholder = isRecording
    ? '正在听...'
    : uploadingAttachment
      ? '附件上传中...'
      : hasMessages
        ? '继续输入问题'
        : '输入你想了解的内容'
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
    const response = await defAiSessionService.CreateAiSession({ title: '新会话', terminal: AI_TERMINAL })
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
      showError(error, '加载消息失败')
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
      showError(error, '加载会话失败')
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
      showError(error, '加载快捷助手失败')
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
        if (!task.finished && !task.aborted) throw new Error('AI 助手流式响应未完整返回')
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
        if (!response.body) throw new Error('AI 助手流式响应为空')
        await readAiEventStream(response.body, (event) => handleAiStreamEvent(event, task), controller.signal)
        if (!task.finished && !task.aborted) throw new Error('AI 助手流式响应未完整返回')
        handledByStream = true
      }
      if (!handledByStream) {
        const response = await defAiMessageService.SendAiMessage(request)
        const nextMessages = normalizeNonStreamMessages(response)
        if (!nextMessages.length) throw new Error('AI 助手响应为空')
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
      showError(error, 'AI 助手请求失败')
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
      showError(error, '创建会话失败')
    }
  }

  const deleteSession = async (sessionID: string) => {
    const session = sessionsRef.current.find((item) => item.id === sessionID)
    const result = await Taro.showModal({
      title: '删除会话',
      content: `是否删除「${session?.title || '当前会话'}」？`,
      confirmText: '删除',
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
      showError(error, '删除会话失败')
    }
  }

  const handleSessionAction = async (session: AiSession) => {
    try {
      const result = await Taro.showActionSheet({ itemList: ['删除会话'] })
      if (result.tapIndex === 0) await deleteSession(session.id)
    } catch {
      // 用户取消操作无需提示。
    }
  }

  const copyMessage = async (item: ChatMessageItem) => {
    await Taro.setClipboardData({ data: item.content })
    await Taro.showToast({ icon: 'none', title: '消息已复制' })
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
      showError(error, '删除消息失败')
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
      showError(error, '重新生成失败')
    } finally {
      setSessionSending(sessionID, false)
    }
  }

  const handleMessageAction = async (item: ChatMessageItem) => {
    try {
      const itemList = item.role === 'ai' ? ['复制', '删除', '重新生成'] : ['复制', '删除']
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
    const text = inputText.trim() || '请结合附件内容回答我的问题'
    const attachments = [...selectedAttachments]
    setInputText('')
    setSelectedAttachments([])
    await sendAiPayload({ text, attachments })
  }

  const handleAttachment = async () => {
    if (uploadingAttachment || currentSessionSending) return
    if (selectedAttachments.length >= MAX_ATTACHMENT_COUNT) {
      await Taro.showToast({ icon: 'none', title: `最多上传 ${MAX_ATTACHMENT_COUNT} 个附件` })
      return
    }
    try {
      const result = await Taro.chooseImage({
        count: MAX_ATTACHMENT_COUNT - selectedAttachments.length,
        sourceType: ['album', 'camera'],
      })
      const files: AttachmentUpload[] = result.tempFilePaths.map((path, index) => ({
        path,
        name: result.tempFiles[index]?.originalFileObj?.name || `图片${index + 1}`,
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
      if (!/cancel/i.test(message)) showError(error, '附件上传失败')
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
          <ArrowLeft size={24} color='#111' />
        </Button>
        <View className='ai-navbar__title'>AI 助手</View>
        <Button className='nav-menu-button' hoverClass='none' onClick={() => setShowSessionDrawer((open) => !open)}>
          <Category size={24} color='#111' />
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
                  className={`bubble ${item.role === 'ai' ? 'ai-bubble' : 'user-bubble'}${item.status === AiMessageStatus.GENERATING_AAMS ? ' is-streaming' : ''}`}
                  onLongPress={() => void handleMessageAction(item)}
                >
                  {item.role === 'ai' && item.model ? (
                    <View className='reply-meta'><Text className='reply-tag'>模型回复</Text><Text className='reply-model'>{item.model}</Text></View>
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
                          <View className='attachment-icon'>{isImageAttachment(attachment) ? '图' : '件'}</View>
                          <View className='attachment-info'>
                            <View className='attachment-name'>{attachment.name}</View>
                            <View className='attachment-meta'>{formatAttachmentMeta(attachment)}</View>
                          </View>
                        </View>
                      ))}
                    </View>
                  ) : null}
                  {item.tools.length ? <View className='tool-row'>已调用：{formatTools(item.tools)}</View> : null}
                </View>
              </View>
            ))}
            <View id='chat-bottom' className='chat-bottom' />
          </View>
        )}
        {loadingSessionID ? <View className='loading-session'>正在加载消息...</View> : null}
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
            void Taro.showToast({ icon: 'none', title: !recording ? '正在识别语音' : '已停止语音输入' })
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
    title: String(session?.title ?? '新会话'),
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
  return list.some((item) => item.status === AiMessageStatus.SUCCESS_AAMS)
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
  const status = Number(message.status ?? AiMessageStatus.SUCCESS_AAMS)
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
    status: AiMessageStatus.GENERATING_AAMS,
    token: { input: 0, output: 0, cache: 0, total: 0 },
    tools: [],
    first_token_ms: 0,
    duration_ms: 0,
  }, 'user')
  message.localOnly = true
  message.status = AiMessageStatus.GENERATING_AAMS
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
      kind: 'text', content: THINKING_MESSAGE_CONTENT, reply_source: '', model: '', fallback: false,
      fallback_reason: '', flow: '', step: '', blocks_json: '',
    },
    attachments: [],
    created_at: { seconds: Math.floor(now / 1000), nanos: (now % 1000) * 1_000_000 },
    status: AiMessageStatus.GENERATING_AAMS,
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
    const baseContent = item.content === THINKING_MESSAGE_CONTENT ? '' : item.content
    return { ...item, content: `${baseContent}${payload.delta}`, status: AiMessageStatus.GENERATING_AAMS }
  })
}

function markThinkingMessageFailed(current: ChatMessageItem[]) {
  return current.map((item) =>
    item.localOnly
      ? {
          ...item,
          status: AiMessageStatus.FAILED_AAMS,
          content: item.role === 'ai' ? '这次回复没有成功返回，你可以直接重试刚才的问题。' : item.content,
        }
      : item,
  )
}

function markStreamingError(current: ChatMessageItem[], payload: AiStreamPayload) {
  const streamKey = buildStreamMessageKey(payload.session_id, payload.message_id)
  return current.map((item) =>
    item.localOnly && item.streamKey === streamKey
      ? { ...item, status: AiMessageStatus.FAILED_AAMS, content: '这次回复没有成功返回，你可以直接重试刚才的问题。' }
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
  if ([...events].reverse().some((item) => item.event === 'error')) throw new Error('AI 助手请求失败')
  return []
}

function sortMessages(list: ChatMessageItem[]) {
  return [...list].sort((left, right) => {
    const difference = resolveTimestamp(left.created_at) - resolveTimestamp(right.created_at)
    if (difference) return difference
    if (left.role !== right.role) return left.role === 'user' ? -1 : 1
    return left.messageID.localeCompare(right.messageID, 'zh-Hans-CN', { numeric: true })
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
  return attachment.size ? `${Math.max(1, Math.round(attachment.size / 1024))} KB` : '附件'
}

function formatTools(tools: AiToolCall[]) {
  return tools.map((item) => item.title || item.name).filter(Boolean).join(' · ')
}

function showError(error: unknown, fallback: string) {
  const message = error instanceof Error ? error.message : fallback
  void Taro.showToast({ icon: 'none', title: message || fallback })
}
