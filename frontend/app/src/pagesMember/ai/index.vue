<script setup lang="ts">
import { onLoad } from '@dcloudio/uni-app'
import { computed, nextTick, onBeforeUnmount, ref } from 'vue'
import { defAiMessageService } from '@/api/base/ai_message'
import { defAiSessionService } from '@/api/base/ai_session'
import type { AiMessage } from '@/rpc/base/v1/ai_session'
import type { AiAttachment, AiSession } from '@/rpc/base/v1/ai_session'
import type { AiShortcut, AiToolCall } from '@/rpc/base/v1/ai_tool'
import { AiMessageStatus, Terminal } from '@/rpc/common/v1/enum'
import { uploadFile } from '@/utils/file'
import { formatSrc } from '@/utils/index'
import Composer from './components/Composer.vue'
import SessionDrawer from './components/SessionDrawer.vue'
import WelcomePanel from './components/WelcomePanel.vue'

type ChatRole = 'user' | 'ai'

type ChatMessageItem = AiMessage & {
  key: string
  messageID: string
  role: ChatRole
  content: string
  tools: AiToolCall[]
  model: string
  replySource: string
  fallback: boolean
  fallbackReason: string
  tokenTotal: number
  firstTokenMs: number
  durationMs: number
  localOnly?: boolean
}

type AttachmentUpload = {
  path: string
  name: string
  size: number
}

const AI_TERMINAL = Terminal.TERMINAL_APP
const THINKING_MESSAGE_CONTENT = '正在回复'
const LOCAL_USER_MESSAGE_PREFIX = 'ai-user-local'
const MAX_ATTACHMENT_COUNT = 6
const STARTER_PROMPT_PAGE_SIZE = 4

const systemInfo = uni.getSystemInfoSync()
const safeAreaTop = systemInfo.safeAreaInsets?.top || systemInfo.statusBarHeight || 0
const safeAreaBottom = Math.max(systemInfo.safeAreaInsets?.bottom || 0, 9)
const windowHeight = systemInfo.windowHeight || systemInfo.screenHeight || 667
const composerBottom = `${safeAreaBottom}px`
const navHeight = `${safeAreaTop + 44}px`
const drawerTopPadding = `${safeAreaTop + 12}px`

const showSessionDrawer = ref(false)
const activeSessionID = ref('')
const inputText = ref('')
const isRecording = ref(false)
const starterPromptGroupIndex = ref(0)
const sessionKeyword = ref('')
const loadingSessions = ref(false)
const loadingSessionID = ref('')
const uploadingAttachment = ref(false)
const sendingSessionMap = ref<Record<string, boolean>>({})
const chatBottomAnchor = ref('')
const sessions = ref<AiSession[]>([])
const messages = ref<Record<string, ChatMessageItem[]>>({})
const selectedAttachments = ref<AiAttachment[]>([])

const starterShortcuts = ref<AiShortcut[]>([
  {
    key: 'summarize',
    title: '帮我总结一段内容',
    prompt: '请帮我总结以下内容',
    action: undefined,
    required_tools: [],
    sort: 1,
    group: '文本助手',
  },
  {
    key: 'rewrite',
    title: '帮我优化一段文字',
    prompt: '请帮我优化以下文字',
    action: undefined,
    required_tools: [],
    sort: 2,
    group: '文本助手',
  },
  {
    key: 'plan',
    title: '帮我制定一个计划',
    prompt: '请帮我制定一个清晰的执行计划',
    action: undefined,
    required_tools: [],
    sort: 3,
    group: '效率助手',
  },
  {
    key: 'ideas',
    title: '给我一些灵感',
    prompt: '请围绕这个主题给我一些新想法',
    action: undefined,
    required_tools: [],
    sort: 4,
    group: '效率助手',
  },
])

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
    return '上午'
  }
  if (hour < 14) {
    return '中午'
  }
  if (hour < 18) {
    return '下午'
  }
  return '晚上'
})
const aiGreetingMessage = computed(
  () => `您好，${aiGreetingPeriod.value}好！今天有什么需要我协助的吗？`,
)
const composerPlaceholder = computed(() => {
  if (isRecording.value) {
    return '正在听...'
  }
  if (uploadingAttachment.value) {
    return '附件上传中...'
  }
  return hasMessages.value ? '继续输入问题' : '输入你想了解的内容'
})
const isSubmitDisabled = computed(
  () =>
    uploadingAttachment.value ||
    currentSessionSending.value ||
    isRecording.value ||
    (!inputText.value.trim() && selectedAttachments.value.length === 0),
)

