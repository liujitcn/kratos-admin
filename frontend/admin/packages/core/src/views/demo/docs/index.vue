<template>
  <div class="component-docs">
    <aside class="docs-sidebar" aria-label="组件目录">
      <el-input v-model="state.query" clearable placeholder="搜索组件名称…" aria-label="搜索组件名称" />
      <el-button class="docs-overview-link" text @click="selectDoc('')">组件总览</el-button>
      <nav aria-label="组件分类">
        <details v-for="group in navigationGroups" :key="group.label" :open="group.open">
          <summary>{{ group.label }}</summary>
          <button
            v-for="entry in group.entries"
            :key="entry.id"
            class="docs-nav-item"
            type="button"
            :aria-current="state.id === entry.id ? 'page' : undefined"
            @click="selectDoc(entry.id)"
          >
            {{ entry.label }}
          </button>
        </details>
      </nav>
      <el-empty v-if="!navigationEntries.length" description="没有匹配的组件" :image-size="50" />
    </aside>

    <div class="docs-mobile-picker">
      <el-input v-model="state.query" clearable placeholder="搜索组件名称…" aria-label="搜索组件名称" />
      <el-select :model-value="state.id" aria-label="选择组件" @change="selectDoc">
        <el-option label="组件总览" value="" />
        <el-option-group v-for="group in navigationGroups" :key="group.label" :label="group.label">
          <el-option v-for="entry in group.entries" :key="entry.id" :label="entry.label" :value="entry.id" />
        </el-option-group>
      </el-select>
    </div>

    <main ref="documentRef" class="docs-document" aria-label="组件说明">
      <template v-if="!selected">
        <h1>组件总览</h1>
        <div class="docs-scopes" aria-label="组件所属范围">
          <el-button
            v-for="owner in owners"
            :key="owner"
            :type="state.owner === owner ? 'primary' : 'default'"
            plain
            size="small"
            :aria-pressed="state.owner === owner"
            @click="state.owner = owner"
            >{{ owner }}</el-button
          >
        </div>
        <section v-for="group in overviewGroups" :key="group.label" class="docs-overview-group">
          <h2>{{ group.label }}</h2>
          <div class="docs-cards">
            <button
              v-for="entry in group.entries"
              :key="entry.id"
              class="docs-component-card"
              type="button"
              @click="selectDoc(entry.id)"
            >
              <strong>{{ entry.label }}</strong>
              <p>{{ entry.description }}</p>
            </button>
          </div>
        </section>
        <el-empty v-if="!filteredEntries.length" description="没有匹配的组件" />
        <DocsPagination
          :page="overview.page"
          :page-size="state.overviewSize"
          :total="filteredEntries.length"
          :sizes="[4, 8, 12]"
          @update:page="state.overviewPage = $event"
          @update:page-size="state.overviewSize = $event"
        />
      </template>

      <template v-else>
        <el-button class="docs-back" size="small" @click="goBack">返回{{ backLabel }}</el-button>
        <h1>
          {{ selected.id }} <span>{{ selected.label }}</span>
        </h1>
        <p class="docs-intro">{{ selected.description }}</p>

        <section id="component-docs-basic" class="docs-section">
          <h2>基础用法</h2>
          <div v-show="selected.example" class="docs-example">
            <div class="docs-preview">
              <KeepAlive :max="20">
                <component :is="activeExample" v-if="activeExample" :key="selected.id" :component-id="selected.id" />
              </KeepAlive>
            </div>
            <div class="docs-example-tools">
              <el-button text size="small" @click="copyCode">复制代码</el-button>
              <el-button text size="small" :aria-expanded="sourceVisible" @click="sourceVisible = !sourceVisible">{{
                sourceVisible ? "收起源码" : "查看源码"
              }}</el-button>
            </div>
            <CodeBlock v-if="sourceVisible" :code="selected.code" language="html" />
          </div>
          <CodeBlock v-if="!selected.example" :code="selected.code" language="html" />
        </section>

        <section id="component-docs-api" class="docs-section">
          <h2>{{ selected.id }} API</h2>
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
          <h2>使用说明</h2>
          <p class="docs-usage">{{ selected.usage }}</p>
          <div v-if="related.length" class="docs-related">
            <el-button v-for="entry in related" :key="entry.id" link type="primary" @click="selectDoc(entry.id)">{{
              entry.label
            }}</el-button>
          </div>
        </section>
      </template>
    </main>

    <nav v-if="selected" class="docs-toc" aria-label="本页目录">
      <strong>本页目录</strong>
      <el-button text @click="scrollTo('basic')">基础用法</el-button>
      <el-button text @click="scrollTo('api')">API</el-button>
      <el-button text @click="scrollTo('usage')">使用说明</el-button>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { computed, defineAsyncComponent, nextTick, reactive, ref, watch, type Component } from "vue";
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
const owners = ["全部", ...new Set(entries.map(entry => entry.owner))];
const groups = [...new Set(entries.map(entry => entry.group))];
const state = reactive(createComponentDocsState());
const documentRef = ref<HTMLElement>();
const sourceVisible = ref(false);
const examples = new Map<string, Component>();
const selected = computed(() => byId.get(state.id));
const filteredEntries = computed(() => filterComponentDocs(entries, state.query, state.owner));
const navigationEntries = computed(() => filterComponentDocs(entries, state.query, "全部"));
const navigationGroups = computed(() =>
  groups
    .map(label => ({
      label,
      entries: navigationEntries.value.filter(entry => entry.group === label),
      open: Boolean((state.query && navigationEntries.value.length <= 6) || selected.value?.group === label)
    }))
    .filter(group => group.entries.length)
);
const overview = computed(() => getComponentDocsPage(filteredEntries.value, state.overviewPage, state.overviewSize));
const overviewGroups = computed(() =>
  groups
    .map(label => ({ label, entries: overview.value.items.filter(entry => entry.group === label) }))
    .filter(group => group.entries.length)
);
const backLabel = computed(() => byId.get(state.history.at(-1)?.id ?? "")?.label ?? "组件总览");
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
  () => [state.query, state.owner, state.overviewSize],
  () => {
    state.overviewPage = 1;
  },
  { flush: "sync" }
);
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
  if (id && !byId.has(id)) return;
  openComponentDoc(state, id);
  await nextTick();
  documentRef.value?.scrollIntoView({ block: "start" });
}

