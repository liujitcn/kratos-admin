import type { EnumProps } from "@/components/ProTable/interface";
import type { OptionBaseDictResponse_BaseDictItem } from "@/rpc/system/admin/v1/base_dict";
import { useDictStoreHook } from "@/stores/modules/dict";

export { resolveTableColumnAlign, type TableAlign } from "./tableColumn";

/** 字典值输出给表格枚举时的目标类型。 */
type DictValueType = "number" | "string";
/** 表格批量操作支持的主键类型。 */
type SelectedId = string | number;

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
