/// <reference path="./vite-env.d.ts" />
/// <reference path="./typings/global.d.ts" />
/// <reference path="./typings/utils.d.ts" />
/// <reference path="../types/generated/auto-imports.d.ts" />
/// <reference path="../types/generated/components.d.ts" />

export * from "./modules";
export * from "./locales";
export { defLanguageService } from "./api/base/language";
export { kratosAdminModule } from "./modules/kratosAdmin";
export { bootstrapAdminApp } from "./bootstrap";
