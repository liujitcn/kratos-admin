import { readdir, readFile, stat } from 'node:fs/promises'
import { dirname, extname, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import ts from 'typescript'
import { workspaceMessage } from './locale-messages.mjs'

const appRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const coreRoot = resolve(appRoot, 'packages/core')
const systemRoot = resolve(appRoot, 'packages/modules/system')
const ignoredDirectories = new Set(['dist', 'node_modules'])
const sourceExtensions = new Set(['.js', '.mjs', '.ts', '.vue'])
const allFiles = await collectFiles(appRoot)
const packageFiles = allFiles.filter((file) => file.endsWith(`${sep}package.json`))
const packages = new Map()
const publicPackageVersions = new Map()
const violations = []

for (const packageFile of packageFiles) {
  const metadata = JSON.parse(await readFile(packageFile, 'utf8'))
  if (!metadata.name || !metadata.exports) continue
  if (!metadata.private) {
    publicPackageVersions.set(metadata.name, metadata.version)
  }
  const packageRoot = dirname(packageFile)
  const exportEntries = Object.entries(metadata.exports)
  packages.set(metadata.name, { packageRoot, exportEntries })
  for (const [, value] of exportEntries) {
    for (const target of collectTargets(value)) {
      const prefix = target.includes('*') ? target.slice(0, target.indexOf('*')) : target
      const checkPath = target.includes('*')
        ? resolve(packageRoot, prefix)
        : resolve(packageRoot, target)
      if (!(await pathExists(checkPath))) {
        throw new Error(workspaceMessage('export_target_missing', { name: metadata.name, target }))
      }
    }
  }
}

const distinctPublicVersions = new Set(publicPackageVersions.values())
if (distinctPublicVersions.size > 1 || [...distinctPublicVersions].some((version) => !version)) {
  const versions = [...publicPackageVersions]
    .map(([name, version]) => `${name}@${version}`)
    .join(workspaceMessage('list_separator'))
  throw new Error(workspaceMessage('public_version_mismatch', { versions }))
}

for (const sourceFile of allFiles.filter((file) => sourceExtensions.has(extname(file)))) {
  if (sourceFile.includes(`${sep}src${sep}rpc${sep}`) || isGeneratedEntry(sourceFile)) continue
  const source = await readFile(sourceFile, 'utf8')
  for (const imported of ts.preProcessFile(source, true, true).importedFiles) {
    const specifier = imported.fileName
    const line = source.slice(0, imported.pos).split('\n').length
    if ((specifier === '@' || specifier.startsWith('@/')) && !isWithin(sourceFile, coreRoot)) {
      violations.push(
        workspaceMessage('core_alias_forbidden', { location: format(sourceFile, line), specifier }),
      )
      continue
    }
    if (specifier.startsWith('@system/') && !isWithin(sourceFile, systemRoot)) {
      violations.push(
        workspaceMessage('system_alias_forbidden', {
          location: format(sourceFile, line),
          specifier,
        }),
      )
      continue
    }
    if (specifier.startsWith('.')) {
      const sourcePackage = findPackageByPath(sourceFile)
      const targetPackage = findPackageByPath(resolve(dirname(sourceFile), specifier))
      if (sourcePackage && targetPackage && sourcePackage !== targetPackage) {
        violations.push(
          workspaceMessage('cross_package_relative_forbidden', {
            location: format(sourceFile, line),
            specifier,
          }),
        )
      }
      continue
    }
    const packageName = [...packages.keys()].find(
      (name) => specifier === name || specifier.startsWith(`${name}/`),
    )
    if (!packageName) continue
    const subpath = specifier === packageName ? '.' : `./${specifier.slice(packageName.length + 1)}`
    const exported = packages
      .get(packageName)
      .exportEntries.some(([pattern]) => matchExportPattern(pattern, subpath))
    if (!exported) {
      violations.push(
        workspaceMessage('export_not_public', {
          location: format(sourceFile, line),
          specifier,
          packageName,
        }),
      )
    }
  }
}

if (violations.length) {
  console.error(
    workspaceMessage('exports_check_failed', {
      violations: violations.map((item) => `  ${item}`).join('\n'),
    }),
  )
  process.exit(1)
}
console.log(
  workspaceMessage('exports_check_passed', {
    packages: [...packages.keys()].join(workspaceMessage('list_separator')),
  }),
)

async function collectFiles(directory) {
  const files = []
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    if (entry.isDirectory() && ignoredDirectories.has(entry.name)) continue
    const path = resolve(directory, entry.name)
    if (entry.isDirectory()) files.push(...(await collectFiles(path)))
    else files.push(path)
  }
  return files
}

function collectTargets(value) {
  if (typeof value === 'string') return [value]
  if (!value || typeof value !== 'object') return []
  return Object.values(value).flatMap(collectTargets)
}

async function pathExists(path) {
  try {
    await stat(path)
    return true
  } catch {
    return false
  }
}

function matchExportPattern(pattern, subpath) {
  if (!pattern.includes('*')) return pattern === subpath
  const escaped = pattern.replace(/[.*+?^${}()|[\]\\]/g, '\\$&').replace('\\*', '.+')
  return new RegExp(`^${escaped}$`).test(subpath)
}

function findPackageByPath(file) {
  return [...packages.values()].find(({ packageRoot }) => isWithin(file, packageRoot))
}

function isWithin(file, directory) {
  const path = relative(directory, file)
  return path === '' || (!path.startsWith('..') && !path.startsWith(sep))
}

function isGeneratedEntry(file) {
  return (
    file === resolve(coreRoot, 'src/module.mjs') ||
    file === resolve(coreRoot, 'src/vite.mjs') ||
    file === resolve(systemRoot, 'src/index.mjs')
  )
}

function format(file, line) {
  return `${relative(appRoot, file)}:${line}`
}
