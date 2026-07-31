import { defineAdminViteConfig } from "@liujitcn/kratos-admin-core/vite.config";
import { adminModuleOptimizeDependencies, adminModulePackages } from "./src/module-manifest";

export default defineAdminViteConfig({
  modulePackages: adminModulePackages,
  optimizeDependencies: adminModuleOptimizeDependencies
});
