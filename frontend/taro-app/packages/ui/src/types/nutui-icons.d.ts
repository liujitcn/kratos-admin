declare module '@nutui/icons-react-taro/dist/es/icons/*.js' {
  import type { ComponentType, CSSProperties, ReactNode } from 'react'

  interface NutIconProps {
    className?: string
    color?: string
    height?: number | string
    name?: string
    onClick?: (event: unknown) => void
    size?: number | string
    style?: CSSProperties
    width?: number | string
    children?: ReactNode
  }

  const Icon: ComponentType<NutIconProps>
  export default Icon
}
