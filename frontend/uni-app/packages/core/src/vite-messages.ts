type ViteLocale = 'zh-CN' | 'en-US'

const messages: Record<ViteLocale, Record<string, string>> = {
  'zh-CN': {
    https_certificate_missing:
      'VITE_APP_HTTPS 已开启，但未找到证书文件，请先在仓库根目录运行 scripts/generate-dev-cert.sh；期望路径：{keyPath} 和 {certPath}',
    transaction_mismatch: 'uni-app 构建事务目录不匹配：{file}',
    assembly_timeout: '等待 uni-app 页面装配锁超时：{file}',
    source_path_not_absolute: '模块源码路径不是绝对路径：{path}',
  },
  'en-US': {
    https_certificate_missing:
      'VITE_APP_HTTPS is enabled, but certificate files were not found. Run scripts/generate-dev-cert.sh from the repository root; expected paths: {keyPath} and {certPath}',
    transaction_mismatch: 'The uni-app build transaction directory does not match: {file}',
    assembly_timeout: 'Timed out waiting for the uni-app page assembly lock: {file}',
    source_path_not_absolute: 'The module source path is not absolute: {path}',
  },
}

/** 根据环境变量选择 uni-app 构建器错误语言。 */
function resolveViteLocale(): ViteLocale {
  const value = process.env.KRATOS_ADMIN_LOCALE ?? process.env.LC_ALL ?? process.env.LANG ?? ''
  return value.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en-US'
}

/** 渲染 uni-app 构建器稳定错误文案。 */
export function viteMessage(key: string, params: Record<string, string> = {}): string {
  const template = messages[resolveViteLocale()][key] ?? messages['en-US'][key] ?? key
  return template.replace(/\{([A-Za-z0-9_]+)\}/g, (_, name: string) => params[name] ?? `{${name}}`)
}
