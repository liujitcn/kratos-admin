<template>
  <el-dialog
    v-model="open"
    :title="source?.name || t('bi.source.model.title')"
    width="min(1680px, 98vw)"
    top="2vh"
    destroy-on-close
    class="bi-source-model-dialog"
  >
    <div v-loading="loading || saving" class="source-model">
      <el-steps :active="step" finish-status="success" simple>
        <el-step :title="t('bi.source.model.step.tables')" />
        <el-step :title="t('bi.source.model.step.relations')" />
        <el-step :title="t('bi.source.model.step.fields')" />
      </el-steps>

      <section v-if="step === 0" class="source-model__tables">
        <div class="source-model__table-toolbar">
          <el-input v-model="tableFilter" clearable :placeholder="t('bi.source.model.search_tables')" />
          <el-select :model-value="model.base_table" :placeholder="t('bi.source.model.field.base_table')" :disabled="model.relations.length > 0" @update:model-value="setBaseTable(String($event ?? ''))">
            <el-option v-for="table in selectedTables" :key="table.key" :label="schemaLabel(table)" :value="table.key" />
          </el-select>
        </div>
        <el-scrollbar class="source-model__table-list">
          <div v-for="table in filteredTables" :key="table.key" class="source-model__table-row">
            <el-checkbox :model-value="isSelected(table.key)" :disabled="!isSelected(table.key) && model.tables.length >= 8" @change="toggleTable(table, Boolean($event))" />
            <div class="source-model__table-info">
              <el-text tag="strong" :title="schemaLabel(table)">{{ schemaLabel(table) }}</el-text>
              <el-text tag="small">{{ table.schema_name }} · {{ table.columns.length }} {{ t('bi.source.model.fields_count') }}</el-text>
            </div>
            <div class="source-model__table-actions">
              <el-select
                v-if="isSelected(table.key)"
                :model-value="modelTable(table.key)?.tenant_column ?? ''"
                clearable
                size="small"
                class="source-model__tenant-select"
                :placeholder="t('bi.source.model.field.shared_table')"
                @update:model-value="setTenantColumn(table.key, String($event ?? ''))"
              >
                <el-option v-for="column in table.columns" :key="column.name" :label="schemaLabel(column)" :value="column.name" />
              </el-select>
              <el-tag size="small" effect="plain">{{ table.table_type }}</el-tag>
              <el-tooltip :content="t('bi.source.model.action.preview_table')" placement="top">
                <el-button text :icon="View" :aria-label="t('bi.source.model.action.preview_table')" @click="openTableData(table)" />
              </el-tooltip>
            </div>
          </div>
          <el-empty v-if="!filteredTables.length" :description="t('bi.source.model.tables_empty')" />
        </el-scrollbar>
        <div class="source-model__selection-count">{{ selectedTables.length }} / 8</div>
      </section>

      <section v-else-if="step === 1" class="source-model__relations">
        <VueFlow
          :id="flowId"
          :nodes="nodes"
          :edges="edges"
          :nodes-connectable="selectedTables.length > 1"
          :edges-updatable="false"
          :min-zoom="0.15"
          :max-zoom="1.6"
          @connect="connectTables"
          @edge-click="selectRelation"
          @node-drag-stop="moveTable"
          @pane-click="selectedRelationIndex = -1"
          @init="fitGraph"
          @nodes-initialized="fitGraph"
        >
          <Background :gap="20" :size="1" pattern-color="var(--el-border-color-lighter)" />
          <Controls :show-interactive="false" />
          <MiniMap position="bottom-center" :width="150" :height="90" pannable zoomable node-color="var(--el-color-primary-light-5)" mask-color="var(--el-fill-color-light)" />
          <Panel position="top-right">
            <el-button size="small" @click="layoutGraph">{{ t('bi.source.model.action.auto_layout') }}</el-button>
          </Panel>
          <template #node-table="{ data }">
            <article class="source-model__node" :class="{ 'is-base': data.key === model.base_table }">
              <Handle id="in" type="target" :position="data.targetPosition" :style="data.key === model.base_table ? { top: '12px' } : undefined" />
              <el-header style="--el-header-height: auto; --el-header-padding: 0">
                <el-text tag="strong" :title="schemaLabel(tableFor(data.key), data.name)">{{ schemaLabel(tableFor(data.key), data.name) }}</el-text>
                <el-tag v-if="data.key === model.base_table" size="small" type="success">{{ t('bi.source.model.field.base_table') }}</el-tag>
              </el-header>
              <el-text tag="small">{{ data.schema_name }}</el-text>
              <div class="source-model__node-columns nodrag nowheel">
                <div v-for="column in tableFor(data.key)?.columns" :key="column.name">
                  <span :title="schemaLabel(column)">{{ schemaLabel(column) }}</span>
                  <el-text tag="small">{{ column.data_type }}</el-text>
                </div>
              </div>
              <Handle v-for="handle in data.handles" :id="handle.id" :key="handle.id" type="source" :position="handle.position" :style="{ top: `${handle.top}%` }" />
            </article>
          </template>
        </VueFlow>

        <el-aside class="source-model__relation-panel" style="--el-aside-width: auto; overflow: visible">
          <template v-if="selectedRelation">
            <div class="source-model__relation-heading">
              <el-text tag="strong">{{ t('bi.source.model.field.relation') }}</el-text>
              <el-button text type="danger" :icon="Delete" @click="removeRelation" />
            </div>
            <el-form>
              <el-form-item :label="schemaLabel(tableFor(selectedRelation.left_table), selectedRelation.left_table)">
                <el-select v-model="selectedRelation.left_column" :placeholder="t('bi.source.model.field.left_column')">
                  <el-option v-for="column in tableFor(selectedRelation.left_table)?.columns" :key="column.name" :label="schemaLabel(column)" :value="column.name" />
                </el-select>
              </el-form-item>
              <el-form-item :label="schemaLabel(tableFor(selectedRelation.right_table), selectedRelation.right_table)">
                <el-select v-model="selectedRelation.right_column" :placeholder="t('bi.source.model.field.right_unique_column')">
                  <el-option v-for="column in uniqueColumns(selectedRelation.right_table)" :key="column.name" :label="schemaLabel(column)" :value="column.name" />
                </el-select>
              </el-form-item>
              <el-form-item :label="t('bi.source.model.field.join_type')">
                <el-radio-group v-model="selectedRelation.join_type">
                  <el-radio-button value="left">LEFT</el-radio-button>
                  <el-radio-button value="inner">INNER</el-radio-button>
                </el-radio-group>
              </el-form-item>
            </el-form>
          </template>
          <el-empty v-else :description="t('bi.source.model.relation_empty')" :image-size="72" />
        </el-aside>
      </section>

      <section v-else class="source-model__semantics">
        <div class="source-model__field-layout">
          <aside class="source-model__browser">
            <div class="source-model__browser-head">
              <el-text tag="strong">{{ t('bi.source.model.browser.title') }}</el-text>
              <el-input v-model="columnFilter" size="small" clearable :prefix-icon="Search" :placeholder="t('bi.source.model.browser.search')" />
            </div>
            <p class="source-model__browser-hint">{{ t('bi.source.model.browser.hint') }}</p>
            <el-scrollbar class="source-model__browser-list">
              <div v-for="table in browserTables" :key="table.key" class="source-model__browser-table">
                <div class="source-model__browser-table-name" :title="schemaLabel(table)">{{ schemaLabel(table) }}<el-tag v-if="table.key === model.base_table" size="small" type="success" effect="plain">{{ t('bi.source.model.field.base_table') }}</el-tag></div>
                <div
                  v-for="column in browserColumns(table)"
                  :key="column.name"
                  class="source-model__browser-column"
                  :class="{'is-dimension': isDimensionAdded(table.key, column.name), 'is-measure': isMeasureColumnAdded(table.key, column.name)}"
                >
                  <span class="source-model__column-type" :data-type="fieldType(table.key, column.name)">{{ typeLabel(fieldType(table.key, column.name)) }}</span>
                  <span class="source-model__browser-column-name" :title="schemaLabel(column)">{{ schemaLabel(column) }}</span>
                  <div class="source-model__browser-column-actions">
                    <template v-if="fieldType(table.key, column.name) === 'number' && table.key === model.base_table">
                      <el-tag v-if="isMeasureColumnAdded(table.key, column.name)" size="small" type="info" effect="plain">{{ t('bi.source.model.browser.added_measure') }}</el-tag>
                      <el-button v-else size="small" text type="primary" @click="addMeasureFrom(table, column)">{{ t('bi.source.model.browser.add_measure') }}</el-button>
                    </template>
                    <el-tooltip v-else-if="fieldType(table.key, column.name) === 'number'" :content="t('bi.source.model.browser.base_only')" placement="top">
                      <el-button size="small" text type="primary" disabled>{{ t('bi.source.model.browser.add_measure') }}</el-button>
                    </el-tooltip>
                    <el-tag v-if="isDimensionAdded(table.key, column.name)" size="small" type="info" effect="plain">{{ t('bi.source.model.browser.added_dimension') }}</el-tag>
                    <el-button v-else size="small" text type="primary" @click="addDimensionFrom(table, column)">{{ t('bi.source.model.browser.add_dimension') }}</el-button>
                  </div>
                </div>
              </div>
              <el-empty v-if="!browserHasColumns" :description="t('bi.source.model.browser.empty')" :image-size="64" />
            </el-scrollbar>
          </aside>

          <div class="source-model__field-lists">
            <div class="source-model__field-list">
              <header class="source-model__field-list-head">
                <div class="source-model__field-list-title">
                  <el-text tag="strong">{{ t('bi.source.model.field.dimensions') }}</el-text>
                  <el-text size="small" type="info">{{ t('bi.source.model.hint.dimension') }}</el-text>
                </div>
                <el-text size="small" type="info" v-if="model.dimensions.length">{{ model.dimensions.length }} 项</el-text>
              </header>
              <p v-if="!model.dimensions.length" class="source-model__list-empty">{{ t('bi.source.model.dimensions_empty') }}</p>
              <div v-else class="source-model__field-rows">
                <el-form-item :label="t('bi.refine.timezone')">
                  <el-input v-model="model.report_timezone" clearable maxlength="64" placeholder="Asia/Shanghai" :aria-label="t('bi.refine.timezone')" />
                </el-form-item>
                <div v-for="(row, index) in model.dimensions" :key="row.code" class="source-model__dimension">
                <div class="source-model__field-row">
                  <span class="source-model__column-type" :data-type="row.type">{{ typeLabel(row.type) }}</span>
                  <el-input v-model="row.name" size="small" :placeholder="t('bi.source.model.field.display_name')" class="source-model__name-input" />
                  <span class="source-model__field-ref" :title="fieldRef(row.table, row.column)">{{ fieldRef(row.table, row.column) }}</span>
                  <div class="source-model__filter-cell">
                    <el-switch v-model="row.filterable" size="small" />
                    <span>{{ t('bi.source.model.field.filterable') }}</span>
                    <el-tooltip :content="t('bi.source.model.hint.filterable')" placement="top">
                      <el-icon class="source-model__filter-help"><QuestionFilled /></el-icon>
                    </el-tooltip>
                  </div>
                  <el-button text type="danger" size="small" :icon="Delete" :aria-label="t('bi.source.model.action.remove_field')" @click="model.dimensions.splice(index, 1)" />
                </div>
                  <el-form v-if="row.hierarchy || hierarchyParentFields(row).length" inline size="small" class="source-model__hierarchy">
                    <el-form-item :label="t('bi.refine.parent_column')">
                      <el-select :model-value="row.hierarchy?.parent_column" clearable :placeholder="t('bi.refine.no_hierarchy')" :aria-label="t('bi.refine.parent_column')" @change="setHierarchyParent(row, String($event ?? ''))">
                        <el-option v-for="column in hierarchyParentFields(row)" :key="column.name" :label="schemaLabel(column)" :value="column.name" />
                      </el-select>
                    </el-form-item>
                    <el-form-item v-if="row.hierarchy" :label="t('bi.refine.root_id')">
                      <el-input v-model="row.hierarchy.root_id" maxlength="20" placeholder="NULL" :aria-label="t('bi.refine.root_id')" />
                    </el-form-item>
                  </el-form>
                </div>
              </div>
            </div>

            <div class="source-model__field-list">
              <header class="source-model__field-list-head">
                <div class="source-model__field-list-title">
                  <el-text tag="strong">{{ t('bi.source.model.field.measures') }}</el-text>
                  <el-text size="small" type="info">{{ t('bi.source.model.hint.measure') }}</el-text>
                </div>
                <el-button size="small" :icon="MagicStick" :disabled="!recommendableMeasures.length" @click="recommendMeasures">{{ t('bi.source.model.action.recommend_measures') }}</el-button>
              </header>
              <p v-if="!model.measures.length" class="source-model__list-empty">{{ t('bi.source.model.measures_empty') }}</p>
              <div v-else class="source-model__field-rows">
                <div v-for="(row, index) in model.measures" :key="row.code" class="source-model__field-row is-measure-row">
                  <el-input v-model="row.name" size="small" :placeholder="t('bi.source.model.field.display_name')" class="source-model__name-input" />
                  <el-select
                    size="small"
                    class="source-model__agg-select"
                    :model-value="aggOf(row.expression)"
                    @update:model-value="setMeasureAgg(row, String($event ?? ''))"
                  >
                    <el-option value="sum" :label="t('bi.card.aggregation.sum')" />
                    <el-option value="count" :label="t('bi.card.aggregation.count')" />
                    <el-option value="avg" :label="t('bi.card.aggregation.avg')" />
                    <el-option value="max" :label="t('bi.card.aggregation.max')" />
                    <el-option value="min" :label="t('bi.card.aggregation.min')" />
                    <el-option value="custom" :label="t('bi.source.model.aggregation.custom')" />
                  </el-select>
                  <span v-if="aggOf(row.expression) !== 'custom'" class="source-model__field-ref source-model__expression-text" :title="row.expression">{{ row.expression }}</span>
                  <CodeEditor v-else v-model="row.expression" language="sql" height="72px" class="source-model__expression" />
                  <el-input v-model="row.unit" size="small" maxlength="32" :placeholder="t('bi.sql.unit')" class="source-model__unit-input" />
                  <el-button text type="danger" size="small" :icon="Delete" :aria-label="t('bi.source.model.action.remove_field')" @click="model.measures.splice(index, 1)" />
                </div>
              </div>
            </div>
          </div>
        </div>

        <el-alert :title="t('bi.sql.shared_notice')" type="info" show-icon :closable="false" />
        <div v-if="impact" class="source-model__preview">
          <el-alert :title="t('bi.sql.impact', { cards: impact.affected_cards ?? 0, dashboards: impact.affected_dashboards ?? 0 })" type="warning" show-icon :closable="false" />
          <div v-for="change in impact.changes ?? []" :key="change.after?.code ?? change.before?.code">
            <el-text tag="strong">{{ change.after?.name ?? change.before?.name }}</el-text>
            <pre>{{ change.before?.expression ?? '∅' }} → {{ change.after?.expression ?? '∅' }}</pre>
          </div>
        </div>
        <div v-if="previewResult" class="source-model__preview">
          <el-header style="--el-header-height: auto; --el-header-padding: 0">
            <el-text tag="strong">{{ t('bi.query.result') }}</el-text>
            <span>{{ t('bi.query.elapsed', {ms: previewResult.elapsed_ms}) }}</span>
          </el-header>
          <el-table :data="previewRows" size="small" border>
            <el-table-column v-for="(column, index) in previewResult.columns" :key="column.code" :label="column.name" min-width="130">
              <template #default="{ row }">{{ row[`value_${index}`] }}</template>
            </el-table-column>
          </el-table>
        </div>
      </section>
    </div>

    <template #footer>
      <div class="source-model__footer">
        <el-button @click="open = false">{{ t('common.action.cancel') }}</el-button>
        <el-button v-if="step > 0" @click="step--">{{ t('bi.source.model.action.previous') }}</el-button>
        <el-button v-if="step < 2" type="primary" @click="nextStep">{{ t('bi.source.model.action.next') }}</el-button>
        <template v-else>
          <el-button :disabled="loading || saving" :loading="analyzing" @click="analyzeModel">{{ t('bi.sql.preview_changes') }}</el-button>
          <el-button :disabled="loading || saving || analyzing" @click="refreshVersion">{{ t('bi.sql.refresh_model') }}</el-button>
          <el-tooltip :content="t('bi.source.model.hint.save_requires_impact')" :disabled="impactReady" placement="top">
            <span>
              <el-button type="primary" :loading="saving" :disabled="!impactReady" @click="saveModel">{{ t('bi.source.model.action.save') }}</el-button>
            </span>
          </el-tooltip>
          <el-button :icon="DataAnalysis" :disabled="!modelSaved" :loading="previewing" @click="previewModel">{{ t('bi.source.model.action.preview') }}</el-button>
          <el-button v-if="modelSaved" @click="open = false">{{ t('bi.source.model.action.done') }}</el-button>
        </template>
      </div>
    </template>
  </el-dialog>

  <el-dialog v-model="tablePreviewOpen" :title="tablePreviewTitle" width="min(1200px, 92vw)" destroy-on-close @closed="closeTablePreview">
    <ProTable
      v-if="previewTable"
      :key="`${props.source?.id ?? 0}-${previewTable.key}`"
      row-key="__row_id"
      :columns="dataPreviewColumns"
      :init-param="previewInitParam"
      :request-api="requestTableData"
      :request-error="handlePreviewError"
      :tool-button="false"
    />
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, h, nextTick, ref, useId, watch } from "vue";
import { VueFlow, Handle, MarkerType, Panel, Position, useVueFlow } from "@vue-flow/core";
import type { Connection, Edge, EdgeMouseEvent, Node, NodeDragEvent } from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import { Controls } from "@vue-flow/controls";
import { MiniMap } from "@vue-flow/minimap";
import { DataAnalysis, Delete, MagicStick, QuestionFilled, Search, View } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import CodeEditor from "@liujitcn/kratos-admin-core/components/CodeEditor/index.vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import type { ColumnProps } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { defBiQueryService } from "../api/bi/admin/v1/bi_query";
import { defBiSourceService } from "../api/bi/admin/v1/bi_source";
import { formatSourceModelLayout } from "./source-model-layout";
import type { BiQueryResult } from "../rpc/bi/admin/v1/bi_query";
import type {
  BiSource,
  BiModelImpact,
  BiSourceDimension,
  BiSourceMeasure,
  BiSourceModel,
  BiSourceModelTable,
  BiSourceRelation,
  BiSourceSchemaColumn,
  BiSourceSchemaTable,
  PageBiSourceTableDataResponse
} from "../rpc/bi/admin/v1/bi_source";
import "@vue-flow/core/dist/style.css";
import "@vue-flow/controls/dist/style.css";
import "@vue-flow/minimap/dist/style.css";

