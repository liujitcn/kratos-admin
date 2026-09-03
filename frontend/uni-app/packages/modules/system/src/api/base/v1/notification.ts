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
  PageNotification(request: PageNotificationRequest): Promise<PageNotificationResponse> {
    const data = Object.fromEntries(
      Object.entries(request).filter(([, value]) => value !== undefined),
    )
    return http({ url: NOTIFICATION_URL, method: 'GET', authMode: 'required', data })
  }
  /** 查询当前可用的消息分类。 */
  ListNotificationCategories(
    request: ListNotificationCategoriesRequest,
  ): Promise<ListNotificationCategoriesResponse> {
    return http({
      url: `${NOTIFICATION_URL}/categories`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
  }
  /** 查询消息详情。 */
  GetNotification(request: GetNotificationRequest): Promise<Notification> {
    return http({ url: `${NOTIFICATION_URL}/${request.id}`, method: 'GET', authMode: 'required' })
  }
  /** 查询未读汇总。 */
  GetNotificationSummary(request: GetNotificationSummaryRequest): Promise<NotificationSummary> {
    return http({
      url: `${NOTIFICATION_URL}/summary`,
      method: 'GET',
      authMode: 'required',
      data: request,
    })
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
