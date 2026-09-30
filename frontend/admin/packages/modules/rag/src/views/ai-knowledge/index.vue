<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="dialogRef"
      :title="t(dialog.editing ? 'common.action.edit' : 'common.action.create')"
      width="min(720px, calc(100vw - 32px))"
      label-width="10em"
      :model="form"
      :fields="fields"
      :rules="rules"
      @confirm="submit"
      @close="resetForm"
    />

    <FormDialog
      v-model="textDialog.visible"
      ref="textDialogRef"
      :title="t('rag.knowledge.doc.action.upload_text')"
      width="min(720px, calc(100vw - 32px))"
      label-width="10em"
      :model="textForm"
      :fields="textFields"
      :rules="textRules"
      @confirm="submitTextDoc"
      @close="resetTextForm"
    />

    <el-drawer
      v-model="docsDrawer.visible"
      :title="t('rag.knowledge.doc.title')"
      size="min(860px, calc(100vw - 32px))"
      destroy-on-close
    >
      <div v-if="docsDrawer.knowledge" class="rag-knowledge-docs">
        <div class="rag-knowledge-docs__head">
          <span class="rag-knowledge-docs__title">{{ docsDrawer.knowledge.name }}</span>
          <span v-if="docsDrawer.knowledge.model_name" class="rag-knowledge-docs__meta">{{
            docsDrawer.knowledge.model_name
          }}</span>
          <span v-if="docsDrawer.knowledge.embedding_dimensions" class="rag-knowledge-docs__meta">{{
            t("rag.knowledge.field.embedding_dimensions_value", { dimensions: docsDrawer.knowledge.embedding_dimensions })
          }}</span>
        </div>

        <div class="rag-knowledge-docs__actions">
          <el-button type="primary" plain :icon="Upload" :loading="uploading" @click="selectDocFile">
            {{ t("rag.knowledge.doc.action.upload_file") }}
          </el-button>
          <el-button :icon="DocumentAdd" :disabled="uploading" @click="openTextDialog">
            {{ t("rag.knowledge.doc.action.upload_text") }}
          </el-button>
          <el-button :icon="Refresh" :loading="loadingDocs" @click="loadDocs">
            {{ t("common.action.refresh") }}
          </el-button>
          <span class="rag-knowledge-docs__hint">{{ t("rag.knowledge.doc.upload.unsupported") }}: txt/md/csv/log/json/xml/yaml/html/docx/pdf</span>
        </div>

        <div v-if="uploading" class="rag-knowledge-docs__progress">
          <el-progress :percentage="uploadPercentage" :stroke-width="8" />
          <span class="rag-knowledge-docs__progress-text">{{
            t("rag.knowledge.doc.upload.progress", {
              current: uploadProgress.current,
              total: uploadProgress.total,
              name: uploadProgress.name
            })
          }}</span>
        </div>

        <el-table v-loading="loadingDocs" :data="docList" row-key="id" size="default">
          <el-table-column prop="name" :label="t('rag.knowledge.doc.field.name')" min-width="180" show-overflow-tooltip />
          <el-table-column prop="chunk_count" :label="t('rag.knowledge.doc.field.chunk_count')" width="90" align="right" />
          <el-table-column :label="t('rag.knowledge.doc.field.status')" width="110" align="center">
            <template #default="{ row }">
              <el-tag
                :type="row.status === AiKnowledgeDocStatus.AI_KNOWLEDGE_DOC_STATUS_READY ? 'success' : 'danger'"
                size="small"
                effect="plain"
              >
                {{
                  row.status === AiKnowledgeDocStatus.AI_KNOWLEDGE_DOC_STATUS_READY
                    ? t("rag.knowledge.doc.status.ready")
                    : t("rag.knowledge.doc.status.failed")
                }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column prop="error_message" :label="t('rag.knowledge.doc.field.error_message')" min-width="180" show-overflow-tooltip />
          <el-table-column prop="created_at" :label="t('common.field.created_at')" width="170" align="center" />
          <el-table-column :label="t('common.field.operation')" width="130" align="center">
            <template #default="{ row }">
              <el-button
                v-if="row.status === AiKnowledgeDocStatus.AI_KNOWLEDGE_DOC_STATUS_FAILED"
                link
                type="primary"
                :loading="reprocessingId === row.id"
                @click="handleReprocessDoc(row as AiKnowledgeDoc)"
              >
                {{ t("rag.knowledge.doc.action.retry") }}
              </el-button>
              <el-button link type="danger" :disabled="!isDefaultTenant || !BUTTONS['base:ai-knowledge:delete']" @click="handleDeleteDoc(row as AiKnowledgeDoc)">
                {{ t("common.action.delete") }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>

        <div class="rag-knowledge-search">
          <div class="rag-knowledge-search__head">
            <span class="rag-knowledge-search__title">{{ t("rag.knowledge.search.title") }}</span>
            <span class="rag-knowledge-search__hint">{{ t("rag.knowledge.search.hint") }}</span>
          </div>
          <div class="rag-knowledge-search__bar">
            <el-input
              v-model="searchQuery"
              :placeholder="t('rag.knowledge.search.placeholder')"
              clearable
              @keyup.enter="runSearch"
            />
            <el-button type="primary" plain :icon="Search" :loading="searching" @click="runSearch">
              {{ t("rag.knowledge.search.action") }}
            </el-button>
          </div>
          <div v-if="searchHits.length" class="rag-knowledge-search__results">
            <div v-for="hit in searchHits" :key="hit.chunk_id" class="rag-knowledge-search__item">
              <div class="rag-knowledge-search__meta">
                <span class="rag-knowledge-search__doc">{{ hit.doc_name }}</span>
                <span class="rag-knowledge-search__score">{{ t("rag.knowledge.search.score") }}: {{ hit.score.toFixed(4) }}</span>
              </div>
              <div class="rag-knowledge-search__content">{{ hit.content }}</div>
            </div>
          </div>
          <div v-else-if="searched" class="rag-knowledge-search__empty">
            {{ t("rag.knowledge.search.empty") }}
          </div>
        </div>
      </div>
    </el-drawer>

    <input
      ref="docFileInputRef"
      class="rag-knowledge-file-input"
      type="file"
      multiple
      accept=".txt,.md,.markdown,.csv,.log,.json,.xml,.yml,.yaml,.html,.htm,.docx,.pdf"
      @change="handleDocFileChange"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import type { FormRules } from "element-plus";
import { CirclePlus, Delete, DocumentAdd, EditPen, Refresh, Search, Upload } from "@element-plus/icons-vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { t } from "@liujitcn/kratos-admin-core";
import { defAiKnowledgeService } from "@liujitcn/kratos-admin-rag/api/rag/admin/v1/ai_knowledge";
import { defAiKnowledgeDocService } from "@liujitcn/kratos-admin-rag/api/rag/admin/v1/ai_knowledge_doc";
import { defAiKnowledgeChunkService } from "@liujitcn/kratos-admin-rag/api/rag/admin/v1/ai_knowledge_chunk";
import type {
  AiKnowledge,
  AiKnowledgeForm,
  AiKnowledgeModel,
  PageAiKnowledgeRequest
} from "@liujitcn/kratos-admin-rag/rpc/rag/admin/v1/ai_knowledge";
import type { AiKnowledgeDoc } from "@liujitcn/kratos-admin-rag/rpc/rag/admin/v1/ai_knowledge_doc";
import { AiKnowledgeDocStatus } from "@liujitcn/kratos-admin-rag/rpc/rag/admin/v1/ai_knowledge_doc";
import type { AiKnowledgeChunkHit } from "@liujitcn/kratos-admin-rag/rpc/rag/admin/v1/ai_knowledge_chunk";

defineOptions({ name: "AiKnowledge", inheritAttrs: false });

const SUPPORTED_EXTENSIONS = [".txt", ".md", ".markdown", ".csv", ".log", ".json", ".xml", ".yml", ".yaml", ".html", ".htm", ".docx", ".pdf"];
const MAX_FILE_SIZE = 10 * 1024 * 1024;

type AiKnowledgeFormState = Omit<AiKnowledgeForm, "model_id" | "embedding_dimensions"> & {
  model_id: number | undefined;
  embedding_dimensions: number | undefined;
};

const { BUTTONS } = useAuthButtons();
const { isDefaultTenant } = useTenantScope();
const table = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const textDialogRef = ref<InstanceType<typeof FormDialog>>();
const docFileInputRef = ref<HTMLInputElement>();
const dialog = reactive({ visible: false, editing: false });
const textDialog = reactive({ visible: false });
const form = reactive<AiKnowledgeFormState>(defaultForm());
const textForm = reactive({ name: "", content: "" });
const knowledgeModels = ref<AiKnowledgeModel[]>([]);
// 模型级联选项：一级供应商、二级模型，叶子值取模型ID。
const modelOptions = computed<ProFormOption[]>(() => {
  const groups = new Map<string, ProFormOption[]>();
  for (const item of knowledgeModels.value) {
    const label = item.display_name ? `${item.display_name}（${item.model_name}）` : item.model_name;
    const children = groups.get(item.provider_name) ?? [];
    children.push({ label, value: item.id });
    groups.set(item.provider_name, children);
  }
  return [...groups.entries()].map(([provider, children]) => ({ label: provider, value: provider, children }));
});
// 维度选项随所选模型联动：模型声明了维度时唯一可选，未声明时提供常用维度。
const dimensionOptions = computed<ProFormOption[]>(() => {
  const selected = knowledgeModels.value.find((item: AiKnowledgeModel) => item.id === form.model_id);
  return (selected?.dimension_options ?? []).map(value => ({ label: `${value}`, value }));
});
const docsDrawer = reactive<{ visible: boolean; knowledge: AiKnowledge | null }>({ visible: false, knowledge: null });
const docList = ref<AiKnowledgeDoc[]>([]);
const loadingDocs = ref(false);
const uploading = ref(false);
const uploadProgress = reactive({ current: 0, total: 0, name: "" });
const uploadPercentage = computed(() =>
  uploadProgress.total ? Math.round((uploadProgress.current / uploadProgress.total) * 100) : 0
);
const reprocessingId = ref<number | undefined>();
const searchQuery = ref("");
const searchHits = ref<AiKnowledgeChunkHit[]>([]);
const searching = ref(false);
const searched = ref(false);

const fields = computed<ProFormField[]>(() => [
  { prop: "name", label: t("rag.knowledge.field.name"), component: "input", colSpan: 24, props: { maxlength: 100 } },
  { prop: "description", label: t("rag.knowledge.field.description"), component: "textarea", colSpan: 24, props: { maxlength: 500, rows: 3 } },
  { prop: "model_id", label: t("rag.knowledge.field.model"), component: "cascader", colSpan: 24, options: modelOptions.value, props: { props: { emitPath: false }, clearable: true, filterable: true, style: "width: 100%" } },
  {
    prop: "embedding_dimensions",
    label: t("rag.knowledge.field.embedding_dimensions"),
    component: "select",
    colSpan: 24,
    options: dimensionOptions.value,
    // 编辑时维度是创建快照不可改；未选模型前无可选维度。
    props: { disabled: dialog.editing || !form.model_id }
  }
]);

// 模型声明了维度或仅有一个可选维度时自动选中；切换模型后清理不再可用的维度。
watch(
  () => form.model_id,
  () => {
    const selected = knowledgeModels.value.find((item: AiKnowledgeModel) => item.id === form.model_id);
    const options = selected?.dimension_options ?? [];
    if (options.length === 1) {
      form.embedding_dimensions = options[0];
      return;
    }
    if (form.embedding_dimensions === undefined || !options.includes(form.embedding_dimensions)) {
      form.embedding_dimensions = undefined;
    }
  }
);

const textFields = computed<ProFormField[]>(() => [
  { prop: "name", label: t("rag.knowledge.doc.field.name"), component: "input", colSpan: 24, props: { maxlength: 200 } },
  { prop: "content", label: t("rag.knowledge.doc.field.content"), component: "textarea", colSpan: 24, props: { rows: 12, maxlength: 1000000, placeholder: t("rag.knowledge.doc.placeholder.content") } }
]);

const rules = computed<FormRules>(() => ({
  name: [{ required: true, message: t("rag.knowledge.validation.name"), trigger: "blur" }],
  model_id: [{ required: true, message: t("rag.knowledge.validation.model"), trigger: "change" }],
  embedding_dimensions: [{ required: true, message: t("rag.knowledge.validation.embedding_dimensions"), trigger: "change" }]
}));

const textRules = computed<FormRules>(() => ({
  name: [{ required: true, message: t("rag.knowledge.doc.validation.name"), trigger: "blur" }],
  content: [{ required: true, message: t("rag.knowledge.doc.validation.content"), trigger: "blur" }]
}));

const columns = computed<ColumnProps[]>(() => [
  { prop: "name", label: t("rag.knowledge.field.name"), minWidth: 160, search: { el: "input" } },
  { prop: "provider_name", label: t("rag.knowledge.field.provider"), minWidth: 140 },
  { prop: "model_name", label: t("rag.knowledge.field.model"), minWidth: 180 },
  { prop: "embedding_dimensions", label: t("rag.knowledge.field.embedding_dimensions"), width: 90, align: "right" },
  { prop: "doc_count", label: t("rag.knowledge.field.doc_count"), width: 90, align: "right" },
  { prop: "updated_at", label: t("common.field.updated_at"), minWidth: 170, align: "center" },
  {
    prop: "operation",
    label: t("common.field.operation"),
    cellType: "actions",
    actions: [
      { label: t("common.action.edit"), link: true, icon: EditPen, hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:ai-knowledge:update"], onClick: scope => openDialog((scope.row as AiKnowledge).id) },
      { label: t("rag.knowledge.action.docs"), link: true, icon: DocumentAdd, onClick: scope => openDocsDrawer(scope.row as AiKnowledge) },
      { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !isDefaultTenant.value || !BUTTONS.value["base:ai-knowledge:delete"], onClick: scope => handleDelete(scope.row as AiKnowledge) }
    ]
  }
]);

const headerActions = computed<HeaderActionProps[]>(() => [
  { label: t("common.action.create"), type: "primary", icon: CirclePlus, hidden: !isDefaultTenant.value || !BUTTONS.value["base:ai-knowledge:create"], onClick: () => openDialog() },
  { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: !isDefaultTenant.value || !BUTTONS.value["base:ai-knowledge:delete"], disabled: scope => !scope.isSelected, onClick: scope => handleDelete(scope.selectedListIds as number[]) }
]);

/** 创建默认知识库表单。 */
function defaultForm(): AiKnowledgeFormState {
  return { id: 0, name: "", description: "", embedding_dimensions: 0, model_id: undefined };
}

/** 请求AI知识库表格数据。 */
async function requestTable(params: Record<string, unknown>) {
  const data = await defAiKnowledgeService.PageAiKnowledge(buildPageRequest<PageAiKnowledgeRequest>(params as unknown as PageAiKnowledgeRequest));
  return { data: { list: data.knowledges ?? [], total: data.total } };
}

/** 加载可选的已启用embedding模型。 */
async function loadModelOptions() {
  try {
    const response = await defAiKnowledgeService.ListAiKnowledgeModels({});
    knowledgeModels.value = response.models ?? [];
  } catch {
    knowledgeModels.value = [];
  }
}

/** 打开知识库编辑表单。 */
async function openDialog(id?: number) {
  await dialogRef.value?.open({
    load: () => (id ? defAiKnowledgeService.GetAiKnowledge({ id }) : undefined),
    commit: value => {
      Object.assign(form, defaultForm(), value ?? {});
      dialog.editing = Boolean(id);
    }
  });
}

/** 提交知识库表单。 */
async function submit() {
  const valid = await dialogRef.value?.validate();
  if (!valid) return;
  const payload = JSON.parse(JSON.stringify(form)) as AiKnowledgeForm;
  if (dialog.editing) await defAiKnowledgeService.UpdateAiKnowledge({ ai_knowledge: payload });
  else await defAiKnowledgeService.CreateAiKnowledge({ ai_knowledge: payload });
  ElMessage.success(t(dialog.editing ? "rag.knowledge.message.update_success" : "rag.knowledge.message.create_success"));
  dialog.visible = false;
  resetForm();
  table.value?.getTableList();
}

/** 重置知识库表单。 */
function resetForm() {
  Object.assign(form, defaultForm());
  dialog.editing = false;
}

/** 批量删除AI知识库。 */
async function handleDelete(target: AiKnowledge | number | Array<number>) {
  const row = typeof target === "object" && !Array.isArray(target) ? target : undefined;
  const ids = (row ? [row.id] : normalizeSelectedIds(target as number | number[])).map(Number);
  if (!ids.length) return;
  try {
    await ElMessageBox.confirm(
      t("rag.knowledge.message.confirm_delete", { name: row?.name ?? ids.join(", ") }),
      t("common.action.delete"),
      { type: "warning" }
    );
  } catch {
    return;
  }
  for (const id of ids) await defAiKnowledgeService.DeleteAiKnowledge({ id });
  ElMessage.success(t("rag.knowledge.message.delete_success"));
  table.value?.getTableList();
}

/** 打开文档管理抽屉。 */
async function openDocsDrawer(row: AiKnowledge) {
  docsDrawer.knowledge = row;
  docsDrawer.visible = true;
  searchQuery.value = "";
  searchHits.value = [];
  searched.value = false;
  await loadDocs();
}

/** 加载当前知识库的文档列表。 */
async function loadDocs() {
  const knowledge = docsDrawer.knowledge;
  if (!knowledge) return;
  loadingDocs.value = true;
  try {
    const response = await defAiKnowledgeDocService.PageAiKnowledgeDoc({ id: knowledge.id, page_num: 1, page_size: 100 });
    docList.value = response.docs ?? [];
  } catch {
    docList.value = [];
  } finally {
    loadingDocs.value = false;
  }
}

/** 触发文档文件选择。 */
function selectDocFile() {
  docFileInputRef.value?.click();
}

/** 读取选择的文档文件并逐个上传，展示整体进度。 */
async function handleDocFileChange(event: Event) {
  const input = event.target as HTMLInputElement;
  const files = Array.from(input.files ?? []);
  input.value = "";
  if (!files.length || !docsDrawer.knowledge) return;
  const accepted: File[] = [];
  const rejected: string[] = [];
  for (const file of files) {
    const extension = `.${file.name.split(".").pop()?.toLowerCase() ?? ""}`;
    if (!SUPPORTED_EXTENSIONS.includes(extension) || file.size > MAX_FILE_SIZE) {
      rejected.push(file.name);
      continue;
    }
    accepted.push(file);
  }
  if (rejected.length) {
    ElMessage.warning(t("rag.knowledge.doc.upload.batch_invalid", { names: rejected.join("、") }));
  }
  if (!accepted.length) return;
  const knowledgeId = docsDrawer.knowledge.id;
  uploading.value = true;
  uploadProgress.total = accepted.length;
  uploadProgress.current = 0;
  uploadProgress.name = accepted[0].name;
  let successCount = 0;
  const failedNames: string[] = [];
  try {
    for (const file of accepted) {
      uploadProgress.name = file.name;
      try {
        const contentBase64 = await readFileAsBase64(file);
        if (!contentBase64) {
          failedNames.push(file.name);
          continue;
        }
        await defAiKnowledgeDocService.UploadAiKnowledgeDocFile({
          id: knowledgeId,
          file_name: file.name,
          // bytes 字段线上按 base64 字符串传输，类型按生成声明适配。
          content_base64: contentBase64 as unknown as Uint8Array,
          doc_name: ""
        });
        successCount++;
      } catch {
        failedNames.push(file.name);
      } finally {
        uploadProgress.current++;
      }
    }
  } finally {
    uploading.value = false;
    await loadDocs();
    table.value?.getTableList();
  }
  if (!failedNames.length) {
    ElMessage.success(t("rag.knowledge.doc.upload.batch_success", { count: successCount }));
  } else if (successCount) {
    ElMessage.warning(t("rag.knowledge.doc.upload.batch_partial", { success: successCount, failed: failedNames.length }));
  } else {
    ElMessage.error(t("rag.knowledge.doc.upload.batch_partial", { success: successCount, failed: failedNames.length }));
  }
}

/** 打开纯文本上传弹窗。 */
function openTextDialog() {
  Object.assign(textForm, { name: "", content: "" });
  textDialog.visible = true;
}

/** 提交纯文本文档。 */
async function submitTextDoc() {
  const valid = await textDialogRef.value?.validate();
  if (!valid || !docsDrawer.knowledge) return;
  await defAiKnowledgeDocService.UploadAiKnowledgeDocText({
    id: docsDrawer.knowledge.id,
    name: textForm.name,
    content: textForm.content
  });
  ElMessage.success(t("rag.knowledge.doc.upload.success"));
  textDialog.visible = false;
  await loadDocs();
  table.value?.getTableList();
}

/** 重置文本上传表单。 */
function resetTextForm() {
  Object.assign(textForm, { name: "", content: "" });
}

/** 删除单个文档。 */
async function handleDeleteDoc(row: AiKnowledgeDoc) {
  try {
    await ElMessageBox.confirm(
      t("rag.knowledge.doc.message.confirm_delete", { name: row.name }),
      t("common.action.delete"),
      { type: "warning" }
    );
  } catch {
    return;
  }
  await defAiKnowledgeDocService.DeleteAiKnowledgeDoc({ doc_id: row.id });
  ElMessage.success(t("rag.knowledge.doc.message.delete_success"));
  await loadDocs();
  table.value?.getTableList();
}

/** 重新处理失败的文档。 */
async function handleReprocessDoc(row: AiKnowledgeDoc) {
  if (reprocessingId.value) return;
  reprocessingId.value = row.id;
  try {
    await defAiKnowledgeDocService.ReprocessAiKnowledgeDoc({ doc_id: row.id });
    ElMessage.success(t("rag.knowledge.doc.message.reprocess_success"));
    await loadDocs();
    table.value?.getTableList();
  } finally {
    reprocessingId.value = undefined;
  }
}

/** 执行知识库检索测试。 */
async function runSearch() {
  const knowledge = docsDrawer.knowledge;
  const query = searchQuery.value.trim();
  if (!knowledge || !query) return;
  searching.value = true;
  try {
    const response = await defAiKnowledgeChunkService.SearchAiKnowledge({ id: knowledge.id, query, top_k: 5 });
    searchHits.value = response.hits ?? [];
    searched.value = true;
  } catch {
    searchHits.value = [];
  } finally {
    searching.value = false;
  }
}

/** 读取文件为 base64 字符串。 */
function readFileAsBase64(file: File): Promise<string | undefined> {
  return new Promise(resolve => {
    const reader = new FileReader();
    reader.onload = () => {
      const result = reader.result;
      if (typeof result !== "string") {
        ElMessage.error(t("rag.knowledge.doc.upload.read_failed"));
        resolve(undefined);
        return;
      }
      resolve(result.split(",").pop() ?? "");
    };
    reader.onerror = () => {
      ElMessage.error(t("rag.knowledge.doc.upload.read_failed"));
      resolve(undefined);
    };
    reader.readAsDataURL(file);
  });
}

loadModelOptions();
</script>

<style scoped>
.rag-knowledge-file-input {
  display: none;
}

.rag-knowledge-docs {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.rag-knowledge-docs__head {
  display: flex;
  align-items: baseline;
  gap: 12px;
  min-width: 0;
}

.rag-knowledge-docs__title {
  font-size: 16px;
  font-weight: 600;
}

.rag-knowledge-docs__meta {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.rag-knowledge-docs__actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.rag-knowledge-docs__hint {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.rag-knowledge-docs__progress {
  display: flex;
  align-items: center;
  gap: 12px;
}

.rag-knowledge-docs__progress :deep(.el-progress) {
  flex: 1;
}

.rag-knowledge-docs__progress-text {
  max-width: 50%;
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rag-knowledge-search {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.rag-knowledge-search__head {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.rag-knowledge-search__title {
  font-weight: 600;
}

.rag-knowledge-search__hint {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}

.rag-knowledge-search__bar {
  display: flex;
  gap: 8px;
}

.rag-knowledge-search__results {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.rag-knowledge-search__item {
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
}

.rag-knowledge-search__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
}

.rag-knowledge-search__doc {
  font-weight: 600;
}

.rag-knowledge-search__score {
  color: var(--el-color-primary);
  font-size: 12px;
}

.rag-knowledge-search__content {
  color: var(--el-text-color-regular);
  font-size: 13px;
  line-height: 1.6;
  white-space: pre-wrap;
}

.rag-knowledge-search__empty {
  padding: 16px;
  color: var(--el-text-color-secondary);
  text-align: center;
}
</style>
