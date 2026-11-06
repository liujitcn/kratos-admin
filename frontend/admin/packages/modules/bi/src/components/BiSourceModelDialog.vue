<template>
  <el-dialog
    v-model="open"
    :title="source?.name || t('bi.source.model.title')"
    width="min(1440px, 96vw)"
    top="3vh"
    destroy-on-close
    class="bi-source-model-dialog"
  >
    <div v-loading="loading" class="source-model">
      <el-steps :active="step" finish-status="success" align-center>
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
              <strong :title="schemaLabel(table)">{{ schemaLabel(table) }}</strong>
              <small>{{ table.schema_name }} · {{ table.columns.length }} {{ t('bi.source.model.fields_count') }}</small>
            </div>
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
              <header>
                <strong :title="schemaLabel(tableFor(data.key), data.name)">{{ schemaLabel(tableFor(data.key), data.name) }}</strong>
                <el-tag v-if="data.key === model.base_table" size="small" type="success">{{ t('bi.source.model.field.base_table') }}</el-tag>
              </header>
              <small>{{ data.schema_name }}</small>
              <div class="source-model__node-columns nodrag nowheel">
                <div v-for="column in tableFor(data.key)?.columns" :key="column.name">
                  <span :title="schemaLabel(column)">{{ schemaLabel(column) }}</span>
                  <small>{{ column.data_type }}</small>
                </div>
              </div>
              <Handle v-for="handle in data.handles" :id="handle.id" :key="handle.id" type="source" :position="handle.position" :style="{ top: `${handle.top}%` }" />
            </article>
          </template>
        </VueFlow>

        <aside class="source-model__relation-panel">
          <template v-if="selectedRelation">
            <div class="source-model__relation-heading">
              <strong>{{ t('bi.source.model.field.relation') }}</strong>
              <el-button text type="danger" :icon="Delete" @click="removeRelation" />
            </div>
            <el-form label-position="top">
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
        </aside>
      </section>

      <section v-else class="source-model__semantics">
        <div class="source-model__semantic-section">
          <header>
            <strong>{{ t('bi.source.model.field.dimensions') }}</strong>
            <el-button size="small" :icon="CirclePlus" @click="addDimension">{{ t('bi.source.model.action.add_dimension') }}</el-button>
          </header>
          <el-table :data="model.dimensions" size="small" border>
            <el-table-column :label="t('bi.source.model.field.code')" min-width="135">
              <template #default="{ row }"><el-input v-model="row.code" /></template>
            </el-table-column>
            <el-table-column :label="t('bi.source.model.field.display_name')" min-width="135">
              <template #default="{ row }"><el-input v-model="row.name" /></template>
            </el-table-column>
            <el-table-column :label="t('bi.source.model.field.table')" min-width="160">
              <template #default="{ row }">
                <el-select v-model="row.table" @change="row.column = ''; row.type = 'string'">
                  <el-option v-for="table in selectedTables" :key="table.key" :label="schemaLabel(table)" :value="table.key" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column :label="t('bi.source.model.field.column')" min-width="160">
              <template #default="{ row }">
                <el-select v-model="row.column" @change="row.type = fieldType(row.table, row.column)">
                  <el-option v-for="column in tableFor(row.table)?.columns" :key="column.name" :label="schemaLabel(column)" :value="column.name" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column :label="t('bi.source.model.field.type')" width="110">
              <template #default="{ row }">{{ row.type }}</template>
            </el-table-column>
            <el-table-column :label="t('bi.source.model.field.filterable')" width="115" align="center">
              <template #default="{ row }"><el-switch v-model="row.filterable" /></template>
            </el-table-column>
            <el-table-column width="64" align="center">
              <template #default="{ $index }"><el-button text type="danger" :icon="Delete" @click="model.dimensions.splice($index, 1)" /></template>
            </el-table-column>
          </el-table>
        </div>

        <div class="source-model__semantic-section">
          <header>
            <strong>{{ t('bi.source.model.field.measures') }}</strong>
            <el-button size="small" :icon="CirclePlus" @click="addMeasure">{{ t('bi.source.model.action.add_measure') }}</el-button>
          </header>
          <el-table :data="model.measures" size="small" border>
            <el-table-column :label="t('bi.source.model.field.code')" min-width="135">
              <template #default="{ row }"><el-input v-model="row.code" /></template>
            </el-table-column>
            <el-table-column :label="t('bi.source.model.field.display_name')" min-width="135">
              <template #default="{ row }"><el-input v-model="row.name" /></template>
            </el-table-column>
            <el-table-column :label="t('bi.source.model.field.column')" min-width="180">
              <template #default="{ row }">
                <el-select v-model="row.column" @change="row.type = 'number'; row.aggregations = defaultAggregations(row.column)">
                  <el-option v-for="column in tableFor(model.base_table)?.columns" :key="column.name" :label="schemaLabel(column)" :value="column.name" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column :label="t('bi.source.model.field.aggregation')" min-width="220">
              <template #default="{ row }">
                <el-select v-model="row.aggregations" multiple collapse-tags>
                  <el-option v-for="aggregation in aggregationOptions(row.column)" :key="aggregation" :label="aggregation" :value="aggregation" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column width="64" align="center">
              <template #default="{ $index }"><el-button text type="danger" :icon="Delete" @click="model.measures.splice($index, 1)" /></template>
            </el-table-column>
          </el-table>
        </div>

        <div v-if="previewResult" class="source-model__preview">
          <header>
            <strong>{{ t('bi.query.result') }}</strong>
            <span>{{ t('bi.query.elapsed', {ms: previewResult.elapsed_ms}) }}</span>
          </header>
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
          <el-button type="primary" :loading="saving" @click="saveModel">{{ t('bi.source.model.action.save') }}</el-button>
          <el-button :icon="DataAnalysis" :disabled="!modelSaved" :loading="previewing" @click="previewModel">{{ t('bi.source.model.action.preview') }}</el-button>
          <el-button v-if="modelSaved" @click="open = false">{{ t('bi.source.model.action.done') }}</el-button>
        </template>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, useId, watch } from "vue";
