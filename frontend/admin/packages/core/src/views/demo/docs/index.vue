<template>
  <div class="component-docs">
    <el-aside class="docs-sidebar" aria-label="组件目录">
      <el-input v-model="state.query" clearable placeholder="搜索组件名称…" aria-label="搜索组件名称" />
      <nav aria-label="组件分类">
        <details v-for="group in navigationGroups" :key="group.label" :open="group.open">
          <summary>{{ group.label }}</summary>
          <el-button
            v-for="entry in group.entries"
            :key="entry.id"
            class="admin-custom-button docs-nav-item"
            native-type="button"
            :aria-current="state.id === entry.id ? 'page' : undefined"
            @click="selectDoc(entry.id)"
          >
            {{ entry.label }}
          </el-button>
        </details>
      </nav>
      <el-empty v-if="!navigationEntries.length" description="没有匹配的组件" :image-size="50" />
    </el-aside>

    <div class="docs-mobile-picker">
      <el-input v-model="state.query" clearable placeholder="搜索组件名称…" aria-label="搜索组件名称" />
      <el-select :model-value="state.id" aria-label="选择组件" @change="selectDoc">
        <el-option-group v-for="group in navigationGroups" :key="group.label" :label="group.label">
          <el-option v-for="entry in group.entries" :key="entry.id" :label="entry.label" :value="entry.id" />
        </el-option-group>
      </el-select>
    </div>

    <el-scrollbar ref="documentRef" class="docs-content" :tabindex="0" aria-label="组件说明">
      <el-main v-if="selected" class="docs-document" aria-label="组件说明">
        <el-button v-if="state.history.length" class="docs-back" @click="goBack">返回{{ backLabel }}</el-button>
        <el-text tag="h1">
          {{ selected.id }} <span>{{ selected.label }}</span>
        </el-text>
        <el-text tag="p" class="docs-intro">{{ selected.description }}</el-text>

        <section id="component-docs-basic" class="docs-section">
          <el-text tag="h2">基础用法</el-text>
          <div v-show="selected.example" class="docs-example">
            <div class="docs-preview">
              <KeepAlive :max="20">
                <component :is="activeExample" v-if="activeExample" :key="selected.id" :component-id="selected.id" />
              </KeepAlive>
            </div>
            <div class="docs-example-tools">
              <el-tooltip content="复制代码" placement="bottom">
                <el-button text :icon="Copy" aria-label="复制代码" @click="copyCode" />
              </el-tooltip>
              <el-tooltip :content="sourceVisible ? '收起源代码' : '查看源代码'" placement="bottom" :tabindex="-1">
                <span class="docs-source-trigger">
                  <el-button
                    text
                    :icon="Code"
                    :aria-label="sourceVisible ? '收起源代码' : '查看源代码'"
                    :aria-expanded="sourceVisible"
                    aria-controls="component-docs-source"
                    @click="sourceVisible = !sourceVisible"
                  />
                </span>
              </el-tooltip>
            </div>
            <CodeBlock v-if="sourceVisible" id="component-docs-source" :code="selected.code" language="html" />
          </div>
          <CodeBlock v-if="!selected.example" :code="selected.code" language="html" />
        </section>

        <section id="component-docs-api" class="docs-section">
          <el-text tag="h2">{{ selected.id }} API</el-text>
          <el-tabs v-model="state.api">
            <el-tab-pane v-for="tab in apiTabs" :key="tab.key" :name="tab.key" :label="tab.label" />
          </el-tabs>
          <el-table v-if="apiEntries.length" :data="api.items" class="docs-api-table">
            <el-table-column label="名称" min-width="145">
              <template #default="scope"
                ><code>{{ scope.row.name }}</code
                ><span v-if="scope.row.required" class="docs-required" aria-label="必填"> *</span></template
              >
            </el-table-column>
            <el-table-column prop="description" label="说明" min-width="175" />
            <el-table-column label="类型 / 参数" min-width="165">
              <template #default="scope"
                ><code>{{ scope.row.type || "—" }}</code></template
              >
            </el-table-column>
            <el-table-column v-if="state.api === 'props'" label="默认值" min-width="95">
              <template #default="scope"
                ><code>{{ scope.row.default || "—" }}</code></template
              >
            </el-table-column>
          </el-table>
          <el-empty v-else description="没有显式声明的此类 API" :image-size="60" />
          <DocsPagination
            v-if="apiEntries.length > 6"
            :page="api.page"
            :page-size="6"
            :total="apiEntries.length"
            @update:page="state.apiPage = $event"
          />
        </section>

        <section id="component-docs-usage" class="docs-section">
          <el-text tag="h2">使用说明</el-text>
          <el-text tag="p" class="docs-usage">{{ selected.usage }}</el-text>
          <div v-if="related.length" class="docs-related">
            <el-button v-for="entry in related" :key="entry.id" link type="primary" @click="selectDoc(entry.id)">{{
              entry.label
            }}</el-button>
          </div>
        </section>
      </el-main>
      <el-empty v-else description="暂无组件文档" />
    </el-scrollbar>

    <nav v-if="selected" class="docs-toc" aria-label="本页目录">
      <el-anchor :key="state.id" :container="documentRef?.wrapRef" :offset="26" select-scroll-top @click.prevent>
        <el-anchor-link href="#component-docs-basic" title="基础用法" />
        <el-anchor-link href="#component-docs-api" title="API" />
        <el-anchor-link href="#component-docs-usage" title="使用说明" />
      </el-anchor>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, reactive, ref, watch, type Component } from "vue";
