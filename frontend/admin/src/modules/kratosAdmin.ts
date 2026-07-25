import type { Component } from "vue";
import { defineAdminModule } from "./index";

const viewModules = import.meta.glob<{ default: Component }>([
  "../views/base/**/*.vue",
  "../views/system/**/*.vue",
  "../views/migration/**/*.vue"
]);

/**
 * kratos-admin 内置基础模块。
 */
export const kratosAdminModule = defineAdminModule({
  name: "kratos-admin",
  views: viewModules
});
