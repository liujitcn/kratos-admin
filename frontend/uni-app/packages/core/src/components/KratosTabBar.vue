<script setup lang="ts">
import { computed } from 'vue'
import { resolveModuleIcon, resolveStaticView } from '../module'
import { APP_MENU_ROOT_ID, useAppNavigation } from '../navigation'
import { resolveRootMenuId } from '../navigation-tree.mjs'

const props = defineProps<{ route: string }>()
const { menus, tabBar, navigate } = useAppNavigation()
const activeMenu = computed(() => {
  const routeMenu = menus.value.find((menu) => resolveStaticView(menu.viewKey) === props.route)
  if (!routeMenu) return
  const tabMenuId = resolveRootMenuId(menus.value, routeMenu.id, APP_MENU_ROOT_ID)
  return tabBar.value.find((menu) => menu.id === tabMenuId)
})
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
  box-sizing: border-box;
  display: flex;
  height: calc(48px + env(safe-area-inset-bottom));
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
  transform: translateY(clamp(1px, env(safe-area-inset-bottom), 7px));
  color: #333;
  font-size: 10px;
  line-height: 12px;
}
.kratos-tab-bar__icon {
  width: 28px;
  height: 28px;
  margin-bottom: 4px;
}
.kratos-tab-bar__text--active {
  color: #27ba9b;
}
</style>
