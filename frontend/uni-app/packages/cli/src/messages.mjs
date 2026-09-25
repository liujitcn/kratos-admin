const messages = {
  'zh-CN': {
    package_version_missing: 'CLI package.json 缺少有效版本',
    target_exists: '目标目录已存在：{target}',
    usage: '用法: kratos-uni-app create <目录> [--module <名称>] [--with <包名>]',
    unknown_argument: '未知参数：{argument}',
    module_name_invalid: '模块名无效：{name}',
    package_name_invalid: '包名无效：{name}',
    https_environment_hint:
      '后端使用 HTTPS 时，在 .env.development-h5.local 中设置 VITE_APP_API_URL=https://localhost:7001。',
  },
  'en-US': {
    package_version_missing: 'CLI package.json is missing a valid version',
    target_exists: 'The target directory already exists: {target}',
    usage: 'Usage: kratos-uni-app create <directory> [--module <name>] [--with <package>]',
    unknown_argument: 'Unknown argument: {argument}',
    module_name_invalid: 'Invalid module name: {name}',
    package_name_invalid: 'Invalid package name: {name}',
    https_environment_hint:
      'When the backend uses HTTPS, set VITE_APP_API_URL=https://localhost:7001 in .env.development-h5.local.',
  },
}

/** 解析脚手架文案所用的语言区域。 */
export function resolveCliLocale() {
  return (process.env.KRATOS_ADMIN_LOCALE ?? process.env.LC_ALL ?? process.env.LANG ?? '')
    .toLowerCase()
    .startsWith('zh')
    ? 'zh-CN'
    : 'en-US'
}

/** 根据环境变量选择 uni-app CLI 文案语言。 */
export function cliMessage(key, params = {}) {
  const locale = resolveCliLocale()
  const template = messages[locale][key] ?? messages['en-US'][key] ?? key
  return template.replace(/\{([A-Za-z0-9_]+)\}/g, (_, name) => params[name] ?? `{${name}}`)
}
