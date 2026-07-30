import { loadAdminModules } from "./module-manifest";

/** 当前宿主启用的全部管理端业务模块。 */
const adminModules = await loadAdminModules();

export default adminModules;
