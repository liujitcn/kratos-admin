<template>
  <FormDialog
    ref="dialogRef"
    v-model="visible"
    :title="t('system.workflow.approval.title.submit')"
    width="640px"
    label-width="110px"
    :model="submitForm"
    :fields="formFields"
    :form-props="{ disabled: submitting }"
    :confirm-loading="submitting"
    destroy-on-close
    :close-on-click-modal="false"
    @confirm="handleSubmit"
    @close="resetState"
  />
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import { t } from "@liujitcn/kratos-admin-core";
import FormDialog from "@liujitcn/kratos-admin-core/components/Dialog/FormDialog.vue";
import type { ProFormField, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { defWorkflowDefinitionService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_definition";
import { defWorkflowInstanceService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_instance";
import { getFormSchemaOptions, type FormSchemaField } from "../definition/form-schema";

defineOptions({ name: "WorkflowSubmitDialog" });

const emit = defineEmits<{ submitted: [] }>();
const dialogRef = ref<InstanceType<typeof FormDialog>>();
const visible = ref(false);
const submitting = ref(false);
let definitionRequestRevision = 0;
const definitionOptions = ref<ProFormOption[]>([]);
const selectedDefinition = ref<{ code: string; version: number } | null>(null);
const fields = ref<FormSchemaField[]>([]);
const submitForm = reactive({
  definition: undefined as number | undefined,
  business_type: "",
  business_id: 0,
  values: [] as (string | number | undefined)[],
  form_json: ""
});

const untrimmedInput = { modelModifiers: { trim: false } };
const numberProps = { controlsPosition: "right", style: { width: "100%" } };
const formFields = computed<ProFormField[]>(() => {
  const definitionLabel = t("system.workflow.definition.resource");
  return [
    {
      prop: "definition",
      label: definitionLabel,
      component: "select",
      options: definitionOptions.value,
      props: { filterable: true, onChange: handleDefinitionChange },
      rules: [{ required: true, message: t("common.validation.required_select", { field: definitionLabel }), trigger: "change" }]
    },
    {
      prop: "business_type",
      label: t("system.workflow.approval.field.business_type"),
      component: "input",
      props: { ...untrimmedInput, maxlength: 50, placeholder: t("system.workflow.approval.placeholder.business_type") }
    },
    {
      prop: "business_id",
      label: t("system.workflow.approval.field.business_id"),
      component: "input-number",
      props: { ...numberProps, min: 0 }
    },
    ...fields.value.map<ProFormField>((field, index) => {
      const input: ProFormField = {
        // 用字段位置绑定控件，提交时按流程定义的原始键组装数据。
        prop: `values.${index}`,
        label: field.label,
        component: "input",
        props: untrimmedInput,
        rules: field.required
          ? [{ required: true, message: t("common.validation.required_input", { field: field.label }), trigger: "blur" }]
          : undefined
      };
      switch (field.type) {
        case "textarea":
          return { ...input, component: "textarea", props: { ...untrimmedInput, rows: 3 } };
        case "number":
          return { ...input, component: "input-number", props: () => ({ ...numberProps, modelValue: submitForm.values[index] ?? 0 }) };
        case "date":
          return { ...input, component: "date-picker", props: { type: "date", valueFormat: "YYYY-MM-DD", style: { width: "100%" } } };
        case "select":
          return {
            ...input,
            component: "select",
            props: { clearable: true },
            options: getFormSchemaOptions(field.options).map(value => ({ label: value, value }))
          };
        default:
          return input;
      }
    }),
    {
      prop: "form_json",
      label: t("system.workflow.approval.field.form_data"),
      component: "textarea",
      props: { ...untrimmedInput, rows: 6, placeholder: t("system.workflow.approval.placeholder.form_json") },
      visible: () => Boolean(selectedDefinition.value) && fields.value.length === 0
    }
  ];
});

/** 加载可用流程定义并打开发起申请弹窗。 */
async function open() {
  if (submitting.value) return;
  await dialogRef.value?.open({
    load: () => defWorkflowDefinitionService.OptionWorkflowDefinition({}),
    commit: data => {
      resetState();
      definitionOptions.value = (data.list ?? []).map(item => ({ label: item.label, value: Number(item.value) }));
    }
  });
}

/** 加载选中流程的申请表单，只应用最后一次选择的结果。 */
async function handleDefinitionChange(definitionId: number) {
  const revision = ++definitionRequestRevision;
  submitForm.values = [];
  submitForm.form_json = "";
  fields.value = [];
  selectedDefinition.value = null;
  if (!definitionId) return;
  const { code, version } = await defWorkflowDefinitionService.GetWorkflowDefinition({ id: definitionId });
  if (revision !== definitionRequestRevision) return;
  const { form_schema } = await defWorkflowDefinitionService.GetWorkflowApplyForm({ code, version });
  if (revision !== definitionRequestRevision) return;

  selectedDefinition.value = { code, version };
  if (!form_schema) return;
  try {
    const parsed = JSON.parse(form_schema) as Partial<FormSchemaField>[];
    fields.value = parsed.filter(field => field.key).map(field => ({
      key: String(field.key),
      label: field.label ?? String(field.key),
      type: field.type ?? "text",
      required: Boolean(field.required),
      options: typeof field.options === "string" ? field.options : undefined
    }));
  } catch {
    ElMessage.warning(t("system.workflow.definition.error.form_schema_json"));
  }
}

/** 提交校验通过的申请并刷新审批列表。 */
async function handleSubmit() {
  const definition = selectedDefinition.value;
  if (submitting.value || !definition) return;
  const formData = fields.value.length
    ? JSON.stringify(Object.fromEntries(fields.value.map((field, index) => [field.key, submitForm.values[index]])))
    : submitForm.form_json || "{}";
  submitting.value = true;
  try {
    await defWorkflowInstanceService.SubmitWorkflowInstance({
      code: definition.code,
      version: definition.version,
      form_data: formData,
      business_type: submitForm.business_type.trim(),
      business_id: submitForm.business_id
    });
    ElMessage.success(t("system.workflow.approval.message.submitted"));
    visible.value = false;
    emit("submitted");
  } finally {
    submitting.value = false;
  }
}

/** 重置申请草稿并使未完成的表单定义请求失效。 */
function resetState() {
  definitionRequestRevision += 1;
  Object.assign(submitForm, { definition: undefined, business_type: "", business_id: 0, values: [], form_json: "" });
  fields.value = [];
  selectedDefinition.value = null;
}

defineExpose({ open });
</script>
