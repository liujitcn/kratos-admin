type RunnerLocale = 'zh-CN' | 'en-US'

const messages: Record<RunnerLocale, Record<string, string>> = {
  'zh-CN': {
    unsupported_type: '不支持的 Taro 类型：{value}',
    unknown_argument: '未知参数：{argument}',
    type_required: '缺少 --type h5|weapp',
    manifest_missing: '模块清单未导出 moduleManifest：{file}',
    static_import_missing: '模块 {name} 缺少静态 import',
    build_export_missing: '{packageName}/build 未导出 buildModule',
    transaction_mismatch: 'Taro 构建事务目录不匹配：{file}',
    assembly_timeout: '等待 Taro 页面装配锁超时：{file}',
  },
  'en-US': {
    unsupported_type: 'Unsupported Taro type: {value}',
    unknown_argument: 'Unknown argument: {argument}',
    type_required: 'Missing --type h5|weapp',
    manifest_missing: 'The module manifest does not export moduleManifest: {file}',
    static_import_missing: 'Module {name} is missing a static import',
    build_export_missing: '{packageName}/build does not export buildModule',
    transaction_mismatch: 'Taro build transaction directory does not match: {file}',
    assembly_timeout: 'Timed out waiting for the Taro page assembly lock: {file}',
  },
}

/** 根据环境变量选择 Taro 构建器错误语言。 */
function resolveRunnerLocale(): RunnerLocale {
  const value = process.env.KRATOS_ADMIN_LOCALE ?? process.env.LC_ALL ?? process.env.LANG ?? ''
  return value.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US'
}

/** 渲染 Taro 构建器稳定错误文案。 */
export function runnerMessage(key: string, params: Record<string, string> = {}): string {
  const template = messages[resolveRunnerLocale()][key] ?? messages['en-US'][key] ?? key
  return template.replace(/\{([A-Za-z0-9_]+)\}/g, (_, name: string) => params[name] ?? `{${name}}`)
}
