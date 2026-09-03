import type { NotificationSummary, PageNotificationResponse } from "../rpc/base/v1/notification";

/** 归一化站内信分页响应。 */
export function normalizePageNotificationResponse(response: Partial<PageNotificationResponse>): PageNotificationResponse {
  return {
    ...response,
    notifications: response.notifications ?? [],
    total: response.total ?? 0,
    next_cursor_id: response.next_cursor_id ?? 0,
    has_more: response.has_more ?? false
  };
}

/** 归一化站内信未读汇总响应。 */
export function normalizeNotificationSummary(response: Partial<NotificationSummary>): NotificationSummary {
  return {
    ...response,
    unread_total: response.unread_total ?? 0,
    latest_delivery_id: response.latest_delivery_id ?? 0,
    category_unread: response.category_unread ?? []
  };
}
