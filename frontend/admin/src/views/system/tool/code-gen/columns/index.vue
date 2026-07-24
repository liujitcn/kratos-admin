<!-- 代码生成字段配置 -->
<template>
  <div v-loading="loading" class="app-container code-gen-sub-page">
    <el-card class="code-gen-sub-card" shadow="never">
      <div v-if="formData.id" class="code-gen-column-pane">
        <div class="code-gen-toolbar code-gen-column-toolbar">
          <div class="code-gen-column-toolbar__meta">
            <strong>数据库字段</strong>
            <!-- 展示当前字段配置对应的业务表。 -->
            <span class="code-gen-column-toolbar__table-name" :title="formData.name">表名：{{ formData.name }}</span>
            <span class="code-gen-column-toolbar__table-comment" :title="formData.comment || '--'">
              表注释：{{ formData.comment || "--" }}
            </span>
            <span>查询 {{ enabledSummary.query }} 项</span>
            <span>列表 {{ enabledSummary.list }} 项</span>
            <span>表单 {{ enabledSummary.form }} 项</span>
          </div>
          <div class="code-gen-column-toolbar__actions">
            <el-button type="primary" :icon="Document" :disabled="!canEdit" @click="handleSaveColumns()">保存</el-button>
          </div>
        </div>

        <div ref="columnTableRef" class="code-gen-column-table">
          <el-table :data="columns" row-key="name" border stripe table-layout="fixed" empty-text="暂无字段配置">
          <el-table-column label="数据库字段" min-width="320" fixed="left">
            <template #default="{ row, $index }">
              <div class="code-gen-field-cell">
                <el-popover trigger="hover" placement="right-start" :width="320" :show-after="250">
                  <template #reference>
                    <div class="code-gen-field-trigger">
                      <span class="code-gen-field-trigger__name">{{ row.name }}</span>
                    </div>
                  </template>
                  <div class="code-gen-field-popover">
                    <div class="code-gen-field-popover__header">
                      <strong>{{ row.name }}</strong>
                      <span>{{ row.comment || row.name }}</span>
                    </div>
                    <div class="code-gen-field-popover__types">
                      <div>
                        <span>数据库</span><b>{{ row.db_type || "--" }}</b>
                      </div>
                      <div>
                        <span>Go</span><b>{{ row.go_type || "--" }}</b>
                      </div>
                      <div>
                        <span>Proto</span><b>{{ row.proto_type || "--" }}</b>
                      </div>
                      <div>
                        <span>TS</span><b>{{ row.ts_type || "--" }}</b>
                      </div>
                    </div>
                    <div class="code-gen-field-popover__flags">
                      <el-tag v-if="row.is_primary" size="small" type="danger" effect="plain">主键</el-tag>
                      <el-tag v-if="row.is_auto_increment" size="small" type="warning" effect="plain">自增</el-tag>
                      <el-tag size="small" :type="row.is_nullable ? 'info' : 'success'" effect="plain">
                        {{ row.is_nullable ? "可空" : "必填" }}
                      </el-tag>
                    </div>
                  </div>
                </el-popover>
                <div class="code-gen-field-order">
                  <span class="code-gen-field-order__index">{{ $index + 1 }}</span>
                  <el-tooltip content="拖拽排序" placement="top">
                    <el-button
                      text
                      size="small"
                      :icon="List"
                      :disabled="!canEdit"
                      class="code-gen-field-order__drag"
                      aria-label="拖拽调整字段顺序"
                    />
                  </el-tooltip>
                </div>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="字段描述" min-width="280">
            <template #default="{ row }">
              <el-input v-model="row.comment" :disabled="!canEdit" maxlength="255" placeholder="请输入字段描述" />
            </template>
          </el-table-column>

          <el-table-column label="查询" min-width="440">
            <template #default="{ row }">
              <div class="code-gen-config-cell">
                <el-switch
                  v-model="row.query_config.enabled"
                  :disabled="!canEdit"
                  inline-prompt
                  active-text="开"
                  inactive-text="关"
                />
                <el-select
                  v-model="row.query_config.operator"
                  :disabled="!canEdit || !row.query_config.enabled"
                  placeholder="查询方式"
                >
                  <el-option
                    v-for="item in queryOperatorOptions"
                    :key="String(item.value)"
                    :label="item.label"
                    :value="item.value"
                  />
                </el-select>
                <el-select
                  v-model="row.query_config.component"
                  :disabled="!canEdit || !row.query_config.enabled"
                  placeholder="查询组件"
                  @change="handleComponentChange(row, 'query')"
                >
                  <el-option
                    v-for="item in queryComponentOptions"
                    :key="String(item.value)"
                    :label="item.label"
                    :value="item.value"
                  />
                </el-select>
                <el-tooltip
                  v-if="shouldShowOptionEntry(row.query_config, 'query')"
                  :content="optionEntryTip(row.query_config.option)"
                  placement="top"
                >
                  <el-button
                    size="small"
                    :type="hasOptionConfig(row.query_config.option) ? 'primary' : 'default'"
                    :icon="Setting"
                    :disabled="!canEdit"
                    @click="openOptionDialog(row, 'query')"
                  >
                    选项
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="列表" min-width="420">
            <template #default="{ row }">
              <div class="code-gen-config-cell">
                <el-switch
                  v-model="row.list_config.enabled"
                  :disabled="!canEdit"
                  inline-prompt
                  active-text="开"
                  inactive-text="关"
                />
                <el-select
                  v-model="row.list_config.component"
                  :disabled="!canEdit || !row.list_config.enabled"
                  placeholder="展示组件"
                  @change="handleComponentChange(row, 'list')"
                >
                  <el-option
                    v-for="item in listComponentOptions"
                    :key="String(item.value)"
                    :label="item.label"
                    :value="item.value"
                  />
                </el-select>
                <el-tooltip
                  v-if="shouldShowOptionEntry(row.list_config, 'list')"
                  :content="optionEntryTip(row.list_config.option)"
                  placement="top"
                >
                  <el-button
                    size="small"
                    :type="hasOptionConfig(row.list_config.option) ? 'primary' : 'default'"
                    :icon="Setting"
                    :disabled="!canEdit"
                    @click="openOptionDialog(row, 'list')"
                  >
                    选项
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>

          <el-table-column label="表单" min-width="440">
            <template #default="{ row }">
              <div class="code-gen-config-cell">
                <el-switch
                  v-model="row.form_config.enabled"
                  :disabled="!canEdit"
                  inline-prompt
                  active-text="开"
                  inactive-text="关"
                  @change="handleFormEnabledChange(row)"
                />
                <el-select
                  v-model="row.form_config.component"
                  :disabled="!canEdit || !row.form_config.enabled"
                  placeholder="录入组件"
                  @change="handleComponentChange(row, 'form')"
                >
                  <el-option
                    v-for="item in formComponentOptions"
                    :key="String(item.value)"
                    :label="item.label"
                    :value="item.value"
                  />
                </el-select>
                <el-checkbox v-model="row.form_config.required" :disabled="!canEdit || !row.form_config.enabled"
                  >必填</el-checkbox
                >
                <el-tooltip
                  v-if="shouldShowOptionEntry(row.form_config, 'form')"
                  :content="optionEntryTip(row.form_config.option)"
                  placement="top"
                >
                  <el-button
                    size="small"
                    :type="hasOptionConfig(row.form_config.option) ? 'primary' : 'default'"
                    :icon="Setting"
                    :disabled="!canEdit"
                    @click="openOptionDialog(row, 'form')"
                  >
                    选项
                  </el-button>
                </el-tooltip>
              </div>
            </template>
          </el-table-column>
          </el-table>
        </div>
      </div>
      <el-empty v-else description="请先选择生成记录" />
    </el-card>

    <ProDialog
      v-model="optionDialog.visible"
      :title="`${optionDialog.scopeLabel}选项 - ${optionDialog.columnName}`"
      width="min(560px, calc(100vw - 32px))"
      destroy-on-close
      :show-close="false"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
      :show-footer="false"
      @closed="handleOptionDialogClosed"
    >
      <template #header="{ titleId, titleClass }">
        <div class="code-gen-config-dialog__header">
          <span :id="titleId" :class="titleClass">{{ `${optionDialog.scopeLabel}选项 - ${optionDialog.columnName}` }}</span>
          <el-button
            type="primary"
            :icon="Document"
            :disabled="!canEdit"
            aria-label="保存并关闭"
            @click="handleSaveOptionDialog"
          >
            保存
          </el-button>
        </div>
      </template>

      <div v-if="optionDialog.option" class="code-gen-option-dialog">
        <div v-if="optionDialog.formConfig" class="code-gen-popover-form__row">
          <span class="code-gen-popover-form__label">选择模式</span>
          <el-radio-group v-model="optionDialog.formConfig.multiple" :disabled="!canEdit">
            <el-radio-button :value="false">单选</el-radio-button>
            <el-tooltip content="多选仅支持 JSON 字段" :disabled="optionDialog.isJSONColumn" placement="top">
              <el-radio-button :value="true" :disabled="!optionDialog.isJSONColumn">多选</el-radio-button>
            </el-tooltip>
          </el-radio-group>
        </div>
        <div
          v-if="
            !['tree', 'switch'].includes(optionDialog.option.kind) &&
            !(optionDialog.scope === 'form' && optionDialog.component === 'dict')
          "
          class="code-gen-popover-form__row"
        >
          <span class="code-gen-popover-form__label">来源</span>
          <el-select
            v-model="optionDialog.option.source_type"
            :disabled="!canEdit"
            clearable
            placeholder="选择来源"
            @change="handleOptionSourceTypeChange"
          >
            <el-option
              v-for="item in sourceTypeOptions"
              :key="String(item.value)"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </div>
        <div v-if="optionDialog.option.source_type === 'dict'" class="code-gen-popover-form__row">
          <span class="code-gen-popover-form__label">字典</span>
          <el-select
            v-model="optionDialog.option.source_value"
            :disabled="!canEdit"
            :loading="loadingDictionaries"
            filterable
            clearable
            placeholder="选择字典"
            @change="handleOptionSourceValueChange"
          >
            <el-option
              v-for="item in dictionaries"
              :key="item.code"
              :label="item.name ? `${item.name}（${item.code}）` : item.code"
              :value="item.code"
            />
          </el-select>
        </div>
        <template v-if="optionDialog.option.kind === 'switch' && optionDialog.option.source_value">
          <div class="code-gen-popover-form__row">
            <span class="code-gen-popover-form__label">开启值</span>
            <el-select
              v-model="optionDialog.option.active_value"
              :disabled="!canEdit"
              filterable
              placeholder="选择开启值"
            >
              <el-option
                v-for="item in dictionaryItemsForEditor"
                :key="item.value"
                :label="`${item.label}（${item.value}）`"
                :value="item.value"
                :disabled="item.value === optionDialog.option.inactive_value"
              />
            </el-select>
          </div>
          <div class="code-gen-popover-form__row">
            <span class="code-gen-popover-form__label">关闭值</span>
            <el-select
              v-model="optionDialog.option.inactive_value"
              :disabled="!canEdit"
              filterable
              placeholder="选择关闭值"
            >
              <el-option
                v-for="item in dictionaryItemsForEditor"
                :key="item.value"
                :label="`${item.label}（${item.value}）`"
                :value="item.value"
                :disabled="item.value === optionDialog.option.active_value"
              />
            </el-select>
          </div>
        </template>
        <div v-else-if="optionDialog.option.source_type === 'table'" class="code-gen-popover-form__row">
          <span class="code-gen-popover-form__label">数据表</span>
          <el-select
            v-model="optionDialog.option.source_value"
            :disabled="!canEdit"
            :loading="loadingDatabaseTables"
            filterable
            clearable
            placeholder="选择数据表"
            @change="handleOptionSourceValueChange"
          >
            <el-option
              v-for="item in databaseTables"
              :key="item.name"
              :label="item.comment ? `${item.name}（${item.comment}）` : item.name"
              :value="item.name"
            />
          </el-select>
        </div>
        <template v-if="optionDialog.option.source_type === 'table'">
          <div v-if="optionDialog.option.kind === 'tree'" class="code-gen-popover-form__row">
            <span class="code-gen-popover-form__label">树父字段</span>
            <el-select
              v-model="optionDialog.option.parent_field"
              :disabled="!canEdit || !optionDialog.option.source_value"
              :loading="loadingDatabaseColumns.has(optionDialog.option.source_value)"
              filterable
              clearable
              placeholder="选择树父字段"
            >
              <el-option
                v-for="item in databaseColumnsForEditor"
                :key="item.name"
                :label="formatDatabaseColumn(item)"
                :value="item.name"
              />
            </el-select>
          </div>
          <div v-if="optionDialog.option.kind === 'tree'" class="code-gen-popover-form__row">
            <span class="code-gen-popover-form__label">加载方式</span>
            <el-checkbox v-model="optionDialog.option.lazy" :disabled="!canEdit">
              懒加载子节点
            </el-checkbox>
          </div>
          <div class="code-gen-popover-form__row">
            <span class="code-gen-popover-form__label">
              {{ optionDialog.option.kind === "tree" ? "树显示字段" : "Label 字段" }}
            </span>
            <el-select
              v-model="optionDialog.option.label_field"
              :disabled="!canEdit || !optionDialog.option.source_value"
              :loading="loadingDatabaseColumns.has(optionDialog.option.source_value)"
              filterable
              clearable
              :placeholder="optionDialog.option.kind === 'tree' ? '选择树显示字段' : '选择显示字段'"
            >
              <el-option
                v-for="item in databaseColumnsForEditor"
                :key="item.name"
                :label="formatDatabaseColumn(item)"
                :value="item.name"
              />
            </el-select>
          </div>
          <div class="code-gen-popover-form__row">
            <span class="code-gen-popover-form__label">
              {{ optionDialog.option.kind === "tree" ? "树值字段" : "Value 字段" }}
            </span>
            <el-select
              v-model="optionDialog.option.value_field"
              :disabled="!canEdit || !optionDialog.option.source_value"
              :loading="loadingDatabaseColumns.has(optionDialog.option.source_value)"
              filterable
              clearable
              :placeholder="optionDialog.option.kind === 'tree' ? '选择树值字段' : '选择值字段'"
            >
              <el-option
                v-for="item in databaseColumnsForEditor"
                :key="item.name"
                :label="formatDatabaseColumn(item)"
                :value="item.name"
              />
            </el-select>
          </div>
        </template>
        <div v-if="optionDialog.option.source_type === 'static'" class="code-gen-static-options">
          <div class="code-gen-static-options__header">
            <span class="code-gen-popover-form__label">静态数据</span>
            <el-button size="small" :icon="Plus" :disabled="!canEdit" @click="addStaticOption">添加</el-button>
          </div>
          <div v-if="staticOptionsForEditor.length" class="code-gen-static-options__list">
            <div v-for="(item, index) in staticOptionsForEditor" :key="index" class="code-gen-static-options__item">
              <el-input v-model="item.label" :disabled="!canEdit" placeholder="Label" @input="syncStaticOptions" />
              <el-input
                :model-value="String(item.value)"
                :disabled="!canEdit"
                placeholder="Value"
                @update:model-value="updateStaticOptionValue(item, $event)"
              />
              <el-tooltip content="删除静态数据" placement="top">
                <el-button
                  :icon="Delete"
                  :disabled="!canEdit"
                  circle
                  text
                  aria-label="删除静态数据"
                  @click="removeStaticOption(index)"
                />
              </el-tooltip>
            </div>
          </div>
          <el-empty v-else :image-size="48" description="暂无静态数据" />
        </div>
      </div>
    </ProDialog>
  </div>
