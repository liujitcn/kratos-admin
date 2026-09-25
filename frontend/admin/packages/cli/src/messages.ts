type CliLocale = "zh-CN" | "en-US";

const messages: Record<CliLocale, Record<string, string>> = {
  "zh-CN": {
    project_name_label: "项目名称",
    module_name_label: "模块名称",
    additional_module_name_label: "额外模块名称",
    examples_heading: "示例:",
    target_exists: "目标目录已存在，拒绝覆盖: {target}",
    unsupported_command: "不支持的命令: {command}",
    usage: "用法: kratos-admin create <project> --module <module[,module...]>",
    package_version_missing: "CLI package.json 缺少有效版本",
    option_missing_value: "选项 {option} 缺少值",
    kebab_required: "{label}必须使用 kebab-case: {value}",
    module_required: "至少需要一个业务模块名称",
    reserved_module: "自有模块名称不能使用保留名称: {name}",
    workspace_created: "已创建业务 workspace: {target}",
    list_separator: "、",
    module_table_row: "| `packages/modules/{name}/` | 可独立发布为 `@{name}/admin-module` 的业务模块。 |"
  },
  "en-US": {
    project_name_label: "Project name",
    module_name_label: "Module name",
    additional_module_name_label: "Additional module name",
    examples_heading: "Examples:",
    target_exists: "The target directory already exists; refusing to overwrite: {target}",
    unsupported_command: "Unsupported command: {command}",
    usage: "Usage: kratos-admin create <project> --module <module[,module...]>",
    package_version_missing: "CLI package.json is missing a valid version",
    option_missing_value: "Option {option} is missing a value",
    kebab_required: "{label} must use kebab-case: {value}",
    module_required: "At least one business module is required",
    reserved_module: "A custom module cannot use the reserved name: {name}",
    workspace_created: "Business workspace created: {target}",
    list_separator: ", ",
    module_table_row: "| `packages/modules/{name}/` | An independently publishable `@{name}/admin-module` business module. |"
  }
};

/** 根据环境变量选择 CLI 文案语言。 */
export function resolveLocale(): CliLocale {
  const value = process.env.KRATOS_ADMIN_LOCALE ?? process.env.LC_ALL ?? process.env.LANG ?? "";
  return value.toLowerCase().startsWith("zh") ? "zh-CN" : "en-US";
}

/** 渲染 CLI 稳定消息并替换命名参数。 */
export function cliMessage(key: string, params: Record<string, string> = {}): string {
  const template = messages[resolveLocale()][key] ?? messages["en-US"][key] ?? key;
  return template.replace(/\{([A-Za-z0-9_]+)\}/g, (_, name: string) => params[name] ?? `{${name}}`);
}
