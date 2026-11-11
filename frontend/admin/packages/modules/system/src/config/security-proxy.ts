import { Connection } from "@element-plus/icons-vue";
import type { RuntimeConfigDefinition } from "./types";

/** 反向代理安全配置表单。 */
export interface SecurityProxyConfig {
  trust_proxy: boolean;
  trusted_proxy_cidrs: string[];
}

/** 反向代理安全配置定义。 */
export const securityProxyConfig: RuntimeConfigDefinition<SecurityProxyConfig> = {
  key: "securityProxy",
  titleKey: "system.base.runtime_config.security_proxy.title",
  descriptionKey: "system.base.runtime_config.security_proxy.description",
  icon: Connection,
  createModel: () => ({ trust_proxy: false, trusted_proxy_cidrs: [] }),
  fields: [
    {
      prop: "trust_proxy",
      labelKey: "system.base.runtime_config.security_proxy.field.trust_proxy",
      component: "switch"
    },
    {
      prop: "trusted_proxy_cidrs",
      labelKey: "system.base.runtime_config.security_proxy.field.trusted_proxy_cidrs",
      component: "dynamic-list",
      colSpan: 24,
      props: { inputProps: { placeholder: "10.0.0.0/8" } }
    }
  ]
};
