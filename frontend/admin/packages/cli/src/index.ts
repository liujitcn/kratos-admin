#!/usr/bin/env node

import { mkdir, readFile, readdir, rm, stat, writeFile } from "node:fs/promises";
import { basename, dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const templateRoot = resolve(packageRoot, "templates/business-workspace");
const gitignoreTemplateName = "_gitignore";
const kebabNamePattern = /^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$/;
const officialModuleOptimizeDependencies: Record<string, string[]> = {
  system: ["swagger-ui-dist/swagger-ui-bundle.js"]
};

/** 创建业务 workspace 的参数。 */
export interface CreateWorkspaceOptions {
  /** 项目目录名称或路径。 */
  projectName: string;
  /** 需要创建的业务模块名称，使用 kebab-case。 */
  moduleNames: string[];
  /** 宿主额外装配的业务模块名称。 */
  additionalModules?: string[];
  /** 生成命令的工作目录。 */
  cwd?: string;
}

/** 创建包含宿主和业务模块包的 pnpm workspace。 */
export async function createBusinessWorkspace(options: CreateWorkspaceOptions): Promise<string> {
  const cwd = options.cwd ?? process.cwd();
  const target = resolve(cwd, options.projectName);
  const projectName = basename(target);
  const moduleNames = normalizeModuleNames(options.moduleNames);
  const primaryModuleName = moduleNames[0];
  const additionalModules = normalizeAdditionalModules(options.additionalModules ?? [], moduleNames);

  validateName(projectName, "项目名称");
  if (await pathExists(target)) throw new Error(`目标目录已存在，拒绝覆盖: ${target}`);

  const packageVersion = await readCliPackageVersion();
  const primaryModuleTokens = createModuleTokens(primaryModuleName, packageVersion);
  const moduleManifestEntries = [
    {
      packageName: "@liujitcn/kratos-admin-system",
      moduleIdentifier: "systemAdminModule",
      optimizeDependencies: officialModuleOptimizeDependencies.system
    },
    ...moduleNames.map(name => ({
      packageName: `@${name}/admin-module`,
      moduleIdentifier: `${toCamelCase(name)}AdminModule`,
      optimizeDependencies: []
    })),
    ...additionalModules.map(name => ({
      packageName: `@liujitcn/kratos-admin-${name}`,
      moduleIdentifier: `${toCamelCase(name)}AdminModule`,
      optimizeDependencies: officialModuleOptimizeDependencies[name] ?? []
    }))
  ];
  const moduleManifest = moduleManifestEntries
    .map(entry => {
      const lines = [
        "  {",
        `    packageName: "${entry.packageName}",`,
        `    load: async () => (await import("${entry.packageName}")).${entry.moduleIdentifier}`
      ];
      if (entry.optimizeDependencies.length > 0) {
        lines[2] += ",";
        lines.push(`    optimizeDependencies: ${JSON.stringify(entry.optimizeDependencies)}`);
      }
      lines.push("  }");
      return lines.join("\n");
    })
    .join(",\n");
  const modulePackages = moduleNames.map(name => `@${name}/admin-module`);
  const appDependencies = {
    "@liujitcn/kratos-admin-core": `^${packageVersion}`,
    "@liujitcn/kratos-admin-system": `^${packageVersion}`,
    ...Object.fromEntries(modulePackages.map(packageName => [packageName, "workspace:*"])),
    ...Object.fromEntries(additionalModules.map(name => [`@liujitcn/kratos-admin-${name}`, `^${packageVersion}`]))
  };
  const modulePaths = Object.fromEntries(
    moduleNames.flatMap(name => {
      const packageName = `@${name}/admin-module`;
      const moduleRoot = `packages/modules/${name}`;
      return [
        [packageName, [`${moduleRoot}/src/index.ts`]],
        [`${packageName}/api/*`, [`${moduleRoot}/src/api/*`]],
        [`${packageName}/package.json`, [`${moduleRoot}/package.json`]],
        [`${packageName}/rpc/*`, [`${moduleRoot}/src/rpc/*`]]
      ];
    })
  );
  const tokens: Record<string, string> = {
    ...primaryModuleTokens,
    __PROJECT_NAME__: projectName,
    __APP_PACKAGE__: `@${primaryModuleName}/admin-app`,
    __APP_DEPENDENCIES__: formatJsonValue(appDependencies, "  "),
    __MODULE_FILTERS__: modulePackages.map(packageName => `--filter=${packageName}`).join(" "),
    __MODULE_MANIFEST__: moduleManifest,
    __MODULE_NAMES__: moduleNames.join("、"),
    __MODULE_PACKAGES__: modulePackages.map(packageName => `\`${packageName}\``).join("、"),
    __MODULE_PATHS__: formatJsonValue(modulePaths, "    "),
    __MODULE_TREE__: createModuleTree(moduleNames),
    __MODULE_TABLE_ROWS__: createModuleTableRows(moduleNames)
  };

  await mkdir(target, { recursive: false });
  try {
    await renderDirectory(templateRoot, target, tokens);
    const moduleTemplateRoot = resolve(templateRoot, "packages/modules/__MODULE_NAME__");
    for (const moduleName of moduleNames.slice(1)) {
      await renderDirectory(moduleTemplateRoot, resolve(target, "packages/modules", moduleName), {
        ...createModuleTokens(moduleName, packageVersion),
        __PROJECT_NAME__: projectName
      });
    }
  } catch (error) {
    await rm(target, { recursive: true, force: true });
    throw error;
  }
  return target;
}

/** 解析命令行并执行对应命令。 */
export async function runCli(args = process.argv.slice(2)): Promise<void> {
  if (args.length === 0 || args.includes("--help") || args.includes("-h")) {
    printHelp();
    return;
  }
  if (args[0] !== "create") throw new Error(`不支持的命令: ${args[0]}`);

  const projectName = args[1];
  const moduleNames = [...readOptions(args, "--module"), ...readOptions(args, "--modules")];
  const withModules = readOptions(args, "--with");
  if (!projectName || moduleNames.length === 0) {
    throw new Error("用法: kratos-admin create <project> --module <module[,module...]>");
  }

  const target = await createBusinessWorkspace({ projectName, moduleNames, additionalModules: withModules });
  process.stdout.write(`已创建业务 workspace: ${target}\n`);
}

/** 渲染模板目录中的路径和文本占位符。 */
async function renderDirectory(source: string, target: string, tokens: Record<string, string>): Promise<void> {
  await mkdir(target, { recursive: true });
  const entries = await readdir(source, { withFileTypes: true });
  for (const entry of entries) {
    const renderedName = entry.name === gitignoreTemplateName ? ".gitignore" : replaceTokens(entry.name, tokens);
    const sourcePath = join(source, entry.name);
    const targetPath = join(target, renderedName);
    if (entry.isDirectory()) {
      await mkdir(targetPath, { recursive: true });
      await renderDirectory(sourcePath, targetPath, tokens);
      continue;
    }
    const content = await readFile(sourcePath, "utf8");
    await writeFile(targetPath, replaceTokens(content, tokens));
  }
}

/** 读取当前 CLI 版本，作为生成项目默认的公开包版本。 */
async function readCliPackageVersion(): Promise<string> {
  const packageJson = JSON.parse(await readFile(resolve(packageRoot, "package.json"), "utf8")) as { version?: unknown };
  if (typeof packageJson.version !== "string" || !packageJson.version) throw new Error("CLI package.json 缺少有效版本");
  return packageJson.version;
}

/** 读取可重复且支持逗号分隔的命令行选项值。 */
function readOptions(args: string[], option: string): string[] {
  return args.flatMap((argument, index) => {
    if (argument !== option) return [];
    const value = args[index + 1];
    if (!value || value.startsWith("--")) throw new Error(`选项 ${option} 缺少值`);
    return value
      .split(",")
      .map(item => item.trim())
      .filter(Boolean);
  });
}

/** 校验项目与模块名称。 */
function validateName(value: string, label: string): void {
  if (!kebabNamePattern.test(value)) throw new Error(`${label}必须使用 kebab-case: ${value}`);
}

/** 规范化自有模块列表并校验保留名称。 */
function normalizeModuleNames(moduleNames: string[]): [string, ...string[]] {
  const normalized = [...new Set(moduleNames.map(name => name.trim()).filter(Boolean))];
  const primaryModuleName = normalized[0];
  if (!primaryModuleName) throw new Error("至少需要一个业务模块名称");
  normalized.forEach(name => validateName(name, "模块名称"));
  const reservedName = normalized.find(name => name === "system" || name === "kratos-admin");
  if (reservedName) throw new Error(`自有模块名称不能使用保留名称: ${reservedName}`);
  return [primaryModuleName, ...normalized.slice(1)];
}

/** 规范化额外模块列表并去重。 */
function normalizeAdditionalModules(moduleNames: string[], currentModules: string[]): string[] {
  const normalized = [...new Set(moduleNames.map(name => name.trim()).filter(Boolean))];
  normalized.forEach(name => validateName(name, "额外模块名称"));
  return normalized.filter(name => name !== "system" && !currentModules.includes(name));
}

/** 创建单个自有模块模板使用的占位符。 */
function createModuleTokens(moduleName: string, packageVersion: string): Record<string, string> {
  return {
    __MODULE_NAME__: moduleName,
    __MODULE_PASCAL__: toPascalCase(moduleName),
    __MODULE_PACKAGE__: `@${moduleName}/admin-module`,
    __MODULE_IDENTIFIER__: `${toCamelCase(moduleName)}AdminModule`,
    __CORE_VERSION__: `^${packageVersion}`
  };
}

/** 格式化嵌入模板的多行 JSON 值。 */
function formatJsonValue(value: unknown, indentation: string): string {
  const lines = JSON.stringify(value, null, 2).split("\n");
  return [lines[0], ...lines.slice(1).map(line => `${indentation}${line}`)].join("\n");
}

/** 创建 README 中的业务模块目录树。 */
function createModuleTree(moduleNames: string[]): string {
  return moduleNames.map(name => `├── packages/modules/${name}`).join("\n");
}

/** 创建 README 中的业务模块目录说明。 */
function createModuleTableRows(moduleNames: string[]): string {
  return moduleNames
    .map(name => `| \`packages/modules/${name}/\` | 可独立发布的 \`@${name}/admin-module\` 业务 module。 |`)
    .join("\n");
}

/** 将 kebab-case 转换为 camelCase。 */
function toCamelCase(value: string): string {
  return value.replace(/-([a-z0-9])/g, (_, character: string) => character.toUpperCase());
}

/** 将 kebab-case 转换为 PascalCase。 */
function toPascalCase(value: string): string {
  const camelCase = toCamelCase(value);
  return `${camelCase.charAt(0).toUpperCase()}${camelCase.slice(1)}`;
}

/** 替换模板文本中的全部占位符。 */
function replaceTokens(value: string, tokens: Record<string, string>): string {
  return Object.entries(tokens).reduce((content, [token, replacement]) => content.replaceAll(token, replacement), value);
}

/** 判断路径是否存在。 */
async function pathExists(path: string): Promise<boolean> {
  try {
    await stat(path);
    return true;
  } catch {
    return false;
  }
}

/** 输出 CLI 使用说明。 */
function printHelp(): void {
  process.stdout.write(
    [
      "kratos-admin create <project> --module <module[,module...]> [--module <module>] [--with other]",
      "",
      "示例:",
      "  kratos-admin create shop-admin --module shop",
      "  kratos-admin create shop-admin --module shop,order",
      "  kratos-admin create shop-admin --module shop --module order",
      "  kratos-admin create shop-admin --module shop,order --with other",
      ""
    ].join("\n")
  );
}

const executedPath = process.argv[1] ? pathToFileURL(resolve(process.argv[1])).href : "";
if (import.meta.url === executedPath) {
  runCli().catch(error => {
    process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`);
    process.exitCode = 1;
  });
}
