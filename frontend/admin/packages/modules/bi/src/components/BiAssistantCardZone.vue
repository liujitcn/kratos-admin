<template>
  <div v-if="cards.length" class="bi-assistant-workbench__cards">
    <el-container
      v-for="(card, index) in cards"
      :key="card.id"
      class="bi-assistant-workbench__card"
      draggable="true"
      @dragstart="emit('drag-start', { zone, index })"
      @dragend="emit('drag-end')"
      @dragover.prevent
      @drop.stop.prevent="emit('drop', zone, index)"
    >
      <BiDataCard :card="card" :refresh-key="refreshKeys[card.id] || 0" :show-category="false">
        <template #actions>
          <div class="bi-assistant-workbench__card-actions">
            <el-tooltip :content="t('bi.card.refresh')" placement="top">
              <el-button text circle :icon="Refresh" @click.stop="emit('refresh', card)" />
            </el-tooltip>
            <el-tooltip content="引用到对话" placement="top">
              <el-button text circle :icon="CopyDocument" @click.stop="emit('quote', card)" />
            </el-tooltip>
            <el-tooltip :content="t('bi.workbench.remove', { name: card.name })" placement="top">
              <el-button text circle :icon="Delete" @click.stop="emit('remove', card)" />
            </el-tooltip>
            <el-tooltip content="导出数据" placement="top">
              <el-button text circle :icon="Download" @click.stop="emit('export', card)" />
            </el-tooltip>
            <el-tooltip content="放大查看" placement="top">
              <el-button text circle :icon="ZoomIn" @click.stop="emit('preview', card)" />
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
    </el-container>
  </div>
  <div v-else class="bi-assistant-workbench__empty">
    <el-icon><DataAnalysis /></el-icon>
    <span>{{ t("bi.card.empty") }}</span>
  </div>
</template>

<script setup lang="ts">
import { CopyDocument, DataAnalysis, Delete, Download, Refresh, ZoomIn } from "@element-plus/icons-vue";
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
  export: [card: BiCard];
  preview: [card: BiCard];
}>();
</script>
