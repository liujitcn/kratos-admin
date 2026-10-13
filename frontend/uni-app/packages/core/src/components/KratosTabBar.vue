<script setup lang="ts">
import { computed } from 'vue'
import { resolveModuleIcon, resolveStaticView } from '../module'
import { APP_MENU_ROOT_ID, useAppMenuBadge, useAppNavigation } from '../navigation'
import { useThemeStore } from '../stores'
import { resolveRootMenuId } from '../navigation-tree.mjs'

const props = defineProps<{ route: string }>()
const { menus, tabBar, navigate } = useAppNavigation()
const themeStore = useThemeStore()

/** 设计稿导航坞线稿图标（viewKey → SVG path）。 */
const TAB_ICON_PATHS: Record<string, string> = {
  HOME: 'M3 11 12 3l9 8M5 10v10h14V10',
  APP_ANALYTIC: 'M4 20V10M10 20V4M16 20v-7M21 20H3',
  APP_REPORT: 'M6 4h9l5 5v11H6zM14 4v5h5',
  MESSAGE_INBOX: 'M4 5h16v12H8l-4 4z',
  PROFILE_HOME: 'M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8ZM4 21c0-4 3.5-6 8-6s8 2 8 6',
}

/** 按设计稿线稿生成随主题与激活态取色的图标地址，未知 viewKey 返回空串走后端图标兜底。 */
const tabIconSrc = (viewKey: string, active: boolean): string => {
  const path = TAB_ICON_PATHS[viewKey]
  if (!path) return ''
  const color = active
    ? themeStore.theme === 'light'
      ? '#039e8c'
      : '#28f0d4'
    : themeStore.theme === 'light'
      ? '#9aabbb'
      : '#4e6378'
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="${color}" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="${path}"/></svg>`
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
}

const activeMenu = computed(() => {
  const routeMenu = menus.value.find((menu) => resolveStaticView(menu.viewKey) === props.route)
  if (!routeMenu) return
  const tabMenuId = resolveRootMenuId(menus.value, routeMenu.id, APP_MENU_ROOT_ID)
  return tabBar.value.find((menu) => menu.id === tabMenuId)
})
const visible = computed(() => Boolean(activeMenu.value && tabBar.value.length))
const messageBadge = useAppMenuBadge('MESSAGE_INBOX')
</script>

<template>
  <view v-if="visible" class="kratos-tab-bar" :class="themeStore.themeClass">
    <view
      v-for="item in tabBar"
      :key="item.id"
      class="kratos-tab-bar__item"
      :class="{ 'kratos-tab-bar__item--active': activeMenu?.id === item.id }"
      @tap="navigate(item.path)"
    >
      <view
        v-if="item.viewKey === 'MESSAGE_INBOX' && messageBadge"
        class="kratos-tab-bar__badge"
        aria-hidden="true"
      />
      <image
        v-if="tabIconSrc(item.viewKey, activeMenu?.id === item.id)"
        class="kratos-tab-bar__icon"
        :src="tabIconSrc(item.viewKey, activeMenu?.id === item.id)"
        mode="aspectFit"
      />
      <image
        v-else-if="resolveModuleIcon(item.icon)"
        class="kratos-tab-bar__icon"
        :src="
          resolveModuleIcon(
            activeMenu?.id === item.id && item.selectedIcon ? item.selectedIcon : item.icon,
          )
        "
        mode="aspectFit"
      />
      <view
        v-else-if="item.viewKey === 'MESSAGE_INBOX'"
        class="kratos-tab-bar__message-icon"
        :class="{ 'kratos-tab-bar__message-icon--active': activeMenu?.id === item.id }"
        aria-hidden="true"
      />
      <text :class="{ 'kratos-tab-bar__text--active': activeMenu?.id === item.id }">
        {{ item.title }}
      </text>
    </view>
  </view>
</template>

<style scoped>
.kratos-tab-bar {
  --kratos-tab-bar-color: var(--kratos-color-text-dim, #4e6378);
  --kratos-tab-bar-active-color: var(--kratos-color-primary, #28f0d4);

  position: fixed;
  right: 20rpx;
  bottom: env(safe-area-inset-bottom);
  left: 20rpx;
  z-index: 999;
  display: flex;
  height: 104rpx;
  padding: 0 8rpx;
  border: 1px solid var(--kratos-color-border-accent, rgba(120, 230, 215, 0.16));
  border-radius: 30rpx;
  background: var(--kratos-dock-bg, rgba(10, 18, 30, 0.72));
  box-shadow: 0 24rpx 64rpx rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(32rpx);
}
.kratos-tab-bar__item {
  position: relative;
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  color: var(--kratos-tab-bar-color);
  font-size: 10px;
  line-height: 12px;
}

.kratos-tab-bar__icon {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  margin-bottom: 4px;
}
.kratos-tab-bar__message-icon {
  position: relative;
  width: 22px;
  height: 18px;
  margin-bottom: 6px;
  border: 2px solid var(--kratos-tab-bar-color);
  border-radius: 6px;
  box-sizing: border-box;
}
.kratos-tab-bar__message-icon::before {
  position: absolute;
  top: 5px;
  left: 4px;
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--kratos-tab-bar-color);
  box-shadow:
    5px 0 var(--kratos-tab-bar-color),
    10px 0 var(--kratos-tab-bar-color);
  content: '';
}
.kratos-tab-bar__message-icon::after {
  position: absolute;
  bottom: -5px;
  left: 3px;
  width: 7px;
  height: 7px;
  border-right: 2px solid var(--kratos-tab-bar-color);
  border-bottom: 2px solid var(--kratos-tab-bar-color);
  background: var(--kratos-dock-bg, rgba(10, 18, 30, 0.72));
  content: '';
  transform: rotate(45deg);
}
.kratos-tab-bar__message-icon--active {
  border-color: var(--kratos-tab-bar-active-color);
}
.kratos-tab-bar__message-icon--active::before {
  background: var(--kratos-tab-bar-active-color);
  box-shadow:
    5px 0 var(--kratos-tab-bar-active-color),
    10px 0 var(--kratos-tab-bar-active-color);
}
.kratos-tab-bar__message-icon--active::after {
  border-color: var(--kratos-tab-bar-active-color);
}
.kratos-tab-bar__text--active {
  color: var(--kratos-tab-bar-active-color);
  font-weight: 700;
  text-shadow: 0 0 12px currentColor;
}
.kratos-tab-bar__badge {
  position: absolute;
  top: 0;
  right: calc(50% - 19px);
  width: 10px;
  height: 10px;
  box-sizing: border-box;
  border: 2px solid var(--kratos-dock-bg, rgba(10, 18, 30, 0.72));
  border-radius: 50%;
  background: var(--kratos-color-danger, #ff4d6d);
  box-shadow: 0 0 12px var(--kratos-color-danger, #ff4d6d);
}
</style>
