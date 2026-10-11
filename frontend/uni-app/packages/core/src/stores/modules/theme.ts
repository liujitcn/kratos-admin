import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'

/** 主题模式：暗黑、明亮、跟随系统。 */
export type ThemeMode = 'dark' | 'light' | 'auto'
/** 实际生效的主题。 */
export type ResolvedTheme = 'dark' | 'light'

const THEME_MODE_STORAGE_KEY = 'kratos_theme_mode'
const DEFAULT_THEME_MODE: ThemeMode = 'light'

/** 判断当前运行环境是否为 H5。 */
const isH5 = () => {
  // #ifdef H5
  return true
  // #endif
  // #ifndef H5
  return false
  // #endif
}

/** 读取系统主题；系统未上报时回退明亮。 */
const readSystemTheme = (): ResolvedTheme => {
  if (isH5() && typeof window.matchMedia === 'function') {
    return window.matchMedia('(prefers-color-scheme: light)').matches ? 'light' : 'dark'
  }
  try {
    return uni.getSystemInfoSync().theme === 'dark' ? 'dark' : 'light'
  } catch {
    return 'light'
  }
}

/** 读取本地持久化的主题模式。 */
const readStoredMode = (): ThemeMode => {
  try {
    const stored = uni.getStorageSync(THEME_MODE_STORAGE_KEY)
    if (stored === 'dark' || stored === 'light' || stored === 'auto') return stored
  } catch {
    // 存储读取失败时使用默认主题。
  }
  return DEFAULT_THEME_MODE
}

/** 主题实际生效时给导航栏与页面底色染色，覆盖 pages.json 的静态浅色配置。 */
const applyChrome = (theme: ResolvedTheme) => {
  try {
    const navigationBarColor = theme === 'light' ? '#f4f9fd' : '#060a12'
    uni.setNavigationBarColor({
      frontColor: theme === 'light' ? '#000000' : '#ffffff',
      backgroundColor: navigationBarColor,
      fail: () => undefined,
    })
    uni.setBackgroundColor({ backgroundColor: navigationBarColor, fail: () => undefined })
  } catch {
    // 部分平台/版本未暴露导航栏染色 API，跳过不影响主题类。
  }
  // H5 弹层（showModal/showToast）挂在 body 下，取不到页面根类，把主题类镜像到 html 上。
  if (isH5() && typeof document !== 'undefined') {
    document.documentElement.classList.toggle('theme-dark', theme === 'dark')
    document.documentElement.classList.toggle('theme-light', theme === 'light')
  }
}

/** 全局主题状态：模式持久化存储，auto 跟随系统深浅色。 */
export const useThemeStore = defineStore('theme', () => {
  const mode = ref<ThemeMode>(readStoredMode())
  const systemTheme = ref<ResolvedTheme>(readSystemTheme())

  const theme = computed<ResolvedTheme>(() =>
    mode.value === 'auto' ? systemTheme.value : mode.value,
  )
  const themeClass = computed(() => `theme-${theme.value}`)

  /** 切换主题模式并持久化。 */
  const setMode = (nextMode: ThemeMode) => {
    mode.value = nextMode
    try {
      uni.setStorageSync(THEME_MODE_STORAGE_KEY, nextMode)
    } catch {
      // 存储写入失败不影响当前会话主题。
    }
  }

  // 跟随系统：H5 监听 matchMedia，小程序监听 onThemeChange。
  if (isH5()) {
    if (typeof window.matchMedia === 'function') {
      const media = window.matchMedia('(prefers-color-scheme: light)')
      const listener = (event: MediaQueryListEvent) => {
        systemTheme.value = event.matches ? 'light' : 'dark'
      }
      if (typeof media.addEventListener === 'function') {
        media.addEventListener('change', listener)
      } else if (typeof media.addListener === 'function') {
        media.addListener(listener)
      }
    }
  } else {
    try {
      uni.onThemeChange?.((result: { theme?: string }) => {
        systemTheme.value = result.theme === 'light' ? 'light' : 'dark'
      })
    } catch {
      // 平台不支持系统主题回调时保持启动时读取的值。
    }
  }

  watch(theme, (value) => applyChrome(value), { immediate: true })

  return { mode, theme, themeClass, setMode }
})
