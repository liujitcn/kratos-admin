const messages = {
  "zh-CN": {
    package_built: "已生成 {name} 发布文件",
    locale_missing_default: "{module} 缺少 zh-CN 语言包",
    locale_set_mismatch: "{module} 语言集合不一致",
    locale_code_invalid: "语言代码无效：{locale}",
    locale_keys_mismatch: "{module}/{locale} 语言键集合不一致",
    locale_bundle_outdated: "语言注册产物过期：{file}，请执行 pnpm i18n:sync"
  },
  "en-US": {
    package_built: "Built package artifacts for {name}",
    locale_missing_default: "{module} is missing the zh-CN locale bundle",
    locale_set_mismatch: "{module} has an inconsistent locale set",
    locale_code_invalid: "Invalid locale code: {locale}",
    locale_keys_mismatch: "Locale message keys differ for {module}/{locale}",
    locale_bundle_outdated: "Locale bundle is out of date: {file}. Run pnpm i18n:sync."
  }
};

/** 根据生成工作区的语言偏好渲染脚手架消息。 */
export function workspaceMessage(key, params = {}) {
  const locale = (process.env.KRATOS_ADMIN_LOCALE ?? process.env.LC_ALL ?? process.env.LANG ?? "")
    .toLowerCase()
    .startsWith("zh")
    ? "zh-CN"
    : "en-US";
  const template = messages[locale][key] ?? messages["en-US"][key] ?? key;
  return template.replace(/\{([A-Za-z0-9_]+)\}/g, (_, name) => params[name] ?? `{${name}}`);
}
