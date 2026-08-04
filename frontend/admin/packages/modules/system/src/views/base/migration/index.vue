<!-- 数据库迁移升级历史 -->
<template>
  <div v-loading="loading" class="table-box migration-page">
    <main class="migration-content">
      <el-form class="migration-filters" :model="filters" inline @submit.prevent="handleSearch">
        <el-form-item :label="t('system.base.migration.field.module')">
          <el-input
            v-model="filters.module"
            clearable
            :placeholder="t('system.base.migration.placeholder.module')"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item :label="t('system.base.migration.field.data_source')">
          <el-input
            v-model="filters.data_source"
            clearable
            :placeholder="t('system.base.migration.placeholder.data_source')"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item :label="t('system.base.migration.field.version')">
          <el-input
            v-model="filters.version"
            clearable
            :placeholder="t('system.base.migration.placeholder.version')"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item class="migration-filters__actions">
          <el-button type="primary" :icon="Search" native-type="submit">{{ t("common.action.search") }}</el-button>
          <el-button :icon="Refresh" @click="handleReset">{{ t("common.action.reset") }}</el-button>
        </el-form-item>
      </el-form>

      <el-empty v-if="!loading && !histories.length" :description="t('system.base.migration.message.empty_history')" />

      <section v-else class="migration-workspace" :aria-label="t('system.base.migration.title.history')">
        <aside class="migration-list-panel">
          <div class="migration-list-scroll" role="listbox" :aria-label="t('system.base.migration.title.list')">
            <button
              v-for="history in histories"
              :key="history.id"
              type="button"
              class="migration-list-item"
              :class="{ 'is-active': selectedMigrationId === history.id }"
              :aria-selected="selectedMigrationId === history.id"
              @click="selectMigration(history.id)"
            >
              <div class="migration-list-item__top">
                <strong>{{ history.version }}</strong>
              </div>
              <div class="migration-list-item__data-source">
                {{ history.module || t("system.base.migration.value.default_module") }} · {{ history.data_source || "default" }}
              </div>
              <time :datetime="history.created_at">{{ formatDate(history.created_at) }}</time>
            </button>
          </div>

          <div v-if="pageable.total > pageable.page_size" class="migration-pagination">
            <span class="migration-pagination__total">{{ t("system.base.migration.message.total", { total: pageable.total }) }}</span>
            <el-pagination
              background
              small
              layout="prev, pager, next"
              :current-page="pageable.page_num"
              :page-size="pageable.page_size"
              :pager-count="5"
              :total="pageable.total"
              @current-change="handleCurrentPageChange"
            />
          </div>
        </aside>

        <section class="migration-detail-panel">
          <div v-if="detailLoading && !selectedMigration" class="detail-loading">
            <el-skeleton :rows="8" animated />
          </div>
          <template v-else-if="selectedMigration">
            <header class="detail-header">
              <div class="detail-header__title">
                <span class="detail-header__data-source">
                  {{ selectedMigration.module || t("system.base.migration.value.default_module") }} ·
                  {{ selectedMigration.data_source || "default" }}
                </span>
                <div class="detail-header__version">
                  <h2>{{ selectedMigration.version }}</h2>
                </div>
              </div>
              <div class="detail-header__meta">
                <time :datetime="selectedMigration.created_at">{{ selectedMigration.created_at }}</time>
              </div>
            </header>

            <div v-loading="detailLoading" class="detail-scroll">
              <template v-if="!detailLoading">
                <section v-if="selectedMigration.description" class="detail-section">
                  <div class="detail-section__title">
                    <el-icon><Document /></el-icon>
                    <span>{{ t("system.base.migration.section.description") }}</span>
                  </div>
                  <MarkdownPreview
                    class="migration-markdown"
                    :model-value="selectedMigration.description"
                    :is-dark="globalStore.isDark"
                    max-code-height="360px"
                  />
                </section>

                <section v-if="selectedMigration.up_sql || selectedMigration.down_sql" class="detail-section">
                  <div class="detail-section__title">
                    <el-icon><Files /></el-icon>
                    <span>{{ t("system.base.migration.section.sql") }}</span>
                  </div>
                  <el-collapse class="release-sql">
                    <el-collapse-item v-if="selectedMigration.up_sql" name="up">
                      <template #title>
                        <span class="sql-title">
                          <el-icon><DocumentAdd /></el-icon>
                          <span>{{ t("system.base.migration.field.up_script") }}</span>
                          <code>up.sql</code>
                        </span>
                      </template>
                      <div class="sql-panel">
                        <pre class="sql-code"><code>{{ selectedMigration.up_sql }}</code></pre>
                        <el-tooltip :content="t('system.base.migration.action.copy_up_script')" placement="top">
                          <el-button
                            class="sql-copy"
                            text
                            circle
                            :icon="CopyDocument"
                            :aria-label="t('system.base.migration.action.copy_up_script')"
                            @click.stop="copySql(selectedMigration.up_sql)"
                          />
                        </el-tooltip>
                      </div>
                    </el-collapse-item>
                    <el-collapse-item v-if="selectedMigration.down_sql" name="down">
                      <template #title>
                        <span class="sql-title">
                          <el-icon><DocumentRemove /></el-icon>
                          <span>{{ t("system.base.migration.field.down_script") }}</span>
                          <code>down.sql</code>
                        </span>
                      </template>
                      <div class="sql-panel">
                        <pre class="sql-code"><code>{{ selectedMigration.down_sql }}</code></pre>
                        <el-tooltip :content="t('system.base.migration.action.copy_down_script')" placement="top">
                          <el-button
                            class="sql-copy"
                            text
                            circle
                            :icon="CopyDocument"
                            :aria-label="t('system.base.migration.action.copy_down_script')"
                            @click.stop="copySql(selectedMigration.down_sql)"
                          />
                        </el-tooltip>
                      </div>
                    </el-collapse-item>
                  </el-collapse>
                </section>

                <div v-if="!hasDetailContent" class="detail-empty">
                  <el-icon><Document /></el-icon>
                  <span>{{ t("system.base.migration.message.empty_detail") }}</span>
                </div>
              </template>
            </div>
          </template>
          <el-empty v-else :description="t('system.base.migration.message.select_record')" />
        </section>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { CopyDocument, Document, DocumentAdd, DocumentRemove, Files, Refresh, Search } from "@element-plus/icons-vue";
