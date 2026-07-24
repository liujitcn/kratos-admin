<!-- 代码生成文件预览 -->
<template>
  <div class="app-container code-gen-code-preview-page">
    <el-card class="code-gen-sub-card" shadow="never">
      <div v-if="BUTTONS['tool:code-gen-table:generate']" class="code-gen-toolbar">
        <div class="code-gen-code-preview-actions">
          <el-button :icon="Clock" :disabled="!progressTaskAvailable" @click="handleOpenProgress">最近任务</el-button>
          <el-button :icon="Promotion" type="primary" :loading="generating" @click="handleGenerate">生成</el-button>
        </div>
      </div>

      <el-alert v-if="previewError" class="code-gen-code-preview-alert" :title="previewError" type="warning" :closable="false" />
      <CodePreviewPane v-if="loading || files.length" class="code-gen-code-preview-table" :files="files" :loading="loading" />
      <el-empty v-else class="code-gen-code-preview-empty" description="暂无生成文件，请先确认表配置和 Proto 接口配置后再生成或预览" />
    </el-card>

    <CodeGenProgressDialog
      v-model="progressDialogVisible"
      :task-id="progressTaskId"
      @update:model-value="handleProgressDialogVisibleChange"
      @completed="handleProgressCompleted"
      @unavailable="handleProgressUnavailable"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { Clock, Promotion } from "@element-plus/icons-vue";
import { useRoute } from "vue-router";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { useTabsStore } from "@/stores/modules/tabs";
import { defCodeGenService } from "@/api/system/code_gen";
import { defCodeGenTableService } from "@/api/system/code_gen_table";
import type { CodeGenPreviewFile } from "@/rpc/system/admin/v1/code_gen";
import type { CodeGenTableForm } from "@/rpc/system/admin/v1/code_gen_table";
import { CodeGenTableStatus } from "@/rpc/system/common/v1/enum";
import CodeGenProgressDialog from "../components/CodeGenProgressDialog.vue";
import CodePreviewPane from "../components/CodePreviewPane.vue";

defineOptions({
  name: "CodeGenCodePreview",
  inheritAttrs: false
});

const codeGenTaskStorageKey = "code-gen-progress-task-id";
const codeGenProgressDialogVisibleStorageKey = "code-gen-progress-dialog-visible";
const codeGenProgressSelectedTableIdsStorageKey = "code-gen-progress-selected-table-ids";
const codeGenStatusDisabled = CodeGenTableStatus.DISABLED_CGTS;

const route = useRoute();
const tabsStore = useTabsStore();
const { BUTTONS } = useAuthButtons();
const table = ref<CodeGenTableForm>();
const files = ref<CodeGenPreviewFile[]>([]);
const loading = ref(false);
const previewError = ref("");
const progressTaskId = ref(typeof window === "undefined" ? "" : (window.sessionStorage.getItem(codeGenTaskStorageKey) ?? ""));
const progressDialogVisible = ref(
  !!progressTaskId.value &&
    typeof window !== "undefined" &&
    window.sessionStorage.getItem(codeGenProgressDialogVisibleStorageKey) === "true"
);
const progressTaskAvailable = ref(!!progressTaskId.value);
const generating = ref(!!progressTaskId.value);

/** 当前代码生成表配置 ID。 */
const tableId = computed(() => {
  const value = route.params.tableId;
  const id = Number(Array.isArray(value) ? value[0] : value);
  return Number.isFinite(id) && id > 0 ? id : 0;
});

/** 当前代码预览页标题。 */
const pageTitle = computed(() => table.value?.comment || table.value?.name || "代码预览");

// 路由生成对象变化时重新载入对应代码预览。
watch(
  tableId,
  () => {
    void loadCodePreview();
  },
  { immediate: true }
);

