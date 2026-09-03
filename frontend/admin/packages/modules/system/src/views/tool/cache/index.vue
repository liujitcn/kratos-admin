<template>
  <div class="table-box cache-page">
    <ProTable ref="proTable" row-key="key" :columns="columns" :request-api="requestCacheTable" />

    <ProDialog
      v-model="detail.visible"
      :title="t('system.cache.detail.title')"
      width="min(960px, calc(100vw - 32px))"
      destroy-on-close
      :show-footer="false"
    >
      <template v-if="detail.entry">
        <div class="cache-detail-meta">
          <code class="cache-detail-key" :title="detail.entry.key">{{ detail.entry.key }}</code>
          <el-tag size="small" effect="plain">{{ cacheTypeLabel(detail.entry.type) }}</el-tag>
          <span class="cache-detail-ttl">{{ formatTtl(detail.entry.ttl_seconds) }}</span>
        </div>

        <div v-if="detail.entry.type === 'hash'" class="cache-fields">
          <el-empty v-if="!detail.entry.fields?.length" :description="t('common.message.no_data')" />
          <article v-for="field in detail.entry.fields" :key="field.key" class="cache-field">
            <div class="cache-field-header">
              <code class="cache-field-key">{{ field.key }}</code>
              <el-tag size="small" effect="plain" type="info">{{ cacheValueKindLabel(field.value) }}</el-tag>
            </div>
            <CacheValueContent :value="field.value" />
          </article>
        </div>

        <section v-else class="cache-string-value">
          <div class="cache-value-header">
            <span>{{ t("system.cache.field.value") }}</span>
            <el-tag size="small" effect="plain" type="info">{{ cacheValueKindLabel(detail.entry.value) }}</el-tag>
          </div>
          <CacheValueContent :value="detail.entry.value" />
        </section>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, reactive, ref } from "vue";
