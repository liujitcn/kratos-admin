<template>
  <div class="bi-designer">
    <el-aside class="bi-designer__library" style="--el-aside-width: auto">
      <el-header class="bi-designer__library-header" style="--el-header-height: auto; --el-header-padding: 0">
        <el-text tag="h3">{{ t("bi.card.library") }}</el-text>
        <el-text tag="p">{{ t("bi.dashboard.choose_cards") }}</el-text>
      </el-header>
      <div class="bi-designer__library-list">
        <div v-for="card in cards" :key="card.id" class="bi-designer__library-item">
          <BiDataCard :card="card" :show-category="false" compact />
          <div class="bi-designer__library-meta">
            <span>{{
              ["metric", "metric_grid"].includes(card.chart_type)
                ? t(`bi.card.chart.${card.chart_type}`)
                : messages(`types.${card.chart_type}`)
            }}</span>
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
      <el-footer class="bi-designer__library-footer" style="--el-footer-height: auto; --el-footer-padding: 0">
        <el-button :icon="plusIcon" plain type="primary" @click="emit('create-card')">{{ t("bi.card.create") }}</el-button>
      </el-footer>
    </el-aside>
    <el-main class="bi-designer__canvas">
      <div v-if="$slots.toolbar" class="bi-designer__toolbar">
        <slot name="toolbar" />
      </div>
      <el-empty v-if="!modelValue.cards.length" :description="t('bi.dashboard.choose_cards')" />
      <div v-else class="bi-designer__grid">
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
          <BiDataCard
            v-if="cardByID.get(item.card_id)"
            :card="cardByID.get(item.card_id)!"
            previewable
            footer-actions
            @preview="previewCard = cardByID.get(item.card_id)"
          >
            <template #actions>
              <div class="bi-data-card__slot-actions bi-designer__card-actions">
                <el-popover trigger="click" :width="220">
                  <template #reference>
                    <el-tooltip :content="t('bi.dashboard.layout_settings')" placement="top">
                      <el-button text circle :icon="layoutIcon" @click.stop="selected = item.id" />
                    </el-tooltip>
                  </template>
                  <el-form size="small">
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
                <el-tooltip :content="t('bi.card.edit')" placement="top">
                  <el-button
                    v-if="cardByID.get(item.card_id)?.can_edit"
                    text
                    circle
                    :icon="editIcon"
                    @click.stop="emit('edit-card', cardByID.get(item.card_id)!)"
                  />
                </el-tooltip>
                <el-tooltip :content="t('bi.workbench.remove', { name: cardByID.get(item.card_id)?.name || '' })" placement="top">
                  <el-button text circle type="danger" :icon="removeIcon" @click.stop="remove(item.id)" />
                </el-tooltip>
              </div>
            </template>
          </BiDataCard>
          <el-alert v-else type="warning" :closable="false" :title="t('bi.dashboard.card_missing')" />
        </div>
      </div>
    </el-main>
    <el-dialog
      :model-value="Boolean(previewCard)"
      :title="t('bi.card.preview')"
      width="min(760px,94vw)"
      destroy-on-close
      @close="previewCard = undefined"
    >
      <BiDataCard v-if="previewCard" :card="previewCard" />
    </el-dialog>
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
const previewCard = ref<BiCard>();
const plusIcon = Plus;
const checkIcon = Check;
const layoutIcon = Setting;
const editIcon = EditPen;
const removeIcon = Delete;
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
/** 通过拖拽调整卡片顺序。 */
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
  grid-template-rows: minmax(0, 1fr);
  gap: 8px;
  height: 100%;
  min-height: 0;
}
.bi-designer__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  margin-bottom: 8px;
}
.bi-designer__toolbar :deep(.el-button + .el-button) {
  margin-left: 0;
}
.bi-designer__library {
  position: sticky;
  top: 0;
  display: flex;
  box-sizing: border-box;
  min-width: 0;
  height: 100%;
  min-height: 0;
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
.bi-designer__library-item :deep(.bi-data-card) {
  border-color: var(--admin-page-divider);
}
.bi-designer__library-item :deep(.bi-data-card__native-actions) {
  display: none;
}
.bi-designer__library-item :deep(.bi-data-card__header) {
  padding: 10px 10px 0;
}
.bi-designer__library-item :deep(.bi-data-card__chart) {
  height: 104px;
  padding: 8px 10px;
}
.bi-designer__library-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  padding-top: 8px;
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
  padding: 8px;
  border: 1px dashed var(--admin-page-card-border-muted);
  background: var(--admin-page-card-bg-soft);
  border-radius: var(--admin-page-radius);
  min-width: 0;
  min-height: 0;
}
.bi-designer__grid {
  display: grid;
  grid-template-columns: repeat(12, minmax(0, 1fr));
  grid-auto-rows: 80px;
  gap: 12px;
}
.bi-designer__canvas > .el-empty {
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
.bi-designer__card-actions {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: nowrap;
  width: max-content;
  min-width: max-content;
  gap: 0;
  align-items: center;
  justify-content: flex-start;
  margin-left: auto;
  white-space: nowrap;
}
.bi-designer__card-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
.bi-designer__card-actions :deep(.el-tooltip__trigger) {
  flex: 0 0 auto;
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
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: auto auto;
    height: auto;
  }
  .bi-designer__library {
    position: static;
    height: 320px;
    min-height: 0;
  }
  .bi-designer__grid {
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
