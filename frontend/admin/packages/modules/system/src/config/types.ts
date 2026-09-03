import type { FormItemRule } from "element-plus";
import type { Component } from "vue";
import type { ProFormComponentType, ProFormField } from "@liujitcn/kratos-admin-core/components/ProForm/interface";

/** 运行配置表单模型。 */
export type RuntimeConfigModel = Record<string, any>;

/** 运行配置国际化参数值。 */
export type RuntimeConfigMessageArg = string | number | boolean | { key: string };

/** 运行配置字段校验规则。 */
export interface RuntimeConfigRule {
  /** 国际化消息键。 */
  messageKey: string;
  /** 国际化消息参数。 */
  messageArgs?: Record<string, RuntimeConfigMessageArg>;
  /** Element Plus 校验触发方式。 */
  trigger?: FormItemRule["trigger"];
  /** 其他 Element Plus 校验参数。 */
  [key: string]: unknown;
}

/** 运行配置字段定义。 */
export interface RuntimeConfigField {
  /** 字段路径。 */
  prop: string;
  /** 字段标题国际化消息键。 */
  labelKey: string;
  /** ProForm 控件类型。 */
  component: ProFormComponentType;
  /** 控件参数。 */
  props?: ProFormField["props"];
  /** 表单项参数。 */
  itemProps?: ProFormField["itemProps"];
  /** 选择项。 */
  options?: ProFormField["options"];
  /** 字段提示国际化消息键。 */
  labelTooltipKey?: string;
  /** 字段是否可见。 */
  visible?: ProFormField["visible"];
  /** 字段校验规则。 */
  rules?: RuntimeConfigRule[];
}

/** 管理端运行配置定义。 */
export interface RuntimeConfigDefinition<T extends RuntimeConfigModel = RuntimeConfigModel> {
  /** 配置稳定键。 */
  key: string;
  /** 配置名称国际化消息键。 */
  titleKey: string;
  /** 配置说明国际化消息键。 */
  descriptionKey: string;
  /** 配置项导航图标。 */
  icon: Component;
  /** 创建表单默认模型。 */
  createModel: () => T;
  /** 配置字段集合。 */
  fields: RuntimeConfigField[];
}
