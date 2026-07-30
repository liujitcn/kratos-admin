<template>
  <div :class="['breadcrumb-box mask-image', !globalStore.breadcrumbIcon && 'no-icon']">
    <el-breadcrumb :separator-icon="ArrowRight">
      <transition-group name="breadcrumb">
        <el-breadcrumb-item v-for="(item, index) in breadcrumbList" :key="`${item.path}-${item.name || index}`">
          <div
            class="el-breadcrumb__inner is-link"
            :class="{ 'item-no-icon': !getRouteMetaIcon(item.meta) }"
            @click="onBreadcrumbClick(item, index)"
          >
            <el-icon v-if="getRouteMetaIcon(item.meta) && globalStore.breadcrumbIcon" class="breadcrumb-icon">
              <component :is="getRouteMetaIcon(item.meta)"></component>
            </el-icon>
            <span class="breadcrumb-title">{{ getRouteMetaTitle(item.meta) }}</span>
          </div>
        </el-breadcrumb-item>
      </transition-group>
    </el-breadcrumb>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { HOME_URL } from "@/config";
import { useRoute, useRouter } from "vue-router";
import { ArrowRight } from "@element-plus/icons-vue";
import { useAuthStore } from "@/stores/modules/auth";
import { useGlobalStore } from "@/stores/modules/global";
import type { RouteItem } from "@/rpc/system/admin/v1/auth";
import { BaseMenuType } from "@/rpc/system/common/v1/enum";
import { getRouteMetaIcon, getRouteMetaTitle, getRouteTarget, isExternalPath } from "@/utils";

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const globalStore = useGlobalStore();

const breadcrumbList = computed(() => {
  let breadcrumbData = authStore.breadcrumbListGet[route.matched[route.matched.length - 1].path] ?? [];
  // 🙅‍♀️不需要首页面包屑可删除以下判断
  if (breadcrumbData[0]?.path !== HOME_URL && getRouteMetaTitle(breadcrumbData[0]?.meta) !== "首页") {
    breadcrumbData = [
      { path: HOME_URL, name: "home", type: BaseMenuType.MENU, meta: { icon: "HomeFilled", title: "首页", params: [] }, children: [] },
      ...breadcrumbData
    ];
  }
  return breadcrumbData;
});

// 处理面包屑点击，兼容模板索引可能被推断为字符串或数字的情况。
const onBreadcrumbClick = (item: RouteItem, index: string | number) => {
  const routeIndex = Number(index);
  const target = getRouteTarget(item);
  if (routeIndex === breadcrumbList.value.length - 1 || !target) return;
  if (isExternalPath(target)) {
    window.open(target, "_blank", "noopener,noreferrer");
    return;
  }
  router.push(target);
};
</script>

<style scoped lang="scss">
.breadcrumb-box {
  display: flex;
  align-items: center;
  overflow: hidden;
  .el-breadcrumb {
    white-space: nowrap;
    .el-breadcrumb__item {
      position: relative;
      float: none;
      display: inline-block;
      .item-no-icon {
        transform: translateY(-3px);
      }
      .el-breadcrumb__inner {
        display: inline-flex;
        &.is-link {
          color: var(--el-header-text-color);
          &:hover {
            color: var(--el-color-primary);
          }
        }
        .breadcrumb-icon {
          margin-top: 1px;
          margin-right: 6px;
          font-size: 16px;
        }
        .breadcrumb-title {
          margin-top: 2px;
        }
      }
      &:last-child .el-breadcrumb__inner,
      &:last-child .el-breadcrumb__inner:hover {
        color: var(--el-header-text-color-regular);
      }
      :deep(.el-breadcrumb__separator) {
        transform: translateY(-1px);
      }
    }
  }
}
.no-icon {
  .el-breadcrumb {
    .el-breadcrumb__item {
      top: -2px;
      :deep(.el-breadcrumb__separator) {
        top: 4px;
      }
      .item-no-icon {
        transform: translateY(0);
      }
    }
  }
}
</style>
