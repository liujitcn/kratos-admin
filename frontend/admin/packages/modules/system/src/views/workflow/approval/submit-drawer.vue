<template>
  <el-drawer
    v-model="visible"
    :title="t('system.workflow.approval.title.submit')"
    size="480px"
    destroy-on-close
    :close-on-click-modal="false"
    @closed="resetState"
  >
    <el-form ref="formRef" :model="submitForm" :rules="formRules" label-width="110px">
      <el-form-item :label="t('system.workflow.definition.resource')" prop="definition">
        <el-select v-model="submitForm.definition" filterable style="width: 100%" @change="handleDefinitionChange">
          <el-option v-for="item in definitionOptions" :key="Number(item.value)" :label="item.label" :value="Number(item.value)" />
        </el-select>
      </el-form-item>

      <template v-if="fields.length">
        <el-form-item
          v-for="field in fields"
          :key="field.key"
          :label="field.label"
          :prop="`form.${field.key}`"
          :rules="field.required ? [{ required: true, message: t('common.validation.required_input', { field: field.label }), trigger: 'blur' }] : undefined"
        >
          <el-input v-if="field.type === 'text'" v-model="submitForm.form[field.key]" />
          <el-input v-else-if="field.type === 'textarea'" v-model="submitForm.form[field.key]" type="textarea" :rows="3" />
          <el-input-number
            v-else-if="field.type === 'number'"
            :model-value="Number(submitForm.form[field.key] ?? 0)"
            @update:model-value="value => (submitForm.form[field.key] = value)"
            controls-position="right"
            style="width: 100%"
          />
          <el-date-picker v-else-if="field.type === 'date'" v-model="submitForm.form[field.key]" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
          <el-select v-else-if="field.type === 'select'" v-model="submitForm.form[field.key]" clearable style="width: 100%">
            <el-option v-for="option in field.options" :key="option" :label="option" :value="option" />
          </el-select>
          <el-input v-else v-model="submitForm.form[field.key]" />
        </el-form-item>
      </template>
      <el-form-item v-else-if="definitionLoaded" :label="t('system.workflow.approval.field.form_data')">
        <el-input v-model="formJsonText" type="textarea" :rows="6" :placeholder="t('system.workflow.approval.placeholder.form_json')" />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="visible = false">{{ t("common.action.cancel") }}</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">{{ t("common.action.confirm") }}</el-button>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";
import type { FormInstance, FormRules } from "element-plus";
import { ElMessage } from "element-plus";
import { t } from "@liujitcn/kratos-admin-core";
import type { ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import { defWorkflowDefinitionService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_definition";
import { defWorkflowInstanceService } from "@liujitcn/kratos-admin-system/api/system/admin/v1/workflow_instance";

defineOptions({ name: "WorkflowSubmitDrawer" });

const emit = defineEmits<{ submitted: [] }>();

/** 表单字段描述，来自流程定义 form_schema。 */
interface FormFieldDesc {
  /** 字段键名。 */
  key: string;
  /** 展示标签。 */
  label: string;
  /** 组件类型。 */
  type: "text" | "textarea" | "number" | "date" | "select";
  /** 是否必填。 */
  required: boolean;
  /** 下拉选项，逗号分隔字符串。 */
  options: string[];
}

const visible = ref(false);
const submitting = ref(false);
const definitionLoaded = ref(false);
const formRef = ref<FormInstance>();
const definitionOptions = ref<ProFormOption[]>([]);
const fields = ref<FormFieldDesc[]>([]);
const formJsonText = ref("");

/** 下拉值结构：`id:code:version`，提交时拆开。 */
const submitForm = reactive({
  /** 选中的流程定义 ID。 */
  definition: undefined as number | undefined,
  /** 表单数据。 */
  form: {} as Record<string, string | number | undefined>
});

const formRules = computed<FormRules>(() => ({
  definition: [
    {
      required: true,
      message: t("common.validation.required_select", { field: t("system.workflow.definition.resource") }),
      trigger: "change"
    }
  ]
}));

/** 打开发起申请抽屉。 */
async function open() {
  resetState();
  const data = await defWorkflowDefinitionService.OptionWorkflowDefinition({});
  definitionOptions.value = data.list ?? [];
  visible.value = true;
}

defineExpose({ open });

/** 选中定义后查询编码版本并加载申请表单定义。 */
async function handleDefinitionChange(definitionId: number) {
  definitionLoaded.value = false;
  fields.value = [];
  formJsonText.value = "";
  submitForm.form = {};
  selectedDefinition.value = null;
  if (!definitionId) return;
  const form = await defWorkflowDefinitionService.GetWorkflowDefinition({ id: definitionId });
  selectedDefinition.value = { code: form.code, version: form.version };
  const applyForm = await defWorkflowDefinitionService.GetWorkflowApplyForm({ code: form.code, version: form.version });
  definitionLoaded.value = true;
  if (applyForm.form_schema) {
    try {
      const parsed = JSON.parse(applyForm.form_schema) as Array<Partial<FormFieldDesc>>;
      fields.value = parsed
        .filter(item => !!item.key)
        .map(item => ({
          key: String(item.key),
          label: item.label ?? item.key ?? "",
          type: (item.type ?? "text") as FormFieldDesc["type"],
          required: !!item.required,
          options: parseFieldOptions(item.options)
        }));
    } catch {
      ElMessage.warning(t("system.workflow.definition.error.form_schema_json"));
    }
  }
}

/** 解析表单字段下拉选项，逗号分隔字符串。 */
function parseFieldOptions(raw: unknown): string[] {
  return typeof raw === "string" && raw ? raw.split(",").map(option => option.trim()).filter(Boolean) : [];
}

/** 当前选中定义的编码与版本。 */
const selectedDefinition = ref<{ code: string; version: number } | null>(null);

/** 提交申请。 */
function handleSubmit() {
  formRef.value?.validate()?.then(valid => {
    if (!valid || !selectedDefinition.value) return;
    let formData = formJsonText.value;
    if (fields.value.length) {
      formData = JSON.stringify(submitForm.form);
    } else if (!formData) {
      formData = "{}";
    }
    submitting.value = true;
    defWorkflowInstanceService
      .SubmitWorkflowInstance({
        code: selectedDefinition.value.code,
        version: selectedDefinition.value.version,
        form_data: formData,
        business_type: "",
        business_id: 0
      })
      .then(() => {
        ElMessage.success(t("system.workflow.approval.message.submitted"));
        visible.value = false;
        emit("submitted");
      })
      .finally(() => {
        submitting.value = false;
      });
  });
}

/** 重置抽屉状态。 */
function resetState() {
  definitionLoaded.value = false;
  fields.value = [];
  formJsonText.value = "";
  submitForm.definition = undefined;
  submitForm.form = {};
  selectedDefinition.value = null;
}
</script>
