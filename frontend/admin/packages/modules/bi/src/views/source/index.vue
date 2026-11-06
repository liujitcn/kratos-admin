<template>
  <div class="table-box">
    <ProTable ref="table" row-key="id" :columns="columns" :header-actions="headerActions" :request-api="requestTable" />
    <FormDialog
      ref="dialogRef"
      v-model="dialog.visible"
      :title="dialog.editing ? t('bi.source.title.edit') : t('bi.source.title.create')"
      width="min(1040px, 96vw)"
      label-width="10em"
      :model="form"
      :fields="fields"
      :rules="rules"
      :form-props="{ disabled: testing }"
      @confirm="submit"
      @close="resetForm"
    >
      <template #footer>
        <div class="dialog-footer">
          <el-button :disabled="testing" @click="dialog.visible = false">{{ t("common.action.cancel") }}</el-button>
          <el-button type="primary" plain :icon="Connection" :loading="testing" @click="testSource">{{ t("bi.source.action.test") }}</el-button>
          <el-button type="primary" :disabled="testing" @click="submit">{{ t("common.action.confirm") }}</el-button>
        </div>
      </template>
    </FormDialog>
    <BiSourceModelDialog v-model="modelDialog" :source="modelSource" @saved="handleModelSaved" />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { CirclePlus, Connection, Delete, EditPen, SetUp } from "@element-plus/icons-vue";
import ProTable from "@liujitcn/kratos-admin-core/components/ProTable";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import type { ColumnProps, HeaderActionProps, ProTableInstance } from "@liujitcn/kratos-admin-core/components/ProTable/interface";
import { useAuthButtons } from "@liujitcn/kratos-admin-core/auth";
import { encodeSecretFields } from "@liujitcn/kratos-admin-core/security";
import { buildPageRequest } from "@liujitcn/kratos-admin-core/table";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { t } from "@liujitcn/kratos-admin-core";
import { defBiSourceService } from "../../api/bi/admin/v1/bi_source";
import BiSourceModelDialog from "../../components/BiSourceModelDialog.vue";
import type { BiSource, BiSourceForm, PageBiSourceRequest } from "../../rpc/bi/admin/v1/bi_source";

interface BiSourceFormState extends BiSourceForm {
  password_configured: boolean;
}

defineOptions({name: "BiSource", inheritAttrs: false});

const { BUTTONS } = useAuthButtons();
const { isDefaultTenant } = useTenantScope();
const table = ref<ProTableInstance>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const dialog = reactive({visible: false, editing: false});
const form = reactive<BiSourceFormState>(defaultForm());
const modelDialog = ref(false);
const modelSource = ref<BiSource>();
const editingModelConfigured = ref(false);
const testing = ref(false);
const driverOptions = [
  {label: "MySQL", value: "mysql"},
  {label: "PostgreSQL", value: "postgres"},
  {label: "Doris", value: "doris"},
  {label: "TiDB", value: "tidb"}
];
const driverDefaultPorts: Record<string, number> = {mysql: 3306, postgres: 5432, doris: 9030, tidb: 4000};

