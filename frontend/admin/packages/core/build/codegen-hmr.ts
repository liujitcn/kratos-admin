import type { HotPayload, Plugin, WebSocketClient } from "vite";

/** 暂缓生成期间的开发热更新，释放后统一刷新以加载完整产物。 */
export function codegenHmrPlugin(): Plugin {
  return {
    name: "admin-codegen-hmr",
    apply: "serve",
    configureServer(server) {
      const clients = new Set<WebSocketClient>();
      const send = server.ws.send.bind(server.ws);
      let pending = false;
      const release = (client: WebSocketClient) => {
        clients.delete(client);
        if (!clients.size && pending) {
          pending = false;
          send({ type: "full-reload", path: "*" });
        }
      };
      server.ws.send = (payload: HotPayload | string, data?: unknown) => {
        if (clients.size && typeof payload !== "string" && (payload.type === "update" || payload.type === "full-reload")) {
          pending = true;
          return;
        }
        if (typeof payload === "string") send(payload, data);
        else send(payload);
      };
      server.ws.on("admin:codegen-hold", (_data, client) => {
        if (!clients.has(client)) {
          clients.add(client);
          client.socket.once("close", () => release(client));
        }
        client.send("admin:codegen-held", {});
      });
      server.ws.on("admin:codegen-release", (_data, client) => release(client));
    }
  };
}
