import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import test from 'node:test'
import ts from 'typescript'

const require = createRequire(new URL('../src/locales/index.ts', import.meta.url))

function loadLocales() {
  const source = readFileSync(new URL('../src/locales/index.ts', import.meta.url), 'utf8')
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  })
  const exports = {}
  const runtimeRequire = (id) => {
    if (id === './generated') return {
      DEFAULT_LOCALE: 'zh-CN',
      SUPPORTED_LOCALES: ['zh-CN', 'en-US', 'ja-JP', 'zh-TW'],
    }
    if (id === '@tarojs/taro') return { default: { setStorageSync() {} } }
    return require(id)
  }
  new Function('require', 'exports', outputText)(runtimeRequire, exports)
  return exports
}

test('自定义文案按语言与 key 覆盖，保留插值并在删除后恢复本地值', async () => {
  const runtime = loadLocales()
  const messages = Object.fromEntries(runtime.SUPPORTED_LOCALES.map((locale) => [locale, {
    'core.test.title': `${locale} {name}`,
    'common.message.unknown': 'unknown',
  }]))
  runtime.registerLocaleMessages([{ name: 'core', messages }])
  runtime.applyCustomLocaleMessages([
    { locale: 'zh-CN', key: 'core.test.title', value: '定制 {name}' },
    { locale: 'en_US', key: 'core.test.title', value: 'Custom {name}' },
    { locale: 'unsupported', key: 'core.test.title', value: '错误覆盖' },
    { locale: 'zh-CN', key: 'core.missing', value: '未知键' },
  ])
  assert.equal(runtime.t('core.test.title', { name: 'A' }), '定制 A')
  assert.equal(runtime.t('core.missing'), 'unknown')
  assert.equal(messages['zh-CN']['core.test.title'], 'zh-CN {name}')
  await runtime.setCurrentLocale('en-US')
  assert.equal(runtime.t('core.test.title', { name: 'B' }), 'Custom B')
  runtime.applyCustomLocaleMessages([{ locale: 'en-US', key: 'core.test.title', value: '' }])
  assert.equal(runtime.t('core.test.title', { name: 'B' }), 'en-US B')
  runtime.applyCustomLocaleMessages([])
  await runtime.setCurrentLocale('zh-CN')
  assert.equal(runtime.t('core.test.title', { name: 'C' }), 'zh-CN C')
})