/** 加载当前表配置与固定项目路径下的代码预览。 */
async function loadCodePreview() {
  table.value = undefined;
  files.value = [];
  previewError.value = "";
  if (!tableId.value) return;
  loading.value = true;
  try {
    const currentTable = await defCodeGenTableService.GetCodeGenTable({ id: tableId.value });
    table.value = currentTable;
    try {
      const preview = await defCodeGenService.PreviewCodeGen({ table_id: tableId.value, output_paths: undefined });
      files.value = preview.files ?? [];
      if (!files.value.length) previewError.value = "当前没有可预览的生成文件，请先检查是否已配置生成字段、接口和输出路径。";
    } catch {
      // 预览错误在页面内转成可操作提示，避免全局错误弹窗只显示“系统出错”。
      previewError.value = "代码预览加载失败，请先检查表配置、字段配置、Proto 接口配置和父级菜单是否完整。";
    }
    syncWorkspaceTitle();
  } finally {
    loading.value = false;
  }
}

/** 同步代码预览页签和浏览器标题。 */
function syncWorkspaceTitle() {
  const title = `${pageTitle.value}代码预览`;
  tabsStore.setTabsTitle(title);
  document.title = `${title} - ${import.meta.env.VITE_GLOB_APP_TITLE}`;
}

/** 启动当前生成对象的代码生成任务。 */
async function handleGenerate() {
  if (!table.value) return;
  if (table.value.status === codeGenStatusDisabled) {
    ElMessage.warning(`代码生成表配置 ${table.value.name} 已停用`);
    return;
  }
  try {
    await ElMessageBox.confirm(`确认生成业务表：${table.value.name}？`, "提示", {
      confirmButtonText: "确认",
      cancelButtonText: "取消",
      type: "warning"
    });
  } catch {
    return;
  }
  generating.value = true;
  try {
    const data = await defCodeGenService.StartCodeGenTask({
      table_ids: [table.value.id]
    });
    progressTaskId.value = data.task_id;
    progressTaskAvailable.value = true;
    window.sessionStorage.setItem(codeGenTaskStorageKey, data.task_id);
    window.sessionStorage.removeItem(codeGenProgressSelectedTableIdsStorageKey);
    handleProgressDialogVisibleChange(true);
  } catch (error) {
    generating.value = false;
    throw error;
  }
}

/** 打开最近一次代码生成任务。 */
function handleOpenProgress() {
  if (progressTaskId.value) handleProgressDialogVisibleChange(true);
}

/** 同步进度弹窗可见状态，确保热更新后仅恢复任务运行期间主动打开的弹窗。 */
function handleProgressDialogVisibleChange(visible: boolean) {
  progressDialogVisible.value = visible;
  if (visible) {
    window.sessionStorage.setItem(codeGenProgressDialogVisibleStorageKey, "true");
    return;
  }
  window.sessionStorage.removeItem(codeGenProgressDialogVisibleStorageKey);
}

/** 生成任务结束后解除当前页面生成锁定。 */
function handleProgressCompleted() {
  generating.value = false;
  window.sessionStorage.removeItem(codeGenProgressDialogVisibleStorageKey);
  window.sessionStorage.removeItem(codeGenProgressSelectedTableIdsStorageKey);
}

/** 清理不可恢复的最近任务。 */
function handleProgressUnavailable() {
  generating.value = false;
  progressTaskId.value = "";
  progressTaskAvailable.value = false;
  handleProgressDialogVisibleChange(false);
  window.sessionStorage.removeItem(codeGenProgressSelectedTableIdsStorageKey);
  window.sessionStorage.removeItem(codeGenTaskStorageKey);
}

</script>

<style scoped lang="scss">
.code-gen-code-preview-page {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}
.code-gen-sub-card {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}
:deep(.code-gen-sub-card .el-card__body) {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}
:deep(.code-gen-toolbar) {
  display: flex;
  flex: none;
  gap: 10px;
  align-items: center;
  justify-content: flex-end;
  margin-bottom: 14px;
}
.code-gen-code-preview-table,
.code-gen-code-preview-empty {
  flex: 1;
  min-height: 0;
}
.code-gen-code-preview-alert {
  flex: none;
  margin-bottom: 12px;
}

@media (width <= 640px) {
  .code-gen-code-preview-actions {
    display: flex;
    justify-content: flex-end;
  }
}
</style>
