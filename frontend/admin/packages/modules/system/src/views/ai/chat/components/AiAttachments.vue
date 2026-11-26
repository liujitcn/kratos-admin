<template>
  <div v-if="files.length || issues?.length" class="ai-attachments" :class="{ 'is-generated': generated }">
    <ul class="ai-attachments__list">
      <li v-for="file in files" :key="attachmentKey(file)" class="ai-attachments__item">
        <div class="ai-attachments__row">
          <component :is="attachmentIcon(file)" class="ai-attachments__icon" :size="16" aria-hidden="true" />
          <div class="ai-attachments__label">
            <button
              v-if="!generated && file.url"
              type="button"
              class="ai-attachments__name ai-attachments__button"
              :title="file.name"
              @click="isAIImageAttachment(file) ? togglePreview(file) : download(file)"
            >
              {{ file.name }}
            </button>
            <strong v-else class="ai-attachments__name" :title="file.name">{{ file.name }}</strong>
            <small v-if="fileIssues[attachmentKey(file)]" role="status">{{ t(fileIssues[attachmentKey(file)]!) }}</small>
            <small v-else-if="generated">{{ attachmentMeta(file) }}</small>
          </div>
          <div class="ai-attachments__actions">
            <button
              v-if="generated && previewKind(file) && !fileIssues[attachmentKey(file)]"
              type="button"
              class="ai-attachments__button"
              :aria-expanded="preview?.id === attachmentKey(file)"
              @click="togglePreview(file)"
            >
              {{ t("system.ai.chat.file.preview") }}
              <ChevronDown :size="13" aria-hidden="true" />
            </button>
            <button
              v-if="file.url && (generated || isAIImageAttachment(file)) && !fileIssues[attachmentKey(file)]"
              type="button"
              class="ai-attachments__button is-icon"
              :aria-label="t('system.ai.chat.file.download') + ' ' + file.name"
              :title="t('system.ai.chat.file.download')"
              @click="download(file)"
            >
              <Download :size="15" aria-hidden="true" />
            </button>
            <button v-if="canReexport && fileIssues[attachmentKey(file)]" type="button" class="ai-attachments__button" @click="emit('reexport', file.name)">
              {{ t("system.ai.chat.file.reexport") }}
            </button>
            <button
              v-if="removable"
              type="button"
              class="ai-attachments__button is-icon"
              :aria-label="t('system.ai.chat.file.remove', { name: file.name })"
              :title="t('system.ai.chat.file.remove', { name: file.name })"
              @click="emit('remove', file)"
            >
              <X :size="14" aria-hidden="true" />
            </button>
          </div>
        </div>

        <div v-if="preview?.id === attachmentKey(file)" class="ai-attachments__preview" :aria-busy="preview.loading">
          <p v-if="preview.loading" role="status">{{ t("system.ai.chat.status.loading") }}</p>
          <p v-else-if="preview.failed" role="status">{{ t("system.base.file.message.preview_failed") }}</p>
          <template v-else-if="generated">
            <img v-if="preview.kind === 'image'" :src="preview.url" :alt="file.name" />
            <iframe v-else-if="preview.kind === 'pdf'" :src="preview.url" :title="file.name" />
            <pre v-else>{{ preview.text }}</pre>
            <p v-if="preview.truncated">{{ t("system.base.file.message.preview_download_hint") }}</p>
          </template>
          <el-image-viewer v-else-if="preview.url" :url-list="[preview.url]" teleported @close="resetPreview" />
        </div>
      </li>
      <li v-for="(issue, index) in issues" :key="`issue:${index}`" class="ai-attachments__item">
        <div class="ai-attachments__row">
          <FileText class="ai-attachments__icon" :size="16" aria-hidden="true" />
          <div class="ai-attachments__label">
            <strong class="ai-attachments__name" :title="issue.name">{{ issue.name }}</strong>
            <small role="status">{{ t(temporaryFileStatus(issue.expires_at)) }}</small>
          </div>
          <button v-if="canReexport" type="button" class="ai-attachments__button" @click="emit('reexport', issue.name)">
            {{ t("system.ai.chat.file.reexport") }}
          </button>
        </div>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref, watch } from "vue";
import { ChevronDown, Download, FileImage, FileSpreadsheet, FileText, X } from "@lucide/vue";
import { t } from "@liujitcn/kratos-admin-core";
import { defFileService } from "@liujitcn/kratos-admin-core/api/base/v1/file";
import { ElMessage } from "element-plus";
import type { AiAttachment } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";
import { getAIAttachmentExtension, getAIImageMimeType, isAIImageAttachment } from "../attachment";
import type { AIFlowBlock } from "../types";