/** 数据库中保存的语义模型 JSON。 */
interface StoredSourceModel {
  /** 事实表标识。 */
  baseTable?: string;
  /** 报表时区。 */
  reportTimezone?: string;
  /** 已选数据表。 */
  tables?: Array<BiSourceModelTable & {tenantColumn?: string}>;
  /** 表关系。 */
  relations?: Array<BiSourceRelation & {leftTable?: string; leftColumn?: string; rightTable?: string; rightColumn?: string; joinType?: string}>;
  /** 语义维度。 */
  dimensions?: Array<Omit<BiSourceDimension, "hierarchy"> & { hierarchy?: { parentColumn: string; rootId: string } }>;
  /** 语义指标。 */
  measures?: BiSourceMeasure[];
}

const props = defineProps<{modelValue: boolean; source?: BiSource}>();
const emit = defineEmits<{"update:modelValue": [value: boolean]; saved: [source: BiSource]}>();
const open = computed({get: () => props.modelValue, set: value => emit("update:modelValue", value)});
const flowId = `bi-source-model-${useId()}`;
const { fitView, updateNodeInternals, viewportHelper } = useVueFlow(flowId);
const loading = ref(false);
const saving = ref(false);
const previewing = ref(false);
const analyzing = ref(false);
const sourceVersion = ref(0);
const impact = ref<BiModelImpact>();
const impactSnapshot = ref("");
const impactReady = computed(() => Boolean(impact.value) && impactSnapshot.value === JSON.stringify(model.value));
let modelRequest = 0;
let modelSession = 0;
const step = ref(0);
const tableFilter = ref("");
const schemaTables = ref<BiSourceSchemaTable[]>([]);
const model = ref<BiSourceModel>(emptyModel());
const selectedRelationIndex = ref(-1);
const savedModelSnapshot = ref("");
const modelSaved = computed(() => savedModelSnapshot.value !== "" && savedModelSnapshot.value === JSON.stringify(model.value));
const previewResult = ref<BiQueryResult>();
const tablePreviewOpen = ref(false);
const previewTable = ref<BiSourceSchemaTable>();
const previewInitParam = ref({id: 0, table_key: "", tenant_column: "", page_size: 20});
let tablePreviewRequest = 0;
const tablePreviewTitle = computed(() => previewTable.value ? schemaLabel(previewTable.value) : t("bi.source.model.preview.title"));
const dataPreviewColumns = computed<ColumnProps[]>(() => (previewTable.value?.columns ?? []).map(column => ({
  prop: column.name,
  label: column.comment || column.name,
  minWidth: 140,
  showOverflowTooltip: true,
  headerRender: () => h("span", {title: column.name}, column.comment || column.name)
})));
const selectedTables = computed(() => schemaTables.value.filter(table => model.value.tables.some(item => item.key === table.key)));
const filteredTables = computed(() => schemaTables.value.filter(table => `${table.schema_name}.${schemaLabel(table)}`.toLowerCase().includes(tableFilter.value.toLowerCase())));
const selectedRelation = computed(() => model.value.relations[selectedRelationIndex.value]);
/** 字段浏览器搜索关键字。 */
const columnFilter = ref("");
/** 指标编码到基础列名的映射，用于聚合方式切换时重建表达式。 */
const measureColumnMap = ref<Record<string, string>>({});
/** 指标仅支持事实表数值字段；推荐指标候选 = 事实表中尚未添加为指标的、非枚举外键且非审计字段的数值字段。 */
const recommendableMeasures = computed(() => (tableFor(model.value.base_table)?.columns ?? [])
  .filter(column => fieldType(model.value.base_table, column.name) === "number" && !isAuxiliaryColumn(column) && !isMeasureColumnAdded(model.value.base_table, column.name)));
