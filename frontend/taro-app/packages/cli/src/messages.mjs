const messages = {
  'zh-CN': {
    package_version_missing: 'CLI package.json 缺少有效版本',
    target_exists: '目标目录已存在：{target}',
    unsupported_command: '不支持的命令：{command}',
    usage: '用法: kratos-taro-app create <目录> [--module <名称>] [--with <包名>]',
    workspace_created: '已创建 Taro workspace：{target}',
    unknown_argument: '未知参数：{argument}',
    option_missing_value: '选项 {option} 缺少值',
    project_name_invalid: '项目名必须使用 kebab-case：{name}',
    module_name_invalid: '模块名无效：{name}',
    package_name_invalid: '包名无效：{name}',
    examples_heading: '示例:',
    example_create: '  kratos-taro-app create customer-app',
    example_modules: '  kratos-taro-app create business-app --module business,report',
    example_package: '  kratos-taro-app create customer-app --with @acme/customer-module',
    https_environment_hint: '后端使用 HTTPS 时，在 .env.development-h5.local 中设置 VITE_APP_API_URL=https://localhost:7001。',
  },
  'en-US': {
    package_version_missing: 'CLI package.json is missing a valid version',
    target_exists: 'The target directory already exists: {target}',
    unsupported_command: 'Unsupported command: {command}',
    usage: 'Usage: kratos-taro-app create <directory> [--module <name>] [--with <package>]',
    workspace_created: 'Taro workspace created: {target}',
    unknown_argument: 'Unknown argument: {argument}',
    option_missing_value: 'Option {option} is missing a value',
    project_name_invalid: 'Project name must use kebab-case: {name}',
    module_name_invalid: 'Invalid module name: {name}',
    package_name_invalid: 'Invalid package name: {name}',
    examples_heading: 'Examples:',
    example_create: '  kratos-taro-app create customer-app',
    example_modules: '  kratos-taro-app create business-app --module business,report',
    example_package: '  kratos-taro-app create customer-app --with @acme/customer-module',
    https_environment_hint: 'When the backend uses HTTPS, set VITE_APP_API_URL=https://localhost:7001 in .env.development-h5.local.',
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

/** 根据环境变量选择 Taro CLI 文案语言。 */
export function cliMessage(key, params = {}) {
  const locale = resolveCliLocale()
  const template = messages[locale][key] ?? messages['en-US'][key] ?? key
  return template.replace(/\{([A-Za-z0-9_]+)\}/g, (_, name) => params[name] ?? `{${name}}`)
}
