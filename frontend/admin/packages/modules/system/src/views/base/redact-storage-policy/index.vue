<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      v-model="dialog.visible"
      ref="dialogRef"
      :title="t(dialog.titleKey)"
      width="1180px"
      top="3vh"
      label-width="150px"
      destroy-on-close
      :model="form"
      :fields="fields"
      :rules="rules"
      @confirm="submit"
      @close="resetForm"
    >
      <template #column_rows>
        <div class="field-table">
          <div class="field-table__summary">
            {{ t("system.base.redact_storage_policy.message.configured_fields", { count: configuredFieldCount }) }}
          </div>
          <el-table :data="form.column_rows" row-key="value" border height="500">
            <el-table-column :label="t('system.base.redact_storage_policy.field.column_name')" min-width="260">
              <template #default="{ row }">
                <div class="field-cell">
                  <div class="field-cell__title">{{ row.name }}</div>
                  <div class="field-cell__meta">{{ row.comment }}<span v-if="row.db_type"> · {{ row.db_type }}</span></div>
                </div>
              </template>
            </el-table-column>
            <el-table-column :label="t('system.base.redact_storage_policy.field.rule')" width="270">
              <template #default="{ row }">
                <el-select v-model="row.rule_id" class="table-control" size="small" clearable @change="handleRuleChange(row)">
                  <el-option v-for="option in ruleOptions" :key="String(option.value)" :label="option.label" :value="option.value" :disabled="option.disabled" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column :label="t('system.base.redact_storage_policy.field.parameters')" min-width="520">
              <template #default="{ row }">
                <div v-if="row.rule_id" class="parameter-grid">
                  <template v-if="row.rule_type === 'MASK'">
                    <ParameterNumber v-model="row.params.keep_first" :label="t('system.base.redact_rule.parameter.keep_first')" />
                    <ParameterNumber v-model="row.params.keep_last" :label="t('system.base.redact_rule.parameter.keep_last')" />
                    <ParameterNumber v-model="row.params.min_mask" :label="t('system.base.redact_rule.parameter.min_mask')" />
                    <ParameterText v-model="row.params.mask_char" :label="t('system.base.redact_rule.parameter.mask_char')" />
                  </template>
                  <template v-else-if="row.rule_type === 'EMAIL'">
                    <ParameterNumber v-model="row.params.keep_local_first" :label="t('system.base.redact_rule.parameter.keep_local_first')" />
                    <div class="parameter-item"><span>{{ t("system.base.redact_rule.parameter.mask_domain") }}</span><el-switch v-model="row.params.mask_domain" /></div>
                    <ParameterText v-model="row.params.mask_char" :label="t('system.base.redact_rule.parameter.mask_char')" />
                  </template>
                  <template v-else-if="row.rule_type === 'REGEX'">
                    <ParameterText v-model="row.params.pattern" :label="t('system.base.redact_rule.parameter.pattern')" />
                    <ParameterText v-model="row.params.replacement" :label="t('system.base.redact_rule.parameter.replacement')" />
                  </template>
                  <template v-else-if="row.rule_type === 'TRUNCATE'">
                    <ParameterNumber v-model="row.params.length" :label="t('system.base.redact_rule.parameter.length')" />
                    <ParameterText v-model="row.params.suffix" :label="t('system.base.redact_rule.parameter.suffix')" />
                  </template>
                  <template v-else-if="row.rule_type === 'HASH'">
                    <div class="parameter-item"><span>{{ t("system.base.redact_rule.parameter.algo") }}</span><el-select v-model="row.params.algo" class="parameter-control" size="small"><el-option label="MD5" value="MD5" /><el-option label="SHA1" value="SHA1" /><el-option label="SHA256" value="SHA256" /></el-select></div>
                  </template>
                  <template v-else-if="row.rule_type === 'IP'">
                    <ParameterNumber v-model="row.params.keep_octets" :label="t('system.base.redact_rule.parameter.keep_octets')" />
                    <ParameterText v-model="row.params.mask_char" :label="t('system.base.redact_rule.parameter.mask_char')" />
                  </template>
                  <template v-else-if="row.rule_type === 'URL'">
                    <div class="parameter-item"><span>{{ t("system.base.redact_rule.parameter.mask_query") }}</span><el-switch v-model="row.params.mask_query" /></div>
                    <ParameterText v-model="row.params.mask_char" :label="t('system.base.redact_rule.parameter.mask_char')" />
                  </template>
                  <template v-else-if="row.rule_type === 'FIXED_LENGTH'">
                    <ParameterText v-model="row.params.char" :label="t('system.base.redact_rule.parameter.char')" />
                  </template>
                </div>
                <span v-else class="empty-value">--</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, reactive, ref } from "vue";
