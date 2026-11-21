<template>
  <div v-if="cards.length" class="bi-assistant-workbench__cards">
    <div
      v-for="(card, index) in cards"
      :key="card.id"
      class="bi-assistant-workbench__card"
      draggable="true"
      @dragstart="emit('drag-start', { zone, index })"
      @dragend="emit('drag-end')"
      @dragover.prevent
      @drop.stop.prevent="emit('drop', zone, index)"
    >
      <BiDataCard
        :card="card"
        :refresh-key="refreshKeys[card.id] || 0"
        :show-category="false"
        previewable
        footer-actions
        compact
        @preview="emit('preview', card)"
      >
        <template #actions>
          <div class="bi-data-card__slot-actions bi-assistant-workbench__card-actions">
            <el-tooltip :content="t('bi.card.refresh')" placement="top">
              <el-button text circle :icon="Refresh" @click.stop="emit('refresh', card)" />
            </el-tooltip>
            <el-tooltip :content="t('bi.card.quote')" placement="top">
              <el-button text circle :icon="CopyDocument" @click.stop="emit('quote', card)" />
            </el-tooltip>
            <el-tooltip :content="t('bi.workbench.remove', { name: card.name })" placement="top">
              <el-button text circle type="danger" :icon="Delete" @click.stop="emit('remove', card)" />
            </el-tooltip>
          </div>
        </template>
        <template #footer>
          <span class="bi-assistant-workbench__card-foot">
            {{ card.source_code || t("bi.card.all_sources") }} ·
            {{ card.query?.time_range?.start?.slice(0, 10) || t("bi.card.period.default") }}
          </span>
        </template>
      </BiDataCard>
    </div>
  </div>
  <div v-else class="bi-assistant-workbench__empty">
    <el-icon><DataAnalysis /></el-icon>
    <span>{{ t("bi.card.empty") }}</span>
  </div>
</template>

<script setup lang="ts">
import { CopyDocument, DataAnalysis, Delete, Refresh } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import type { BiCard } from "../rpc/bi/admin/v1/bi_card";
import BiDataCard from "./BiDataCard.vue";

type Zone = "left" | "right";

defineProps<{
  zone: Zone;
  cards: BiCard[];
  refreshKeys: Record<number, number>;
}>();

const emit = defineEmits<{
  "drag-start": [value: { zone: Zone; index: number }];
  "drag-end": [];
  drop: [zone: Zone, index?: number];
  refresh: [card: BiCard];
  quote: [card: BiCard];
  remove: [card: BiCard];
  preview: [card: BiCard];
}>();
</script>

<style scoped>
.bi-assistant-workbench__card-actions {
  display: inline-flex;
  flex: 0 0 auto;
  flex-wrap: nowrap;
  min-width: max-content;
  align-items: center;
  gap: 0;
  margin-left: auto;
  white-space: nowrap;
}
.bi-assistant-workbench__card-actions :deep(.el-tooltip__trigger) {
  display: inline-flex;
  flex: 0 0 auto;
}
.bi-assistant-workbench__card-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
</style>
