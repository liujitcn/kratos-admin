import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { copyFile, cp, mkdir, mkdtemp, readFile, rm, stat } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import test from "node:test";
import { promisify } from "node:util";
import { fileURLToPath, pathToFileURL } from "node:url";
import { createBusinessWorkspace, runCli } from "./index.js";

const execFileAsync = promisify(execFile);
const packageRoot = join(dirname(fileURLToPath(import.meta.url)), "..");

test("生成包含宿主和业务模块的 pnpm workspace", async () => {
  const root = await mkdtemp(join(tmpdir(), "kratos-admin-cli-"));
  try {
    const target = await createBusinessWorkspace({
      cwd: root,
      projectName: "business-admin",
      moduleNames: ["business", "report"],
      additionalModules: ["log"]
    });
    await stat(join(target, "pnpm-workspace.yaml"));
    await stat(join(target, ".gitignore"));
    await stat(join(target, "README.md"));
    await stat(join(target, "apps/admin/src/main.ts"));
    await stat(join(target, "apps/admin/src/modules.ts"));
    await stat(join(target, "apps/admin/src/module-manifest.ts"));
    await stat(join(target, "apps/admin/favicon.svg"));
    await stat(join(target, "apps/admin/vite.config.ts"));
    await stat(join(target, "apps/admin/README.md"));
    const adminEnvironment = await readFile(join(target, "apps/admin/.env.development"), "utf8");
    assert.match(adminEnvironment, /VITE_HTTPS_KEY = \.\.\/\.\.\/certs\/dev-key\.pem/);
    await stat(join(target, "packages/modules/business/src/module.ts"));
    await stat(join(target, "packages/modules/business/src/rpc/README.md"));
    await stat(join(target, "packages/modules/business/README.md"));
    await stat(join(target, "packages/modules/report/src/module.ts"));
    await stat(join(target, "packages/modules/report/README.md"));

    const workspaceReadme = await readFile(join(target, "README.md"), "utf8");
    assert.match(workspaceReadme, /# business-admin/);
    assert.match(workspaceReadme, /packages\/modules\/business/);
    assert.match(workspaceReadme, /packages\/modules\/report/);
    assert.doesNotMatch(workspaceReadme, /__[A-Z_]+__/);

    const appReadme = await readFile(join(target, "apps/admin/README.md"), "utf8");
    assert.match(appReadme, /# @business\/admin-app/);
    assert.doesNotMatch(appReadme, /__[A-Z_]+__/);

    const moduleReadme = await readFile(join(target, "packages/modules/business/README.md"), "utf8");
    assert.match(moduleReadme, /# @business\/admin-module/);
    assert.doesNotMatch(moduleReadme, /__[A-Z_]+__/);

    const orderModuleReadme = await readFile(join(target, "packages/modules/report/README.md"), "utf8");
    assert.match(orderModuleReadme, /# @report\/admin-module/);
    assert.doesNotMatch(orderModuleReadme, /__[A-Z_]+__/);

    const manifest = await readFile(join(target, "apps/admin/src/module-manifest.ts"), "utf8");
    assert.match(manifest, /adminModuleManifest/);
    assert.match(manifest, /packageName: "@business\/admin-module"/);
    assert.match(manifest, /import\("@business\/admin-module"\)\)\.businessAdminModule/);
    assert.match(manifest, /packageName: "@report\/admin-module"/);
    assert.match(manifest, /import\("@report\/admin-module"\)\)\.reportAdminModule/);
    assert.match(manifest, /packageName: "@liujitcn\/kratos-admin-system"/);
    assert.match(manifest, /import\("@liujitcn\/kratos-admin-system"\)\)\.systemAdminModule/);
    assert.match(manifest, /packageName: "@liujitcn\/kratos-admin-log"/);
    assert.match(manifest, /swagger-ui-dist\/swagger-ui-bundle\.js/);
    assert.ok(manifest.indexOf("@liujitcn/kratos-admin-system") < manifest.indexOf("@business/admin-module"));

    const modules = await readFile(join(target, "apps/admin/src/modules.ts"), "utf8");
    assert.match(modules, /const adminModules = await loadAdminModules\(\)/);
    assert.match(modules, /export default adminModules/);

    const main = await readFile(join(target, "apps/admin/src/main.ts"), "utf8");
    assert.match(main, /import adminModules from "\.\/modules"/);

    const viteConfig = await readFile(join(target, "apps/admin/vite.config.ts"), "utf8");
    assert.match(viteConfig, /modulePackages: adminModulePackages/);
    assert.match(viteConfig, /optimizeDependencies: adminModuleOptimizeDependencies/);
    assert.match(viteConfig, /from "\.\/src\/module-manifest"/);
    assert.doesNotMatch(viteConfig, /@liujitcn\/kratos-admin-system/);

    const packageJson = JSON.parse(await readFile(join(target, "apps/admin/package.json"), "utf8"));
    const cliPackageJson = JSON.parse(await readFile(join(packageRoot, "package.json"), "utf8"));
    assert.equal(packageJson.dependencies["@liujitcn/kratos-admin-core"], `^${cliPackageJson.version}`);
    assert.equal(packageJson.dependencies["@liujitcn/kratos-admin-system"], `^${cliPackageJson.version}`);
    assert.equal(packageJson.dependencies["@liujitcn/kratos-admin-log"], `^${cliPackageJson.version}`);
    assert.equal(packageJson.dependencies["@business/admin-module"], "workspace:*");
    assert.equal(packageJson.dependencies["@report/admin-module"], "workspace:*");
    assert.equal(packageJson.dependencies["@liujitcn/kratos-admin"], undefined);

    const modulePackageJson = JSON.parse(await readFile(join(target, "packages/modules/business/package.json"), "utf8"));
    assert.match(modulePackageJson.devDependencies["@liujitcn/kratos-admin-core"], /^\^\d+\.\d+\.\d+$/);
    assert.match(modulePackageJson.peerDependencies["@liujitcn/kratos-admin-core"], /^\^\d+\.\d+\.\d+$/);
    assert.equal(modulePackageJson.exports["./rpc/*"].default, "./dist/package/src/rpc/*.ts");
    assert.equal(modulePackageJson.exports["./components/*.vue"], undefined);
    assert.equal(modulePackageJson.exports["./views/*.vue"], undefined);

    const tsconfig = JSON.parse(await readFile(join(target, "tsconfig.json"), "utf8"));
    assert.equal(tsconfig.compilerOptions.paths["@business/admin-module/*"], undefined);
    assert.deepEqual(tsconfig.compilerOptions.paths["@business/admin-module/api/*"], ["packages/modules/business/src/api/*"]);
    assert.deepEqual(tsconfig.compilerOptions.paths["@report/admin-module/api/*"], ["packages/modules/report/src/api/*"]);

    const workspacePackageJson = JSON.parse(await readFile(join(target, "package.json"), "utf8"));
    assert.match(workspacePackageJson.devDependencies.sass, /^\^\d+\.\d+\.\d+$/);
    assert.match(workspacePackageJson.scripts["build:package"], /--filter=@business\/admin-module/);
    assert.match(workspacePackageJson.scripts["build:package"], /--filter=@report\/admin-module/);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test("发布包包含 gitignore 模板占位文件", async () => {
  const result = await execFileAsync("npm", ["pack", "--dry-run", "--ignore-scripts", "--json"], { cwd: packageRoot });
  const packResult = JSON.parse(result.stdout) as Array<{ files: Array<{ path: string }> }>;
  assert.ok(packResult[0]?.files.some(file => file.path === "templates/business-workspace/_gitignore"));
});

test("发布目录中的 CLI 不依赖仓库兄弟 core 包", async () => {
  const root = await mkdtemp(join(tmpdir(), "kratos-admin-cli-install-"));
  const installedRoot = join(root, "node_modules", "@liujitcn", "kratos-admin-cli");
  try {
    await mkdir(join(installedRoot, "dist"), { recursive: true });
    await copyFile(join(packageRoot, "dist/index.js"), join(installedRoot, "dist/index.js"));
    await copyFile(join(packageRoot, "dist/messages.js"), join(installedRoot, "dist/messages.js"));
    await copyFile(join(packageRoot, "package.json"), join(installedRoot, "package.json"));
    await cp(join(packageRoot, "templates"), join(installedRoot, "templates"), { recursive: true });
    const installedCli = (await import(
      `${pathToFileURL(join(installedRoot, "dist/index.js")).href}?standalone`
    )) as typeof import("./index.js");
    const target = await installedCli.createBusinessWorkspace({
      cwd: root,
      projectName: "standalone-admin",
      moduleNames: ["standalone"]
    });
    const appPackageJson = JSON.parse(await readFile(join(target, "apps/admin/package.json"), "utf8"));
    const cliPackageJson = JSON.parse(await readFile(join(installedRoot, "package.json"), "utf8"));
    assert.equal(appPackageJson.dependencies["@liujitcn/kratos-admin-core"], `^${cliPackageJson.version}`);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test("拒绝覆盖已存在的目标目录", async () => {
  const root = await mkdtemp(join(tmpdir(), "kratos-admin-cli-"));
  const previousLocale = process.env.KRATOS_ADMIN_LOCALE;
  process.env.KRATOS_ADMIN_LOCALE = "zh-CN";
  try {
    await createBusinessWorkspace({ cwd: root, projectName: "business-admin", moduleNames: ["business"] });
    await assert.rejects(createBusinessWorkspace({ cwd: root, projectName: "business-admin", moduleNames: ["business"] }), /拒绝覆盖/);
  } finally {
    if (previousLocale === undefined) delete process.env.KRATOS_ADMIN_LOCALE;
    else process.env.KRATOS_ADMIN_LOCALE = previousLocale;
    await rm(root, { recursive: true, force: true });
  }
});

test("英文 CLI 校验错误使用本地化名称标签", async () => {
  const root = await mkdtemp(join(tmpdir(), "kratos-admin-cli-locale-"));
  const previousLocale = process.env.KRATOS_ADMIN_LOCALE;
  process.env.KRATOS_ADMIN_LOCALE = "en-US";
  try {
    await assert.rejects(
      createBusinessWorkspace({ cwd: root, projectName: "invalid name", moduleNames: ["business"] }),
      /Project name must use kebab-case: invalid name/
    );
    await assert.rejects(
      createBusinessWorkspace({ cwd: root, projectName: "valid-name", moduleNames: ["invalid_name"] }),
      /Module name must use kebab-case: invalid_name/
    );
    await assert.rejects(
      createBusinessWorkspace({ cwd: root, projectName: "valid-name", moduleNames: ["business"], additionalModules: ["invalid_name"] }),
      /Additional module name must use kebab-case: invalid_name/
    );
  } finally {
    if (previousLocale === undefined) delete process.env.KRATOS_ADMIN_LOCALE;
    else process.env.KRATOS_ADMIN_LOCALE = previousLocale;
    await rm(root, { recursive: true, force: true });
  }
});

test("命令行支持逗号分隔创建多个业务模块", async () => {
  const root = await mkdtemp(join(tmpdir(), "kratos-admin-cli-"));
  try {
    const target = join(root, "multi-admin");
    await runCli(["create", target, "--module", "business,report"]);
    await stat(join(target, "packages/modules/business/src/module.ts"));
    await stat(join(target, "packages/modules/report/src/module.ts"));
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test("CLI 直接生成本地 system 并保留内置源码扫描与语言资源", async () => {
  const root = await mkdtemp(join(tmpdir(), "kratos-admin-system-"));
  try {
    const target = await createBusinessWorkspace({
      cwd: root,
      projectName: "admin",
      moduleNames: ["system", "report"],
      kratosProject: true
    });
    const manifest = await readFile(join(target, "apps/admin/src/module-manifest.ts"), "utf8");
    assert.doesNotMatch(manifest, /import\("@liujitcn\/kratos-admin-system"\)/);
    assert.match(manifest, /adminBuildModules[\s\S]*@liujitcn\/kratos-admin-system/);
    const module = await readFile(join(target, "packages/modules/system/src/module.ts"), "utf8");
    assert.match(module, /baseSystemAdminModule.messages/);
    for (const name of ["system", "report"]) {
      const locales = await readFile(join(target, `packages/modules/${name}/src/locales/generated.ts`), "utf8");
      for (const locale of ["zh-CN", "en-US", "zh-TW", "ja-JP"]) assert.ok(locales.includes(locale));
    }
    await execFileAsync(process.execPath, [join(target, "scripts/sync-locales.mjs")]);
    assert.match(await readFile(join(root, "Makefile"), "utf8"), /BUSINESS_MODULES := system report/);
    assert.match(await readFile(join(target, "apps/admin/vite.config.ts"), "utf8"), /backend\/web\/admin/);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test("CLI 按所选语言生成 workspace 文档并替换动态模块说明", async () => {
  const root = await mkdtemp(join(tmpdir(), "kratos-admin-cli-readme-locale-"));
  const previousLocale = process.env.KRATOS_ADMIN_LOCALE;
  try {
    process.env.KRATOS_ADMIN_LOCALE = "en-US";
    const englishTarget = await createBusinessWorkspace({
      cwd: root,
      projectName: "english-admin",
      moduleNames: ["business", "report"]
    });
    const englishWorkspaceReadme = await readFile(join(englishTarget, "README.md"), "utf8");
    assert.match(englishWorkspaceReadme, /custom modules \(business, report\)/);
    assert.match(englishWorkspaceReadme, /independently publishable/i);
    assert.doesNotMatch(englishWorkspaceReadme, /自有 module|可独立发布|MODULE_TABLE_ROWS/);
    assert.match(await readFile(join(englishTarget, "apps/admin/README.md"), "utf8"), /The admin host/);
    assert.match(await readFile(join(englishTarget, "packages/modules/business/README.md"), "utf8"), /business module for the admin app/);
    assert.match(await readFile(join(englishTarget, "packages/modules/business/src/rpc/README.md"), "utf8"), /TypeScript generated/);

    process.env.KRATOS_ADMIN_LOCALE = "zh-CN";
    const chineseTarget = await createBusinessWorkspace({
      cwd: root,
      projectName: "chinese-admin",
      moduleNames: ["business", "report"]
    });
    const chineseWorkspaceReadme = await readFile(join(chineseTarget, "README.md"), "utf8");
    assert.match(chineseWorkspaceReadme, /业务管理端 workspace/);
    assert.match(chineseWorkspaceReadme, /自有 module（business、report）/);
    assert.match(chineseWorkspaceReadme, /可独立发布为/);
    assert.doesNotMatch(chineseWorkspaceReadme, /__MODULE_NAMES__|__MODULE_TABLE_ROWS__/);
  } finally {
    if (previousLocale === undefined) delete process.env.KRATOS_ADMIN_LOCALE;
    else process.env.KRATOS_ADMIN_LOCALE = previousLocale;
    await rm(root, { recursive: true, force: true });
  }
});

test("生成工作区的 package 构建消息按 CLI 语言本地化", async () => {
  const root = await mkdtemp(join(tmpdir(), "kratos-admin-cli-build-message-"));
  const previousLocale = process.env.KRATOS_ADMIN_LOCALE;
  try {
    const target = await createBusinessWorkspace({
      cwd: root,
      projectName: "message-workspace",
      moduleNames: ["business"]
    });
    const buildScript = await readFile(join(target, "scripts/build-package.mjs"), "utf8");
    assert.match(buildScript, /workspaceMessage\("package_built"/);

    const generatedMessages = await import(pathToFileURL(join(target, "scripts/locale-messages.mjs")).href);
    process.env.KRATOS_ADMIN_LOCALE = "en-US";
    assert.equal(generatedMessages.workspaceMessage("package_built", { name: "@business/admin-module" }), "Built package artifacts for @business/admin-module");
    assert.equal(generatedMessages.workspaceMessage("locale_missing_default", { module: "sample" }), "sample is missing the zh-CN locale bundle");
    process.env.KRATOS_ADMIN_LOCALE = "zh-CN";
    assert.equal(generatedMessages.workspaceMessage("package_built", { name: "@business/admin-module" }), "已生成 @business/admin-module 发布文件");
    assert.equal(generatedMessages.workspaceMessage("locale_missing_default", { module: "sample" }), "sample 缺少 zh-CN 语言包");
  } finally {
    if (previousLocale === undefined) delete process.env.KRATOS_ADMIN_LOCALE;
    else process.env.KRATOS_ADMIN_LOCALE = previousLocale;
    await rm(root, { recursive: true, force: true });
  }
});
