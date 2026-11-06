<template>
  <main class="bi-workbench">
    <el-skeleton v-if="loading" :rows="6" animated /><el-empty
      v-else-if="!dashboards.length"
      :description="t('bi.workbench.empty')"
      ><el-button @click="router.push('/bi/manage')">{{ t("bi.dashboard.manage") }}</el-button></el-empty
    >
    <div v-else class="bi-workbench__grid" :aria-busy="saving">
      <article
        v-for="(item, index) in dashboards"
        :key="item.id"
        class="bi-workbench__entry"
        :class="{ 'is-dragging': dragged === index }"
        :draggable="!saving"
        tabindex="0"
        :aria-label="`${item.name}：${t('bi.workbench.drag')}`"
        aria-keyshortcuts="Alt+ArrowLeft Alt+ArrowRight"
        @dragstart="startDrag($event, index)"
        @dragend="dragged = -1"
        @dragover.prevent
        @drop.prevent="move(dragged, index)"
        @keydown.alt.left.prevent="move(index, index - 1)"
        @keydown.alt.right.prevent="move(index, index + 1)"
      >
        <BiWorkbenchPreview :key="`${item.id}:${item.published_version}`" :dashboard="item" />
        <div class="bi-workbench__info">
          <div class="bi-workbench__title">
            <h3 :title="item.name">{{ item.name }}</h3>
            <el-tag size="small" effect="light">{{ t("bi.dashboard.status_published") }}</el-tag>
          </div>
          <p class="bi-workbench__meta">
            {{ t("bi.card.total", { count: parseDashboardLayout(item.schema_json).cards.length }) }} · v{{
              item.published_version
            }}
          </p>
          <p v-if="item.description" class="bi-workbench__description">{{ item.description }}</p>
          <footer>
            <el-button type="primary" @click="router.push(`/bi/dashboard/${item.id}`)"
              >{{ t("bi.workbench.open_dashboard") }}<el-icon class="el-icon--right"><ArrowRight /></el-icon
            ></el-button>
          </footer>
        </div>
      </article>
    </div>
  </main>
</template>
<script setup lang="ts">
import { onActivated, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { t } from "@liujitcn/kratos-admin-core";
import type { BiDashboard } from "../../rpc/bi/admin/v1/bi_dashboard";
import { orderWorkbenchDashboards } from "../../card-schema";
import { defBiCardService } from "../../api/bi/admin/v1/bi_card";
import { parseDashboardLayout } from "../../bi-dashboard-schema";
import BiWorkbenchPreview from "../../components/BiWorkbenchPreview.vue";
defineOptions({ name: "BiWorkbench", inheritAttrs: false });
const router = useRouter(),
  dashboards = ref<BiDashboard[]>([]),
  loading = ref(false),
  saving = ref(false),
  dragged = ref(-1);
let loaded = false;
/** 读取收藏的仪表盘，重新进入时同步管理页的收藏变化。 */
async function load() {
  loading.value = true;
  try {
    const config = await defBiCardService.GetBiWorkbench({});
    dashboards.value = orderWorkbenchDashboards(config.dashboard_ids, config.dashboards);
    loaded = true;
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    loading.value = false;
  }
}
/** 卡片默认允许拖动，按钮保留原有点击操作。 */
function startDrag(event: DragEvent, index: number) {
  if (saving.value || (event.target as HTMLElement).closest("button")) {
    event.preventDefault();
    return;
  }
  dragged.value = index;
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", String(dashboards.value[index].id));
  }
}
/** 拖动或键盘排序后自动保存，失败恢复原顺序并保留卡片收藏。 */
async function move(from: number, to: number) {
  if (saving.value || from === to || from < 0 || to < 0 || from >= dashboards.value.length || to >= dashboards.value.length)
    return;
  const previous = dashboards.value;
  const next = [...previous];
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item);
  dashboards.value = next;
  dragged.value = -1;
  saving.value = true;
  try {
    const latest = await defBiCardService.GetBiWorkbench({});
    const saved = await defBiCardService.SaveBiWorkbench({
      dashboard_ids: dashboards.value.map(item => item.id),
      favorite_ids: latest.favorite_ids
    });
    dashboards.value = orderWorkbenchDashboards(saved.dashboard_ids, saved.dashboards);
    ElMessage.success(t("bi.workbench.saved"));
  } catch (cause) {
    dashboards.value = previous;
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    saving.value = false;
  }
}
onMounted(() => void load());
onActivated(() => {
  if (loaded && !saving.value) void load();
});
</script>
<style scoped>
.bi-workbench {
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.bi-workbench__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}
.bi-workbench__entry {
  min-width: 0;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--admin-page-radius);
  overflow: hidden;
  background: var(--admin-page-card-bg);
  cursor: grab;
}
.bi-workbench__entry:active {
  cursor: grabbing;
}
.bi-workbench__entry:focus-visible {
  outline: 2px solid var(--el-color-primary);
  outline-offset: 2px;
}
.bi-workbench__entry.is-dragging {
  border-color: var(--el-color-primary-light-5);
  opacity: 0.5;
}
.bi-workbench__info {
  padding: 16px;
}
.bi-workbench__title {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}
.bi-workbench__title .el-tag {
  flex: none;
}
.bi-workbench h3 {
  color: var(--el-text-color-primary);
  font-size: 15px;
  font-weight: 500;
  line-height: 22px;
  margin: 0;
  overflow-wrap: anywhere;
}
.bi-workbench__meta,
.bi-workbench__description {
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}
.bi-workbench__meta {
  margin: 6px 0 10px;
}
.bi-workbench__description {
  margin: 0 0 10px;
}
.bi-workbench footer {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
  padding-top: 12px;
  margin-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
@container (max-width: 1400px) {
  .bi-workbench__grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
@container (max-width: 1050px) {
  .bi-workbench__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@container (max-width: 650px) {
  .bi-workbench__grid {
    grid-template-columns: 1fr;
  }
}
</style>
