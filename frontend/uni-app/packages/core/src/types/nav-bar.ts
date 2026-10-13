/** KratosNavBarAction 导航栏操作按钮，统一渲染尺寸、图标与无障碍文案。 */
export interface KratosNavBarAction {
  /** 按钮名称，同时用作可见文字与无障碍文案。 */
  label: string
  /** 内置 settings、read-all、history 图标名称，或自定义图片地址。 */
  icon?: string
  /** 显示方式；默认有图标时显示图标，否则显示文字。 */
  display?: 'icon' | 'text' | 'both'
  /** 点击按钮后执行的业务操作。 */
  onClick: () => void
}

/** KratosNavBarProps 普通页面与 tabBar 页面共用的顶部导航属性。 */
export interface KratosNavBarProps {
  /** 居中显示的页面标题。 */
  title: string
  /** 普通页面默认显示返回按钮，tabBar 页面默认隐藏返回按钮。 */
  variant?: 'page' | 'tab'
  /** 显式覆盖返回按钮的显隐。 */
  back?: boolean
  /** 可选的页面操作按钮。 */
  action?: KratosNavBarAction
  /** 自定义返回操作；不传时返回上一页，失败后回到首页。 */
  onBack?: () => void
}
