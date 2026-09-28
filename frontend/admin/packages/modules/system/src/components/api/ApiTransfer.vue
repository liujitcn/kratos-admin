<template>
  <el-transfer
    v-model="selectedValues"
    class="api-transfer"
    :data="options"
    :props="{ key: 'value', label: 'label', disabled: 'disabled' }"
    :titles="transferTitles"
    :filterable="true"
    :style="{ '--api-transfer-panel-height': `${panelHeight}px` }"
  >
    <template #default="{ option }">
      <slot :option="option">{{ option.label }}</slot>
    </template>
  </el-transfer>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { ApiTransferOption } from "./apiTransfer";

/** API 穿梭框属性。 */
interface ApiTransferProps {
  /** API 选项列表。 */
  options: ApiTransferOption[];
  /** 左右面板标题。 */
  titles: string[];
  /** 穿梭框面板高度。 */
  panelHeight?: number;
}

const props = withDefaults(defineProps<ApiTransferProps>(), {
  panelHeight: 360
});

const selectedValues = defineModel<string[]>({ required: true });
const transferTitles = computed<[string, string]>(() => [props.titles[0] ?? "", props.titles[1] ?? ""]);

defineSlots<{
  default(props: { option: ApiTransferOption }): unknown;
}>();
</script>

<style scoped lang="scss">
.api-transfer {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr);
  align-items: center;
  gap: 16px;
  width: 100%;
}

.api-transfer :deep(.el-transfer-panel) {
  width: 100%;
  min-width: 0;
}

.api-transfer :deep(.el-transfer__buttons) {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 0;
}

.api-transfer :deep(.el-transfer__button) {
  margin: 0;
}

.api-transfer :deep(.el-transfer-panel__body) {
  height: var(--api-transfer-panel-height);
}

.api-transfer :deep(.el-transfer-panel__list.is-filterable) {
  height: calc(var(--api-transfer-panel-height) - 60px);
}

.api-transfer :deep(.el-transfer-panel__item.el-checkbox) {
  display: flex !important;
  align-items: flex-start;
  box-sizing: border-box;
  height: auto;
  min-height: 36px;
  margin-right: 0;
  padding: 8px 12px 8px 15px;
  line-height: 20px;
}

.api-transfer :deep(.el-transfer-panel__item .el-checkbox__input) {
  position: relative;
  top: 3px;
  flex: none;
}

.api-transfer :deep(.el-transfer-panel__item .el-checkbox__label) {
  display: block;
  flex: 1;
  width: auto;
  min-width: 0;
  height: auto;
  padding-left: 8px;
  overflow: visible;
  line-height: 20px;
  text-overflow: clip;
  white-space: normal;
  overflow-wrap: anywhere;
}

@media (max-width: 900px) {
  .api-transfer {
    grid-template-columns: minmax(0, 1fr);
  }

  .api-transfer :deep(.el-transfer__buttons) {
    flex-direction: row;
    justify-content: center;
  }
}
</style>