/** 字段浏览器是否有可展示的字段。 */
const browserHasColumns = computed(() => selectedTables.value.some(table => browserColumns(table).length > 0));
/** 字段浏览器的表顺序：事实表优先，其余按选择顺序。 */
const browserTables = computed(() => [
  ...selectedTables.value.filter(table => table.key === model.value.base_table),
  ...selectedTables.value.filter(table => table.key !== model.value.base_table)
]);
/** 按搜索关键字过滤指定表的字段。 */
function browserColumns(table: BiSourceSchemaTable): BiSourceSchemaColumn[] {
  return table.columns.filter(column => `${column.name} ${column.comment}`.toLowerCase().includes(columnFilter.value.toLowerCase()));
}
/** 数据表节点及按目标纵向顺序分布的连接点。 */
type TableNodeData = {key: string; name: string; schema_name: string; targetPosition: Position; handles: Array<{id: string; top: number; position: Position}>};
const nodes = computed<Node<TableNodeData>[]>(() => selectedTables.value.map((table, index) => {
  const saved = model.value.tables.find(item => item.key === table.key);
  const incoming = model.value.relations.find(relation => relation.right_table === table.key);
  const targetPosition = incoming && (modelTable(incoming.left_table)?.x ?? 0) > (saved?.x ?? 0) ? Position.Right : Position.Left;
  const outgoing = model.value.relations.map((relation, relationIndex) => ({
    relation,
    relationIndex,
    position: (modelTable(relation.right_table)?.x ?? 0) < (saved?.x ?? 0) ? Position.Left : Position.Right
  })).filter(({relation}) => relation.left_table === table.key)
    .sort((left, right) => (modelTable(left.relation.right_table)?.y ?? 0) - (modelTable(right.relation.right_table)?.y ?? 0));
  return {
    id: table.key,
    type: "table",
    position: {x: saved?.x ?? 80 + (index % 3) * 330, y: saved?.y ?? 60 + Math.floor(index / 3) * 290},
    data: {
      key: table.key,
      name: table.name,
      schema_name: table.schema_name,
      targetPosition,
      handles: outgoing.length ? outgoing.map(handle => {
        const siblings = outgoing.filter(item => item.position === handle.position);
        return {id: `relation-${handle.relationIndex}`, top: (siblings.indexOf(handle) + 1) * 100 / (siblings.length + 1), position: handle.position};
      }) : [{id: "out", top: 50, position: targetPosition === Position.Right ? Position.Left : Position.Right}]
    }
  };
}));
const edges = computed<Edge[]>(() => model.value.relations.map((relation, index) => ({
  id: `relation-${index}`,
  source: relation.left_table,
  target: relation.right_table,
  sourceHandle: `relation-${index}`,
  targetHandle: "in",
  type: "default",
  selected: selectedRelationIndex.value === index,
  label: `${schemaLabel(tableFor(relation.left_table)?.columns.find(column => column.name === relation.left_column), relation.left_column || "?")} → ${schemaLabel(tableFor(relation.right_table)?.columns.find(column => column.name === relation.right_column), relation.right_column || "?")}`,
  markerEnd: MarkerType.ArrowClosed,
  interactionWidth: 20,
  style: {stroke: selectedRelationIndex.value === index ? "var(--el-color-primary)" : "var(--el-text-color-placeholder)"},
  labelStyle: {fill: "var(--el-text-color-regular)", fontSize: 12},
  labelBgStyle: {fill: "var(--el-bg-color)", stroke: "var(--el-border-color-lighter)"},
  labelBgPadding: [6, 4],
  labelBgBorderRadius: 4
})));
const previewRows = computed(() => previewResult.value?.rows.map(row => Object.fromEntries(row.values.map((value, index) => [`value_${index}`, value]))) ?? []);

