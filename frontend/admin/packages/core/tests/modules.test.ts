import assert from "node:assert/strict";
import test from "node:test";
import type { Component } from "vue";
import {
  ADMIN_STATIC_VIEWS,
  createAdminViewRegistry,
  getAdminTabPath,
  getRegisteredAdminModules,
  registerAdminModule,
  registerAdminModules,
  type AdminModule,
  type AdminViewLoader
} from "../src/modules/index.js";

const component = {} as Component;

test("普通页面必须使用模块前缀且跨模块同名页面互不冲突", () => {
  const orderLoader = createViewLoader();
  const systemLoader = createViewLoader();
  const registry = createAdminViewRegistry([
    { name: "order", views: { "./views/list/index.vue": orderLoader } },
    { name: "system", views: { "./views/list/index.vue": systemLoader } }
  ]);

  assert.equal(registry.resolve("order/list/index"), orderLoader);
  assert.equal(registry.resolve("system/list/index"), systemLoader);
  assert.equal(registry.resolve("list/index"), undefined);
});

test("静态页面以后注册模块的实现为准", () => {
  const defaultLoader = createViewLoader();
  const replacementLoader = createViewLoader();
  const registry = createAdminViewRegistry([
    { name: "core", staticViews: { [ADMIN_STATIC_VIEWS.NOT_FOUND]: defaultLoader } },
    { name: "custom", staticViews: { [ADMIN_STATIC_VIEWS.NOT_FOUND]: replacementLoader } }
  ]);

  assert.equal(registry.resolve(ADMIN_STATIC_VIEWS.NOT_FOUND), replacementLoader);
});

test("视图路径兼容省略和保留 index 的写法", () => {
  const loader = createViewLoader();
  const registry = createAdminViewRegistry([{ name: "order", views: { "./views/list/index.vue": loader } }]);

  assert.equal(registry.resolve("order/list"), loader);
  assert.equal(registry.resolve("/order/list/index.vue"), loader);
});

test("配置查询参数复用时页签使用路由路径", () => {
  registerAdminModule({ name: "test-profile-tabs", routeOptions: { Profile: { reuseTabAcrossQuery: true } } });

  assert.equal(getAdminTabPath({ name: "Profile", path: "/profile", fullPath: "/profile?oauth_bind_success=1" }), "/profile");
  assert.equal(getAdminTabPath({ name: "Other", path: "/other", fullPath: "/other?page=2" }), "/other?page=2");
});

test("重复模块名直接拒绝且不覆盖原模块", () => {
  const moduleName = "test-duplicate-module";
  const originalModule = { name: moduleName };
  registerAdminModule(originalModule);

  assert.throws(() => registerAdminModule({ name: moduleName }), /管理端模块名称重复/);
  assert.equal(
    getRegisteredAdminModules().find(module => module.name === moduleName),
    originalModule
  );
});

test("重复顶部工具名称直接拒绝", () => {
  assertRegistrationConflict(
    [
      { name: "test-header-a", headerTools: [{ name: "duplicate-header", component }] },
      { name: "test-header-b", headerTools: [{ name: "duplicate-header", component }] }
    ],
    /管理端顶部工具名称重复/
  );
});

test("重复用户菜单名称直接拒绝", () => {
  assertRegistrationConflict(
    [
      {
        name: "test-user-menu-a",
        userMenuActions: [{ name: "duplicate-user-menu", labelKey: "test.user_menu.a", menuName: "MenuA" }]
      },
      {
        name: "test-user-menu-b",
        userMenuActions: [{ name: "duplicate-user-menu", labelKey: "test.user_menu.b", menuName: "MenuB" }]
      }
    ],
    /管理端用户菜单名称重复/
  );
});

test("重复路由配置名称直接拒绝", () => {
  assertRegistrationConflict(
    [
      { name: "test-route-a", routeOptions: { DuplicateRoute: { reuseTabAcrossQuery: true } } },
      { name: "test-route-b", routeOptions: { DuplicateRoute: { reuseTabAcrossQuery: false } } }
    ],
    /管理端路由配置名称重复/
  );
});

/** 创建仅用于比较身份的视图加载器。 */
function createViewLoader(): AdminViewLoader {
  return async () => ({ default: component });
}

/** 断言批量注册在发生冲突时不会写入任何候选模块。 */
function assertRegistrationConflict(modules: AdminModule[], expected: RegExp): void {
  assert.throws(() => registerAdminModules(modules), expected);
  const registeredNames = new Set(getRegisteredAdminModules().map(module => module.name));
  modules.forEach(module => assert.equal(registeredNames.has(module.name), false));
}
