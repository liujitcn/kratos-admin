import type { AdminModule } from "@liujitcn/kratos-admin-core";

/** 管理端宿主模块清单项。 */
export interface AdminModuleManifestItem {
  /** 业务模块 npm 包名，供 Vite 扫描模块源码。 */
  packageName: string;
  /** 加载业务模块运行时定义。 */
  load: () => Promise<AdminModule>;
  /** 业务模块需要预构建的依赖。 */
  optimizeDependencies?: string[];
}

/** 当前宿主启用的管理端业务模块清单。 */
export const adminModuleManifest = [__MODULE_MANIFEST__] satisfies AdminModuleManifestItem[];

/** 编译清单保留内置 System 源码，供继承它的本地模块完成自动导入。 */
const adminBuildModules: Pick<AdminModuleManifestItem, "packageName" | "optimizeDependencies">[] = [
  { packageName: "@liujitcn/kratos-admin-system", optimizeDependencies: ["swagger-ui-dist/swagger-ui-bundle.js"] },
  ...adminModuleManifest.filter(item => item.packageName !== "@liujitcn/kratos-admin-system")
];

/** 当前宿主需要扫描的业务模块包。 */
export const adminModulePackages = adminBuildModules.map(item => item.packageName);

/** 当前宿主需要预构建的业务模块依赖。 */
export const adminModuleOptimizeDependencies = adminBuildModules.flatMap(item => {
  return item.optimizeDependencies?.map(dependency => `${item.packageName} > ${dependency}`) ?? [];
});

/** 加载当前宿主启用的全部业务模块。 */
export async function loadAdminModules(): Promise<AdminModule[]> {
  return Promise.all(adminModuleManifest.map(item => item.load()));
}
