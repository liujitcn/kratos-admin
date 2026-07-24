import { EventStreamContentType, fetchEventSource, type EventSourceMessage } from "@microsoft/fetch-event-source";
import type { SubscribeSseRequest } from "@/rpc/base/v1/sse";
import type { CodeGenTask } from "@/rpc/system/admin/v1/code_gen";
import { getRequestAccessToken, handleAuthExpired } from "@/utils/request";

const SSE_URL = "/events";
/** 代码生成 SSE 流标识。 */
const sseStreamCodeGen = "system.admin.codegen";
/** 代码生成进度 SSE 事件标识。 */
const sseEventCodeGenProgress = "codegen.progress";

/** SSE 取消订阅函数。 */
export type SseStop = () => void;

/** 代码生成任务进度处理函数。 */
export type CodeGenProgressHandler = (task: CodeGenTask) => void;

/** 同一条 SSE 流的共享连接记录。 */
interface SharedSseConnection {
  /** 关闭当前 SSE 连接。 */
  close: () => void;
  /** 已注册的事件监听器。 */
  listeners: Map<string, Set<(message: EventSourceMessage) => void>>;
  /** 当前流上已绑定的事件监听器数量。 */
  refCount: number;
}

/** SSE 不可恢复异常。 */
class SseFatalError extends Error {}

/** SSE 可重试异常。 */
class SseRetriableError extends Error {}

/** Base SSE 服务。 */
export class SseServiceImpl {
  private readonly sharedConnections = new Map<string, SharedSseConnection>();

  /** 创建或复用 SSE 订阅连接。 */
  SubscribeSse(request: SubscribeSseRequest): SharedSseConnection | null {
    if (typeof window === "undefined" || typeof AbortController === "undefined") {
      return null;
    }

    const url = this.buildSubscribeURL(request);
    if (!url) {
      return null;
    }

    const connectionKey = this.buildConnectionKey(request);
    const cachedConnection = this.sharedConnections.get(connectionKey);
    if (cachedConnection) {
      return cachedConnection;
    }

    const controller = new AbortController();
    const listeners = new Map<string, Set<(message: EventSourceMessage) => void>>();
    const connection: SharedSseConnection = {
      close: () => controller.abort(),
      listeners,
      refCount: 0
    };
    this.sharedConnections.set(connectionKey, connection);
    void this.openFetchEventSource(connectionKey, url, controller, connection);
    return connection;
  }

  /** 释放 SSE 共享连接。 */
  ReleaseSse(request: SubscribeSseRequest) {
    const connectionKey = this.buildConnectionKey(request);
    const cachedConnection = this.sharedConnections.get(connectionKey);
    if (!cachedConnection) {
      return;
    }
    cachedConnection.refCount -= 1;
    if (cachedConnection.refCount > 0) {
      return;
    }
    cachedConnection.close();
    this.sharedConnections.delete(connectionKey);
  }

  /** 构建 SSE 订阅地址。 */
  private buildSubscribeURL(request: SubscribeSseRequest) {
    const url = new URL(`${SSE_URL}/${request.stream}`, window.location.origin);
    if (request.channel_id) {
      url.searchParams.set("channel_id", request.channel_id);
    }
    return url.toString();
  }

  /** 构建 SSE 共享连接键。 */
  private buildConnectionKey(request: SubscribeSseRequest) {
    return `${request.stream}:${request.channel_id ?? ""}`;
  }

  /** 打开基于 fetch 的 SSE 长连接。 */
  private async openFetchEventSource(
    connectionKey: string,
    url: string,
    controller: AbortController,
    connection: SharedSseConnection
  ) {
    try {
      const accessToken = await getRequestAccessToken();
      if (!accessToken) {
        this.sharedConnections.delete(connectionKey);
        return;
      }

      await fetchEventSource(url, {
        method: "GET",
        signal: controller.signal,
        openWhenHidden: true,
        headers: {
          Accept: EventStreamContentType,
          Authorization: accessToken
        },
        async onopen(response) {
          const contentType = response.headers.get("content-type") ?? "";
          if (response.ok && contentType.startsWith(EventStreamContentType)) {
            return;
          }
          if (response.status === 401 || response.status === 403) {
            throw new SseFatalError("SSE 认证已失效");
          }
          throw new SseRetriableError(`SSE 连接失败: ${response.status}`);
        },
        onmessage: message => {
          const eventName = message.event || "";
          if (!eventName) {
            return;
          }
          const eventListeners = connection.listeners.get(eventName);
          if (!eventListeners || eventListeners.size === 0) {
            return;
          }
          eventListeners.forEach(listener => listener(message));
        },
        onclose: () => {
          throw new SseRetriableError("SSE 连接已关闭");
        },
        onerror: error => {
          if (controller.signal.aborted) {
            return;
          }
          if (error instanceof SseFatalError) {
            throw error;
          }
          return 1000;
        }
      });
    } catch (error) {
      if (controller.signal.aborted) {
        return;
      }
      // SSE 与常规请求复用同一套登录失效处理，避免页面静默断流后用户无感知。
      if (error instanceof SseFatalError) {
        handleAuthExpired();
      }
      this.sharedConnections.delete(connectionKey);
    }
  }
}

export const defSseService = new SseServiceImpl();

/** 订阅指定代码生成任务的实时进度。 */
export function subscribeCodeGenProgress(taskId: string, handler: CodeGenProgressHandler): SseStop {
  return subscribeSseEvent(
    { stream: sseStreamCodeGen, channel_id: taskId },
    sseEventCodeGenProgress,
    raw => parseCodeGenProgress(raw, taskId),
    handler
  );
}

/** 订阅指定 SSE 事件。 */
export function subscribeSseEvent<T>(
  request: SubscribeSseRequest,
  event: string,
  parser: (raw: string) => T | null,
  handler: (payload: T) => void
): SseStop {
  const connection = defSseService.SubscribeSse(request);
  if (!connection) return () => undefined;

  connection.refCount += 1;
  const eventName = event;
  let eventListeners = connection.listeners.get(eventName);
  if (!eventListeners) {
    eventListeners = new Set();
    connection.listeners.set(eventName, eventListeners);
  }

  const listener = (message: EventSourceMessage) => {
    const payload = parser(message.data);
    if (!payload) return;
    handler(payload);
  };
  eventListeners.add(listener);

  return () => {
    const currentListeners = connection.listeners.get(eventName);
    currentListeners?.delete(listener);
    if (currentListeners && currentListeners.size === 0) {
      connection.listeners.delete(eventName);
    }
    defSseService.ReleaseSse(request);
  };
}

/** 解析并校验代码生成任务进度负载。 */
function parseCodeGenProgress(raw: string, taskId: string): CodeGenTask | null {
  if (!raw) {
    return null;
  }
  try {
    const task = JSON.parse(raw) as CodeGenTask;
    return task.task_id === taskId && Array.isArray(task.tables) ? task : null;
  } catch {
    return null;
  }
}
