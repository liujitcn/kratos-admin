import { access, cp, mkdir, readFile, readdir, rm, writeFile } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import { dirname, extname, join, relative, resolve, sep } from "node:path";

const packageRoot = resolve(process.cwd(), process.argv[2] ?? ".");
const sourceRoot = resolve(packageRoot, "src");
const typesRoot = resolve(packageRoot, "types");
const declarationRoot = resolve(packageRoot, "dist/declarations");
const packageOutputRoot = resolve(packageRoot, "dist/package");
const outputRoot = resolve(packageOutputRoot, "src");
const declarationSourceRoot = resolve(packageOutputRoot, "declarations/src");
const packageJson = JSON.parse(await readFile(resolve(packageRoot, "package.json"), "utf8"));
const sourceAlias = packageJson.adminBuild?.sourceAlias;
const textExtensions = new Set([".css", ".js", ".jsx", ".less", ".scss", ".ts", ".tsx", ".vue"]);

await rm(packageOutputRoot, { recursive: true, force: true });
await rm(declarationRoot, { recursive: true, force: true });
execFileSync("pnpm", ["exec", "vue-tsc", "-p", "tsconfig.package.json"], {
  cwd: packageRoot,
  stdio: "inherit"
});
await mkdir(outputRoot, { recursive: true });
await cp(sourceRoot, outputRoot, { recursive: true });
await copyIfExists(typesRoot, resolve(packageOutputRoot, "types"));
await cp(declarationRoot, resolve(packageOutputRoot, "declarations"), { recursive: true });
await writeDeclarationEntry();

const files = await collectFiles(packageOutputRoot);
let transformedFiles = 0;
let transformedReferences = 0;

if (sourceAlias) {
  for (const file of files) {
    if (!textExtensions.has(extname(file))) continue;

    const source = await readFile(file, "utf8");
    const references = source.split(sourceAlias).length - 1;
    if (references === 0) continue;

    await writeFile(file, source.replaceAll(sourceAlias, resolveSourceAliasPrefix(file)));
    transformedFiles += 1;
    transformedReferences += references;
  }

  if (transformedReferences === 0) {
    throw new Error(`未找到需要转换的 ${sourceAlias} 源码别名: ${packageJson.name}`);
  }
}

/** 将源码别名转换为发布目录内的相对引用，避免内部实现占用 npm 公共子路径。 */
function resolveSourceAliasPrefix(file) {
  const aliasRoot = file.startsWith(`${outputRoot}${sep}`) ? outputRoot : declarationSourceRoot;
  const relativeRoot = relative(dirname(file), aliasRoot).split(sep).join("/");
  if (!relativeRoot) return "./";
  return `${relativeRoot.startsWith(".") ? relativeRoot : `./${relativeRoot}`}/`;
}

await rm(declarationRoot, { recursive: true, force: true });
console.log(
  `已生成 ${packageJson.name} 的 ${files.length} 个发布文件，转换 ${transformedFiles} 个文件中的 ${transformedReferences} 处源码别名`
);

/** 写入 npm 包声明入口，并为 core 补充全局声明引用。 */
async function writeDeclarationEntry() {
  const lines = [];
  if (await pathExists(typesRoot)) {
    lines.push(
      '/// <reference path="../src/vite-env.d.ts" />',
      '/// <reference path="../src/typings/global.d.ts" />',
      '/// <reference path="../src/typings/utils.d.ts" />',
      '/// <reference path="../types/generated/auto-imports.d.ts" />',
      '/// <reference path="../types/generated/components.d.ts" />',
      ""
    );
  }
  lines.push('export * from "./src/index";', "");
  await writeFile(resolve(packageOutputRoot, "declarations/index.d.ts"), lines.join("\n"));
}

/** 目标存在时递归复制。 */
async function copyIfExists(source, target) {
  if (!(await pathExists(source))) return;
  await cp(source, target, { recursive: true });
}

/** 判断文件或目录是否存在。 */
async function pathExists(path) {
  try {
    await access(path);
    return true;
  } catch {
    return false;
  }
}

/** 递归收集目录中的文件。 */
async function collectFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const nestedFiles = await Promise.all(
    entries.map(entry => {
      const path = join(directory, entry.name);
      return entry.isDirectory() ? collectFiles(path) : [path];
    })
  );

  return nestedFiles.flat();
}