</template>

<script setup lang="ts">
import Sortable from "sortablejs";
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { Delete, Document, List, Plus, Setting } from "@element-plus/icons-vue";
import { useRoute, useRouter } from "vue-router";
import ProDialog from "@/components/Dialog/ProDialog.vue";
import { useAuthButtons } from "@/hooks/useAuthButtons";
import { useTabsStore } from "@/stores/modules/tabs";
import { defBaseDictService } from "@/api/system/base_dict";
import { defCodeGenColumnService } from "@/api/system/code_gen_column";
import { defCodeGenTableService } from "@/api/system/code_gen_table";
import type { OptionBaseDictResponse_BaseDict } from "@/rpc/system/admin/v1/base_dict";
import type {
  CodeGenColumn as CodeGenColumnDTO,
  CodeGenColumnFormConfig,
  CodeGenColumnListConfig,
  CodeGenColumnOptionConfig,
  CodeGenColumnQueryConfig,
  CodeGenDatabaseColumn
} from "@/rpc/system/admin/v1/code_gen_column";
import type { CodeGenDatabaseTable, CodeGenTableForm } from "@/rpc/system/admin/v1/code_gen_table";
import {
  copyCodeGenOptionToEmptyMatches,
  copyFirstMatchingCodeGenOption,
  fillMissingCodeGenOptionConfigs,
  getCodeGenOptionContainer,
  isCompleteCodeGenOptionConfig
} from "./option-copy";
import type { CodeGenOptionContainer, CodeGenOptionScope } from "./option-copy";
import {
  codeGenFormComponentOptions as formComponentOptions,
  codeGenListComponentOptions as listComponentOptions,
  codeGenQueryComponentOptions as queryComponentOptions,
  codeGenQueryOperatorOptions as queryOperatorOptions,
  codeGenSourceTypeOptions as sourceTypeOptions,
  createDefaultCodeGenFormConfig,
  createDefaultCodeGenListConfig,
  createDefaultCodeGenOptionConfig,
  createDefaultCodeGenQueryConfig,
  createDefaultCodeGenTableForm
} from "../config";

