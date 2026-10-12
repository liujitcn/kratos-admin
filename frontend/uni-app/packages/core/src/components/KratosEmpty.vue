<script setup lang="ts">
defineOptions({ name: 'KratosEmpty' })

withDefaults(
  defineProps<{
    /** 图标名称（uni-icons），不传则不渲染图标位。 */
    icon?: string
    /** 主文案。 */
    title?: string
    /** 次级提示文案。 */
    description?: string
  }>(),
  {
    icon: '',
    title: '',
    description: '',
  },
)

defineSlots<{
  /** 操作区插槽：渲染引导按钮等自定义内容。 */
  default?: () => unknown
}>()
</script>

<template>
  <view class="k-empty">
    <view v-if="icon" class="k-empty__icon">
      <uni-icons :type="icon" size="26" color="var(--kratos-color-primary)" aria-hidden="true" />
    </view>
    <text v-if="title" class="k-empty__title">{{ title }}</text>
    <text v-if="description" class="k-empty__desc">{{ description }}</text>
    <slot />
  </view>
</template>