/** 返回来源组件和原筛选、页码。 */
async function goBack() {
  returnComponentDoc(state);
  await nextTick();
  documentRef.value?.scrollIntoView({ block: "start" });
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

/** 定位当前组件的说明分项。 */
function scrollTo(section: string) {
  documentRef.value?.querySelector(`#component-docs-${section}`)?.scrollIntoView({ block: "start", behavior: "smooth" });
}
</script>

<style scoped lang="scss">
.component-docs {
  display: grid;
  flex-shrink: 0;
  grid-template-columns: 188px minmax(0, 1fr);
  align-items: start;
  width: 100%;
  min-width: 0;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}
.docs-sidebar {
  align-self: stretch;
  min-width: 0;
  padding: 20px 12px;
  border-right: 1px solid var(--admin-page-divider);
  details {
    margin-top: 8px;
  }
  summary {
    padding: 8px 4px;
    font-size: 12px;
    color: var(--admin-page-text-secondary);
    cursor: pointer;
  }
}
.docs-overview-link {
  width: 100%;
  margin-top: 12px;
  margin-bottom: 8px;
}
.docs-nav-item {
  display: block;
  width: 100%;
  padding: 8px 10px;
  margin-bottom: 2px;
  font: inherit;
  font-size: 12px;
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
.docs-document {
  min-width: 0;
  padding: 26px 28px;
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
  font-size: 13px;
  line-height: 1.9;
  color: var(--admin-page-text-secondary);
  white-space: pre-line;
}
.docs-scopes {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 20px;
  .el-button {
    margin-left: 0;
  }
}
.docs-overview-group {
  margin-top: 24px;
}
.docs-cards {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.docs-component-card {
  padding: 16px;
  font: inherit;
  color: var(--admin-page-text-primary);
  text-align: left;
  cursor: pointer;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  strong {
    font-size: 14px;
    font-weight: 500;
  }
  p {
    margin: 8px 0 0;
    font-size: 12px;
    line-height: 1.8;
    color: var(--admin-page-text-secondary);
  }
  &:hover {
    border-color: var(--el-color-primary);
  }
}
.docs-section {
  margin-top: 30px;
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
}
.docs-api-table code {
  font-size: 12px;
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
  padding: 30px 12px;
  font-size: 12px;
  .el-button {
    justify-content: flex-start;
    margin-left: 0;
  }
}
@media (min-width: 1200px) {
  .component-docs {
    grid-template-columns: 188px minmax(0, 1fr) 115px;
  }
  .docs-toc {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }
}
@media (max-width: 720px) {
  .component-docs {
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
  .docs-cards {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
