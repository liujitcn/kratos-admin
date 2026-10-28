import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import test from 'node:test'

const baseStyles = readFileSync(resolve(import.meta.dirname, '../src/styles/base.scss'), 'utf8')

test('页面壳层首屏使用主题背景兜底，避免首次导航黑屏', () => {
  assert.match(
    baseStyles,
    /html,[\s\S]*#app,[\s\S]*uni-page-wrapper,[\s\S]*background-color: var\(--kratos-color-background, #eef3f8\)/,
  )
})
