import type { FormRules } from "element-plus";
import { t } from "@liujitcn/kratos-admin-core";
import type { ProFormComponentType, ProFormOption } from "@liujitcn/kratos-admin-core/components/ProForm/interface";
import type {
  CodeGenColumnFormConfig,
  CodeGenColumnListConfig,
  CodeGenColumnOptionConfig,
  CodeGenColumnQueryConfig
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/code_gen_column";
import type { CodeGenLeftTreeConfig, CodeGenTableForm } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/code_gen_table";
import { CodeGenTableStatus } from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/code_gen_table";

/** 代码生成表配置页面类型选项。 */
export function codeGenPageTypeOptions(): ProFormOption[] {
  return [
    { label: t("system.code.gen.page_type.normal"), value: "normal" },
    { label: t("system.code.gen.page_type.tree"), value: "tree" },
    { label: t("system.code.gen.page_type.tree_lazy"), value: "tree_lazy" },
    { label: t("system.code.gen.page_type.left_tree"), value: "left_tree" }
  ];
}

/** 判断页面类型是否使用树形表格配置。 */
export function isCodeGenTreePageType(pageType?: string) {
  return pageType === "tree" || pageType === "tree_lazy";
}

/** 左树数据源类型选项。 */
export function codeGenSourceTypeOptions(): ProFormOption[] {
  return [
    { label: t("system.code.gen.source.static"), value: "static" },
    { label: t("system.code.gen.source.dict"), value: "dict" },
    { label: t("system.code.gen.source.table"), value: "table" }
  ];
}

/** 查询操作符选项。 */
export function codeGenQueryOperatorOptions(): ProFormOption[] {
  return [
    { label: t("system.code.gen.operator.eq"), value: "eq" },
    { label: t("system.code.gen.operator.like"), value: "like" },
    { label: t("system.code.gen.operator.between"), value: "between" }
  ];
}

/** 查询组件选项。 */
export function codeGenQueryComponentOptions(): ProFormOption[] {
  return [
    { label: t("system.code.gen.component.input"), value: "input" },
    { label: t("system.code.gen.component.input_number"), value: "input-number" },
    { label: t("system.code.gen.component.select"), value: "select" },
    { label: t("system.code.gen.component.tree_select"), value: "tree-select" },
    { label: t("system.code.gen.component.date_picker"), value: "date-picker" }
  ];
}

/** 列表展示组件选项。 */
export function codeGenListComponentOptions(): ProFormOption[] {
  return [
    { label: t("system.code.gen.component.text"), value: "text" },
    { label: t("system.code.gen.component.switch"), value: "switch" },
    { label: t("system.code.gen.component.select_short"), value: "select" },
    { label: t("system.code.gen.component.tree"), value: "tree-select" },
    { label: t("system.code.gen.component.image"), value: "image" },
    { label: t("system.code.gen.component.money"), value: "money" },
    { label: t("system.code.gen.component.date"), value: "date" }
  ];
}

/** ProForm 全量组件类型对应的中文名称。 */
const codeGenFormComponentLabelKeys: Record<ProFormComponentType, string> = {
  input: "system.code.gen.component.input",
  password: "system.code.gen.component.password",
  textarea: "system.code.gen.component.textarea",
  "input-number": "system.code.gen.component.input_number",
  "color-picker": "system.code.gen.component.color_picker",
  segmented: "system.code.gen.component.segmented",
  switch: "system.code.gen.component.switch",
  checkbox: "system.code.gen.component.checkbox",
  select: "system.code.gen.component.select",
  dict: "system.code.gen.component.dict",
  "radio-group": "system.code.gen.component.radio_group",
  "checkbox-group": "system.code.gen.component.checkbox_group",
  "tree-select": "system.code.gen.component.tree_select",
  "date-picker": "system.code.gen.component.date_picker",
  "cron-expression": "system.code.gen.component.cron",
  transfer: "system.code.gen.component.transfer",
  "image-upload": "system.code.gen.component.image_upload",
  "images-upload": "system.code.gen.component.images_upload",
  "file-upload": "system.code.gen.component.file_upload",
  "files-upload": "system.code.gen.component.files_upload",
  "rich-text": "system.code.gen.component.rich_text",
  "dynamic-list": "system.code.gen.component.dynamic_list",
  "kv-list": "system.code.gen.component.kv_list",
  slot: "system.code.gen.component.slot"
};

/** 表单录入组件选项，保持与 ProForm 支持类型完整一致。 */
export function codeGenFormComponentOptions(): ProFormOption[] {
  return (Object.entries(codeGenFormComponentLabelKeys) as Array<[ProFormComponentType, string]>).map(([value, labelKey]) => ({
    label: t(labelKey),
    value
  }));
}

/** 代码生成表配置校验规则。 */
export function codeGenTableRules(): FormRules {
  return {
    source_name: [{ required: true, message: t("system.code.gen.validation.source_required"), trigger: "change" }],
    name: [{ required: true, max: 128, message: t("system.code.gen.validation.table_required"), trigger: "change" }],
    business_module: [
      {
        required: true,
        max: 64,
        pattern: /^[a-z][a-z0-9_]*$/,
        message: t("system.code.gen.validation.module_required"),
        trigger: "change"
      }
    ],
    comment: [{ max: 128, message: t("system.code.gen.validation.comment_length"), trigger: "blur" }],
    parent_menu_id: [
      { required: true, type: "number", min: 1, message: t("system.code.gen.validation.parent_menu_required"), trigger: "change" }
    ],
    page_type: [{ required: true, max: 32, message: t("system.code.gen.validation.page_type_required"), trigger: "change" }],
    parent_column: [{ required: true, max: 64, message: t("system.code.gen.validation.parent_column_required"), trigger: "change" }],
    tree_label_column: [
      { required: true, max: 64, message: t("system.code.gen.validation.tree_label_required"), trigger: "change" }
    ],
    remark: [{ max: 500, message: t("system.code.gen.validation.remark_length"), trigger: "blur" }],
    "left_tree_config.table_name": [
      { required: true, message: t("system.code.gen.validation.left_tree_table_required"), trigger: "change" }
    ],
    "left_tree_config.filter_column": [
      { required: true, message: t("system.code.gen.validation.filter_column_required"), trigger: "change" }
    ],
    "left_tree_config.parent_column": [
      { required: true, message: t("system.code.gen.validation.left_tree_parent_required"), trigger: ["blur", "change"] }
    ],
    "left_tree_config.label_column": [
      { required: true, message: t("system.code.gen.validation.left_tree_label_required"), trigger: ["blur", "change"] }
    ],
    "left_tree_config.value_column": [
      { required: true, message: t("system.code.gen.validation.left_tree_value_required"), trigger: ["blur", "change"] }
    ]
  };
}

/** 创建默认左树右表页面配置。 */
export function createDefaultCodeGenLeftTreeConfig(): CodeGenLeftTreeConfig {
  return {
    table_name: "",
    filter_column: "",
    parent_column: "",
    label_column: "",
    value_column: "",
    comment: "",
    lazy: false
  };
}

/** 创建代码生成表配置默认表单。 */
export function createDefaultCodeGenTableForm(): CodeGenTableForm {
  return {
    id: 0,
    source_name: "",
    name: "",
    comment: "",
    business_module: "",
    page_type: "normal",
    parent_column: "",
    tree_label_column: "",
    left_tree_config: createDefaultCodeGenLeftTreeConfig(),
    gen_backend: true,
    gen_frontend: true,
    gen_sql: true,
    parent_menu_id: 0,
    status: CodeGenTableStatus.CODE_GEN_TABLE_STATUS_DRAFT,
    remark: "",
    i18n_config: new Map()
  };
}

/** 创建默认字段查询配置。 */
export function createDefaultCodeGenQueryConfig(): CodeGenColumnQueryConfig {
  return { enabled: false, operator: "like", component: "input", option: createDefaultCodeGenOptionConfig() };
}

/** 创建默认字段列表配置。 */
export function createDefaultCodeGenListConfig(): CodeGenColumnListConfig {
  return {
    enabled: true,
    component: "text",
    option: createDefaultCodeGenOptionConfig()
  };
}

/** 创建默认字段表单配置。 */
export function createDefaultCodeGenFormConfig(): CodeGenColumnFormConfig {
  return { enabled: true, component: "input", required: false, multiple: false, option: createDefaultCodeGenOptionConfig() };
}

/** 创建一份独立的字段选项配置。 */
export function createDefaultCodeGenOptionConfig(): CodeGenColumnOptionConfig {
  return {
    kind: "",
    source_type: "",
    source_value: "",
    label_field: "",
    value_field: "",
    parent_field: "",
    active_value: "",
    inactive_value: "",
    lazy: false
  };
}
