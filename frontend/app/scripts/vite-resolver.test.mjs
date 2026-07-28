import assert from 'node:assert/strict'
import { existsSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { loadConfigFromFile } from 'vite'

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const loaded = await loadConfigFromFile(
  { command: 'serve', mode: 'development-h5' },
  resolve(packageRoot, 'vite.config.ts'),
  undefined,
  'silent',
)
const plugins = (loaded?.config.plugins ?? []).flat(Infinity)
const plugin = plugins.find((item) => item?.name === 'kratos-app-pages')

assert.ok(plugin, '应加载 kratos-app-pages 插件')

const importer = resolve(packageRoot, 'src/api/base/login.ts')
const resolved = await plugin.resolveId('@/utils/http', importer)

assert.equal(resolved, resolve(packageRoot, 'src/utils/http.ts'))
assert.ok(existsSync(resolved), '共享包别名应解析为存在的源码文件')

const hostImporter = resolve(packageRoot, '../host/src/main.ts')
assert.equal(await plugin.resolveId('@/utils/http', hostImporter), undefined)

const pluginConfig = await plugin.config()
const optimizePlugin = pluginConfig?.optimizeDeps?.esbuildOptions?.plugins?.find(
  (item) => item.name === 'kratos-app-optimize-deps-resolver',
)

assert.ok(optimizePlugin, '应注册共享包依赖预构建解析器')

let optimizeResolver
optimizePlugin.setup({
  onResolve(options, callback) {
    if (options.filter.test('@/utils/http')) {
      optimizeResolver = callback
    }
  },
})

assert.ok(optimizeResolver, '依赖预构建解析器应处理 @/ 导入')
assert.deepEqual(await optimizeResolver({ path: '@/utils/http', importer }), {
  path: resolve(packageRoot, 'src/utils/http.ts'),
})
assert.equal(await optimizeResolver({ path: '@/utils/http', importer: hostImporter }), undefined)
