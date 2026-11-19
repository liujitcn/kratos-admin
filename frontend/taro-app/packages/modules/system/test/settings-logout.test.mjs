import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'

test('设置页退出使用真实模块源码并在返回失败时重定向登录', async () => {
  const source = await readFile(
    new URL('../src/views/pagesMember/settings/settings.tsx', import.meta.url),
    'utf8',
  )

  assert.match(source, /className='k-set-logout'/)
  assert.match(source, /await logout\(\)/)
  assert.match(source, /await Taro\.navigateBack\(\)/)
  assert.match(source, /catch \{[\s\S]*?navigateToLogin\(\)/)
})