onLoad(() => {
  void ensureSessionsLoaded()
})

onBeforeUnmount(() => {
  activeSessionID.value = ''
})

const toggleSessionDrawer = () => {
  showSessionDrawer.value = !showSessionDrawer.value
}

const selectSession = (sessionID: string) => {
  activeSessionID.value = sessionID
  showSessionDrawer.value = false
  if (!messages.value[sessionID]) {
    void loadMessages(sessionID)
  } else {
    scrollChatToBottom()
  }
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
    showError(error, '创建会话失败')
  }
}

const deleteSession = async (sessionID: string) => {
  const session = sessions.value.find((item) => item.id === sessionID)
  const result = await uni.showModal({
    title: '删除会话',
    content: `是否删除「${session?.title || '当前会话'}」？`,
    confirmText: '删除',
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
    showError(error, '删除会话失败')
  }
}

const handleSessionAction = (session: AiSession) => {
  uni.showActionSheet({
    itemList: ['删除会话'],
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
    success: () => uni.showToast({ icon: 'none', title: '消息已复制' }),
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
    showError(error, '删除消息失败')
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
    showError(error, '重新生成失败')
  } finally {
    setSessionSending(activeSessionID.value, false)
  }
}

const handleMessageAction = (item: ChatMessageItem) => {
  const itemList = item.role === 'ai' ? ['复制', '删除', '重新生成'] : ['复制', '删除']
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
  uni.switchTab({ url: '/pages/index/index' })
}

const handleSend = async () => {
  if (isSubmitDisabled.value) {
    return
  }
  const text = inputText.value.trim() || '请结合附件内容回答我的问题'
  inputText.value = ''
  const attachments = [...selectedAttachments.value]
  selectedAttachments.value = []
  await sendAiPayload(text, attachments)
}

const handleStarterPrompt = async (shortcut: AiShortcut) => {
  if (currentSessionSending.value || loadingSessions.value) {
    return
  }
  await sendAiPayload(shortcut.prompt || shortcut.title, [])
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
    title: isRecording.value ? '正在识别语音' : '已停止语音输入',
  })
}

