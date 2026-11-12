import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import test from "node:test";
import {
  canConfigureAssistant,
  hasConnectedPluginConnection,
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

test("已连接的 MCP 插件不能重复连接", () => {
  assert.equal(hasConnectedPluginConnection("1", [{ plugin_id: "1", state: "connected", can_manage: true }]), true);
  assert.equal(hasConnectedPluginConnection("1", [{ plugin_id: "1", state: "connected", can_manage: false }]), false);
  assert.equal(hasConnectedPluginConnection("1", [{ plugin_id: "1", state: "disconnected", can_manage: true }]), false);
  assert.equal(hasConnectedPluginConnection("1", [{ plugin_id: "2", state: "connected", can_manage: true }]), false);
});

test("工具页在插件列表就绪后自动选择并加载首个插件", async () => {
  const source = await readFile(join(process.cwd(), "src/views/ai/chat/workspace/PluginPanel.vue"), "utf8");
  assert.match(source, /watch\(\s*\[tab, managedPlugins\]/);
  assert.match(source, /const selectedPluginID = [\s\S]*plugins\[0\]\.id/);
  assert.match(source, /toolPluginID\.value = selectedPluginID/);
  assert.match(source, /void loadTools\(\)/);
});

test("连接和工具操作列的按钮带图标并保留点击处理", async () => {
  const source = await readFile(join(process.cwd(), "src/views/ai/chat/workspace/PluginPanel.vue"), "utf8");

  const iconImport = source.match(/import \{[^}]+\} from "@element-plus\/icons-vue"/)?.[0] ?? "";
  for (const icon of ["ChatDotRound", "Refresh", "Key", "Share", "SwitchButton", "View"]) assert.match(iconImport, new RegExp(icon));
  const connectionActions = source.slice(source.indexOf("const connectionColumns"), source.indexOf("const toolColumns"));
  assert.match(connectionActions, /label: t\("system\.ai\.workspace\.use_chat"\),\s*icon: ChatDotRound,\s*hidden: \(\{ row \}\) => row\.state !== "connected",[\s\S]*onClick:/);
  assert.match(connectionActions, /label: t\("system\.ai\.workspace\.refresh_tools"\),\s*icon: Refresh,[\s\S]*onClick:/);
  assert.match(connectionActions, /label: t\("system\.ai\.workspace\.authorize"\),\s*icon: Key,[\s\S]*onClick:/);
  assert.match(connectionActions, /label: t\("system\.ai\.workspace\.share"\),\s*icon: Share,[\s\S]*onClick:/);
  assert.match(connectionActions, /label: t\("system\.ai\.workspace\.unshare"\),\s*icon: Share,[\s\S]*onClick:/);
  assert.match(connectionActions, /label: t\("system\.ai\.workspace\.disconnect"\),\s*icon: SwitchButton,[\s\S]*onClick:/);
  const toolActions = source.slice(source.indexOf("const toolColumns"), source.indexOf("const fields"));
  assert.match(toolActions, /label: t\("system\.ai\.workspace\.review"\),\s*icon: View,[\s\S]*onClick:/);
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

test("AI聊天展示流式运行活动并在完成后保留时间线", async () => {
  const [types, stream, message, chatPanel] = await Promise.all([
    readFile(join(process.cwd(), "src/views/ai/chat/types.ts"), "utf8"),
    readFile(join(process.cwd(), "src/views/ai/chat/stream.ts"), "utf8"),
    readFile(join(process.cwd(), "src/views/ai/chat/message.ts"), "utf8"),
    readFile(join(process.cwd(), "src/views/ai/chat/components/ChatPanel.vue"), "utf8")
  ]);
  assert.match(types, /AiActivityItem/);
  assert.match(stream, /"activity"/);
  assert.match(message, /upsertStreamingActivity/);
  assert.match(message, /completeStreamingActivities/);
  assert.match(chatPanel, /agent-message-activity/);
});