import { VueFlow, Handle, MarkerType, Panel, Position, useVueFlow } from "@vue-flow/core";
import type { Connection, Edge, EdgeMouseEvent, Node, NodeDragEvent } from "@vue-flow/core";
import { Background } from "@vue-flow/background";
import { Controls } from "@vue-flow/controls";
import { MiniMap } from "@vue-flow/minimap";
import { CirclePlus, DataAnalysis, Delete } from "@element-plus/icons-vue";
import { t } from "@liujitcn/kratos-admin-core";
import { defBiQueryService } from "../api/bi/admin/v1/bi_query";
import { defBiSourceService } from "../api/bi/admin/v1/bi_source";
import { formatSourceModelLayout } from "./source-model-layout";
import type { BiQueryResult } from "../rpc/bi/admin/v1/bi_query";
import type {
  BiSource,
  BiSourceDimension,
  BiSourceMeasure,
  BiSourceModel,
  BiSourceModelTable,
  BiSourceRelation,
  BiSourceSchemaColumn,
  BiSourceSchemaTable
} from "../rpc/bi/admin/v1/bi_source";
import "@vue-flow/core/dist/style.css";
import "@vue-flow/controls/dist/style.css";
import "@vue-flow/minimap/dist/style.css";

/** 数据库中保存的语义模型 JSON。 */
interface StoredSourceModel {
  /** 事实表标识。 */
  baseTable?: string;
  /** 已选数据表。 */
  tables?: Array<BiSourceModelTable & {tenantColumn?: string}>;
  /** 表关系。 */
  relations?: Array<BiSourceRelation & {leftTable?: string; leftColumn?: string; rightTable?: string; rightColumn?: string; joinType?: string}>;
  /** 语义维度。 */
  dimensions?: Array<BiSourceDimension & {filterable?: boolean}>;
  /** 语义指标。 */
  measures?: Array<BiSourceMeasure & {aggregations?: string[]}>;
}

