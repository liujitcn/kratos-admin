<template>
  <div class="agent-sender-wrap">
    <BaseXSender
      ref="senderRef"
      class="agent-sender"
      variant="updown"
      submit-type="enter"
      placeholder="输入任何问题，或上传附件后继续提问"
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
              <button class="agent-icon-button" type="button" :disabled="sending || uploading" aria-label="上传附件">
                <el-icon :class="{ 'is-loading': uploading }">
                  <Loading v-if="uploading" />
                  <Paperclip v-else />
                </el-icon>
              </button>
            </template>
            <div class="agent-popover-card">
              <div class="agent-popover-title">上传附件</div>
              <div class="agent-popover-desc">可补充图片或文本类附件，发送后用于当前问题分析。</div>
              <button class="agent-popover-action" type="button" :disabled="sending || uploading" @click="handleSelectAttachment">
                {{ uploading ? "上传中..." : "选择本地文件" }}
              </button>
            </div>
          </el-popover>

          <span class="agent-action-text">{{ actionHintText }}</span>
        </div>
      </template>

      <template #action-list>
        <div class="agent-sender-actions">
          <el-tooltip :content="recording ? '停止语音输入' : '语音输入'" placement="top">
            <button
              class="agent-icon-button"
              :class="{ 'is-active': recording }"
              type="button"
              :disabled="sending"
              :aria-pressed="recording"
              :aria-label="recording ? '停止语音输入' : '语音输入'"
              @click="handleToggleRecord"
            >
              <el-icon :class="{ 'is-loading': recording }">
                <Loading v-if="recording" />
                <Microphone v-else />
              </el-icon>
            </button>
          </el-tooltip>
          <el-tooltip content="发送" placement="top">
            <button
              class="agent-send-button"
              type="button"
              :disabled="sending || uploading || recording || isSubmitDisabled"
              aria-label="发送"
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
import { computed, ref, watch } from "vue";
import { Attachments, useRecord, XSender as BaseXSender } from "vue-element-plus-x";
import type { FilesCardProps } from "vue-element-plus-x/types/FilesCard";
import { Loading, Microphone, Paperclip, Promotion } from "@element-plus/icons-vue";
import { ElMessage } from "element-plus";
import { defFileService } from "@/api/base/file";
import type { AiAttachment } from "@/rpc/base/v1/ai_session";
import type { SubmitPayload } from "../types";
import { buildAIAttachmentFileCard } from "../attachment";

/** 语音识别错误的最小字段，兼容浏览器事件和不支持错误。 */
type RecordError = {
  code?: number;
  error?: string;
  message?: string;
};

const props = defineProps<{
  /** 消息发送加载状态。 */
  sending: boolean;
}>();

const emit = defineEmits<{
  /** 提交输入器内容。 */
  submit: [payload: SubmitPayload];
}>();

const senderRef = ref<InstanceType<typeof BaseXSender>>();
const fileInputRef = ref<HTMLInputElement>();
const inputText = ref("");
const selectedAttachments = ref<AiAttachment[]>([]);
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

const {
  loading: recording,
  start: startRecord,
  stop: stopRecord,
  value: recordText
} = useRecord({
  onEnd: handleRecordEnd,
  onError: handleRecordError
});

