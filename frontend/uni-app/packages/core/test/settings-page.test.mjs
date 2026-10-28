import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import test from 'node:test'
import { parse } from '@vue/compiler-sfc'

test('设置页把 extensions 插槽直接放在公共设置之前', async () => {
  const source = await readFile(
    resolve(import.meta.dirname, '../src/components/KratosSettingsPage.vue'),
    'utf8',
  )
  const { descriptor } = parse(source)
  const template = descriptor.template?.content || ''
  const styles = descriptor.styles.map((style) => style.content).join('\n')

  assert.match(
    template,
    /<view class="settings-page" :class="themeStore.themeClass">\s*<slot name="extensions" \/>/,
  )
  assert.ok(
    template.indexOf('<slot name="extensions" />') <
      template.indexOf('<view class="settings-content">'),
  )
  assert.match(styles, /\.settings-row__value\s*\{[\s\S]*?min-width:\s*0/)
  assert.match(styles, /\.settings-row__value\s*\{[\s\S]*?overflow:\s*hidden/)
  assert.match(styles, /\.settings-row__value\s*\{[\s\S]*?text-overflow:\s*ellipsis/)
  assert.match(template, /core\.settings\.theme/)
  assert.match(template, /<slot name="security" \/>/)
  assert.ok(
    template.indexOf('core.settings.section_security') <
      template.indexOf('core.settings.section_appearance'),
  )
  assert.ok(
    template.indexOf('common.field.language') >
      template.indexOf('core.settings.section_appearance'),
  )
})