const handleAttachment = () => {
  if (uploadingAttachment.value || currentSessionSending.value) {
    return
  }
  if (selectedAttachments.value.length >= MAX_ATTACHMENT_COUNT) {
    uni.showToast({ icon: 'none', title: `最多上传 ${MAX_ATTACHMENT_COUNT} 个附件` })
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
        name: (tempFiles[index] as { name?: string } | undefined)?.name || `图片${index + 1}`,
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
        showError(error, '附件上传失败')
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

async function sendAiPayload(text: string, attachments: AiAttachment[]) {
  const sessionID = await ensureActiveSession()
  if (!sessionID) {
    return
  }
  setSessionSending(sessionID, true)

  const userMessageID = `${LOCAL_USER_MESSAGE_PREFIX}-${Date.now()}`
  const pendingMessageID = `pending-${Date.now()}`
  const current = messages.value[sessionID] ?? []
  const userMessage = createLocalMessage(userMessageID, 'user', text, attachments)
  const pendingMessage = createLocalMessage(
    pendingMessageID,
    'ai',
    THINKING_MESSAGE_CONTENT,
    [],
    AiMessageStatus.GENERATING_AAMS,
  )
  messages.value[sessionID] = [...current, userMessage, pendingMessage]
  scrollChatToBottom()

  try {
    const response = await defAiMessageService.SendAiMessage({
      session_id: sessionID,
      content: text,
      attachments,
      action: undefined,
    })
    messages.value[sessionID] = normalizeMessageList(response.messages)
    if (response.session) {
      upsertSession(normalizeSession(response.session))
    }
    scrollChatToBottom()
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : '回复失败，请稍后重试'
    messages.value[sessionID] = (messages.value[sessionID] ?? []).map((item) =>
      item.messageID === pendingMessageID
        ? { ...item, status: AiMessageStatus.FAILED_AAMS, content: errorMessage }
        : item,
    )
  } finally {
    setSessionSending(sessionID, false)
  }
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
    showError(error, '加载会话失败')
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
    title: '新会话',
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
    messages.value[sessionID] = normalizeMessageList(response.messages)
    scrollChatToBottom()
  } catch (error) {
    showError(error, '加载消息失败')
  } finally {
    loadingSessionID.value = ''
  }
}

function normalizeSession(session: AiSession): AiSession {
  return {
    ...session,
    title: session.title || '新会话',
    summary: session.summary || '',
  }
}

function normalizeSessionList(list?: AiSession[]) {
  return (list || []).filter((item) => item?.id).map(normalizeSession)
}

function normalizeMessageList(list?: AiMessage[]) {
  const result: ChatMessageItem[] = []
  for (const message of list || []) {
    const input = message.input_content?.content || ''
    if (input) {
      result.push(createMessageItem(message, 'user', input))
    }
    const output = message.output_content
    if (output?.content || message.status === AiMessageStatus.GENERATING_AAMS) {
      result.push(createMessageItem(message, 'ai', output?.content || THINKING_MESSAGE_CONTENT))
    }
  }
  return result
}

function createMessageItem(message: AiMessage, role: ChatRole, content: string) {
  const output = message.output_content
  return {
    ...message,
    key: `${message.id}:${role}`,
    messageID: message.id,
    role,
    content,
    tools: message.tools || [],
    model: output?.model || '',
    replySource: output?.reply_source || '',
    fallback: Boolean(output?.fallback),
    fallbackReason: output?.fallback_reason || '',
    tokenTotal: Number(message.token?.total || 0),
    firstTokenMs: Number(message.first_token_ms || 0),
    durationMs: Number(message.duration_ms || 0),
  } as ChatMessageItem
}

function createLocalMessage(
  messageID: string,
  role: ChatRole,
  content: string,
  attachments: AiAttachment[],
  status = AiMessageStatus.SUCCESS_AAMS,
) {
  const message = {
    id: messageID,
    input_content: role === 'user' ? { kind: 'text', content } : undefined,
    output_content: role === 'ai' ? { kind: 'text', content } : undefined,
    attachments,
    created_at: undefined,
    status,
    token: undefined,
    tools: [],
    first_token_ms: 0,
    duration_ms: 0,
  } as AiMessage
  return {
    ...createMessageItem(message, role, content),
    localOnly: true,
  }
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
    return '附件'
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
    <view class="ai-navbar" :style="{ height: navHeight, paddingTop: `${safeAreaTop}px` }">
      <button class="nav-back-button" hover-class="none" @tap="navigateBack">
        <uni-icons type="left" size="24" color="#111" />
      </button>
      <view class="ai-navbar__title">AI 助手</view>
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
          :loading="loadingSessions"
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
              item.status === AiMessageStatus.GENERATING_AAMS ? 'is-streaming' : '',
            ]"
            @longpress="handleMessageAction(item)"
          >
            <view v-if="item.role === 'ai' && item.model" class="reply-meta">
              <text class="reply-tag">模型回复</text>
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
                  isImageAttachment(attachment) ? '图' : '件'
                }}</view>
                <view class="attachment-info">
                  <view class="attachment-name">{{ attachment.name }}</view>
                  <view class="attachment-meta">{{ formatAttachmentMeta(attachment) }}</view>
                </view>
              </view>
            </view>
            <view v-if="item.tools.length" class="tool-row"
              >已调用：{{ formatTools(item.tools) }}</view
            >
          </view>
        </view>
        <view id="chat-bottom" class="chat-bottom"></view>
      </view>
      <view v-if="loadingSessionID" class="loading-session">正在加载消息...</view>
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
  height: 100%;
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
}

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
