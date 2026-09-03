<template>
  <div class="table-box">
    <ProTable
      ref="proTable"
      row-key="id"
      :columns="columns"
      :request-api="requestBaseFileTable"
    />

    <ProDialog
      v-model="detailVisible"
      :title="detail?.file_name || t('system.base.file.detail')"
      width="min(960px, calc(100vw - 32px))"
      top="5vh"
      destroy-on-close
      :show-footer="false"
      @closed="resetDetail"
    >
      <template v-if="detail">
        <section class="file-preview" aria-live="polite">
          <el-skeleton v-if="previewLoading" :rows="8" animated />
          <template v-else-if="previewUrl && previewKind === 'image'">
            <img class="file-preview__image" :src="previewUrl" :alt="detail.file_name" />
          </template>
          <video v-else-if="previewUrl && previewKind === 'video'" class="file-preview__media" controls :src="previewUrl">
            {{ t("system.base.file.message.preview_unsupported") }}
          </video>
          <audio v-else-if="previewUrl && previewKind === 'audio'" class="file-preview__audio" controls :src="previewUrl">
            {{ t("system.base.file.message.preview_unsupported") }}
          </audio>
          <iframe
            v-else-if="previewUrl && previewKind === 'pdf'"
            class="file-preview__document"
            :src="previewUrl"
            :title="detail.file_name"
          />
          <pre v-else-if="previewKind === 'text' && !previewError" class="file-preview__text">{{ previewText }}</pre>
          <el-result
            v-else-if="previewError"
            class="file-preview__result"
            icon="error"
            :title="t('system.base.file.message.preview_failed')"
            :sub-title="t('system.base.file.message.preview_retry_download')"
          />
          <el-result
            v-else
            class="file-preview__result"
            icon="info"
            :title="t('system.base.file.message.preview_unsupported')"
            :sub-title="t('system.base.file.message.preview_download_hint')"
          />
        </section>

        <el-descriptions class="file-meta" :column="2" border size="small">
          <el-descriptions-item :label="t('system.base.file.field.name')" :span="2">
            <span class="file-meta__value" :title="detail.file_name">{{ detail.file_name }}</span>
          </el-descriptions-item>
          <el-descriptions-item :label="t('system.base.file.field.mime')">{{ detail.mime_type || "-" }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.base.file.field.size')">{{ formatFileSize(detail.size) }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.base.file.field.path')" :span="2">
            <code class="file-meta__value">{{ detail.link_url || `${detail.file_directory}/${detail.save_file_name}` }}</code>
          </el-descriptions-item>
          <el-descriptions-item :label="t('system.base.file.field.hash')" :span="2">
            <code class="file-meta__value">{{ detail.content_hash || "-" }}</code>
          </el-descriptions-item>
        </el-descriptions>

        <div class="file-detail__actions">
          <el-button type="primary" :icon="Download" @click="downloadDetail">
            {{ t("common.action.download") }}
          </el-button>
        </div>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { Download, Delete, View } from "@element-plus/icons-vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import { defFileService } from "@liujitcn/kratos-admin-core/api/base/v1/file";
import { defBaseFileService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_file";
import type { BaseFile, PageBaseFileRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_file";

defineOptions({ name: "BaseFile", inheritAttrs: false });

const { BUTTONS } = useAuthButtons();
const proTable = ref<ProTableInstance>();
const detailVisible = ref(false);
const detail = ref<BaseFile>();
const previewLoading = ref(false);
const previewError = ref(false);
const previewUrl = ref("");
const previewText = ref("");

/** 文件内容预览类型。 */
type PreviewKind = "image" | "video" | "audio" | "pdf" | "text" | "unsupported";

const previewKind = computed<PreviewKind>(() => (detail.value ? getPreviewKind(detail.value) : "unsupported"));

const columns = computed<ColumnProps[]>(() => [
  { prop: "file_name", label: t("system.base.file.field.name"), minWidth: 220, search: { el: "input" } },
  { prop: "extension", label: t("system.base.file.field.extension"), width: 100, search: { el: "input" } },
  { prop: "mime_type", label: t("system.base.file.field.mime"), minWidth: 180 },
  { prop: "size", label: t("system.base.file.field.size"), width: 120, render: scope => formatFileSize((scope.row as BaseFile).size) },
  { prop: "content_hash", label: t("system.base.file.field.hash"), minWidth: 220 },
  { prop: "tenant_id", label: t("system.base.file.field.tenant"), width: 100 },
  { prop: "created_at", label: t("common.field.created_at"), minWidth: 180 },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 150,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.view"),
        type: "primary",
        link: true,
        icon: View,
        hidden: () => !BUTTONS.value["base:file:info"],
        onClick: scope => handleView(scope.row as BaseFile)
      },
      {
        label: t("common.action.delete"),
        type: "danger",
        link: true,
        icon: Delete,
        hidden: () => !BUTTONS.value["base:file:delete"],
        onClick: scope => handleDelete(scope.row as BaseFile)
      }
    ]
  }
]);

