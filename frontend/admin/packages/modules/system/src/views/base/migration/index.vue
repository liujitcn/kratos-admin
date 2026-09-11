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
          <el-button type="primary" native-type="submit"
            ><template #icon><Search /></template>{{ t("common.action.search") }}</el-button
          >
          <el-button @click="handleReset"
            ><template #icon><Refresh /></template>{{ t("common.action.reset") }}</el-button
          >
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
            <span class="migration-pagination__total">{{
              t("system.base.migration.message.total", { total: pageable.total })
            }}</span>
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
            <div v-loading="detailLoading" class="detail-scroll">
              <template v-if="!detailLoading">
                <el-tabs v-if="hasDetailContent" v-model="activeFilePath" class="migration-file-tabs">
                  <el-tab-pane
                    v-for="file in selectedMigration.files"
                    :key="file.path"
                    :name="file.path"
                    :label="file.path.split('/').pop()"
                    lazy
                  >
                    <template v-if="activeFilePath === file.path">
                      <div class="migration-file-path">{{ file.path }}</div>
                      <MarkdownPreview
                        v-if="file.path.endsWith('.md')"
                        class="migration-markdown"
                        :model-value="file.content"
                        :is-dark="globalStore.isDark"
                        max-code-height="360px"
                      />
                      <div v-else class="sql-panel">
                        <pre class="sql-code"><code>{{ file.content }}</code></pre>
                        <el-tooltip
                          :content="
                            t(
                              file.path.endsWith('.down.sql')
                                ? 'system.base.migration.action.copy_down_script'
                                : 'system.base.migration.action.copy_up_script'
                            )
                          "
                          placement="top"
                        >
                          <el-button
                            class="sql-copy"
                            text
                            circle
                            :aria-label="
                              t(
                                file.path.endsWith('.down.sql')
                                  ? 'system.base.migration.action.copy_down_script'
                                  : 'system.base.migration.action.copy_up_script'
                              )
                            "
                            @click="copySql(file.content)"
                            ><template #icon><CopyDocument /></template
                          ></el-button>
                        </el-tooltip>
                      </div>
                    </template>
                  </el-tab-pane>
                </el-tabs>

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
import MarkdownPreview from "@liujitcn/kratos-admin-core/components/MarkdownPreview/index.vue";
import { useGlobalStore } from "@liujitcn/kratos-admin-core/stores/runtime";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { defBaseMigrationService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_migration";
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
const activeFilePath = ref("");
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
const hasDetailContent = computed(() => Boolean(selectedMigration.value?.files.length));

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
  activeFilePath.value = "";
  detailLoading.value = true;
  const requestToken = detailRequestToken.value + 1;
  detailRequestToken.value = requestToken;
  try {
    const detail = await defBaseMigrationService.GetBaseMigration({ id });
    if (detailRequestToken.value === requestToken && selectedMigrationId.value === id) {
      selectedMigration.value = detail;
      activeFilePath.value = detail.files[0]?.path ?? "";
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

.detail-scroll {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
  padding: 24px;
}

.migration-markdown {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  width: 100%;
  max-width: 100%;
  overflow-wrap: anywhere;
}

.migration-file-tabs {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}

.migration-file-tabs :deep(.el-tabs__header) {
  flex-shrink: 0;
}

.migration-file-tabs :deep(.el-tabs__content) {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.migration-file-tabs :deep(.el-tab-pane) {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.migration-file-path {
  flex-shrink: 0;
  margin-bottom: 16px;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
  overflow-wrap: anywhere;
}

.sql-panel {
  position: relative;
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
}

.sql-code {
  flex: 1;
  min-width: 0;
  min-height: 0;
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
    flex: 0 0 480px;
    min-height: 480px;
    border-left: 0;
  }

  .migration-pagination {
    justify-content: center;
    padding-right: 12px;
    padding-left: 12px;
    overflow-x: auto;
  }

  .detail-scroll {
    padding-right: 16px;
    padding-left: 16px;
  }
}
</style>