import MarkdownPreview from "@liujitcn/kratos-admin-core/components/MarkdownPreview/index.vue";
import { useGlobalStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { defBaseMigrationService } from "@liujitcn/kratos-admin-system/api/system/base_migration";
import type {
  BaseMigration,
  BaseMigrationListItem,
  PageBaseMigrationRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_migration";
import { getCurrentLocale, t } from "@liujitcn/kratos-admin-core";

defineOptions({
  name: "BaseMigration",
  inheritAttrs: false
});

/** 数据库升级历史筛选条件。 */
interface MigrationFilters {
  /** 迁移版本号。 */
  version: string | undefined;
  /** 迁移模块名称。 */
  module: string | undefined;
  /** 数据源名称。 */
  data_source: string | undefined;
}

/** 数据库升级历史分页状态。 */
interface MigrationPageable {
  /** 当前页码。 */
  page_num: number;
  /** 每页条数。 */
  page_size: number;
  /** 总记录数。 */
  total: number;
}

const globalStore = useGlobalStore();
const loading = ref(false);
const detailLoading = ref(false);
const histories = ref<BaseMigrationListItem[]>([]);
const selectedMigrationId = ref<number | null>(null);
const selectedMigration = ref<BaseMigration | null>(null);
const detailRequestToken = ref(0);
const filters = reactive<MigrationFilters>({
  version: undefined,
  module: undefined,
  data_source: undefined
});
const pageable = reactive<MigrationPageable>({
  page_num: 1,
  page_size: 10,
  total: 0
});
const hasDetailContent = computed(() =>
  Boolean(selectedMigration.value?.description || selectedMigration.value?.up_sql || selectedMigration.value?.down_sql)
);

/**
 * 加载数据库升级历史列表。
 */
async function loadMigrationHistory() {
  loading.value = true;
  try {
    const params: PageBaseMigrationRequest = {
      data_source: filters.data_source ?? "",
      version: filters.version,
      module: filters.module,
      page_num: pageable.page_num,
      page_size: pageable.page_size
    };
    const data = await defBaseMigrationService.PageBaseMigration(buildPageRequest(params));
    histories.value = data.base_migrations ?? [];
    pageable.total = Number(data.total ?? 0);
    const nextId = histories.value.find(item => item.id === selectedMigrationId.value)?.id ?? histories.value[0]?.id;
    if (nextId === undefined) {
      selectedMigrationId.value = null;
      selectedMigration.value = null;
      detailRequestToken.value += 1;
    } else {
      await selectMigration(nextId);
    }
  } finally {
    loading.value = false;
  }
}

/**
 * 选择升级记录并查询右侧详情。
 */
async function selectMigration(id: number) {
  selectedMigrationId.value = id;
  selectedMigration.value = null;
  detailLoading.value = true;
  const requestToken = detailRequestToken.value + 1;
  detailRequestToken.value = requestToken;
  try {
    const detail = await defBaseMigrationService.GetBaseMigration({ id });
    if (detailRequestToken.value === requestToken && selectedMigrationId.value === id) {
      selectedMigration.value = detail;
    }
  } finally {
    if (detailRequestToken.value === requestToken) {
      detailLoading.value = false;
    }
  }
}

/**
 * 按版本号重新查询升级历史。
 */
function handleSearch() {
  pageable.page_num = 1;
  loadMigrationHistory();
}

/**
 * 清空版本筛选条件并重新查询。
 */
function handleReset() {
  filters.version = undefined;
  filters.module = undefined;
  filters.data_source = undefined;
  handleSearch();
}

/**
 * 切换升级历史分页。
 */
function handleCurrentPageChange(page: number) {
  pageable.page_num = page;
  loadMigrationHistory();
}

/**
 * 复制 SQL 脚本内容。
 */
async function copySql(sql: string) {
  try {
    await navigator.clipboard.writeText(sql);
    ElMessage.success(t("system.base.migration.message.copy_success"));
  } catch {
    ElMessage.error(t("system.base.migration.message.copy_failed"));
  }
}

/**
 * 格式化列表日期。
 */
function formatDate(value: string) {
  if (!value) return "--";
  const date = new Date(value.replace(" ", "T"));
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(getCurrentLocale(), {
    year: "numeric",
    month: "short",
    day: "numeric"
  }).format(date);
}

onMounted(() => {
  loadMigrationHistory();
});
</script>

<style scoped lang="scss">
.migration-page {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  width: 100%;
  min-height: 100%;
  padding: 0;
  color: var(--admin-page-text-primary);
  background: var(--el-bg-color-page);
}

.migration-content {
  box-sizing: border-box;
  display: flex;
  flex: 1;
  flex-direction: column;
  width: 100%;
  min-height: 0;
  margin: 0;
  overflow: hidden;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}

.migration-filters {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 0 12px;
  width: 100%;
  padding: 18px 24px 14px;
  margin: 0;
  border-bottom: 1px solid var(--admin-page-divider);
}

.migration-filters :deep(.el-form-item) {
  margin: 0 0 8px;
}

.migration-filters :deep(.el-input) {
  width: 220px;
}

.migration-filters__actions {
  display: flex;
  gap: 8px;
}

.migration-filters__actions :deep(.el-form-item__content) {
  gap: 8px;
}

.migration-workspace {
  display: grid;
  flex: 1;
  grid-template-columns: 360px minmax(0, 1fr);
  gap: 0;
  width: 100%;
  height: auto;
  min-height: 440px;
  max-height: none;
  margin: 0;
}

.migration-list-panel,
.migration-detail-panel {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  background: transparent;
}

.migration-detail-panel {
  border-left: 1px solid var(--admin-page-divider);
}

.migration-list-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.migration-list-item {
  display: block;
  width: 100%;
  padding: 14px 18px;
  color: var(--admin-page-text-primary);
  text-align: left;
  cursor: pointer;
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--admin-page-divider);
  transition:
    background-color 0.2s ease,
    box-shadow 0.2s ease;
}

.migration-list-item:hover {
  background: var(--el-fill-color-light);
}

.migration-list-item.is-active {
  background: var(--admin-page-accent-soft-bg);
  box-shadow: inset 3px 0 0 var(--el-color-primary);
}

.migration-list-item__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.migration-list-item__top strong {
  min-width: 0;
  overflow: hidden;
  font-size: 15px;
  font-weight: 650;
  color: var(--el-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.migration-list-item__data-source {
  margin-top: 8px;
  overflow: hidden;
  font-size: 13px;
  color: var(--admin-page-text-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.migration-list-item time {
  display: block;
  margin-top: 5px;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}

.migration-pagination {
  display: flex;
  align-items: center;
  gap: 12px;
  justify-content: flex-end;
  min-width: 0;
  overflow: hidden;
  padding: 12px 18px;
  border-top: 1px solid var(--admin-page-divider);
}

.migration-pagination__total {
  flex: 0 0 auto;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
  white-space: nowrap;
}

.migration-pagination :deep(.el-pagination) {
  flex: 1 1 auto;
  min-width: 0;
  margin: 0;
  overflow-x: auto;
  padding: 0;
  white-space: nowrap;
}

.detail-header {
  display: flex;
  flex: 0 0 auto;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding: 22px 24px 20px;
  border-bottom: 1px solid var(--admin-page-divider);
}

.detail-header__title {
  min-width: 0;
}

.detail-header__data-source {
  display: block;
  overflow: hidden;
  font-size: 12px;
  font-weight: 600;
  color: var(--admin-page-text-secondary);
  letter-spacing: 0.04em;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-header__version {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
  margin-top: 8px;
}

.detail-header h2 {
  margin: 0;
  font-size: 24px;
  font-weight: 650;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.detail-header__meta {
  display: flex;
  flex: 0 0 auto;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  padding-top: 3px;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
  text-align: right;
}

.detail-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 24px;
}

.detail-section + .detail-section {
  margin-top: 26px;
}

.detail-section__title {
  display: flex;
  gap: 7px;
  align-items: center;
  padding-bottom: 10px;
  margin-bottom: 16px;
  font-size: 14px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
  border-bottom: 1px solid var(--admin-page-divider);
}

.detail-section__title :deep(.el-icon) {
  color: var(--el-color-primary);
}

.migration-markdown {
  width: 100%;
  max-width: 100%;
  overflow-wrap: anywhere;
}

.release-sql {
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}

.release-sql :deep(.el-collapse) {
  border: 0;
}

.release-sql :deep(.el-collapse-item__header) {
  height: 46px;
  padding: 0 14px;
  color: var(--admin-page-text-primary);
  background: var(--admin-page-card-bg-soft);
  border-bottom-color: var(--admin-page-card-border-soft);
}

.release-sql :deep(.el-collapse-item__wrap) {
  background: var(--admin-page-card-bg);
  border-bottom-color: var(--admin-page-card-border-soft);
}

.release-sql :deep(.el-collapse-item__content) {
  padding: 14px;
}

.sql-title {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  min-width: 0;
  font-size: 13px;
  font-weight: 600;
}

.sql-title :deep(.el-icon) {
  color: var(--el-color-primary);
}

.sql-title code {
  padding: 2px 6px;
  font-size: 11px;
  font-weight: 500;
  color: var(--admin-page-text-secondary);
  background: var(--admin-page-card-bg-muted);
  border-radius: 4px;
}

.sql-panel {
  position: relative;
  min-width: 0;
}

.sql-code {
  max-height: 360px;
  padding: 14px 48px 14px 16px;
  margin: 0;
  overflow: auto;
  color: var(--admin-page-text-primary);
  white-space: pre;
  background: var(--admin-page-card-bg-muted);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}

.sql-code code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 12px;
  line-height: 1.65;
}

.sql-copy {
  position: absolute;
  top: 6px;
  right: 6px;
  color: var(--admin-page-text-secondary);
}

.sql-copy:hover {
  color: var(--el-color-primary);
}

.detail-empty {
  display: flex;
  min-height: 180px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 13px;
  color: var(--admin-page-text-placeholder);
}

.detail-empty :deep(.el-icon) {
  font-size: 28px;
}

.detail-loading {
  padding: 24px;
}

@media (max-width: 900px) {
  .migration-page {
    padding: 12px;
  }

  .migration-workspace {
    grid-template-columns: 280px minmax(0, 1fr);
  }
}

@media (max-width: 720px) {
  .migration-filters {
    padding-right: 16px;
    padding-left: 16px;
  }

  .migration-filters :deep(.el-input) {
    width: 100%;
  }

  .migration-workspace {
    display: flex;
    height: auto;
    min-height: 0;
    flex-direction: column;
  }

  .migration-list-panel {
    height: 320px;
    flex: 0 0 auto;
    border-bottom: 1px solid var(--admin-page-divider);
  }

  .migration-detail-panel {
    min-height: 480px;
    border-left: 0;
  }

  .migration-pagination {
    justify-content: center;
    padding-right: 12px;
    padding-left: 12px;
    overflow-x: auto;
  }

  .detail-header,
  .detail-scroll {
    padding-right: 16px;
    padding-left: 16px;
  }

  .detail-header {
    flex-direction: column;
    gap: 10px;
  }

  .detail-header__meta {
    align-items: flex-start;
    text-align: left;
  }
}
</style>
