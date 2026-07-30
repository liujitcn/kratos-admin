import { h, type Component } from "vue";
import dingtalkIconUrl from "@/assets/images/oauth/dingtalk.png";
import feishuIconUrl from "@/assets/images/oauth/feishu.png";
import giteeIconUrl from "@/assets/images/oauth/gitee.png";
import githubIconUrl from "@/assets/images/oauth/github.png";
import googleIconUrl from "@/assets/images/oauth/google.png";
import wechatIconUrl from "@/assets/images/oauth/wechat.png";
import wechatworkIconUrl from "@/assets/images/oauth/wechatwork.png";

/** 三方登录展示信息。 */
export interface OauthProviderDisplay {
  /** 登录方式名称。 */
  name: string;
  /** 登录方式图标标识。 */
  icon: string;
}

/** 带三方登录 provider 标识的数据。 */
export interface OauthProviderSource {
  /** 登录方式标识。 */
  provider: string;
}

const oauthProviderDisplayMap: Record<string, OauthProviderDisplay> = {
  github: { name: "GitHub", icon: "github" },
  wechat: { name: "微信", icon: "wechat" },
  wechatmp: { name: "微信公众号", icon: "wechat" },
  gitee: { name: "Gitee", icon: "gitee" },
  google: { name: "Google", icon: "google" },
  dingtalk: { name: "钉钉", icon: "dingtalk" },
  feishu: { name: "飞书", icon: "feishu" },
  wechatwork: { name: "企业微信", icon: "wechatwork" }
};

/** 创建本地官方图标图片组件。 */
const createOauthImageIcon = (src: string, icon: string): Component => {
  return () =>
    h("img", {
      src,
      alt: "",
      class: `oauth-provider-img oauth-provider-img--${icon}`,
      "aria-hidden": "true"
    });
};

/** 三方登录真实品牌图标组件映射。 */
const oauthIconMap: Record<string, Component> = {
  github: createOauthImageIcon(githubIconUrl, "github"),
  wechat: createOauthImageIcon(wechatIconUrl, "wechat"),
  google: createOauthImageIcon(googleIconUrl, "google"),
  gitee: createOauthImageIcon(giteeIconUrl, "gitee"),
  dingtalk: createOauthImageIcon(dingtalkIconUrl, "dingtalk"),
  feishu: createOauthImageIcon(feishuIconUrl, "feishu"),
  wechatwork: createOauthImageIcon(wechatworkIconUrl, "wechatwork")
};

/** 根据 provider 标识获取前端展示信息。 */
export function getOauthProviderDisplay(provider: string): OauthProviderDisplay {
  return oauthProviderDisplayMap[provider] || { name: provider || "未知方式", icon: provider };
}

/** 创建未知 provider 的文字兜底图标。 */
function createOauthFallbackIcon(provider: OauthProviderSource & OauthProviderDisplay): Component {
  const label = (provider.name || provider.provider || "?").slice(0, 1).toUpperCase();
  return () => h("span", { class: "oauth-provider-fallback", "aria-hidden": "true" }, label);
}

/** 获取三方登录真实品牌图标组件。 */
export function getOauthProviderIcon(provider: OauthProviderSource & OauthProviderDisplay): Component {
  return oauthIconMap[provider.icon] || oauthIconMap[provider.provider] || createOauthFallbackIcon(provider);
}

/** 为接口返回的三方登录数据补齐前端展示信息。 */
export function withOauthProviderDisplay<T extends OauthProviderSource>(item: T): T & OauthProviderDisplay {
  return {
    ...item,
    ...getOauthProviderDisplay(item.provider)
  };
}