const props = defineProps<{modelValue: boolean; source?: BiSource}>();
const emit = defineEmits<{"update:modelValue": [value: boolean]; saved: [source: BiSource]}>();
const open = computed({get: () => props.modelValue, set: value => emit("update:modelValue", value)});
const flowId = `bi-source-model-${useId()}`;
const { fitView, updateNodeInternals, viewportHelper } = useVueFlow(flowId);
const loading = ref(false);
const saving = ref(false);
const previewing = ref(false);
const step = ref(0);
const tableFilter = ref("");
const schemaTables = ref<BiSourceSchemaTable[]>([]);
const model = ref<BiSourceModel>(emptyModel());
const selectedRelationIndex = ref(-1);
const savedModelSnapshot = ref("");
const modelSaved = computed(() => savedModelSnapshot.value !== "" && savedModelSnapshot.value === JSON.stringify(model.value));
const previewResult = ref<BiQueryResult>();
const selectedTables = computed(() => schemaTables.value.filter(table => model.value.tables.some(item => item.key === table.key)));
const filteredTables = computed(() => schemaTables.value.filter(table => `${table.schema_name}.${schemaLabel(table)}`.toLowerCase().includes(tableFilter.value.toLowerCase())));
const selectedRelation = computed(() => model.value.relations[selectedRelationIndex.value]);
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

watch(() => props.modelValue, visible => {
  if (visible) void loadModel();
});
watch(model, () => { previewResult.value = undefined; }, {deep: true});
watch(nodes, async () => {
  await nextTick();
  updateNodeInternals();
}, {flush: "post"});

/** 初始化数据库结构和已保存的语义模型。 */
async function loadModel() {
  if (!props.source) return;
  loading.value = true;
  step.value = 0;
	savedModelSnapshot.value = "";
  previewResult.value = undefined;
  try {
    const result = await defBiSourceService.ListBiSourceTables({id: props.source.id});
    schemaTables.value = result.tables ?? [];
    model.value = readStoredModel(props.source.schema_json);
    if (!model.value.tables.length) model.value = emptyModel();
	savedModelSnapshot.value = props.source.model_configured ? JSON.stringify(model.value) : "";
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    loading.value = false;
  }
}

/** 将数据库中保存的 JSON 转换为前端模型。 */
function readStoredModel(value: string): BiSourceModel {
  try {
    const stored = JSON.parse(value) as StoredSourceModel;
    return {
      base_table: stored.baseTable ?? "",
      tables: (stored.tables ?? []).map(table => ({key: table.key, tenant_column: table.tenantColumn ?? table.tenant_column ?? "", x: table.x ?? 0, y: table.y ?? 0})),
      relations: (stored.relations ?? []).map(relation => ({
        left_table: relation.leftTable ?? relation.left_table ?? "",
        left_column: relation.leftColumn ?? relation.left_column ?? "",
        right_table: relation.rightTable ?? relation.right_table ?? "",
        right_column: relation.rightColumn ?? relation.right_column ?? "",
        join_type: relation.joinType ?? relation.join_type ?? "left"
      })),
      dimensions: (stored.dimensions ?? []).map(dimension => ({...dimension, filterable: dimension.filterable ?? false})),
      measures: (stored.measures ?? []).map(measure => ({...measure, aggregations: measure.aggregations ?? []}))
    };
  } catch {
    return emptyModel();
  }
}

/** 创建空语义模型。 */
function emptyModel(): BiSourceModel {
  return {base_table: "", tables: [], relations: [], dimensions: [], measures: []};
}

/** 判断表是否已加入模型。 */
function isSelected(key: string): boolean {
  return model.value.tables.some(table => table.key === key);
}

/** 更换事实表时仅保留仍属于新事实表的指标。 */
function setBaseTable(key: string) {
  model.value.base_table = key;
  model.value.measures = model.value.measures.filter(measure => measure.table === key);
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
  model.value.measures = model.value.measures.filter(item => item.table !== table.key);
  if (model.value.base_table === table.key) model.value.base_table = model.value.tables[0]?.key ?? "";
  selectedRelationIndex.value = -1;
}

