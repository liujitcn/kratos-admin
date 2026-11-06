<template>
  <main class="bi-query-page">
    <header class="bi-query-page__header">
      <div>
        <h2>{{ t("bi.query.subtitle") }}</h2>
        <p>{{ t("bi.query.guide") }}</p>
      </div>
      <div>
        <el-button v-if="historyCollapsed" @click="historyCollapsed = false">{{ t("bi.query.history") }}</el-button
        ><el-button @click="router.push('/bi/card')">{{ t("bi.card.library") }}</el-button>
      </div>
    </header>
    <div class="bi-query-layout" :class="{ 'is-history-collapsed': historyCollapsed }">
      <SessionPanel
        v-if="!historyCollapsed"
        v-model:keyword="keyword"
        :active="activeRecord"
        :sessions="sessions"
        :editable="false" :title="t('bi.query.history')" :search-placeholder="t('bi.query.search_history')"
        @change="item => selectRecord(Number(item.id))"
        @create="newQuestion"
        @toggle-collapse="historyCollapsed = true"
      />
      <section class="bi-query-chat" :aria-label="t('bi.query.title')">
        <div class="bi-query-messages" aria-live="polite">
          <div v-if="submittedQuestion" class="bi-query-bubble">{{ submittedQuestion }}</div>
          <el-skeleton v-if="loading" :rows="5" animated /><el-alert
            v-else-if="error"
            :title="error"
            type="error"
            show-icon
            :closable="false"
          /><template v-else-if="response"
            ><div class="bi-query-assistant">{{ t("bi.query.assistant") }}</div>
            <h3>{{ response.title }}</h3>
            <p>{{ t("bi.query.result_ready") }}</p>
            <div class="bi-query-facts">
              <p>
                {{ t("bi.card.source") }}：{{
                  sources.find(item => item.code === response?.query?.source_code)?.name || response.query?.source_code
                }}
              </p>
              <p>{{ t("bi.card.dimensions") }}：{{ response.query?.dimensions.join("、") || t("bi.card.ungrouped") }}</p>
              <p>{{ t("bi.card.measure") }}：{{ response.query?.measures.map(item => item.code).join("、") }}</p>
            </div>
            <el-button link type="primary" @click="showTable = !showTable">{{
              t(showTable ? "bi.query.hide_details" : "bi.card.table_tab")
            }}</el-button
            ><el-table v-if="showTable" :data="tableRows" border stripe
              ><el-table-column
                v-for="(column, index) in response.result?.columns || []"
                :key="column.code"
                :prop="String(index)"
                :label="column.name"
                min-width="120" /></el-table></template
          ><el-empty v-else :description="t('bi.query.empty')" />
        </div>
        <div class="bi-query-compose">
          <div class="bi-query-compose__input">
            <el-input
              v-model="question"
              type="textarea"
              :autosize="{ minRows: 2, maxRows: 5 }"
              :placeholder="t('bi.query.placeholder')"
              :aria-label="t('bi.query.placeholder')"
              :disabled="loading"
              :readonly="recording"
              @keydown.enter="handleInputEnter($event as KeyboardEvent)"
            />
            <div class="bi-query-compose__footer">
              <span role="status">{{
                recording
                  ? t("bi.query.recognizing_voice")
                  : sources.find(source => source.code === response?.query?.source_code)?.name
              }}</span>
              <div class="bi-query-compose__actions">
                <AiModelSelector
                  :model-providers="modelProviders"
                  :provider-id="selectedModel.provider_id"
                  :model-name="selectedModel.model_name"
                  :disabled="loading || loadingModels"
                  @change="handleModelChange"
                />
                <el-tooltip :content="t(recording ? 'bi.query.stop_voice_input' : 'bi.query.voice_input')" placement="top">
                  <button
                    class="bi-query-compose__button"
                    :class="{ 'is-active': recording }"
                    type="button"
                    :disabled="loading"
                    :aria-pressed="recording"
                    :aria-label="t(recording ? 'bi.query.stop_voice_input' : 'bi.query.voice_input')"
                    @click="recording ? stopRecord() : startRecord()"
                  >
                    <el-icon :class="{ 'is-loading': recording }"
                      ><component :is="recording ? icons.Loading : icons.Microphone"
                    /></el-icon>
                  </button>
                </el-tooltip>
                <el-tooltip :content="t('bi.query.ask')" placement="top">
                  <button
                    class="bi-query-compose__button bi-query-compose__button--send"
                    type="button"
                    :disabled="loading || loadingModels || recording || !question.trim()"
                    :aria-label="t('bi.query.ask')"
                    @click="ask"
                  >
                    <el-icon :class="{ 'is-loading': loading }"
                      ><component :is="loading ? icons.Loading : icons.Promotion"
                    /></el-icon>
                  </button>
                </el-tooltip>
              </div>
            </div>
          </div>
        </div>
      </section>
      <aside class="bi-query-preview">
        <header>
          <h3>{{ t("bi.card.preview") }}</h3>
        </header>
        <BiDataCard
          v-if="response?.result && response.query"
          :card="previewCard"
          :preview-result="response.result"
          :source-name="sources.find(item => item.code === previewCard.source_code)?.name || previewCard.source_code"
          ><template #footer
            ><el-button link type="primary" @click="cardDialog = true">{{ t("bi.card.edit") }}</el-button
            ><el-button type="primary" @click="cardDialog = true">{{ t("bi.card.save_result") }}</el-button></template
          ></BiDataCard
        ><el-empty v-else :description="t('bi.card.preview_empty')" />
      </aside>
    </div>
    <BiCardFormDialog
      v-model="cardDialog"
      :sources="sources"
      :initial-query="response?.query"
      :initial-result="response?.result"
      :initial-title="response?.title"
      :initial-chart-type="previewCard.chart_type"
      @saved="loadHistory"
    />
  </main>