watch(() => [props.modelValue, props.source?.id] as const, ([visible]) => {
  modelSession++;
  modelRequest++;
  tablePreviewRequest++;
  tablePreviewOpen.value = false;
  previewTable.value = undefined;
  loading.value = saving.value = analyzing.value = previewing.value = false;
  impact.value = undefined;
  previewResult.value = undefined;
  if (visible) void loadModel();
});
watch(model, () => { previewResult.value = undefined; impact.value = undefined; modelRequest++; }, {deep: true, flush: "sync"});
watch(nodes, async () => {
  await nextTick();
  updateNodeInternals();
}, {flush: "post"});

/** 初始化数据库结构和已保存的语义模型。 */
async function loadModel() {
  if (!props.source) return;
  const currentSession = modelSession;
  const sourceID = props.source.id;
  loading.value = true;
  step.value = 0;
  savedModelSnapshot.value = "";
  sourceVersion.value = 0;
  schemaTables.value = [];
  model.value = emptyModel();
  columnFilter.value = "";
  measureColumnMap.value = {};
  previewResult.value = undefined;
  try {
    const [current, result] = await Promise.all([
      defBiSourceService.GetBiSource({id: sourceID}),
      defBiSourceService.ListBiSourceTables({id: sourceID})
    ]);
    if (!isCurrentSession(currentSession, sourceID)) return;
    sourceVersion.value = current.version;
    schemaTables.value = result.tables ?? [];
    model.value = readStoredModel(current.schema_json);
    if (!model.value.tables.length) model.value = emptyModel();
    savedModelSnapshot.value = current.model_configured ? JSON.stringify(model.value) : "";
  } catch (cause) {
    if (isCurrentSession(currentSession, sourceID)) ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (isCurrentSession(currentSession, sourceID)) loading.value = false;
  }
}

