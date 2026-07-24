import previewAvatar from "@/assets/images/avatar.png";
import type { ProFormOption } from "@/components/ProForm/interface";
import type { OptionBaseDictResponse_BaseDict } from "@/rpc/system/admin/v1/base_dict";
import type { CodeGenColumn, CodeGenColumnOptionConfig, CodeGenColumnQueryConfig } from "@/rpc/system/admin/v1/code_gen_column";
import type { CodeGenLeftTreeConfig, CodeGenTableForm } from "@/rpc/system/admin/v1/code_gen_table";

/** 页面预览中的字段配置范围。 */
export type CodeGenPreviewScope = "query" | "list" | "form";

/** 页面预览模拟记录。 */
export type CodeGenPreviewRow = Record<string, any> & {
  /** 树形表格子记录。 */
  children?: CodeGenPreviewRow[];
  /** 树形懒加载表格是否存在子记录。 */
  has_children?: boolean;
};

/** 代码生成页面预览快照。 */
export interface CodeGenPagePreviewSnapshot {
  /** 当前代码生成表配置。 */
  table: CodeGenTableForm;
  /** 当前已经保存的字段配置。 */
  columns: CodeGenColumn[];
}

/** 页面预览中按字段和范围隔离的模拟选项。 */
export type CodeGenPreviewOptionMap = Record<string, ProFormOption[]>;

/** 创建字段指定配置范围的预览选项缓存键。 */
export function createCodeGenPreviewOptionKey(columnName: string, scope: CodeGenPreviewScope) {
  return `${columnName}:${scope}`;
}

/** 根据当前字段配置和字典数据创建查询、列表和表单各自的模拟选项。 */
export function createCodeGenPreviewOptionMap(
  columns: CodeGenColumn[],
  dictionaries: OptionBaseDictResponse_BaseDict[]
): CodeGenPreviewOptionMap {
  const dictionaryMap = new Map(dictionaries.map(dictionary => [dictionary.code, dictionary]));
  return columns.reduce<CodeGenPreviewOptionMap>((optionMap, column) => {
    const label = column.comment || column.name;
    const configs: Array<[CodeGenPreviewScope, CodeGenColumnOptionConfig | undefined, boolean]> = [
      [
        "query",
        column.query_config?.option,
        Boolean(column.query_config?.enabled && hasPreviewOptions(column.query_config.component))
      ],
      [
        "list",
        column.list_config?.option,
        Boolean(column.list_config?.enabled && ["switch", "select", "tree-select"].includes(column.list_config.component))
      ],
      [
        "form",
        column.form_config?.option,
        Boolean(column.form_config?.enabled && hasPreviewOptions(column.form_config.component))
      ]
    ];
    configs.forEach(([scope, option, enabled]) => {
      optionMap[createCodeGenPreviewOptionKey(column.name, scope)] = enabled
        ? createCodeGenPreviewOptions(label, option, dictionaryMap)
        : [];
    });
    return optionMap;
  }, {});
}

/** 根据左树配置创建结构相符的模拟树节点。 */
export function createCodeGenLeftTreeOptions(config?: CodeGenLeftTreeConfig): ProFormOption[] {
  if (!config?.table_name) return [];
  const option: CodeGenColumnOptionConfig = {
    kind: "tree",
    source_type: "table",
    source_value: config.table_name,
    label_field: config.label_column,
    value_field: config.value_column,
    parent_field: config.parent_column,
    active_value: "",
    inactive_value: "",
    lazy: config.lazy
  };
  return createCodeGenPreviewOptions(config.label_column || "分类", option, new Map());
}

/** 创建页面列表所需的模拟业务记录。 */
export function createCodeGenPreviewRows(
  snapshot: CodeGenPagePreviewSnapshot,
  optionMap: CodeGenPreviewOptionMap,
  leftTreeOptions: ProFormOption[]
) {
  const { table, columns } = snapshot;
  const primaryColumn = resolveCodeGenPrimaryColumn(columns);
  const leftTreeValues = flattenCodeGenPreviewOptions(leftTreeOptions).map(option => option.value);
  const rows = Array.from(
    { length: table.page_type === "tree" || table.page_type === "tree_lazy" ? 12 : 18 },
    (_, rowIndex) => {
      const row: CodeGenPreviewRow = {};
      columns.forEach(column => {
        const options = resolveColumnPreviewOptions(optionMap, column);
        row[column.name] = createCodeGenPreviewValue(column, rowIndex, options);
      });
      if (!(primaryColumn in row)) row[primaryColumn] = rowIndex + 1;
      if (table.page_type === "left_tree" && table.left_tree_config?.filter_column && leftTreeValues.length) {
        row[table.left_tree_config.filter_column] = leftTreeValues[rowIndex % leftTreeValues.length];
      }
      return row;
    }
  );

  // 树形页面按照真实父节点字段构造层级，列表字段和值仍来自当前数据库配置。
  if (
    (table.page_type === "tree" || table.page_type === "tree_lazy") &&
    table.parent_column &&
    table.parent_column !== primaryColumn
  ) {
    rows.forEach((row, rowIndex) => {
      if (rowIndex === 0 || rowIndex % 4 === 0) {
        row[table.parent_column] = 0;
        return;
      }
      const rootIndex = Math.floor(rowIndex / 4) * 4;
      row[table.parent_column] = rows[rootIndex][primaryColumn];
    });
  }
  if (table.page_type === "tree_lazy" && table.parent_column) {
    rows.forEach(row => {
      row.has_children = rows.some(child => String(child[table.parent_column!]) === String(row[primaryColumn]));
    });
  }
  return rows;
}

