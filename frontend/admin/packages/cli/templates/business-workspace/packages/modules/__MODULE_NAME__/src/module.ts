import type { Component } from "vue";
import { defineAdminModule } from "@liujitcn/kratos-admin";

const viewModules = import.meta.glob<{ default: Component }>("./views/**/*.vue");

/** __MODULE_PASCAL__ 管理端业务模块。 */
export const __MODULE_IDENTIFIER__ = defineAdminModule({
  name: "__MODULE_NAME__",
  views: viewModules
});
