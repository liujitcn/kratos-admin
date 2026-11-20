import assert from "node:assert/strict";
import test from "node:test";
import { createAiEventConsumer, readAiMessageEvents } from "../src/views/ai/chat/stream.js";
import type { AiStreamEvent } from "../src/views/ai/chat/types.js";
import type { ListAiMessageEventResponse } from "../src/rpc/base/v1/ai_message.js";

test("执行事件使用无损游标去重，旧运行不能覆盖新运行", () => {
  const received: AiStreamEvent[] = [];
  const consume = createAiEventConsumer(event => received.push(event));
  const item = (id: string, run = "r1"): AiStreamEvent => ({
    event: "activity",
    payload: { session_id: "1", message_id: "2", event_id: id, run_id: run }
  });
  consume(item("9007199254740993"));
  consume(item("9007199254740993"));
  consume(item("9007199254740992"));
  consume(item("9007199254740994", "r2"));
  consume(item("9007199254740993", "r1"));
  assert.equal(received.length, 2);
});

test("补读在 events 省略时继续轮询，只有权威消息决定终态", async () => {
  const received: AiStreamEvent[] = [];
  const queries: string[] = [];
  const page: ListAiMessageEventResponse = {
    events: [],
    next_cursor: "8",
    has_more: false,
    run_id: "r2",
    messages: [
      {
        id: "2",
        tenant_id: 7,
        input_content: undefined,
        output_content: undefined,
        attachments: [],
        created_at: undefined,
        status: 3,
        token: undefined,
        tools: [],
        first_token_ms: 0,
        duration_ms: 0
      }
    ],
    session: {
      id: "1",
      tenant_id: 7,
      terminal: 2,
      title: "test",
      summary: "",
      plugins: [],
      automation_id: "",
      automation_run_id: "",
      read_only: false,
      updated_at: undefined
    }
  };
  await readAiMessageEvents(
    async cursor => {
      queries.push(cursor);
      return JSON.parse(
        JSON.stringify({
          ...page,
          events: undefined,
          messages: page.messages.map(item => ({ ...item, status: queries.length === 1 ? 1 : 3 }))
        })
      );
    },
    event => received.push(event),
    () => false
  );
  assert.deepEqual(queries, ["", "8"]);
  assert.deepEqual(received.map(item => item.event), ["error"]);
  await readAiMessageEvents(
    async () => {
      queries.push("");
      return page;
    },
    () => assert.fail("取消后不应派发事件"),
    () => true
  );
  assert.deepEqual(queries, ["", "8"]);
});