/** 查询指定表的发现结构。 */
function tableFor(key: string): BiSourceSchemaTable | undefined {
  return schemaTables.value.find(table => table.key === key);
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
  if (/date|time/.test(value)) return "date";
  return "string";
}

/** 返回当前度量字段可用的聚合函数。 */
function aggregationOptions(column: string): string[] {
  return fieldType(model.value.base_table, column) === "number" ? ["count", "sum", "avg", "min", "max"] : ["count"];
}

/** 为新增度量选择默认聚合方式。 */
function defaultAggregations(column: string): string[] {
  return fieldType(model.value.base_table, column) === "number" ? ["sum"] : ["count"];
}

/** 增加一条语义维度。 */
function addDimension() {
  const table = selectedTables.value[0];
  const column = table?.columns[0];
  if (!table || !column) return;
  const code = semanticCode(table.name, column.name, model.value.dimensions.length);
  model.value.dimensions.push({code, name: column.name, type: fieldType(table.key, column.name), table: table.key, column: column.name, filterable: true});
}

/** 增加一条事实表指标。 */
function addMeasure() {
  const table = tableFor(model.value.base_table);
  const column = table?.columns.find(item => fieldType(table.key, item.name) === "number") ?? table?.columns[0];
  if (!table || !column) return;
  const code = semanticCode(table.name, column.name, model.value.measures.length);
  model.value.measures.push({code, name: column.name, type: "number", table: table.key, column: column.name, aggregations: defaultAggregations(column.name)});
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

/** 保存关系模型并启用数据源。 */
async function saveModel() {
  if (!props.source) return;
  saving.value = true;
  try {
    const result = await defBiSourceService.SaveBiSourceModel({id: props.source.id, model: model.value});
	savedModelSnapshot.value = JSON.stringify(model.value);
    previewResult.value = undefined;
    emit("saved", result);
    ElMessage.success(t("bi.source.model.message.saved"));
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    saving.value = false;
  }
}

/** 使用一个维度和指标预览已保存模型。 */
async function previewModel() {
  if (!props.source || !modelSaved.value || !model.value.measures.length) return;
  previewing.value = true;
  try {
    const dimension = model.value.dimensions[0];
    const measure = model.value.measures[0];
    previewResult.value = await defBiQueryService.PreviewBiQuery({query: {
      source_code: props.source.code,
      dimensions: dimension ? [dimension.code] : [],
      measures: [{code: measure.code, aggregation: measure.aggregations[0] ?? "count"}],
      filters: [],
      limit: 10,
      time_range: undefined,
      orders: []
    }});
  } catch (cause) {
    ElMessage.error(cause instanceof Error ? cause.message : t("bi.query.error"));
  } finally {
    previewing.value = false;
  }
}
</script>

<style scoped lang="scss">
.source-model {
  min-height: 560px;
  padding-top: 4px;
}

.source-model__tables {
  display: grid;
  grid-template-rows: auto 1fr auto;
  height: 510px;
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
  min-height: 52px;
  align-items: center;
  gap: 12px;
  padding: 8px 14px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.source-model__table-info {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 3px;
}

.source-model__table-info strong,
.source-model__table-info small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.source-model__table-info small,
.source-model__selection-count {
  color: var(--el-text-color-secondary);
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
  gap: 12px;
  height: 510px;
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
  display: grid;
  gap: 14px;
  max-height: 510px;
  overflow: auto;
  margin-top: 20px;
  padding-right: 2px;
}

.source-model__semantic-section,
.source-model__preview {
  display: grid;
  gap: 10px;
}

.source-model__semantic-section header strong,
.source-model__preview header strong {
  font-weight: 600;
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
  .source-model__table-toolbar,
  .source-model__relations {
    grid-template-columns: 1fr;
  }

  .source-model__table-row {
    flex-wrap: wrap;
  }

  .source-model__table-info {
    flex-basis: calc(100% - 44px);
  }

  .source-model__relations {
    height: auto;
  }

  .source-model__tenant-select {
    width: 130px;
  }

  .source-model__relations :deep(.vue-flow) {
    height: 420px;
  }

  .source-model__relation-panel {
    min-height: 200px;
  }
}
</style>