defineOptions({
  name: "CodeGenColumn",
  inheritAttrs: false
});

const route = useRoute();
const router = useRouter();
const tabsStore = useTabsStore();
const { BUTTONS } = useAuthButtons();

const loading = ref(false);
const columns = ref<CodeGenColumnView[]>([]);
const columnTableRef = ref<HTMLElement>();
const formData = reactive<CodeGenTableForm>(createDefaultCodeGenTableForm());
const dictionaries = ref<OptionBaseDictResponse_BaseDict[]>([]);
const databaseTables = ref<CodeGenDatabaseTable[]>([]);
const databaseColumns = reactive<Record<string, CodeGenDatabaseColumn[]>>({});
const staticOptions = reactive(new Map<string, CodeGenStaticOption[]>());
const loadingDictionaries = ref(false);
const loadingDatabaseTables = ref(false);
const loadingDatabaseColumns = reactive(new Set<string>());
let dictionariesLoaded = false;
let databaseTablesLoaded = false;
let columnSortable: Sortable | undefined;

/** 开关组件默认使用的状态字典和值。 */
const codeGenDefaultSwitchOption = {
  sourceValue: "status",
  activeValue: "1",
  inactiveValue: "2"
};

/** 字段配置页面使用的完整结构化编辑模型。 */
type CodeGenColumnView = Omit<CodeGenColumnDTO, "query_config" | "list_config" | "form_config"> & {
  query_config: CodeGenColumnQueryConfig & { option: CodeGenColumnOptionConfig };
  list_config: CodeGenColumnListConfig & { option: CodeGenColumnOptionConfig };
  form_config: CodeGenColumnFormConfig & { option: CodeGenColumnOptionConfig };
};

