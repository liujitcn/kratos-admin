import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { runInNewContext } from 'node:vm'
import test from 'node:test'
import { transform } from 'esbuild'

/** loadProfile 用实际页面和可控请求复现首屏加载，保留 Hook 状态以检查后续渲染。 */
async function loadProfile() {
  const source = await readFile(new URL('../src/views/pagesMember/profile/profile.tsx', import.meta.url), 'utf8')
  const { code } = await transform(source, { loader: 'tsx', format: 'cjs', jsxFactory: 'jsx', jsxFragment: 'fragment', define: { 'process.env.TARO_ENV': '"h5"' } })
  const state = []
  let cursor = 0
  let onLoad
  let resolveProfile
  let rejectProfile
  let resolveGender
  const profile = new Promise((resolve, reject) => { resolveProfile = resolve; rejectProfile = reject })
  const gender = new Promise((resolve) => { resolveGender = resolve })
  const store = { ensureAuthenticated: () => true, getUserProfile: () => profile }
  const useUserStore = (selector) => selector(store)
  useUserStore.getState = () => store
  const pageModule = { exports: {} }
  runInNewContext(code, {
    module: pageModule, exports: pageModule.exports,
    jsx: (type, props, ...children) => ({ type, props: props || {}, children: children.flat(Infinity) }),
    fragment: 'fragment',
    require(name) {
      if (name === 'react') return {
        useState(initial) {
          const index = cursor++
          if (!(index in state)) state[index] = typeof initial === 'function' ? initial() : initial
          return [state[index], (next) => { state[index] = typeof next === 'function' ? next(state[index]) : next }]
        },
        useRef(initial) {
          const index = cursor++
          if (!(index in state)) state[index] = { current: initial }
          return state[index]
        },
        useEffect: () => {},
      }
      if (name === '@tarojs/components') return Object.fromEntries(['Button', 'Image', 'Input', 'Label', 'Picker', 'Radio', 'RadioGroup', 'Text', 'View'].map(tag => [tag, tag]))
      if (name === '@tarojs/taro') return { useLoad: (callback) => { onLoad ||= callback } }
      if (name.endsWith('/base_dict')) return { defBaseDictService: { GetBaseDict: () => gender } }
      if (name.endsWith('/common')) return { BaseUserIDType: Object.fromEntries(['UNSPECIFIED', 'ID_CARD', 'PASSPORT', 'HK_MACAO_PERMIT', 'TAIWAN_PERMIT', 'FOREIGN_PERMANENT_RESIDENCE', 'OTHER'].map((idType, value) => [`BASE_USER_ID_TYPE_${idType}`, value])) }
      if (name === '@liujitcn/kratos-taro-app-core') return {
        useUserStore,
        useI18n: () => ({ t: key => key }),
        useThemeStore: () => ({ theme: 'light', colors: {} }),
        KratosNavBar: 'NavBar',
      }
      return {}
    },
  })
  return {
    render: () => { cursor = 0; return pageModule.exports.default() },
    start: () => onLoad(), resolveProfile, rejectProfile, resolveGender,
  }
}

/** findNode 按渲染结果定位表单区域或输入控件。 */
function findNode(node, predicate) {
  if (!node || typeof node !== 'object') return undefined
  if (predicate(node)) return node
  for (const child of node.children || []) {
    const found = findNode(child, predicate)
    if (found) return found
  }
}

test('资料和字典加载齐之前不显示空表单，加载后一次展示真实内容', async () => {
  const runtime = await loadProfile()
  const isForm = node => node.props.className?.includes('profile-form')
  assert.equal(findNode(runtime.render(), isForm), undefined)
  runtime.start()
  runtime.resolveProfile({ user_name: 'tester', nick_name: 'Test', email: 'profile@example.test', id_type: 1 })
  await new Promise(setImmediate)
  assert.equal(findNode(runtime.render(), isForm), undefined)
  runtime.resolveGender({ items: [{ value: '1', label: 'Test gender' }] })
  await new Promise(setImmediate)
  const tree = runtime.render()
  assert.ok(findNode(tree, isForm))
  assert.equal(findNode(tree, node => node.type === 'Input' && node.props.value === 'profile@example.test').props.value, 'profile@example.test')
  assert.ok(findNode(tree, node => node.type === 'Picker' && node.props.value === 1))
})

test('资料加载失败时保留错误提示，不开放默认值表单', async () => {
  const runtime = await loadProfile()
  runtime.render()
  runtime.start()
  runtime.rejectProfile(new Error('profile unavailable'))
  runtime.resolveGender({ items: [] })
  await new Promise(setImmediate)
  const tree = runtime.render()
  assert.equal(findNode(tree, node => node.props.className?.includes('profile-form')), undefined)
  assert.ok(findNode(tree, node => node.children?.includes('core.status.PAGE_UNAVAILABLE')))
})
