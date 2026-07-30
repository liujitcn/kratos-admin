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
  refreshToken: string;
  tokenType: string;
  tokenExpiresAt: number;
  userInfo: UserInfoForm;
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
}
