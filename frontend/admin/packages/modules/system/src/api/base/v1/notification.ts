import service from "@liujitcn/kratos-admin-core/request";
import type {
  ArchiveNotificationRequest,
  DeleteNotificationRequest,
  GetNotificationRequest,
  GetNotificationSummaryRequest,
  ListNotificationCategoriesRequest,
  ListNotificationCategoriesResponse,
  MarkAllNotificationReadRequest,
  MarkNotificationReadRequest,
  MarkNotificationUnreadRequest,
  Notification,
  NotificationService,
  NotificationSummary,
  PageNotificationRequest,
  PageNotificationResponse,
  RestoreNotificationRequest
} from "@liujitcn/kratos-admin-system/rpc/base/v1/notification";
import type { Empty } from "@liujitcn/kratos-admin-system/rpc/google/protobuf/empty";
import { normalizeNotificationSummary, normalizePageNotificationResponse } from "../../../utils/notification-response";

const NOTIFICATION_URL = "/v1/base/notification";

/** 当前用户站内信服务实现。 */
export class NotificationServiceImpl implements NotificationService {
  /** 分页查询当前用户收件箱。 */
  async PageNotification(request: PageNotificationRequest): Promise<PageNotificationResponse> {
    const response = await service<PageNotificationRequest, PageNotificationResponse>({
      url: NOTIFICATION_URL,
      method: "get",
      params: request
    });
    return normalizePageNotificationResponse(response);
  }

  /** 查询当前可用的消息分类。 */
  ListNotificationCategories(request: ListNotificationCategoriesRequest): Promise<ListNotificationCategoriesResponse> {
    return service<ListNotificationCategoriesRequest, ListNotificationCategoriesResponse>({
      url: `${NOTIFICATION_URL}/categories`,
      method: "get",
      params: request
    });
  }

  /** 查询当前用户消息详情。 */
  GetNotification(request: GetNotificationRequest): Promise<Notification> {
    return service<GetNotificationRequest, Notification>({ url: `${NOTIFICATION_URL}/${request.id}`, method: "get" });
  }

  /** 查询当前用户未读汇总。 */
  async GetNotificationSummary(request: GetNotificationSummaryRequest): Promise<NotificationSummary> {
    const response = await service<GetNotificationSummaryRequest, NotificationSummary>({
      url: `${NOTIFICATION_URL}/summary`,
      method: "get",
      params: request
    });
    return normalizeNotificationSummary(response);
  }

  /** 标记消息为已读。 */
  MarkNotificationRead(request: MarkNotificationReadRequest): Promise<Empty> {
    return service<MarkNotificationReadRequest, Empty>({ url: `${NOTIFICATION_URL}/read`, method: "put", data: request });
  }

  /** 标记消息为未读。 */
  MarkNotificationUnread(request: MarkNotificationUnreadRequest): Promise<Empty> {
    return service<MarkNotificationUnreadRequest, Empty>({ url: `${NOTIFICATION_URL}/unread`, method: "put", data: request });
  }

  /** 标记全部消息为已读。 */
  MarkAllNotificationRead(request: MarkAllNotificationReadRequest): Promise<Empty> {
    return service<MarkAllNotificationReadRequest, Empty>({ url: `${NOTIFICATION_URL}/read/all`, method: "put", data: request });
  }

  /** 归档消息。 */
  ArchiveNotification(request: ArchiveNotificationRequest): Promise<Empty> {
    return service<ArchiveNotificationRequest, Empty>({
      url: `${NOTIFICATION_URL}/${request.id}/archive`,
      method: "put",
      data: request
    });
  }

  /** 恢复已归档消息。 */
  RestoreNotification(request: RestoreNotificationRequest): Promise<Empty> {
    return service<RestoreNotificationRequest, Empty>({
      url: `${NOTIFICATION_URL}/${request.id}/restore`,
      method: "put",
      data: request
    });
  }

  /** 从个人收件箱删除消息。 */
  DeleteNotification(request: DeleteNotificationRequest): Promise<Empty> {
    return service<DeleteNotificationRequest, Empty>({ url: `${NOTIFICATION_URL}/${request.id}`, method: "delete" });
  }
}

export const defNotificationService = new NotificationServiceImpl();
