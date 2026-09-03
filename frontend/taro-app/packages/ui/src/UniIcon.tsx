import { Text } from '@tarojs/components'
import './styles/uni-icons.scss'

/** uni-app uni-icons 中输入器使用的图标字形。 */
export type UniIconName =
  | 'plusempty'
  | 'mic'
  | 'paperplane'
  | 'paperplane-filled'
  | 'left'
  | 'bars'
  | 'notification'
  | 'notification-filled'
  | 'locked-filled'
  | 'list'
  | 'chatboxes-filled'
  | 'info-filled'
  | 'flag-filled'
  | 'checkbox-filled'
  | 'settings-filled'
  | 'person-filled'
  | 'calendar-filled'
  | 'chatbubble-filled'

/** Taro 端复用 uni-app 图标字体的图标组件。 */
export interface UniIconProps {
  className?: string
  type: UniIconName
  size?: number | string
  color?: string
}

const iconUnicode: Record<UniIconName, string> = {
  plusempty: '\ue67b',
  mic: '\ue671',
  paperplane: '\ue672',
  'paperplane-filled': '\ue675',
  left: '\ue6b7',
  bars: '\ue627',
  notification: '\ue6a6',
  'notification-filled': '\ue6c1',
  'locked-filled': '\ue668',
  list: '\ue644',
  'chatboxes-filled': '\ue692',
  'info-filled': '\ue649',
  'flag-filled': '\ue660',
  'checkbox-filled': '\ue62c',
  'settings-filled': '\ue6ce',
  'person-filled': '\ue69d',
  'calendar-filled': '\ue6c0',
  'chatbubble-filled': '\ue694',
}

/** 渲染与 uni-app uni-icons 相同的图标字形。 */
export default function UniIcon({ className, type, size = 16, color = '#333' }: UniIconProps) {
  const fontSize = typeof size === 'number' ? `${size}px` : size
  return (
    <Text
      className={['kratos-uni-icon', className].filter(Boolean).join(' ')}
      style={{ fontSize, color }}
    >
      {iconUnicode[type]}
    </Text>
  )
}
