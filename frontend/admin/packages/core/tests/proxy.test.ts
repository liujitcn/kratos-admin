import assert from "node:assert/strict";
import { once } from "node:events";
import { createServer, type ServerResponse } from "node:http";
import test from "node:test";
import { createServer as createViteServer } from "vite";
import { createProxy } from "../build/proxy.js";

// 注入首字前的 300 字节截断，验证开发代理能将断流传到浏览器。
test("开发代理收到上游 SSE 截断时关闭下游响应", async () => {
  const firstEvent = 'id: 1\nevent: activity\ndata: {"session_id":"1","message_id":"2","activity":{"id":"reasoning:0","phase":"reasoning","status":"running"}}\n\n';
  let upstreamReply: ServerResponse | undefined;
  const upstream = createServer((_, reply) => {
    upstreamReply = reply;
    reply.writeHead(200, { "Content-Type": "text/event-stream" });
    reply.write(firstEvent);
  }).listen(0, "127.0.0.1");
  await once(upstream, "listening");
  const address = upstream.address();
  assert.ok(address && typeof address !== "string");
  const vite = await createViteServer({
    configFile: false,
    logLevel: "silent",
    server: { host: "127.0.0.1", port: 0, proxy: createProxy([["/api", `http://127.0.0.1:${address.port}`]]) }
  });
  try {
    await vite.listen();
    const proxyAddress = vite.httpServer?.address();
    assert.ok(proxyAddress && typeof proxyAddress !== "string");
    const response = await fetch(`http://127.0.0.1:${proxyAddress.port}/api/stream`, { method: "POST", signal: AbortSignal.timeout(2000) });
    assert.equal(response.status, 200);
    assert.ok(response.body);
    const reader = response.body.getReader();
    const first = await reader.read();
    assert.equal(new TextDecoder().decode(first.value), firstEvent);
    assert.ok(upstreamReply);
    upstreamReply.write("x".repeat(300 - Buffer.byteLength(firstEvent)));
    upstreamReply.destroy();
    // 浏览器可以读到 EOF 或报告断流，两种结果都会触发缺少终态时的恢复。
    try {
      for (let tail = await reader.read(); !tail.done; tail = await reader.read()) {
        assert.ok(tail.value.byteLength <= 300 - Buffer.byteLength(firstEvent));
      }
    } catch (error) {
      assert.ok(error instanceof TypeError, `上游断开后应立即结束读取，实际为 ${error}`);
    }
  } finally {
    upstream.closeAllConnections();
    upstream.close();
    await vite.close();
  }
});
