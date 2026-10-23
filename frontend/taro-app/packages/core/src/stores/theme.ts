import Taro from '@tarojs/taro'
import { create } from 'zustand'

/** 主题模式：暗黑、明亮、跟随系统。 */
export type ThemeMode = 'dark' | 'light' | 'auto'
/** 实际生效的主题。 */
export type ResolvedTheme = 'dark' | 'light'

/** 当前主题可供脚本使用的语义颜色。 */
export interface ThemeColors {
  background: string
  surface: string
  surface2: string
  surfaceRgb: string
  primary: string
  primaryDeep: string
  onPrimary: string
  info: string
  warning: string
  danger: string
  violet: string
  text: string
  textMuted: string
  textDim: string
  aiPrimary: string
  aiDanger: string
  aiWarning: string
  aiViolet: string
  aiTextMuted: string
  aiTextDim: string
  navigationBarText: string
}

/** 引用模块可按主题覆盖的颜色字段。未提供的字段继续使用 Core 默认值。 */
export type ThemeColorOverrides = {
  light?: Partial<ThemeColors>
  dark?: Partial<ThemeColors>
}

/** 明暗主题的统一语义颜色表；页面脚本禁止直接维护主题颜色三元表达式。 */
export const themeColors: Record<ResolvedTheme, ThemeColors> = {
  light: {
    background: '#f5f5f5',
    surface: '#ffffff',
    surface2: '#f7f7f7',
    surfaceRgb: '255, 255, 255',
    primary: '#e1251b',
    primaryDeep: '#c81623',
    onPrimary: '#ffffff',
    info: '#1677ff',
    warning: '#fa8c16',
    danger: '#e1251b',
    violet: '#722ed1',
    text: '#1f1f1f',
    textMuted: '#666666',
    textDim: '#8c8c8c',
    aiPrimary: '#039e8c',
    aiDanger: '#e0324f',
    aiWarning: '#d9880f',
    aiViolet: '#5f4fe0',
    aiTextMuted: '#5a7083',
    aiTextDim: '#9aabbb',
    navigationBarText: '#000000',
  },
  dark: {
    background: '#141414',
    surface: '#1f1f1f',
    surface2: '#262626',
    surfaceRgb: '31, 31, 31',
    primary: '#ff4d4f',
    primaryDeep: '#e1251b',
    onPrimary: '#ffffff',
    info: '#4096ff',
    warning: '#faad14',
    danger: '#ff4d4f',
    violet: '#b37feb',
    text: '#f5f5f5',
    textMuted: '#bfbfbf',
    textDim: '#8c8c8c',
    aiPrimary: '#28f0d4',
    aiDanger: '#ff4d6d',
    aiWarning: '#ffb648',
    aiViolet: '#9b8cff',
    aiTextMuted: '#8aa0b4',
    aiTextDim: '#8aa0b4',
    navigationBarText: '#ffffff',
  },
}

const themeColorOverrides: Record<ResolvedTheme, Partial<ThemeColors>> = {
  light: {},
  dark: {},
}

/** 获取指定主题的语义颜色。 */
export const getThemeColors = (theme: ResolvedTheme): ThemeColors => ({
  ...themeColors[theme],
  ...themeColorOverrides[theme],
})

const THEME_MODE_STORAGE_KEY = 'kratos_theme_mode'
const DEFAULT_THEME_MODE: ThemeMode = 'light'

/** 读取系统主题；系统未上报时回退明亮。 */
const readSystemTheme = (): ResolvedTheme => {
  if (process.env.TARO_ENV === 'h5' && typeof window !== 'undefined') {
    return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
  }
  try {
    return Taro.getSystemInfoSync().theme === 'dark' ? 'dark' : 'light'
  } catch {
    return 'light'
  }
}

/** 读取本地持久化的主题模式。 */
const readStoredMode = (): ThemeMode => {
  try {
    const stored = Taro.getStorageSync(THEME_MODE_STORAGE_KEY)
    if (stored === 'dark' || stored === 'light' || stored === 'auto') return stored
  } catch {
    // 存储读取失败时使用默认主题。
  }
  return DEFAULT_THEME_MODE
}

