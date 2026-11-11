<template>
  <div class="bi-designer">
    <aside class="bi-designer__library">
      <header class="bi-designer__library-header">
        <h3>{{ t("bi.card.library") }}</h3>
        <p>{{ t("bi.dashboard.choose_cards") }}</p>
      </header>
      <div class="bi-designer__library-list">
        <div v-for="card in cards" :key="card.id" class="bi-designer__library-item">
          <h4>{{ card.name }}</h4>
          <div class="bi-designer__library-meta">
            <span>{{ card.chart_type === "metric" ? t("bi.card.chart.metric") : messages(`types.${card.chart_type}`) }}</span>
            <span v-if="addedCardIDs.has(card.id)" class="bi-designer__added">
              <el-icon><component :is="checkIcon" /></el-icon>{{ t("bi.card.added") }}
            </span>
            <el-button
              v-else
              link
              type="primary"
              size="small"
              :icon="plusIcon"
              :aria-label="`${t('bi.card.add')}：${card.name}`"
              :disabled="modelValue.cards.length >= 20"
              @click="add(card)"
              >{{ t("bi.card.add_short") }}</el-button
            >
          </div>
        </div>
        <el-empty v-if="!cards.length" :image-size="48" :description="t('bi.card.empty')" />
      </div>
      <footer class="bi-designer__library-footer">
        <el-button :icon="plusIcon" plain type="primary" @click="emit('create-card')">{{ t("bi.card.create") }}</el-button>
      </footer>
    </aside>
    <main class="bi-designer__canvas">
      <div
        v-for="(item, index) in modelValue.cards"
        :key="item.id"
        class="bi-designer__item"
        :class="{ 'is-selected': selected === item.id }"
        :style="{ gridColumn: `${(item.x || 0) + 1} / span ${item.w}`, gridRow: `${(item.y || 0) + 1} / span ${item.h}` }"
        draggable="true"
        @dragstart="dragged = index"
        @dragover.prevent
        @drop.prevent="move(dragged, index)"
        @click="selected = item.id"
      >
        <BiDataCard v-if="cardByID.get(item.card_id)" :card="cardByID.get(item.card_id)!">
          <template #footer
            ><el-button link :disabled="index === 0" @click.stop="move(index, index - 1)">{{
              t("bi.workbench.move_before", { name: "" })
            }}</el-button
            ><el-button link :disabled="index === modelValue.cards.length - 1" @click.stop="move(index, index + 1)">{{
              t("bi.workbench.move_after", { name: "" })
            }}</el-button>
            <el-popover trigger="click" :width="220">
              <template #reference>
                <el-button link type="primary" @click.stop="selected = item.id">{{
                  t("bi.dashboard.layout_settings")
                }}</el-button>
              </template>
              <el-form label-position="top" size="small">
                <el-form-item :label="t('bi.dashboard.card_width')">
                  <el-select :model-value="item.w" @change="resize(item.id, { w: $event })">
                    <el-option :value="3" :label="t('bi.dashboard.width_quarter')" />
                    <el-option :value="6" :label="t('bi.dashboard.width_half')" />
                    <el-option :value="12" :label="t('bi.dashboard.width_full')" />
                  </el-select>
                </el-form-item>
                <el-form-item :label="t('bi.dashboard.card_height')">
                  <el-input-number
                    :model-value="item.h"
                    :min="3"
                    :max="12"
                    :precision="0"
                    @update:model-value="resize(item.id, { h: $event || 4 })"
                  />
                </el-form-item>
              </el-form>
            </el-popover>
            <el-button
              v-if="cardByID.get(item.card_id)?.can_edit"
              link
              type="primary"
              @click.stop="emit('edit-card', cardByID.get(item.card_id)!)"
              >{{ t("bi.card.edit") }}</el-button
            ><el-button link type="danger" @click.stop="remove(item.id)">{{
              t("bi.workbench.remove", { name: "" })
            }}</el-button></template
          >
        </BiDataCard>
        <el-alert v-else type="warning" :closable="false" :title="t('bi.dashboard.card_missing')" />
      </div>
      <el-empty v-if="!modelValue.cards.length" :description="t('bi.dashboard.choose_cards')" />
    </main>
  </div>