/** 可直接在消息或图片查看器中展示的附件类型。 */
type PreviewKind = "image" | "pdf" | "text";

/** 当前处于展开状态的附件预览。 */
type AttachmentPreview = {
  id: string;
  kind: PreviewKind;
  loading: boolean;
  failed?: boolean;
  url?: string;
  text?: string;
  truncated?: boolean;
};

const props = defineProps<{
  /** 用户附件、草稿附件或 AI 生成文件。 */
  files: AiAttachment[];
  /** 允许从草稿移除附件。 */
  removable?: boolean;
  /** 生成文件使用带元信息和展开预览的纵向文件卡。 */
  generated?: boolean;
  /** 服务端未能转存的生成文件及签名有效期。 */
  issues?: AIFlowBlock[];
  /** 允许将当前文件的重新导出要求填入草稿。 */
  canReexport?: boolean;
}>();

const emit = defineEmits<{
  /** 删除草稿中的对应文件。 */
  remove: [file: AiAttachment];
  /** 准备指定文件的重新导出请求。 */
  reexport: [name: string];
}>();
const preview = ref<AttachmentPreview>();
const fileIssues = ref<Partial<Record<string, string>>>({});
let previewRevision = 0;
const PREVIEW_TEXT_LIMIT = 32 * 1024;

watch(() => props.files, files => {
  if (preview.value && !files.some(file => attachmentKey(file) === preview.value?.id)) resetPreview();
});
onBeforeUnmount(resetPreview);

/** 获取历史附件与新附件可共用的稳定键。 */
function attachmentKey(file: AiAttachment) {
  return file.id || file.url || file.name;
}

/** 根据常见文件类型选择列表图标。 */
function attachmentIcon(file: AiAttachment) {
  if (isAIImageAttachment(file)) return FileImage;
  return ["csv", "xls", "xlsx"].includes(getAIAttachmentExtension(file.name)) ? FileSpreadsheet : FileText;
}

/** 组合生成文件的扩展名与大小。 */
function attachmentMeta(file: AiAttachment) {
  const bytes = Number(file.size ?? 0);
  const extension = getAIAttachmentExtension(file.name).toUpperCase();
  let size = "";
  if (Number.isFinite(bytes) && bytes > 0) {
    if (bytes < 1024) size = `${bytes} B`;
    else if (bytes < 1024 * 1024) size = `${(bytes / 1024).toFixed(1)} KB`;
    else size = `${(bytes / 1024 / 1024).toFixed(1)} MB`;
  }
  return [extension, size].filter(Boolean).join(" · ");
}

/** 区分已知过期的临时产物与尚未转存的文件。 */
function temporaryFileStatus(expiresAt: unknown) {
  return typeof expiresAt === "string" && Date.parse(expiresAt) <= Date.now()
    ? "system.ai.chat.file.expired"
    : "system.ai.chat.file.not_saved";
}

/** 判定浏览器可安全渲染的预览类型。 */
function previewKind(file: AiAttachment): PreviewKind | undefined {
  if (!file.url) return;
  if (isAIImageAttachment(file)) return "image";
  if (!props.generated) return;
  const extension = getAIAttachmentExtension(file.name);
  if (extension === "pdf") return "pdf";
  if (["csv", "txt", "md", "markdown", "log", "json", "xml"].includes(extension)) return "text";
}

/** 受保护图片和生成文件使用文件接口，较大的文本仅展示开头。 */
async function togglePreview(file: AiAttachment) {
  const id = attachmentKey(file);
  const wasOpen = preview.value?.id === id;
  resetPreview();
  const kind = previewKind(file);
  if (wasOpen || !kind) return;

  const requestId = previewRevision;
  preview.value = { id, kind, loading: true };
  try {
    const blob = await defFileService.GetFileBlob(file.url, file.name);
    if (requestId !== previewRevision) return;
    if (kind === "text") {
      const text = await blob.slice(0, PREVIEW_TEXT_LIMIT).text();
      if (requestId === previewRevision) preview.value = { id, kind, loading: false, text, truncated: blob.size > PREVIEW_TEXT_LIMIT };
    } else {
      const mimeType = kind === "pdf" ? "application/pdf" : getAIImageMimeType(file);
      const source = blob.slice(0, blob.size, mimeType);
      preview.value = { id, kind, loading: false, url: URL.createObjectURL(source) };
    }
  } catch (error) {
    if (requestId !== previewRevision) return;
    if (recordUnavailableFile(file, error)) resetPreview();
    else preview.value = { id, kind, loading: false, failed: true };
  }
}

