export { ADMIN_AI_EXTENSION, getAdminAiExtension } from "./ai";
export type { AdminAiExtension } from "./ai";
export { default as TenantProjectManager } from "./components/tenant-project/TenantProjectManager.vue";
export {
  mergeTenantProjectExtraData,
  tenantProjectKey,
  type TenantProjectAction,
  type TenantProjectContext,
  type TenantProjectExtraData,
  type TenantProjectExtraDataLoadContext,
  type TenantProjectExtraDataLoader,
  type TenantProjectKey,
  type TenantProjectManagerProps
} from "./components/tenant-project/tenant-project-manager";
export { systemAdminModule } from "./module";