const fields = computed<ProFormField[]>(() => [
  {prop: "code", label: t("bi.source.field.code"), component: "input", colSpan: 12, props: {maxlength: 128, disabled: dialog.editing}},
  {prop: "name", label: t("bi.source.field.name"), component: "input", colSpan: 12, props: {maxlength: 128}},
  {prop: "description", label: t("bi.source.field.description"), component: "input", colSpan: 24, props: {maxlength: 500}},
  {
    prop: "source_name",
    label: t("bi.source.field.source_name"),
    component: "input",
    colSpan: 24,
    visible: () => isDefaultTenant.value,
    props: {maxlength: 64, placeholder: t("bi.source.placeholder.source_name")}
  },
  {prop: "driver", label: t("bi.source.field.driver"), component: "select", colSpan: 12, options: driverOptions, props: {onChange: handleDriverChange}},
  ...(!form.source_name ? [
    {prop: "host", label: t("bi.source.field.host"), component: "input" as const, colSpan: 12, props: {maxlength: 255}},
    {prop: "port", label: t("bi.source.field.port"), component: "input-number" as const, colSpan: 12, props: {min: 1, max: 65535, precision: 0, controlsPosition: "right"}},
    {prop: "database_name", label: t("bi.source.field.database_name"), component: "input" as const, colSpan: 12, props: {maxlength: 128}},
    {prop: "username", label: t("bi.source.field.username"), component: "input" as const, colSpan: 12, props: {maxlength: 128, autocomplete: "off"}},
    {prop: "password", label: t("bi.source.field.password"), component: "input" as const, colSpan: 12, secret: {resource: "bi_source", field: "password"}, props: {type: "password", showPassword: true, maxlength: 512, autocomplete: "new-password", placeholder: form.password_configured ? t("bi.source.placeholder.keep_password") : t("bi.source.placeholder.password")}},
    {prop: "ssl_mode", label: t("bi.source.field.ssl_mode"), component: "input" as const, colSpan: 12, props: {maxlength: 32, placeholder: "disable"}}
  ] : []),
  {prop: "schema_name", label: t("bi.source.field.schema_name"), component: "input", colSpan: 12, props: {maxlength: 128}},
  {prop: "max_rows", label: t("bi.source.field.max_rows"), component: "input-number", colSpan: 12, props: {min: 1, max: 1000, precision: 0, controlsPosition: "right"}}
]);

const rules = computed(() => ({
  code: [{required: true, message: t("bi.source.validation.code"), trigger: "blur"}],
  name: [{required: true, message: t("bi.source.validation.name"), trigger: "blur"}],
  driver: [{required: true, message: t("bi.source.validation.driver"), trigger: "change"}],
  host: [{required: !form.source_name, message: t("bi.source.validation.host"), trigger: "blur"}],
  database_name: [{required: !form.source_name, message: t("bi.source.validation.database_name"), trigger: "blur"}],
  username: [{required: !form.source_name, message: t("bi.source.validation.username"), trigger: "blur"}],
  password: [{required: !form.source_name && !dialog.editing, message: t("bi.source.validation.password"), trigger: "blur"}]
}));

const columns = computed<ColumnProps[]>(() => [
  {prop: "code", label: t("bi.source.field.code"), minWidth: 150, search: {el: "input"}},
  {prop: "name", label: t("bi.source.field.name"), minWidth: 140},
  {prop: "driver", label: t("bi.source.field.driver"), width: 110},
  {prop: "source_name", label: t("bi.source.field.connection"), minWidth: 190, render: ({row}) => row.host ? `${row.host}:${row.port}/${row.database_name}` : row.source_name},
  {prop: "table_count", label: t("bi.source.model.field.tables"), width: 120, render: ({row}) => row.model_configured ? String(row.table_count) : t("bi.source.value.not_configured")},
  {prop: "password_configured", label: t("bi.source.field.password"), width: 120, render: ({row}) => row.password_configured ? t("bi.source.value.configured") : t("bi.source.value.not_configured")},
  {prop: "status", label: t("common.field.status"), width: 100, cellType: "status", statusProps: {activeValue: 1, inactiveValue: 2, activeText: t("common.status.enabled"), inactiveText: t("common.status.disabled"), disabled: scope => !BUTTONS.value["bi:source:status"] || !(scope.row as BiSource).model_configured, beforeChange: async scope => { await setStatus(scope.row as BiSource); return true; }}},
  {prop: "updated_at", label: t("common.field.updated_at"), minWidth: 175},
  {prop: "operation", label: t("common.field.operation"), cellType: "actions", fixed: "right", width: 250, actions: [
    {label: t("bi.source.model.action.configure"), link: true, icon: SetUp, hidden: () => !BUTTONS.value["bi:source:update"], onClick: scope => openModel((scope.row as BiSource).id)},
    {label: t("common.action.edit"), link: true, icon: EditPen, hidden: () => !BUTTONS.value["bi:source:update"], onClick: scope => openDialog((scope.row as BiSource).id)},
    {label: t("common.action.delete"), type: "danger", link: true, icon: Delete, hidden: () => !BUTTONS.value["bi:source:delete"], onClick: scope => remove(scope.row as BiSource)}
  ]}
]);

