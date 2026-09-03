<template>
  <div v-loading="loading" class="table-box project-doc-page">
    <main class="project-doc-shell">
      <aside class="document-navigation" :aria-label="t('system.project.document.title.navigation')">
        <header class="navigation-header">
          <el-input v-model="keyword" clearable :placeholder="t('system.project.document.placeholder.search')" :prefix-icon="Search" />
          <span class="document-count">{{ filteredDocumentCount }} / {{ documents.length }}</span>
        </header>

        <div v-if="documents.length" class="document-tree-scroll">
          <el-tree
            v-show="filteredDocumentCount > 0"
            ref="documentTreeRef"
            class="document-tree"
            :data="documentTree"
            node-key="key"
            :props="{ children: 'children', label: 'label' }"
            :current-node-key="selectedDocumentNodeKey"
            :filter-node-method="filterTreeNode"
            default-expand-all
            highlight-current
            @node-click="handleTreeNodeClick"
          >
            <template #default="{ data }">
              <span class="document-tree-node" :class="`is-${data.kind}`" :title="data.path || data.label">
                <el-icon v-if="data.kind === 'document'" class="document-tree-node__icon"><Document /></el-icon>
                <span class="document-tree-node__label">{{ data.label }}</span>
                <code v-if="data.kind === 'project'">{{ data.projectKey }}</code>
              </span>
            </template>
          </el-tree>
          <el-empty v-if="filteredDocumentCount === 0" :description="t('system.project.document.message.no_match')" :image-size="72" />
        </div>
        <el-empty v-else :description="t('system.project.document.message.empty')" :image-size="72" />
      </aside>

      <section class="document-reader">
        <div v-if="detailLoading && !selectedDocument" class="reader-loading">
          <el-skeleton :rows="10" animated />
        </div>
        <template v-else-if="selectedDocument">
          <header class="reader-header">
            <div class="reader-header__identity">
              <el-icon><Document /></el-icon>
              <div class="reader-header__title">
                <strong>{{ selectedDocumentName }}</strong>
                <span :title="selectedDocument.path">{{ selectedDocument.project_name }} / {{ selectedDocument.path }}</span>
              </div>
            </div>
            <time v-if="selectedDocumentUpdatedAt" :datetime="selectedDocumentUpdatedAt" :title="selectedDocumentUpdatedAt">
              {{ t("system.project.document.value.updated_at", { time: formatDocumentUpdatedAt(selectedDocumentUpdatedAt) }) }}
            </time>
          </header>
          <div ref="readerScrollRef" v-loading="detailLoading" class="reader-scroll">
            <div ref="markdownRef" class="project-markdown" @click="handleMarkdownClick">
              <MarkdownPreview
                id="project-doc-preview"
                class="project-markdown__preview"
                :model-value="selectedDocument.content"
                :is-dark="globalStore.isDark"
              />
            </div>
          </div>
        </template>
        <el-empty v-else :description="t('system.project.document.message.select')" />
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from "vue";
import { Search } from "@element-plus/icons-vue";
import type { TreeNodeData } from "element-plus";
import { getCurrentLocale, t, useLocaleStore } from "@liujitcn/kratos-admin-core";
import MarkdownPreview from "@liujitcn/kratos-admin-core/components/MarkdownPreview/index.vue";
import { useGlobalStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import { defProjectDocumentService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/project_document";
import type {
  ProjectDocument,
  ProjectDocumentDirectory,
  ProjectDocumentListItem,
  ProjectDocumentProject
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/project_document";

defineOptions({
  name: "ProjectDocument",
  inheritAttrs: false
});

/** 项目文档树节点类型。 */
type ProjectDocumentTreeNodeKind = "project" | "directory" | "document";

/** 管理端项目文档树节点。 */
interface ProjectDocumentTreeNode {
  /** 树节点稳定标识。 */
  key: string;
  /** 树节点展示名称。 */
  label: string;
  /** 树节点类型。 */
  kind: ProjectDocumentTreeNodeKind;
  /** 所属项目标识。 */
  projectKey: string;
  /** 所属项目名称。 */
  projectName: string;
  /** 项目内相对路径。 */
  path: string;
  /** 文档叶子节点数据。 */
  document?: ProjectDocumentListItem;
  /** 子节点。 */
  children: ProjectDocumentTreeNode[];
}

const globalStore = useGlobalStore();
const { locale } = useLocaleStore();
const loading = ref(false);
const detailLoading = ref(false);
const keyword = ref("");
const projects = ref<ProjectDocumentProject[]>([]);
const selectedDocumentId = ref("");
const selectedDocument = ref<ProjectDocument>();
const detailRequestToken = ref(0);
const documentTreeRef = ref<InstanceType<typeof ElTree>>();
const markdownRef = ref<HTMLElement>();
const readerScrollRef = ref<HTMLElement>();
const documentTree = computed<ProjectDocumentTreeNode[]>(() => projects.value.map(buildProjectTreeNode));
const documents = computed<ProjectDocumentTreeNode[]>(() => flattenDocumentTree(documentTree.value));
const documentPathIndex = computed(() => {
  const index = new Map<string, ProjectDocumentListItem>();
  for (const node of documents.value) {
    if (node.document) {
      index.set(`${node.projectKey}\0${node.path}`, node.document);
    }
  }
  return index;
});
const filteredDocumentCount = computed(() => {
  const search = keyword.value.trim().toLocaleLowerCase();
  if (!search) return documents.value.length;
  return documents.value.filter(node =>
    [node.label, node.projectKey, node.projectName, node.path].some(value => value.toLocaleLowerCase().includes(search))
  ).length;
});
const selectedDocumentNodeKey = computed(() => (selectedDocumentId.value ? buildDocumentNodeKey(selectedDocumentId.value) : ""));
const selectedDocumentName = computed(() => selectedDocument.value?.name ?? "");
const selectedDocumentUpdatedAt = computed(
  () => documents.value.find(node => node.document?.id === selectedDocumentId.value)?.document?.updated_at ?? ""
);

/** 加载项目文档目录并恢复当前选择。 */
async function loadDocuments() {
  loading.value = true;
  try {
    const response = await defProjectDocumentService.TreeProjectDocument({});
    projects.value = response.projects ?? [];
    const nextId = documents.value.some(node => node.document?.id === selectedDocumentId.value)
      ? selectedDocumentId.value
      : (documents.value[0]?.document?.id ?? "");
    await nextTick();
    documentTreeRef.value?.filter(keyword.value);
    if (nextId) {
      await selectDocument(nextId);
    } else {
      selectedDocumentId.value = "";
      selectedDocument.value = undefined;
      detailRequestToken.value += 1;
    }
  } finally {
    loading.value = false;
  }
}

/** 选择项目文档并加载 Markdown 详情。 */
async function selectDocument(id: string, anchor = "") {
  if (!id) return;
  selectedDocumentId.value = id;
  detailLoading.value = true;
  const requestToken = detailRequestToken.value + 1;
  detailRequestToken.value = requestToken;
  try {
    const detail = await defProjectDocumentService.GetProjectDocument({ id });
    if (detailRequestToken.value === requestToken && selectedDocumentId.value === id) {
      selectedDocument.value = detail;
      await nextTick();
      documentTreeRef.value?.setCurrentKey(buildDocumentNodeKey(id));
      scrollToAnchor(anchor);
    }
  } finally {
    if (detailRequestToken.value === requestToken) {
      detailLoading.value = false;
    }
  }
}

/** 处理项目文档树节点选择。 */
function handleTreeNodeClick(node: ProjectDocumentTreeNode) {
  if (!node.document) return;
  void selectDocument(node.document.id);
}

/** 按项目、目录或路径过滤项目文档树。 */
function filterTreeNode(value: unknown, data: TreeNodeData) {
  const node = data as ProjectDocumentTreeNode;
  const search = String(value ?? "")
    .trim()
    .toLocaleLowerCase();
  if (!search) return true;
  return [node.label, node.projectKey, node.projectName, node.path].some(field => field.toLocaleLowerCase().includes(search));
}

/** 处理 Markdown 中的页内锚点、相对文档和外部链接。 */
function handleMarkdownClick(event: MouseEvent) {
  const target = event.target;
  if (!(target instanceof Element)) return;
  const anchor = target.closest("a");
  const href = anchor?.getAttribute("href");
  if (!anchor || !href || !selectedDocument.value) return;
  if (href.startsWith("#")) {
    event.preventDefault();
    scrollToAnchor(href.slice(1));
    return;
  }
  if (/^(?:[a-z][a-z\d+.-]*:|\/\/)/i.test(href)) {
    event.preventDefault();
    window.open(href, "_blank", "noopener,noreferrer");
    return;
  }

  event.preventDefault();
  const [rawPath, rawAnchor = ""] = href.split("#", 2);
  const linkPath = decodeURIComponent(rawPath.split("?", 1)[0] ?? "");
  const documentPath = resolveDocumentPath(selectedDocument.value.path, linkPath);
  const linkedDocument = documentPathIndex.value.get(`${selectedDocument.value.project_key}\0${documentPath}`);
  if (!linkedDocument) {
    ElMessage.warning(t("system.project.document.message.uncollected", { path: documentPath }));
    return;
  }
  void selectDocument(linkedDocument.id, decodeURIComponent(rawAnchor));
}

/** 基于当前文档目录解析项目内相对路径。 */
function resolveDocumentPath(currentPath: string, linkPath: string) {
  const baseSegments = linkPath.startsWith("/") ? [] : currentPath.split("/").slice(0, -1);
  for (const segment of linkPath.split("/")) {
    if (!segment || segment === ".") continue;
    if (segment === "..") {
      baseSegments.pop();
      continue;
    }
    baseSegments.push(segment);
  }
  return baseSegments.join("/");
}

/** 在当前 Markdown 阅读区滚动到指定标题锚点。 */
function scrollToAnchor(anchor: string) {
  if (!anchor) {
    readerScrollRef.value?.scrollTo({ top: 0 });
    return;
  }
  const decodedAnchor = decodeURIComponent(anchor);
  const heading = markdownRef.value?.querySelector<HTMLElement>(`#${CSS.escape(decodedAnchor)}`);
  heading?.scrollIntoView({ block: "start" });
}

/** 将 RFC3339 更新时间格式化为紧凑的本地时间。 */
function formatDocumentUpdatedAt(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(getCurrentLocale(), {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23"
  })
    .format(date)
    .replaceAll("/", "-");
}

/** 将接口项目节点转换为管理端树节点。 */
function buildProjectTreeNode(project: ProjectDocumentProject): ProjectDocumentTreeNode {
  return {
    key: `project:${project.key}`,
    label: project.name,
    kind: "project",
    projectKey: project.key,
    projectName: project.name,
    path: "",
    children: [
      ...(project.documents ?? []).map(document => buildDocumentTreeNode(document, project.key, project.name)),
      ...(project.directories ?? []).map(directory => buildDirectoryTreeNode(directory, project.key, project.name))
    ]
  };
}

/** 递归转换接口目录节点。 */
function buildDirectoryTreeNode(
  directory: ProjectDocumentDirectory,
  projectKey: string,
  projectName: string
): ProjectDocumentTreeNode {
  return {
    key: `directory:${projectKey}:${directory.path}`,
    label: directory.name,
    kind: "directory",
    projectKey,
    projectName,
    path: directory.path,
    children: [
      ...(directory.documents ?? []).map(document => buildDocumentTreeNode(document, projectKey, projectName)),
      ...(directory.directories ?? []).map(child => buildDirectoryTreeNode(child, projectKey, projectName))
    ]
  };
}

/** 将接口文档目录项转换为树叶子节点。 */
function buildDocumentTreeNode(
  document: ProjectDocumentListItem,
  projectKey: string,
  projectName: string
): ProjectDocumentTreeNode {
  return {
    key: buildDocumentNodeKey(document.id),
    label: document.name,
    kind: "document",
    projectKey,
    projectName,
    path: document.path,
    document,
    children: []
  };
}

/** 构造文档树叶子节点稳定标识。 */
function buildDocumentNodeKey(id: string) {
  return `document:${id}`;
}

/** 将树中的文档叶子节点展开为导航节点。 */
function flattenDocumentTree(nodes: ProjectDocumentTreeNode[]): ProjectDocumentTreeNode[] {
  const documents: ProjectDocumentTreeNode[] = [];
  for (const node of nodes) {
    if (node.document) {
      documents.push(node);
    }
    documents.push(...flattenDocumentTree(node.children));
  }
  return documents;
}

watch(keyword, value => {
  documentTreeRef.value?.filter(value);
});

watch(locale, () => {
  void loadDocuments();
});

onMounted(() => {
  void loadDocuments();
});
</script>

<style scoped lang="scss">
.project-doc-page {
  box-sizing: border-box;
  color: var(--admin-page-text-primary);
  background: var(--el-bg-color-page);
}
.project-doc-shell {
  display: grid;
  flex: 1;
  grid-template-columns: 340px minmax(0, 1fr);
  min-width: 0;
  min-height: 520px;
  overflow: hidden;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}
.document-navigation,
.document-reader {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}
.document-navigation {
  border-right: 1px solid var(--admin-page-divider);
}
.navigation-header {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid var(--admin-page-divider);
}
.document-count {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
  white-space: nowrap;
}
.document-tree-scroll {
  flex: 1;
  min-height: 0;
  padding: 8px;
  overflow-y: auto;
}
.document-tree {
  min-width: 100%;
  color: var(--admin-page-text-primary);
  background: transparent;

  --el-tree-node-hover-bg-color: var(--el-fill-color-light);
}
.document-tree :deep(.el-tree-node__content) {
  height: 34px;
  padding-right: 8px;
  border-radius: var(--admin-page-radius);
}
.document-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
  color: var(--el-color-primary);
  background: var(--admin-page-accent-soft-bg);
  box-shadow: inset 3px 0 0 var(--el-color-primary);
}
.document-tree-node {
  display: flex;
  flex: 1;
  gap: 7px;
  align-items: center;
  min-width: 0;
  overflow: hidden;
}
.document-tree-node__icon {
  flex: 0 0 auto;
  font-size: 15px;
  color: var(--admin-page-text-secondary);
}
.document-tree-node__label {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.document-tree-node.is-project .document-tree-node__label {
  font-weight: 650;
}
.document-tree-node.is-directory .document-tree-node__label {
  font-weight: 550;
}
.document-tree-node code {
  flex: 0 0 auto;
  margin-left: auto;
  font-size: 11px;
  color: var(--admin-page-text-secondary);
}
.reader-header {
  box-sizing: border-box;
  display: flex;
  gap: 20px;
  align-items: center;
  justify-content: space-between;
  min-height: 64px;
  padding: 10px 24px;
  border-bottom: 1px solid var(--admin-page-divider);
}
.reader-header__identity {
  display: flex;
  gap: 10px;
  align-items: center;
  min-width: 0;
}
.reader-header__identity > :deep(.el-icon) {
  flex: 0 0 auto;
  font-size: 18px;
  color: var(--el-color-primary);
}
.reader-header__title {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}
.reader-header__title strong,
.reader-header__title span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.reader-header__title strong {
  font-size: 15px;
  color: var(--admin-page-text-primary);
}
.reader-header__title span,
.reader-header time {
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}
.reader-header time {
  flex: 0 0 auto;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
.reader-scroll {
  flex: 1;
  min-height: 0;
  padding: 28px clamp(24px, 4vw, 52px) 40px;
  overflow-y: auto;
  scrollbar-gutter: stable;
}
.project-markdown {
  width: 100%;
  max-width: 1040px;
  margin: 0 auto;
  overflow-wrap: anywhere;
}
.reader-loading {
  padding: 24px;
}

@media (width <= 900px) {
  .project-doc-shell {
    grid-template-columns: 280px minmax(0, 1fr);
  }
}

@media (width <= 720px) {
  .project-doc-shell {
    display: flex;
    flex-direction: column;
  }
  .document-navigation {
    flex: 0 0 auto;
    height: 240px;
    border-right: 0;
    border-bottom: 1px solid var(--admin-page-divider);
  }
  .document-reader {
    flex: 1;
    min-height: 0;
  }
  .reader-scroll {
    padding-top: 24px;
    padding-right: 16px;
    padding-left: 16px;
  }
  .reader-header {
    padding-right: 16px;
    padding-left: 16px;
  }
  .reader-header time {
    display: none;
  }
  .project-markdown :deep(h1) {
    font-size: 26px;
  }
  .project-markdown :deep(h2) {
    font-size: 21px;
  }
}
</style>
