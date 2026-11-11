import { User } from "@element-plus/icons-vue";
import type { RuntimeConfigDefinition } from "./types";

/** 账号安全配置表单。 */
export interface SecurityIdentityConfig {
  reserved_usernames: string[];
  account_renames: Array<{ key: string; value: string }>;
}

/** 账号安全配置定义。 */
export const securityIdentityConfig: RuntimeConfigDefinition<SecurityIdentityConfig> = {
  key: "securityIdentity",
  titleKey: "system.base.runtime_config.security_identity.title",
  descriptionKey: "system.base.runtime_config.security_identity.description",
  icon: User,
  createModel: () => ({ reserved_usernames: [], account_renames: [] }),
  fields: [
    {
      prop: "reserved_usernames",
      labelKey: "system.base.runtime_config.security_identity.field.reserved_usernames",
      component: "dynamic-list",
      colSpan: 24,
      props: { inputProps: { placeholder: "admin" } }
    },
    {
      prop: "account_renames",
      labelKey: "system.base.runtime_config.security_identity.field.account_renames",
      component: "kv-list",
      colSpan: 24,
      props: {
        keyInputProps: { placeholder: "用户 ID" },
        valueInputProps: { placeholder: "新用户名" }
      }
    }
  ]
};
