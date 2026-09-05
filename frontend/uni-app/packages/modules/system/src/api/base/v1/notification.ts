import { http } from '@liujitcn/kratos-uni-app-core/utils/http'
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
  RestoreNotificationRequest,
} from '../../../rpc/base/v1/notification'
import type { Empty } from '../../../rpc/google/protobuf/empty'

const NOTIFICATION_URL = '/v1/base/notification'

/** uni-app 当前用户站内信服务。 */
export class NotificationServiceImpl implements NotificationService {
  /** 分页查询收件箱。 */
  async PageNotification(request: PageNotificationRequest): Promise<PageNotificationResponse> {
    const data = Object.fromEntries(
      Object.entries(request).filter(([, value]) => value !== undefined),
    )
    const response = await http<Partial<PageNotificationResponse>>({
      url: NOTIFICATION_URL,
      method: 'GET',
      authMode: 'required',
      data,
    })
    return {
      ...response,
      notifications: Array.isArray(response.notifications) ? response.notifications : [],
      total: response.total ?? 0,
      next_cursor_id: response.next_cursor_id ?? 0,
      has_more: response.has_more ?? false,
    }
  }
  /** 查询当前可用的消息分类。 */
  async ListNotificationCategories(
    request: ListNotificationCategoriesRequest,
  ): Promise<ListNotificationCategoriesResponse> {
    const response = await http<Partial<ListNotificationCategoriesResponse>>({
      url: `${NOTIFICATION_URL}/categories`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
    return {
      ...response,
      categories: Array.isArray(response.categories) ? response.categories : [],
    }
  }
  /** 查询消息详情。 */
  GetNotification(request: GetNotificationRequest): Promise<Notification> {
    return http({ url: `${NOTIFICATION_URL}/${request.id}`, method: 'GET', authMode: 'required' })
  }
  /** 查询未读汇总。 */
  async GetNotificationSummary(
    request: GetNotificationSummaryRequest,
  ): Promise<NotificationSummary> {
    const response = await http<Partial<NotificationSummary>>({
      url: `${NOTIFICATION_URL}/summary`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
    return {
      ...response,
      unread_total: response.unread_total ?? 0,
      latest_delivery_id: response.latest_delivery_id ?? 0,
      category_unread: Array.isArray(response.category_unread) ? response.category_unread : [],
    }
  }
  /** 标记消息已读。 */
  MarkNotificationRead(request: MarkNotificationReadRequest): Promise<Empty> {
    return http({
      url: `${NOTIFICATION_URL}/read`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }
  /** 标记消息未读。 */
  MarkNotificationUnread(request: MarkNotificationUnreadRequest): Promise<Empty> {
    return http({
      url: `${NOTIFICATION_URL}/unread`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }
  /** 标记全部消息已读。 */
  MarkAllNotificationRead(request: MarkAllNotificationReadRequest): Promise<Empty> {
    return http({
      url: `${NOTIFICATION_URL}/read/all`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }
  /** 归档消息。 */
  ArchiveNotification(request: ArchiveNotificationRequest): Promise<Empty> {
    return http({
      url: `${NOTIFICATION_URL}/${request.id}/archive`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }
  /** 恢复已归档消息。 */
  RestoreNotification(request: RestoreNotificationRequest): Promise<Empty> {
    return http({
      url: `${NOTIFICATION_URL}/${request.id}/restore`,
      method: 'PUT',
      authMode: 'required',
      data: request,
    })
  }
  /** 从个人收件箱删除消息。 */
  DeleteNotification(request: DeleteNotificationRequest): Promise<Empty> {
    return http({
      url: `${NOTIFICATION_URL}/${request.id}`,
      method: 'DELETE',
      authMode: 'required',
    })
  }
}

export const defNotificationService = new NotificationServiceImpl()