/** 主题实际生效时给导航栏与页面底色染色，覆盖全局配置的静态浅色。 */
const applyChrome = (theme: ResolvedTheme) => {
  try {
    const colors = getThemeColors(theme)
    void Taro.setNavigationBarColor({
      frontColor: colors.navigationBarText,
      backgroundColor: colors.surface,
      fail: () => undefined,
    })
    void Taro.setBackgroundColor({ backgroundColor: colors.background, fail: () => undefined })
  } catch {
    // 部分平台/版本未暴露导航栏染色 API，跳过不影响主题类。
  }
  // H5 弹层（showModal/showToast）挂在 body 下，取不到页面根类，把主题类镜像到 html 上。
  if (process.env.TARO_ENV === 'h5' && typeof document !== 'undefined') {
    document.documentElement.classList.toggle('theme-dark', theme === 'dark')
    document.documentElement.classList.toggle('theme-light', theme === 'light')
  }
}

/** 订阅系统深浅色变化；H5 用 matchMedia，小程序用 onThemeChange。 */
const watchSystemTheme = (onChange: (theme: ResolvedTheme) => void) => {
  if (process.env.TARO_ENV === 'h5') {
    if (typeof window !== 'undefined' && typeof window.matchMedia === 'function') {
      const media = window.matchMedia('(prefers-color-scheme: light)')
      const listener = (event: MediaQueryListEvent) => onChange(event.matches ? 'light' : 'dark')
      if (typeof media.addEventListener === 'function') {
        media.addEventListener('change', listener)
      } else if (typeof media.addListener === 'function') {
        media.addListener(listener)
      }
    }
    return
  }
  try {
    Taro.onThemeChange?.((result) => onChange(result.theme === 'light' ? 'light' : 'dark'))
  } catch {
    // 平台不支持系统主题回调时保持启动时读取的值。
  }
}

/** 全局主题状态及操作。 */
export interface ThemeStoreState {
  mode: ThemeMode
  theme: ResolvedTheme
  colors: ThemeColors
  setMode: (mode: ThemeMode) => void
}

/** 全局主题 Zustand Store：模式持久化存储，auto 跟随系统深浅色。 */
export const useThemeStore = create<ThemeStoreState>((set) => ({
  mode: readStoredMode(),
  theme: 'dark',
  colors: getThemeColors('dark'),
  setMode(mode) {
    const theme = mode === 'auto' ? readSystemTheme() : mode
    try {
      Taro.setStorageSync(THEME_MODE_STORAGE_KEY, mode)
    } catch {
      // 存储写入失败不影响当前会话主题。
    }
    set({ mode, theme, colors: getThemeColors(theme) })
  },
}))

/** 注册引用模块的主题颜色覆盖，并立即刷新当前主题颜色。 */
export const registerThemeColors = (overrides: ThemeColorOverrides): void => {
  if (overrides.light) Object.assign(themeColorOverrides.light, overrides.light)
  if (overrides.dark) Object.assign(themeColorOverrides.dark, overrides.dark)
  useThemeStore.setState({ colors: getThemeColors(useThemeStore.getState().theme) })
}

// 初始化：解析启动主题、染色导航栏并订阅系统主题变化。
const initialMode = useThemeStore.getState().mode
const initialTheme = initialMode === 'auto' ? readSystemTheme() : initialMode
useThemeStore.setState({ theme: initialTheme, colors: getThemeColors(initialTheme) })
watchSystemTheme((theme) => {
  if (useThemeStore.getState().mode === 'auto') {
    useThemeStore.setState({ theme, colors: getThemeColors(theme) })
  }
})
useThemeStore.subscribe((state, prev) => {
  if (state.theme !== prev.theme) applyChrome(state.theme)
})
applyChrome(initialTheme)
