import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import test from 'node:test'

const navigation = readFileSync(resolve(import.meta.dirname, '../src/navigation.ts'), 'utf8')

test('tab 导航复用页面栈并忽略当前页和未完成的重复跳转', () => {
  assert.match(navigation, /if \(options\.replace\) \{\s+uni\.reLaunch\(\{ url \}\)/)
  assert.match(navigation, /if \(resolved\.menu\.inTabBar\) \{\s+navigateTabRoute\(url\)/)
  assert.match(navigation, /if \(currentRoute === targetRoute \|\| tabNavigationTarget\) return/)
  assert.match(navigation, /uni\.switchTab\(\{\s+url,/)
  assert.match(navigation, /uni\.navigateBack\(\{\s+delta: currentIndex - targetIndex,/)
  assert.match(navigation, /uni\.navigateTo\(\{\s+url,/)
})
