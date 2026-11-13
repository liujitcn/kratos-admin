<template>
  <div v-if="active" class="component-common-example">
    <template v-if="componentId === 'SelectIcon'">
      <SelectIcon v-model:icon-value="icon" title="选择图标" />
      <p class="example-tip">当前图标：{{ icon || "未选择" }}</p>
    </template>
    <template v-else-if="componentId === 'SvgIcon'">
      <svg class="svg-symbol" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
        <symbol id="component-docs-circle" viewBox="0 0 24 24">
          <circle cx="12" cy="12" r="9" fill="none" stroke="currentColor" stroke-width="2" />
          <path d="m8 12 3 3 5-6" fill="none" stroke="currentColor" stroke-width="2" />
        </symbol>
      </svg>
      <div class="icon-sizes">
        <SvgIcon v-for="size in [20, 32, 48]" :key="size" icon-class="circle" prefix="component-docs" :size="size" />
      </div>
      <p class="example-tip">按 SVG symbol 名称引用，同一图标可设置不同尺寸。</p>
    </template>
    <template v-else-if="componentId === 'SwitchDark'">
      <SwitchDark />
      <p class="example-tip">切换后立即应用当前管理端的明暗主题。</p>
    </template>
    <template v-else-if="componentId === 'LocaleSwitch'">
      <div class="locale-control"><LocaleSwitch /><span>点击选择语言</span></div>
      <p class="example-tip">使用管理端全局语言设置；当前语言：{{ locale }}。</p>
    </template>
    <template v-else-if="componentId === 'Loading'">
      <el-switch v-model="loading" active-text="显示加载状态" />
      <div class="loading-example">
        <Loading v-if="loading" />
        <p v-else class="example-tip">内容已经准备就绪。</p>
      </div>
    </template>
    <div v-else class="error-example"><ErrorMessage code="404" :image="notFound" message="未找到你要访问的页面" /></div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import SelectIcon from "@/components/SelectIcon/index.vue";
import SvgIcon from "@/components/SvgIcon/index.vue";
import SwitchDark from "@/components/SwitchDark/index.vue";
import LocaleSwitch from "@/components/LocaleSwitch/index.vue";
import Loading from "@/components/Loading/index.vue";
import ErrorMessage from "@/components/ErrorMessage/index.vue";
import { useLocaleStore } from "@/locales";
import notFound from "@/assets/images/404.png";
import { useDemoActive } from "./useDemoActive";

defineProps<{ componentId: string }>();
const active = useDemoActive();
const icon = ref("Setting");
const loading = ref(true);
const { locale } = useLocaleStore();
</script>

<style scoped lang="scss">
.component-common-example {
  min-width: 0;
}
.example-tip {
  margin: 16px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
.svg-symbol {
  position: absolute;
  width: 0;
  height: 0;
  overflow: hidden;
}
.icon-sizes {
  display: flex;
  gap: 28px;
  align-items: center;
  color: var(--el-color-primary);
}
.locale-control {
  display: flex;
  gap: 12px;
  align-items: center;
  color: var(--admin-page-text-primary);
}
.locale-control :deep(svg) {
  cursor: pointer;
}
.loading-example {
  display: grid;
  min-height: 240px;
  place-items: center;
}
.error-example {
  min-height: 300px;
}
.error-example :deep(.not-img) {
  width: min(38%, 240px);
  margin-right: 28px;
}
.error-example :deep(.not-detail h4) {
  font-size: 15px;
}
</style>
