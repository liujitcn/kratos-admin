import "./vite-env.d.ts";
import "./typings/global.d.ts";
import "./typings/utils.d.ts";
import "../types/generated/auto-imports.d.ts";
import "../types/generated/components.d.ts";

export * from "./modules";
export * from "./locales";
export { defLanguageService } from "./api/base/v1/language";
export { setAdminDocumentTitle } from "./documentTitle";
export { kratosAdminModule } from "./modules/kratosAdmin";
export { bootstrapAdminApp } from "./bootstrap";
