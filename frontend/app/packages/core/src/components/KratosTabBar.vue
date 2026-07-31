<script setup lang="ts">
import { computed } from 'vue'
import { resolveModuleIcon, resolveStaticView } from '../module'
import { useAppNavigation } from '../navigation'

const props = defineProps<{ route: string }>()
const { tabBar, navigate } = useAppNavigation()
const activeMenu = computed(() =>
  tabBar.value.find((menu) => resolveStaticView(menu.viewKey) === props.route),
)
const visible = computed(() => Boolean(activeMenu.value && tabBar.value.length))
</script>

<template>
  <view v-if="visible" class="kratos-tab-bar">
    <view
      v-for="item in tabBar"
      :key="item.id"
      class="kratos-tab-bar__item"
      @tap="navigate(item.path)"
    >
      <image
        v-if="resolveModuleIcon(item.icon)"
        class="kratos-tab-bar__icon"
        :src="
          resolveModuleIcon(
            activeMenu?.id === item.id && item.selectedIcon ? item.selectedIcon : item.icon,
          )
        "
        mode="aspectFit"
      />
      <text :class="{ 'kratos-tab-bar__text--active': activeMenu?.id === item.id }">
        {{ item.title }}
      </text>
    </view>
  </view>
</template>

<style scoped>
.kratos-tab-bar {
  position: fixed;
  right: 0;
  bottom: 0;
  left: 0;
  z-index: 999;
  display: flex;
  min-height: 100rpx;
  padding-bottom: env(safe-area-inset-bottom);
  border-top: 1px solid #eee;
  background: #fff;
}
.kratos-tab-bar__item {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  color: #333;
  font-size: 24rpx;
}
.kratos-tab-bar__icon {
  width: 42rpx;
  height: 42rpx;
  margin-bottom: 4rpx;
}
.kratos-tab-bar__text--active {
  color: #27ba9b;
}
</style>
