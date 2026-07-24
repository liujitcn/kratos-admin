import type { CodeGenColumnOptionConfig } from "@/rpc/system/admin/v1/code_gen_column";

/** 字段选项配置所属范围。 */
export type CodeGenOptionScope = "query" | "list" | "form";

/** 单个范围内参与选项复刻的配置。 */
export interface CodeGenOptionContainer {
  enabled: boolean;
  component: string;
  option: CodeGenColumnOptionConfig;
}

/** 查询、列表和表单三范围的选项配置集合。 */
export interface CodeGenOptionCopyColumn {
  query_config: CodeGenOptionContainer;
  list_config: CodeGenOptionContainer;
  form_config: CodeGenOptionContainer;
}

const optionScopes: CodeGenOptionScope[] = ["query", "list", "form"];
const optionSourceTypes = new Set(["static", "dict", "table"]);

/** 返回指定范围的选项配置容器。 */
export function getCodeGenOptionContainer(column: CodeGenOptionCopyColumn, scope: CodeGenOptionScope) {
  switch (scope) {
    case "query":
      return column.query_config;
    case "list":
      return column.list_config;
    case "form":
      return column.form_config;
  }
}

/** 判断静态选项 JSON 是否至少包含一条完整数据。 */
function hasCompleteStaticOptions(value: string) {
  try {
    const options = JSON.parse(value) as unknown;
    return (
      Array.isArray(options) &&
      options.length > 0 &&
      options.every(item => {
        if (!item || typeof item !== "object") return false;
        const label = Reflect.get(item, "label");
        const optionValue = Reflect.get(item, "value");
        return (
          (typeof label === "string" || typeof label === "number") &&
          label !== "" &&
          (typeof optionValue === "string" || typeof optionValue === "number" || typeof optionValue === "boolean") &&
          optionValue !== ""
        );
      })
    );
  } catch {
    return false;
  }
}

/** 判断选项配置是否满足当前保存规则并可作为复刻来源。 */
export function isCompleteCodeGenOptionConfig(option: CodeGenColumnOptionConfig) {
  if (!option.kind) return false;
  if (option.kind === "switch") {
    return (
      option.source_type === "dict" &&
      !!option.source_value &&
      !!option.active_value &&
      !!option.inactive_value &&
      option.active_value !== option.inactive_value
    );
  }
  if (option.active_value || option.inactive_value) return false;
  if (!optionSourceTypes.has(option.source_type) || !option.source_value) return false;
  if (option.kind === "tree") {
    return option.source_type === "table" && !!option.label_field && !!option.value_field && !!option.parent_field;
  }
  if (option.source_type === "static") return hasCompleteStaticOptions(option.source_value);
  if (option.source_type === "table") return !!option.label_field && !!option.value_field;
  return true;
}

/** 将来源选项字段复制到目标的独立配置对象。 */
function copyCodeGenOption(source: CodeGenOptionContainer, target: CodeGenOptionContainer) {
  target.option = { ...source.option };
}

/** 从第一个相同组件的完整范围复刻选项配置。 */
export function copyFirstMatchingCodeGenOption(column: CodeGenOptionCopyColumn, targetScope: CodeGenOptionScope) {
  const target = getCodeGenOptionContainer(column, targetScope);
  if (!target.enabled) return false;
  const source = optionScopes
    .filter(scope => scope !== targetScope)
    .map(scope => getCodeGenOptionContainer(column, scope))
    .find(
      item =>
        item.enabled &&
        item.component === target.component &&
        isCompleteCodeGenOptionConfig(item.option)
    );
  if (!source) return false;
  copyCodeGenOption(source, target);
  return true;
}

/** 用当前范围补齐相同组件中尚未完整配置的其他范围。 */
export function copyCodeGenOptionToEmptyMatches(column: CodeGenOptionCopyColumn, sourceScope: CodeGenOptionScope) {
  const source = getCodeGenOptionContainer(column, sourceScope);
  if (!source.enabled || !isCompleteCodeGenOptionConfig(source.option)) return [];
  return optionScopes.filter(scope => {
    if (scope === sourceScope) return false;
    const target = getCodeGenOptionContainer(column, scope);
    if (!target.enabled || target.component !== source.component || isCompleteCodeGenOptionConfig(target.option)) {
      return false;
    }
    copyCodeGenOption(source, target);
    return true;
  });
}

/** 页面加载时按固定优先级补齐相同组件的空配置。 */
export function fillMissingCodeGenOptionConfigs(column: CodeGenOptionCopyColumn) {
  return optionScopes.filter(scope => {
    const target = getCodeGenOptionContainer(column, scope);
    return !isCompleteCodeGenOptionConfig(target.option) && copyFirstMatchingCodeGenOption(column, scope);
  });
}
