import type { FormItemRule } from "element-plus";

/** ProForm 支持的字段组件类型。 */
export type ProFormComponentType =
  | "input"
  | "password"
  | "textarea"
  | "input-number"
  | "segmented"
  | "switch"
  | "checkbox"
  | "select"
  | "dict"
  | "radio-group"
  | "checkbox-group"
  | "tree-select"
  | "date-picker"
  | "cron-expression"
  | "transfer"
  | "image-upload"
  | "images-upload"
  | "file-upload"
  | "files-upload"
  | "rich-text"
  | "dynamic-list"
  | "kv-list"
  | "slot";

/** ProForm 选择型字段选项。 */
export interface ProFormOption {
  label: string;
  value: string | number | boolean;
  disabled?: boolean;
  children?: ProFormOption[];
  /** Element Plus 懒加载树节点是否为叶子节点。 */
  isLeaf?: boolean;
}

/** ProForm 字段配置。 */
export interface ProFormField {
  /** 字段绑定路径，支持点路径访问嵌套对象。 */
  prop: string;
  /** 表单项标题。 */
  label: string;
  /** 字段渲染组件类型。 */
  component: ProFormComponentType;
  /** 字段组件参数，支持按当前表单模型动态生成。 */
  props?: Record<string, any> | ((model: Record<string, any>) => Record<string, any>);
  /** Element Plus 表单项参数，支持按当前表单模型动态生成。 */
  itemProps?: Record<string, any> | ((model: Record<string, any>) => Record<string, any>);
  /** 选择型字段选项，支持按当前表单模型动态生成。 */
  options?: ProFormOption[] | ((model: Record<string, any>) => ProFormOption[]);
  /** 栅格占位列数。 */
  colSpan?: number;
  /** 是否在当前字段前强制换行。 */
  rowBreakBefore?: boolean;
  /** 自定义插槽名称。 */
  slotName?: string;
  /** 标题提示文案。 */
  labelTooltip?: string;
  /** 字段校验规则。 */
  rules?: FormItemRule[];
  /** 字段是否显示。 */
  visible?: (model: Record<string, any>) => boolean;
  /** 单个复选框显示文案，未配置时使用表单项标题。 */
  checkboxLabel?: string;
}

/** ProForm 对外暴露的实例方法。 */
export interface ProFormInstance {
  validate: () => Promise<boolean | undefined> | undefined;
  resetFields: () => void;
  clearValidate: (props?: string | string[]) => void;
}
