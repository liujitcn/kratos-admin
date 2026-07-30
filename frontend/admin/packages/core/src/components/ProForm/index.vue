<template>
  <el-form ref="formRef" :model="model" :rules="formRules" :label-width="labelWidth" v-bind="$attrs">
    <el-row :gutter="gutter">
      <template v-for="field in visibleFields" :key="field.prop">
        <span v-if="field.rowBreakBefore" class="pro-form__row-break" aria-hidden="true" />
        <el-col :span="field.colSpan ?? colSpan">
          <el-form-item :label="field.label" :prop="field.prop" :rules="field.rules" v-bind="resolveFieldItemProps(field)">
            <template v-if="field.labelTooltip" #label>
              <div class="pro-form__label">
                <span>{{ field.label }}</span>
                <el-tooltip placement="bottom" effect="light">
                  <template #content>{{ field.labelTooltip }}</template>
                  <el-icon class="pro-form__label-icon">
                    <QuestionFilled />
                  </el-icon>
                </el-tooltip>
              </div>
            </template>
            <ProFormItem :field="field" :model="model">
              <template v-for="slotName in slotNames" #[slotName]="slotScope">
                <slot :name="slotName" v-bind="slotScope" />
              </template>
            </ProFormItem>
          </el-form-item>
        </el-col>
      </template>
    </el-row>
  </el-form>
</template>

<script setup lang="ts" name="ProForm">
import { computed, ref, useSlots } from "vue";
import { QuestionFilled } from "@element-plus/icons-vue";
import type { FormInstance, FormRules } from "element-plus";
import type { ProFormField } from "@/components/ProForm/interface";
import ProFormItem from "./components/ProFormItem.vue";

/** ProForm 组件属性。 */
interface ProFormProps {
  model: Record<string, any>;
  fields: ProFormField[];
  rules?: FormRules;
  labelWidth?: string;
  gutter?: number;
  colSpan?: number;
}

const props = withDefaults(defineProps<ProFormProps>(), {
  rules: () => ({}),
  labelWidth: "110px",
  gutter: 20,
  colSpan: 24
});
defineSlots<Record<string, (props: any) => any>>();

const slots = useSlots();
const formRef = ref<FormInstance>();

/** 计算当前可见的表单字段。 */
const visibleFields = computed(() => {
  return props.fields.filter(field => (field.visible ? field.visible(props.model) : true));
});

/** 解析字段级表单项参数，支持静态对象和函数。 */
function resolveFieldItemProps(field: ProFormField) {
  if (!field.itemProps) return {};
  return typeof field.itemProps === "function" ? field.itemProps(props.model) : field.itemProps;
}

/** 汇总外部传入的插槽名称，便于向下透传。 */
const slotNames = computed(() => Object.keys(slots));

/** 合并表单规则，优先使用字段自身规则。 */
const formRules = computed(() => {
  const mergedRules: FormRules = { ...props.rules };
  visibleFields.value.forEach(field => {
    if (!field.rules?.length) return;
    mergedRules[field.prop] = field.rules;
  });
  return mergedRules;
});

/** 校验表单。 */
async function validate() {
  if (!formRef.value) return false;
  return formRef.value.validate().catch(() => false);
}

/** 重置表单。 */
function resetFields() {
  formRef.value?.resetFields();
}

/** 清理表单校验状态。 */
function clearValidate(props?: string | string[]) {
  formRef.value?.clearValidate(props);
}

defineExpose({
  validate,
  resetFields,
  clearValidate
});
</script>

<style scoped lang="scss">
.pro-form__row-break {
  flex: 0 0 100%;
  height: 0;
}
.pro-form__label {
  display: inline-flex;
  gap: 4px;
  align-items: center;
}
.pro-form__label-icon {
  color: var(--el-color-primary);
  cursor: pointer;
}
</style>
