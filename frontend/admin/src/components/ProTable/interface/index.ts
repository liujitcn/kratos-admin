import { ComponentPublicInstance, Ref, VNode } from "vue";
import { BreakPoint, Responsive } from "@/components/Grid/interface";
import { TableColumnCtx } from "element-plus/es/components/table/src/table-column/defaults";
import { ProTableProps } from "@/components/ProTable/index.vue";
import ProTable from "@/components/ProTable/index.vue";

/** ProTable 默认行数据结构。 */
type DefaultRow = Record<string, any>;

/** 表格枚举选项结构，供搜索项、字典和状态列复用。 */
export interface EnumProps {
  label?: string; // 选项框显示的文字
  value?: string | number | boolean | any[]; // 选项框值
  disabled?: boolean; // 是否禁用此选项
  tagType?: string; // 当 tag 为 true 时，此选择会指定 tag 显示类型
  children?: EnumProps[]; // 为树形选择时，可以通过 children 属性指定子选项
  [key: string]: any;
}

/** Element Plus 表格内置列类型。 */
export type TypeProps = "index" | "selection" | "radio" | "expand" | "sort";

/** ProTable 搜索表单支持的控件类型。 */
export type SearchType =
  | "input"
  | "input-number"
  | "select"
  | "select-v2"
  | "tree-select"
  | "cascader"
  | "date-picker"
  | "time-picker"
  | "time-select"
  | "switch"
  | "slider";

/** 字典值在表格枚举中的目标类型。 */
export type DictValueType = "string" | "number";

/** 自定义搜索控件渲染时暴露的上下文。 */
export type SearchRenderScope = {
  searchParam: { [key: string]: any };
  placeholder: string;
  clearable: boolean;
  options: EnumProps[];
  data: EnumProps[];
};

/** ProTable 单列搜索配置。 */
export type SearchProps = {
  el?: SearchType; // 当前项搜索框的类型
  label?: string; // 当前项搜索框的 label
  props?: any; // 搜索项参数，根据 element plus 官方文档来传递，该属性所有值会透传到组件
  key?: string; // 当搜索项 key 不为 prop 属性时，可通过 key 指定
  tooltip?: string; // 搜索提示
  order?: number; // 搜索项排序（从大到小）
  span?: number; // 搜索项所占用的列数，默认为 1 列
  offset?: number; // 搜索字段左侧偏移列数
  defaultValue?: string | number | boolean | any[] | Ref<any>; // 搜索项默认值
  dictCode?: string; // 搜索项独立使用的字典编码
  dictValueType?: DictValueType; // 搜索项字典值类型
  enum?: EnumProps[] | Ref<EnumProps[]>; // 搜索项独立使用的枚举字典
  render?: (scope: SearchRenderScope) => VNode; // 自定义搜索内容渲染（tsx语法）
} & Partial<Record<BreakPoint, Responsive>>;

/** 枚举或树形数据字段映射配置。 */
export type FieldNamesProps = {
  label: string;
  value: string;
  children?: string;
};

/** 单元格渲染和行内操作共享的上下文。 */
export type RenderScope<T extends DefaultRow = DefaultRow> = {
  row: T;
  $index: number;
  column: TableColumnCtx<T>;
  [key: string]: any;
};

/** 表头自定义渲染上下文。 */
export type HeaderRenderScope<T extends DefaultRow = DefaultRow> = {
  $index: number;
  column: TableColumnCtx<T>;
  [key: string]: any;
};

/** ProTable 预置单元格渲染类型。 */
export type CellType = "image" | "status" | "actions" | "money";

/** 行内操作附加参数，支持静态对象或按行动态生成。 */
export type ColumnActionParams<T extends DefaultRow = DefaultRow> =
  | Record<string, any>
  | ((scope: RenderScope<T>) => Record<string, any>);

/** 表头操作按钮渲染和点击时的选择态上下文。 */
export type HeaderActionScope<T extends DefaultRow = DefaultRow> = {
  selectedList: T[];
  selectedListIds: Array<string | number>;
  isSelected: boolean;
};

/** 表头操作附加参数，支持静态对象或按选择态动态生成。 */
export type HeaderActionParams<T extends DefaultRow = DefaultRow> =
  | Record<string, any>
  | ((scope: HeaderActionScope<T>) => Record<string, any>);

