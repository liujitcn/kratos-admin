import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { stripTypeScriptTypes } from "node:module";
import { resolve } from "node:path";
import test from "node:test";
import type { AiStreamEvent } from "../src/views/ai/chat/types.js";

test("终态立即释放发送状态，旧流结束不能覆盖审批后的新流", async () => {
  const source = await readFile(resolve(import.meta.dirname, "../src/components/ai/useAiChatStream.ts"), "utf8");
  const body = stripTypeScriptTypes(source.replace(/^import[\s\S]*?;\n/gm, "").replace("export function", "function"));
  const handlers: Array<(event: AiStreamEvent) => void> = [];
  const release: Array<() => void> = [];
  const sending: boolean[] = [];
  const create = new Function(
    "onBeforeUnmount", "defAiMessageService", "readAiEventStream", "normalizeMessageList", "completeStreamingActivities",
    "replacePendingMessages", `${body}; return useAiChatStream;`
  )(
    () => undefined,
    { StreamAiMessage: async () => ({ body: {} }) },
    async (_body: unknown, handler: (event: AiStreamEvent) => void) => {
      handlers.push(handler);
      await new Promise<void>(resolve => release.push(resolve));
    },
    () => [],
    (messages: unknown) => messages,
    (messages: unknown) => messages
  );
  const chat = create({ messages: { value: { "1": [] } }, setSessionSending: (_id: string, value: boolean) => sending.push(value) });
  const payload = { text: "test", attachments: [] };
  const request = { providerID: 1, modelName: "test", knowledgeBaseIDs: [] };
  const first = chat.run("1", payload, request);
  await Promise.resolve();
  handlers[0]({ event: "approval_required", payload: { session_id: "1", message_id: "2", messages: [] } });
  assert.deepEqual(sending, [], "审批卡片出现时保留原流任务，等待同一流的后续事件");
  const next = chat.run("1", payload, request);
  await Promise.resolve();
  release[0]();
  await first;
  assert.deepEqual(sending, [], "旧流 finally 不能结束新请求");
  handlers[0]({ event: "finish", payload: { session_id: "1", message_id: "2", messages: [] } });
  assert.deepEqual(sending, [], "旧流的晚到事件必须忽略");
  handlers[1]({ event: "finish", payload: { session_id: "1", message_id: "3", messages: [] } });
  assert.deepEqual(sending, [false]);
  release[1]();
  await next;
});

test("审批要求后同一条流仍可接收最终消息", async () => {
  const source = await readFile(resolve(import.meta.dirname, "../src/components/ai/useAiChatStream.ts"), "utf8");
  const body = stripTypeScriptTypes(source.replace(/^import[\s\S]*?;\n/gm, "").replace("export function", "function"));
  const handlers: Array<(event: AiStreamEvent) => void> = [];
  const release: Array<() => void> = [];
  const sending: boolean[] = [];
  const create = new Function(
    "onBeforeUnmount", "defAiMessageService", "readAiEventStream", "normalizeMessageList", "completeStreamingActivities",
    "replacePendingMessages", `${body}; return useAiChatStream;`
  )(
    () => undefined,
    { StreamAiMessage: async () => ({ body: {} }) },
    async (_body: unknown, handler: (event: AiStreamEvent) => void) => {
      handlers.push(handler);
      await new Promise<void>(resolve => release.push(resolve));
    },
    () => [],
    (messages: unknown) => messages,
    (messages: unknown) => messages
  );
  const chat = create({ messages: { value: { "1": [] } }, setSessionSending: (_id: string, value: boolean) => sending.push(value) });
  const run = chat.run("1", { text: "test", attachments: [] }, { providerID: 1, modelName: "test", knowledgeBaseIDs: [] });
  await Promise.resolve();
  handlers[0]({ event: "approval_required", payload: { session_id: "1", message_id: "2", messages: [] } });
  assert.deepEqual(sending, [], "审批等待期间保留当前流任务，避免其被 abort");
  handlers[0]({ event: "finish", payload: { session_id: "1", message_id: "2", messages: [] } });
  assert.deepEqual(sending, [false], "同一条流完成后才释放发送状态");
  release[0]();
  await run;
});
