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
  try {
    await createBusinessWorkspace({ cwd: root, projectName: "business-admin", moduleNames: ["business"] });
    await assert.rejects(createBusinessWorkspace({ cwd: root, projectName: "business-admin", moduleNames: ["business"] }), /拒绝覆盖/);
  } finally {
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
    assert.match(await readFile(join(target, "apps/admin/vite.config.ts"), "utf8"), /backend\/data\/admin/);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});
