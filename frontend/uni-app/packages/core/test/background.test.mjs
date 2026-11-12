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

test('H5 picker 弹层提供暗色主题覆盖', () => {
  assert.match(baseStyles, /theme-dark[\s\S]*uni-picker-container[\s\S]*uni-picker-select/)
  assert.match(baseStyles, /uni-picker-action-confirm[\s\S]*kratos-color-primary/)
})

test('H5 modal 保留旧版按钮边界样式', () => {
  assert.match(baseStyles, /uni-modal__ft::after[\s\S]*uni-modal__btn::after/)
})
