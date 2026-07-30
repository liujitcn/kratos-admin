import { defineAdminAppViteConfig } from "@liujitcn/admin-vite-config/admin";
import { adminModuleOptimizeDependencies, adminModulePackages } from "./src/module-manifest";

export default defineAdminAppViteConfig({
  modulePackages: adminModulePackages,
  optimizeDependencies: adminModuleOptimizeDependencies
});