/** 图片列预置渲染配置。 */
export interface ImageCellProps<T extends DefaultRow = DefaultRow> {
  src?: string | ((scope: RenderScope<T>) => string);
  previewSrc?: string | ((scope: RenderScope<T>) => string);
  width?: number | string;
  height?: number | string;
  previewWidth?: number | string;
  previewHeight?: number | string;
}

/** 状态列预置渲染和变更配置。 */
export interface StatusCellProps<T extends DefaultRow = DefaultRow> {
  activeValue: string | number | boolean;
  inactiveValue: string | number | boolean;
  activeText?: string;
  inactiveText?: string;
  disabled?: boolean | ((scope: RenderScope<T>) => boolean);
  beforeChange?: (scope: RenderScope<T>, params?: Record<string, any>) => boolean | Promise<boolean>;
  onChange?: (value: string | number | boolean, scope: RenderScope<T>, params?: Record<string, any>) => void | Promise<void>;
  params?: ColumnActionParams<T>;
}

/** 金额列预置格式化配置。 */
export interface MoneyCellProps<T extends DefaultRow = DefaultRow> {
  value?: number | string | ((scope: RenderScope<T>) => number | string | undefined | null);
  prefix?: string;
  suffix?: string;
}

/** 行内操作按钮配置。 */
export interface TableActionProps<T extends DefaultRow = DefaultRow> {
  label: string;
  icon?: any;
  type?: "primary" | "success" | "warning" | "danger" | "info";
  link?: boolean;
  disabled?: boolean | ((scope: RenderScope<T>) => boolean);
  hidden?: boolean | ((scope: RenderScope<T>) => boolean);
  params?: ColumnActionParams<T>;
  onClick: (scope: RenderScope<T>, params?: Record<string, any>) => void | Promise<void>;
}

/** 表头操作按钮配置。 */
export interface HeaderActionProps<T extends DefaultRow = DefaultRow> {
  label: string;
  icon?: any;
  type?: "primary" | "success" | "warning" | "danger" | "info";
  disabled?: boolean | ((scope: HeaderActionScope<T>) => boolean);
  hidden?: boolean | ((scope: HeaderActionScope<T>) => boolean);
  params?: HeaderActionParams<T>;
  onClick: (scope: HeaderActionScope<T>, params?: Record<string, any>) => void | Promise<void>;
}

/** ProTable 列配置，扩展 Element Plus 表格列能力。 */
export interface ColumnProps<T extends DefaultRow = DefaultRow> extends Partial<
  Omit<TableColumnCtx<T>, "type" | "children" | "renderCell" | "renderHeader">
> {
  type?: TypeProps; // 列类型
  cellType?: CellType; // 预置单元格类型
  tag?: boolean | Ref<boolean>; // 是否是标签展示
  isShow?: boolean | Ref<boolean>; // 是否显示在表格当中
  isSetting?: boolean | Ref<boolean>; // 是否在 ColSetting 中可配置
  search?: SearchProps | undefined; // 搜索项配置
  dictCode?: string; // 字典编码，配置后优先使用 Dict / DictLabel 组件
  dictValueType?: DictValueType; // 字典值类型
  enum?: EnumProps[] | Ref<EnumProps[]> | ((params?: any) => Promise<any>); // 枚举字典
  isFilterEnum?: boolean | Ref<boolean>; // 当前单元格值是否根据 enum 格式化（示例：enum 只作为搜索项数据）
  fieldNames?: FieldNamesProps; // 指定 label && value && children 的 key 值
  headerRender?: (scope: HeaderRenderScope<T>) => VNode; // 自定义表头内容渲染（tsx语法）
  render?: (scope: RenderScope<T>) => VNode | string; // 自定义单元格内容渲染（tsx语法）
  imageProps?: ImageCellProps<T>; // 图片列配置
  statusProps?: StatusCellProps<T>; // 状态列配置
  moneyProps?: MoneyCellProps<T>; // 金额列配置
  actions?: TableActionProps<T>[]; // 操作按钮配置
  _children?: ColumnProps<T>[]; // 多级表头
}

/** ProTable 对外暴露的组件实例方法类型。 */
export type ProTableInstance = Omit<InstanceType<typeof ProTable>, keyof ComponentPublicInstance | keyof ProTableProps>;
