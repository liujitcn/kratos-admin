<script setup lang="ts">
import { computed } from 'vue'
import { resolveModuleIcon, resolveStaticView } from '../module'
import { APP_MENU_ROOT_ID, useAppMenuBadge, useAppNavigation } from '../navigation'
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
const messageBadge = useAppMenuBadge('MESSAGE_INBOX')
</script>

<template>
  <view v-if="visible" class="kratos-tab-bar">
    <view
      v-for="item in tabBar"
      :key="item.id"
      class="kratos-tab-bar__item"
      @tap="navigate(item.path)"
    >
      <view
        v-if="item.viewKey === 'MESSAGE_INBOX' && messageBadge"
        class="kratos-tab-bar__badge"
        aria-hidden="true"
      />
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
  position: relative;
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
.kratos-tab-bar__message-icon {
  position: relative;
  width: 22px;
  height: 18px;
  margin-bottom: 6px;
  border: 2px solid #9aa4b2;
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
  background: #9aa4b2;
  box-shadow:
    5px 0 #9aa4b2,
    10px 0 #9aa4b2;
  content: '';
}
.kratos-tab-bar__message-icon::after {
  position: absolute;
  bottom: -5px;
  left: 3px;
  width: 7px;
  height: 7px;
  border-right: 2px solid #9aa4b2;
  border-bottom: 2px solid #9aa4b2;
  background: #fff;
  content: '';
  transform: rotate(45deg);
}
.kratos-tab-bar__message-icon--active {
  border-color: #27ba9b;
}
.kratos-tab-bar__message-icon--active::before {
  background: #27ba9b;
  box-shadow:
    5px 0 #27ba9b,
    10px 0 #27ba9b;
}
.kratos-tab-bar__message-icon--active::after {
  border-color: #27ba9b;
}
.kratos-tab-bar__text--active {
  color: #27ba9b;
}
.kratos-tab-bar__badge {
  position: absolute;
  top: 0;
  right: calc(50% - 19px);
  width: 10px;
  height: 10px;
  box-sizing: border-box;
  border: 2px solid #fff;
  border-radius: 50%;
  background: #e5484d;
}
</style>
