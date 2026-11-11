import { Position } from "@element-plus/icons-vue";
import type { RuntimeConfigDefinition } from "./types";

/** 出站网络安全配置表单。 */
export interface SecurityOutboundConfig {
  allowed_hosts: string[];
  allowed_cidrs: string[];
  ai_allowed_hosts: string[];
  ai_allowed_cidrs: string[];
}

/** 出站网络安全配置定义。 */
export const securityOutboundConfig: RuntimeConfigDefinition<SecurityOutboundConfig> = {
  key: "securityOutbound",
  titleKey: "system.base.runtime_config.security_outbound.title",
  descriptionKey: "system.base.runtime_config.security_outbound.description",
  icon: Position,
  createModel: () => ({ allowed_hosts: [], allowed_cidrs: [], ai_allowed_hosts: [], ai_allowed_cidrs: [] }),
  fields: [
    {
      prop: "allowed_hosts",
      labelKey: "system.base.runtime_config.security_outbound.field.allowed_hosts",
      component: "dynamic-list",
      colSpan: 24,
      props: { inputProps: { placeholder: "api.example.com:443" } }
    },
    {
      prop: "allowed_cidrs",
      labelKey: "system.base.runtime_config.security_outbound.field.allowed_cidrs",
      component: "dynamic-list",
      colSpan: 24,
      props: { inputProps: { placeholder: "10.0.0.0/8" } }
    },
    {
      prop: "ai_allowed_hosts",
      labelKey: "system.base.runtime_config.security_outbound.field.ai_allowed_hosts",
      component: "dynamic-list",
      colSpan: 24,
      props: { inputProps: { placeholder: "model.internal:8443" } }
    },
    {
      prop: "ai_allowed_cidrs",
      labelKey: "system.base.runtime_config.security_outbound.field.ai_allowed_cidrs",
      component: "dynamic-list",
      colSpan: 24,
      props: { inputProps: { placeholder: "10.0.0.0/8" } }
    }
  ]
};
