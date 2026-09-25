const messages = {
  'zh-CN': {
    missing_default_locale: '{module} 缺少 zh-CN 语言包',
    locale_set_mismatch: '{module} 语言集合不一致',
    locale_code_invalid: '语言代码无效：{locale}',
    key_set_mismatch: '{module}/{locale} 语言键集合不一致',
    bundle_outdated: '语言注册产物过期：{file}，请执行 pnpm i18n:sync',
    export_target_missing: '{name} 导出目标不存在：{target}',
    public_version_mismatch: 'Taro 公开包版本必须保持一致：{versions}',
    core_alias_forbidden: '{location}: 非 core 代码不得使用 core 私有别名 {specifier}',
    ui_alias_forbidden: '{location}: 非 UI 代码不得使用 UI 私有别名 {specifier}',
    system_alias_forbidden: '{location}: 非 system 代码不得使用 system 私有别名 {specifier}',
    cross_package_relative_forbidden: '{location}: 禁止通过相对路径跨包引用 {specifier}',
    export_not_public: '{location}: {specifier} 未在 {packageName} exports 中公开',
    core_dir_unexpected: '{packageName}: core 包目录异常',
    ui_dir_unexpected: '{packageName}: UI 包目录异常',
    system_dir_unexpected: '{packageName}: system 包目录异常',
    exports_check_failed: 'Taro package exports 检查失败：\n{violations}',
    exports_check_passed: 'Taro package exports 检查通过（{packages}）',
    list_separator: '、',
  },
  'en-US': {
    missing_default_locale: '{module} is missing the zh-CN locale bundle',
    locale_set_mismatch: '{module} has a different locale set',
    locale_code_invalid: 'Invalid locale code: {locale}',
    key_set_mismatch: 'Locale message keys differ for {module}/{locale}',
    bundle_outdated: 'Locale bundle is out of date: {file}. Run pnpm i18n:sync.',
    export_target_missing: '{name} export target does not exist: {target}',
    public_version_mismatch: 'Taro public package versions must match: {versions}',
    core_alias_forbidden: '{location}: Core private aliases cannot be used outside core: {specifier}',
    ui_alias_forbidden: '{location}: UI private aliases cannot be used outside UI: {specifier}',
    system_alias_forbidden: '{location}: System private aliases cannot be used outside system: {specifier}',
    cross_package_relative_forbidden: '{location}: Cross-package relative import is forbidden: {specifier}',
    export_not_public: '{location}: {specifier} is not exposed by {packageName} exports',
    core_dir_unexpected: '{packageName}: unexpected core package directory',
    ui_dir_unexpected: '{packageName}: unexpected UI package directory',
    system_dir_unexpected: '{packageName}: unexpected system package directory',
    exports_check_failed: 'Taro package exports check failed:\n{violations}',
    exports_check_passed: 'Taro package exports check passed ({packages})',
    list_separator: ', ',
  },
}

/** 根据工作区语言偏好渲染脚手架诊断文案。 */
export function workspaceMessage(key, params = {}) {
  const locale = (process.env.KRATOS_ADMIN_LOCALE ?? process.env.LC_ALL ?? process.env.LANG ?? '')
    .toLowerCase()
    .startsWith('zh')
    ? 'zh-CN'
    : 'en-US'
  const template = messages[locale][key] ?? messages['en-US'][key] ?? key
  return template.replace(/\{([A-Za-z0-9_]+)\}/g, (_, name) => params[name] ?? `{${name}}`)
}
