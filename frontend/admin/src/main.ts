import { createApp } from "vue";
import App from "./App.vue";
// reset style sheet
import "@/styles/reset.scss";
// CSS common style sheet
import "@/styles/common.scss";
// iconfont css
import "@/assets/iconfont/iconfont.scss";
// font css
import "@/assets/fonts/font.scss";
// element css
import "element-plus/dist/index.css";
// element dark css
import "element-plus/theme-chalk/dark/css-vars.css";
// custom element dark css
import "@/styles/element-dark.scss";
// custom element css
import "@/styles/element.scss";
// svg icons
import "virtual:svg-icons-register";
// element icons
import * as Icons from "@element-plus/icons-vue";
import { ElLoading } from "element-plus";
// custom directives
import directives from "@/directives/index";
// vue Router
import router from "@/routers";
// pinia store
import pinia from "@/stores";
// 字典组件
import Dict from "@/components/Dict/index.vue";
import DictLabel from "@/components/Dict/DictLabel.vue";
import SvgIcon from "@/components/SvgIcon/index.vue";
// errorHandler
import errorHandler from "@/utils/errorHandler";
import { useConfigStore } from "@/stores/modules/config";

/**
 * 启动前先完成全局配置初始化，确保首屏展示尽量使用最新站点配置。
 */
async function bootstrap() {
  const app = createApp(App);

  app.config.errorHandler = errorHandler;

  // 后端菜单图标以字符串形式返回，需要在运行时全局注册后才能被动态组件解析。
  Object.keys(Icons).forEach(key => {
    app.component(key, Icons[key as keyof typeof Icons]);
  });

  // 注册字典相关全局组件
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

  app.mount("#app");
}

bootstrap();