/** 静态选项编辑项，保留字符串、数字和布尔值类型。 */
interface CodeGenStaticOption {
  label: string;
  value: string | number | boolean;
}

/** 单个选项弹窗的编辑上下文。 */
interface CodeGenOptionDialog {
  visible: boolean;
  scope: CodeGenOptionScope;
  scopeLabel: string;
  columnName: string;
  component: string;
  isJSONColumn: boolean;
  cacheKey: string;
  option: CodeGenColumnOptionConfig | null;
  formConfig: CodeGenColumnView["form_config"] | null;
}

/** 保存前需要打开选项编辑器的字段配置问题。 */
interface CodeGenColumnOptionIssue {
  row: CodeGenColumnView;
  scope: CodeGenOptionScope;
  message: string;
}

/** 保存前需要确认选项来源不一致的字段配置问题。 */
interface CodeGenColumnOptionConsistencyIssue {
  columnName: string;
  scopes: string;
}

const optionDialog = reactive<CodeGenOptionDialog>({
  visible: false,
  scope: "query",
  scopeLabel: "查询",
  columnName: "",
  component: "",
  isJSONColumn: false,
  cacheKey: "",
  option: null,
  formConfig: null
});
/** 当前生成对象 ID。 */
const tableId = computed(() => {
  const value = route.params.tableId ?? route.query.tableId;
  const id = Number(Array.isArray(value) ? value[0] : value);
  return Number.isFinite(id) && id > 0 ? id : 0;
});

/** 是否可以维护字段配置。 */
const canEdit = computed(() => !!BUTTONS.value["tool:code-gen-table:column"]);

/** 已启用字段配置统计。 */
const enabledSummary = computed(() => ({
  query: columns.value.filter(item => item.query_config.enabled).length,
  list: columns.value.filter(item => item.list_config.enabled).length,
  form: columns.value.filter(item => item.form_config.enabled).length
}));

/** 当前选项编辑器可用的数据库字段。 */
const databaseColumnsForEditor = computed(() => {
  const tableName = optionDialog.option?.source_value ?? "";
  return databaseColumns[tableName] ?? [];
});

/** 当前开关选中字典内的可用字典项。 */
const dictionaryItemsForEditor = computed(
  () => dictionaries.value.find(item => item.code === optionDialog.option?.source_value)?.items ?? []
);

/** 当前选项编辑器中的静态数据。 */
const staticOptionsForEditor = computed(() => staticOptions.get(optionDialog.cacheKey) ?? []);

// 路由生成对象变化时重新加载字段配置。
watch(tableId, () => {
  void handleQuery();
});

// 权限数据加载完成后再同步拖拽能力，避免只读用户创建可拖拽实例。
watch(canEdit, () => {
  void nextTick(initColumnSortable);
});

/** 查询生成对象字段配置。 */
async function handleQuery() {
  loading.value = true;
  try {
    destroyColumnSortable();
    Object.assign(formData, createDefaultCodeGenTableForm());
    columns.value = [];
    staticOptions.clear();
    if (!tableId.value) return;
    const [table, response] = await Promise.all([
      defCodeGenTableService.GetCodeGenTable({ id: tableId.value }),
      defCodeGenColumnService.ListCodeGenColumn({ table_id: tableId.value })
    ]);
    Object.assign(formData, table);
    // 字段配置和预览都保留数据库完整字段快照，由用户决定是否调整默认配置。
    columns.value = (response.code_gen_columns ?? []).map(normalizeColumn);
    syncWorkspaceTitle();
    await nextTick();
    initColumnSortable();
  } finally {
    loading.value = false;
  }
}

/** 同步当前页签和浏览器标题。 */
function syncWorkspaceTitle() {
  const tableTitle = formData.comment || formData.name;
  const title = tableTitle ? `${tableTitle}字段配置` : "字段配置";
  tabsStore.setTabsTitle(title);
  document.title = `${title} - ${import.meta.env.VITE_GLOB_APP_TITLE}`;
}

