import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import test from 'node:test'

const navigation = readFileSync(resolve(import.meta.dirname, '../src/navigation.ts'), 'utf8')

test('tab 导航忽略当前页和未完成的重复 reLaunch', () => {
  assert.match(navigation, /if \(resolved\.menu\.inTabBar\) \{\s+navigateTabRoute\(url\)/)
  assert.match(
    navigation,
    /if \(currentPage\?\.route === targetRoute \|\| tabNavigationTarget\) return/,
  )
  assert.match(
    navigation,
    /uni\.reLaunch\(\{ url, success: release, fail: release, complete: release \}\)/,
  )
})
