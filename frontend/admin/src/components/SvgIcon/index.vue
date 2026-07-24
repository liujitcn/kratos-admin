<template>
  <svg :style="iconStyle" aria-hidden="true">
    <use :xlink:href="symbolId" />
  </svg>
</template>

<script setup lang="ts" name="SvgIcon">
import { computed, CSSProperties } from "vue";

/** SvgIcon 组件属性。 */
interface SvgProps {
  iconClass: string; // 图标名称
  prefix?: string; // 图标的前缀 ==> 非必传（默认为"icon"）
  size?: string | number; // 图标尺寸 ==> 非必传
  iconStyle?: CSSProperties; // 图标的样式 ==> 非必传
}

const props = withDefaults(defineProps<SvgProps>(), {
  prefix: "icon",
  size: "",
  iconStyle: () => ({})
});

const iconStyle = computed<CSSProperties>(() => {
  const style: CSSProperties = { ...props.iconStyle };
  if (props.size !== "" && props.size !== undefined) {
    const sizeValue = typeof props.size === "number" ? `${props.size}px` : props.size;
    style.width = sizeValue;
    style.height = sizeValue;
  }
  return style;
});

const symbolId = computed(() => `#${props.prefix}-${props.iconClass}`);
</script>