const headerActions = computed<HeaderActionProps[]>(() => [{label: t("common.action.create"), type: "primary", icon: CirclePlus, hidden: () => !BUTTONS.value["bi:source:create"], onClick: () => openDialog()}]);

/** 创建空的数据源连接表单。 */
function defaultForm(): BiSourceFormState {
  return {id: 0, code: "", name: "", description: "", source_name: "", driver: "mysql", host: "", port: 3306, database_name: "", username: "", password: undefined, password_configured: false, ssl_mode: "disable", schema_name: "", max_rows: 100, status: 2};
}

/** 根据所选数据库驱动更新默认端口。 */
function handleDriverChange(driver: string) {
  form.port = driverDefaultPorts[driver] ?? 3306;
}

/** 请求数据源列表。 */
async function requestTable(params: Record<string, unknown>) {
  const data = await defBiSourceService.PageBiSource(buildPageRequest<PageBiSourceRequest>(params as unknown as PageBiSourceRequest));
  return {data: {list: data.sources ?? [], total: data.total}};
}

/** 打开连接表单，普通租户的已有宿主绑定需要改为独立连接。 */
async function openDialog(id?: number) {
  const source = id ? await defBiSourceService.GetBiSource({id}) : undefined;
  await dialogRef.value?.open({
    load: () => source,
    commit: value => {
      Object.assign(form, defaultForm(), value ?? {});
      if (!isDefaultTenant.value) form.source_name = "";
      dialog.editing = Boolean(id);
      editingModelConfigured.value = Boolean((value as BiSource | undefined)?.model_configured);
    }
  });
}

/** 保存连接配置，连接变化导致模型失效时重新配置。 */
async function submit() {
  const valid = await dialogRef.value?.validate();
  if (!valid) return;
  const encoded = await encodeSecretFields(fields.value, form);
  const {password_configured: _passwordConfigured, ...source} = encoded as BiSourceFormState;
  const result = dialog.editing ? await defBiSourceService.UpdateBiSource({source}) : await defBiSourceService.CreateBiSource({source});
  ElMessage.success(t(dialog.editing ? "bi.source.message.update_success" : "bi.source.message.create_success"));
  dialog.visible = false;
  resetForm();
  table.value?.getTableList();
  if (editingModelConfigured.value && !result.model_configured) await openModel(result);
}

/** 测试表单中的连接，不要求先选择表。 */
async function testSource() {
  const valid = await dialogRef.value?.validate();
  if (!valid) return;
  testing.value = true;
  try {
    const encoded = await encodeSecretFields(fields.value, form);
    const {password_configured: _passwordConfigured, ...source} = encoded as BiSourceFormState;
    const result = await defBiSourceService.TestBiSource({source});
    ElMessage[result.success ? "success" : "warning"](result.message || t(result.success ? "bi.source.message.test_success" : "bi.source.message.test_failed"));
  } finally {
    testing.value = false;
  }
}

/** 打开表结构与关系模型配置。 */
async function openModel(sourceOrID: BiSource | number) {
  modelSource.value = typeof sourceOrID === "number" ? await defBiSourceService.GetBiSource({id: sourceOrID}) : sourceOrID;
  modelDialog.value = true;
}

/** 保存模型后刷新连接列表。 */
function handleModelSaved(source: BiSource) {
  modelSource.value = source;
  table.value?.getTableList();
}

/** 切换数据源启用状态。 */
async function setStatus(row: BiSource) {
  await defBiSourceService.SetBiSourceStatus({id: row.id, status: row.status === 1 ? 2 : 1});
  table.value?.getTableList();
}

/** 删除数据源连接。 */
async function remove(row: BiSource) {
  await ElMessageBox.confirm(t("bi.source.dialog.delete_confirm", {name: row.name}), t("common.title.warning"));
  await defBiSourceService.DeleteBiSource({id: row.id});
  ElMessage.success(t("bi.source.message.delete_success"));
  table.value?.getTableList();
}

/** 重置表单和校验状态。 */
function resetForm() {
  Object.assign(form, defaultForm());
  dialog.editing = false;
  editingModelConfigured.value = false;
}
</script>