import type { ScrollbarInstance } from "element-plus";
import { Code, Copy } from "@lucide/vue";
import { getAdminComponentDocs } from "@/demo";
import CodeBlock from "@/components/CodeBlock/index.vue";
import { copyText } from "@/utils/clipboard";
import DocsPagination from "./DocsPagination.vue";
import {
  createComponentDocsState,
  filterComponentDocs,
  getComponentDocsPage,
  openComponentDoc,
  returnComponentDoc
} from "./navigation";

defineOptions({ name: "DemoComponentDocs" });

const entries = getAdminComponentDocs();
const byId = new Map(entries.map(entry => [entry.id, entry]));
const groups = [...new Set(entries.map(entry => entry.group))];
const state = reactive(createComponentDocsState(entries[0]?.id ?? ""));
const documentRef = ref<ScrollbarInstance>();
const sourceVisible = ref(false);
const examples = new Map<string, Component>();
const selected = computed(() => byId.get(state.id));
const navigationEntries = computed(() => filterComponentDocs(entries, state.query));
const navigationGroups = computed(() =>
  groups
    .map(label => ({
      label,
      entries: navigationEntries.value.filter(entry => entry.group === label),
      open: Boolean((state.query && navigationEntries.value.length <= 6) || selected.value?.group === label)
    }))
    .filter(group => group.entries.length)
);
const backLabel = computed(() => byId.get(state.history.at(-1)?.id ?? "")?.label);
const apiTabs = [
  { key: "props", label: "属性 Props" },
  { key: "events", label: "事件 Events" },
  { key: "slots", label: "插槽 Slots" },
  { key: "methods", label: "方法 Exposes" }
];
const apiEntries = computed(() => selected.value?.api[state.api] ?? []);
const api = computed(() => getComponentDocsPage(apiEntries.value, state.apiPage, 6));
const related = computed(() =>
  (selected.value?.related ?? []).flatMap(id => {
    const entry = byId.get(id);
    return entry ? [entry] : [];
  })
);
const activeExample = computed(() => {
  const entry = selected.value;
  if (!entry?.example) return undefined;
  if (!examples.has(entry.id)) examples.set(entry.id, defineAsyncComponent(entry.example));
  return examples.get(entry.id);
});

watch(
  () => state.api,
  () => {
    state.apiPage = 1;
  },
  { flush: "sync" }
);
watch(
  () => state.id,
  () => {
    sourceVisible.value = false;
  }
);

/** 切换组件后展示文档页首，使返回入口始终可见。 */
async function selectDoc(id: string) {
  if (!byId.has(id)) return;
  openComponentDoc(state, id);
  await nextTick();
  documentRef.value?.setScrollTop(0);
}

/** 返回来源组件和原筛选、页码。 */
async function goBack() {
  returnComponentDoc(state);
  await nextTick();
  documentRef.value?.setScrollTop(0);
}