/** 保存字段配置。 */
async function handleSaveColumns(showMessage = true) {
  if (!formData.id) return false;
  syncColumnSorts();
  columns.value.forEach(syncColumnOptionKinds);
  if (columns.value.some(item => !item.name || !item.db_type)) {
    ElMessage.warning("字段名和数据库类型不能为空");
    return false;
  }
  const optionIssue = findCodeGenColumnOptionIssue(columns.value);
  if (optionIssue) {
    ElMessage.warning(optionIssue.message);
    await openOptionDialog(optionIssue.row, optionIssue.scope);
    return false;
  }
  const consistencyIssue = findCodeGenColumnOptionConsistencyIssue(columns.value);
  if (consistencyIssue) {
    try {
      await ElMessageBox.confirm(
        `字段 ${consistencyIssue.columnName} 的${consistencyIssue.scopes}选项来源不一致，是否继续保存？`,
        "选项配置警告",
        {
          confirmButtonText: "继续保存",
          cancelButtonText: "返回修改",
          type: "warning"
        }
      );
    } catch {
      return false;
    }
  }
  await defCodeGenColumnService.SaveCodeGenColumn({
    table_id: formData.id,
    code_gen_columns: columns.value.map((item, index) => ({
      ...item,
      table_id: formData.id,
      sort: index + 1
    }))
  });
  if (showMessage) ElMessage.success("保存成功");
  // 路由切换前保留当前页签地址，避免误删目标页签。
  const currentPath = route.fullPath;
  await router.push("/code/gen/table");
  await tabsStore.removeTabs(currentPath, false);
  return true;
}

/** 初始化字段表格拖拽排序，并限制仅通过排序手柄触发。 */
function initColumnSortable() {
  destroyColumnSortable();
  if (!canEdit.value) return;
  const tbody =
    columnTableRef.value?.querySelector<HTMLElement>(".el-table__fixed-body-wrapper tbody") ??
    columnTableRef.value?.querySelector<HTMLElement>(".el-table__body-wrapper tbody");
  if (!tbody) return;
  columnSortable = Sortable.create(tbody, {
    handle: ".code-gen-field-order__drag",
    animation: 150,
    ghostClass: "code-gen-column-sortable-ghost",
    chosenClass: "code-gen-column-sortable-chosen",
    onEnd({ newIndex, oldIndex }) {
      if (oldIndex === undefined || newIndex === undefined || oldIndex === newIndex) return;
      const column = columns.value.splice(oldIndex, 1)[0];
      if (!column) return;
      columns.value.splice(newIndex, 0, column);
      syncColumnSorts();
    }
  });
}

/** 销毁字段表格拖拽实例，避免路由切换后保留 DOM 事件。 */
function destroyColumnSortable() {
  columnSortable?.destroy();
  columnSortable = undefined;
}

/** 按当前字段表格行顺序更新持久化排序值。 */
function syncColumnSorts() {
  columns.value.forEach((item, index) => {
    item.sort = index + 1;
  });
}

/** 将接口字段配置补齐为三份互不共享的选项对象。 */
function normalizeColumn(column: CodeGenColumnDTO): CodeGenColumnView {
  const query = column.query_config ?? createDefaultCodeGenQueryConfig();
  const list = column.list_config ?? createDefaultCodeGenListConfig();
  const form = column.form_config ?? createDefaultCodeGenFormConfig();
  const normalizedColumn: CodeGenColumnView = {
    ...column,
    query_config: {
      ...query,
      option: { ...(query.option ?? createDefaultCodeGenOptionConfig()) }
    },
    list_config: {
      ...list,
      option: { ...(list.option ?? createDefaultCodeGenOptionConfig()) }
    },
    form_config: {
      ...form,
      option: { ...(form.option ?? createDefaultCodeGenOptionConfig()) }
    }
  };
  syncColumnOptionKinds(normalizedColumn);
  fillMissingCodeGenOptionConfigs(normalizedColumn);
  return normalizedColumn;
}

/** 组件变化时清空旧类型配置，并从相同组件范围重新复刻。 */
function handleComponentChange(row: CodeGenColumnView, scope: CodeGenOptionScope) {
  const config = getCodeGenOptionContainer(row, scope);
  Object.assign(config.option, createDefaultCodeGenOptionConfig());
  syncOptionKind(config, scope);
  if (scope === "form" && row.form_config.component !== "tree-select") row.form_config.multiple = false;
  copyFirstMatchingCodeGenOption(row, scope);
}

/** 关闭表单展示时同步关闭必填约束。 */
function handleFormEnabledChange(row: CodeGenColumnView) {
  if (!row.form_config.enabled) {
    row.form_config.required = false;
    row.form_config.multiple = false;
  }
}

/** 打开查询、列表或表单自己的选项编辑弹窗。 */
async function openOptionDialog(row: CodeGenColumnView, scope: CodeGenOptionScope) {
  const config = getCodeGenOptionContainer(row, scope);
  syncOptionKind(config, scope);
  optionDialog.scope = scope;
  optionDialog.scopeLabel = scope === "query" ? "查询" : scope === "list" ? "列表" : "表单";
  optionDialog.columnName = row.name;
  optionDialog.component = config.component;
  optionDialog.isJSONColumn = row.db_type.trim().toLowerCase() === "json";
  optionDialog.cacheKey = `${row.table_id}:${row.name}:${scope}`;
  optionDialog.option = config.option;
  optionDialog.formConfig = scope === "form" && config.component === "tree-select" ? row.form_config : null;
  optionDialog.visible = true;
  await prepareOptionEditor();
}

/** 保存选项配置并关闭弹窗。 */
function handleSaveOptionDialog() {
  const row = columns.value.find(item => item.name === optionDialog.columnName);
  if (row) copyCodeGenOptionToEmptyMatches(row, optionDialog.scope);
  optionDialog.visible = false;
}

/** 清理选项配置弹窗上下文。 */
function handleOptionDialogClosed() {
  optionDialog.option = null;
  optionDialog.formConfig = null;
}

/** 按当前选项来源准备弹窗所需数据。 */
async function prepareOptionEditor() {
  const option = optionDialog.option;
  if (!option) return;
  if (option.source_type === "static") {
    if (!staticOptions.has(optionDialog.cacheKey)) {
      staticOptions.set(optionDialog.cacheKey, parseCodeGenStaticOptions(option.source_value));
    }
    return;
  }
  if (option.source_type === "dict") {
    option.label_field = "label";
    option.value_field = "value";
    await loadDictionaries();
    return;
  }
  if (option.source_type === "table") {
    await loadDatabaseTables();
    await loadDatabaseColumns(option.source_value);
    applyTableOptionDefaultFields(option, optionDialog.component);
  }
}

