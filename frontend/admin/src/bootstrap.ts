import { createApp } from "vue";
import App from "@/App.vue";
import "@/styles/reset.scss";
import "@/styles/common.scss";
import "@/assets/iconfont/iconfont.scss";
import "@/assets/fonts/font.scss";
import "element-plus/dist/index.css";
import "element-plus/theme-chalk/dark/css-vars.css";
import "@/styles/element-dark.scss";
import "@/styles/element.scss";
import "virtual:svg-icons-register";
import * as Icons from "@element-plus/icons-vue";
import { ElLoading } from "element-plus";
import directives from "@/directives/index";
import router from "@/routers";
import pinia from "@/stores";
import Dict from "@/components/Dict/index.vue";
import DictLabel from "@/components/Dict/DictLabel.vue";
import SvgIcon from "@/components/SvgIcon/index.vue";
import errorHandler from "@/utils/errorHandler";
import { useConfigStore } from "@/stores/modules/config";
import { kratosAdminModule } from "./modules/kratosAdmin";
import { registerAdminModules } from "./modules";
import type { AdminModule } from "./modules";

/**
 * 管理端启动参数。
 */
export interface AdminBootstrapOptions {
  /** 业务模块列表。 */
  modules?: AdminModule[];
  /** Vue 应用挂载节点。 */
  mount?: string;
}

/**
 * 创建并启动管理端应用，业务项目可通过 modules 注册自己的页面。
 */
export async function bootstrapAdminApp(options: AdminBootstrapOptions = {}) {
  registerAdminModules([kratosAdminModule, ...(options.modules ?? [])]);

  const app = createApp(App);

  app.config.errorHandler = errorHandler;

  // 后端菜单图标以字符串形式返回，需要在运行时全局注册后才能被动态组件解析。
  Object.keys(Icons).forEach(key => {
    app.component(key, Icons[key as keyof typeof Icons]);
  });

  // 注册字典相关全局组件。
  app.component("Dict", Dict);
  app.component("DictLabel", DictLabel);
  app.component("SvgIcon", SvgIcon);

  // 按需加载模式不会自动注册 v-loading 指令，需要显式挂载。
  app.directive("loading", ElLoading.directive);

  app.use(directives).use(pinia).use(router);

  try {
    await useConfigStore().loadDisplayConfig();
  } catch {
    // 配置接口失败时继续使用本地默认配置，避免阻断应用启动。
  }

  app.mount(options.mount ?? "#app");
  return app;
}
