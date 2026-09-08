import { existsSync, readdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const modulesRoot = resolve(root, 'packages/modules')
const write = process.argv.includes('--write')
let expectedLocales
for (const entry of existsSync(modulesRoot) ? readdirSync(modulesRoot, { withFileTypes: true }) : []) {
  if (!entry.isDirectory()) continue
  const directory = resolve(modulesRoot, entry.name, 'src/locales')
  const locales = readdirSync(directory).filter(file => file.endsWith('.json')).map(file => file.slice(0, -5))
    .sort((a, b) => a === 'zh-CN' ? -1 : b === 'zh-CN' ? 1 : a.localeCompare(b))
  if (!locales.includes('zh-CN')) throw new Error(`${entry.name} 缺少 zh-CN 语言包`)
  if (expectedLocales && locales.join(',') !== expectedLocales) throw new Error(`${entry.name} 语言集合不一致`)
  expectedLocales = locales.join(',')
  const messages = Object.fromEntries(locales.map(locale => {
    if (!/^[A-Za-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$/.test(locale)) throw new Error(`语言代码无效：${locale}`)
    return [locale, JSON.parse(readFileSync(resolve(directory, `${locale}.json`), 'utf8'))]
  }))
  const keys = Object.keys(messages['zh-CN']).sort().join('\0')
  for (const locale of locales) {
    if (Object.keys(messages[locale]).sort().join('\0') !== keys) throw new Error(`${entry.name}/${locale} 语言键集合不一致`)
  }
  const imports = locales.map((locale, index) => `import locale${index} from './${locale}.json'`)
  const content = [
    '/* 此文件由 scripts/sync-locales.mjs 生成，请勿手工修改。 */',
    ...imports, '', 'export const LOCALE_MESSAGES = {',
    ...locales.map((locale, index) => `  '${locale}': locale${index},`),
    '} as const satisfies Record<string, Record<string, string>>', '',
    'export type GeneratedLocale = keyof typeof LOCALE_MESSAGES',
    "export const DEFAULT_LOCALE: GeneratedLocale = 'zh-CN'",
    'export const SUPPORTED_LOCALES = Object.keys(LOCALE_MESSAGES) as GeneratedLocale[]', '',
  ].join('\n')
  const file = resolve(directory, 'generated.ts')
  if (write) writeFileSync(file, content)
  else if (!existsSync(file) || readFileSync(file, 'utf8') !== content) throw new Error(`语言注册产物过期：${file}，请执行 pnpm i18n:sync`)
}