/** 切换选项来源时清理当前范围的旧来源字段。 */
async function handleOptionSourceTypeChange() {
  const option = optionDialog.option;
  if (!option) return;
  option.source_value = "";
  option.label_field = "";
  option.value_field = "";
  option.parent_field = "";
  option.active_value = "";
  option.inactive_value = "";
  option.lazy = false;
  staticOptions.delete(optionDialog.cacheKey);
  if (option.source_type === "static") {
    staticOptions.set(optionDialog.cacheKey, []);
    option.source_value = serializeCodeGenStaticOptions([]);
    option.label_field = "label";
    option.value_field = "value";
    return;
  }
  if (option.source_type === "dict") {
    option.label_field = "label";
    option.value_field = "value";
    await loadDictionaries();
    return;
  }
  if (option.source_type === "table") await loadDatabaseTables();
}

/** 选择字典或数据表后同步当前范围的字段配置。 */
async function handleOptionSourceValueChange() {
  const option = optionDialog.option;
  if (!option) return;
  option.label_field = option.source_type === "dict" ? "label" : "";
  option.value_field = option.source_type === "dict" ? "value" : "";
  option.parent_field = "";
  option.active_value = "";
  option.inactive_value = "";
  option.lazy = false;
  if (option.source_type === "table") {
    await loadDatabaseColumns(option.source_value);
    applyTableOptionDefaultFields(option, optionDialog.component);
  }
}

/** 加载可用字典列表。 */
async function loadDictionaries() {
  if (dictionariesLoaded || loadingDictionaries.value) return;
  loadingDictionaries.value = true;
  try {
    const data = await defBaseDictService.OptionBaseDict({});
    dictionaries.value = data.base_dicts ?? [];
    dictionariesLoaded = true;
  } finally {
    loadingDictionaries.value = false;
  }
}

/** 加载可用数据库表列表。 */
async function loadDatabaseTables() {
  if (databaseTablesLoaded || loadingDatabaseTables.value) return;
  loadingDatabaseTables.value = true;
  try {
    const data = await defCodeGenTableService.ListCodeGenDatabaseTable({});
    databaseTables.value = data.tables ?? [];
    databaseTablesLoaded = true;
  } finally {
    loadingDatabaseTables.value = false;
  }
}

/** 按数据表加载字段并缓存。 */
async function loadDatabaseColumns(tableName: string) {
  if (!tableName || databaseColumns[tableName] || loadingDatabaseColumns.has(tableName)) return;
  loadingDatabaseColumns.add(tableName);
  try {
    const data = await defCodeGenColumnService.ListCodeGenDatabaseColumn({ table_name: tableName });
    databaseColumns[tableName] = data.columns ?? [];
  } finally {
    loadingDatabaseColumns.delete(tableName);
  }
}

/** 为下拉或树形选项补充所选数据表中存在的常用字段。 */
function applyTableOptionDefaultFields(option: CodeGenColumnOptionConfig, component: string) {
  if (!option.source_value) return;
  const columnNames = new Set((databaseColumns[option.source_value] ?? []).map(item => item.name));
  if (option.kind === "tree") {
    if (!option.parent_field && columnNames.has("parent_id")) option.parent_field = "parent_id";
    if (!option.label_field && columnNames.has("name")) option.label_field = "name";
    if (!option.value_field && columnNames.has("id")) option.value_field = "id";
    return;
  }
  if (component !== "select") return;
  if (!option.label_field && columnNames.has("name")) option.label_field = "name";
  if (!option.value_field && columnNames.has("id")) option.value_field = "id";
}

/** 格式化数据表字段选项。 */
function formatDatabaseColumn(column: CodeGenDatabaseColumn) {
  const columnType = column.column_type || column.db_type;
  return column.comment
    ? `${column.name}（${column.comment} / ${columnType}）`
    : `${column.name}（${columnType}）`;
}

/** 添加一条空白静态选项。 */
function addStaticOption() {
  const items = staticOptions.get(optionDialog.cacheKey) ?? [];
  items.push({ label: "", value: "" });
  staticOptions.set(optionDialog.cacheKey, items);
  syncStaticOptions();
}

/** 删除指定静态选项。 */
function removeStaticOption(index: number) {
  const items = staticOptions.get(optionDialog.cacheKey) ?? [];
  items.splice(index, 1);
  syncStaticOptions();
}

/** 保留已有静态值类型，并将无法按原类型解析的编辑值回退为字符串。 */
function updateStaticOptionValue(option: CodeGenStaticOption, value: string) {
  if (typeof option.value === "number") {
    const parsedValue = Number(value);
    option.value = value !== "" && Number.isFinite(parsedValue) ? parsedValue : value;
  } else if (typeof option.value === "boolean" && (value === "true" || value === "false")) {
    option.value = value === "true";
  } else {
    option.value = value;
  }
  syncStaticOptions();
}

/** 将当前范围的静态选项同步回字段配置。 */
function syncStaticOptions() {
  if (!optionDialog.option) return;
  optionDialog.option.source_value = serializeCodeGenStaticOptions(staticOptions.get(optionDialog.cacheKey) ?? []);
}

/** 解析静态选项 JSON，并保留数字和布尔值类型。 */
function parseCodeGenStaticOptions(value: string): CodeGenStaticOption[] {
  if (!value) return [];
  try {
    const items = JSON.parse(value) as unknown;
    if (!Array.isArray(items)) return [];
    return items.flatMap(item => {
      if (!item || typeof item !== "object") return [];
      const label = Reflect.get(item, "label");
      const optionValue = Reflect.get(item, "value");
      if (
        (typeof label !== "string" && typeof label !== "number") ||
        (typeof optionValue !== "string" && typeof optionValue !== "number" && typeof optionValue !== "boolean")
      ) {
        return [];
      }
      return [{ label: String(label), value: optionValue }];
    });
  } catch {
    return [];
  }
}

/** 将静态选项序列化到数据源值字段。 */
function serializeCodeGenStaticOptions(options: CodeGenStaticOption[]) {
  return JSON.stringify(options);
}

