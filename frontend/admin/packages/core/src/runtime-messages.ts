type RuntimeLocale = "zh-CN" | "en-US";

const messages: Record<RuntimeLocale, Record<string, string>> = {
  "zh-CN": {
    missing_source_entry: "管理端模块缺少源码入口：{root}",
    https_certificate_missing: "VITE_HTTPS 已开启，但未找到证书文件，请先在仓库根目录运行 scripts/generate-dev-cert.sh；期望路径：{keyPath} 和 {certPath}",
    "label.module": "模块",
    "label.header_tool": "顶部工具",
    "label.user_menu": "用户菜单",
    "label.route_option": "路由配置",
    module_duplicate: "管理端{label}名称重复: {key}（模块 {owner} 与 {module}）",
    locale_duplicate: "{locale} 语言键重复: {key}",
    locale_namespace_invalid: "{module} 的语言键命名空间无效: {key}"
  },
  "en-US": {
    missing_source_entry: "The admin module is missing a source entry: {root}",
    https_certificate_missing: "VITE_HTTPS is enabled, but certificate files were not found. Run scripts/generate-dev-cert.sh from the repository root; expected paths: {keyPath} and {certPath}",
    "label.module": "module",
    "label.header_tool": "header tool",
    "label.user_menu": "user menu",
    "label.route_option": "route configuration",
    module_duplicate: "Duplicate admin {label} name: {key} (modules {owner} and {module})",
    locale_duplicate: "Duplicate locale key in {locale}: {key}",
    locale_namespace_invalid: "Invalid locale key namespace for {module}: {key}"
  }
};

/** 根据环境变量选择管理端启动期错误语言。 */
function resolveRuntimeLocale(): RuntimeLocale {
  const value = (globalThis as { process?: { env?: Record<string, string | undefined> } }).process?.env;
  const locale = value?.KRATOS_ADMIN_LOCALE ?? value?.LC_ALL ?? value?.LANG ?? "";
  return locale.toLowerCase().startsWith("zh") ? "zh-CN" : "en-US";
}

/** 渲染管理端无框架运行时错误。 */
export function runtimeMessage(key: string, params: Record<string, string> = {}): string {
  const template = messages[resolveRuntimeLocale()][key] ?? messages["en-US"][key] ?? key;
  return template.replace(/\{([A-Za-z0-9_]+)\}/g, (_, name: string) => params[name] ?? `{${name}}`);
}
