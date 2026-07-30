import { defineAdminViteConfig } from "@liujitcn/kratos-admin/vite.config";
import { adminModuleOptimizeDependencies, adminModulePackages } from "./src/module-manifest";

export default defineAdminViteConfig({
  modulePackages: adminModulePackages,
  optimizeDependencies: adminModuleOptimizeDependencies
});