import type { FormRules } from "element-plus";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { buildPageRequest, normalizeSelectedIds } from "@liujitcn/kratos-admin-core/table";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { t } from "@liujitcn/kratos-admin-core";
import { defBaseTableSourceService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_table_source";
import { defBaseRedactRuleService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_redact_rule";
import { defBaseRedactStoragePolicyService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_redact_storage_policy";
import type { BaseRedactRule } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_rule";
import type { BaseRedactStoragePolicy, BaseRedactStoragePolicyForm, PageBaseRedactStoragePolicyRequest } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_redact_storage_policy";
import { Status } from "@liujitcn/kratos-admin-system/rpc/common/v1/enum";

/** 脱敏规则参数。 */
interface RuleParams {
  keep_first?: number;
  keep_last?: number;
  min_mask?: number;
  mask_char?: string;
  keep_local_first?: number;
  mask_domain?: boolean;
  pattern?: string;
  replacement?: string;
  length?: number;
  suffix?: string;
  algo?: string;
  keep_octets?: number;
  mask_query?: boolean;
  char?: string;
}

/** 入库字段表格行。 */
interface StorageColumnRow extends ProFormOption {
  /** 数据库字段名。 */
  name: string;
  /** 数据库字段注释。 */
  comment: string;
  /** 数据库字段类型。 */
  db_type: string;
  /** 已有策略 ID。 */
  id: number;
  /** 脱敏规则 ID。 */
  rule_id?: number;
  /** 脱敏规则类型。 */
  rule_type: string;
  /** 脱敏规则参数对象。 */
  params: RuleParams;
  /** 脱敏规则参数 JSON。 */
  rule_params: string;
}

/** 入库脱敏表单状态，新增时租户保持未选择。 */
interface StorageFormState extends Omit<BaseRedactStoragePolicyForm, "tenant_id"> {
  /** 租户ID。 */
  tenant_id?: number;
  /** 当前选择的字段表格。 */
  column_rows: StorageColumnRow[];
}

/** 入库表格行事件使用的最小字段视图。 */
interface StorageTableRow {
  /** 脱敏规则 ID。 */
  rule_id?: number;
  /** 脱敏规则类型。 */
  rule_type?: string;
  /** 脱敏规则参数对象。 */
  params?: RuleParams;
  /** 脱敏规则参数 JSON。 */
  rule_params?: string;
}

const STORAGE_AUDIT_COLUMN_NAMES = new Set([
  "id", "created_at", "created_by", "updated_at", "updated_by", "deleted_at", "deleted_by",
  "create_time", "creator", "creator_id", "update_time", "updater", "updater_id", "delete_time", "deleter", "deleter_id"
]);
const ParameterText = defineComponent({
  props: { modelValue: { type: [String, Number], default: "" }, label: { type: String, required: true } },
  emits: ["update:modelValue"],
  setup(props, { emit }) {
    return () => h("div", { class: "parameter-item" }, [h("span", props.label), h("input", { class: "el-input__inner", value: props.modelValue, onInput: (event: Event) => emit("update:modelValue", (event.target as HTMLInputElement).value) })]);
  }
});
const ParameterNumber = defineComponent({
  props: { modelValue: { type: Number, default: 0 }, label: { type: String, required: true } },
  emits: ["update:modelValue"],
  setup(props, { emit }) {
    return () => h("div", { class: "parameter-item" }, [h("span", props.label), h("input", { class: "el-input__inner", type: "number", min: 0, value: props.modelValue, onInput: (event: Event) => emit("update:modelValue", Number((event.target as HTMLInputElement).value)) })]);
  }
});

defineOptions({ name: "BaseRedactStoragePolicy", inheritAttrs: false });
const { BUTTONS } = useAuthButtons();
const { isDefaultTenant, tenantColumns, tenantFormField, toRequestTenantId, loadTenantOptions } = useTenantScope();
onMounted(() => {
  void loadTenantOptions();
});
const table = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const sourceOptions = ref<ProFormOption[]>([]);
const tableOptions = ref<ProFormOption[]>([]);
const tableCommentMap = ref(new Map<string, string>());
const ruleOptions = ref<ProFormOption[]>([]);
const ruleCatalog = ref<BaseRedactRule[]>([]);
const dialog = reactive({ visible: false, titleKey: "common.action.create_resource" });
const form = reactive<StorageFormState>(defaultForm());
let columnsRequestRevision = 0;
const statusOptions = computed<ProFormOption[]>(() => [{ label: t("common.status.enabled"), value: Status.STATUS_ENABLE }, { label: t("common.status.disabled"), value: Status.STATUS_DISABLE }]);
const configuredFieldCount = computed(() => form.column_rows.filter(row => Boolean(row.rule_id)).length);
const fields = computed<ProFormField[]>(() => [
  tenantFormField({ label: t("common.field.tenant"), disabledOnEdit: true }),
  { prop: "source_name", label: t("system.base.redact_storage_policy.field.source_name"), component: "select", props: { filterable: true, disabled: Boolean(form.id), onChange: handleSourceChange }, options: sourceOptions.value },
  { prop: "table_name", label: t("system.base.redact_storage_policy.field.table_name"), component: "select", props: { filterable: true, disabled: Boolean(form.id) || !form.source_name, onChange: handleTableChange }, options: tableOptions.value },
  { prop: "column_rows", label: t("system.base.redact_storage_policy.field.column_name"), component: "slot", slotName: "column_rows", colSpan: 24, visible: () => Boolean(form.table_name) },
  { prop: "status", label: t("common.field.status"), component: "radio-group", options: statusOptions.value },
  { prop: "remark", label: t("common.field.remark"), component: "textarea" }
]);
const rules = computed<FormRules>(() => ({
  tenant_id: [{ required: isDefaultTenant.value, message: t("common.validation.required_select", { field: t("common.field.tenant") }), trigger: "change" }],
  source_name: [{ required: true, message: t("system.base.redact_storage_policy.validation.source_name"), trigger: "change" }],
  table_name: [{ required: true, message: t("system.base.redact_storage_policy.validation.table_name"), trigger: "change" }],
  column_rows: [{ validator: (_rule, value: StorageColumnRow[], callback) => value?.some(row => Boolean(row.rule_id)) ? callback() : callback(new Error(t("system.base.redact_storage_policy.validation.configured_fields"))), trigger: "change" }]
}));
const columns = computed<ColumnProps[]>(() => [
  { type: "selection", width: 55 },
  ...tenantColumns({ label: t("common.field.tenant"), order: 1 }),
  { prop: "source_name", label: t("system.base.redact_storage_policy.field.source_name"), minWidth: 130, search: { el: "input" } },
  { prop: "table_name", label: t("system.base.redact_storage_policy.field.table_name"), minWidth: 180, search: { el: "input" }, render: scope => tableLabel(scope.row as BaseRedactStoragePolicy) },
  { prop: "column_name", label: t("system.base.redact_storage_policy.field.column_name"), minWidth: 150, search: { el: "input" } },
  { prop: "rule_name", label: t("system.base.redact_storage_policy.field.rule"), minWidth: 150 },
  { prop: "status", label: t("common.field.status"), width: 100, cellType: "status", statusProps: { activeValue: Status.STATUS_ENABLE, inactiveValue: Status.STATUS_DISABLE, activeText: t("common.status.enabled"), inactiveText: t("common.status.disabled"), disabled: () => !BUTTONS.value["base:redact-storage-policy:status"], beforeChange: scope => changeStatus(scope.row as BaseRedactStoragePolicy) } },
  { prop: "actions", label: t("common.field.operation"), cellType: "actions", actions: [{ label: t("common.action.edit"), type: "primary", link: true, icon: EditPen, hidden: () => !BUTTONS.value["base:redact-storage-policy:update"], onClick: scope => openDialog((scope.row as BaseRedactStoragePolicy).id) }, { label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !BUTTONS.value["base:redact-storage-policy:delete"], onClick: scope => deleteItems(scope.row as BaseRedactStoragePolicy) }] }
]);
const headerActions = computed<HeaderActionProps[]>(() => [{ label: t("common.action.create"), type: "success", icon: CirclePlus, hidden: () => !BUTTONS.value["base:redact-storage-policy:create"], onClick: () => openDialog() }, { label: t("common.action.delete"), type: "danger", icon: Delete, hidden: () => !BUTTONS.value["base:redact-storage-policy:delete"], disabled: scope => !scope.selectedList.length, onClick: scope => deleteItems(scope.selectedList as BaseRedactStoragePolicy[]) }]);

/** 请求入库脱敏策略分页列表。 */
async function requestTable(params: PageBaseRedactStoragePolicyRequest) { const data = await defBaseRedactStoragePolicyService.PageBaseRedactStoragePolicy({ ...buildPageRequest(params), tenant_id: toRequestTenantId(params.tenant_id) }); return { data: { list: data.base_redact_storage_policies ?? [], total: data.total } }; }
/** 加载脱敏规则选项。 */
async function loadRules() { const rules = await requestRules(); ruleCatalog.value = rules.catalog; ruleOptions.value = rules.options; }
/** 请求脱敏规则选项。 */
async function requestRules() { const data = await defBaseRedactRuleService.PageBaseRedactRule({ code: "", name: "", rule_type: "", page_num: 1, page_size: 100 }); const catalog = data.base_redact_rules ?? []; return { catalog, options: catalog.map(item => ({ label: `${item.name} (${item.code})`, value: item.id, disabled: item.status !== Status.STATUS_ENABLE })) }; }
/** 数据源变更后清空下级选择并加载数据表。 */
async function handleSourceChange(sourceName: string) { columnsRequestRevision += 1; form.source_name = sourceName; form.table_name = ""; form.column_name = ""; form.column_rows = []; tableOptions.value = []; await loadTables(); }
/** 数据表变更后加载字段表格。 */
async function handleTableChange(tableName: string) { columnsRequestRevision += 1; form.table_name = tableName; form.column_name = ""; form.column_rows = []; await loadColumns(); }
/** 加载数据源选项。 */
async function loadSourceOptions() { sourceOptions.value = await requestSourceOptions(); }
/** 请求数据源选项。 */
async function requestSourceOptions() { const data = await defBaseTableSourceService.OptionBaseTableSource({}); return (data.value ?? []).map(value => ({ label: value, value })); }
/** 加载指定数据源的数据表选项。 */
async function loadTables(sourceName = form.source_name) { tableOptions.value = await requestTables(sourceName); }
/** 请求指定数据源的数据表选项。 */
async function requestTables(sourceName: string) { if (!sourceName) return []; const data = await defBaseRedactStoragePolicyService.ListBaseRedactStorageTable({ source_name: sourceName }); return (data.tables ?? []).map(item => ({ label: item.comment && item.comment !== item.name ? `${item.comment}（${item.name}）` : item.name, value: item.name })); }
/** 预加载数据表中文注释，供列表显示。 */
async function loadTableComments() { await loadSourceOptions(); const comments = new Map<string, string>(); await Promise.all(sourceOptions.value.map(async option => { const sourceName = String(option.value); const data = await defBaseRedactStoragePolicyService.ListBaseRedactStorageTable({ source_name: sourceName }); for (const item of data.tables ?? []) { if (item.comment) comments.set(`${sourceName}\x00${item.name}`, item.comment); } })); tableCommentMap.value = comments; }
/** 加载并过滤数据库字段。 */
async function loadColumns() { const revision = ++columnsRequestRevision; const rows = await requestColumns(form.source_name, form.table_name); if (revision === columnsRequestRevision) form.column_rows = rows; }
/** 请求并转换数据库字段。 */
async function requestColumns(sourceName: string, tableName: string) { if (!sourceName || !tableName) return []; const data = await defBaseRedactStoragePolicyService.ListBaseRedactStorageColumn({ source_name: sourceName, table_name: tableName }); return (data.columns ?? []).filter(item => !STORAGE_AUDIT_COLUMN_NAMES.has(item.name.toLowerCase())).map(item => createColumnRow({ label: item.name, value: item.name, name: item.name, comment: item.comment, db_type: item.db_type })); }
/** 打开新增或编辑弹窗。 */
async function openDialog(id?: number) { await dialogRef.value?.open({ load: async () => { await loadTenantOptions(); const [sources, rules, data] = await Promise.all([requestSourceOptions(), requestRules(), id !== undefined ? defBaseRedactStoragePolicyService.GetBaseRedactStoragePolicy({ id }) : Promise.resolve(undefined)]); const sourceName = data?.source_name ?? ""; const tables = await requestTables(sourceName); const columns = data ? await requestColumns(sourceName, data.table_name) : []; return { sources, rules, data, tables, columns }; }, commit: ({ sources, rules, data, tables, columns }) => { resetForm(); sourceOptions.value = sources; tableOptions.value = tables; ruleCatalog.value = rules.catalog; ruleOptions.value = rules.options; if (data) { Object.assign(form, data); form.column_rows = columns; const row = form.column_rows.find(item => item.name === data.column_name); if (row) applyPolicy(row, data); } dialog.titleKey = id !== undefined ? "common.action.edit_resource" : "common.action.create_resource"; } }); }
/** 重置弹窗表单。 */
function resetForm() { dialog.visible = false; dialogRef.value?.resetFields(); Object.assign(form, defaultForm()); sourceOptions.value = []; tableOptions.value = []; }
/** 保存表格中已配置的全部字段。 */
async function submit() {
  const valid = await dialogRef.value?.validate();
  const rows = form.column_rows.filter(row => Boolean(row.rule_id));
  const removedIds = form.column_rows.filter(row => row.id > 0 && !row.rule_id).map(row => row.id);
  if (!valid || (!rows.length && !removedIds.length)) return;
  const createPolicies = rows.filter(row => !row.id).map(buildPayload);
  const updatePolicies = rows.filter(row => Boolean(row.id)).map(buildPayload);
  const requests: Promise<unknown>[] = [];
  if (createPolicies.length) requests.push(defBaseRedactStoragePolicyService.CreateBaseRedactStoragePolicy({ base_redact_storage_policy: createPolicies }));
  if (updatePolicies.length) requests.push(defBaseRedactStoragePolicyService.UpdateBaseRedactStoragePolicy({ base_redact_storage_policy: updatePolicies }));
  if (removedIds.length) requests.push(defBaseRedactStoragePolicyService.DeleteBaseRedactStoragePolicy({ id: removedIds.join(",") }));
  await Promise.all(requests);
  ElMessage.success(t("system.base.redact_storage_policy.message.batch_save_success", { count: rows.length }));
  resetForm();
  table.value?.getTableList();
}
/** 将表格行转换为入库策略请求。 */
function buildPayload(row: StorageColumnRow): BaseRedactStoragePolicyForm { syncRowParams(row); return { id: row.id, tenant_id: form.tenant_id ?? 0, source_name: form.source_name, table_name: form.table_name, column_name: row.name, rule_id: row.rule_id ?? 0, rule_params: row.rule_params, status: form.status, remark: form.remark }; }
/** 将已保存的策略回填到对应字段行。 */
function applyPolicy(row: StorageColumnRow, data: BaseRedactStoragePolicyForm) { row.id = data.id; row.rule_id = data.rule_id || undefined; row.rule_params = data.rule_params; if (row.rule_id) { const rule = ruleCatalog.value.find(item => item.id === row.rule_id); row.rule_type = rule?.rule_type ?? ""; row.params = rule ? parseParams(row.rule_type, row.rule_params, rule.rule) : {}; } }
/** 处理行规则变更。 */
function handleRuleChange(row: StorageTableRow) { const rule = ruleCatalog.value.find(item => item.id === row.rule_id); row.rule_type = rule?.rule_type ?? ""; row.params = rule ? parseParams(rule.rule_type, rule.rule) : {}; row.rule_params = rule ? rule.rule : "{}"; }
/** 同步单行规则参数。 */
function syncRowParams(row: StorageColumnRow) { const rule = ruleCatalog.value.find(item => item.id === row.rule_id); if (rule) { row.rule_type = rule.rule_type; row.rule_params = JSON.stringify({ [rule.rule_type.toLowerCase()]: row.params }); } }
/** 创建空的入库字段行。 */
function createColumnRow(option: Pick<StorageColumnRow, "label" | "value" | "name" | "comment" | "db_type">): StorageColumnRow { return { ...option, id: 0, rule_id: undefined, rule_type: "", params: {}, rule_params: "{}" }; }
/** 解析规则参数，空策略参数时回退到数据库规则参数。 */
function parseParams(ruleType: string, raw: string, fallbackRaw = ""): RuleParams {
  const candidates = fallbackRaw && fallbackRaw !== raw ? [raw, fallbackRaw] : [raw];
  for (const candidate of candidates) {
    try {
      const value = JSON.parse(candidate) as Record<string, RuleParams>;
      const params = value[ruleType.toLowerCase()];
      if (params && typeof params === "object" && !Array.isArray(params)) return params;
    } catch {
      continue;
    }
  }
  return {};
}
/** 修改入库策略状态。 */
async function changeStatus(row: BaseRedactStoragePolicy) { const next = row.status === Status.STATUS_ENABLE ? Status.STATUS_DISABLE : Status.STATUS_ENABLE; try { await ElMessageBox.confirm(t("common.dialog.status_change", { action: t(next === Status.STATUS_ENABLE ? "common.status.enabled" : "common.status.disabled"), resource: t("system.base.redact_storage_policy.title"), field: t("system.base.redact_storage_policy.field.column_name"), value: row.column_name }), t("common.title.warning"), { type: "warning" }); await defBaseRedactStoragePolicyService.SetBaseRedactStoragePolicyStatus({ id: row.id, status: next }); table.value?.getTableList(); return true; } catch { return false; } }
/** 删除选中的入库策略。 */
function deleteItems(selected?: BaseRedactStoragePolicy | BaseRedactStoragePolicy[] | number | string | Array<number | string>) { const items = Array.isArray(selected) ? selected.filter((item): item is BaseRedactStoragePolicy => typeof item === "object") : selected && typeof selected === "object" ? [selected] : []; const ids = items.length ? items.map(item => item.id) : normalizeSelectedIds(selected as number | string | Array<number | string>); if (!ids.length) { ElMessage.warning(t("common.message.select_delete_item")); return; } ElMessageBox.confirm(t("common.dialog.delete_selected", { resource: t("system.base.redact_storage_policy.title") }), t("common.title.warning"), { type: "warning" }).then(async () => { await defBaseRedactStoragePolicyService.DeleteBaseRedactStoragePolicy({ id: ids.join(",") }); ElMessage.success(t("common.message.delete_success", { resource: t("system.base.redact_storage_policy.title") })); table.value?.getTableList(); }); }
/** 创建默认入库表单。 */
function defaultForm(): StorageFormState { return { id: 0, tenant_id: undefined, source_name: "", table_name: "", column_name: "", column_rows: [], rule_id: 0, rule_params: "{}", status: Status.STATUS_ENABLE, remark: "" }; }

/** 格式化列表中的数据表名称。 */
function tableLabel(row: BaseRedactStoragePolicy) { const comment = tableCommentMap.value.get(`${row.source_name}\x00${row.table_name}`); return comment && comment !== row.table_name ? `${comment}（${row.table_name}）` : row.table_name; }

void loadTableComments().catch(() => undefined);
</script>

<style scoped lang="scss">
.field-table { width: 100%; }
.field-table__summary { margin-bottom: 8px; color: var(--el-text-color-secondary); line-height: 1.5; }
.field-cell { min-width: 0; line-height: 1.35; }
.field-cell__title { overflow: hidden; color: var(--el-text-color-regular); text-overflow: ellipsis; white-space: nowrap; }
.field-cell__meta { overflow: hidden; color: var(--el-text-color-secondary); font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
.table-control { width: 100%; }
.parameter-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; min-width: 480px; }
.parameter-item { display: flex; flex-direction: column; gap: 4px; min-width: 0; color: var(--el-text-color-regular); font-size: 12px; }
.parameter-control { width: 100%; }
.empty-value { color: var(--el-text-color-secondary); }
@media (max-width: 900px) { .parameter-grid { grid-template-columns: minmax(0, 1fr); min-width: 0; } }
</style>
