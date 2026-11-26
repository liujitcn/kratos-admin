<template>
  <section class="chat-composer" :aria-label="t('system.ai.chat.placeholder.input')" :aria-busy="sending">
    <p v-if="blockedReason" class="chat-composer__notice" role="status">
      <Info :size="15" aria-hidden="true" />
      <span>{{ blockedReason }}</span>
    </p>
    <div v-if="voiceNotice" class="chat-composer__notice is-warning" role="status">
      <MicOff :size="15" aria-hidden="true" />
      <div>
        <p>{{ voiceNotice }}</p>
        <el-button link type="primary" @click="useKeyboard">{{ t("system.ai.chat.action.keyboard_input") }}</el-button>
      </div>
    </div>

    <AiAttachments :files="draftValue.attachments" :removable="!sending" @remove="removeAttachment" />
    <ul v-if="uploads.length" class="chat-composer__uploads">
      <li v-for="upload in uploads" :key="upload.id" class="chat-composer__upload" :class="{ 'is-error': upload.state !== 'uploading' }">
        <FileUp v-if="upload.state === 'uploading'" :size="16" aria-hidden="true" />
        <CircleAlert v-else :size="16" aria-hidden="true" />
        <div class="chat-composer__upload-file">
          <strong>{{ upload.file.name }}</strong>
          <template v-if="upload.state === 'uploading'">
            <span>{{ upload.percent === undefined ? t("system.ai.chat.status.uploading") : t("system.ai.chat.status.upload_progress", { percent: upload.percent }) }}</span>
            <progress :value="upload.percent" max="100" :aria-label="upload.file.name" />
          </template>
          <span v-else>{{ upload.error }}</span>
        </div>
        <div class="chat-composer__upload-actions">
          <el-button v-if="upload.state === 'uploading'" link type="primary" @click="removeUpload(upload)">
            {{ t("system.ai.chat.action.cancel_upload") }}
          </el-button>
          <template v-else>
            <el-button v-if="upload.state === 'failed'" link type="primary" @click="uploadAttachment(upload)">
              {{ t("system.ai.chat.action.retry_upload") }}
            </el-button>
            <button type="button" class="chat-composer__icon-button" :aria-label="t('system.ai.chat.file.remove', { name: upload.file.name })" @click="removeUpload(upload)">
              <X :size="16" aria-hidden="true" />
            </button>
          </template>
        </div>
      </li>
    </ul>

    <div class="chat-composer__editor">
      <textarea
        ref="inputRef"
        v-model="inputText"
        class="chat-composer__input"
        rows="1"
        :disabled="sending"
        :readonly="recording"
        :aria-label="t('system.ai.chat.placeholder.input')"
        :placeholder="t('system.ai.chat.placeholder.input')"
        @keydown.enter="handleEnter"
        @paste="handlePaste"
      />
      <button
        v-if="inputText && !sending && !recording"
        type="button"
        class="chat-composer__icon-button chat-composer__clear"
        :aria-label="t('system.ai.chat.action.clear_input')"
        :title="t('system.ai.chat.action.clear_input')"
        @click="clearText"
      >
        <X :size="14" aria-hidden="true" />
      </button>
    </div>

    <div class="chat-composer__toolbar">
      <div class="chat-composer__actions">
        <slot name="before-attachment" />
        <el-popover placement="top-start" :width="244" trigger="click" :disabled="sending || uploading">
          <template #reference>
            <button
              type="button"
              class="chat-composer__icon-button"
              :disabled="sending || uploading"
              :aria-label="t('system.ai.chat.action.upload_attachment')"
              :title="t('system.ai.chat.action.upload_attachment')"
            >
              <Paperclip :size="15" aria-hidden="true" />
            </button>
          </template>
          <div class="chat-composer__attachment-menu">
            <span>{{ t("system.ai.chat.message.attachment_hint") }}</span>
            <el-button :disabled="sending || uploading" @click="fileInputRef?.click()">
              {{ t("system.ai.chat.action.select_local_file") }}
            </el-button>
          </div>
        </el-popover>
      </div>

      <div class="chat-composer__actions chat-composer__actions--end">
        <AiModelSelector
          :model-providers="modelProviders"
          :provider-id="providerId"
          :model-name="modelName"
          :disabled="sending"
          @change="(id, name) => emit('model-change', id, name)"
        />
        <el-popover
          ref="knowledgePopoverRef"
          placement="top-start"
          :width="324"
          trigger="click"
          :show-arrow="false"
          :persistent="false"
          :disabled="sending || !knowledgeOptions.length"
          popper-class="agent-model-popper"
        >
          <template #reference>
            <el-button
              class="admin-custom-button agent-model-selector"
              :disabled="sending || !knowledgeOptions.length"
              :aria-label="t('system.ai.knowledge.title')"
            >
              <BookOpen :size="14" aria-hidden="true" />
              <span class="agent-model-selector__label">{{ knowledgeLabel }}</span>
              <ChevronDown :size="12" aria-hidden="true" />
            </el-button>
          </template>
          <div class="agent-model-panel">
            <div class="agent-model-panel__head">
              <span class="agent-model-panel__title">{{ t("system.ai.knowledge.title") }}</span>
              <span class="agent-model-panel__meta">{{ t("system.ai.knowledge.select_hint") }}</span>
            </div>
            <div class="agent-model-panel__list">
              <el-button
                v-for="option in knowledgeOptions"
                :key="option.id"
                class="admin-custom-button agent-model-row"
                :aria-pressed="knowledgeIds.includes(option.id)"
                :class="{ 'is-selected': knowledgeIds.includes(option.id) }"
                :disabled="sending"
                @click="toggleKnowledge(option.id)"
              >
                <span class="agent-model-row__name">{{ option.name }}</span>
                <span class="agent-model-row__raw">{{ t("system.ai.knowledge.doc_count", { count: option.doc_count ?? 0 }) }}</span>
                <Check v-if="knowledgeIds.includes(option.id)" :size="15" aria-hidden="true" />
              </el-button>
            </div>
            <el-button v-if="aiKnowledgeRoute" class="admin-custom-button agent-model-panel__manage" @click="openKnowledgeManage">
              <Settings :size="14" aria-hidden="true" />
              <span>{{ t("system.ai.knowledge.manage") }}</span>
              <ArrowRight :size="14" class="agent-model-panel__manage-arrow" aria-hidden="true" />
            </el-button>
          </div>
        </el-popover>
        <button
          type="button"
          class="chat-composer__icon-button"
          :class="{ 'is-active': recording }"
          :disabled="sending"
          :aria-pressed="recording"
          :aria-label="voiceActionLabel"
          :title="voiceActionLabel"
          @click="toggleRecord"
        >
          <LoaderCircle v-if="recording" class="is-loading" :size="16" aria-hidden="true" />
          <Mic v-else :size="16" aria-hidden="true" />
        </button>
        <button
          type="button"
          class="chat-composer__icon-button chat-composer__send"
          :disabled="!canSubmit"
          :aria-label="t(sending ? 'system.ai.chat.status.answering' : 'system.ai.chat.action.send')"
          :title="t('system.ai.chat.action.send')"
          @click="submit"
        >
          <LoaderCircle v-if="sending" class="is-loading" :size="16" aria-hidden="true" />
          <Send v-else :size="16" aria-hidden="true" />
        </button>
      </div>
    </div>

    <input ref="fileInputRef" type="file" multiple hidden :accept="AI_ATTACHMENT_ACCEPT" @change="handleFileChange" />
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, ref, shallowRef, watch } from "vue";
import { useRouter } from "vue-router";
import { ArrowRight, BookOpen, Check, ChevronDown, CircleAlert, FileUp, Info, LoaderCircle, Mic, MicOff, Paperclip, Send, Settings, X } from "@lucide/vue";
import type { PopoverInstance } from "element-plus";
import { resolveRequestErrorMessage, t, useSpeechRecognition } from "@liujitcn/kratos-admin-core";
import type { SpeechRecognitionError } from "@liujitcn/kratos-admin-core";
import { navigateTo } from "@liujitcn/kratos-admin-core/navigation";
import request from "@liujitcn/kratos-admin-core/request";
import { BaseFileAccessMode, type FileInfo } from "@liujitcn/kratos-admin-core/rpc/base/v1/file";
import { useAuthStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import type { AiAttachment } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import type { AiProviderModelOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_provider";
import type { AiKnowledgeOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_knowledge";
import AiModelSelector from "@liujitcn/kratos-admin-system/components/AiModelSelector/index.vue";
import {
  AI_ATTACHMENT_ACCEPT,
  AI_ATTACHMENT_MAX_COUNT,
  AI_ATTACHMENT_MAX_SIZE_MB,
  planAIAttachmentUploads,
  type AIAttachmentUploadIssue
} from "../attachment";
import type { SubmitPayload } from "../types";
import AiAttachments from "./AiAttachments.vue";

/** 本地附件传输状态，只有上传成功的文件会进入会话草稿。 */
interface ComposerUpload {
  id: number;
  file: File;
  state: "uploading" | "failed" | "rejected";
  percent?: number;
  error?: string;
  controller?: AbortController;
}

const props = defineProps<{
  /** 当前会话是否正在生成回复。 */
  sending: boolean;
  /** 无可用模型或待确认操作时展示的原因。 */
  blockedReason?: string;
  /** 当前可供选择的供应商与模型。 */
  modelProviders: AiProviderModelOption[];
  /** 当前选择的供应商编号。 */
  providerId: number;
  /** 当前选择的模型名称。 */
  modelName: string;
  /** 当前可供选择的知识库。 */
  knowledgeOptions: AiKnowledgeOption[];
  /** 当前选择的知识库编号列表。 */
  knowledgeIds: number[];
  /** 按会话保存的输入与附件。 */
  draft?: SubmitPayload;
}>();

const emit = defineEmits<{
  /** 提交输入，接受后由会话清除对应的草稿。 */
  submit: [payload: SubmitPayload];
  /** 更新当前选择的供应商和模型。 */
  "model-change": [providerId: number, modelName: string];
  /** 更新当前选择的知识库编号列表。 */
  "knowledge-change": [knowledgeIds: number[]];
  /** 写入当前会话草稿。 */
  "draft-change": [value: SubmitPayload];
}>();

const inputRef = ref<HTMLTextAreaElement>();
const fileInputRef = ref<HTMLInputElement>();
const knowledgePopoverRef = ref<PopoverInstance>();
const router = useRouter();
const authStore = useAuthStore();
const aiKnowledgeRoute = computed(() => authStore.flatMenuListGet.find(item => item.name === "AiKnowledge" && item.path));
const emptyDraft: SubmitPayload = { text: "", attachments: [] };
const draftValue = shallowRef(props.draft ?? emptyDraft);
const inputText = computed({
  get: () => draftValue.value.text,
  set: text => updateDraft({ ...draftValue.value, text })
});
watch(() => props.draft, draft => { draftValue.value = draft ?? emptyDraft; });
if (typeof CSS !== "undefined" && !CSS.supports("field-sizing", "content")) {
  watch([inputRef, inputText], ([input]) => {
    if (!input) return;
    input.style.height = "auto";
    input.style.height = `${input.scrollHeight}px`;
  }, { flush: "post" });
}

const uploads = ref<ComposerUpload[]>([]);
const uploading = computed(() => uploads.value.some(upload => upload.state === "uploading"));
let nextUploadId = 0;
onBeforeUnmount(() => {
  for (const upload of uploads.value) upload.controller?.abort();
});

const voiceNotice = ref("");
let voiceBaseText = "";
const { loading: recording, start: startRecord, stop: stopRecord } = useSpeechRecognition({
  onStart: () => { voiceBaseText = inputText.value; },
  onResult: recordText => {
    const text = recordText.replace(/\s+/g, " ").trim();
    if (text) inputText.value = voiceBaseText && !/\s$/.test(voiceBaseText) ? `${voiceBaseText} ${text}` : voiceBaseText + text;
  },
  onError: showRecordError
});
const voiceActionLabel = computed(() =>
  t(recording.value ? "system.ai.chat.action.stop_voice_input" : "system.ai.chat.action.voice_input")
);
const canSubmit = computed(() =>
  !props.sending && !props.blockedReason && !recording.value && !uploads.value.length &&
  (Boolean(inputText.value.trim()) || Boolean(draftValue.value.attachments.length))
);
const knowledgeLabel = computed(() => {
  if (props.knowledgeIds.length === 1) {
    const option = props.knowledgeOptions.find(item => item.id === props.knowledgeIds[0]);
    if (option) return option.name;
  }
  return props.knowledgeIds.length
    ? t("system.ai.knowledge.selected_count", { count: props.knowledgeIds.length })
    : t("system.ai.knowledge.title");
});

defineExpose({ focus });

/** 聚焦输入框，供会话结果和快捷入口填入草稿后调用。 */
function focus() {
  inputRef.value?.focus();
}

/** 将本地编辑立即同步到当前会话，避免切换时合并到其他会话。 */
function updateDraft(value: SubmitPayload) {
  draftValue.value = value;
  emit("draft-change", value);
}

/** 处理回车发送，保留换行与中文输入法的确认键。 */
function handleEnter(event: KeyboardEvent) {
  if (event.shiftKey || event.isComposing || event.keyCode === 229) return;
  event.preventDefault();
  if (!event.repeat) submit();
}

/** 提交文本或单独附件，等待会话接受后更新草稿。 */
function submit() {
  if (!canSubmit.value) return;
  emit("submit", {
    text: inputText.value.trim() || t("system.ai.chat.value.analyze_attachments"),
    attachments: [...draftValue.value.attachments]
  });
}

/** 清空文本并返回输入框，保留已选附件。 */
function clearText() {
  inputText.value = "";
  focus();
}

/** 切换当前选择的知识库。 */
function toggleKnowledge(id: number) {
  const ids = props.knowledgeIds;
  emit("knowledge-change", ids.includes(id) ? ids.filter(item => item !== id) : [...ids, id]);
}

/** 跳转到具备菜单权限的知识库管理页面。 */
async function openKnowledgeManage() {
  if (!aiKnowledgeRoute.value?.path) return;
  knowledgePopoverRef.value?.hide();
  await navigateTo(router, aiKnowledgeRoute.value.path);
}

/** 启动浏览器的语音识别，实时显示识别状态。 */
function toggleRecord() {
  if (recording.value) {
    stopRecord();
  } else {
    voiceNotice.value = "";
    startRecord();
  }
}

/** 显示实际浏览器返回的语音权限或识别提示。 */
function showRecordError(error: SpeechRecognitionError) {
  if (error.code === -1) {
    voiceNotice.value = t("system.ai.chat.message.voice_unsupported");
  } else if (["not-allowed", "service-not-allowed", "permission-denied"].includes(error.error ?? "")) {
    voiceNotice.value = t("system.ai.chat.message.microphone_denied");
  } else {
    voiceNotice.value = t("system.ai.chat.message.voice_failed");
  }
}

/** 关闭语音提示并继续键盘输入。 */
function useKeyboard() {
  voiceNotice.value = "";
  focus();
}

/** 读取原生选择器返回的完整批次，允许重复选择同名文件。 */
function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const files = Array.from(input.files ?? []);
  input.value = "";
  addFiles(files);
}

/** 上传剪贴板文件，同时让包含文本的粘贴保留原生文本行为。 */
function handlePaste(event: ClipboardEvent) {
  const files = Array.from(event.clipboardData?.files ?? []);
  if (!files.length || props.sending || uploading.value) return;
  if (!event.clipboardData?.getData("text/plain")) event.preventDefault();
  addFiles(files);
}

/** 校验待上传文件，为有效文件建立独立的传输与重试状态。 */
function addFiles(files: File[]) {
  if (!files.length || props.sending || uploading.value) return;
  const plan = planAIAttachmentUploads(files, [
    ...draftValue.value.attachments.map(file => Number(file.size ?? 0)),
    ...uploads.value.filter(upload => upload.state !== "rejected").map(upload => upload.file.size)
  ]);
  if (plan.truncated) {
    ElMessage.warning(
      plan.remaining
        ? t("system.ai.chat.message.attachment_remaining", { count: plan.remaining })
        : t("system.ai.chat.message.attachment_limit", { count: AI_ATTACHMENT_MAX_COUNT })
    );
  }
  for (const { file, reason } of plan.rejected) {
    uploads.value.push({ id: ++nextUploadId, file, state: "rejected", error: uploadIssueMessage(file, reason) });
  }
  for (const file of plan.accepted) {
    const upload = reactive<ComposerUpload>({ id: ++nextUploadId, file, state: "uploading" });
    uploads.value.push(upload);
    void uploadAttachment(upload);
  }
}

/** 根据校验原因生成对应的附件行文案。 */
function uploadIssueMessage(file: File, reason: AIAttachmentUploadIssue) {
  switch (reason) {
    case "unsupported":
      return t("system.ai.chat.message.attachment_unsupported", { name: file.name });
    case "too_large":
      return t("system.ai.chat.message.attachment_too_large", { name: file.name, size: AI_ATTACHMENT_MAX_SIZE_MB });
    case "total_too_large":
      return t("system.ai.chat.message.attachment_total_too_large", { size: AI_ATTACHMENT_MAX_SIZE_MB });
  }
}

/** 通过鉴权请求上传单个附件，使用实际传输进度，取消后忽略迟到响应。 */
async function uploadAttachment(upload: ComposerUpload) {
  const controller = new AbortController();
  upload.controller = controller;
  upload.state = "uploading";
  upload.percent = undefined;
  upload.error = "";

  const data = new FormData();
  data.append("file", upload.file);
  data.append("fileType", "ai");
  data.append("accessMode", String(BaseFileAccessMode.BASE_FILE_ACCESS_MODE_AUTHORIZED));
  try {
    const response = await request<FormData, FileInfo>({
      url: "/v1/base/file",
      method: "post",
      data,
      signal: controller.signal,
      headers: { "Content-Type": "multipart/form-data" },
      onUploadProgress: ({ progress }) => {
        if (!controller.signal.aborted && progress !== undefined) upload.percent = Math.round(progress * 100);
      }
    });
    if (controller.signal.aborted) return;
    if (!response.url) throw new Error(t("system.ai.chat.message.attachment_upload_failed"));
    const attachment: AiAttachment = {
      id: response.url,
      url: response.url,
      name: response.name || upload.file.name,
      size: upload.file.size,
      mime_type: upload.file.type
    };
    if (!draftValue.value.attachments.some(file => file.url === attachment.url)) {
      updateDraft({ ...draftValue.value, attachments: [...draftValue.value.attachments, attachment] });
    }
    uploads.value = uploads.value.filter(item => item.id !== upload.id);
  } catch (error) {
    if (controller.signal.aborted) return;
    upload.state = "failed";
    upload.error = resolveRequestErrorMessage(error, t("system.ai.chat.message.attachment_upload_failed"));
  } finally {
    if (upload.controller === controller) upload.controller = undefined;
  }
}

/** 取消当前传输或移除失败的本地文件。 */
function removeUpload(upload: ComposerUpload) {
  upload.controller?.abort();
  uploads.value = uploads.value.filter(item => item.id !== upload.id);
}

/** 从当前草稿移除已上传的附件。 */
function removeAttachment(attachment: AiAttachment) {
  updateDraft({
    ...draftValue.value,
    attachments: draftValue.value.attachments.filter(file => file !== attachment)
  });
}
</script>

<style scoped lang="scss">
.chat-composer {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  min-width: 0;
  padding: 8px;
  box-sizing: border-box;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius);

  &:focus-within {
    border-color: var(--el-color-primary-light-3);
  }

  :deep(.agent-model-selector.el-button.admin-custom-button) {
    gap: 5px;
    width: auto;
    max-width: min(15rem, 100%);
    height: 30px;
    padding: 0 6px;
    font-size: 12px;
    background: transparent;
    border-color: transparent;

    &:hover {
      background: var(--el-fill-color-light);
    }

    .agent-model-selector__label {
      font-size: inherit;
    }
  }
}

.chat-composer__editor {
  position: relative;
}

.chat-composer__input {
  display: block;
  width: 100%;
  min-height: 36px;
  max-height: 132px;
  padding: 6px 31px 6px 6px;
  box-sizing: border-box;
  overflow: auto;
  color: var(--el-text-color-primary);
  font-family: inherit;
  font-size: 14px;
  line-height: 1.65;
  resize: none;
  field-sizing: content;
  background: transparent;
  border: 0;
  outline: none;

  &::placeholder {
    color: var(--el-text-color-placeholder);
  }

  &:disabled {
    color: var(--el-text-color-disabled);
    cursor: not-allowed;
  }
}

.chat-composer__toolbar,
.chat-composer__actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.chat-composer__toolbar {
  justify-content: space-between;
}

.chat-composer__actions--end {
  justify-content: flex-end;
  margin-inline-start: auto;
}

.chat-composer__icon-button {
  display: inline-grid;
  flex: none;
  width: 30px;
  height: 30px;
  place-items: center;
  padding: 0;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--admin-page-radius);

  &:hover:not(:disabled),
  &.is-active {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  &:disabled {
    color: var(--el-text-color-disabled);
    cursor: not-allowed;
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 1px;
  }
}

.chat-composer__clear {
  position: absolute;
  inset-block-start: 2px;
  inset-inline-end: 0;
  width: 26px;
  height: 26px;
}

.chat-composer__icon-button.chat-composer__send {
  color: var(--el-color-white);
  background: var(--el-color-primary);

  &:hover:not(:disabled) {
    color: var(--el-color-white);
    background: var(--el-color-primary-dark-2);
  }

  &:disabled {
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-light);
  }
}