/** 将数据库中保存的 JSON 转换为前端模型。 */
function readStoredModel(value: string): BiSourceModel {
  try {
    const stored = JSON.parse(value) as StoredSourceModel;
    return {
      base_table: stored.baseTable ?? "",
      report_timezone: stored.reportTimezone ?? "",
      tables: (stored.tables ?? []).map(table => ({key: table.key, tenant_column: table.tenantColumn ?? table.tenant_column ?? "", x: table.x ?? 0, y: table.y ?? 0})),
      relations: (stored.relations ?? []).map(relation => ({
        left_table: relation.leftTable ?? relation.left_table ?? "",
        left_column: relation.leftColumn ?? relation.left_column ?? "",
        right_table: relation.rightTable ?? relation.right_table ?? "",
        right_column: relation.rightColumn ?? relation.right_column ?? "",
        join_type: relation.joinType ?? relation.join_type ?? "left"
      })),
      dimensions: (stored.dimensions ?? []).map(({ hierarchy, ...dimension }) => ({
        ...dimension, filterable: dimension.filterable ?? false,
        hierarchy: hierarchy ? { parent_column: hierarchy.parentColumn, root_id: hierarchy.rootId ?? "" } : undefined
      })),
      measures: (stored.measures ?? []).map(measure => ({...measure, unit: measure.unit ?? "", revision: measure.revision ?? 0}))
    };
  } catch {
    return emptyModel();
  }
}

/** 候选父编号与左连接目标使用同一种整数类型。 */
function hierarchyParentFields(dimension: BiSourceDimension): BiSourceSchemaColumn[] {
  if (dimension.type !== "string" || props.source?.driver === "doris") return [];
  const relation = model.value.relations.find(item =>
    item.right_table === dimension.table && item.left_table === model.value.base_table && item.join_type === "left"
  );
  const table = tableFor(dimension.table);
  const id = table?.columns.find(column => column.name === relation?.right_column);
  if (!id || !id.data_type.toLowerCase().includes("int")) return [];
  return (table?.columns ?? []).filter(column => column.name !== id.name && column.data_type === id.data_type);
}

/** 清除父列关闭层级。常见的 parent_id 使用 0 作为顶层父编号。 */
function setHierarchyParent(dimension: BiSourceDimension, parent: string) {
  dimension.hierarchy = parent ? { parent_column: parent, root_id: dimension.hierarchy?.root_id ?? (parent === "parent_id" ? "0" : "") } : undefined;
}

/** 创建空语义模型。 */
function emptyModel(): BiSourceModel {
  return {base_table: "", tables: [], relations: [], dimensions: [], measures: [], report_timezone: ""};
}

/** 判断表是否已加入模型。 */
function isSelected(key: string): boolean {
  return model.value.tables.some(table => table.key === key);
}

/** 更换事实表后保留公式草稿，保存前由服务端按新事实表校验字段。 */
function setBaseTable(key: string) {
  model.value.base_table = key;
}

/** 查询模型表节点以编辑租户隔离字段。 */
function modelTable(key: string): BiSourceModelTable | undefined {
  return model.value.tables.find(table => table.key === key);
}

/** 更新表的租户隔离字段。 */
function setTenantColumn(key: string, column: string) {
  const table = modelTable(key);
  if (table) table.tenant_column = column;
}

/** 加入或移除建模表并清理失效的关系和字段。 */
function toggleTable(table: BiSourceSchemaTable, selected: boolean) {
  if (selected) {
    if (model.value.tables.length >= 8) return;
    const tenantColumn = table.columns.find(column => column.name === "tenant_id")?.name ?? "";
    model.value.tables.push({key: table.key, tenant_column: tenantColumn, x: 80 + (model.value.tables.length % 3) * 330, y: 60 + Math.floor(model.value.tables.length / 3) * 290});
    if (!model.value.base_table) model.value.base_table = table.key;
    return;
  }
  model.value.tables = model.value.tables.filter(item => item.key !== table.key);
  model.value.relations = model.value.relations.filter(item => item.left_table !== table.key && item.right_table !== table.key);
  model.value.dimensions = model.value.dimensions.filter(item => item.table !== table.key);
  if (model.value.base_table === table.key) model.value.base_table = model.value.tables[0]?.key ?? "";
  selectedRelationIndex.value = -1;
}

/** 查询指定表的发现结构。 */
function tableFor(key: string): BiSourceSchemaTable | undefined {
  return schemaTables.value.find(table => table.key === key);
}

/** 打开指定表的数据预览并冻结当前租户列配置。 */
function openTableData(table: BiSourceSchemaTable) {
  if (!props.source) return;
  previewTable.value = table;
  previewInitParam.value = {
    id: props.source.id,
    table_key: table.key,
    tenant_column: modelTable(table.key)?.tenant_column ?? table.columns.find(column => column.name === "tenant_id")?.name ?? "",
    page_size: 20
  };
  tablePreviewRequest++;
  tablePreviewOpen.value = true;
}

/** 关闭表数据预览并使未完成请求失效。 */
function closeTablePreview() {
  tablePreviewRequest++;
  previewTable.value = undefined;
}

/** 将表数据响应转换为 ProTable 所需的分页结构。 */
function mapTableData(result: PageBiSourceTableDataResponse, tableKey: string, pageNum: number) {
  return {
    data: {
      list: (result.rows ?? []).map((row, rowIndex) => ({
        __row_id: `${tableKey}:${pageNum}:${rowIndex}`,
        ...Object.fromEntries(result.columns.map((column, index) => [column.name, row.values[index] ?? ""]))
      })),
      total: result.total
    }
  };
}

/** 请求当前表的数据页，并忽略切换表或数据源后的迟到响应。 */
async function requestTableData(params: Record<string, unknown>) {
  const currentRequest = tablePreviewRequest;
  const sourceID = props.source?.id ?? 0;
  const table = previewTable.value;
  if (!sourceID || !table || !tablePreviewOpen.value) return {data: {list: [], total: 0}};
  const request = buildPageRequest({
    ...previewInitParam.value,
    ...params,
    id: sourceID,
    table_key: table.key,
    tenant_column: previewInitParam.value.tenant_column
  });
  const result = await defBiSourceService.PageBiSourceTableData(request);
  if (currentRequest !== tablePreviewRequest || !tablePreviewOpen.value || sourceID !== props.source?.id || table.key !== previewTable.value?.key) {
    return {data: {list: [], total: 0}};
  }
  return mapTableData(result, table.key, request.page_num);
}

