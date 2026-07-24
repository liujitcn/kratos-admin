import type { ColumnProps, EnumProps, TypeProps } from "@/components/ProTable/interface";
import type { OptionBaseDictResponse_BaseDictItem } from "@/rpc/system/admin/v1/base_dict";
import { useDictStoreHook } from "@/stores/modules/dict";
import { isRef } from "vue";

/** 字典值输出给表格枚举时的目标类型。 */
type DictValueType = "number" | "string";
/** 表格批量操作支持的主键类型。 */
type SelectedId = string | number;

/** 表格单元格支持的对齐方式。 */
export type TableAlign = "left" | "center" | "right";

/** 表格分页入参。 */
type PageRequestParams = Record<string, any> & {
  /** 当前页码。 */
  page_num?: string | number;
  /** 每页条数。 */
  page_size?: string | number;
};

/** 归一化后的分页请求。 */
type NormalizedPageRequest<T extends PageRequestParams> = T & {
  /** 接口请求当前页码。 */
  page_num: number;
  /** 接口请求每页条数。 */
  page_size: number;
};

/**
 * 获取表格行的列值，支持点号分隔的多级字段。
 */
function getTableCellValue(row: Record<string, any>, prop: string) {
  return prop.split(".").reduce((value, key) => (value == null ? undefined : value[key]), row as any);
}

/**
 * 判断值是否为可用于数值对齐的 Number 类型。
 */
function isPureNumber(value: unknown) {
  return typeof value === "number" && Number.isFinite(value);
}

/**
 * 判断枚举是否来自数据表映射，远程函数和响应式选项都属于动态数据映射。
 */
function isDataTableEnum(enumValue: ColumnProps["enum"]) {
  return typeof enumValue === "function" || isRef(enumValue);
}

/**
 * 根据列语义和当前表格数据解析单元格对齐方式。
 *
 * 显式 align 始终优先；数据表映射按文本左对齐，字典、静态枚举及预置状态列居中，
 * 金额列和纯数字列右对齐，其余内容左对齐。空表时会先按列语义返回默认值，
 * 数据加载后由响应式表格重新解析。
 */
export function resolveTableColumnAlign(column: ColumnProps, rows: Record<string, any>[] = []): TableAlign {
  if (column.align === "left" || column.align === "center" || column.align === "right") return column.align;

  const centeredTypes: TypeProps[] = ["selection", "radio", "index", "expand", "sort"];
  if (centeredTypes.includes(column.type as TypeProps)) return "center";
  if (column.cellType === "actions" || column.cellType === "status" || column.cellType === "image") return "center";
  if (column.cellType === "money") return "right";
  if (column.dictCode || column.tag) return "center";
  if (column.enum) return isDataTableEnum(column.enum) ? "left" : "center";

  if (column.prop) {
    const values = rows
      .map(row => getTableCellValue(row, column.prop as string))
      .filter(value => value !== undefined && value !== null && value !== "");
    if (values.length && values.every(isPureNumber)) return "right";
  }

  return "left";
}

/**
 * 按配置将字典值转换为表格枚举可识别的类型。
 */
function transformDictValue(dictItem: OptionBaseDictResponse_BaseDictItem, valueType: DictValueType) {
  if (valueType === "number") return Number(dictItem.value);
  return dictItem.value;
}

/**
 * 将字典缓存转换为 ProTable 搜索枚举数据。
 */
export async function buildDictEnum(code: string, valueType: DictValueType = "number") {
  const dictStore = useDictStoreHook();
  const dictList = await dictStore.ensureDictionary(code);

  const data: EnumProps[] = dictList.map(dictItem => ({
    label: dictItem.label,
    value: transformDictValue(dictItem, valueType),
    tagType: dictItem.tag_type
  }));

  return { data };
}

/**
 * 统一补齐分页请求参数，避免组件透传时出现字符串页码。
 */
export function buildPageRequest<T extends PageRequestParams>(params: T): NormalizedPageRequest<T> {
  const pageNum = Number(params.page_num ?? 1);
  const pageSize = Number(params.page_size ?? 10);
  return {
    ...params,
    page_num: pageNum,
    page_size: pageSize
  } as NormalizedPageRequest<T>;
}

/**
 * 统一整理表格多选和单条操作产生的 ID 集合。
 */
export function normalizeSelectedIds(selected?: SelectedId | SelectedId[]) {
  if (Array.isArray(selected)) {
    return selected.filter(item => item !== undefined && item !== null && item !== "");
  }
  if (selected === undefined || selected === null || selected === "") return [];
  return [selected];
}
