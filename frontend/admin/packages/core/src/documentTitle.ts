import pinia from "@/stores";
import { useConfigStore } from "@/stores/modules/config";

/** setAdminDocumentTitle 使用后端站点名称更新管理端浏览器标题。 */
export function setAdminDocumentTitle(pageTitle = "") {
  const appTitle = useConfigStore(pinia).display.sysName || import.meta.env.VITE_GLOB_APP_TITLE;
  document.title = pageTitle ? `${pageTitle} - ${appTitle}` : appTitle;
}
