import { subscribeSseEvent, type SseStop } from "../api/base/v1/sse";
import type {
  OpsNodesResponse,
  OpsServicesResponse,
  OpsStorageResponse,
  OpsTrafficResponse
} from "@liujitcn/kratos-admin-system/rpc/system/admin/v1/ops_monitoring";

const OPS_MONITORING_SSE_STREAM = "system.admin.ops-monitoring";
const OPS_MONITORING_SSE_TRAFFIC = "ops.traffic";
const OPS_MONITORING_SSE_SERVICES = "ops.services";
const OPS_MONITORING_SSE_STORAGE = "ops.storage";
const OPS_MONITORING_SSE_NODES = "ops.nodes";

/** 运维监控 SSE 取消订阅函数。 */
export type OpsMonitoringSseStop = SseStop;

/** 订阅运维监控流量实时事件。 */
export function subscribeOpsMonitoringTraffic(handler: (payload: OpsTrafficResponse) => void): SseStop {
  return subscribeOpsMonitoringEvent(OPS_MONITORING_SSE_TRAFFIC, parsePayload<OpsTrafficResponse>, handler);
}

/** 订阅运维监控服务实时事件。 */
export function subscribeOpsMonitoringServices(handler: (payload: OpsServicesResponse) => void): SseStop {
  return subscribeOpsMonitoringEvent(OPS_MONITORING_SSE_SERVICES, parsePayload<OpsServicesResponse>, handler);
}

/** 订阅运维监控存储实时事件。 */
export function subscribeOpsMonitoringStorage(handler: (payload: OpsStorageResponse) => void): SseStop {
  return subscribeOpsMonitoringEvent(OPS_MONITORING_SSE_STORAGE, parsePayload<OpsStorageResponse>, handler);
}

/** 订阅运维监控实例实时事件。 */
export function subscribeOpsMonitoringNodes(handler: (payload: OpsNodesResponse) => void): SseStop {
  return subscribeOpsMonitoringEvent(OPS_MONITORING_SSE_NODES, parsePayload<OpsNodesResponse>, handler);
}

function subscribeOpsMonitoringEvent<T>(event: string, parser: (raw: string) => T | null, handler: (payload: T) => void): SseStop {
  return subscribeSseEvent({ stream: OPS_MONITORING_SSE_STREAM }, event, parser, handler);
}

function parsePayload<T>(raw: string): T | null {
  if (!raw) return null;
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}
