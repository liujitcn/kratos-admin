import assert from 'node:assert/strict'
import { mkdtempSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import test from 'node:test'
import { build } from 'esbuild'

async function loadFormatSrc(taroEnv) {
  const root = mkdtempSync(resolve(tmpdir(), 'kratos-taro-format-src-'))
  const output = resolve(root, 'utils.mjs')
  await build({
    entryPoints: [resolve(import.meta.dirname, '../src/utils/index.ts')],
    bundle: true,
    platform: 'browser',
    format: 'esm',
    outfile: output,
    define: {
      'process.env.TARO_ENV': JSON.stringify(taroEnv),
      'process.env.VITE_APP_API_URL': JSON.stringify('https://api.example.com'),
      'process.env.VITE_APP_STATIC_URL': JSON.stringify('https://static.example.com'),
    },
  })
  return {
    formatSrc: (await import(pathToFileURL(output).href)).formatSrc,
    dispose: () => rmSync(root, { recursive: true, force: true }),
  }
}

test('微信小程序忽略 Taro window 占位域名并使用静态资源域名', async () => {
  const originalWindow = global.window
  const runtime = await loadFormatSrc('weapp')
  try {
    global.window = { location: { origin: 'https://taro.com' } }
    assert.equal(
      runtime.formatSrc('/data/config/images/logo.jpg'),
      'https://static.example.com/data/config/images/logo.jpg',
    )
  } finally {
    global.window = originalWindow
    runtime.dispose()
  }
})

test('H5 静态资源继续使用浏览器当前域名', async () => {
  const originalWindow = global.window
  const runtime = await loadFormatSrc('h5')
  try {
    global.window = { location: { origin: 'https://app.example.com' } }
    assert.equal(
      runtime.formatSrc('/data/config/images/logo.jpg'),
      'https://app.example.com/data/config/images/logo.jpg',
    )
  } finally {
    global.window = originalWindow
    runtime.dispose()
  }
})
