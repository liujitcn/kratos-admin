<template>
  <main class="bi-card-library">
    <el-card class="admin-page-card" shadow="never">
      <el-tabs v-model="category" @tab-change="search">
        <el-tab-pane v-for="item in categories" :key="item" :name="item">
          <template #label>
            <span>{{ t(`bi.card.category.${item || "all"}`) }}</span>
            <small class="bi-card-library__count">{{ counts[item || "all"] }}</small>
          </template>
        </el-tab-pane>
      </el-tabs>
      <div class="bi-card-library__filters">
        <el-input v-model="name" :placeholder="t('bi.card.search')" clearable @keyup.enter="search" @clear="search" />
        <el-select v-model="sourceCode" :placeholder="t('bi.card.all_sources')" clearable @change="search">
          <el-option v-for="item in sources" :key="item.code" :label="item.name" :value="item.code" />
        </el-select>
        <el-button type="primary" :icon="icons.Search" :loading="loading" @click="search">{{
          t("common.action.search")
        }}</el-button>
        <span>{{ t("bi.card.total", { count: total }) }}</span>
      </div>
    </el-card>
    <el-skeleton v-if="loading" :rows="6" animated /><el-empty v-else-if="!cards.length" :description="t('bi.card.empty')" />
    <div v-else class="bi-card-library__grid">
      <BiDataCard
        v-for="card in cards"
        :key="card.id"
        :card="card"
        :source-name="sources.find(item => item.code === card.source_code)?.name || card.source_code"
        ><template #actions>
          <el-tooltip :content="t(workbench.favorite_ids.includes(card.id) ? 'bi.card.unfavorite' : 'bi.card.favorite', { name: card.name })">
            <el-button
              text
              circle
              :type="workbench.favorite_ids.includes(card.id) ? 'warning' : 'default'"
              :icon="workbench.favorite_ids.includes(card.id) ? icons.StarFilled : icons.Star"
              :aria-label="t(workbench.favorite_ids.includes(card.id) ? 'bi.card.unfavorite' : 'bi.card.favorite', { name: card.name })"
              :aria-pressed="workbench.favorite_ids.includes(card.id)"
              @click="toggleFavorite(card)"
            />
          </el-tooltip>
        </template
        ><template #footer
          ><div>
            <el-button link type="primary" @click="previewCard = card">{{ t("bi.card.preview") }}</el-button
            ><el-button v-if="card.can_edit" link @click="openEditor(card)">{{ t("common.action.edit") }}</el-button
            ><el-button v-if="card.can_edit" link type="danger" @click="remove(card)">{{ t("common.action.delete") }}</el-button>
          </div>
          <el-button size="small" type="primary" @click="pickDashboard(card)">{{ t("bi.card.add_to_dashboard") }}</el-button
          ><el-button
            v-if="card.query_record_id"
            link
            :disabled="!card.can_edit"
            @click="router.push({ path: '/bi/query', query: { record_id: String(card.query_record_id) } })"
            >{{ t("bi.card.source_record", { id: card.query_record_id }) }}</el-button
          ></template
        ></BiDataCard
      >
    </div>
    <el-pagination
      v-if="total > 100"
      v-model:current-page="page"
      :page-size="100"
      :total="total"
      layout="prev,pager,next"
      @current-change="loadCards"
    /><BiCardFormDialog v-model="editorOpen" :sources="sources" :card="editingCard" @saved="reload" /><el-dialog
      :model-value="Boolean(previewCard)"
      :title="t('bi.card.preview')"
      width="min(760px,94vw)"
      destroy-on-close
      @close="previewCard = undefined"
      ><BiDataCard
        v-if="previewCard"
        :card="previewCard"
        :source-name="sources.find(item => item.code === previewCard?.source_code)?.name || previewCard.source_code" /></el-dialog
    ><FormDialog
      v-model="dashboardOpen"
      :model="target"
      :title="t('bi.card.add_to_dashboard')"
      :fields="dashboardFields"
      :rules="{ id: [{ required: true, message: t('bi.card.select_dashboard'), trigger: 'change' }] }"
      :confirm-loading="saving"
      @confirm="addToDashboard"
      @closed="target.id = undefined"
    />
  </main>
