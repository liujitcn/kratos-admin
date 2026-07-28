import { access, cp, mkdir, readFile, readdir, rm, writeFile } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import { dirname, extname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const sourceRoot = resolve(packageRoot, "src");
const typesRoot = resolve(packageRoot, "types");
const declarationRoot = resolve(packageRoot, "dist/declarations");
const packageOutputRoot = resolve(packageRoot, "dist/package");
const outputRoot = resolve(packageOutputRoot, "src");
const packageJson = JSON.parse(await readFile(resolve(packageRoot, "package.json"), "utf8"));
const packagePrefix = `${packageJson.name}/`;
const textExtensions = new Set([".css", ".js", ".jsx", ".less", ".scss", ".ts", ".tsx", ".vue"]);

await rm(packageOutputRoot, { recursive: true, force: true });
await rm(declarationRoot, { recursive: true, force: true });
execFileSync("pnpm", ["exec", "vue-tsc", "-p", "tsconfig.package.json"], {
  cwd: packageRoot,
  stdio: "inherit"
});
await mkdir(outputRoot, { recursive: true });
await cp(sourceRoot, outputRoot, { recursive: true });
await cp(typesRoot, resolve(packageOutputRoot, "types"), { recursive: true });
await cp(declarationRoot, resolve(packageOutputRoot, "declarations"), { recursive: true });
await writeFile(
  resolve(packageOutputRoot, "declarations/index.d.ts"),
  [
    '/// <reference path="../src/vite-env.d.ts" />',
    '/// <reference path="../src/typings/global.d.ts" />',
    '/// <reference path="../src/typings/utils.d.ts" />',
    '/// <reference path="../types/generated/auto-imports.d.ts" />',
    '/// <reference path="../types/generated/components.d.ts" />',
    "",
    'export * from "./src/index";',
    ""
  ].join("\n")
);

const files = await collectFiles(packageOutputRoot);
let transformedFiles = 0;
let transformedReferences = 0;
const packageReferences = new Set();

for (const file of files) {
  if (!textExtensions.has(extname(file))) {
    continue;
  }

  const source = await readFile(file, "utf8");
  const references = source.match(/@\//g)?.length ?? 0;
  if (references === 0) {
    continue;
  }

  for (const match of source.matchAll(/@\/[\w./-]+/g)) {
    packageReferences.add(`${packageJson.name}/${match[0].slice(2)}`);
  }
  await writeFile(file, source.replaceAll("@/", packagePrefix));
  transformedFiles += 1;
  transformedReferences += references;
}

if (transformedReferences === 0) {
  throw new Error("未找到需要转换的 @ 源码别名，请检查管理端发布配置");
}

for (const packageReference of packageReferences) {
  const resolvedReference = fileURLToPath(import.meta.resolve(packageReference));
  await access(resolvedReference);
}

await rm(declarationRoot, { recursive: true, force: true });
console.log(`已生成 ${files.length} 个发布文件，转换 ${transformedFiles} 个文件中的 ${transformedReferences} 处 @ 别名`);

/**
 * 递归收集目录中的文件。
 */
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
