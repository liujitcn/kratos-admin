import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { test } from "node:test";

/** 以模块目录为基准读取仓库内源码文件，供测试做静态文本断言。 */
function readSource(path: string) {
  return readFile(join(process.cwd(), path), "utf8");
}

test("强制改密弹窗挂载到 body，遮罩覆盖完整布局", async () => {
  const source = await readSource("src/components/account/ForcedPasswordDialog.vue");

  assert.match(source, /<ProDialog[\s\S]*append-to-body/);
  assert.match(source, /:show-footer="false"/);
});

test("密码过期时先进入个人中心密码页，避免加载受保护业务页", async () => {
  const source = await readSource("../../core/src/views/login/components/LoginForm.vue");

  assert.match(source, /await navigateTo\(router, "\/profile", \{ tab: "password" \}\)/);
  assert.match(source, /await navigateTo\(router, getLoginRedirectPath\(\)\)/);
});

test("仅普通登录加载当前租户自定义翻译", async () => {
  const [loginSource, configSource] = await Promise.all([
    readSource("../../core/src/views/login/components/LoginForm.vue"),
    readSource("../../core/src/stores/modules/config.ts")
  ]);
  const loginResponseBlock = loginSource.match(/const handleLoginResponse = async[\s\S]*?\n\};/)?.[0];
  const displayConfigBlock = configSource.match(/async loadDisplayConfig\(\)[\s\S]*?\n    \},/)?.[0];

  assert.ok(loginResponseBlock, "缺少登录响应处理方法");
  assert.ok(displayConfigBlock, "缺少公开站点配置加载方法");
  assert.ok(loginResponseBlock.indexOf("updateTokenAuth") < loginResponseBlock.indexOf("loadI18nCustom"));
  assert.ok(loginResponseBlock.indexOf("loadI18nCustom") < loginResponseBlock.indexOf("finishLogin"));
  assert.match(loginResponseBlock, /if \(!mustChangePassword\) \{[\s\S]*?await configStore\.loadI18nCustom\(\);/);
  assert.doesNotMatch(displayConfigBlock, /applyCustomLocaleMessages/);
});