</template>
<script setup lang="ts">
import { computed, onMounted, onActivated, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import type { BiCard, BiCardCounts, BiWorkbench } from "../rpc/bi/admin/v1/bi_card";
import type { BiDashboard } from "../rpc/bi/admin/v1/bi_dashboard";
import type { BiSourceDescriptor } from "../rpc/bi/admin/v1/bi_query";
import { defBiCardService } from "../api/bi/admin/v1/bi_card";
import { defBiQueryService } from "../api/bi/admin/v1/bi_query";
import { defBiDashboardService } from "../api/bi/admin/v1/bi_dashboard";
import { parseDashboardLayout, reflowDashboardCards } from "../bi-dashboard-schema";
import BiDataCard from "./BiDataCard.vue";
import BiCardFormDialog from "./BiCardFormDialog.vue";
const icons = { Star, StarFilled, Search };
const router = useRouter(),
  cards = ref<BiCard[]>([]),
  sources = ref<BiSourceDescriptor[]>([]),
  name = ref(""),
  sourceCode = ref(""),
  category = ref(""),
  page = ref(1),
  total = ref(0),
  loading = ref(false),
  saving = ref(false),
  editorOpen = ref(false),
  dashboardOpen = ref(false),
  editingCard = ref<BiCard>(),
  previewCard = ref<BiCard>(),
  addingCard = ref<BiCard>(),
  dashboards = ref<BiDashboard[]>([]);
const categories = ["", "mine", "favorite", "shared", "system"];
const counts = ref<BiCardCounts>({ all: 0, system: 0, mine: 0, favorite: 0, shared: 0 });
const workbench = ref<BiWorkbench>({ dashboard_ids: [], favorite_ids: [], dashboards: [] });
const target = reactive({ id: undefined as number | undefined });
let request = 0,
  loaded = false;
const dashboardFields = computed<ProFormField[]>(() => [
  {
    prop: "id",
    label: t("bi.dashboard.title"),
    component: "select",
    options: dashboards.value.map(item => ({ label: item.name, value: item.id })),
    props: { filterable: true }
  }
]);
/** 按同一过滤条件查询当前可见卡片，丢弃迟到请求。 */
async function loadCards() {
  const current = ++request;
  loading.value = true;
  try {
    const data = await defBiCardService.PageBiCard({
      name: name.value,
      source_code: sourceCode.value,
      category: category.value,
      search_source_codes: sources.value.filter(item => item.name.includes(name.value)).map(item => item.code),
      page_num: page.value,
      page_size: 100
    });
    if (current !== request) return;
    cards.value = data.cards;
    total.value = data.total;
    counts.value = data.counts!;
    loaded = true;
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (current === request) loading.value = false;
  }
}
/** 搜索或切换分类时从第一页开始。 */
function search() {
  page.value = 1;
  void loadCards();
}
/** 预加载查询目录和收藏后刷新列表。 */
async function reload() {
  try {
    const [dictionary, config] = await Promise.all([defBiQueryService.ListBiSources({}), defBiCardService.GetBiWorkbench({})]);
    sources.value = dictionary.sources;
    workbench.value = config;
    await loadCards();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  }
}
/** 打开已有卡片的编辑表单。 */
function openEditor(card: BiCard) {
  editingCard.value = card;
  editorOpen.value = true;
}
/** 保存收藏后只更新星标选中状态，保持当前列表和图表实例。 */
async function toggleFavorite(card: BiCard) {
  try {
    const latest = await defBiCardService.GetBiWorkbench({});
    const ids = latest.favorite_ids.includes(card.id)
      ? latest.favorite_ids.filter(id => id !== card.id)
      : [...latest.favorite_ids, card.id];
    const saved = await defBiCardService.SaveBiWorkbench({ dashboard_ids: latest.dashboard_ids, favorite_ids: ids });
    workbench.value.favorite_ids = saved.favorite_ids;
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.card.save_failed"));
  }
}
/** 删除当前用户自己的卡片，确认时展示卡片名称。 */
async function remove(card: BiCard) {
  try {
    await ElMessageBox.confirm(t("bi.card.delete_confirm", { name: card.name }), t("common.action.delete"), { type: "warning" });
  } catch {
    return;
  }
  try {
    await defBiCardService.DeleteBiCard({ id: card.id });
    await reload();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.card.delete_failed"));
  }
}
/** 选择当前用户可编辑的仪表盘草稿。 */
async function pickDashboard(card: BiCard) {
  try {
    const result: BiDashboard[] = [];
    let current = 1;
    while (true) {
      const data = await defBiDashboardService.PageBiDashboard({ name: "", page_num: current, page_size: 100 });
      result.push(...data.dashboards.filter(item => item.can_edit));
      if (current * 100 >= data.total || !data.dashboards.length) break;
      current++;
    }
    if (!result.length) {
      ElMessage.info(t("bi.card.create_dashboard_first"));
      return;
    }
    dashboards.value = result;
    addingCard.value = card;
    target.id = undefined;
    dashboardOpen.value = true;
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  }
}
/** 只追加卡片引用，发布之前不改变正式仪表盘。 */
async function addToDashboard() {
  if (!target.id || !addingCard.value || saving.value) return;
  saving.value = true;
  try {
    const item = await defBiDashboardService.GetBiDashboard({ id: target.id });
    const layout = parseDashboardLayout(item.schema_json);
    if (layout.cards.some(item => item.card_id === addingCard.value!.id)) {
      ElMessage.info(t("bi.card.already_in_dashboard"));
      return;
    }
    layout.cards = reflowDashboardCards([
      ...layout.cards,
      { id: `card_${addingCard.value.id}`, card_id: addingCard.value.id, w: 6, h: 4 }
    ]);
    await defBiDashboardService.SaveBiDashboard({ dashboard: { ...item, schema_json: JSON.stringify(layout) } });
    dashboardOpen.value = false;
    ElMessage.success(t("bi.card.add_dashboard_success"));
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.card.save_failed"));
  } finally {
    saving.value = false;
  }
}
onMounted(() => void reload());
onActivated(() => {
  if (loaded) void reload();
});
</script>
<style scoped>
.bi-card-library {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.bi-card-library__count {
  margin-left: 8px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.bi-card-library__filters {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}
.bi-card-library__filters > .el-input {
  width: 260px;
}
.bi-card-library__filters > .el-select {
  width: 200px;
}
.bi-card-library__filters > span {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.bi-card-library__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}
.bi-card-library :deep(.el-pagination) {
  justify-content: flex-end;
}
@media (max-width: 1100px) {
  .bi-card-library__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 650px) {
  .bi-card-library__grid {
    grid-template-columns: 1fr;
  }
  .bi-card-library__filters > .el-input,
  .bi-card-library__filters > .el-select {
    width: 100%;
  }
}
</style>
