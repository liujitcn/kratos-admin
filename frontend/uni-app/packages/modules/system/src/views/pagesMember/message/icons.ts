/** 消息分类在 uni-app 中可渲染的图标名称。 */
export type MessageCategoryIcon =
  | 'notification'
  | 'notification-filled'
  | 'locked-filled'
  | 'list'
  | 'chatboxes-filled'
  | 'info-filled'
  | 'flag-filled'
  | 'checkbox-filled'
  | 'paperplane-filled'
  | 'settings-filled'
  | 'person-filled'
  | 'calendar-filled'
  | 'chatbubble-filled'
  | 'bars'

const categoryIconMap: Record<string, MessageCategoryIcon> = {
  Bell: 'notification-filled',
  Lock: 'locked-filled',
  List: 'list',
  Message: 'chatboxes-filled',
  InfoFilled: 'info-filled',
  WarningFilled: 'flag-filled',
  CircleCheckFilled: 'checkbox-filled',
  Promotion: 'paperplane-filled',
  Setting: 'settings-filled',
  User: 'person-filled',
  Calendar: 'calendar-filled',
  ChatDotRound: 'chatbubble-filled',
  DataAnalysis: 'bars',
  CollectionTag: 'list',
}

/** 将管理端维护的 Element Plus 图标名转换为 uni-icons 图标名。 */
export function resolveMessageCategoryIcon(icon?: string): MessageCategoryIcon {
  return categoryIconMap[icon ?? ''] ?? 'notification'
}
