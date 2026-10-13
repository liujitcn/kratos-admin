import { existsSync, readFileSync, readdirSync } from 'node:fs'
import { dirname, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const workspaceRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const outputRoot = process.argv[2]
  ? resolve(process.argv[2])
  : resolve(workspaceRoot, 'apps/uni-app', process.env.UNI_OUTPUT_DIR || 'dist/build/mp-weixin')
const app = JSON.parse(readFileSync(resolve(outputRoot, 'app.json'), 'utf8'))
const violations = []
let checked = 0

for (const templateFile of collectTemplates(outputRoot)) {
  const template = readFileSync(templateFile, 'utf8')
  const tags = new Set([...template.matchAll(/<(kratos-[\w-]+)\b/g)].map((match) => match[1]))
  if (!tags.size) continue
  const config = JSON.parse(readFileSync(templateFile.replace(/\.wxml$/, '.json'), 'utf8'))
  const components = { ...app.usingComponents, ...config.usingComponents }
  for (const tag of tags) {
    checked++
    const target = components[tag]
    if (!target) {
      violations.push(`${relative(outputRoot, templateFile)}: ${tag} 未注册到 usingComponents`)
      continue
    }
    const componentPath = target.startsWith('/')
      ? resolve(outputRoot, target.slice(1))
      : resolve(dirname(templateFile), target)
    if (!existsSync(`${componentPath}.wxml`) || !existsSync(`${componentPath}.json`)) {
      violations.push(`${relative(outputRoot, templateFile)}: ${tag} 的组件产物不存在`)
    }
  }
}

if (violations.length) {
  console.error(`微信小程序组件注册检查失败：\n${violations.map((item) => `  ${item}`).join('\n')}`)
  process.exit(1)
}
console.log(`微信小程序组件注册检查通过（${checked} 处组件引用）`)

/** collectTemplates 收集编译产物中的 WXML，用实际调用位置检查小程序组件注册。 */
function collectTemplates(directory) {
  const templates = []
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const path = resolve(directory, entry.name)
    if (entry.isDirectory()) templates.push(...collectTemplates(path))
    else if (entry.name.endsWith('.wxml')) templates.push(path)
  }
  return templates
}
