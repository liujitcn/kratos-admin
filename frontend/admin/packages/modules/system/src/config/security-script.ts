import { SetUp } from "@element-plus/icons-vue";
import type { RuntimeConfigDefinition } from "./types";

/** 脚本沙箱安全配置表单。 */
export interface SecurityScriptConfig {
  read_only_paths: string[];
  writable_paths: string[];
  commands: string[];
  allowed_hosts: string[];
  allowed_cidrs: string[];
}

/** 脚本沙箱安全配置定义。 */
export const securityScriptConfig: RuntimeConfigDefinition<SecurityScriptConfig> = {
  key: "securityScript",
  titleKey: "system.base.runtime_config.security_script.title",
  descriptionKey: "system.base.runtime_config.security_script.description",
  icon: SetUp,
  createModel: () => ({ read_only_paths: [], writable_paths: [], commands: [], allowed_hosts: [], allowed_cidrs: [] }),
  fields: [
    {
      prop: "read_only_paths",
      labelKey: "system.base.runtime_config.security_script.field.read_only_paths",
      component: "dynamic-list",
      colSpan: 24,
      props: { inputProps: { placeholder: "/data/read-only" } }
    },
    {
      prop: "writable_paths",
      labelKey: "system.base.runtime_config.security_script.field.writable_paths",
      component: "dynamic-list",
      colSpan: 24,
      props: { inputProps: { placeholder: "/data/work" } }
    },
    {
      prop: "commands",
      labelKey: "system.base.runtime_config.security_script.field.commands",
      component: "dynamic-list",
      colSpan: 24,
      props: { inputProps: { placeholder: "/usr/bin/tool" } }
    },
    {
      prop: "allowed_hosts",
      labelKey: "system.base.runtime_config.security_script.field.allowed_hosts",
      component: "dynamic-list",
      colSpan: 24,
      props: { inputProps: { placeholder: "api.example.com:443" } }
    },
    {
      prop: "allowed_cidrs",
      labelKey: "system.base.runtime_config.security_script.field.allowed_cidrs",
      component: "dynamic-list",
      colSpan: 24,
      props: { inputProps: { placeholder: "10.0.0.0/8" } }
    }
  ]
};