/** 展示表数据预览请求错误。 */
function handlePreviewError(cause: unknown) {
  if (tablePreviewOpen.value) ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
}

/** 表和字段展示注释，选择值仍使用原始英文标识。 */
function schemaLabel(item: Pick<BiSourceSchemaColumn, "name" | "comment"> | undefined, name = item?.name ?? ""): string {
  return item?.comment ? `${name}（${item.comment}）` : name;
}

/** 返回维度表中可作为多对一目标的唯一字段。 */
function uniqueColumns(key: string): BiSourceSchemaColumn[] {
  return tableFor(key)?.columns.filter(column => column.unique) ?? [];
}

/** 按数据库字段类型推导 BI 展示类型。 */
function fieldType(table: string, column: string): string {
  const value = tableFor(table)?.columns.find(item => item.name === column)?.data_type.toLowerCase() ?? "";
  if (/int|decimal|numeric|real|double|float/.test(value)) return "number";
  if (/^time(?:tz)?(?: |$)/.test(value)) return "string";
  if (/date|time/.test(value)) return "date";
  return "string";
}

/** 显示字段的 BI 类型标签。 */
function typeLabel(type: string): string {
  return type === "number" ? t("bi.source.model.type.number") : type === "date" ? t("bi.source.model.type.date") : t("bi.source.model.type.string");
}

/** 判断维度是否已经覆盖指定表字段。 */
function isDimensionAdded(tableKey: string, columnName: string): boolean {
  return model.value.dimensions.some(dimension => dimension.table === tableKey && dimension.column === columnName);
}

/** 判断数值字段是否不适合作为指标：注释带"枚举"标记，或字段名为 *_id、*_type、*_status、*_state、*_by、*_at、is_*、has_* 形态（外键、枚举和创建人、时间等通用审计字段）。 */
function isAuxiliaryColumn(column: BiSourceSchemaColumn): boolean {
  if (column.comment.includes("枚举")) return true;
  return /_(id|type|status|state|by|at)$/.test(column.name.toLowerCase()) || /^(is|has)_/.test(column.name.toLowerCase());
}

/** 从聚合表达式解析基础列名，无法解析时返回 null。 */
function parseMeasureColumn(expression: string): string | null {
  const match = /^\s*([A-Za-z]+)\s*\((.+)\)\s*$/.exec(expression);
  return match ? match[2].trim() : null;
}

/** 判断事实表数值字段是否已经添加为指标。 */
function isMeasureColumnAdded(tableKey: string, columnName: string): boolean {
  if (tableKey !== model.value.base_table) return false;
  return model.value.measures.some(measure => (measureColumnMap.value[measure.code] ?? parseMeasureColumn(measure.expression)) === columnName);
}

/** 解析指标的聚合方式，无法识别时视为自定义 SQL。 */
function aggOf(expression: string): string {
  const match = /^\s*([A-Za-z]+)\s*\((.*)\)\s*$/.exec(expression);
  const agg = match?.[1].toLowerCase() ?? "";
  return ["sum", "count", "avg", "max", "min"].includes(agg) ? agg : "custom";
}

/** 切换聚合方式后重建表达式；自定义 SQL 保留原文。 */
function setMeasureAgg(row: BiSourceMeasure, agg: string) {
  if (agg === "custom") return;
  const column = measureColumnMap.value[row.code] ?? parseMeasureColumn(row.expression);
  if (!column) return;
  row.expression = `${agg.toUpperCase()}(${column})`;
}

/** 展示维度所属表和字段。 */
function fieldRef(tableKey: string, column: string): string {
  return `${tableFor(tableKey)?.name ?? tableKey}.${column}`;
}

/** 从字段浏览器把字段添加为语义维度。 */
function addDimensionFrom(table: BiSourceSchemaTable, column: BiSourceSchemaColumn) {
  if (isDimensionAdded(table.key, column.name)) return;
  const code = semanticCode(table.name, column.name, model.value.dimensions.length);
  model.value.dimensions.push({code, name: column.comment || column.name, type: fieldType(table.key, column.name), table: table.key, column: column.name, filterable: true, hierarchy: undefined});
}

/** 从字段浏览器把事实表数值字段添加为指标。 */
function addMeasureFrom(table: BiSourceSchemaTable, column: BiSourceSchemaColumn) {
  if (table.key !== model.value.base_table || isMeasureColumnAdded(table.key, column.name)) return;
  const code = semanticCode(table.name, column.name, model.value.measures.length);
  // id 形态字段（主键/雪花 ID）默认按计数聚合，避免"数量"类问数生成 SUM(大整数) 的无意义结果
  const expression = /(^|_)id$/i.test(column.name) ? `COUNT(${column.name})` : `SUM(${column.name})`;
  model.value.measures.push({code, name: column.comment || column.name, type: "number", expression, unit: "", revision: 0});
  measureColumnMap.value[code] = column.name;
}

/** 一键把事实表全部数值字段建为指标。 */
function recommendMeasures() {
  for (const column of recommendableMeasures.value) addMeasureFrom(tableFor(model.value.base_table)!, column);
}

/** 生成合法且稳定的语义字段编码。 */
function semanticCode(table: string, column: string, index: number): string {
	const prefix = `${table}_${column}`.toLowerCase().replace(/[^a-z0-9_]/g, "_");
	let suffix = index + 1;
	let code = `${prefix}_${suffix}`;
	while ([...model.value.dimensions, ...model.value.measures].some(field => field.code === code)) {
		code = `${prefix}_${++suffix}`;
	}
	return code;
}

/** 拖动画布连接点后创建一条待配置关系。 */
function connectTables(connection: Connection) {
  const source = connection.source;
  const target = connection.target;
  if (!source || !target || source === target || model.value.relations.some(item => item.right_table === target)) return;
  let leftTable = source;
  let rightTable = target;
  const sourceReachable = reachableFromBase(source);
  const targetReachable = reachableFromBase(target);
  if (targetReachable && !sourceReachable) [leftTable, rightTable] = [target, source];
  else if (sourceReachable && targetReachable) return;
  model.value.relations.push({left_table: leftTable, left_column: "", right_table: rightTable, right_column: "", join_type: "left"});
  selectedRelationIndex.value = model.value.relations.length - 1;
  void layoutGraph();
}

/** 判断表是否已经从事实表连通。 */
function reachableFromBase(key: string): boolean {
  const reachable = new Set([model.value.base_table]);
  for (let changed = true; changed;) {
    changed = false;
    for (const relation of model.value.relations) {
      if (reachable.has(relation.left_table) && !reachable.has(relation.right_table)) {
        reachable.add(relation.right_table);
        changed = true;
      }
    }
  }
  return reachable.has(key);
}

