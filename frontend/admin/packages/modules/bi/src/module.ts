import { defineAsyncComponent, type Component } from "vue";
import { ADMIN_COMPONENT_DOCS, defineAdminModule } from "@liujitcn/kratos-admin-core";
import { ADMIN_AI_EXTENSION, type AdminAiExtension } from "@liujitcn/kratos-admin-system";
import { biComponentDocs } from "./demo/catalog";
import { LOCALE_MESSAGES } from "./locales/generated";

const viewModules = import.meta.glob<{ default: Component }>("./views/**/*.vue");

/** BI 分析管理端业务模块。 */
export const biAdminModule = defineAdminModule({
  name: "bi",
  views: viewModules,
  extensions: {
    [ADMIN_COMPONENT_DOCS]: biComponentDocs,
    [ADMIN_AI_EXTENSION]: {
      biCardDialog: defineAsyncComponent(() => import("./views/query/components/BiCardGuideDialog.vue"))
    } satisfies AdminAiExtension
  },
  messages: LOCALE_MESSAGES
});