/** 返回保存前首个需要补齐来源配置的字段选项。 */
function findCodeGenColumnOptionIssue(items: CodeGenColumnView[]): CodeGenColumnOptionIssue | undefined {
  for (const column of items) {
    const optionConfigs: Array<[CodeGenOptionScope, CodeGenColumnOptionConfig]> = [
      ["query", column.query_config.option],
      ["list", column.list_config.option],
      ["form", column.form_config.option]
    ];
    for (const [scope, option] of optionConfigs) {
      const scopeLabel = scope === "query" ? "查询" : scope === "list" ? "列表" : "表单";
      const message = getCodeGenOptionValidationMessage(column.name, scopeLabel, option);
      if (message) return { row: column, scope, message };
    }
  }
}

/** 检查同一字段查询、列表和表单选项来源是否一致。 */
function findCodeGenColumnOptionConsistencyIssue(items: CodeGenColumnView[]): CodeGenColumnOptionConsistencyIssue | undefined {
  for (const column of items) {
    const entries = (
      [
        ["查询", column.query_config, "query"],
        ["列表", column.list_config, "list"],
        ["表单", column.form_config, "form"]
      ] as const
    ).filter(([, config, scope]) => config.enabled && hasOptionComponent(config.component, scope) && isCompleteCodeGenOptionConfig(config.option));
    if (entries.length < 2) continue;
    const firstSignature = codeGenOptionSourceSignature(entries[0][1].option);
    if (entries.every(([, config]) => codeGenOptionSourceSignature(config.option) === firstSignature)) continue;
    return {
      columnName: column.name,
      scopes: entries.map(([label]) => label).join("、")
    };
  }
}

/** 返回用于比较字段选项数据源的稳定签名。 */
function codeGenOptionSourceSignature(option: CodeGenColumnOptionConfig) {
  return JSON.stringify({
    source_type: option.source_type,
    source_value: option.source_value,
    label_field: option.label_field,
    value_field: option.value_field,
    parent_field: option.parent_field,
    lazy: option.lazy
  });
}

/** 返回单个范围内不完整选项配置的提示文案。 */
function getCodeGenOptionValidationMessage(columnName: string, scope: string, option: CodeGenColumnOptionConfig) {
  const hasSourceFields = !!(
    option.source_type ||
    option.source_value ||
    option.label_field ||
    option.value_field ||
    option.parent_field ||
    option.active_value ||
    option.inactive_value ||
    option.lazy
  );
  if (!option.kind) return hasSourceFields ? `字段 ${columnName} 的${scope}选项配置无对应组件` : "";
  if (option.kind === "switch") {
    if (option.source_type !== "dict" || !option.source_value || !option.active_value || !option.inactive_value) {
      return `字段 ${columnName} 的${scope}开关配置不完整`;
    }
    if (option.active_value === option.inactive_value) return `字段 ${columnName} 的${scope}开启值和关闭值不能相同`;
    return "";
  }
  if (option.active_value || option.inactive_value) return `字段 ${columnName} 的${scope}选项不能配置开关值`;
  if (option.lazy && option.kind !== "tree") return `字段 ${columnName} 的${scope}仅树形选项支持懒加载`;
  if (option.kind === "tree" && option.source_type !== "table") {
    return `字段 ${columnName} 的${scope}树形选项只能使用数据表来源`;
  }
  if (!new Set(["static", "dict", "table"]).has(option.source_type) || !option.source_value) {
    return `字段 ${columnName} 的${scope}选项来源配置不完整`;
  }
  if (option.source_type === "static") {
    const options = parseCodeGenStaticOptions(option.source_value);
    if (!options.length || options.some(item => item.label === "" || item.value === "")) {
      return `字段 ${columnName} 的${scope}选项至少需要一条完整的静态数据`;
    }
  }
  if (
    option.source_type === "table" &&
    (!option.label_field || !option.value_field || (option.kind === "tree" && !option.parent_field))
  ) {
    return `字段 ${columnName} 的${scope}数据表选项字段配置不完整`;
  }
  return "";
}

/** 判断当前配置是否需要展示选项入口。 */
function shouldShowOptionEntry(config: CodeGenOptionContainer, scope: CodeGenOptionScope) {
  return config.enabled && hasOptionComponent(config.component, scope);
}

/** 判断当前范围是否已经填写选项配置。 */
function hasOptionConfig(option: CodeGenColumnOptionConfig) {
  return isCompleteCodeGenOptionConfig(option);
}

/** 返回选项入口的当前状态提示。 */
function optionEntryTip(option: CodeGenColumnOptionConfig) {
  return hasOptionConfig(option) ? "已配置，可直接保存；需要覆盖时可打开修改" : "配置选项";
}

/** 判断组件是否依赖选择数据源。 */
function hasOptionComponent(component: string, scope: CodeGenOptionScope) {
  return (
    (scope !== "query" && component === "switch") ||
    ["segmented", "select", "dict", "radio-group", "checkbox-group", "tree-select", "transfer"].includes(component)
  );
}

/** 同步字段在查询、列表和表单范围内由组件决定的选项形态。 */
function syncColumnOptionKinds(column: CodeGenColumnView) {
  syncOptionKind(column.query_config, "query");
  syncOptionKind(column.list_config, "list");
  syncOptionKind(column.form_config, "form");
  if (!column.form_config.enabled || column.form_config.component !== "tree-select") column.form_config.multiple = false;
}

