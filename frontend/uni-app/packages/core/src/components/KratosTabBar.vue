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
      ? '#e1251b'
      : '#ff4d4f'
    : themeStore.theme === 'light'
      ? '#8c8c8c'
      : '#8c8c8c'
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
      <text
        class="kratos-tab-bar__text"
        :class="{ 'kratos-tab-bar__text--active': activeMenu?.id === item.id }"
      >
        {{ item.title }}
      </text>
    </view>
  </view>
</template>

<style scoped>
.kratos-tab-bar {
  --kratos-tab-bar-color: var(--kratos-color-text-dim, #8c8c8c);
  --kratos-tab-bar-active-color: var(--kratos-color-primary, #ff4d4f);

  position: fixed;
  right: 20rpx;
  bottom: env(safe-area-inset-bottom);
  left: 20rpx;
  z-index: 999;
  display: flex;
  box-sizing: border-box;
  height: 104rpx;
  padding: 0 8rpx;
  border: 1rpx solid var(--kratos-color-border-accent, rgba(120, 230, 215, 0.16));
  border-radius: 30rpx;
  background: var(--kratos-dock-bg, rgba(10, 18, 30, 0.72));
  box-shadow: 0 12rpx 32rpx rgba(0, 0, 0, 0.2);
}
.kratos-tab-bar__item {
  position: relative;
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  height: 100%;
  padding: 10rpx 0 8rpx;
  box-sizing: border-box;
  color: var(--kratos-tab-bar-color);
  font-size: 20rpx;
  line-height: 28rpx;
}

.kratos-tab-bar__icon {
  flex-shrink: 0;
  width: 44rpx;
  height: 44rpx;
  margin-bottom: 4rpx;
}
.kratos-tab-bar__message-icon {
  position: relative;
  width: 36rpx;
  height: 30rpx;
  margin-bottom: 8rpx;
  border: 2rpx solid var(--kratos-tab-bar-color);
  border-radius: 8rpx;
  box-sizing: border-box;
}
.kratos-tab-bar__message-icon::before {
  position: absolute;
  top: 8rpx;
  left: 7rpx;
  width: 4rpx;
  height: 4rpx;
  border-radius: 50%;
  background: var(--kratos-tab-bar-color);
  box-shadow:
    8rpx 0 var(--kratos-tab-bar-color),
    16rpx 0 var(--kratos-tab-bar-color);
  content: '';
}
.kratos-tab-bar__message-icon::after {
  position: absolute;
  bottom: -8rpx;
  left: 5rpx;
  width: 10rpx;
  height: 10rpx;
  border-right: 2rpx solid var(--kratos-tab-bar-color);
  border-bottom: 2rpx solid var(--kratos-tab-bar-color);
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
    8rpx 0 var(--kratos-tab-bar-active-color),
    16rpx 0 var(--kratos-tab-bar-active-color);
}
.kratos-tab-bar__message-icon--active::after {
  border-color: var(--kratos-tab-bar-active-color);
}
.kratos-tab-bar__text {
  display: block;
  min-height: 28rpx;
  text-align: center;
  white-space: nowrap;
}
.kratos-tab-bar__text--active {
  color: var(--kratos-tab-bar-active-color);
  font-weight: 700;
}
.kratos-tab-bar__badge {
  position: absolute;
  top: 0;
  right: calc(50% - 30rpx);
  width: 16rpx;
  height: 16rpx;
  box-sizing: border-box;
  border: 2rpx solid var(--kratos-dock-bg, rgba(10, 18, 30, 0.72));
  border-radius: 50%;
  background: var(--kratos-color-danger, #ff4d4f);
  box-shadow: 0 0 12rpx var(--kratos-color-danger, #ff4d4f);
}
</style>
