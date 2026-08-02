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
import { CodeGenTableStatus } from "@liujitcn/kratos-admin-system/rpc/system/common/v1/enum";

/** 代码生成表配置页面类型选项。 */
export function codeGenPageTypeOptions(): ProFormOption[] {
  return [
    { label: t("system.codegen.pageType.normal"), value: "normal" },
    { label: t("system.codegen.pageType.tree"), value: "tree" },
    { label: t("system.codegen.pageType.treeLazy"), value: "tree_lazy" },
    { label: t("system.codegen.pageType.leftTree"), value: "left_tree" }
  ];
}

/** 判断页面类型是否使用树形表格配置。 */
export function isCodeGenTreePageType(pageType?: string) {
  return pageType === "tree" || pageType === "tree_lazy";
}

/** 左树数据源类型选项。 */
export function codeGenSourceTypeOptions(): ProFormOption[] {
  return [
    { label: t("system.codegen.source.static"), value: "static" },
    { label: t("system.codegen.source.dict"), value: "dict" },
    { label: t("system.codegen.source.table"), value: "table" }
  ];
}

/** 查询操作符选项。 */
export function codeGenQueryOperatorOptions(): ProFormOption[] {
  return [
    { label: t("system.codegen.operator.eq"), value: "eq" },
    { label: t("system.codegen.operator.like"), value: "like" },
    { label: t("system.codegen.operator.between"), value: "between" }
  ];
}

/** 查询组件选项。 */
export function codeGenQueryComponentOptions(): ProFormOption[] {
  return [
    { label: t("system.codegen.component.input"), value: "input" },
    { label: t("system.codegen.component.inputNumber"), value: "input-number" },
    { label: t("system.codegen.component.select"), value: "select" },
    { label: t("system.codegen.component.treeSelect"), value: "tree-select" },
    { label: t("system.codegen.component.datePicker"), value: "date-picker" }
  ];
}

/** 列表展示组件选项。 */
export function codeGenListComponentOptions(): ProFormOption[] {
  return [
    { label: t("system.codegen.component.text"), value: "text" },
    { label: t("system.codegen.component.switch"), value: "switch" },
    { label: t("system.codegen.component.selectShort"), value: "select" },
    { label: t("system.codegen.component.tree"), value: "tree-select" },
    { label: t("system.codegen.component.image"), value: "image" },
    { label: t("system.codegen.component.money"), value: "money" },
    { label: t("system.codegen.component.date"), value: "date" }
  ];
}

/** ProForm 全量组件类型对应的中文名称。 */
const codeGenFormComponentLabelKeys: Record<ProFormComponentType, string> = {
  input: "system.codegen.component.input",
  password: "system.codegen.component.password",
  textarea: "system.codegen.component.textarea",
  "input-number": "system.codegen.component.inputNumber",
  segmented: "system.codegen.component.segmented",
  switch: "system.codegen.component.switch",
  checkbox: "system.codegen.component.checkbox",
  select: "system.codegen.component.select",
  dict: "system.codegen.component.dict",
  "radio-group": "system.codegen.component.radioGroup",
  "checkbox-group": "system.codegen.component.checkboxGroup",
  "tree-select": "system.codegen.component.treeSelect",
  "date-picker": "system.codegen.component.datePicker",
  "cron-expression": "system.codegen.component.cron",
  transfer: "system.codegen.component.transfer",
  "image-upload": "system.codegen.component.imageUpload",
  "images-upload": "system.codegen.component.imagesUpload",
  "file-upload": "system.codegen.component.fileUpload",
  "files-upload": "system.codegen.component.filesUpload",
  "rich-text": "system.codegen.component.richText",
  "dynamic-list": "system.codegen.component.dynamicList",
  "kv-list": "system.codegen.component.kvList",
  slot: "system.codegen.component.slot"
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
    name: [{ required: true, max: 128, message: t("system.codegen.validation.tableRequired"), trigger: "change" }],
    business_module: [
      {
        required: true,
        max: 64,
        pattern: /^[a-z][a-z0-9_]*$/,
        message: t("system.codegen.validation.moduleRequired"),
        trigger: "change"
      }
    ],
    comment: [{ max: 128, message: t("system.codegen.validation.commentLength"), trigger: "blur" }],
    parent_menu_id: [
      { required: true, type: "number", min: 1, message: t("system.codegen.validation.parentMenuRequired"), trigger: "change" }
    ],
    page_type: [{ required: true, max: 32, message: t("system.codegen.validation.pageTypeRequired"), trigger: "change" }],
    parent_column: [{ required: true, max: 64, message: t("system.codegen.validation.parentColumnRequired"), trigger: "change" }],
    tree_label_column: [
      { required: true, max: 64, message: t("system.codegen.validation.treeLabelRequired"), trigger: "change" }
    ],
    remark: [{ max: 500, message: t("system.codegen.validation.remarkLength"), trigger: "blur" }],
    "left_tree_config.table_name": [
      { required: true, message: t("system.codegen.validation.leftTreeTableRequired"), trigger: "change" }
    ],
    "left_tree_config.filter_column": [
      { required: true, message: t("system.codegen.validation.filterColumnRequired"), trigger: "change" }
    ],
    "left_tree_config.parent_column": [
      { required: true, message: t("system.codegen.validation.leftTreeParentRequired"), trigger: ["blur", "change"] }
    ],
    "left_tree_config.label_column": [
      { required: true, message: t("system.codegen.validation.leftTreeLabelRequired"), trigger: ["blur", "change"] }
    ],
    "left_tree_config.value_column": [
      { required: true, message: t("system.codegen.validation.leftTreeValueRequired"), trigger: ["blur", "change"] }
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
    status: CodeGenTableStatus.DRAFT_CGTS,
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