/** 复制当前用法示例，失败时保留可选取的源码。 */
async function copyCode() {
  if (!selected.value) return;
  try {
    await copyText(selected.value.code);
    ElMessage.success("代码已复制");
  } catch {
    sourceVisible.value = true;
    ElMessage.warning("请在源码中选择并复制代码");
  }
}
</script>

<style scoped lang="scss">
.component-docs {
  box-sizing: border-box;
  display: grid;
  flex: 1;
  grid-template-rows: minmax(0, 1fr);
  grid-template-columns: 188px minmax(0, 1fr);
  width: 100%;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}
.docs-sidebar {
  --el-aside-width: auto;

  min-width: 0;
  min-height: 0;
  padding: 20px 12px;
  overscroll-behavior: contain;
  border-right: 1px solid var(--admin-page-divider);
  details {
    margin-top: 8px;
  }
  summary {
    padding: 8px 4px;
    font-size: var(--el-font-size-base);
    color: var(--admin-page-text-secondary);
    cursor: pointer;
  }
}
.el-button.docs-nav-item {
  display: block;
  width: 100%;
  padding: 8px 10px;
  margin-bottom: 2px;
  font: inherit;
  font-size: var(--el-font-size-base);
  color: var(--admin-page-text-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-radius: 4px;
  &:hover {
    background: var(--el-fill-color-light);
  }
  &[aria-current="page"] {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }
}
.docs-mobile-picker {
  display: none;
}
.docs-content {
  min-width: 0;
  min-height: 0;
  container-type: size;
  :deep(.el-scrollbar__wrap) {
    overscroll-behavior: contain;
  }
}
.docs-document {
  min-width: 0;
  padding: 26px 28px;
  overflow: visible;
  color: var(--admin-page-text-primary);
  h1 {
    margin: 0;
    font-size: 26px;
    font-weight: 600;
    line-height: 1.4;
    overflow-wrap: anywhere;
  }
  h1 span {
    font-size: 22px;
  }
  h2 {
    margin: 0 0 12px;
    font-size: 17px;
    font-weight: 500;
  }
}
.docs-back {
  margin-bottom: 14px;
}
.docs-intro,
.docs-usage {
  margin: 12px 0 0;
  font-size: var(--el-font-size-medium);
  line-height: 1.9;
  color: var(--admin-page-text-secondary);
  white-space: pre-line;
}
.docs-section {
  margin-top: 30px;
  &:last-child {
    // 为末节保留可滚动空间，短文档也能定位到顶部并联动目录高亮。
    min-height: calc(100cqh - 52px);
  }
}
.docs-example {
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
}
.docs-preview {
  min-width: 0;
  padding: 20px;
  overflow-x: auto;
}
.docs-example-tools {
  display: flex;
  justify-content: flex-end;
  padding: 6px 10px;
  border-top: 1px solid var(--admin-page-divider);
  .docs-source-trigger {
    display: inline-flex;
  }
  .el-button {
    padding: 8px;
    margin-left: 0;
  }
}
.docs-api-table code {
  font-size: var(--el-font-size-base);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}
.docs-required {
  color: var(--el-color-danger);
}
.docs-related {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 16px;
  .el-button {
    margin-left: 0;
  }
}
.docs-toc {
  display: none;
  min-height: 0;
  padding: 30px 12px;
  overflow-y: auto;
  font-size: var(--el-font-size-base);
  .el-anchor {
    --el-anchor-bg-color: transparent;
  }
}

@media (width >= 1200px) {
  .component-docs {
    grid-template-columns: 188px minmax(0, 1fr) 140px;
  }
  .docs-toc {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
}

@media (width <= 720px) {
  .component-docs {
    grid-template-rows: auto minmax(0, 1fr);
    grid-template-columns: minmax(0, 1fr);
  }
  .docs-sidebar {
    display: none;
  }
  .docs-mobile-picker {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 10px;
    padding: 16px 16px 0;
  }
  .docs-document {
    padding: 20px 16px;
    h1 {
      font-size: 22px;
    }
    h1 span {
      font-size: 18px;
    }
  }
  .docs-preview {
    padding: 12px;
  }
}
</style>
