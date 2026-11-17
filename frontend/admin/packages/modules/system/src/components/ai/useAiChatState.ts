import { computed, ref } from "vue";
import type { AiProviderModelOption } from "../../rpc/base/v1/ai_provider";
import type { AiKnowledgeOption } from "../../rpc/base/v1/ai_knowledge";
import type { AiSession } from "../../rpc/base/v1/ai_session";
import type { AiShortcut } from "../../rpc/base/v1/ai_tool";
import { resolveTimestamp } from "../../views/ai/chat/message";
import type { ChatMessageItem, SubmitPayload } from "../../views/ai/chat/types";

/** useAiChatState 管理可被不同聊天布局复用的 AI 会话展示状态。 */
export function useAiChatState() {
  const drafts = ref<Record<string, SubmitPayload>>({});
  const activeSessionID = ref("");
  const sendingSessionMap = ref<Record<string, boolean>>({});
  const loadingSessions = ref(false);
  const loadingSessionID = ref("");
  const loadingShortcuts = ref(false);
  const sessions = ref<AiSession[]>([]);
  const starterShortcuts = ref<AiShortcut[]>([]);
  const modelProviders = ref<AiProviderModelOption[]>([]);
  const selectedProviderId = ref(0);
  const selectedModelName = ref("");
  const knowledgeOptions = ref<AiKnowledgeOption[]>([]);
  const selectedKnowledgeIds = ref<number[]>([]);
  const messages = ref<Record<string, ChatMessageItem[]>>({});

  const activeSession = computed(() => sessions.value.find(item => item.id === activeSessionID.value));
  const currentMessages = computed(() => messages.value[activeSessionID.value] ?? []);
  const currentSessionSending = computed(() => isSessionSending(activeSessionID.value));

  /** 判断指定会话是否仍有请求在途。 */
  function isSessionSending(sessionID: string) {
    return Boolean(sessionID && sendingSessionMap.value[sessionID]);
  }

  /** 更新指定会话的发送状态，不影响其他会话的输入。 */
  function setSessionSending(sessionID: string, sending: boolean) {
    if (!sessionID) return;
    const nextMap = { ...sendingSessionMap.value };
    if (sending) {
      nextMap[sessionID] = true;
    } else {
      delete nextMap[sessionID];
    }
    sendingSessionMap.value = nextMap;
  }

  /** 更新或插入会话，并按更新时间保持最近会话优先。 */
  function upsertSession(session: AiSession) {
    const nextList = sessions.value.filter(item => item.id !== session.id);
    nextList.unshift(session);
    sessions.value = nextList.sort((left, right) => resolveTimestamp(right.updated_at) - resolveTimestamp(left.updated_at));
  }

  return {
    drafts,
    activeSessionID,
    loadingSessions,
    loadingSessionID,
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
  };
}