.chat-composer__notice,
.chat-composer__attachment-menu {
  display: flex;
  gap: 6px;
  align-items: flex-start;
  margin: 0;
  padding: 5px 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;

  svg {
    flex: none;
  }

  p {
    margin: 0;
  }
}

.chat-composer__notice.is-warning {
  color: var(--el-text-color-primary);
  background: var(--el-color-warning-light-9);
  border-radius: var(--admin-page-radius);

  > svg {
    color: var(--el-color-warning);
  }
}

.chat-composer__attachment-menu {
  flex-direction: column;
}

.chat-composer__uploads {
  display: grid;
  gap: 4px;
  max-height: 12rem;
  padding: 0;
  margin: 0;
  overflow: auto;
  list-style: none;
}

.chat-composer__upload {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 6px;
  align-items: start;
  padding: 6px;
  color: var(--el-text-color-secondary);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius);

  &.is-error > svg {
    color: var(--el-color-danger);
  }
}

.chat-composer__upload-file {
  display: grid;
  gap: 2px;
  font-size: 12px;
  overflow-wrap: anywhere;

  strong {
    color: var(--el-text-color-primary);
  }

  progress {
    width: min(100%, 12rem);
    height: 8px;
    accent-color: var(--el-color-primary);
  }
}

.chat-composer__upload-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 3px;
  align-items: center;
  justify-content: flex-end;

  .chat-composer__icon-button {
    width: 24px;
    height: 24px;
  }
}

@media (width <= 480px) {
  .chat-composer__input {
    font-size: 16px;
  }
}
</style>
