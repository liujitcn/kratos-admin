<template>
  <div v-if="active" class="session-example">
    <AiSessionPanel
      v-model:active="current"
      v-model:keyword="keyword"
      :sessions="filteredSessions"
      :hide-collapse="true"
      title="本地演示会话"
      @create="createSession"
      @action="handleAction"
    >
      <template #navigation><p class="session-example__hint">搜索、切换和编辑仅影响本页数据。</p></template>
    </AiSessionPanel>
    <p class="session-example__status" aria-live="polite">{{ feedback }} · 当前会话：{{ current || "未选择" }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onDeactivated, ref } from "vue";
import AiSessionPanel from "@liujitcn/kratos-admin-system/components/AiSessionPanel/index.vue";
import type { AiSession } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_session";

defineProps<{ componentId: string }>();
/** 会话列表示例只保留组件实际使用的字段。 */
type DemoSession = Pick<AiSession, "id" | "title" | "summary">;
const active = ref(true);
const keyword = ref("");
const current = ref("demo-1");
const feedback = ref("可从会话右侧菜单重命名或删除");
const sessions = ref<DemoSession[]>([
  { id: "demo-1", title: "分析本周服务情况", summary: "整理业务指标与变化" },
  { id: "demo-2", title: "编写接口说明", summary: "生成字段和调用示例" }
]);
let sequence = 2;
const filteredSessions = computed(() => sessions.value.filter(item => `${item.title}${item.summary}`.includes(keyword.value)));

/** 在内存中添加会话并选中。 */
function createSession() {
  const item = { id: `demo-${++sequence}`, title: `新会话 ${sequence}`, summary: "本地会话示例" };
  sessions.value.unshift(item);
  current.value = item.id;
  keyword.value = "";
  feedback.value = "已在本地创建会话";
}

/** 演示父页面接收编辑事件，不调用会话服务。 */
function handleAction({ action, item }: { action: "rename" | "delete"; item: DemoSession }) {
  if (action === "rename") {
    item.title = `${item.title}（已重命名）`;
    feedback.value = "已在本地重命名会话";
    return;
  }
  sessions.value = sessions.value.filter(session => session.id !== item.id);
  if (current.value === item.id) current.value = sessions.value[0]?.id ?? "";
  feedback.value = "已从本地列表移除会话";
}

onActivated(() => {
  active.value = true;
});
onDeactivated(() => {
  active.value = false;
});
</script>

<style scoped>
.session-example {
  max-width: 420px;
  overflow: hidden;
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
}
.session-example :deep(.agent-session-panel) {
  display: flex;
  min-height: 320px;
  max-height: 460px;
  border-right: 0;
}
.session-example__hint,
.session-example__status {
  padding: 0 14px;
  color: var(--admin-page-text-secondary);
  font-size: 12px;
}
.session-example__status {
  overflow-wrap: anywhere;
}
</style>
