<template>
  <div class="generate-confirm">
    <p class="generate-confirm__message">{{ message }}</p>
    <template v-if="count">
      <el-alert
        :title="t('system.code.gen.table.dialog.missing_count', { count })"
        :description="t('system.code.gen.table.dialog.missing_hint')"
        type="warning"
        show-icon
        :closable="false"
      />
      <div class="generate-confirm__groups" tabindex="0" :aria-label="t('system.code.gen.table.dialog.missing_count', { count })">
        <details v-for="group in groups" :key="group.name" :open="groups.length === 1">
          <summary>
            <span class="generate-confirm__name">{{ group.name }}</span>
            <span class="generate-confirm__count">{{ group.items.length }}</span>
          </summary>
          <ul>
            <li v-for="(item, index) in group.items" :key="index">{{ item }}</li>
          </ul>
        </details>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { t } from "@liujitcn/kratos-admin-core";

/** 生成确认内容，按业务表保留缺失翻译明细。 */
const props = defineProps<{
  message: string;
  groups: Array<{ name: string; items: string[] }>;
}>();

const count = computed(() => props.groups.reduce((total, group) => total + group.items.length, 0));
</script>

<style scoped lang="scss">
:global(.code-gen-generate-confirm) {
  width: 600px;
  max-width: calc(100vw - 32px);
}

:global(.code-gen-generate-confirm .el-message-box__message) {
  width: 100%;
  min-width: 0;
}

.generate-confirm {
  display: grid;
  gap: 16px;
  min-width: 0;
  &__message {
    margin: 0;
    color: var(--el-text-color-primary);
    font-size: 15px;
    line-height: 1.6;
    overflow-wrap: anywhere;
  }
  &__groups {
    max-height: min(320px, 35vh);
    overflow: auto;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    scrollbar-gutter: stable;
  }
  details + details {
    border-top: 1px solid var(--el-border-color-lighter);
  }
  summary {
    padding: 12px 16px;
    color: var(--el-text-color-primary);
    cursor: pointer;
    background: var(--el-fill-color-light);
    overflow-wrap: anywhere;
    &:hover {
      background: var(--el-fill-color);
    }
    &:focus-visible {
      outline: 2px solid var(--el-color-primary);
      outline-offset: -2px;
    }
  }
  &__name {
    font-weight: 600;
  }
  &__count {
    display: inline-block;
    padding: 0 8px;
    margin-left: 8px;
    font-size: 12px;
    line-height: 20px;
    color: var(--el-color-warning);
    background: var(--el-color-warning-light-9);
    border-radius: 10px;
  }
  ul {
    padding: 4px 16px;
    margin: 0;
    list-style: none;
  }
  li {
    padding: 8px 0;
    font-size: 13px;
    line-height: 1.6;
    color: var(--el-text-color-regular);
    overflow-wrap: anywhere;
    & + li {
      border-top: 1px solid var(--el-border-color-extra-light);
    }
  }
}
</style>
