import type { ReactNode } from 'react'
import { Text, View } from '@tarojs/components'

/** 空态组件属性。 */
export interface KratosEmptyProps {
  /** 图标节点（如 <UniIcon />），不传则不渲染图标位。 */
  icon?: ReactNode
  /** 主文案。 */
  title?: string
  /** 次级提示文案。 */
  description?: string
  /** 操作区内容（如引导按钮）。 */
  children?: ReactNode
}

/** 标准空态组件：图标 + 主文案 + 提示文案 + 可选操作区。 */
export default function KratosEmpty({ icon, title, description, children }: KratosEmptyProps) {
  return (
    <View className='k-empty'>
      {icon && <View className='k-empty__icon'>{icon}</View>}
      {title && <Text className='k-empty__title'>{title}</Text>}
      {description && <Text className='k-empty__desc'>{description}</Text>}
      {children}
    </View>
  )
}
