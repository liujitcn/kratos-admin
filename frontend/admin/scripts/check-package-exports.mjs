import { readdir, readFile } from "node:fs/promises";
import { dirname, extname, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";
import ts from "typescript";

const adminRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = resolve(adminRoot, "../..");
const coreSourceRoot = resolve(adminRoot, "packages/core/src");
const businessSourceRoot = resolve(adminRoot, "packages/modules");
const ignoredDirectories = new Set([".turbo", "dist", "node_modules"]);
const sourceExtensions = new Set([".cjs", ".js", ".jsx", ".mjs", ".ts", ".tsx", ".vue"]);

const packageJsonFiles = (await collectFiles(adminRoot)).filter(file => {
  return file.endsWith(`${sep}package.json`) && !file.includes(`${sep}templates${sep}`);
});
const exportedPackages = new Map();

for (const packageJsonFile of packageJsonFiles) {
  const packageJson = JSON.parse(await readFile(packageJsonFile, "utf8"));
  if (!packageJson.name || !packageJson.exports) continue;

  exportedPackages.set(packageJson.name, {
    exportEntries: Object.entries(packageJson.exports).filter(([, value]) => value !== null),
    packageRoot: dirname(packageJsonFile)
  });
}

const sourceFiles = (await collectFiles(adminRoot)).filter(file => sourceExtensions.has(extname(file)));
const violations = [];

for (const sourceFile of sourceFiles) {
  const source = await readFile(sourceFile, "utf8");
  const imports = ts.preProcessFile(source, true, true).importedFiles;

  for (const importedFile of imports) {
    const specifier = importedFile.fileName;
    const line = source.slice(0, importedFile.pos).split("\n").length;

    if ((specifier === "@" || specifier.startsWith("@/")) && !isWithin(sourceFile, coreSourceRoot)) {
      violations.push(`${formatLocation(sourceFile, line)}: 非 core 代码不得使用 core 私有别名 ${specifier}`);
      continue;
    }

    if (specifier.startsWith(".") && isWithin(sourceFile, businessSourceRoot)) {
      const sourcePackage = findWorkspacePackageByPath(sourceFile);
      const targetPackage = findWorkspacePackageByPath(resolve(dirname(sourceFile), specifier));
      if (targetPackage && sourcePackage !== targetPackage) {
        violations.push(`${formatLocation(sourceFile, line)}: 业务模块不得通过相对路径跨包引用 ${specifier}`);
      }
      continue;
    }

    const packageName = findWorkspacePackageName(specifier);
    if (!packageName) continue;

    const packageInfo = exportedPackages.get(packageName);
    if (!isExportedSubpath(packageName, specifier, packageInfo.exportEntries)) {
      violations.push(`${formatLocation(sourceFile, line)}: ${specifier} 未在 ${packageName} 的 package.json#exports 中公开`);
      continue;
    }
  }
}

if (violations.length > 0) {
  console.error(`前端 package exports 检查失败：\n${violations.map(item => `  ${item}`).join("\n")}`);
  process.exit(1);
}

console.log(`前端 package exports 检查通过（公开包：${[...exportedPackages.keys()].join("、")}）。`);

/** 递归收集目录中的文件。 */
async function collectFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    if (entry.isDirectory() && ignoredDirectories.has(entry.name)) continue;

    const entryPath = resolve(directory, entry.name);
    if (entry.isDirectory()) files.push(...(await collectFiles(entryPath)));
    if (entry.isFile()) files.push(entryPath);
  }

  return files;
}

/** 查找导入说明符所属的 workspace 包。 */
function findWorkspacePackageName(specifier) {
  return [...exportedPackages.keys()].find(packageName => {
    return specifier === packageName || specifier.startsWith(`${packageName}/`);
  });
}

/** 查找文件所属的 workspace 公开包。 */
function findWorkspacePackageByPath(file) {
  return [...exportedPackages.values()].find(packageInfo => isWithin(file, packageInfo.packageRoot));
}

/** 判断导入子路径是否由包公开。 */
function isExportedSubpath(packageName, specifier, exportEntries) {
  const subpath = specifier === packageName ? "." : `./${specifier.slice(packageName.length + 1)}`;
  return exportEntries.some(([pattern]) => {
    if (!pattern.includes("*")) return subpath === pattern;
    const matcher = new RegExp(`^${escapeRegExp(pattern).replace("\\*", ".+")}$`);
    return matcher.test(subpath);
  });
}

/** 判断目标路径是否位于指定目录内。 */
function isWithin(target, directory) {
  const relativePath = relative(directory, target);
  return relativePath === "" || (!relativePath.startsWith("..") && !relativePath.startsWith(sep));
}

/** 格式化检查结果中的仓库相对位置。 */
function formatLocation(file, line) {
  return `${relative(repoRoot, file)}:${line}`;
}

/** 将字符串转换为正则表达式字面量。 */
function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