</template>
<script setup lang="ts">
import { computed, onBeforeUnmount, onDeactivated, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { t, useSpeechRecognition } from "@liujitcn/kratos-admin-core";
import AiModelSelector from "@liujitcn/kratos-admin-system/components/AiModelSelector/index.vue";
import {
  resolveModelSelection,
  saveModelSelection
} from "@liujitcn/kratos-admin-system/components/AiModelSelector/modelSelection";
import type { AiModelSelection } from "@liujitcn/kratos-admin-system/components/AiModelSelector/modelSelection";
import { defAiModelService } from "@liujitcn/kratos-admin-system/api/base/v1/ai_provider";
import type { AiProviderModelOption } from "@liujitcn/kratos-admin-system/rpc/base/v1/ai_provider";
import SessionPanel from "@liujitcn/kratos-admin-system/components/AiSessionPanel/index.vue";
import BiDataCard from "../../components/BiDataCard.vue";
import BiCardFormDialog from "../../components/BiCardFormDialog.vue";
import { normalizeBiQueryResult, normalizeBiQuerySpec } from "../../bi-response";
import { defBiQueryService } from "../../api/bi/admin/v1/bi_query";
import { defBiQueryRecordService } from "../../api/bi/admin/v1/bi_query_record";
import type { BiQueryRecord } from "../../rpc/bi/admin/v1/bi_query_record";
import type { AskBiQueryResponse, BiQuerySpec, BiSourceDescriptor } from "../../rpc/bi/admin/v1/bi_query";
import type { BiCard } from "../../rpc/bi/admin/v1/bi_card";
defineOptions({ name: "BiQuery", inheritAttrs: false });
const route = useRoute(),
  router = useRouter();
const icons = { Promotion, Microphone, Loading };
const question = ref(""),
  submittedQuestion = ref(""),
  loading = ref(false),
  error = ref(""),
  sources = ref<BiSourceDescriptor[]>([]),
  response = ref<AskBiQueryResponse>(),
  records = ref<BiQueryRecord[]>([]),
  keyword = ref(""),
  activeRecord = ref(""),
  historyCollapsed = ref(false),
  cardDialog = ref(false),
  showTable = ref(false);
const modelProviders = ref<AiProviderModelOption[]>([]),
  selectedModel = ref<AiModelSelection>({ provider_id: 0, model_name: "" }),
  loadingModels = ref(true);
let request = 0,
  historyRequest = 0,
  voiceBaseText = "";
let searchTimer: ReturnType<typeof setTimeout> | undefined;
const sessions = computed(() =>
  records.value.map(item => ({
    id: String(item.id),
    title: item.question,
    summary: `${sources.value.find(source => source.code === item.source_code)?.name || item.source_code || ""} · ${item.created_at || ""}`
  }))
);
const {
  loading: recording,
  start: startRecord,
  stop: stopRecord
} = useSpeechRecognition({
  onStart: () => {
    voiceBaseText = question.value;
  },
  onResult: text => {
    question.value = [voiceBaseText, text].filter(Boolean).join(" ");
  },
  onError: cause => {
    const key =
      cause.code === -1
        ? "bi.query.voice_unsupported"
        : ["not-allowed", "service-not-allowed", "permission-denied"].includes(cause.error ?? "")
          ? "bi.query.microphone_denied"
          : "bi.query.voice_failed";
    ElMessage.warning(t(key));
  }
});

const previewCard = computed<BiCard>(() => ({
  id: 0,
  tenant_id: 0,
  project_id: 0,
  owner_id: 0,
  name: response.value?.title || "BI",
  source_code: response.value?.query?.source_code || "",
  query: response.value?.query,
  chart_type: response.value?.query?.dimensions.length ? response.value.chart_type || "bar" : "metric",
  chart_json: "",
  query_record_id: response.value?.result?.query_record_id || 0,
  refresh_seconds: 0,
  is_shared: false,
  updated_at: "",
  category: "mine",
  can_edit: true
}));
const tableRows = computed(() =>
  (response.value?.result?.rows || []).map(row => Object.fromEntries(row.values.map((value, index) => [String(index), value])))
);
/** Enter发送、Shift+Enter换行，输入法选字不提交。 */
function handleInputEnter(event: KeyboardEvent) {
  if (event.shiftKey || event.isComposing || event.keyCode === 229) return;
  event.preventDefault();
  void ask();
}
/** 发起真实语义问数并关联服务端生成的结果记录。 */
async function ask() {
  if (!question.value.trim() || loading.value || loadingModels.value || recording.value) return;
  const current = ++request;
  submittedQuestion.value = question.value;
  question.value = "";
  response.value = undefined;
  error.value = "";
  loading.value = true;
  showTable.value = false;
  try {
    const result = await defBiQueryService.AskBiQuery({ question: submittedQuestion.value, ...selectedModel.value });
    if (current !== request) return;
    response.value = result;
    activeRecord.value = String(result.result?.query_record_id || "");
    await loadHistory();
  } catch (cause) {
    if (current === request) error.value = resolveAskError(cause);
  } finally {
    if (current === request) loading.value = false;
  }
}
/** 左侧历史仅展示当前用户的自然语言问数，搜索仍由服务端过滤。 */
async function loadHistory() {
  const current = ++historyRequest;
  try {
    const result = await defBiQueryRecordService.PageBiQuery({
      question: keyword.value,
      status: undefined,
      source_code: "",
      page_num: 1,
      page_size: 100
    });
    if (current === historyRequest) records.value = result.queries || [];
  } catch (cause) {
    ElMessage.error(resolveAskError(cause));
  }
}
/** 恢复历史问题和完整类型化结果，切换记录时丢弃旧请求。 */
async function selectRecord(id: number) {
  if (!Number.isSafeInteger(id) || id <= 0) return;
  const current = ++request;
  stopRecord();
  loading.value = true;
  error.value = "";
  response.value = undefined;
  activeRecord.value = String(id);
  showTable.value = false;
  try {
    const item = await defBiQueryRecordService.GetBiQuery({ id });
    if (current !== request) return;
    submittedQuestion.value = item.question;
    if (!item.result) {
      error.value = item.error_msg || t("bi.query.error");
      return;
    }
    const spec = normalizeBiQuerySpec(JSON.parse(item.query_json) as BiQuerySpec);
    response.value = {
      query: spec,
      result: normalizeBiQueryResult(item.result),
      title: item.question || t("bi.query.result"),
      chart_type: spec.dimensions.length ? "bar" : "metric"
    };
  } catch (cause) {
    if (current === request) error.value = resolveAskError(cause);
  } finally {
    if (current === request) loading.value = false;
  }
}
/** 新问题只清空当前输入与预览，不删除已有记录。 */
function newQuestion() {
  request++;
  stopRecord();
  loading.value = false;
  submittedQuestion.value = "";
  question.value = "";
  response.value = undefined;
  activeRecord.value = "";
  error.value = "";
}
/** 加载与 AI 助手一致的模型选项，并恢复仍可用的本地选择。 */
async function loadModels() {
  loadingModels.value = true;
  try {
    const result = await defAiModelService.ListAiProviderModelOptions({});
    modelProviders.value = result.providers ?? [];
    selectedModel.value = resolveModelSelection(modelProviders.value);
  } catch (cause) {
    error.value = resolveAskError(cause);
  } finally {
    loadingModels.value = false;
  }
}

/** 更新本次问数的模型，同时保存为 AI 助手和 BI 共用的本地选择。 */
function handleModelChange(providerId: number, modelName: string) {
  selectedModel.value = { provider_id: providerId, model_name: modelName };
  saveModelSelection(selectedModel.value);
}

/** 优先展示服务端返回的业务错误信息，请求失败时回退到通用错误文案。 */
function resolveAskError(cause: unknown) {
  const responseData = (cause as { response?: { data?: { message?: string } } })?.response?.data;
  if (typeof responseData?.message === "string" && responseData.message) return responseData.message;
  return cause instanceof Error ? cause.message : t("bi.query.error");
}

watch(keyword, () => {
  clearTimeout(searchTimer);
  searchTimer = setTimeout(() => void loadHistory(), 250);
});
watch(
  () => route.query.record_id,
  value => {
    if (value) void selectRecord(Number(value));
  },
  { immediate: true }
);
onMounted(async () => {
  await Promise.all([loadHistory(), loadModels()]);
  try {
    sources.value = (await defBiQueryService.ListBiSources({})).sources;
  } catch (cause) {
    error.value = resolveAskError(cause);
  }
});
onDeactivated(() => {
  request++;
  loading.value = false;
  stopRecord();
});
onBeforeUnmount(() => {
  request++;
  historyRequest++;
  clearTimeout(searchTimer);
  stopRecord();
});
</script>
<style scoped>
.bi-query-page {
  padding: 20px 22px;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius);
  min-width: 0;
}
.bi-query-page__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}
.bi-query-page__header h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 500;
}
.bi-query-page__header p {
  margin: 8px 0 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.bi-query-page__header > div:last-child {
  display: flex;
  gap: 8px;
}
.bi-query-layout {
  display: grid;
  grid-template-columns: 260px minmax(0, 7fr) minmax(0, 3fr);
  height: calc(100vh - 250px);
  min-height: 580px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius);
}
.bi-query-layout.is-history-collapsed {
  grid-template-columns: minmax(0, 7fr) minmax(0, 3fr);
}
.bi-query-chat {
  display: flex;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  padding: 18px;
}
.bi-query-messages {
  flex: 1;
  min-height: 0;
  overflow: auto;
  font-size: 14px;
}
.bi-query-bubble {
  padding: 12px 15px;
  border-radius: var(--admin-page-radius);
  background: var(--el-fill-color-light);
  margin-bottom: 22px;
}
.bi-query-assistant {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.bi-query-messages h3 {
  font-size: 16px;
  font-weight: 500;
}
.bi-query-facts {
  border-left: 2px solid var(--el-border-color);
  padding-left: 12px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  margin: 16px 0;
}
.bi-query-facts p {
  margin: 7px 0;
}
.bi-query-preview {
  padding: 18px 14px;
  border-left: 1px solid var(--el-border-color-lighter);
  background: var(--el-fill-color-lighter);
  overflow: auto;
  min-width: 0;
}
.bi-query-preview h3 {
  font-size: 14px;
  font-weight: 500;
  margin: 0 0 18px;
}
.bi-query-compose {
  flex-shrink: 0;
  margin-top: auto;
  padding-top: 24px;
}
.bi-query-compose__input {
  padding: 8px 10px 10px;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius);
  transition: border-color 0.2s ease;
}
.bi-query-compose__input:focus-within {
  border-color: var(--el-color-primary);
}
.bi-query-compose__input :deep(.el-textarea__inner) {
  min-height: 64px;
  padding: 8px 10px;
  font-size: 14px;
  line-height: 22px;
  resize: none;
  background: transparent;
  box-shadow: none;
}
.bi-query-compose__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 10px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.bi-query-compose__actions {
  display: inline-flex;
  flex: 0 1 auto;
  min-width: 0;
  align-items: center;
  gap: 8px;
}
.bi-query-compose__footer > span {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.bi-query-compose__button {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  color: var(--admin-page-text-secondary);
  cursor: pointer;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius);
  transition:
    color 0.2s ease,
    border-color 0.2s ease,
    background-color 0.2s ease,
    opacity 0.2s ease;
}
.bi-query-compose__button:hover,
.bi-query-compose__button.is-active,
.bi-query-compose__button--send {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary-light-5);
}
.bi-query-compose__button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

@media (max-width: 1180px) {
  .bi-query-layout {
    grid-template-columns: 240px minmax(0, 1fr);
    height: auto;
  }
  .bi-query-layout.is-history-collapsed {
    grid-template-columns: 1fr;
  }
  .bi-query-layout :deep(.agent-session-panel) {
    display: flex;
    max-height: 560px;
  }
  .bi-query-chat {
    min-height: 500px;
  }
  .bi-query-preview {
    grid-column: 1/-1;
    border-left: 0;
    border-top: 1px solid var(--el-border-color-light);
  }
}
@media (max-width: 650px) {
  .bi-query-page {
    padding: 14px;
  }
  .bi-query-layout {
    grid-template-columns: 1fr;
  }
  .bi-query-layout :deep(.agent-session-panel) {
    max-height: 220px;
    border-right: 0;
    border-bottom: 1px solid var(--el-border-color-light);
  }
  .bi-query-chat {
    padding: 14px;
    min-height: 450px;
  }
  .bi-query-compose__actions {
    min-width: 0;
    flex-shrink: 1;
  }
  .bi-query-compose__button {
    flex-shrink: 0;
  }
}
</style>