/** 按查询配置过滤页面模拟记录。 */
export function filterCodeGenPreviewRows(rows: CodeGenPreviewRow[], columns: CodeGenColumn[], params: Record<string, any>) {
  const queryColumns = columns.filter(column => column.query_config?.enabled);
  return rows.filter(row =>
    queryColumns.every(column => {
      const queryValue = params[column.name];
      if (isEmptyPreviewValue(queryValue)) return true;
      return matchCodeGenPreviewValue(row[column.name], queryValue, column.query_config);
    })
  );
}

/** 将扁平模拟记录转换成 Element Plus 表格树。 */
export function buildCodeGenPreviewTree(rows: CodeGenPreviewRow[], primaryColumn: string, parentColumn: string) {
  const rowMap = new Map<string, CodeGenPreviewRow>();
  const roots: CodeGenPreviewRow[] = [];
  rows.forEach(row => {
    rowMap.set(String(row[primaryColumn]), { ...row, children: [] });
  });
  rowMap.forEach(row => {
    const parent = rowMap.get(String(row[parentColumn]));
    if (parent && parent !== row) {
      parent.children?.push(row);
      return;
    }
    roots.push(row);
  });
  return roots;
}

/** 将树形选项扁平化，供模拟记录赋值和标签匹配使用。 */
export function flattenCodeGenPreviewOptions(options: ProFormOption[]): ProFormOption[] {
  return options.flatMap(option => [option, ...flattenCodeGenPreviewOptions(option.children ?? [])]);
}

/** 返回当前表的真实主键字段，缺少主键时使用预览内部编号。 */
export function resolveCodeGenPrimaryColumn(columns: CodeGenColumn[]) {
  return columns.find(column => column.is_primary)?.name || "__preview_id";
}

/** 返回指定字段范围的模拟选项。 */
export function resolveCodeGenPreviewOptions(optionMap: CodeGenPreviewOptionMap, columnName: string, scope: CodeGenPreviewScope) {
  return optionMap[createCodeGenPreviewOptionKey(columnName, scope)] ?? [];
}

/** 创建单个字段在下拉、树选择和开关展示中使用的模拟选项。 */
function createCodeGenPreviewOptions(
  label: string,
  option: CodeGenColumnOptionConfig | undefined,
  dictionaryMap: Map<string, OptionBaseDictResponse_BaseDict>
): ProFormOption[] {
  if (option?.source_type === "static") {
    const staticOptions = parseCodeGenStaticOptions(option.source_value);
    if (staticOptions.length) return staticOptions;
  }
  if (option?.source_type === "dict") {
    const dictionaryItems = dictionaryMap.get(option.source_value)?.items ?? [];
    // 开关只使用用户明确选择的开启项和关闭项，避免模拟出第三种状态。
    if (option.kind === "switch") {
      return [option.active_value, option.inactive_value].flatMap(value => {
        const item = dictionaryItems.find(dictionaryItem => dictionaryItem.value === value);
        return item ? [{ label: item.label, value: item.value }] : [];
      });
    }
    return dictionaryItems.map(item => ({ label: item.label, value: item.value }));
  }
  const sourceLabel = option?.source_value || label;
  // 树形组件用两级节点表现最终布局，其字段名和来源标识均取当前真实配置。
  if (option?.kind === "tree") {
    return Array.from({ length: 3 }, (_, rootIndex) => {
      const rootLabel = `${sourceLabel} ${rootIndex + 1}`;
      const children = Array.from({ length: 2 }, (_, childIndex) => ({
        label: `${rootLabel}/${option.label_field || label} ${rootIndex + 1}-${childIndex + 1}`,
        value: `${sourceLabel}-${rootIndex + 1}-${childIndex + 1}`,
        has_children: false
      }));
      return {
        label: rootLabel,
        value: `${sourceLabel}-${rootIndex + 1}`,
        has_children: children.length > 0,
        children
      };
    });
  }
  return Array.from({ length: 4 }, (_, optionIndex) => ({
    label: `${sourceLabel}选项 ${optionIndex + 1}`,
    value: `${sourceLabel}-${optionIndex + 1}`
  }));
}

