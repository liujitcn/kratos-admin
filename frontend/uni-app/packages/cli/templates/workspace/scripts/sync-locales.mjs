import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { workspaceMessage } from './locale-messages.mjs'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const modulesRoot = resolve(root, 'packages/modules')
const write = process.argv.includes('--write')
let expectedLocales
for (const entry of existsSync(modulesRoot)
  ? readdirSync(modulesRoot, { withFileTypes: true })
  : []) {
  if (!entry.isDirectory()) continue
  const directory = resolve(modulesRoot, entry.name, 'src/locales')
  const locales = readdirSync(directory)
    .filter((file) => file.endsWith('.json'))
    .map((file) => file.slice(0, -5))
    .sort((a, b) => (a === 'zh-CN' ? -1 : b === 'zh-CN' ? 1 : a.localeCompare(b)))
  if (!locales.includes('zh-CN')) {
    throw new Error(workspaceMessage('missing_default_locale', { module: entry.name }))
  }
  if (expectedLocales && locales.join(',') !== expectedLocales)
    throw new Error(workspaceMessage('locale_set_mismatch', { module: entry.name }))
  expectedLocales = locales.join(',')
  const messages = Object.fromEntries(
    locales.map((locale) => {
      if (!/^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$/.test(locale))
        throw new Error(workspaceMessage('locale_code_invalid', { locale }))
      return [locale, JSON.parse(readFileSync(resolve(directory, `${locale}.json`), 'utf8'))]
    }),
  )
  const keys = Object.keys(messages['zh-CN']).sort().join('\0')
  for (const locale of locales) {
    if (Object.keys(messages[locale]).sort().join('\0') !== keys)
      throw new Error(workspaceMessage('key_set_mismatch', { module: entry.name, locale }))
  }
  // 模块在 Vite 配置阶段由 Node 直接加载，使用无需 TS/JSON loader 的原生 ESM。
  const content = [
    '/* 此文件由 scripts/sync-locales.mjs 生成，请勿手工修改。 */',
    `export const LOCALE_MESSAGES = ${JSON.stringify(messages, null, 2)}`,
    "export const DEFAULT_LOCALE = 'zh-CN'",
    'export const SUPPORTED_LOCALES = Object.keys(LOCALE_MESSAGES)',
    '',
  ].join('\n')
  const file = resolve(directory, 'generated.mjs')
  if (write) writeFileSync(file, content)
  else if (!existsSync(file) || readFileSync(file, 'utf8') !== content)
    throw new Error(workspaceMessage('bundle_outdated', { file }))
}
