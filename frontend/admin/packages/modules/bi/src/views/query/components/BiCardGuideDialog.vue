<template>
  <BiCardFormDialog
    :model-value="visible"
    :sources="sources"
    :initial-query="guide?.query"
    :initial-result="result"
    :initial-title="guide?.default_name"
    @update:model-value="emit('update:modelValue', $event)"
    @saved="emit('saved')"
  />
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import BiCardFormDialog from "../../../components/BiCardFormDialog.vue";
import { defBiQueryService } from "../../../api/bi/admin/v1/bi_query";
import { defBiQueryRecordService } from "../../../api/bi/admin/v1/bi_query_record";
import { normalizeBiQueryResult } from "../../../bi-response";
import type { BiQueryResult, BiQuerySpec, BiSourceDescriptor } from "../../../rpc/bi/admin/v1/bi_query";

/** 助手消息中的 BI 建卡引导，由问数工具提供。 */
interface BiCardGuide {
  query?: BiQuerySpec;
  query_record_id?: number;
  default_name?: string;
}

const props = defineProps<{ modelValue: boolean; guide?: BiCardGuide }>();
const emit = defineEmits<{ "update:modelValue": [value: boolean]; saved: [] }>();
const visible = ref(false);
const sources = ref<BiSourceDescriptor[]>([]);
const result = ref<BiQueryResult>();

/** 使用 BI 请求封装准备完整数据，关闭或切换消息后忽略迟到响应。 */
watch(
  () => [props.modelValue, props.guide] as const,
  async ([value], _previous, onCleanup) => {
    visible.value = false;
    if (!value) return;
    let active = true;
    onCleanup(() => {
      active = false;
    });
    result.value = undefined;
    try {
      if (!sources.value.length) {
        const listed = await defBiQueryService.ListBiSources({});
        if (!active) return;
        sources.value = listed.sources;
      }
      const recordID = props.guide?.query_record_id ?? 0;
      if (recordID > 0) {
        const detail = await defBiQueryRecordService.GetBiQuery({ id: recordID });
        if (!active) return;
        result.value = detail.result ? normalizeBiQueryResult(detail.result) : undefined;
      }
      visible.value = true;
    } catch {
      // 请求封装已提示服务端错误，这里同步关闭助手中的弹窗状态。
      if (active) emit("update:modelValue", false);
    }
  },
  { immediate: true }
);
</script>