</template>
<script setup lang="ts">
import { computed, ref } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import type { BiCard } from "../rpc/bi/admin/v1/bi_card";
import type { DashboardLayout, DashboardCardLayout } from "../bi-dashboard-schema";
import { reflowDashboardCards } from "../bi-dashboard-schema";
import BiDataCard from "./BiDataCard.vue";
import { useBiI18n } from "../env";
const props = defineProps<{ modelValue: DashboardLayout; cards: BiCard[] }>();
const emit = defineEmits<{ "update:modelValue": [value: DashboardLayout]; "edit-card": [value: BiCard]; "create-card": [] }>();
const selected = ref("");
const dragged = ref(-1);
const plusIcon = Plus;
const checkIcon = Check;
const { t: messages } = useBiI18n();
const cardByID = computed(() => new Map(props.cards.map(card => [card.id, card])));
const addedCardIDs = computed(() => new Set(props.modelValue.cards.map(item => item.card_id)));
/** 加入已保存卡片，复用栅格算法安排位置。 */
function add(card: BiCard) {
  if (addedCardIDs.value.has(card.id) || props.modelValue.cards.length >= 20) return;
  const item: DashboardCardLayout = { id: `card_${card.id}`, card_id: card.id, w: card.chart_type === "metric" ? 3 : 6, h: 4 };
  emit("update:modelValue", { ...props.modelValue, cards: reflowDashboardCards([...props.modelValue.cards, item]) });
  selected.value = item.id;
}
/** 调整卡片顺序，拖动和方向按钮共用同一入口。 */
function move(from: number, to: number) {
  if (from < 0 || to < 0 || from >= props.modelValue.cards.length || to >= props.modelValue.cards.length) return;
  const cards = [...props.modelValue.cards];
  const [item] = cards.splice(from, 1);
  cards.splice(to, 0, item);
  emit("update:modelValue", { ...props.modelValue, cards: reflowDashboardCards(cards) });
  dragged.value = -1;
}
/** 移除当前仪表盘的引用，不删除源卡片。 */
function remove(id: string) {
  emit("update:modelValue", {
    ...props.modelValue,
    cards: reflowDashboardCards(props.modelValue.cards.filter(item => item.id !== id))
  });
  if (selected.value === id) selected.value = "";
}
/** 在卡片内调整尺寸并重新整理画布。 */
function resize(id: string, value: Partial<DashboardCardLayout>) {
  emit("update:modelValue", {
    ...props.modelValue,
    cards: reflowDashboardCards(props.modelValue.cards.map(item => (item.id === id ? { ...item, ...value } : item)))
  });
}
</script>
<style scoped>
.bi-designer {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  align-items: start;
  gap: 16px;
}
.bi-designer__library {
  position: sticky;
  top: 0;
  display: flex;
  box-sizing: border-box;
  min-width: 0;
  height: min(760px, calc(100vh - 220px));
  min-height: 360px;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  background: var(--admin-page-card-bg);
}
.bi-designer__library-header {
  padding: 16px;
  border-bottom: 1px solid var(--admin-page-divider);
}
.bi-designer h3 {
  margin: 0;
  color: var(--admin-page-text-primary);
  font-size: 14px;
  font-weight: 600;
  line-height: 22px;
}
.bi-designer__library p {
  margin: 6px 0 0;
  color: var(--admin-page-text-secondary);
  font-size: 12px;
  line-height: 20px;
}
.bi-designer__library-list {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 0 16px;
}
.bi-designer__library-item {
  padding: 14px 0;
  border-bottom: 1px solid var(--admin-page-divider);
}
.bi-designer__library-item:last-child {
  border-bottom: 0;
}
.bi-designer__library-item h4 {
  margin: 0 0 8px;
  color: var(--admin-page-text-primary);
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
  overflow-wrap: anywhere;
}
.bi-designer__library-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  color: var(--admin-page-text-secondary);
  font-size: 12px;
  line-height: 20px;
}
.bi-designer__added {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--el-color-primary);
}
.bi-designer__library-footer {
  padding: 12px 16px;
  border-top: 1px solid var(--admin-page-divider);
}
.bi-designer__library-footer .el-button {
  width: 100%;
}
.bi-designer__canvas {
  box-sizing: border-box;
  display: grid;
  grid-template-columns: repeat(12, minmax(0, 1fr));
  grid-auto-rows: 80px;
  align-content: start;
  gap: 12px;
  padding: 12px;
  border: 1px dashed var(--admin-page-card-border-muted);
  background: var(--admin-page-card-bg-soft);
  border-radius: var(--admin-page-radius);
  min-width: 0;
  min-height: max(600px, calc(100vh - 220px));
}
.bi-designer__canvas > .el-empty {
  grid-column: 1 / -1;
  min-height: 400px;
}
.bi-designer__item {
  min-width: 0;
  min-height: 0;
  cursor: move;
}
.bi-designer__item :deep(.bi-data-card) {
  height: 100%;
}
.bi-designer__item :deep(.bi-data-card__chart) {
  flex: 1;
  min-height: 70px;
  height: auto;
}
.bi-designer__item :deep(.bi-data-card__footer .el-button) {
  margin-left: 0;
}
.bi-designer__item.is-selected :deep(.bi-data-card) {
  border-color: var(--el-color-primary);
}
@media (max-width: 1100px) {
  .bi-designer {
    grid-template-columns: 220px minmax(0, 1fr);
  }
}
@media (max-width: 700px) {
  .bi-designer {
    grid-template-columns: 1fr;
  }
  .bi-designer__library {
    position: static;
    height: 320px;
    min-height: 0;
  }
  .bi-designer__canvas {
    grid-template-columns: 1fr;
    grid-auto-rows: auto;
  }
  .bi-designer__item {
    grid-column: 1 !important;
    grid-row: auto !important;
    min-height: 360px;
  }
}
</style>
