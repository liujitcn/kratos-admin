import type { Component } from "vue";
import { User } from "@element-plus/icons-vue";
import { defineAdminModule } from "@liujitcn/kratos-admin";
import Ai from "./components/Ai.vue";

const viewModules = import.meta.glob<{ default: Component }>("./views/**/*.vue");

/** System 管理端业务模块。 */
export const systemAdminModule = defineAdminModule({
  name: "system",
  views: viewModules,
  headerTools: [{ name: "ai", component: Ai }],
  userMenuActions: [{ name: "profile", label: "个人中心", menuName: "Profile", icon: User }],
  routeOptions: {
    Profile: { reuseTabAcrossQuery: true }
  }
});
