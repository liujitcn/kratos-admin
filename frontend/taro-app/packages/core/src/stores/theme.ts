import Taro from '@tarojs/taro'
import { create } from 'zustand'

/** 主题模式：暗黑、明亮、跟随系统。 */
export type ThemeMode = 'dark' | 'light' | 'auto'
/** 实际生效的主题。 */
export type ResolvedTheme = 'dark' | 'light'

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
    const backgroundColor = theme === 'light' ? '#ffffff' : '#141414'
    void Taro.setNavigationBarColor({
      frontColor: theme === 'light' ? '#000000' : '#ffffff',
      backgroundColor,
      fail: () => undefined,
    })
    void Taro.setBackgroundColor({ backgroundColor, fail: () => undefined })
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
  setMode: (mode: ThemeMode) => void
}

/** 全局主题 Zustand Store：模式持久化存储，auto 跟随系统深浅色。 */
export const useThemeStore = create<ThemeStoreState>((set) => ({
  mode: readStoredMode(),
  theme: 'dark',
  setMode(mode) {
    const theme = mode === 'auto' ? readSystemTheme() : mode
    try {
      Taro.setStorageSync(THEME_MODE_STORAGE_KEY, mode)
    } catch {
      // 存储写入失败不影响当前会话主题。
    }
    set({ mode, theme })
  },
}))

// 初始化：解析启动主题、染色导航栏并订阅系统主题变化。
const initialMode = useThemeStore.getState().mode
const initialTheme = initialMode === 'auto' ? readSystemTheme() : initialMode
useThemeStore.setState({ theme: initialTheme })
watchSystemTheme((theme) => {
  if (useThemeStore.getState().mode === 'auto') useThemeStore.setState({ theme })
})
useThemeStore.subscribe((state, prev) => {
  if (state.theme !== prev.theme) applyChrome(state.theme)
})
applyChrome(initialTheme)
