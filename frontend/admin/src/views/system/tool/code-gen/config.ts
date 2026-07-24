import type { FormRules } from "element-plus";
import type { ProFormComponentType, ProFormOption } from "@/components/ProForm/interface";
import type {
  CodeGenColumnFormConfig,
  CodeGenColumnListConfig,
  CodeGenColumnOptionConfig,
  CodeGenColumnQueryConfig
} from "@/rpc/system/admin/v1/code_gen_column";
import type { CodeGenLeftTreeConfig, CodeGenTableForm } from "@/rpc/system/admin/v1/code_gen_table";
import { CodeGenTableStatus } from "@/rpc/system/common/v1/enum";

/** 代码生成表配置页面类型选项。 */
export const codeGenPageTypeOptions: ProFormOption[] = [
  { label: "普通表格", value: "normal" },
  { label: "树形表格", value: "tree" },
  { label: "树形懒加载", value: "tree_lazy" },
  { label: "左树右表", value: "left_tree" }
];

/** 判断页面类型是否使用树形表格配置。 */
export function isCodeGenTreePageType(pageType?: string) {
  return pageType === "tree" || pageType === "tree_lazy";
}

/** 左树数据源类型选项。 */
export const codeGenSourceTypeOptions: ProFormOption[] = [
  { label: "静态数据", value: "static" },
  { label: "字典", value: "dict" },
  { label: "数据表", value: "table" }
];

/** 查询操作符选项。 */
export const codeGenQueryOperatorOptions: ProFormOption[] = [
  { label: "等于", value: "eq" },
  { label: "模糊", value: "like" },
  { label: "区间", value: "between" }
];

/** 查询组件选项。 */
export const codeGenQueryComponentOptions: ProFormOption[] = [
  { label: "输入框", value: "input" },
  { label: "数字输入", value: "input-number" },
  { label: "下拉选择", value: "select" },
  { label: "树形选择", value: "tree-select" },
  { label: "日期", value: "date-picker" }
];

/** 列表展示组件选项。 */
export const codeGenListComponentOptions: ProFormOption[] = [
  { label: "文本", value: "text" },
  { label: "开关", value: "switch" },
  { label: "下拉", value: "select" },
  { label: "树形", value: "tree-select" },
  { label: "图片", value: "image" },
  { label: "金额", value: "money" },
  { label: "日期", value: "date" }
];

/** ProForm 全量组件类型对应的中文名称。 */
const codeGenFormComponentLabels: Record<ProFormComponentType, string> = {
  input: "输入框",
  password: "密码框",
  textarea: "文本域",
  "input-number": "数字输入",
  segmented: "分段选择",
  switch: "开关",
  checkbox: "复选框",
  select: "下拉选择",
  dict: "字典选择",
  "radio-group": "单选组",
  "checkbox-group": "复选组",
  "tree-select": "树形选择",
  "date-picker": "日期选择",
  "cron-expression": "Cron 表达式",
  transfer: "穿梭框",
  "image-upload": "单图上传",
  "images-upload": "多图上传",
  "file-upload": "单文件上传",
  "files-upload": "多文件上传",
  "rich-text": "富文本",
  "dynamic-list": "动态列表",
  "kv-list": "键值列表",
  slot: "自定义插槽"
};

/** 表单录入组件选项，保持与 ProForm 支持类型完整一致。 */
export const codeGenFormComponentOptions: ProFormOption[] = (
  Object.entries(codeGenFormComponentLabels) as Array<[ProFormComponentType, string]>
).map(([value, label]) => ({ label, value }));

/** 代码生成表配置校验规则。 */
export const codeGenTableRules: FormRules = {
  name: [{ required: true, max: 128, message: "请选择业务表", trigger: "change" }],
  business_module: [
    { required: true, max: 64, pattern: /^[a-z][a-z0-9_]*$/, message: "请选择有效业务模块", trigger: "change" }
  ],
  comment: [{ max: 128, message: "业务表描述不能超过128个字符", trigger: "blur" }],
  parent_menu_id: [{ required: true, type: "number", min: 1, message: "请选择父级菜单", trigger: "change" }],
  page_type: [{ required: true, max: 32, message: "请选择页面类型", trigger: "change" }],
  parent_column: [{ required: true, max: 64, message: "请选择父节点字段", trigger: "change" }],
  tree_label_column: [{ required: true, max: 64, message: "请选择树显示字段", trigger: "change" }],
  remark: [{ max: 500, message: "备注不能超过500个字符", trigger: "blur" }],
  "left_tree_config.table_name": [{ required: true, message: "请选择左树数据表", trigger: "change" }],
  "left_tree_config.filter_column": [{ required: true, message: "请选择筛选字段", trigger: "change" }],
  "left_tree_config.parent_column": [{ required: true, message: "请配置左树父字段", trigger: ["blur", "change"] }],
  "left_tree_config.label_column": [{ required: true, message: "请配置左树显示字段", trigger: ["blur", "change"] }],
  "left_tree_config.value_column": [{ required: true, message: "请配置左树值字段", trigger: ["blur", "change"] }]
};

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
    remark: ""
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
