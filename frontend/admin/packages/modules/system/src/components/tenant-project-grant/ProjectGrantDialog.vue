<template>
  <FormDialog
    ref="formDialogRef"
    v-model="visible"
    :title="t('system.base.tenant_project.grant')"
    :model="form"
    :fields="fields"
    :rules="rules"
    width="640px"
    @confirm="save"
    @close="reset"
  />
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { t } from "@liujitcn/kratos-admin-core";
import { useTenantScope } from "@liujitcn/kratos-admin-core/tenant";
import { defBaseTenantProjectService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_tenant_project";
import { defBaseTenantProjectGrantService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/base_tenant_project_grant";
import type {
  BaseTenantProjectGrant,
  BaseTenantProjectGrantSubjectType
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/base_tenant_project_grant";
import type { SelectOptionResponse_Option } from "@liujitcn/kratos-admin-system/rpc/common/v1/common";

const visible = ref(false);
const formDialogRef = ref<InstanceType<typeof FormDialog>>();
const loading = ref(false);
const projectOptions = ref<SelectOptionResponse_Option[]>([]);
const form = reactive({ tenant_id: 0, all: false, project_id: [] as number[] });
const { tenantFormField, loadTenantOptions } = useTenantScope();
let subjectType: BaseTenantProjectGrantSubjectType;
let subjectId = 0;
let revision = 0;
const rules = computed(() => ({ tenant_id: [{ required: true, type: "number", min: 1, message: t("common.validation.required_select", { field: t("common.field.tenant") }), trigger: "change" }] }));
const fields = computed<ProFormField[]>(() => [
  tenantFormField({ label: t("common.field.tenant") }),
  { prop: "all", label: t("system.base.tenant_project.all"), component: "switch", props: { disabled: loading.value } },
  { prop: "project_id", label: t("system.base.tenant_project.projects"), component: "select", options: projectOptions.value, visible: () => !form.all, props: { multiple: true, filterable: true, disabled: loading.value } }
]);

/** 打开指定主体的授权表单，目标租户切换后重新加载直接授权。 */
async function open(type: BaseTenantProjectGrantSubjectType, id: number, tenantId: number) {
  await formDialogRef.value?.open({
    load: async () => {
      await loadTenantOptions(true);
      const request = { tenant_id: tenantId, subject_type: type, subject_id: id };
      const [options, grant] = await Promise.all([
        defBaseTenantProjectService.OptionBaseTenantProject({ tenant_id: request.tenant_id }),
        defBaseTenantProjectGrantService.GetBaseTenantProjectGrant(request)
      ]);
      return { request, projects: options.list ?? [], grant };
    },
    commit: ({ request, projects, grant }) => {
      reset();
      subjectType = request.subject_type;
      subjectId = request.subject_id;
      form.tenant_id = request.tenant_id;
      projectOptions.value = projects;
      form.all = grant.project_id.length === 1 && grant.project_id[0] === 0;
      form.project_id = form.all ? [] : grant.project_id;
    }
  });
}

/** 加载目标租户项目和当前主体的直接授权，避免快速切换时旧响应覆盖新结果。 */
async function load() {
  const current = ++revision;
  form.all = false;
  form.project_id = [];
  projectOptions.value = [];
  if (!visible.value || form.tenant_id <= 0) return;
  loading.value = true;
  try {
    const request = { tenant_id: form.tenant_id, subject_type: subjectType, subject_id: subjectId };
    const [options, grant] = await Promise.all([
      defBaseTenantProjectService.OptionBaseTenantProject({ tenant_id: request.tenant_id }),
      defBaseTenantProjectGrantService.GetBaseTenantProjectGrant(request)
    ]);
    if (current !== revision) return;
    projectOptions.value = options.list ?? [];
    form.all = grant.project_id.length === 1 && grant.project_id[0] === 0;
    form.project_id = form.all ? [] : grant.project_id;
  } finally {
    if (current === revision) loading.value = false;
  }
}

/** 保存当前来源的授权，其他三个来源保持独立。 */
async function save() {
  if (loading.value) return;
  const grant: BaseTenantProjectGrant = {
    tenant_id: form.tenant_id,
    subject_type: subjectType,
    subject_id: subjectId,
    project_id: form.all ? [0] : form.project_id,
    tenant_name: "",
    subject_name: "",
    subject_code: "",
    project_names: [],
    grant_key: ""
  };
  await defBaseTenantProjectGrantService.UpdateBaseTenantProjectGrant({ base_tenant_project_grant: grant });
  visible.value = false;
}

/** 关闭后清空主体及项目，阻止未完成请求覆盖下次表单。 */
function reset() {
  ++revision;
  visible.value = false;
  subjectId = 0;
  form.tenant_id = 0;
  form.all = false;
  form.project_id = [];
  loading.value = false;
}

watch(() => form.tenant_id, () => { if (visible.value) void load(); });
defineExpose({ open });
</script>
