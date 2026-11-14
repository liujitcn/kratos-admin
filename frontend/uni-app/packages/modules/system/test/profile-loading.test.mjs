import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import { runInNewContext } from 'node:vm'
import test from 'node:test'
import { compileScript, parse } from '@vue/compiler-sfc'
import * as vue from 'vue'
import ts from 'typescript'

const require = createRequire(import.meta.url)
const fromVue = createRequire(require.resolve('vue/package.json'))
const { renderToString } = fromVue('@vue/server-renderer')

/** loadProfile 编译真实模板，用可控请求检查加载前后渲染内容。 */
async function loadProfile() {
  const source = await readFile(
    new URL('../src/views/pagesMember/profile/profile.uvue', import.meta.url),
    'utf8',
  )
  const { descriptor } = parse(source, { filename: 'profile.vue' })
  const script = compileScript(descriptor, { id: 'profile-test', inlineTemplate: true })
  const { outputText } = ts.transpileModule(script.content, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  })
  let onLoad
  let resolveProfile
  let rejectProfile
  let resolveGender
  const profile = new Promise((resolve, reject) => {
    resolveProfile = resolve
    rejectProfile = reject
  })
  const gender = new Promise((resolve) => {
    resolveGender = resolve
  })
  const store = { ensureAuthenticated: () => true, userInfo: {} }
  const exports = {}
  runInNewContext(outputText, {
    exports,
    require(name) {
      if (name === 'vue')
        return {
          ...vue,
          onBeforeUnmount: () => {},
          resolveComponent: (name) =>
            ['picker', 'radio', 'radio-group'].includes(name) ? name : vue.resolveComponent(name),
        }
      if (name === '@dcloudio/uni-app')
        return {
          onLoad: (callback) => {
            onLoad = callback
          },
        }
      if (name.endsWith('/stores'))
        return {
          useUserStore: () => store,
          useThemeStore: () => ({ themeClass: 'theme-light', colors: {} }),
        }
      if (name.endsWith('/auth')) return { defAuthService: { GetUserProfile: () => profile } }
      if (name.endsWith('/base_dict')) return { defBaseDictService: { GetBaseDict: () => gender } }
      if (name.endsWith('/common'))
        return {
          BaseUserIDType: Object.fromEntries(
            [
              'UNSPECIFIED',
              'ID_CARD',
              'PASSPORT',
              'HK_MACAO_PERMIT',
              'TAIWAN_PERMIT',
              'FOREIGN_PERMANENT_RESIDENCE',
              'OTHER',
            ].map((name, value) => [`BASE_USER_ID_TYPE_${name}`, value]),
          ),
        }
      if (name === '@liujitcn/kratos-uni-app-core') return { useI18n: () => ({ t: (key) => key }) }
      if (name.endsWith('KratosNavBar.uvue')) return { default: 'nav' }
      return {}
    },
  })
  const render = exports.default.setup({}, { expose: () => {} })
  return {
    render: () => renderToString(vue.createSSRApp({ render })),
    start: () => onLoad(),
    resolveProfile,
    rejectProfile,
    resolveGender,
  }
}

test('资料和字典加载齐之前不显示空表单，加载后一次展示真实内容', async () => {
  const runtime = await loadProfile()
  assert.doesNotMatch(await runtime.render(), /class="[^"]*profile-form/)
  runtime.start()
  runtime.resolveProfile({
    user_name: 'tester',
    nick_name: 'Test',
    email: 'profile@example.test',
    id_type: 1,
  })
  await new Promise(setImmediate)
  assert.doesNotMatch(await runtime.render(), /class="[^"]*profile-form/)
  runtime.resolveGender({ items: [{ value: '1', label: 'Test gender' }] })
  await new Promise(setImmediate)
  const html = await runtime.render()
  assert.match(html, /class="[^"]*profile-form/)
  assert.match(html, /profile@example\.test/)
  assert.match(html, /system\.profile\.id_type\.id_card/)
})

test('资料加载失败时保留错误提示，不开放默认值表单', async () => {
  const runtime = await loadProfile()
  runtime.start()
  runtime.rejectProfile(new Error('profile unavailable'))
  runtime.resolveGender({ items: [] })
  await new Promise(setImmediate)
  const html = await runtime.render()
  assert.doesNotMatch(html, /class="[^"]*profile-form/)
  assert.match(html, /core\.status\.PAGE_UNAVAILABLE/)
})