/** 选择画布关系并打开其配置。 */
function selectRelation({edge}: EdgeMouseEvent) {
  selectedRelationIndex.value = Number(edge.id.replace("relation-", ""));
}

/** 删除当前选中的关系。 */
function removeRelation() {
  if (selectedRelationIndex.value < 0) return;
  model.value.relations.splice(selectedRelationIndex.value, 1);
  selectedRelationIndex.value = -1;
}

/** 将画布节点坐标保存到语义模型。 */
function moveTable({node}: NodeDragEvent) {
  const table = model.value.tables.find(item => item.key === node.id);
  if (!table) return;
  table.x = Math.round(node.position.x);
  table.y = Math.round(node.position.y);
}

/** 整理表关系布局后适配画布视口。 */
async function layoutGraph() {
  formatSourceModelLayout(model.value);
  await nextTick();
  fitGraph();
}

/** 初始化或恢复画布视口。 */
function fitGraph() {
  if (viewportHelper.value.viewportInitialized) void fitView({padding: 0.18, maxZoom: 1});
}

/** 校验当前步骤后进入下一步。 */
function nextStep() {
  if (step.value === 0 && (!model.value.base_table || selectedTables.value.length === 0)) {
    ElMessage.warning(t("bi.source.model.validation.select_tables"));
    return;
  }
  if (step.value === 0) formatSourceModelLayout(model.value);
  step.value++;
}

/** 预览公共定义差异和全部引用卡片的静态编译影响。 */
async function analyzeModel() {
  if (!props.source || loading.value || analyzing.value || saving.value) return;
  const currentSession = modelSession;
  const sourceID = props.source.id;
  const request = ++modelRequest;
  const snapshot = JSON.stringify(model.value);
  analyzing.value = true;
  try {
    const result = await defBiSourceService.PreviewBiSourceModel({id: sourceID, model: model.value, expected_source_version: sourceVersion.value});
    if (request !== modelRequest || !isCurrentSession(currentSession, sourceID)) return;
    impact.value = result;
    impactSnapshot.value = snapshot;
  } catch (cause) {
    if (isCurrentSession(currentSession, sourceID)) ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (isCurrentSession(currentSession, sourceID)) analyzing.value = false;
  }
}

/** 版本冲突后保留当前草稿，刷新版本并要求重新检查全部差异。 */
async function refreshVersion() {
  if (!props.source) return;
  const currentSession = modelSession;
  const sourceID = props.source.id;
  loading.value = true;
  try {
    const result = await defBiSourceService.GetBiSource({id: sourceID});
    if (!isCurrentSession(currentSession, sourceID)) return;
    sourceVersion.value = result.version;
    impact.value = undefined;
    modelRequest++;
    ElMessage.info(t("bi.sql.repreview"));
  } catch (cause) {
    if (isCurrentSession(currentSession, sourceID)) ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (isCurrentSession(currentSession, sourceID)) loading.value = false;
  }
}

/** 保存关系模型并启用数据源。 */
async function saveModel() {
  if (!props.source || loading.value || !impactReady.value || saving.value) return;
  const currentSession = modelSession;
  const sourceID = props.source.id;
  let closeAfterSave = false;
  saving.value = true;
  try {
    const result = await defBiSourceService.SaveBiSourceModel({id: sourceID, model: model.value, expected_source_version: sourceVersion.value});
    if (!isCurrentSession(currentSession, sourceID)) return;
    sourceVersion.value = result.version;
    model.value = readStoredModel(result.schema_json);
    savedModelSnapshot.value = JSON.stringify(model.value);
    previewResult.value = undefined;
    emit("saved", result);
    ElMessage.success(t("bi.sql.synced", { cards: result.sync_summary?.affected_cards ?? 0, dashboards: result.sync_summary?.affected_dashboards ?? 0 }));
    closeAfterSave = true;
  } catch (cause) {
    if (isCurrentSession(currentSession, sourceID)) ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (isCurrentSession(currentSession, sourceID)) {
      saving.value = false;
      if (closeAfterSave) open.value = false;
    }
  }
}

/** 使用一个维度和指标预览已保存模型。 */
async function previewModel() {
  if (!props.source || !modelSaved.value || !model.value.measures.length) return;
  const currentSession = modelSession;
  const sourceID = props.source.id;
  const request = modelRequest;
  previewing.value = true;
  try {
    const dimension = model.value.dimensions[0];
    const measure = model.value.measures[0];
    const result = await defBiQueryService.PreviewBiQuery({query: {
      source_code: props.source.code,
      dimensions: dimension ? [dimension.code] : [],
      measures: [{code: measure.code}],
      filters: [],
      limit: 10,
      time_range: undefined,
      orders: [],
      grouping_options: []
    }});
    if (request !== modelRequest || !isCurrentSession(currentSession, sourceID)) return;
    previewResult.value = result;
  } catch (cause) {
    if (isCurrentSession(currentSession, sourceID)) ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    if (isCurrentSession(currentSession, sourceID)) previewing.value = false;
  }
}

/** 关闭、重开或切换源后，旧请求不能写入当前模型或清除新请求状态。 */
function isCurrentSession(currentSession: number, sourceID: number): boolean {
  return currentSession === modelSession && props.modelValue && sourceID === props.source?.id;
}
</script>

<style scoped lang="scss">
.source-model {
  display: flex;
  height: 100%;
  flex-direction: column;
  padding-top: 4px;
}

.source-model > .el-steps {
  flex-shrink: 0;
}

.source-model__tables {
  display: grid;
  grid-template-rows: auto 1fr auto;
  flex: 1;
  min-height: 0;
  margin-top: 20px;
}

.source-model__table-toolbar {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) minmax(220px, 320px);
  gap: 12px;
  padding-bottom: 12px;
}

.source-model__table-list {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);
}

.source-model__table-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 14px;
  flex-wrap: nowrap;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.source-model__table-info {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: baseline;
  gap: 8px;
  justify-content: flex-start;
  text-align: left;
}

.source-model__table-info strong,
.source-model__table-info small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-model__table-info strong {
  flex: 1 1 auto;
  min-width: 0;
}

.source-model__table-info small,
.source-model__selection-count {
  flex-shrink: 0;
  color: var(--el-text-color-secondary);
}

.source-model__table-actions {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}

.source-model__tenant-select {
  width: 190px;
}

.source-model__selection-count {
  padding-top: 8px;
  text-align: right;
}

.source-model__relations {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  flex: 1;
  min-height: 0;
  gap: 12px;
  margin-top: 20px;
}

.source-model__relations :deep(.vue-flow) {
  overflow: hidden;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);
  background: var(--el-fill-color-lighter);
}

.source-model__node {
  display: flex;
  width: 250px;
  height: 230px;
  box-sizing: border-box;
  flex-direction: column;
  padding: 10px;
  color: var(--el-text-color-primary);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: var(--admin-page-radius, 6px);
  box-shadow: var(--el-box-shadow-light);
}