/** 根据当前组件自动确定选项形态，并移除不再适用的选项配置。 */
function syncOptionKind(config: CodeGenOptionContainer, scope: CodeGenOptionScope) {
  if (!config.enabled || !hasOptionComponent(config.component, scope)) {
    Object.assign(config.option, createDefaultCodeGenOptionConfig());
    return;
  }
  const kind =
    config.component === "tree-select"
      ? "tree"
      : scope !== "query" && config.component === "switch"
        ? "switch"
        : "option";
  if (kind === "tree" && config.option.source_type !== "table") {
    Object.assign(config.option, createDefaultCodeGenOptionConfig());
    config.option.source_type = "table";
  }
  if (kind === "switch") {
    if (config.option.source_type !== "dict") {
      Object.assign(config.option, createDefaultCodeGenOptionConfig());
      config.option.source_type = "dict";
    }
    config.option.label_field = "label";
    config.option.value_field = "value";
    if (!config.option.source_value) config.option.source_value = codeGenDefaultSwitchOption.sourceValue;
    if (!config.option.active_value) config.option.active_value = codeGenDefaultSwitchOption.activeValue;
    if (!config.option.inactive_value) config.option.inactive_value = codeGenDefaultSwitchOption.inactiveValue;
  }
  if (scope === "form" && config.component === "dict") {
    if (config.option.source_type !== "dict") {
      Object.assign(config.option, createDefaultCodeGenOptionConfig());
      config.option.source_type = "dict";
    }
    config.option.label_field = "label";
    config.option.value_field = "value";
  }
  config.option.kind = kind;
  if (config.option.kind !== "tree") {
    config.option.parent_field = "";
    config.option.lazy = false;
  }
  if (config.option.kind !== "switch") {
    config.option.active_value = "";
    config.option.inactive_value = "";
  }
}

onMounted(() => {
  syncWorkspaceTitle();
  void handleQuery();
});

onBeforeUnmount(() => {
  destroyColumnSortable();
});
</script>

<style scoped lang="scss">
.code-gen-sub-card {
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border);
  border-radius: var(--admin-page-radius);
  box-shadow: var(--admin-page-shadow);
}

:deep(.code-gen-toolbar) {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: flex-end;
  margin-bottom: 14px;
}

.code-gen-column-pane {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.code-gen-column-toolbar {
  justify-content: space-between;
  padding: 12px 14px;
  background: var(--admin-page-card-bg-soft);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: var(--admin-page-radius);
}

.code-gen-column-toolbar__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  min-width: 0;
  color: var(--admin-page-text-secondary);
}

.code-gen-column-toolbar__meta strong {
  font-size: 14px;
  color: var(--admin-page-text-primary);
}

.code-gen-column-toolbar__meta span {
  padding: 3px 8px;
  overflow: hidden;
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
  background: var(--admin-page-card-bg);
  border: 1px solid var(--admin-page-card-border-soft);
  border-radius: 999px;
}

.code-gen-column-toolbar__table-name {
  max-width: 220px;
}

.code-gen-column-toolbar__table-comment {
  max-width: 280px;
}

.code-gen-column-toolbar__actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
  margin-left: auto;
}

.code-gen-column-table {
  overflow: hidden;
  border-radius: var(--admin-page-radius);
}

:deep(.code-gen-column-table .el-table__header th) {
  color: var(--admin-page-text-secondary);
  background: var(--admin-page-card-bg-soft);
}

:deep(.code-gen-column-table .el-table__cell) {
  padding: 8px 0;
  vertical-align: middle;
}

.code-gen-field-cell,
.code-gen-field-trigger,
.code-gen-field-order {
  display: flex;
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.code-gen-field-cell {
  justify-content: space-between;
}

.code-gen-field-trigger {
  flex: 1;
  min-height: 28px;
  cursor: help;
}

.code-gen-field-order {
  gap: 4px;
  flex-shrink: 0;
}

.code-gen-field-order__index {
  min-width: 18px;
  font-size: 12px;
  color: var(--admin-page-text-secondary);
  text-align: right;
}

.code-gen-field-order__drag {
  width: 24px;
  height: 24px;
  padding: 0;
  cursor: grab;
}

.code-gen-field-order__drag:active {
  cursor: grabbing;
}

:deep(.code-gen-column-sortable-ghost > td) {
  background: var(--admin-page-card-bg-soft);
}

:deep(.code-gen-column-sortable-chosen > td) {
  background: var(--admin-page-card-bg-soft);
}

.code-gen-config-cell,
.code-gen-static-options__header,
.code-gen-static-options__item {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.code-gen-field-trigger__name {
  flex-shrink: 0;
  max-width: 135px;
  overflow: hidden;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  font-size: 13px;
  font-weight: 700;
  color: var(--admin-page-text-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.code-gen-field-popover {
  display: grid;
  gap: 12px;
}

.code-gen-field-popover__header {
  display: grid;
  gap: 3px;
}

.code-gen-field-popover__header strong,
.code-gen-field-popover__types b {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
  color: var(--admin-page-text-primary);
}

.code-gen-field-popover__header span {
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}

.code-gen-field-popover__types {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 7px 12px;
}

.code-gen-field-popover__types div {
  display: grid;
  grid-template-columns: 52px minmax(0, 1fr);
  gap: 6px;
  align-items: center;
  min-width: 0;
  font-size: 12px;
}

.code-gen-field-popover__types span {
  color: var(--admin-page-text-secondary);
}

.code-gen-field-popover__types b {
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.code-gen-field-popover__flags {
  display: flex;
  gap: 6px;
  align-items: center;
  flex-wrap: wrap;
}

.code-gen-config-cell .el-select {
  width: 124px;
}

.code-gen-config-cell .el-checkbox {
  margin-right: 0;
}

.code-gen-option-dialog,
.code-gen-static-options,
.code-gen-static-options__list {
  display: grid;
  gap: 10px;
}

.code-gen-config-dialog__header {
  display: flex;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.code-gen-popover-form__row {
  display: grid;
  grid-template-columns: 92px minmax(0, 1fr);
  gap: 10px;
  align-items: center;
}

.code-gen-popover-form__label {
  font-size: 12px;
  color: var(--admin-page-text-secondary);
}

.code-gen-static-options {
  padding-top: 2px;
}

.code-gen-static-options__header {
  justify-content: space-between;
}

.code-gen-static-options__item {
  flex-wrap: nowrap;
}

.code-gen-static-options__item .el-input {
  min-width: 0;
}

@media (width <= 900px) {
  .code-gen-column-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .code-gen-column-toolbar__actions {
    margin-left: 0;
  }
}
</style>
