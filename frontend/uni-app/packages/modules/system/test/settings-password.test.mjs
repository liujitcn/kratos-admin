import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import test from 'node:test'

test('修改密码入口只在 H5 装配并调用 Base 账户安全接口', async () => {
  const settings = await readFile(
    resolve(import.meta.dirname, '../src/views/pagesMember/settings/settings.vue'),
    'utf8',
  )
  const component = await readFile(
    resolve(
      import.meta.dirname,
      '../src/views/pagesMember/settings/components/ChangePasswordEntry.vue',
    ),
    'utf8',
  )
  const page = await readFile(
    resolve(
      import.meta.dirname,
      '../src/views/pagesMember/password/components/ChangePasswordPage.vue',
    ),
    'utf8',
  )
  const pages = await readFile(resolve(import.meta.dirname, '../src/pages.ts'), 'utf8')
  const api = await readFile(
    resolve(import.meta.dirname, '../../../core/src/api/base/v1/account.ts'),
    'utf8',
  )

  assert.match(settings, /#ifdef H5[\s\S]*ChangePasswordEntry[\s\S]*#endif/)
  assert.match(component, /navigateAppRoute\('app\/password'\)/)
  assert.doesNotMatch(component, /change-password-dialog/)
  assert.match(page, /KratosNavBar/)
  assert.match(page, /defAccountService\.GetCurrentPasswordPolicy/)
  assert.match(page, /defAccountService\.UpdateUserPassword/)
  assert.match(page, /password_policy_min_length/)
  assert.match(page, /reLaunch\(\{ url: '\/pages\/login\/login' \}\)/)
  assert.match(pages, /pagesMember\/password\/password/)
  assert.match(api, /const ACCOUNT_URL = '\/v1\/base\/account'/)
  assert.doesNotMatch(component, /\/v1\/admin\/auth/)
  assert.doesNotMatch(api, /\/v1\/admin\/auth/)
})
