import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { mkdtemp, readFile, rm, stat } from "node:fs/promises";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import test from "node:test";
import { promisify } from "node:util";
import { fileURLToPath } from "node:url";
import { createBusinessWorkspace, runCli } from "./index.js";

const execFileAsync = promisify(execFile);
const packageRoot = join(dirname(fileURLToPath(import.meta.url)), "..");

test("生成包含宿主和业务模块的 pnpm workspace", async () => {
  const root = await mkdtemp(join(tmpdir(), "kratos-admin-cli-"));
  try {
    const target = await createBusinessWorkspace({
      cwd: root,
      projectName: "shop-admin",
      moduleNames: ["shop", "order"],
      additionalModules: ["audit"]
    });
    await stat(join(target, "pnpm-workspace.yaml"));
    await stat(join(target, ".gitignore"));
    await stat(join(target, "README.md"));
    await stat(join(target, "apps/admin/src/main.ts"));
    await stat(join(target, "apps/admin/src/modules.ts"));
    await stat(join(target, "apps/admin/src/module-manifest.ts"));
    await stat(join(target, "apps/admin/vite.config.ts"));
    await stat(join(target, "apps/admin/README.md"));
    await stat(join(target, "packages/modules/shop/src/module.ts"));
    await stat(join(target, "packages/modules/shop/src/rpc/README.md"));
    await stat(join(target, "packages/modules/shop/README.md"));
    await stat(join(target, "packages/modules/order/src/module.ts"));
    await stat(join(target, "packages/modules/order/README.md"));

    const workspaceReadme = await readFile(join(target, "README.md"), "utf8");
    assert.match(workspaceReadme, /# shop-admin/);
    assert.match(workspaceReadme, /packages\/modules\/shop/);
    assert.match(workspaceReadme, /packages\/modules\/order/);
    assert.doesNotMatch(workspaceReadme, /__[A-Z_]+__/);

    const appReadme = await readFile(join(target, "apps/admin/README.md"), "utf8");
    assert.match(appReadme, /# @shop\/admin-app/);
    assert.doesNotMatch(appReadme, /__[A-Z_]+__/);

    const moduleReadme = await readFile(join(target, "packages/modules/shop/README.md"), "utf8");
    assert.match(moduleReadme, /# @shop\/admin-module/);
    assert.doesNotMatch(moduleReadme, /__[A-Z_]+__/);

    const orderModuleReadme = await readFile(join(target, "packages/modules/order/README.md"), "utf8");
    assert.match(orderModuleReadme, /# @order\/admin-module/);
    assert.doesNotMatch(orderModuleReadme, /__[A-Z_]+__/);

    const manifest = await readFile(join(target, "apps/admin/src/module-manifest.ts"), "utf8");
    assert.match(manifest, /adminModuleManifest/);
    assert.match(manifest, /packageName: "@shop\/admin-module"/);
    assert.match(manifest, /import\("@shop\/admin-module"\)\)\.shopAdminModule/);
    assert.match(manifest, /packageName: "@order\/admin-module"/);
    assert.match(manifest, /import\("@order\/admin-module"\)\)\.orderAdminModule/);
    assert.match(manifest, /packageName: "@liujitcn\/kratos-admin-system"/);
    assert.match(manifest, /import\("@liujitcn\/kratos-admin-system"\)\)\.systemAdminModule/);
    assert.match(manifest, /packageName: "@liujitcn\/kratos-admin-audit"/);
    assert.match(manifest, /swagger-ui-dist\/swagger-ui-bundle\.js/);
    assert.ok(manifest.indexOf("@liujitcn/kratos-admin-system") < manifest.indexOf("@shop/admin-module"));

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
    assert.equal(packageJson.dependencies["@shop/admin-module"], "workspace:*");
    assert.equal(packageJson.dependencies["@order/admin-module"], "workspace:*");
    assert.match(packageJson.dependencies["@liujitcn/kratos-admin-core"], /^\^\d+\.\d+\.\d+$/);
    assert.match(packageJson.dependencies["@liujitcn/kratos-admin-system"], /^\^\d+\.\d+\.\d+$/);
    assert.match(packageJson.dependencies["@liujitcn/kratos-admin-audit"], /^\^\d+\.\d+\.\d+$/);
    assert.equal(packageJson.dependencies["@liujitcn/kratos-admin"], undefined);

    const modulePackageJson = JSON.parse(await readFile(join(target, "packages/modules/shop/package.json"), "utf8"));
    assert.match(modulePackageJson.devDependencies["@liujitcn/kratos-admin-core"], /^\^\d+\.\d+\.\d+$/);
    assert.match(modulePackageJson.peerDependencies["@liujitcn/kratos-admin-core"], /^\^\d+\.\d+\.\d+$/);
    assert.equal(modulePackageJson.exports["./rpc/*"].default, "./dist/package/src/rpc/*.ts");
    assert.equal(modulePackageJson.exports["./components/*.vue"], undefined);
    assert.equal(modulePackageJson.exports["./views/*.vue"], undefined);

    const tsconfig = JSON.parse(await readFile(join(target, "tsconfig.json"), "utf8"));
    assert.equal(tsconfig.compilerOptions.paths["@shop/admin-module/*"], undefined);
    assert.deepEqual(tsconfig.compilerOptions.paths["@shop/admin-module/api/*"], ["packages/modules/shop/src/api/*"]);
    assert.deepEqual(tsconfig.compilerOptions.paths["@order/admin-module/api/*"], ["packages/modules/order/src/api/*"]);

    const workspacePackageJson = JSON.parse(await readFile(join(target, "package.json"), "utf8"));
    assert.match(workspacePackageJson.devDependencies.sass, /^\^\d+\.\d+\.\d+$/);
    assert.match(workspacePackageJson.scripts["build:package"], /--filter=@shop\/admin-module/);
    assert.match(workspacePackageJson.scripts["build:package"], /--filter=@order\/admin-module/);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test("发布包包含 gitignore 模板占位文件", async () => {
  const result = await execFileAsync("npm", ["pack", "--dry-run", "--ignore-scripts", "--json"], { cwd: packageRoot });
  const packResult = JSON.parse(result.stdout) as Array<{ files: Array<{ path: string }> }>;
  assert.ok(packResult[0]?.files.some(file => file.path === "templates/business-workspace/_gitignore"));
});

test("拒绝覆盖已存在的目标目录", async () => {
  const root = await mkdtemp(join(tmpdir(), "kratos-admin-cli-"));
  try {
    await createBusinessWorkspace({ cwd: root, projectName: "order-admin", moduleNames: ["order"] });
    await assert.rejects(createBusinessWorkspace({ cwd: root, projectName: "order-admin", moduleNames: ["order"] }), /拒绝覆盖/);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test("命令行支持逗号分隔创建多个业务模块", async () => {
  const root = await mkdtemp(join(tmpdir(), "kratos-admin-cli-"));
  try {
    const target = join(root, "multi-admin");
    await runCli(["create", target, "--module", "shop,order"]);
    await stat(join(target, "packages/modules/shop/src/module.ts"));
    await stat(join(target, "packages/modules/order/src/module.ts"));
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});
