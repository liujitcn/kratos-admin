<template>
  <div class="table-box mqtt-message-page">
    <ProTable ref="proTable" row-key="id" :columns="columns" :request-api="requestTable" />

    <ProDialog
      v-model="detail.visible"
      :title="t('system.mqtt_message.title.detail')"
      width="min(960px, calc(100vw - 32px))"
      destroy-on-close
      :show-footer="false"
    >
      <template v-if="detail.item">
        <el-descriptions :column="2" border>
          <el-descriptions-item :label="t('system.mqtt_message.field.id')">{{ detail.item.id }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.mqtt_message.field.direction')">
            <el-tag size="small" effect="plain">{{ directionLabel(detail.item.direction) }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item :label="t('system.mqtt_message.field.topic')" :span="2">
            <code>{{ detail.item.topic }}</code>
          </el-descriptions-item>
          <el-descriptions-item :label="t('system.mqtt_message.field.format')">
            {{ payloadFormatLabel(detail.item.payload_format) }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('system.mqtt_message.field.status')">
            {{ statusLabel(detail.item.status) }}
          </el-descriptions-item>
          <el-descriptions-item :label="t('system.mqtt_message.field.size')">{{ detail.item.payload_size }} B</el-descriptions-item>
          <el-descriptions-item :label="t('system.mqtt_message.field.qos')">{{ detail.item.qos }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.mqtt_message.field.created_at')">{{ formatDateTime(detail.item.created_at) }}</el-descriptions-item>
          <el-descriptions-item :label="t('system.mqtt_message.field.handled_at')">{{ formatDateTime(detail.item.handled_at) }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.item.message_id" :label="t('system.mqtt_message.field.message_id')" :span="2">{{ detail.item.message_id }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.item.error_message" :label="t('system.mqtt_message.field.error')" :span="2">{{ detail.item.error_message }}</el-descriptions-item>
        </el-descriptions>

        <section v-if="detail.item.payload_json" class="mqtt-payload-section">
          <h3>{{ t('system.mqtt_message.field.payload_json') }}</h3>
          <pre class="mqtt-payload">{{ prettyJson(detail.item.payload_json) }}</pre>
        </section>
        <section class="mqtt-payload-section">
          <h3>{{ t('system.mqtt_message.field.payload_raw') }}</h3>
          <pre class="mqtt-payload">{{ decodePayload(detail.item.payload_raw, detail.item.payload_format) }}</pre>
        </section>
      </template>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive, ref } from "vue";
import { ElTag } from "element-plus";
import { View } from "@element-plus/icons-vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import ProDialog from "@liujitcn/kratos-admin-core/components/Dialog/ProDialog.vue";
import type { ColumnProps, EnumProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { formatDateTime } from "@liujitcn/kratos-admin-core/format";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseDictService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_dict";
import { defMqttMessageService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/mqtt_message";
import type {
  MqttMessageDetail,
  MqttMessageDirection,
  MqttMessagePayloadFormat,
  MqttMessageSummary,
  MqttMessageStatus,
  PageMqttMessageRequest
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/mqtt_message";
import {
  MqttMessageDirection as MqttMessageDirectionEnum,
  MqttMessagePayloadFormat as MqttMessagePayloadFormatEnum,
  MqttMessageStatus as MqttMessageStatusEnum
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/mqtt_message";
import type { OptionBaseDictResponse_BaseDictItem } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_dict";

defineOptions({ name: "MqttMessage", inheritAttrs: false });

const proTable = ref<ProTableInstance>();
const detail = reactive<{ visible: boolean; item?: MqttMessageDetail }>({ visible: false });
const DICTIONARY_CODES = {
  direction: "mqtt_message_direction",
  payloadFormat: "mqtt_message_payload_format",
  status: "mqtt_message_status"
} as const;

const dictionaryItems = ref<Record<string, OptionBaseDictResponse_BaseDictItem[]>>({});
let dictionaryLoadPromise: Promise<void> | undefined;

const directionValues: Record<string, MqttMessageDirection> = {
  PUBLISH: MqttMessageDirectionEnum.MQTT_MESSAGE_DIRECTION_PUBLISH,
  SUBSCRIBE: MqttMessageDirectionEnum.MQTT_MESSAGE_DIRECTION_SUBSCRIBE
};
const payloadFormatValues: Record<string, MqttMessagePayloadFormat> = {
  JSON: MqttMessagePayloadFormatEnum.MQTT_MESSAGE_PAYLOAD_FORMAT_JSON,
  TEXT: MqttMessagePayloadFormatEnum.MQTT_MESSAGE_PAYLOAD_FORMAT_TEXT,
  BINARY: MqttMessagePayloadFormatEnum.MQTT_MESSAGE_PAYLOAD_FORMAT_BINARY
};
const statusValues: Record<string, MqttMessageStatus> = {
  SENT: MqttMessageStatusEnum.MQTT_MESSAGE_STATUS_SENT,
  HANDLED: MqttMessageStatusEnum.MQTT_MESSAGE_STATUS_HANDLED,
  FAILED: MqttMessageStatusEnum.MQTT_MESSAGE_STATUS_FAILED
};

/** 加载 MQTT 页面依赖的三个字典。 */
async function loadDictionaries() {
  if (!dictionaryLoadPromise) {
    dictionaryLoadPromise = defBaseDictService.OptionBaseDict({}).then(response => {
      const codes = new Set<string>(Object.values(DICTIONARY_CODES));
      dictionaryItems.value = Object.fromEntries(
        (response.base_dicts ?? [])
          .filter(dictionary => codes.has(dictionary.code))
          .map(dictionary => [dictionary.code, dictionary.items ?? []])
      );
    });
  }
  await dictionaryLoadPromise;
}

/** 将字典项转换为表格搜索使用的 Proto 枚举选项。 */
async function loadEnumOptions(
  code: string,
  values: Record<string, number>
): Promise<EnumProps[]> {
  await loadDictionaries();
  return (dictionaryItems.value[code] ?? []).flatMap(item => {
    const value = values[item.value];
    return value === undefined ? [] : [{ label: item.label, value, tagType: item.tag_type }];
  });
}

const directionOptions = () => loadEnumOptions(DICTIONARY_CODES.direction, directionValues);
const payloadFormatOptions = () => loadEnumOptions(DICTIONARY_CODES.payloadFormat, payloadFormatValues);
const statusOptions = () => loadEnumOptions(DICTIONARY_CODES.status, statusValues);

const columns = computed(() => {
  const data: ColumnProps[] = [
    {
      prop: "direction",
      label: t("system.mqtt_message.field.direction"),
      width: 120,
      enum: directionOptions,
      search: { el: "select" },
      render: scope => h(ElTag, { size: "small", effect: "plain" }, () => directionLabel((scope.row as MqttMessageSummary).direction))
    },
    {
      prop: "topic",
      label: t("system.mqtt_message.field.topic"),
      minWidth: 280,
      search: { el: "input", props: { clearable: true } },
      render: scope => h("code", { title: scope.row.topic }, scope.row.topic)
    },
    {
      prop: "payload_format",
      label: t("system.mqtt_message.field.format"),
      width: 110,
      enum: payloadFormatOptions,
      search: { el: "select" },
      render: scope => payloadFormatLabel((scope.row as MqttMessageSummary).payload_format)
    },
    { prop: "payload_size", label: t("system.mqtt_message.field.size"), width: 120, align: "right" },
    { prop: "qos", label: t("system.mqtt_message.field.qos"), width: 80, align: "center" },
    {
      prop: "status",
      label: t("system.mqtt_message.field.status"),
      width: 120,
      enum: statusOptions,
      search: { el: "select" },
      render: scope => statusLabel((scope.row as MqttMessageSummary).status)
    },
    { prop: "latency_ms", label: t("system.mqtt_message.field.latency"), width: 120, align: "right" },
    { prop: "created_at", label: t("system.mqtt_message.field.created_at"), minWidth: 190, align: "center", render: scope => formatDateTime(scope.row.created_at) },
    {
      prop: "operation",
      label: t("common.field.operation"),
      cellType: "actions",
      actions: [{ label: t("common.action.view"), type: "primary", link: true, icon: View, onClick: scope => void openDetail(scope.row as MqttMessageSummary) }]
    }
  ];
  return data;
});

/** 请求 MQTT 消息分页列表。 */
async function requestTable(params: PageMqttMessageRequest) {
  const response = await defMqttMessageService.PageMqttMessage(buildPageRequest(params));
  return { data: { list: response.mqtt_messages ?? [], total: response.total } };
}

/** 打开 MQTT 消息详情。 */
async function openDetail(item: MqttMessageSummary) {
  detail.item = await defMqttMessageService.GetMqttMessage({ id: item.id });
  await loadDictionaries();
  detail.visible = true;
}

/** 获取消息方向本地化名称。 */
function directionLabel(direction: MqttMessageDirection | string) {
  return dictionaryLabel(DICTIONARY_CODES.direction, enumDictionaryValue(direction, directionValues));
}

/** 获取消息格式本地化名称。 */
function payloadFormatLabel(format: MqttMessagePayloadFormat | string) {
  return dictionaryLabel(DICTIONARY_CODES.payloadFormat, enumDictionaryValue(format, payloadFormatValues));
}

/** 获取消息状态本地化名称。 */
function statusLabel(status: MqttMessageStatus | string) {
  return dictionaryLabel(DICTIONARY_CODES.status, enumDictionaryValue(status, statusValues));
}

/** 将接口枚举值转换为字典中的稳定值，兼容字符串枚举响应。 */
function enumDictionaryValue(value: number | string, values: Record<string, number>) {
  const numericValue = typeof value === "number" ? value : Number(value);
  const matchedValue = Object.entries(values).find(([, enumValue]) => enumValue === numericValue)?.[0];
  if (matchedValue) return matchedValue;
  return typeof value === "string" ? value.replace(/^MQTT_MESSAGE_(?:DIRECTION|PAYLOAD_FORMAT|STATUS)_/, "") : "";
}

/** 读取字典标签，字典尚未加载或值未知时回退到稳定值。 */
function dictionaryLabel(code: string, value: string) {
  return dictionaryItems.value[code]?.find(item => item.value === value)?.label ?? (value || "-");
}

/** 将已解析 JSON 以缩进格式展示。 */
function prettyJson(value: string) {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

/** 将原始字节内容转换为可读文本，无法解码时展示 Base64。 */
function decodePayload(value: Uint8Array | string | undefined, format: MqttMessagePayloadFormat | string) {
  if (!value) return "";
  if (typeof value === "string") return value;
  if (enumDictionaryValue(format, payloadFormatValues) === "BINARY") return toBase64(value);
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(value);
  } catch {
    return toBase64(value);
  }
}

/** 将字节内容分块编码为 Base64，避免大 Payload 触发调用栈限制。 */
function toBase64(value: Uint8Array) {
  let binary = "";
  const chunkSize = 0x8000;
  for (let index = 0; index < value.length; index += chunkSize) {
    binary += String.fromCharCode(...value.subarray(index, index + chunkSize));
  }
  return btoa(binary);
}

onMounted(() => {
  void loadDictionaries();
});
</script>

<style scoped>
.mqtt-payload-section {
  margin-top: 18px;
}

.mqtt-payload-section h3 {
  margin: 0 0 8px;
  font-size: 14px;
}

.mqtt-payload {
  max-height: 420px;
  overflow: auto;
  padding: 12px;
  background: var(--el-fill-color-light);
  white-space: pre-wrap;
  word-break: break-word;
}
</style>
