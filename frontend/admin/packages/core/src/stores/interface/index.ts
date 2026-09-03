import type { RouteItem, UserInfoForm } from "@/rpc/system/admin/v1/auth";
import type { OptionBaseDictResponse_BaseDictItem } from "@/rpc/system/admin/v1/base_dict";

/** 后台整体布局模式。 */
export type LayoutType = "vertical" | "classic" | "transverse" | "columns";

/** Element Plus 组件默认尺寸。 */
export type AssemblySizeType = "large" | "default" | "small";

/** 站点展示配置 */
export interface SiteDisplayConfig {
  /** 项目名称 */
  sysName: string;
  /** ICP 备案号 */
  icp: string;
  /** 版权文案 */
  copyright: string;
  /** 页面水印文案 */
  watermark: string;
  /** 管理端 Logo 地址 */
  adminLogo: string;
  /** 登录页左侧背景图地址 */
  background: string;
}

/** 登录验证码配置 */
export interface LoginCaptchaConfig {
  /** 验证码类型 */
  type: string;
}

/** 全局 UI 偏好和布局状态。 */
export interface GlobalState {
  layout: LayoutType;
  assemblySize: AssemblySizeType;
  maximize: boolean;
  primary: string;
  isDark: boolean;
  isGrey: boolean;
  isWeak: boolean;
  asideInverted: boolean;
  headerInverted: boolean;
  isCollapse: boolean;
  accordion: boolean;
  watermark: boolean;
  breadcrumb: boolean;
  breadcrumbIcon: boolean;
  tabs: boolean;
  tabsIcon: boolean;
  footer: boolean;
}

/** 当前登录用户认证状态。 */
export interface UserState {
  token: string;
  tokenType: string;
  tokenExpiresAt: number;
  /** 是否正在执行退出登录，阻止后台请求刷新令牌。 */
  isLoggingOut: boolean;
  /** 认证状态版本，用于丢弃退出期间完成的旧刷新请求。 */
  authVersion: number;
  /** 当前认证是否已被本地显式清理。 */
  authInvalidated: boolean;
  userInfo: UserInfoForm;
}

/** 当前管理端锁屏状态。 */
export interface LockScreenState {
  /** 是否处于锁屏状态。 */
  isLocked: boolean;
  /** 当前锁屏密码的摘要，仅在锁屏期间持久化。 */
  passwordHash: string;
  /** 是否显示设置锁屏密码界面。 */
  setupVisible: boolean;
  /** 是否显示解锁输入界面。 */
  unlockVisible: boolean;
  /** 最近一次解锁是否失败。 */
  unlockError: boolean;
}

/** 标签页菜单项状态。 */
export interface TabsMenuProps {
  icon: string;
  title: string;
  path: string;
  name: string;
  close: boolean;
  isKeepAlive: boolean;
}

/** 多标签页状态。 */
export interface TabsState {
  tabsMenuList: TabsMenuProps[];
}

/** 动态路由和按钮权限状态。 */
export interface AuthState {
  routeName: string;
  authButtonList: {
    [key: string]: string[];
  };
  authMenuList: RouteItem[];
}

/** 需要缓存的页面组件名称状态。 */
export interface KeepAliveState {
  keepAliveName: string[];
}

/** 全局字典缓存状态。 */
export interface DictState {
  dictionary: Record<string, OptionBaseDictResponse_BaseDictItem[]>;
}

/** 站点配置状态 */
export interface SiteConfigState {
  /** 当前站点展示配置 */
  display: SiteDisplayConfig;
  /** 当前登录验证码配置 */
  captcha: LoginCaptchaConfig;
  /** 是否允许显式生成机器翻译草稿。 */
  i18nDraftEnabled: boolean;
}
