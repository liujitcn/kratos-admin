<template>
  <div class="agent-sender-wrap">
    <BaseXSender
      ref="senderRef"
      class="agent-sender"
      variant="updown"
      submit-type="enter"
      :placeholder="t('system.ai.chat.placeholder.input')"
      :loading="sending"
      :clearable="true"
      :tip-config="false"
      @change="handleInputChange"
      @submit="handleSubmit"
      @paste-file="handlePasteFile"
    >
      <template v-if="selectedAttachments.length" #header>
        <Attachments
          class="agent-attachments"
          :items="attachmentItems"
          overflow="wrap"
          :hide-upload="true"
          @delete-card="handleDeleteCard"
        />
      </template>

      <template #prefix>
        <div class="agent-prefix-actions">
          <el-popover placement="top-start" :width="236" trigger="click" popper-class="agent-sender-popover">
            <template #reference>
              <button
                class="agent-icon-button"
                type="button"
                :disabled="sending || uploading"
                :aria-label="t('system.ai.chat.action.upload_attachment')"
              >
                <el-icon :class="{ 'is-loading': uploading }">
                  <Loading v-if="uploading" />
                  <Paperclip v-else />
                </el-icon>
              </button>
            </template>
            <div class="agent-popover-card">
              <div class="agent-popover-title">{{ t("system.ai.chat.action.upload_attachment") }}</div>
              <div class="agent-popover-desc">{{ t("system.ai.chat.message.attachment_hint") }}</div>
              <button class="agent-popover-action" type="button" :disabled="sending || uploading" @click="handleSelectAttachment">
                {{ uploading ? t("system.ai.chat.status.uploading") : t("system.ai.chat.action.select_local_file") }}
              </button>
            </div>
          </el-popover>
          <span class="agent-action-text">{{ actionHintText }}</span>
        </div>
      </template>

      <template #action-list>
        <div class="agent-sender-actions">
          <el-popover
            ref="modelPopoverRef"
            placement="top-start"
            :width="324"
            trigger="click"
            :show-arrow="false"
            :persistent="false"
            popper-class="agent-model-popper"
          >
            <template #reference>
              <button
                class="agent-model-selector"
                :class="{ 'is-disabled': sending || !modelOptions.length }"
                type="button"
                :disabled="sending || !modelOptions.length"
                :aria-label="t('system.ai.model.title')"
              >
                <el-icon class="agent-model-selector__spark"><MagicStick /></el-icon>
                <span class="agent-model-selector__label">{{
                  selectedModelDisplayName || t("system.ai.model.model_placeholder")
                }}</span>
                <el-icon class="agent-model-selector__arrow"><ArrowDown /></el-icon>
              </button>
            </template>
            <div v-if="modelOptions.length" class="agent-model-panel">
              <div class="agent-model-panel__head">
                <span class="agent-model-panel__title">{{ t("system.ai.model.title") }}</span>
                <span class="agent-model-panel__meta">{{
                  t("system.ai.model.provider_model_count", {
                    providers: modelOptions.length,
                    models: modelOptionCount
                  })
                }}</span>
              </div>
              <div class="agent-model-panel__list">
                <template v-for="provider in modelOptions" :key="provider.providerId">
                  <div class="agent-model-panel__group">{{ provider.label }}</div>
                  <button
                    v-for="model in provider.models"
                    :key="model.name"
                    class="agent-model-row"
                    :class="{ 'is-selected': isModelSelected(provider.providerId, model.name) }"
                    type="button"
                    @click="handleModelSelect(provider.providerId, model.name)"
                  >
                    <span class="agent-model-row__name">{{ model.displayName }}</span>
                    <span v-if="model.displayName !== model.name" class="agent-model-row__raw">{{
                      model.name
                    }}</span>
                    <el-icon
                      v-if="isModelSelected(provider.providerId, model.name)"
                      class="agent-model-row__check"
                    >
                      <Check />
                    </el-icon>
                  </button>
                </template>
              </div>
              <button
                v-if="aiProviderRoute"
                class="agent-model-panel__manage"
                type="button"
                @click="openModelManage"
              >
                <el-icon><Setting /></el-icon>
                <span>{{ t("system.ai.model.manage") }}</span>
                <el-icon class="agent-model-panel__manage-arrow"><ArrowRight /></el-icon>
              </button>
            </div>
            <div v-else class="agent-model-panel agent-model-panel--empty">
              {{ t("system.ai.model.empty") }}
            </div>
          </el-popover>
          <el-popover
            ref="knowledgePopoverRef"
            placement="top-start"
            :width="324"
            trigger="click"
            :show-arrow="false"
            :persistent="false"
            popper-class="agent-model-popper"
          >
            <template #reference>
              <button
                class="agent-model-selector"
                :class="{ 'is-disabled': sending || !knowledgeOptions.length, 'is-selected': selectedKnowledgeCount > 0 }"
                type="button"
                :disabled="sending || !knowledgeOptions.length"
                :aria-label="t('system.ai.knowledge.title')"
              >
                <el-icon class="agent-model-selector__spark"><Collection /></el-icon>
                <span class="agent-model-selector__label">{{
                  selectedKnowledgeCount > 0
                    ? t("system.ai.knowledge.selected_count", { count: selectedKnowledgeCount })
                    : t("system.ai.knowledge.title")
                }}</span>
                <el-icon class="agent-model-selector__arrow"><ArrowDown /></el-icon>
              </button>
            </template>
            <div v-if="knowledgeOptions.length" class="agent-model-panel">
              <div class="agent-model-panel__head">
                <span class="agent-model-panel__title">{{ t("system.ai.knowledge.title") }}</span>
                <span class="agent-model-panel__meta">{{
                  t("system.ai.knowledge.select_hint")
                }}</span>
              </div>
              <div class="agent-model-panel__list">
                <button
                  v-for="option in knowledgeOptions"
                  :key="option.id"
                  class="agent-model-row"
                  :class="{ 'is-selected': isKnowledgeSelected(option.id) }"
                  type="button"
                  @click="handleKnowledgeToggle(option.id)"
                >
                  <span class="agent-model-row__name">{{ option.name }}</span>
                  <!-- proto3 JSON 会省略零值字段，计数为 0 时前端补默认值。 -->
                  <span class="agent-model-row__raw">{{
                    t("system.ai.knowledge.doc_count", { count: option.doc_count ?? 0 })
                  }}</span>
                  <el-icon v-if="isKnowledgeSelected(option.id)" class="agent-model-row__check">
                    <Check />
                  </el-icon>
                </button>
              </div>
              <button
                v-if="aiKnowledgeRoute"
                class="agent-model-panel__manage"
                type="button"
                @click="openKnowledgeManage"
              >
                <el-icon><Setting /></el-icon>
                <span>{{ t("system.ai.knowledge.manage") }}</span>
                <el-icon class="agent-model-panel__manage-arrow"><ArrowRight /></el-icon>
              </button>
            </div>
            <div v-else class="agent-model-panel agent-model-panel--empty">
              {{ t("system.ai.knowledge.empty") }}
            </div>
          </el-popover>
          <el-tooltip
            :content="recording ? t('system.ai.chat.action.stop_voice_input') : t('system.ai.chat.action.voice_input')"
            placement="top"
          >
            <button
              class="agent-icon-button"
              :class="{ 'is-active': recording }"
              type="button"
              :disabled="sending"
              :aria-pressed="recording"
              :aria-label="recording ? t('system.ai.chat.action.stop_voice_input') : t('system.ai.chat.action.voice_input')"
              @click="handleToggleRecord"
            >
              <el-icon :class="{ 'is-loading': recording }">
                <Loading v-if="recording" />
                <Microphone v-else />
              </el-icon>
            </button>
          </el-tooltip>
          <el-tooltip :content="t('system.ai.chat.action.send')" placement="top">
            <button
              class="agent-send-button"
              type="button"
              :disabled="sending || uploading || recording || isSubmitDisabled"
              :aria-label="t('system.ai.chat.action.send')"
              @click="handleSubmit()"
            >
              <el-icon v-if="!sending"><Promotion /></el-icon>
              <el-icon v-else class="is-loading"><Loading /></el-icon>
            </button>
          </el-tooltip>
        </div>
      </template>
    </BaseXSender>

    <input
      ref="fileInputRef"
      class="agent-file-input"
      type="file"
      multiple
      :accept="acceptedAttachmentTypes"
      @change="handleFileChange"
    />
  </div>
