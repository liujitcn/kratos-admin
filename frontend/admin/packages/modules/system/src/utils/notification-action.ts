/** 管理端通知动作的路由位置。 */
export interface NotificationActionLocation {
  /** 目标路由名称。 */
  name: string;
  /** 目标路由查询参数。 */
  query?: Record<string, string>;
}

/** 判断通知动作对应的动态路由是否已注册。 */
export function isNotificationActionRouteAvailable(
  router: { hasRoute: (name: string) => boolean },
  action?: NotificationActionLocation
): boolean {
  return Boolean(action && router.hasRoute(action.name));
}

const viewKeyRoutes: Record<string, string> = {
  MESSAGE_INBOX: "NotificationInbox",
  PROFILE: "Profile",
  AI: "AiChat"
};

/** 将消息动作参数解析为管理端路由，无法识别时回退到原 viewKey 路由。 */
export function resolveNotificationAction(actionTarget?: string, actionParams?: string): NotificationActionLocation | undefined {
  const routeName = actionTarget ? viewKeyRoutes[actionTarget] : undefined;
  if (!routeName) return undefined;
  if (routeName !== "NotificationInbox") return { name: routeName };

  let parsed: unknown;
  try {
    parsed = JSON.parse(actionParams || "{}");
  } catch {
    return { name: routeName };
  }
  if (!isRecord(parsed) || !isRecord(parsed.admin) || parsed.admin.route !== "WorkflowApproval") {
    return { name: routeName };
  }

  const taskId = readActionID(parsed.admin.task_id);
  const instanceId = readActionID(parsed.admin.instance_id);
  if (!taskId && !instanceId) return { name: routeName };

  const query: Record<string, string> = {};
  for (const key of ["source", "task_id", "instance_id", "list_type"]) {
    const value = readActionValue(parsed.admin[key]);
    if (value) query[key] = value;
  }
  return { name: "WorkflowApproval", query };
}

/** 判断 JSON 值是否为对象。 */
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

/** 读取可用于路由的字符串或整数参数。 */
function readActionValue(value: unknown): string | undefined {
  if (typeof value === "string" && value !== "") return value;
  if (typeof value === "number" && Number.isInteger(value)) return String(value);
  return undefined;
}

/** 读取可安全转换为管理端请求编号的十进制字符串。 */
function readActionID(value: unknown): string | undefined {
  const result = readActionValue(value);
  if (!result || !/^\d+$/.test(result)) return undefined;
  const numberValue = Number(result);
  return Number.isSafeInteger(numberValue) && numberValue > 0 ? result : undefined;
}
