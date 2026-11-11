<template>
  <section class="ai-file-panel">
    <div class="file-toolbar">
      <el-tabs v-model="activeView" class="file-tabs" :aria-label="t('system.ai.workspace.file_views')" @tab-change="changeView">
        <el-tab-pane v-for="tab in tabs" :key="tab.view" :name="tab.view">
          <template #label>
            <span class="file-tab-label">
              <span>{{ tab.label }}</span>
              <span v-if="totals[tab.view] !== undefined" class="file-tab-count">{{ totals[tab.view] }}</span>
            </span>
          </template>
        </el-tab-pane>
      </el-tabs>
      <el-input
        v-model="keyword"
        class="file-search"
        clearable
        :placeholder="t('system.ai.workspace.search_files')"
        @keyup.enter="search"
        @clear="search"
      >
        <template #prefix
          ><el-icon><Search /></el-icon
        ></template>
      </el-input>
    </div>

    <el-alert v-if="error" class="file-error" :title="error" type="error" :closable="false" show-icon />
    <div v-loading="loading" class="file-list">
      <div class="file-list-head">
        <span>{{ t("system.ai.workspace.file_name") }}</span>
        <span>{{ t("system.ai.workspace.file_space") }}</span>
        <span>{{ t("system.ai.workspace.file_size") }}</span>
        <span />
      </div>
      <el-empty v-if="!loading && !files.length" :description="t('system.ai.workspace.no_files')" />
      <div v-for="file in files" :key="file.id" class="file-row">
        <div class="file-name-cell">
          <span class="file-type-icon" :class="fileTypeClass(file)"
            ><el-icon><component :is="fileIcon(file)" /></el-icon
          ></span>
          <span class="file-name-content">
            <strong :title="file.file_name">{{ file.file_name }}</strong>
            <small>{{ sourceLabel(file) }}</small>
          </span>
        </div>
        <span class="file-space">{{ t("system.ai.workspace.personal_space") }}</span>
        <span class="file-size">{{ formatFileSize(file.size) }}</span>
        <div class="file-actions">
          <el-tooltip
            v-if="activeView !== AiFileView.AI_FILE_VIEW_TRASHED"
            :content="t('common.action.download')"
            placement="top"
          >
            <el-button text circle :aria-label="t('common.action.download')" @click="downloadFile(file)">
              <el-icon><Download /></el-icon>
            </el-button>
          </el-tooltip>
          <el-dropdown trigger="click" @command="command => handleCommand(command, file)">
            <el-button text circle :aria-label="t('system.ai.workspace.file_more_actions')">
              <el-icon><MoreFilled /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item v-if="activeView !== AiFileView.AI_FILE_VIEW_TRASHED" command="rename" :icon="EditPen">
                  {{ t("system.ai.workspace.rename_file") }}
                </el-dropdown-item>
                <el-dropdown-item v-if="activeView !== AiFileView.AI_FILE_VIEW_TRASHED" command="trash" :icon="Delete">
                  {{ t("system.ai.workspace.move_to_trash") }}
                </el-dropdown-item>
                <el-dropdown-item v-else command="restore" :icon="RefreshLeft">
                  {{ t("system.ai.workspace.restore_file") }}
                </el-dropdown-item>
                <el-dropdown-item v-if="activeView === AiFileView.AI_FILE_VIEW_TRASHED" command="delete" :icon="Delete" divided>
                  {{ t("system.ai.workspace.delete_permanently") }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>

    <el-pagination
      v-if="total > pageSize"
      v-model:current-page="page"
      class="file-pagination"
      layout="prev, pager, next"
      :page-size="pageSize"
      :total="total"
      @current-change="loadFiles"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  Delete,
  Document,
  Download,
  EditPen,
  Headset,
  MoreFilled,
  Picture,
  RefreshLeft,
  Search,
  VideoCamera
} from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import { defFileService } from "@liujitcn/kratos-admin-core/api/base/v1/file";
import { defAiFileService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_file";
import { AiFileView, type AiFile } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_file";

defineOptions({ name: "AiFilePanel" });

const pageSize = 20;
const activeView = ref(AiFileView.AI_FILE_VIEW_ALL);
const keyword = ref("");
const page = ref(1);
const files = ref<AiFile[]>([]);
const totals = ref<Record<number, number>>({});
const total = computed(() => totals.value[activeView.value] ?? files.value.length);
const loading = ref(false);
const error = ref("");
const tabs = computed(() => [
  { view: AiFileView.AI_FILE_VIEW_ALL, label: t("system.ai.workspace.all_files") },
  { view: AiFileView.AI_FILE_VIEW_UPLOADED, label: t("system.ai.workspace.uploaded_files") },
  { view: AiFileView.AI_FILE_VIEW_GENERATED, label: t("system.ai.workspace.generated_files") },
  { view: AiFileView.AI_FILE_VIEW_TRASHED, label: t("system.ai.workspace.trash") }
]);

onMounted(() => {
  void Promise.all([loadFiles(), loadTotals()]);
});

/** 切换文件分类。 */
function changeView() {
  page.value = 1;
  void loadFiles();
}

/** 按当前关键词重新查询。 */
function search() {
  page.value = 1;
  void loadFiles();
}

/** 请求当前分类的 AI 文件。 */
async function loadFiles() {
  loading.value = true;
  error.value = "";
  try {
    const response = await defAiFileService.PageAiFile({
      view: activeView.value,
      keyword: keyword.value,
      page_num: page.value,
      page_size: pageSize
    });
    files.value = response.files ?? [];
    totals.value[activeView.value] = response.total;
  } catch (loadError) {
    error.value = loadError instanceof Error ? loadError.message : t("system.ai.workspace.file_load_failed");
  } finally {
    loading.value = false;
  }
}

/** 查询四个分类的数量。 */
async function loadTotals() {
  try {
    const responses = await Promise.all(
      tabs.value.map(tab =>
        defAiFileService
          .PageAiFile({ view: tab.view, keyword: "", page_num: 1, page_size: 1 })
          .then(response => [tab.view, response.total] as const)
      )
    );
    totals.value = Object.fromEntries(responses);
  } catch {
    // 列表接口仍可独立展示，数量失败时保留已知数量。
  }
}

/** 执行文件菜单动作。 */
async function handleCommand(command: string | number | object, file: AiFile) {
  try {
    if (command === "rename") {
      const result = await ElMessageBox.prompt(
        t("system.ai.workspace.rename_file_prompt"),
        t("system.ai.workspace.rename_file"),
        {
          inputValue: file.file_name,
          inputPattern: /^(?!\s*$).{1,255}$/,
          inputErrorMessage: t("system.ai.workspace.file_name_invalid"),
          confirmButtonText: t("common.action.confirm"),
          cancelButtonText: t("common.action.cancel")
        }
      );
      await defAiFileService.RenameAiFile({ id: file.id, file_name: result.value });
      ElMessage.success(t("system.ai.workspace.file_renamed"));
    } else if (command === "trash") {
      await ElMessageBox.confirm(
        t("system.ai.workspace.move_to_trash_confirm", { name: file.file_name }),
        t("common.title.warning"),
        {
          type: "warning",
          confirmButtonText: t("common.action.confirm"),
          cancelButtonText: t("common.action.cancel")
        }
      );
      await defAiFileService.TrashAiFile({ id: file.id });
      ElMessage.success(t("system.ai.workspace.file_trashed"));
    } else if (command === "restore") {
      await defAiFileService.RestoreAiFile({ id: file.id });
      ElMessage.success(t("system.ai.workspace.file_restored"));
    } else if (command === "delete") {
      await ElMessageBox.confirm(
        t("system.ai.workspace.delete_file_confirm", { name: file.file_name }),
        t("common.title.warning"),
        {
          type: "warning",
          confirmButtonText: t("common.action.confirm"),
          cancelButtonText: t("common.action.cancel")
        }
      );
      await defAiFileService.DeleteAiFile({ id: file.id });
      ElMessage.success(t("system.ai.workspace.file_deleted"));
    } else {
      return;
    }
    await Promise.all([loadFiles(), loadTotals()]);
  } catch {
    // 取消确认和请求层错误都不改变当前列表状态。
  }
}

/** 下载活动 AI 文件。 */
async function downloadFile(file: AiFile) {
  await defFileService.DownloadFile(file.link_url, file.file_name);
}

/** 返回文件来源文案。 */
function sourceLabel(file: AiFile) {
  return file.source_type === "generated" ? t("system.ai.workspace.generated_file") : t("system.ai.workspace.uploaded_file");
}

/** 返回适合文件内容的图标。 */
function fileIcon(file: AiFile) {
  const mimeType = file.mime_type.toLowerCase();
  if (mimeType.startsWith("image/")) return Picture;
  if (mimeType.startsWith("video/")) return VideoCamera;
  if (mimeType.startsWith("audio/")) return Headset;
  return Document;
}

/** 返回文件图标颜色分类。 */
function fileTypeClass(file: AiFile) {
  const mimeType = file.mime_type.toLowerCase();
  if (mimeType.startsWith("image/")) return "is-image";
  if (mimeType.startsWith("video/")) return "is-video";
  if (mimeType.startsWith("audio/")) return "is-audio";
  return "is-document";
}

/** 格式化文件大小。 */
function formatFileSize(size: number) {
  if (!size) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let value = size;
  let index = 0;
  while (value >= 1024 && index < units.length - 1) {
    value /= 1024;
    index += 1;
  }
  return `${value.toFixed(index === 0 ? 0 : 1)} ${units[index]}`;
}
</script>

<style scoped lang="scss">
.ai-file-panel {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: auto;
  color: var(--admin-page-text-primary);
}

.file-toolbar {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin: 0 32px;
}

.file-tabs {
  flex: 1;
  min-width: 0;
}

.file-tabs :deep(.el-tabs__header) {
  margin-bottom: 0;
}

.file-tab-label {
  display: inline-flex;
  align-items: center;
  min-height: 1em;
  line-height: 1;
}

.file-tab-count {
  margin-left: 8px;
  font-size: var(--el-font-size-small);
  line-height: 1;
  color: var(--admin-page-text-secondary);
}

.file-search {
  width: min(360px, 35vw);
  margin-bottom: 10px;
}

.file-error {
  margin: 16px 32px 0;
}

.file-list {
  flex: 1;
  min-height: 240px;
  margin: 16px 32px 0;
}

.file-list-head,
.file-row {
  display: grid;
  grid-template-columns: minmax(260px, 1.7fr) minmax(120px, 1fr) 120px 88px;
  align-items: center;
  gap: 16px;
}

.file-list-head {
  min-height: 48px;
  padding: 0 16px;
  color: var(--admin-page-text-secondary);
  background: var(--el-fill-color-lighter);
  border-bottom: 1px solid var(--admin-page-divider-strong);
}

.file-row {
  min-height: 84px;
  padding: 0 16px;
  border-bottom: 1px solid var(--admin-page-divider);
}

.file-row:hover {
  background: var(--el-fill-color-lighter);
}

.file-name-cell {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 14px;
}

.file-type-icon {
  display: grid;
  width: 32px;
  height: 38px;
  flex: 0 0 auto;
  place-items: center;
  border-radius: 5px;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  font-size: 20px;
}

.file-type-icon.is-image {
  color: var(--el-color-success);
  background: var(--el-color-success-light-9);
}

.file-type-icon.is-video {
  color: var(--el-color-warning);
  background: var(--el-color-warning-light-9);
}

.file-type-icon.is-audio {
  color: var(--el-color-danger);
  background: var(--el-color-danger-light-9);
}

.file-name-content {
  display: grid;
  min-width: 0;
  gap: 6px;
}

.file-name-content strong {
  overflow: hidden;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-name-content small,
.file-space,
.file-size {
  color: var(--admin-page-text-secondary);
}

.file-actions {
  display: flex;
  justify-content: flex-end;
}

.file-pagination {
  justify-content: center;
  margin: 20px 0 24px;
}

@media (max-width: 768px) {
  .file-toolbar {
    display: grid;
    gap: 8px;
    margin: 0 16px;
  }

  .file-search {
    width: 100%;
    margin: 0 0 10px;
  }

  .file-list {
    margin: 12px 16px 0;
  }

  .file-list-head {
    display: none;
  }

  .file-row {
    grid-template-columns: minmax(0, 1fr) auto;
    min-height: 72px;
    padding: 10px 0;
  }

  .file-space {
    display: none;
  }

  .file-size {
    grid-column: 1;
    grid-row: 2;
    margin-left: 46px;
    font-size: 12px;
  }

  .file-actions {
    grid-column: 2;
    grid-row: 1 / span 2;
  }
}
</style>