</template>

<script setup lang="ts" name="XSender">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { Attachments, XSender as BaseXSender } from "vue-element-plus-x";
import type { FilesCardProps } from "vue-element-plus-x/types/FilesCard";
import {
  ArrowDown,
  ArrowRight,
  Check,
  Collection,
  Loading,
  MagicStick,
  Microphone,
  Paperclip,
  Promotion,
  Setting
} from "@element-plus/icons-vue";
import { ElMessage, ElPopover } from "element-plus";
import { t } from "@liujitcn/kratos-admin-core";
import { navigateTo } from "@liujitcn/kratos-admin-core/navigation";
import { useAuthStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import { defFileService } from "@liujitcn/kratos-admin-core/api/base/v1/file";
import type { AiAttachment } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import type { AiProviderModelOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_provider";
import type { AiKnowledgeOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_knowledge";
import type { SubmitPayload } from "../types";
import { buildAIAttachmentFileCard } from "../attachment";
import { fallbackModelDisplayName } from "../modelDisplay";
import { useSpeechRecognition } from "./speech-recognition";
import type { SpeechRecognitionError } from "./speech-recognition";

const props = defineProps<{
  /** 消息发送加载状态。 */
  sending: boolean;
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
}>();

const emit = defineEmits<{
  /** 提交输入器内容。 */
  submit: [payload: SubmitPayload];
  /** 更新当前选择的供应商和模型。 */
  "model-change": [providerId: number, modelName: string];
  /** 更新当前选择的知识库编号列表。 */
  "knowledge-change": [knowledgeIds: number[]];
}>();

const senderRef = ref<InstanceType<typeof BaseXSender>>();
const fileInputRef = ref<HTMLInputElement>();
const modelPopoverRef = ref<InstanceType<typeof ElPopover>>();
const router = useRouter();
const authStore = useAuthStore();
const inputText = ref("");
const selectedAttachments = ref<AiAttachment[]>([]);
const modelOptions = computed(() =>
  props.modelProviders.map(provider => ({
    providerId: provider.provider_id,
    label: provider.name,
    models: provider.models.map(model => ({
      name: model.model_name,
      displayName: model.display_name || fallbackModelDisplayName(model.model_name)
    }))
  }))
);
const modelOptionCount = computed(() =>
  modelOptions.value.reduce((total, provider) => total + provider.models.length, 0)
);
const selectedModelDisplayName = computed(() => {
  const provider = modelOptions.value.find(item => item.providerId === props.providerId);
  return (
    provider?.models.find(model => model.name === props.modelName)?.displayName ??
    fallbackModelDisplayName(props.modelName)
  );
});
const aiProviderRoute = computed(() =>
  authStore.flatMenuListGet.find(item => item.name === "AiProvider" && item.path)
);
const aiKnowledgeRoute = computed(() =>
  authStore.flatMenuListGet.find(item => item.name === "AiKnowledge" && item.path)
);
const uploading = ref(false);
const maxAttachmentCount = 6;
const maxAttachmentSizeMB = 20;
const maxAttachmentSize = maxAttachmentSizeMB * 1024 * 1024;
const acceptedAttachmentExtensions = [
  ".txt",
  ".md",
  ".markdown",
  ".log",
  ".json",
  ".xml",
  ".csv",
  ".png",
  ".jpg",
  ".jpeg",
  ".gif",
  ".webp"
];
const acceptedAttachmentTypes = acceptedAttachmentExtensions.join(",");

const voiceBaseText = ref("");
const {
  loading: recording,
  start: startRecord,
  stop: stopRecord
} = useSpeechRecognition({
  onStart: () => {
    voiceBaseText.value = inputText.value.trim();
  },
  onEnd: handleRecordEnd,
  onResult: setRecordText,
  onError: handleRecordError
});

const actionHintText = computed(() => {
  if (recording.value) return t("system.ai.chat.status.recognizing_voice");
  if (uploading.value) return t("system.ai.chat.status.uploading_attachments");
  if (selectedAttachments.value.length) {
    return t("system.ai.chat.value.selected_attachments", { count: selectedAttachments.value.length });
  }
  return t("system.ai.chat.value.attachments_available");
});

const attachmentItems = computed<FilesCardProps[]>(() =>
  selectedAttachments.value.map(item =>
    buildAIAttachmentFileCard(item, {
      showDelIcon: true,
      maxWidth: "220px"
    })
  )
);

const isSubmitDisabled = computed(() => {
  return uploading.value || (!inputText.value.trim() && selectedAttachments.value.length === 0);
});

/** 判断指定供应商与模型是否为当前选择。 */
function isModelSelected(providerId: number, modelName: string) {
  return props.providerId === providerId && props.modelName === modelName;
}

/** 选中模型并关闭选择面板。 */
function handleModelSelect(providerId: number, modelName: string) {
  emit("model-change", providerId, modelName);
  modelPopoverRef.value?.hide();
}

/** 跳转到 AI 供应商配置页。 */
async function openModelManage() {
  if (!aiProviderRoute.value?.path) return;
  modelPopoverRef.value?.hide();
  await navigateTo(router, aiProviderRoute.value.path);
}

const knowledgePopoverRef = ref<InstanceType<typeof ElPopover>>();

const selectedKnowledgeCount = computed(() => props.knowledgeIds.length);

/** 判断知识库是否在当前选择中。 */
function isKnowledgeSelected(knowledgeId: number) {
  return props.knowledgeIds.includes(knowledgeId);
}

/** 切换知识库选择状态并同步父组件。 */
function handleKnowledgeToggle(knowledgeId: number) {
  const current = new Set(props.knowledgeIds);
  if (current.has(knowledgeId)) current.delete(knowledgeId);
  else current.add(knowledgeId);
  emit("knowledge-change", [...current]);
}

/** 跳转到 AI 知识库管理页。 */
async function openKnowledgeManage() {
  if (!aiKnowledgeRoute.value?.path) return;
  knowledgePopoverRef.value?.hide();
  await navigateTo(router, aiKnowledgeRoute.value.path);
}

/** 读取输入内容并发送给父组件。 */
function handleSubmit() {
  if (uploading.value) return;
  if (recording.value) {
    stopRecord();
    return;
  }
  const trimmedText = inputText.value.trim();
  if (!trimmedText && selectedAttachments.value.length === 0) return;

  emit("submit", {
    text: trimmedText || t("system.ai.chat.value.analyze_attachments"),
    attachments: [...selectedAttachments.value]
  });
  inputText.value = "";
  selectedAttachments.value = [];
  senderRef.value?.clear();
  resetFileInput();
}

/** 切换浏览器语音识别状态。 */
function handleToggleRecord() {
  if (recording.value) {
    stopRecord();
    return;
  }
  startRecord();
}

/** 语音识别结束后保留当前识别文本。 */
function handleRecordEnd(result: string) {
  setRecordText(result);
  voiceBaseText.value = "";
}

/** 处理语音识别失败，提示浏览器不支持或麦克风授权异常。 */
function handleRecordError(error: SpeechRecognitionError) {
  const message = resolveRecordErrorMessage(error);
  ElMessage.warning(message);
}

/** 将识别文本与识别前内容合并后写回输入器。 */
function setRecordText(recordText: string) {
  const normalizedRecordText = normalizeRecordText(recordText);
  if (!normalizedRecordText) return;
  const nextText = [voiceBaseText.value, normalizedRecordText].filter(Boolean).join(" ");
  senderRef.value?.setText(nextText);
  inputText.value = nextText;
}

/** 标准化语音识别文本，压掉多余空白。 */
function normalizeRecordText(recordText: string) {
  return recordText.replace(/\s+/g, " ").trim();
}

/** 根据浏览器语音识别错误类型生成用户可理解的提示。 */
function resolveRecordErrorMessage(error: SpeechRecognitionError) {
  if (error.code === -1) return t("system.ai.chat.message.voice_unsupported");
  const errorName = error.error ?? "";
  if (["not-allowed", "service-not-allowed", "permission-denied"].includes(errorName)) {
    return t("system.ai.chat.message.microphone_denied");
  }
  return t("system.ai.chat.message.voice_failed");
}

/** 同步输入器内部文本，保证发送按钮禁用态能实时响应。 */
function handleInputChange() {
  inputText.value = senderRef.value?.getModelValue().text ?? "";
}

/** 打开本地文件选择框。 */
function handleSelectAttachment() {
  if (uploading.value || props.sending) return;
  fileInputRef.value?.click();
}

/** 将本地文件列表上传后同步到输入器附件区。 */
async function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement;
  await uploadAttachments(Array.from(target.files ?? []));
}

/** 处理粘贴文件，和点击上传保持同一份附件状态。 */
async function handlePasteFile(firstFile: File, fileList: FileList) {
  await uploadAttachments([firstFile, ...Array.from(fileList).slice(1)]);
}

/** 上传附件并按文件地址去重。 */
async function uploadAttachments(files: File[]) {
  if (uploading.value || props.sending) return;
  const uploadFiles = filterUploadFiles(files);
  if (!uploadFiles.length) return;
  uploading.value = true;
  try {
    const response = await defFileService.MultiUploadFile(uploadFiles, "ai");
    const fileMap = new Map<string, File[]>();
    uploadFiles.forEach(file => {
      const group = fileMap.get(file.name) ?? [];
      group.push(file);
      fileMap.set(file.name, group);
    });
    const attachmentMap = new Map(selectedAttachments.value.map(item => [item.url || item.id, item]));
    response?.files?.forEach(item => {
      const matchedFile = fileMap.get(item.name || "")?.shift();
      const attachment = buildAttachmentItem(item.url || "", item.name || "", matchedFile);
      attachmentMap.set(attachment.url || attachment.id, attachment);
    });
    selectedAttachments.value = Array.from(attachmentMap.values());
    if (response?.files?.length) {
      ElMessage.success(t("system.ai.chat.message.attachments_uploaded", { count: response.files.length }));
    }
  } catch {
    ElMessage.error(t("system.ai.chat.message.attachment_upload_failed"));
  } finally {
    uploading.value = false;
    resetFileInput();
  }
}

/** 过滤不符合数量和大小约束的附件。 */
function filterUploadFiles(files: File[]) {
  const remainingCount = maxAttachmentCount - selectedAttachments.value.length;
  if (remainingCount <= 0) {
    ElMessage.warning(t("system.ai.chat.message.attachment_limit", { count: maxAttachmentCount }));
    return [];
  }
  const validFiles = files.filter(file => {
    if (!isAcceptedAttachmentFile(file)) {
      ElMessage.warning(t("system.ai.chat.message.attachment_unsupported", { name: file.name }));
      return false;
    }
    if (file.size > maxAttachmentSize) {
      ElMessage.warning(t("system.ai.chat.message.attachment_too_large", { name: file.name, size: maxAttachmentSizeMB }));
      return false;
    }
    return true;
  });
  if (validFiles.length > remainingCount) {
    ElMessage.warning(t("system.ai.chat.message.attachment_remaining", { count: remainingCount }));
  }
  return validFiles.slice(0, remainingCount);
}

/** 判断附件类型是否在当前 AI 助手可解析范围内。 */
function isAcceptedAttachmentFile(file: File) {
  const fileName = file.name.toLowerCase();
  return acceptedAttachmentExtensions.some(extension => fileName.endsWith(extension));
}

/** 根据上传结果构建附件展示项。 */
function buildAttachmentItem(url: string, name: string, file?: File): AiAttachment {
  return {
    id: url || `${name}-${file?.size ?? 0}-${file?.lastModified ?? Date.now()}`,
    name,
    size: file?.size ?? 0,
    url,
    mime_type: file?.type ?? ""
  };
}

/** 复用 Attachments 删除事件，保持附件状态与组件展示同步。 */
function handleDeleteCard(item: { uid?: string | number }) {
  if (!item.uid) return;
  selectedAttachments.value = selectedAttachments.value.filter(attachment => attachment.id !== String(item.uid));
  if (!selectedAttachments.value.length) resetFileInput();
}

/** 清空文件输入框，保证重复选择同名文件时仍能触发 change。 */
function resetFileInput() {
  if (fileInputRef.value) fileInputRef.value.value = "";
}
</script>

<style scoped lang="scss">
.agent-sender-wrap {
  width: 100%;
}
.agent-sender {
  :deep(.elx-x-sender) {
    background: var(--admin-page-card-bg);
    border-radius: var(--admin-page-radius);
    box-shadow: none;
  }
  :deep(.elx-x-sender__content) {
    padding: 8px 10px 10px;
    background: transparent;
    border: 0;
    border-radius: inherit;
  }
  :deep(.elx-x-sender__content--variant-updown) {
    gap: 10px;
  }
  :deep(.chat-rich-text) {
    min-height: 64px;
    max-height: 120px;
    padding: 8px 10px;
    font-size: 14px;
    line-height: 22px;
  }
  :deep(.chat-placeholder-wrap) {
    padding: 8px 10px;
    font-size: 14px;
    font-weight: 400;
  }
  :deep(.elx-x-sender__updown-action-list) {
    align-items: center;
    justify-content: space-between;
  }
  :deep(.elx-x-sender__updown-action-list .elx-x-sender__prefix) {
    flex: 1 1 auto;
    width: 100%;
    min-width: 0;
    height: auto;
    padding-left: 0;
  }
  :deep(.elx-x-sender__action-list) {
    height: auto;
  }
}
.agent-attachments {
  padding: 12px 12px 0;
  :deep(.elx-files-card) {
    max-width: 220px;
    border-radius: var(--admin-page-radius);
  }
  :deep(.elx-files-card-img),
  :deep(.elx-files-card__image-preview),
  :deep(.elx-files-card-delete-icon) {
    border-radius: var(--admin-page-radius);
  }
}
.agent-prefix-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  width: 100%;
  min-height: 36px;
}
.agent-model-selector {
  display: inline-flex;
  flex: 0 1 auto;
  align-items: center;
  gap: 6px;
  width: clamp(180px, 18vw, 220px);
  min-width: 0;
  height: 34px;
  padding: 0 10px;
  overflow: hidden;
  box-sizing: border-box;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    border-color 0.2s ease,
    background-color 0.2s ease;
  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-5);
  }
  &:disabled {
    cursor: not-allowed;
    opacity: 0.55;
  }
  &__spark {
    flex-shrink: 0;
    font-size: 15px;
  }
  &__label {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    font-size: 14px;
    font-weight: 500;
    line-height: 20px;
    text-align: left;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  &__arrow {
    flex-shrink: 0;
    font-size: 14px;
  }
}
.agent-sender-actions {
  display: inline-flex;
  gap: 8px;
  align-items: center;
}
.agent-icon-button,
.agent-send-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    border-color 0.2s ease,
    background-color 0.2s ease,
    opacity 0.2s ease;
  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary-light-5);
  }
  &:disabled {
    cursor: not-allowed;
    opacity: 0.55;
  }
}
.agent-send-button {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-5);
}
.agent-icon-button.is-active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-5);
}
.agent-action-text {
  font-size: 12px;
  line-height: 18px;
  color: var(--admin-page-text-secondary);
}
.is-loading {
  animation: rotating 1s linear infinite;
}
.agent-file-input {
  display: none;
}
:global(.agent-sender-popover) {
  padding: 0 !important;
  border-radius: var(--admin-page-radius) !important;
}
.agent-popover-card {
  padding: 14px;
}
.agent-popover-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
}
.agent-popover-desc {
  margin-top: 6px;
  font-size: 12px;
  line-height: 20px;
  color: var(--admin-page-text-secondary);
}
.agent-popover-action {
  width: 100%;
  height: 36px;
  margin-top: 12px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-color-primary);
  cursor: pointer;
  background: var(--el-color-primary-light-9);
  border: 0;
  border-radius: var(--admin-page-radius);
  &:disabled {
    cursor: not-allowed;
    opacity: 0.55;
  }
}
:global(.agent-sender-popover.el-popover) {
  box-shadow: 0 14px 36px rgb(15 23 42 / 10%);
}
:global(.agent-model-popper.el-popover) {
  padding: 0 !important;
  border-radius: var(--admin-page-radius) !important;
  box-shadow: 0 14px 36px rgb(15 23 42 / 16%) !important;
}
.agent-model-panel {
  padding: 10px;
  &__list {
    max-height: min(52vh, 420px);
    overflow-y: auto;
    scrollbar-width: thin;
    scrollbar-color: var(--el-border-color-darker) transparent;
  }
  &__head {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    padding: 4px 6px 10px;
  }
  &__title {
    font-size: 14px;
    font-weight: 700;
    color: var(--admin-page-text-primary);
  }
  &__meta {
    font-size: 11px;
    color: var(--el-text-color-placeholder);
  }
  &__group {
    padding: 8px 6px 6px;
    font-size: 12px;
    color: var(--admin-page-text-secondary);
  }
  &__manage {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    width: 100%;
    height: 36px;
    margin-top: 6px;
    padding: 0 6px;
    font-size: 12.5px;
    font-weight: 600;
    color: var(--el-color-primary);
    cursor: pointer;
    background: transparent;
    border: 0;
    border-top: 1px solid var(--el-border-color-lighter);
    border-radius: 0;
    &:hover {
      color: var(--el-color-primary-light-3);
    }
  }
  &__manage-arrow {
    margin-left: auto;
  }
  &--empty {
    padding: 20px 10px;
    font-size: 13px;
    color: var(--admin-page-text-secondary);
    text-align: center;
  }
}
.agent-model-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 42px;
  margin-bottom: 2px;
  padding: 0 10px;
  font-size: 13.5px;
  color: var(--admin-page-text-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--admin-page-radius);
  &:hover {
    background: var(--el-fill-color-light);
  }
  &.is-selected {
    font-weight: 600;
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
  &__name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  &__raw {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    font-size: 12px;
    font-weight: 400;
    color: var(--el-text-color-placeholder);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  &__check {
    flex-shrink: 0;
    font-size: 15px;
  }
}

@media screen and (width <= 768px) {
  .agent-prefix-actions {
    gap: 8px;
  }
  .agent-model-selector {
    width: clamp(128px, calc(100vw - 220px), 180px);
  }
  .agent-action-text {
    display: none;
  }
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