const actionHintText = computed(() => {
  if (recording.value) return "正在识别语音...";
  if (uploading.value) return "附件上传中...";
  if (selectedAttachments.value.length) return `已选 ${selectedAttachments.value.length} 个附件`;
  return "可上传附件";
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
    text: trimmedText || "请结合附件内容继续分析",
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
}

/** 处理语音识别失败，提示浏览器不支持或麦克风授权异常。 */
function handleRecordError(error: RecordError) {
  const message = resolveRecordErrorMessage(error);
  ElMessage.warning(message);
}

/** 将识别文本与识别前内容合并后写回输入器。 */
function setRecordText(recordText: string) {
  const normalizedRecordText = normalizeRecordText(recordText);
  if (!normalizedRecordText) return;
  senderRef.value?.setText(normalizedRecordText);
  inputText.value = normalizedRecordText;
}

/** 标准化语音识别文本，压掉多余空白。 */
function normalizeRecordText(recordText: string) {
  return collapseCumulativeRecordText(recordText.replace(/\s+/g, " ").trim());
}

/** 压缩 useRecord 在中文连续识别时返回的前缀累计文本。 */
function collapseCumulativeRecordText(text: string) {
  if (text.length < 4) return text;
  for (let candidateLength = 2; candidateLength <= Math.floor(text.length / 2); candidateLength++) {
    const candidate = text.slice(-candidateLength);
    const matchResult = matchCumulativeRecordCandidate(text, candidate);
    if (matchResult.matched && (matchResult.hasPartial || matchResult.segmentCount >= 3)) {
      return candidate;
    }
  }
  return text;
}

/** 判断文本是否由同一识别结果的前缀片段累计组成。 */
function matchCumulativeRecordCandidate(text: string, candidate: string) {
  let index = 0;
  let segmentCount = 0;
  let hasPartial = false;
  while (index < text.length) {
    let matchedLength = 0;
    const maxLength = Math.min(candidate.length, text.length - index);
    for (let length = maxLength; length >= 1; length--) {
      if (candidate.startsWith(text.slice(index, index + length))) {
        matchedLength = length;
        break;
      }
    }
    if (!matchedLength) return { matched: false, segmentCount, hasPartial };
    if (matchedLength < candidate.length) hasPartial = true;
    segmentCount++;
    index += matchedLength;
  }
  return { matched: true, segmentCount, hasPartial };
}

/** 根据浏览器语音识别错误类型生成用户可理解的提示。 */
function resolveRecordErrorMessage(error: RecordError) {
  if (error.code === -1) return "当前浏览器不支持语音识别";
  const errorName = error.error ?? "";
  if (["not-allowed", "service-not-allowed", "permission-denied"].includes(errorName)) return "麦克风授权失败，请检查浏览器权限";
  return "语音识别失败，请稍后重试";
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
      ElMessage.success(`已上传 ${response.files.length} 个附件`);
    }
  } catch {
    ElMessage.error("附件上传失败");
  } finally {
    uploading.value = false;
    resetFileInput();
  }
}

/** 过滤不符合数量和大小约束的附件。 */
function filterUploadFiles(files: File[]) {
  const remainingCount = maxAttachmentCount - selectedAttachments.value.length;
  if (remainingCount <= 0) {
    ElMessage.warning(`最多上传 ${maxAttachmentCount} 个附件`);
    return [];
  }
  const validFiles = files.filter(file => {
    if (!isAcceptedAttachmentFile(file)) {
      ElMessage.warning(`附件「${file.name}」暂不支持，已跳过`);
      return false;
    }
    if (file.size > maxAttachmentSize) {
      ElMessage.warning(`附件「${file.name}」超过 ${maxAttachmentSizeMB}MB，已跳过`);
      return false;
    }
    return true;
  });
  if (validFiles.length > remainingCount) {
    ElMessage.warning(`最多还能选择 ${remainingCount} 个附件，已自动截取`);
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

/** 按 useRecord 文档监听识别文本，并同步到 XSender。 */
watch(
  recordText,
  value => {
    setRecordText(value);
  },
  { deep: true }
);

</script>

<style scoped lang="scss">
.agent-sender-wrap {
  width: 100%;
}
.agent-sender {
  :deep(.elx-x-sender) {
    border-radius: var(--admin-page-radius);
    box-shadow: none;
  }
  :deep(.elx-x-sender__content) {
    padding: 8px 10px 10px;
    background: var(--admin-page-card-bg);
    border: 1px solid var(--admin-page-card-border);
    border-radius: var(--admin-page-radius);
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
  gap: 10px;
  align-items: center;
  min-height: 36px;
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

@media screen and (width <= 768px) {
  .agent-prefix-actions {
    gap: 8px;
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