/** 收起预览并释放浏览器对象地址，忽略已经失效的请求。 */
function resetPreview() {
  previewRevision++;
  if (preview.value?.url) URL.revokeObjectURL(preview.value.url);
  preview.value = undefined;
}

/** 以服务端存储路径调用鉴权下载，无需对外公开附件地址。 */
async function download(file: AiAttachment) {
  if (!props.generated) {
    await defFileService.DownloadFile(file.url, file.name);
    return;
  }
  try {
    const url = URL.createObjectURL(await defFileService.GetFileBlob(file.url, file.name));
    const link = document.createElement("a");
    link.href = url;
    link.download = file.name;
    link.click();
    window.setTimeout(() => URL.revokeObjectURL(url), 0);
  } catch (error) {
    if (!recordUnavailableFile(file, error)) ElMessage.error(t("system.ai.chat.file.unavailable"));
  }
}

/** 只有确切的失效响应才收起生成文件的预览和下载操作。 */
function recordUnavailableFile(file: AiAttachment, error: unknown) {
  const status = (error as { response?: { status?: number } } | undefined)?.response?.status;
  if (!props.generated || (status !== 404 && status !== 410)) return false;
  fileIssues.value[attachmentKey(file)] = status === 410 ? "system.ai.chat.file.expired" : "system.ai.chat.file.unavailable";
  return true;
}
</script>

<style scoped lang="scss">
.ai-attachments__list {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  padding: 0;
  margin: 0;
  list-style: none;
}

.ai-attachments__item {
  min-width: 0;
  max-width: min(100%, 240px);
}

.ai-attachments__row {
  display: flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
  padding: 2px 5px;
  color: var(--el-text-color-regular);
  font-size: 12px;
  background: var(--el-fill-color-light);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius);
}

.ai-attachments__icon {
  flex: none;
  color: var(--el-text-color-secondary);
}

.ai-attachments__label {
  flex: 1;
  min-width: 0;

  small {
    display: block;
    margin-top: 2px;
    color: var(--el-text-color-secondary);
  }
}

.ai-attachments__name {
  display: block;
  max-width: 100%;
  overflow: hidden;
  font-weight: 500;
  text-align: start;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-attachments__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  flex: none;
  gap: 2px;

  button[aria-expanded="true"] svg {
    transform: rotate(180deg);
  }
}

.ai-attachments__button {
  min-height: 24px;
  padding: 0 3px;
  color: inherit;
  font: inherit;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: var(--admin-page-radius);

  &:not(.ai-attachments__name) {
    display: inline-flex;
    align-items: center;
    gap: 3px;
    color: var(--el-text-color-secondary);
  }

  &.is-icon {
    justify-content: center;
    width: 24px;
  }

  &:focus-visible {
    outline: 2px solid var(--el-color-primary);
    outline-offset: 1px;
  }

  &:hover {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}

.ai-attachments__preview {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  overflow-wrap: anywhere;

  p {
    margin: 4px 0 0;
  }

  img {
    display: block;
    max-width: 100%;
    max-height: 44vh;
    object-fit: contain;
  }

  iframe {
    width: 100%;
    height: 42vh;
    border: 0;
  }

  pre {
    max-height: 16rem;
    padding: 8px;
    margin: 0;
    overflow: auto;
    color: var(--el-text-color-primary);
    background: var(--el-fill-color-light);
    border-radius: var(--admin-page-radius);
  }
}

.ai-attachments.is-generated {
  .ai-attachments__list {
    flex-direction: column;
  }

  .ai-attachments__item {
    width: min(100%, 480px);
    max-width: 100%;
    background: var(--admin-page-card-bg);
    border: 1px solid var(--el-border-color);
    border-radius: var(--admin-page-radius);
  }

  .ai-attachments__row {
    gap: 8px;
    padding: 6px 8px;
    background: transparent;
    border: 0;
  }

  .ai-attachments__name {
    color: var(--el-text-color-primary);
  }

  .ai-attachments__preview {
    padding: 8px;
    border-top: 1px solid var(--el-border-color-lighter);
  }
}
</style>