/** 判断组件是否需要模拟选项集合。 */
function hasPreviewOptions(component?: string) {
  return ["segmented", "select", "dict", "radio-group", "checkbox-group", "tree-select", "transfer"].includes(component || "");
}

/** 解析字段配置中已经维护的静态选项和树形子节点。 */
function parseCodeGenStaticOptions(value: string): ProFormOption[] {
  if (!value) return [];
  try {
    const options = JSON.parse(value) as unknown;
    if (!Array.isArray(options)) return [];
    return normalizeStaticOptions(options);
  } catch {
    return [];
  }
}

/** 递归过滤不符合 ProForm 结构的静态选项。 */
function normalizeStaticOptions(options: unknown[]): ProFormOption[] {
  return options.flatMap(option => {
    if (!option || typeof option !== "object") return [];
    const label = Reflect.get(option, "label");
    const value = Reflect.get(option, "value");
    if (
      (typeof label !== "string" && typeof label !== "number") ||
      (typeof value !== "string" && typeof value !== "number" && typeof value !== "boolean")
    ) {
      return [];
    }
    const children = Reflect.get(option, "children");
    return [
      {
        label: String(label),
        value,
        disabled: Boolean(Reflect.get(option, "disabled")),
        children: Array.isArray(children) ? normalizeStaticOptions(children) : undefined
      }
    ];
  });
}

/** 按表单、列表、查询的优先级选择模拟记录使用的字段选项。 */
function resolveColumnPreviewOptions(optionMap: CodeGenPreviewOptionMap, column: CodeGenColumn) {
  for (const scope of ["list", "query", "form"] as const) {
    const options = resolveCodeGenPreviewOptions(optionMap, column.name, scope);
    if (options.length) return flattenCodeGenPreviewOptions(options);
  }
  return [];
}

/** 根据数据库属性和组件配置创建单元格模拟值。 */
function createCodeGenPreviewValue(column: CodeGenColumn, rowIndex: number, options: ProFormOption[]) {
  const sequence = rowIndex + 1;
  const optionValue = options[rowIndex % Math.max(options.length, 1)]?.value;
  if (optionValue !== undefined) return optionValue;
  if (column.is_primary) return sequence;
  if (column.name === "created_at" || column.name === "updated_at" || isDateTimeColumn(column)) {
    return `2026-07-${String((rowIndex % 28) + 1).padStart(2, "0")} ${String(8 + (rowIndex % 10)).padStart(2, "0")}:30:00`;
  }
  if (["image", "image-upload", "images-upload"].includes(column.list_config?.component || column.form_config?.component || "")) {
    return column.form_config?.component === "images-upload" ? [previewAvatar] : previewAvatar;
  }
  if (isBooleanColumn(column)) return rowIndex % 2 === 0;
  if (isNumericColumn(column)) return column.list_config?.component === "money" ? 12000 + rowIndex * 1350 : sequence;
  if (/phone|mobile/.test(column.name)) return `1380000${String(sequence).padStart(4, "0")}`;
  if (column.name.includes("email")) return `preview${sequence}@example.com`;
  if (column.name.includes("code")) return `CODE_${String(sequence).padStart(3, "0")}`;
  const label = column.comment || column.name;
  return `${label}${String(sequence).padStart(2, "0")}`;
}

/** 判断字段是否为日期时间类型。 */
function isDateTimeColumn(column: CodeGenColumn) {
  const dbType = column.db_type.toLowerCase();
  return dbType.includes("date") || dbType.includes("time") || column.form_config?.component === "date-picker";
}

/** 判断字段是否为布尔类型。 */
function isBooleanColumn(column: CodeGenColumn) {
  const dbType = column.db_type.toLowerCase();
  return column.go_type === "bool" || dbType === "bool" || dbType === "boolean" || dbType.includes("tinyint(1)");
}

/** 判断字段是否为数值类型。 */
export function isNumericColumn(column: CodeGenColumn) {
  const dbType = column.db_type.toLowerCase();
  return column.ts_type === "number" || ["int", "decimal", "float", "double"].some(type => dbType.includes(type));
}

/** 判断查询值是否为空。 */
function isEmptyPreviewValue(value: unknown) {
  return value === undefined || value === null || value === "" || (Array.isArray(value) && !value.length);
}

/** 按等于、模糊和区间操作符匹配单个模拟字段。 */
function matchCodeGenPreviewValue(rowValue: unknown, queryValue: unknown, config?: CodeGenColumnQueryConfig) {
  if (config?.operator === "between" && Array.isArray(queryValue) && queryValue.length === 2) {
    const target = new Date(String(rowValue)).getTime();
    const start = new Date(queryValue[0]).getTime();
    const end = new Date(queryValue[1]).getTime();
    return Number.isFinite(target) && target >= start && target <= end;
  }
  if (config?.operator === "like") {
    return String(rowValue ?? "")
      .toLowerCase()
      .includes(String(queryValue).toLowerCase());
  }
  return String(rowValue ?? "") === String(queryValue);
}
