<template>
  <div>
    <template v-if="componentId === 'TreeFilter'">
      <div class="tree-example">
        <TreeFilter
          title="项目分类"
          :data="treeData"
          :multiple="multiple"
          :default-value="selected"
          @change="selected = $event"
        />
        <div class="tree-result">
          <el-switch v-model="multiple" active-text="多选" inactive-text="单选" @change="selected = multiple ? [] : ''" />
          <p class="example-tip">选择结果：{{ JSON.stringify(selected) }}</p>
          <p class="example-tip">可以搜索节点名称，并展开或收起树。</p>
        </div>
      </div>
    </template>
    <template v-else>
      <SelectFilter :data="filterData" :default-values="filterDefaults" @change="selected = $event" />
      <pre class="example-result">{{ JSON.stringify(selected, null, 2) }}</pre>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import TreeFilter from "@/components/TreeFilter/index.vue";
import SelectFilter from "@/components/SelectFilter/index.vue";

defineProps<{ componentId: string }>();
const multiple = ref(false);
const selected = ref<unknown>("");
const filterDefaults = { status: "", channels: [""] };
const treeData = [
  {
    id: "campus",
    label: "校园服务",
    children: [
      { id: "payment", label: "校园支付" },
      { id: "visitor", label: "访客管理" }
    ]
  },
  {
    id: "teaching",
    label: "教学应用",
    children: [
      { id: "course", label: "课程管理" },
      { id: "schedule", label: "课表服务" }
    ]
  }
];
const filterData = [
  {
    title: "项目状态",
    key: "status",
    options: [
      { label: "全部", value: "" },
      { label: "启用", value: "enabled" },
      { label: "停用", value: "disabled" }
    ]
  },
  {
    title: "通知渠道",
    key: "channels",
    multiple: true,
    options: [
      { label: "全部", value: "" },
      { label: "电子邮件", value: "email" },
      { label: "站内消息", value: "inbox" }
    ]
  }
];
</script>

<style scoped lang="scss">
.tree-example {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
}
.tree-example :deep(.tree-filter) {
  width: 260px;
  min-height: 300px;
  padding: 16px;
  box-shadow: none;
}
.tree-result {
  flex: 1;
  min-width: 180px;
  padding-top: 12px;
}
.example-tip {
  font-size: 13px;
  line-height: 1.7;
  color: var(--admin-page-text-secondary);
}
.example-result {
  padding: 14px;
  margin-top: 20px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--admin-page-text-secondary);
  background: var(--admin-page-card-bg-muted);
  border-radius: var(--admin-page-radius);
}
</style>