/** 请求文件资产分页列表。 */
async function requestBaseFileTable(params: PageBaseFileRequest) {
  const data = await defBaseFileService.PageBaseFile(buildPageRequest(params));
  return { data: { list: data.base_files ?? [], total: data.total } };
}

/** 查看文件资产详情。 */
async function handleView(row: BaseFile) {
  detailVisible.value = true;
  detail.value = undefined;
  resetPreview();
  try {
    const loadedDetail = await defBaseFileService.GetBaseFile({ id: row.id });
    detail.value = loadedDetail;
    await loadPreview(loadedDetail);
  } catch {
    detailVisible.value = false;
  }
}

/** 加载文件内容并准备浏览器预览。 */
async function loadPreview(file: BaseFile) {
  const kind = getPreviewKind(file);
  if (kind === "unsupported") return;

  previewLoading.value = true;
  previewError.value = false;
  try {
    const blob = await defFileService.GetFileBlob(file.link_url, file.file_name);
    const previewBlob = new Blob([blob], { type: file.mime_type || blob.type || "application/octet-stream" });
    if (kind === "text") {
      previewText.value = await previewBlob.text();
    } else {
      previewUrl.value = URL.createObjectURL(previewBlob);
    }
  } catch {
    previewError.value = true;
  } finally {
    previewLoading.value = false;
  }
}

/** 下载当前详情文件。 */
async function downloadDetail() {
  if (!detail.value) return;
  await defFileService.DownloadFile(detail.value.link_url, detail.value.file_name);
}

/** 关闭详情时释放预览资源并清空状态。 */
function resetDetail() {
  detail.value = undefined;
  resetPreview();
}

/** 清理当前预览状态。 */
function resetPreview() {
  if (previewUrl.value) URL.revokeObjectURL(previewUrl.value);
  previewUrl.value = "";
  previewText.value = "";
  previewError.value = false;
  previewLoading.value = false;
}

/** 根据 MIME 类型选择安全的在线预览方式。 */
function getPreviewKind(file: BaseFile): PreviewKind {
  const mimeType = file.mime_type.toLowerCase().split(";", 1)[0];
  if (mimeType.startsWith("image/")) return "image";
  if (mimeType.startsWith("video/")) return "video";
  if (mimeType.startsWith("audio/")) return "audio";
  if (mimeType === "application/pdf") return "pdf";
  if (mimeType.startsWith("text/") || mimeType === "application/json") return "text";
  return "unsupported";
}

/** 删除文件资产。 */
async function handleDelete(row: BaseFile) {
  try {
    await ElMessageBox.confirm(
      t("system.base.file.confirm_delete", { name: row.file_name }),
      t("common.title.warning"),
      { type: "warning", confirmButtonText: t("common.action.confirm"), cancelButtonText: t("common.action.cancel") }
    );
    await defBaseFileService.DeleteBaseFile({ id: row.id });
    ElMessage.success(t("common.message.delete_success", { resource: t("system.base.file.resource") }));
    proTable.value?.getTableList();
  } catch {
    // 用户取消确认或接口失败时由请求层展示错误。
  }
}

function formatFileSize(size: number) {
  if (!size) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let value = size;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(index === 0 ? 0 : 2)} ${units[index]}`;
}
</script>

<style scoped lang="scss">
.file-preview {
  min-height: 220px;
  display: grid;
  place-items: center;
  padding: 16px;
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-lighter);
}

.file-preview__image {
  display: block;
  max-width: 100%;
  max-height: 48vh;
  object-fit: contain;
}

.file-preview__media {
  display: block;
  width: min(100%, 760px);
  max-height: 48vh;
}

.file-preview__audio {
  width: min(100%, 560px);
}

.file-preview__document {
  display: block;
  width: 100%;
  height: 48vh;
  border: 0;
}

.file-preview__text {
  width: 100%;
  max-height: 48vh;
  margin: 0;
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  text-align: left;
  color: var(--el-text-color-primary);
}

.file-preview__result {
  padding: 24px 0;
}

.file-meta {
  margin-top: 16px;
}

.file-meta__value {
  display: block;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-detail__actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>
