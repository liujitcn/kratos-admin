<template>
  <div v-if="active" class="tenant-example">
    <template v-if="componentId === 'TenantSelect'">
      <TenantSelect v-model="tenantId" :options="options" placeholder="选择示例租户" clearable style="width: 260px" />
      <p v-if="!isDefaultTenant" class="example-tip">当前账号不是默认租户，组件按租户权限规则隐藏选择框。</p>
      <p class="example-tip">当前选择：<TenantText :tenant-id="tenantId" :options="options" /></p>
    </template>
    <template v-else>
      <el-radio-group v-model="tenantId"
        ><el-radio-button :value="1">北校区</el-radio-button><el-radio-button :value="2">南校区</el-radio-button></el-radio-group
      >
      <p class="example-tip">选项匹配：<TenantText :tenant-id="tenantId" :options="options" /></p>
      <p class="example-tip">直接传入名称：<TenantText tenant-name="智慧校园示例租户" /></p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import TenantSelect from "@/components/TenantSelect/index.vue";
import TenantText from "@/components/TenantText/index.vue";
import { useTenantScope } from "@/tenant";
import { useDemoActive } from "./useDemoActive";

defineProps<{ componentId: string }>();
const active = useDemoActive();
const tenantId = ref<string | number>(1);
const { isDefaultTenant } = useTenantScope();
const options = [
  { label: "北校区", value: 1, disabled: false },
  { label: "南校区", value: 2, disabled: false }
];
</script>

<style scoped lang="scss">
.example-tip {
  margin: 16px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
</style>
