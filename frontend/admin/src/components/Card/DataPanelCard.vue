<template>
  <article class="data-panel-card" :class="cardClass">
    <div class="data-panel-card__header">
      <div>
        <h3 class="data-panel-card__title">{{ title }}</h3>
        <p v-if="description" class="data-panel-card__desc">{{ description }}</p>
      </div>
      <slot name="extra" />
    </div>
    <div class="data-panel-card__body">
      <slot />
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = withDefaults(
  defineProps<{
    title: string;
    description?: string;
    primary?: boolean;
  }>(),
  {
    description: "",
    primary: false
  }
);

const cardClass = computed(() => ({
  "data-panel-card--primary": props.primary,
  "data-panel-card--secondary": !props.primary
}));
</script>

<style scoped lang="scss">
.data-panel-card {
  padding: 18px;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}
.data-panel-card__header {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 12px;
}
.data-panel-card__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--admin-page-text-primary);
}
.data-panel-card__desc {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
.data-panel-card__body {
  height: 360px;
}
</style>
