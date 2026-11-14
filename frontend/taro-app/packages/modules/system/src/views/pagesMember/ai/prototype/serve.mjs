import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createServer } from 'node:http'
import { createRequire } from 'node:module'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const prototypeDir = dirname(fileURLToPath(import.meta.url))
const workspaceDir = resolve(prototypeDir, '../../../../../../../../')
const requireWorkspace = createRequire(resolve(workspaceDir, 'package.json'))
const requireHost = createRequire(resolve(workspaceDir, 'apps/taro-app/package.json'))
const requireRunner = createRequire(requireHost.resolve('@tarojs/webpack5-runner/package.json'))
const sass = requireWorkspace('sass')
const postcss = requireRunner('postcss')
const pxtransform = requireRunner('postcss-pxtransform')

// 直接编译现有组件样式，使用宿主相同的 750px → H5 rem 转换。
const stylePaths = [
  'packages/ui/src/styles/theme.scss',
  'packages/ui/src/styles/uni-icons.scss',
  'packages/core/src/styles/base.scss',
  'packages/core/src/components/KratosNavBar.scss',
  'packages/core/src/components/KratosTabBar.scss',
  'packages/modules/system/src/views/pagesMember/ai/index.scss',
  'packages/modules/system/src/views/pagesMember/ai/components/composer.scss',
  'packages/modules/system/src/views/pagesMember/ai/components/SessionDrawer.scss',
  'packages/modules/system/src/views/pagesMember/ai/components/WelcomePanel.scss',
  'packages/modules/system/src/views/pagesMember/ai/components/KnowledgePicker.scss',
]
const source = stylePaths.map(path => `@use '${path}';`).join('\n')
const compiled = sass.compileString(`${source}\n@use 'preview.scss';`, { loadPaths: [workspaceDir, prototypeDir] })
const result = await postcss([pxtransform({ platform: 'h5', designWidth: 750 })])
  .process(compiled.css, { from: undefined })
const css = result.css.replace("../static/fonts/uniicons.ttf", '/uniicons.ttf')
assert(css.includes('width: 14rem'), '历史抽屉必须保持 Taro 的 560px 设计宽度')
assert(css.includes('font-size: 0.8rem'), '导航字号必须经过 Taro 尺寸转换')

const routes = new Map([
  ['/', ['text/html; charset=utf-8', readFileSync(resolve(prototypeDir, 'index.html'))]],
  ['/preview.css', ['text/css; charset=utf-8', css]],
  ['/app.mjs', ['text/javascript; charset=utf-8', readFileSync(resolve(prototypeDir, 'app.mjs'))]],
  ['/demo-data.mjs', ['text/javascript; charset=utf-8', readFileSync(resolve(prototypeDir, 'demo-data.mjs'))]],
  ['/uniicons.ttf', ['font/ttf', readFileSync(resolve(workspaceDir, 'packages/ui/src/static/fonts/uniicons.ttf'))]],
])
createServer((request, response) => {
  const route = routes.get(new URL(request.url, 'http://127.0.0.1').pathname)
  if (!route) {
    response.writeHead(404).end()
    return
  }
  response.writeHead(200, { 'Content-Type': route[0], 'Cache-Control': 'no-store' })
  response.end(route[1])
}).listen(4174, '127.0.0.1', () => {
  process.stdout.write('Taro 原样预览：http://127.0.0.1:4174/?drawer=1\n')
})