import { View } from "@element-plus/icons-vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import type { ColumnProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { t } from "@liujitcn/kratos-admin-core";
import RichTextPreview from "@liujitcn/kratos-admin-core/components/RichTextPreview/index.vue";
import { defCacheService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/cache";
import type { CacheEntry, PageCacheRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/cache";

defineOptions({ name: "Cache", inheritAttrs: false });

const proTable = ref<ProTableInstance>();
const detail = reactive<{ visible: boolean; entry?: CacheEntry }>({ visible: false });

/** 缓存值展示格式。 */
type CacheValueKind = "text" | "json" | "rich-text";

/** 缓存值解析结果。 */
interface CacheValuePresentation {
  /** 值的展示格式。 */
  kind: CacheValueKind;
  /** 值的展示文本。 */
  value: string;
}

/** 缓存值内容组件，按值的格式选择安全富文本或代码块展示。 */
const CacheValueContent = defineComponent({
  name: "CacheValueContent",
  props: {
    value: { type: String, default: "" }
  },
  setup(props) {
    return () => {
      const presentation = getCacheValuePresentation(props.value);
      if (presentation.kind === "rich-text") {
        return h(RichTextPreview, { class: "cache-rich-text", modelValue: props.value });
      }

      return h(
        "pre",
        { class: ["cache-value", { "cache-value--plain": presentation.kind === "text" }] },
        presentation.value || "-"
      );
    };
  }
});

const columns = computed<ColumnProps[]>(() => [
  {
    prop: "key",
    label: t("system.cache.field.key"),
    minWidth: 280,
    search: { el: "input", props: { clearable: true } },
    render: scope => h("code", { class: "cache-key-cell", title: scope.row.key }, scope.row.key)
  },
  {
    prop: "type",
    label: t("system.cache.field.type"),
    width: 100,
    render: scope => h(ElTag, { size: "small", effect: "plain" }, () => cacheTypeLabel(scope.row.type))
  },
  {
    prop: "value",
    label: t("system.cache.field.value"),
    minWidth: 260,
    showOverflowTooltip: false,
    render: scope => renderCachePreview(scope.row as CacheEntry)
  },
  {
    prop: "ttl_seconds",
    label: t("system.cache.field.ttl"),
    width: 130,
    render: scope => formatTtl((scope.row as CacheEntry).ttl_seconds)
  },
  {
    prop: "expires_at",
    label: t("system.cache.field.expires_at"),
    minWidth: 185,
    render: scope => formatDate(scope.row.expires_at)
  },
  {
    prop: "created_at",
    label: t("system.cache.field.created_at"),
    minWidth: 185,
    render: scope => formatDate(scope.row.created_at)
  },
  {
    prop: "updated_at",
    label: t("system.cache.field.updated_at"),
    minWidth: 185,
    render: scope => formatDate(scope.row.updated_at)
  },
  {
    prop: "operation",
    label: t("common.field.operation"),
    width: 90,
    fixed: "right",
    cellType: "actions",
    actions: [
      {
        label: t("common.action.view"),
        type: "primary",
        link: true,
        icon: View,
        onClick: scope => openDetail(scope.row as CacheEntry)
      }
    ]
  }
]);

/** 请求缓存分页列表。 */
async function requestCacheTable(params: PageCacheRequest) {
  const response = await defCacheService.PageCache(buildPageRequest(params));
  return { data: { list: response.cache_entries ?? [], total: response.total } };
}

/** 打开缓存条目详情。 */
function openDetail(entry: CacheEntry) {
  detail.entry = entry;
  detail.visible = true;
}

/** 渲染列表中的缓存值摘要，避免长文本撑开表格或触发整段浮层。 */
function renderCachePreview(entry: CacheEntry) {
  if (entry.type === "hash") {
    return h(
      "div",
      { class: "cache-preview" },
      h("span", { class: "cache-preview-text" }, t("system.cache.value.hash_fields", { count: entry.fields?.length ?? 0 }))
    );
  }

  const presentation = getCacheValuePresentation(entry.value);
  return h("div", { class: "cache-preview" }, [
    h(ElTag, { size: "small", effect: "plain", type: "info" }, () => cacheValueKindLabel(entry.value)),
    h("span", { class: "cache-preview-text" }, summarizeCacheValue(presentation.value, presentation.kind))
  ]);
}

/** 获取缓存类型的本地化名称。 */
function cacheTypeLabel(type: string) {
  return type === "hash" ? t("system.cache.type.hash") : t("system.cache.type.string");
}

/** 获取缓存值格式的本地化名称。 */
function cacheValueKindLabel(value: string) {
  const { kind } = getCacheValuePresentation(value);
  return t(`system.cache.value.kind.${kind}`);
}

/** 解析缓存值，优先识别对象/数组 JSON，再识别 HTML 富文本。 */
function getCacheValuePresentation(value: string): CacheValuePresentation {
  const normalizedValue = value ?? "";
  try {
    const parsedValue: unknown = JSON.parse(normalizedValue);
    if (parsedValue !== null && typeof parsedValue === "object") {
      return { kind: "json", value: JSON.stringify(parsedValue, null, 2) };
    }
  } catch {
    // 普通文本和富文本不是 JSON，继续按内容特征识别。
  }

  if (isRichTextValue(normalizedValue)) return { kind: "rich-text", value: normalizedValue };
  return { kind: "text", value: normalizedValue };
}

/** 判断字符串是否包含可渲染的 HTML 元素。 */
function isRichTextValue(value: string) {
  if (!/<[a-z][^>]*>/i.test(value)) return false;
  if (typeof DOMParser === "undefined") return /<\/[a-z][^>]*>/i.test(value);
  const document = new DOMParser().parseFromString(value, "text/html");
  return document.body.children.length > 0;
}

/** 将值压缩为适合列表单元格的单行摘要。 */
function summarizeCacheValue(value: string, kind: CacheValueKind, maxLength = 120) {
  let summarySource = value;
  if (kind === "rich-text") {
    const document = typeof DOMParser === "undefined" ? undefined : new DOMParser().parseFromString(value, "text/html");
    summarySource = document?.body.textContent ?? value.replace(/<[^>]+>/g, " ");
  }
  const summary = summarySource.replace(/\s+/g, " ").trim();
  if (!summary) return "-";
  return summary.length > maxLength ? `${summary.slice(0, maxLength)}...` : summary;
}

/** 格式化缓存剩余有效期。 */
function formatTtl(ttl: number) {
  if (ttl < 0) return t("system.cache.value.persistent");
  if (ttl === 0) return t("system.cache.value.expiring");
  if (ttl < 60) return `${ttl}s`;
  if (ttl < 3600) return `${Math.floor(ttl / 60)}m ${ttl % 60}s`;
  return `${Math.floor(ttl / 3600)}h ${Math.floor((ttl % 3600) / 60)}m`;
}

/** 格式化时间字段。 */
function formatDate(value: string) {
  if (!value) return "-";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}
</script>

<style lang="scss" scoped>
.cache-page {
  padding: 16px;
}
:deep(.cache-preview) {
  display: flex;
  gap: 8px;
  align-items: center;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
}
:deep(.cache-preview-text) {
  display: block;
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  color: var(--admin-page-text-secondary);
  white-space: nowrap;
}
.cache-key-cell {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cache-detail-meta {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 16px;
}
.cache-detail-key {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  font-family: var(--el-font-family-monospace);
  color: var(--admin-page-text-primary);
  white-space: nowrap;
}
.cache-detail-ttl {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.cache-fields {
  max-height: 62vh;
  padding-right: 4px;
  overflow: auto;
}
.cache-field,
.cache-string-value {
  padding: 14px;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}
.cache-field + .cache-field {
  margin-top: 12px;
}
.cache-field-header,
.cache-value-header {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}
.cache-field-key {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  font-family: var(--el-font-family-monospace);
  color: var(--admin-page-text-primary);
  white-space: nowrap;
}
.cache-value-header {
  font-size: 13px;
  color: var(--admin-page-text-secondary);
}
.cache-value {
  max-height: 460px;
  padding: 14px;
  margin: 0;
  overflow: auto;
  font-family: var(--el-font-family-monospace);
  font-size: 13px;
  line-height: 1.55;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
  white-space: pre-wrap;
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--admin-page-radius);
}
.cache-value--plain {
  font-family: inherit;
  font-size: 14px;
}
.cache-rich-text {
  max-height: 460px;
  padding: 2px 4px;
  overflow: auto;
}
</style>
