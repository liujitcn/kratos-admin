import type { Component } from "vue";
import { defineAdminModule } from "@liujitcn/kratos-admin-core";

// 语言包由同步脚本生成并注册，供 RAG 知识库页面使用。
import { LOCALE_MESSAGES } from "./locales/generated";

const viewModules = import.meta.glob<{ default: Component }>("./views/**/*.vue");

/** RAG 知识库管理端业务模块。 */
export const ragAdminModule = defineAdminModule({
  name: "rag",
  views: viewModules,
  messages: LOCALE_MESSAGES
});
