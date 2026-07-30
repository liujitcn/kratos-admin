<template>
  <template v-for="subItem in menuList" :key="getRouteMenuKey(subItem)">
    <el-sub-menu v-if="isSubMenu(subItem)" :index="getRouteMenuKey(subItem)">
      <template #title>
        <el-icon v-if="getRouteMetaIcon(subItem.meta)">
          <component :is="getRouteMetaIcon(subItem.meta)"></component>
        </el-icon>
        <span class="sle">{{ getRouteMetaTitle(subItem.meta) }}</span>
      </template>
      <SubMenu :menu-list="getSubMenuChildren(subItem)" />
    </el-sub-menu>
    <el-menu-item v-else :index="getRouteMenuKey(getMenuItem(subItem))" @click="handleClickMenu(getMenuItem(subItem))">
      <el-icon v-if="getRouteMetaIcon(getMenuItem(subItem).meta)">
        <component :is="getRouteMetaIcon(getMenuItem(subItem).meta)"></component>
      </el-icon>
      <template #title>
        <span class="sle">{{ getRouteMetaTitle(getMenuItem(subItem).meta) }}</span>
      </template>
    </el-menu-item>
  </template>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import type { RouteItem } from "@/rpc/system/admin/v1/auth";
import {
  getRouteTarget,
  getRouteMenuKey,
  getRouteMetaAlwaysShow,
  getRouteMetaHidden,
  getRouteMetaIcon,
  getRouteMetaTitle,
  isExternalPath
} from "@/utils";

defineProps<{ menuList: RouteItem[] }>();

const router = useRouter();
/** 侧边菜单渲染时必须具备路径的菜单项。 */
type VisibleRouteItem = RouteItem & { path: string };

const ensureRoutePath = (subItem: RouteItem): VisibleRouteItem => {
  return {
    ...subItem,
    path: subItem.path ?? ""
  };
};

/** 获取当前菜单节点应继续渲染的可见子菜单。 */
const getSubMenuChildren = (subItem: RouteItem) => {
  const visibleChildren = (subItem.children ?? []).filter(item => !getRouteMetaHidden(item.meta));
  if (getRouteMetaAlwaysShow(subItem.meta) || visibleChildren.length !== 1) return visibleChildren;
  return visibleChildren[0].children ?? [];
};

const getMenuItem = (subItem: RouteItem): VisibleRouteItem => {
  const visibleChildren = (subItem.children ?? []).filter(item => !getRouteMetaHidden(item.meta));
  if (getRouteMetaAlwaysShow(subItem.meta) || visibleChildren.length !== 1) return ensureRoutePath(subItem);
  return ensureRoutePath(visibleChildren[0]);
};

/** 判断当前节点是否需要渲染为子菜单容器。 */
const isSubMenu = (subItem: RouteItem) => {
  const visibleChildren = (subItem.children ?? []).filter(item => !getRouteMetaHidden(item.meta));
  if (!visibleChildren.length) return false;
  return getRouteMetaAlwaysShow(subItem.meta) || visibleChildren.length > 1;
};

/** 处理侧边菜单点击并跳转内部路由或外链。 */
const handleClickMenu = (subItem: RouteItem) => {
  const menuItem = getMenuItem(subItem);
  const target = getRouteTarget(menuItem);
  if (!target) return;
  if (isExternalPath(target)) {
    window.open(target, "_blank", "noopener,noreferrer");
    return;
  }
  router.push(target);
};
</script>

<style lang="scss">
.el-sub-menu .el-sub-menu__title:hover {
  color: var(--el-menu-hover-text-color) !important;
  background-color: transparent !important;
}
.el-menu--collapse {
  .is-active {
    .el-sub-menu__title {
      color: #ffffff !important;
      background-color: var(--el-color-primary) !important;
    }
  }
}
.el-menu-item {
  &:hover {
    color: var(--el-menu-hover-text-color);
  }
  &.is-active {
    color: var(--el-menu-active-color) !important;
    background-color: var(--el-menu-active-bg-color) !important;
    &::before {
      position: absolute;
      top: 0;
      bottom: 0;
      width: 4px;
      content: "";
      background-color: var(--el-color-primary);
    }
  }
}
.vertical,
.classic,
.transverse {
  .el-menu-item {
    &.is-active {
      &::before {
        left: 0;
      }
    }
  }
}
.columns {
  .el-menu-item {
    &.is-active {
      &::before {
        right: 0;
      }
    }
  }
}
</style>
