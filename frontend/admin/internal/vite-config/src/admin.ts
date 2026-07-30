import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { defineAdminViteConfig, type AdminViteConfigOptions } from "@liujitcn/kratos-admin/vite.config";

const currentDirectory = dirname(fileURLToPath(import.meta.url));

/** 默认管理端宿主可由 module 清单提供的 Vite 配置。 */
export type AdminAppViteConfigOptions = Pick<AdminViteConfigOptions, "modulePackages" | "optimizeDependencies">;

/** 创建默认管理端宿主的 Vite 配置。 */
export function defineAdminAppViteConfig(options: AdminAppViteConfigOptions) {
  return defineAdminViteConfig({
    ...options,
    outputDirectory: resolve(currentDirectory, "../../../../../backend/data/admin")
  });
}
