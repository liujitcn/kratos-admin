<template>
  <main class="bi-designer-page">
    <el-card class="admin-page-card bi-designer-page__toolbar" shadow="never">
      <header>
        <h2>{{ dashboard?.name || t("bi.dashboard.edit") }}</h2>
        <div>
          <el-button :icon="icons.ArrowLeft" @click="router.push('/bi/manage')">{{ t("common.action.back") }}</el-button
          ><el-button :icon="icons.Document" :loading="saving" :disabled="!dashboard" @click="save(false)">{{
            t("bi.dashboard.save")
          }}</el-button
          ><el-button
            :icon="icons.Promotion"
            type="primary"
            :loading="saving"
            :disabled="!layout.cards.length"
            @click="save(true)"
            >{{ t("bi.dashboard.publish") }}</el-button
          >
        </div>
      </header>
    </el-card>
    <el-skeleton v-if="loading" :rows="6" animated /><el-alert
      v-else-if="!dashboard"
      :title="t('bi.dashboard.select_first')"
      type="info"
      show-icon
    /><BiDesigner v-else v-model="layout" :cards="cards" @edit-card="editCard" @create-card="editCard()" /><BiCardFormDialog
      v-model="cardOpen"
      :sources="sources"
      :card="editingCard"
      @saved="loadCards"
    />
  </main>
</template>
<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import BiDesigner from "../../components/BiDesigner.vue";
import BiCardFormDialog from "../../components/BiCardFormDialog.vue";
import { parseDashboardLayout } from "../../bi-dashboard-schema";
import type { DashboardLayout } from "../../bi-dashboard-schema";
import type { BiCard } from "../../rpc/bi/admin/v1/bi_card";
import type { BiDashboard } from "../../rpc/bi/admin/v1/bi_dashboard";
import type { BiSourceDescriptor } from "../../rpc/bi/admin/v1/bi_query";
import { defBiDashboardService } from "../../api/bi/admin/v1/bi_dashboard";
import { defBiCardService } from "../../api/bi/admin/v1/bi_card";
import { defBiQueryService } from "../../api/bi/admin/v1/bi_query";
defineOptions({ name: "BiDesignerPage", inheritAttrs: false });
const icons = { ArrowLeft, Document, Promotion };
const route = useRoute(),
  router = useRouter(),
  id = computed(() => Number(route.query.id || 0));
const dashboard = ref<BiDashboard>(),
  cards = ref<BiCard[]>([]),
  sources = ref<BiSourceDescriptor[]>([]),
  layout = ref<DashboardLayout>({ schemaVersion: 3, name: "", cards: [] });
const loading = ref(false),
  saving = ref(false),
  cardOpen = ref(false),
  editingCard = ref<BiCard>();
let request = 0;
/** 编辑入口按仪表盘编号切换，卡片来自当前用户可见的统一卡片库。 */
async function load() {
  const current = ++request;
  dashboard.value = undefined;
  if (!id.value) return;
  loading.value = true;
  try {
    const [record, dictionary] = await Promise.all([
      defBiDashboardService.GetBiDashboard({ id: id.value }),
      defBiQueryService.ListBiSources({})
    ]);
    if (current !== request) return;
    if (!record.can_edit) throw new Error(t("bi.dashboard.read_only"));
    dashboard.value = record;
    layout.value = { ...parseDashboardLayout(record.schema_json), name: record.name };
    sources.value = dictionary.sources;
    await loadCards();
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (current === request) loading.value = false;
  }
}
/** 分页加载可见卡片，避免设计器只能使用第一页的图表。 */
async function loadCards() {
  const result: BiCard[] = [];
  let page = 1;
  while (true) {
    const data = await defBiCardService.PageBiCard({
      name: "",
      source_code: "",
      category: "",
      search_source_codes: [],
      page_num: page,
      page_size: 100
    });
    result.push(...data.cards);
    if (result.length >= data.total || !data.cards.length) break;
    page++;
  }
  cards.value = result;
}
/** 新建与编辑统一复用卡片表单。 */
function editCard(card?: BiCard) {
  if (!card) {
    void router.push("/bi/query");
    return;
  }
  editingCard.value = card;
  cardOpen.value = true;
}
/** 保存引用布局，发布时由服务端生成独立卡片配置快照。 */
async function save(publish: boolean) {
  if (!dashboard.value || saving.value) return;
  saving.value = true;
  try {
    const saved = await defBiDashboardService.SaveBiDashboard({
      dashboard: { ...dashboard.value, name: layout.value.name, schema_json: JSON.stringify(layout.value) }
    });
    dashboard.value = saved;
    if (publish) {
      dashboard.value = await defBiDashboardService.PublishBiDashboard({ id: saved.id, version: saved.draft_version });
      void router.push(`/bi/dashboard/${saved.id}`);
    }
    ElMessage.success(t(publish ? "bi.dashboard.publish_success" : "bi.dashboard.save_success"));
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    saving.value = false;
  }
}
watch(id, () => void load(), { immediate: true });
</script>
<style scoped>
.bi-designer-page {
  min-width: 0;
}
.bi-designer-page__toolbar {
  margin-bottom: 16px;
}
.bi-designer-page__toolbar header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
}
.bi-designer-page h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}
.bi-designer-page__toolbar header > div {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.bi-designer-page__toolbar .el-button + .el-button {
  margin-left: 0;
}
</style>
