import { defineStore } from "pinia";
import { defConfigService } from "@/api/base/config";
import { BaseConfigSite } from "@/rpc/common/v1/enum";
import type { LoginCaptchaConfig, SiteConfigState, SiteDisplayConfig } from "@/stores/interface";
import piniaPersistConfig from "@/stores/helper/persist";
import defaultLogoUrl from "@/assets/images/logo.svg";
import defaultBackgroundUrl from "@/assets/images/login_left.png";

const CAPTCHA_TYPE_KEY = "captchaType";

const DEFAULT_SITE_DISPLAY_CONFIG: SiteDisplayConfig = {
  sysName: "Admin",
  icp: "",
  copyright: "2026 © Admin",
  watermark: "Admin",
  adminLogo: defaultLogoUrl,
  background: defaultBackgroundUrl
};

const DEFAULT_LOGIN_CAPTCHA_CONFIG: LoginCaptchaConfig = {
  type: "digit"
};

/**
 * 合并站点展示配置，忽略空字符串配置，避免后端返回空值覆盖默认值。
 */
function mergeSiteDisplayConfig(
  currentDisplayConfig: SiteDisplayConfig,
  nextDisplayConfig: Partial<SiteDisplayConfig>
): SiteDisplayConfig {
  const mergedDisplayConfig = { ...currentDisplayConfig };

  Object.entries(nextDisplayConfig).forEach(([key, value]) => {
    if (typeof value !== "string") return;
    if (!value.trim()) return;
    mergedDisplayConfig[key as keyof SiteDisplayConfig] = value;
  });

  return mergedDisplayConfig;
}

/**
 * 将服务端配置项转换为站点展示配置字段。
 */
function normalizeSiteDisplayConfig(configMap: Record<string, string>) {
  return {
    sysName: configMap.sysName,
    icp: configMap.icp,
    copyright: configMap.copyright,
    watermark: configMap.watermark,
    adminLogo: configMap.adminLogo,
    background: configMap.background
  } satisfies Partial<SiteDisplayConfig>;
}

/**
 * 将服务端配置项转换为登录验证码配置字段。
 */
function normalizeLoginCaptchaConfig(configMap: Record<string, string>) {
  return {
    type: configMap[CAPTCHA_TYPE_KEY]
  } satisfies Partial<LoginCaptchaConfig>;
}

/**
 * 将公共配置列表转换为便于读取的键值映射。
 */
function buildConfigMap(configs: Array<{ key?: string; value?: string }>) {
  const configMap: Record<string, string> = {};
  configs.forEach(configItem => {
    if (!configItem.key) return;
    configMap[configItem.key] = configItem.value ?? "";
  });
  return configMap;
}

export const useConfigStore = defineStore("admin-config", {
  state: (): SiteConfigState => ({
    display: { ...DEFAULT_SITE_DISPLAY_CONFIG },
    captcha: { ...DEFAULT_LOGIN_CAPTCHA_CONFIG }
  }),
  getters: {},
  actions: {
    /**
     * 设置站点展示配置。
     */
    setDisplayConfig(nextDisplayConfig: Partial<SiteDisplayConfig>) {
      this.display = mergeSiteDisplayConfig(this.display, nextDisplayConfig);
    },
    /**
     * 设置登录验证码配置。
     */
    setLoginCaptchaConfig(nextCaptchaConfig: Partial<LoginCaptchaConfig>) {
      this.captcha = {
        ...this.captcha,
        ...Object.fromEntries(Object.entries(nextCaptchaConfig).filter(([, value]) => typeof value === "string" && value))
      };
    },
    /**
     * 重置为默认站点展示配置。
     */
    resetDisplayConfig() {
      this.display = { ...DEFAULT_SITE_DISPLAY_CONFIG };
      this.captcha = { ...DEFAULT_LOGIN_CAPTCHA_CONFIG };
    },
    /**
     * 加载管理端站点配置，并以服务端返回值覆盖本地默认值。
     */
    async loadDisplayConfig() {
      const configResponse = await defConfigService.GetConfig({
        site: BaseConfigSite.ADMIN
      });
      const configMap = buildConfigMap(configResponse.configs ?? []);

      this.setDisplayConfig(normalizeSiteDisplayConfig(configMap));
      this.setLoginCaptchaConfig(normalizeLoginCaptchaConfig(configMap));
      return this.display;
    }
  },
  persist: piniaPersistConfig("admin-config")
});
