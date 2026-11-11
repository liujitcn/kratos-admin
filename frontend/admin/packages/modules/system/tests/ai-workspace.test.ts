import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import test from "node:test";
import {
  canConfigureAssistant,
  isAssistantView,
  pluginSelectionKey,
  selectionFromKey,
  readAiPluginOAuthCallback
} from "../src/views/ai/chat/workspace/state.js";

test("助手导航包含文件入口并拒绝未知视图", () => {
  assert.equal(isAssistantView("files"), true);
  assert.equal(isAssistantView("unknown"), false);
});

test("手机管理员也不能配置助手", () => {
  assert.equal(canConfigureAssistant(true, true), false);
  assert.equal(canConfigureAssistant(false, false), false);
  assert.equal(canConfigureAssistant(false, true), true);
});

test("工具页在插件列表就绪后自动选择并加载首个插件", async () => {
  const source = await readFile(join(process.cwd(), "src/views/ai/chat/workspace/PluginPanel.vue"), "utf8");
  assert.match(source, /watch\(\s*\[tab, managedPlugins\]/);
  assert.match(source, /const selectedPluginID = [\s\S]*plugins\[0\]\.id/);
  assert.match(source, /toolPluginID\.value = selectedPluginID/);
  assert.match(source, /void loadTools\(\)/);
});

test("插件选择保留字符串ID与指定连接，不丢失大整数精度", () => {
  const selection = { plugin_id: "9223372036854775806", connection_id: "9223372036854775807" };
  assert.deepEqual(selectionFromKey(pluginSelectionKey(selection)), selection);
  assert.deepEqual(selectionFromKey("123:"), { plugin_id: "123", connection_id: "" });
});

test("专用OAuth回调保留issuer和拒绝原因，不消费普通登录OAuth参数", () => {
  assert.equal(readAiPluginOAuthCallback("?state=login&code=login"), undefined);
  assert.deepEqual(readAiPluginOAuthCallback("?ai_plugin_oauth=1&state=once&code=auth&iss=https%3A%2F%2Fissuer.example"), {
    state: "once",
    code: "auth",
    error: "",
    iss: "https://issuer.example"
  });
  assert.deepEqual(readAiPluginOAuthCallback("?ai_plugin_oauth=1&state=once&error=access_denied"), {
    state: "once",
    code: "",
    error: "access_denied",
    iss: ""
  });
});