.source-model__node.is-base {
  border-color: var(--el-color-success);
}

.source-model__node header,
.source-model__relation-heading,
.source-model__semantic-section header,
.source-model__preview header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.source-model__node header strong {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-model__node header .el-tag {
  flex-shrink: 0;
}

.source-model__node > small {
  color: var(--el-text-color-secondary);
}

.source-model__node-columns {
  min-height: 0;
  flex: 1;
  overflow: auto;
  margin-top: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}

.source-model__node-columns > div {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  padding: 4px 2px;
  font-size: 12px;
}

.source-model__node-columns small {
  color: var(--el-text-color-secondary);
}

.source-model__node-columns span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-model__relations :deep(.vue-flow__edge-path) {
  stroke-width: 1.5;
}

.source-model__relations :deep(.vue-flow__edge.selected .vue-flow__edge-path) {
  stroke-width: 2.5;
}

.source-model__relations :deep(.vue-flow__handle) {
  width: 8px;
  height: 8px;
  background: var(--el-color-primary);
  border: 2px solid var(--el-bg-color);
}

.source-model__relations :deep(.vue-flow__controls),
.source-model__relations :deep(.vue-flow__minimap) {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);
}

.source-model__relations :deep(.vue-flow__controls-button) {
  color: var(--el-text-color-primary);
  fill: currentColor;
  background: var(--el-bg-color);
  border-color: var(--el-border-color-lighter);
}

.source-model__relation-panel {
  min-width: 0;
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);
}

.source-model__relation-heading {
  margin-bottom: 14px;
}

.source-model__semantics {
  display: flex;
  flex: 1;
  min-height: 0;
  flex-direction: column;
  gap: 14px;
  margin-top: 20px;
  padding-right: 2px;
}

.source-model__field-layout {
  display: grid;
  flex: 1;
  min-height: 0;
  grid-template-columns: 300px minmax(0, 1fr);
  gap: 14px;
}

.source-model__browser {
  display: grid;
  grid-template-rows: auto auto 1fr;
  min-height: 0;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);
}

.source-model__browser-head {
  display: grid;
  gap: 8px;
  padding: 10px 12px 0;
}

.source-model__browser-hint {
  margin: 8px 12px 4px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
  border-radius: var(--admin-page-radius, 6px);
  padding: 7px 9px;
}

.source-model__browser-list {
  min-height: 0;
  padding: 2px 8px 10px;
}

.source-model__browser-table-name {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 4px 4px;
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-regular);
}

.source-model__browser-column {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 4px;
  border-radius: var(--admin-page-radius, 6px);
}

.source-model__browser-column:hover {
  background: var(--el-fill-color-light);
}

.source-model__column-type {
  flex-shrink: 0;
  padding: 1px 5px;
  font-size: 12px;
  border-radius: 4px;
  background: var(--el-fill-color);
  color: var(--el-text-color-secondary);
}

.source-model__column-type[data-type="number"] {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
}

.source-model__column-type[data-type="date"] {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
}

.source-model__browser-column-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-model__browser-column-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: 4px;
}

.source-model__browser-column-actions .el-button {
  padding: 2px 7px;
}

.source-model__field-lists {
  display: grid;
  min-height: 0;
  align-content: start;
  gap: 14px;
  overflow: auto;
  padding-right: 2px;
}

.source-model__field-list,
.source-model__preview {
  display: grid;
  flex-shrink: 0;
  gap: 10px;
}

.source-model__field-list-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-bottom: none;
  border-radius: var(--admin-page-radius, 6px) var(--admin-page-radius, 6px) 0 0;
}

.source-model__field-list-title {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: baseline;
  gap: 8px;
}

.source-model__field-list-head strong,
.source-model__preview header strong {
  font-weight: 600;
}

.source-model__hierarchy { padding: 0 8px; }
.source-model__hierarchy :deep(.el-select) { min-width: 150px; }
.source-model__field-rows {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 0 0 var(--admin-page-radius, 6px) var(--admin-page-radius, 6px);
}

.source-model__field-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
}

.source-model__field-row + .source-model__field-row {
  border-top: 1px solid var(--el-border-color-lighter);
}

.source-model__field-row:hover {
  background: var(--el-fill-color-lighter);
}

.source-model__name-input {
  width: 140px;
  flex-shrink: 0;
}

.source-model__agg-select {
  width: 110px;
  flex-shrink: 0;
}

.source-model__field-ref {
  min-width: 0;
  overflow: hidden;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-model__expression-text {
  flex: 1;
  font-family: var(--el-font-family, monospace);
}

.source-model__expression {
  flex: 1;
  min-width: 0;
}

.source-model__unit-input {
  width: 80px;
  flex-shrink: 0;
}

.source-model__filter-cell {
  display: flex;
  margin-left: auto;
  align-items: center;
  gap: 5px;
  flex-shrink: 0;
  font-size: 12px;
  color: var(--el-text-color-regular);
}

.source-model__filter-help {
  color: var(--el-text-color-placeholder);
  cursor: help;
}

.source-model__list-empty {
  padding: 16px 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  text-align: center;
  border: 1px dashed var(--el-border-color-lighter);
  border-radius: var(--admin-page-radius, 6px);
}

.source-model__preview header span {
  color: var(--el-text-color-secondary);
}

.source-model__footer {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}

@media (max-width: 760px) {
  .source-model {
    min-height: 0;
  }

  .source-model__table-toolbar,
  .source-model__relations,
  .source-model__field-layout {
    grid-template-columns: 1fr;
  }

  .source-model__field-layout {
    flex: none;
  }

  .source-model__semantics {
    overflow-y: auto;
  }

  .source-model__browser {
    max-height: 300px;
  }

  .source-model__table-row {
    gap: 8px;
    padding-inline: 10px;
  }

  .source-model__relations {
    height: auto;
  }

  .source-model__tenant-select {
    width: 130px;
  }

  .source-model__table-info {
    gap: 6px;
  }

  .source-model__table-info small {
    display: none;
  }

  .source-model__table-actions {
    gap: 4px;
  }

  .source-model__relations :deep(.vue-flow) {
    height: 420px;
  }

  .source-model__relation-panel {
    min-height: 200px;
  }
}
</style>

<style lang="scss">
/* 弹窗整体不超过视口：主体弹性填充剩余空间，保证底部按钮始终可见、页面不出滚动条。 */
.bi-source-model-dialog {
  display: flex;

  // 显式高度打破 flex 内容高度循环：el-dialog 高度由内容决定，body 为 flex-basis 0、
  // 内容 .source-model 又 height:100% 反向依赖 body，缺显式高度会双向塌陷（body 仅剩 padding 高度）。
  height: min(92vh, 920px);
  max-height: 96dvh;
  flex-direction: column;

  .el-dialog__body {
    flex: 1 1 0;
    min-height: 0;
    overflow: auto;
  }
}
</style>
