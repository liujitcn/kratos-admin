import { Text } from '@tarojs/components'
import './styles/uni-icons.scss'

/** uni-app uni-icons 中输入器使用的图标字形。 */
export type UniIconName = 'plusempty' | 'mic' | 'paperplane' | 'left' | 'bars'

/** Taro 端复用 uni-app 图标字体的图标组件。 */
export interface UniIconProps {
  type: UniIconName
  size?: number | string
  color?: string
}

const iconUnicode: Record<UniIconName, string> = {
  plusempty: '\ue67b',
  mic: '\ue671',
  paperplane: '\ue672',
  left: '\ue6b7',
  bars: '\ue627',
}

/** 渲染与 uni-app uni-icons 相同的图标字形。 */
export default function UniIcon({ type, size = 16, color = '#333' }: UniIconProps) {
  const fontSize = typeof size === 'number' ? `${size}px` : size
  return (
    <Text className='kratos-uni-icon' style={{ fontSize, color }}>
      {iconUnicode[type]}
    </Text>
  )
}
