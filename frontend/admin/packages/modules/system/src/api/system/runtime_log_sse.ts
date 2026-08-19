import { subscribeSseEvent, type SseStop } from "../base/sse";
import type {
  RuntimeLogEntry,
  RuntimeLogGap
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/runtime_log";

const RUNTIME_LOG_SSE_STREAM = "system.admin.runtime-console";
const RUNTIME_LOG_SSE_ENTRY = "runtime.log";
const RUNTIME_LOG_SSE_GAP = "runtime.gap";

/** 实时日志事件处理函数。 */
export type RuntimeLogEntryHandler = (entry: RuntimeLogEntry) => void;

/** 实时日志丢失事件处理函数。 */
export type RuntimeLogGapHandler = (gap: RuntimeLogGap) => void;

/** 订阅指定用户频道的实时日志和丢失提示事件。 */
export function subscribeRuntimeLog(
  channelId: string,
  entryHandler: RuntimeLogEntryHandler,
  gapHandler: RuntimeLogGapHandler
): SseStop {
  const request = { stream: RUNTIME_LOG_SSE_STREAM, channel_id: channelId };
  const stopEntry = subscribeSseEvent(request, RUNTIME_LOG_SSE_ENTRY, parsePayload<RuntimeLogEntry>, entryHandler);
  const stopGap = subscribeSseEvent(request, RUNTIME_LOG_SSE_GAP, parsePayload<RuntimeLogGap>, gapHandler);
  return () => {
    stopGap();
    stopEntry();
  };
}

/** 解析实时日志 SSE JSON 负载。 */
function parsePayload<T>(raw: string): T | null {
  if (!raw) return null;
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}
