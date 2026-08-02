import { createRequire } from "node:module";
import { existsSync, readFileSync } from "node:fs";
import { createLogger, defineConfig, loadEnv, type ConfigEnv, type Logger, type UserConfig } from "vite";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import dayjs from "dayjs";
import { wrapperEnv } from "./build/getEnv";
import { createProxy } from "./build/proxy";
import { createVitePlugins } from "./build/plugins";

const coreRoot = dirname(fileURLToPath(import.meta.url));
const viteLogger = createLogger();
const ignoredBuildWarningMatchers = ["[lightningcss minify] 'deep' is not recognized as a valid pseudo-class"];
const ignoredRolldownWarningSources = ["node_modules/.pnpm/@vueuse+core@"];

/** 管理端源码包的名称和目录信息。 */
interface AdminSourcePackage {
  packageName: string;
  packageRoot: string;
  sourceRoot: string;
}

/** 管理端宿主 Vite 配置参数。 */
export interface AdminViteConfigOptions {
  /** 宿主启用的业务模块 npm 包名。 */
  modulePackages?: string[];
  /** 业务模块需要预构建的依赖。 */
  optimizeDependencies?: string[];
  /** 静态资源构建输出目录。 */
  outputDirectory?: string;
}

/** 创建管理端宿主 Vite 配置。 */
export function defineAdminViteConfig(options: AdminViteConfigOptions = {}) {
  return defineConfig(({ mode }: ConfigEnv): UserConfig => {
    const root = process.cwd();
    const env = loadEnv(mode, root);
    const viteEnv = wrapperEnv(env);
    const packageInfo = readPackageInfo(root);
    const coreSourceRoot = resolveSourceRoot(coreRoot);
    const corePackage = {
      packageName: "@liujitcn/kratos-admin-core",
      packageRoot: coreRoot,
      sourceRoot: coreSourceRoot
    };
    const modulePackages = (options.modulePackages ?? []).map(packageName => ({
      packageName,
      ...resolvePackageSource(root, packageName)
    }));
    const sourceRoots = [resolve(root, "src"), coreSourceRoot, ...modulePackages.map(item => item.sourceRoot)];
    const sourcePatterns = [...new Set(sourceRoots)].map(
      sourceRoot => new RegExp(`${escapeRegExp(sourceRoot)}[\\/\\\\].*\\.(?:vue|[jt]sx?)(?:\\?.*)?$`)
    );
    const aliases = [
      { find: "@", replacement: coreSourceRoot },
      ...createPackageAliases(corePackage),
      ...modulePackages.flatMap(createPackageAliases)
    ];
    const appInfo = {
      pkg: packageInfo,
      lastBuildTime: dayjs().format("YYYY-MM-DD HH:mm:ss")
    };

    return {
      base: viteEnv.VITE_PUBLIC_PATH,
      root,
      resolve: { alias: aliases },
      define: {
        __APP_INFO__: JSON.stringify(appInfo)
      },
      customLogger: createBuildLogger(),
      css: {
        preprocessorOptions: {
          scss: {
            additionalData: `@use "${resolve(coreSourceRoot, "styles/var.scss")}" as *;`
          }
        }
      },
      optimizeDeps: {
        include: [
          "@liujitcn/kratos-admin-core > dayjs",
          // Day.js locale 是 CommonJS，必须与核心实例一起预构建，避免开发环境丢失 dayjs.locale。
          "@liujitcn/kratos-admin-core > dayjs/locale/en",
          "@liujitcn/kratos-admin-core > dayjs/locale/ja",
          "@liujitcn/kratos-admin-core > dayjs/locale/zh-cn",
          "@liujitcn/kratos-admin-core > dayjs/locale/zh-tw",
          "@liujitcn/kratos-admin-core > dayjs/locale/ko",
          "@liujitcn/kratos-admin-core > dayjs/locale/fr",
          "@liujitcn/kratos-admin-core > dayjs/locale/es",
          "@liujitcn/kratos-admin-core > dayjs/plugin/advancedFormat.js",
          "@liujitcn/kratos-admin-core > dayjs/plugin/customParseFormat.js",
          "@liujitcn/kratos-admin-core > dayjs/plugin/dayOfYear.js",
          "@liujitcn/kratos-admin-core > dayjs/plugin/isSameOrAfter.js",
          "@liujitcn/kratos-admin-core > dayjs/plugin/isSameOrBefore.js",
          "@liujitcn/kratos-admin-core > dayjs/plugin/localeData.js",
          "@liujitcn/kratos-admin-core > dayjs/plugin/weekOfYear.js",
          "@liujitcn/kratos-admin-core > dayjs/plugin/weekYear.js",
          "@liujitcn/kratos-admin-core > dompurify",
          "@liujitcn/kratos-admin-core > highlight.js",
          "@liujitcn/kratos-admin-core > md-editor-v3",
          "@liujitcn/kratos-admin-core > nprogress",
          "@liujitcn/kratos-admin-core > qs",
          ...(options.optimizeDependencies ?? [])
        ],
        noDiscovery: true,
        exclude: ["vue", "@liujitcn/kratos-admin-core", ...(options.modulePackages ?? [])]
      },
      server: {
        host: "0.0.0.0",
        port: viteEnv.VITE_PORT,
        open: viteEnv.VITE_OPEN,
        cors: true,
        proxy: createProxy(viteEnv.VITE_PROXY)
      },
      plugins: createVitePlugins(viteEnv, {
        sourcePatterns,
        autoImportDts: false,
        componentDts: false,
        iconDirectories: [resolve(coreSourceRoot, "assets/icons")]
      }),
      build: {
        outDir: options.outputDirectory ?? resolve(root, "dist"),
        emptyOutDir: true,
        minify: "terser",
        terserOptions: {
          compress: {
            drop_console: viteEnv.VITE_DROP_CONSOLE,
            drop_debugger: true
          }
        },
        sourcemap: false,
        reportCompressedSize: false,
        chunkSizeWarningLimit: 2000,
        rolldownOptions: {
          onLog(level, log, defaultHandler) {
            const id = log.id ?? "";
            const shouldIgnoreInvalidAnnotation =
              level === "warn" &&
              log.code === "INVALID_ANNOTATION" &&
              ignoredRolldownWarningSources.some(source => id.includes(source));

            if (shouldIgnoreInvalidAnnotation || log.code === "PLUGIN_TIMINGS") return;
            defaultHandler(level, log);
          },
          output: {
            chunkFileNames: "assets/js/[name]-[hash].js",
            entryFileNames: "assets/js/[name]-[hash].js",
            assetFileNames: "assets/[ext]/[name]-[hash].[ext]"
          }
        }
      }
    };
  });
}

/** 创建构建日志过滤器，仅隐藏第三方依赖产生的已知噪音。 */
function createBuildLogger(): Logger {
  const warn = viteLogger.warn;
  const warnOnce = viteLogger.warnOnce;
  const shouldIgnoreWarning = (message: string) => {
    return ignoredBuildWarningMatchers.some(matcher => message.includes(matcher));
  };

  return {
    ...viteLogger,
    warn(message, logOptions) {
      if (shouldIgnoreWarning(message)) return;
      warn.call(viteLogger, message, logOptions);
    },
    warnOnce(message, logOptions) {
      if (shouldIgnoreWarning(message)) return;
      warnOnce.call(viteLogger, message, logOptions);
    }
  };
}

/** 读取宿主 package.json 中用于构建信息展示的字段。 */
function readPackageInfo(root: string) {
  const packageJson = JSON.parse(readFileSync(resolve(root, "package.json"), "utf8"));
  const { dependencies = {}, devDependencies = {}, name = "admin-app", version = "0.0.0" } = packageJson;
  return { dependencies, devDependencies, name, version };
}

/** 解析业务模块在当前宿主中的源码目录。 */
function resolvePackageSource(root: string, packageName: string): Pick<AdminSourcePackage, "packageRoot" | "sourceRoot"> {
  const require = createRequire(resolve(root, "package.json"));
  const packageJsonPath = require.resolve(`${packageName}/package.json`);
  const packageRoot = dirname(packageJsonPath);
  return { packageRoot, sourceRoot: resolveSourceRoot(packageRoot) };
}

/** 兼容 workspace 源码包与 npm 发布包的源码目录。 */
function resolveSourceRoot(packageRoot: string): string {
  const workspaceSourceRoot = resolve(packageRoot, "src");
  if (existsSync(resolve(workspaceSourceRoot, "index.ts"))) return workspaceSourceRoot;

  const publishedSourceRoot = resolve(packageRoot, "dist/package/src");
  if (existsSync(resolve(publishedSourceRoot, "index.ts"))) return publishedSourceRoot;
  throw new Error(`管理端模块缺少源码入口: ${packageRoot}`);
}

/** 根据 package.json exports 创建源码别名。 */
function createPackageAliases(sourcePackage: AdminSourcePackage) {
  const packageJson = JSON.parse(readFileSync(resolve(sourcePackage.packageRoot, "package.json"), "utf8")) as {
    exports?: Record<string, unknown>;
  };

  return Object.entries(packageJson.exports ?? {}).flatMap(([subpath, exportValue]) => {
    const target = resolveRuntimeExportTarget(exportValue);
    if (!target) return [];

    const importPath = subpath === "." ? sourcePackage.packageName : `${sourcePackage.packageName}/${subpath.slice(2)}`;
    const replacement = resolveSourceExportTarget(sourcePackage, target);
    if (!importPath.includes("*")) {
      return [{ find: new RegExp(`^${escapeRegExp(importPath)}$`), replacement }];
    }

    return [
      {
        find: new RegExp(`^${escapeRegExp(importPath).replace("\\*", "(.+)")}$`),
        replacement: replacement.replace("*", "$1")
      }
    ];
  });
}

/** 从条件导出中读取运行时目标。 */
function resolveRuntimeExportTarget(exportValue: unknown): string | undefined {
  if (typeof exportValue === "string") return exportValue;
  if (!exportValue || typeof exportValue !== "object") return undefined;

  const conditions = exportValue as Record<string, unknown>;
  return resolveRuntimeExportTarget(conditions.default ?? conditions.import);
}

/** 将发布目录中的导出目标映射到当前源码目录。 */
function resolveSourceExportTarget(sourcePackage: AdminSourcePackage, target: string): string {
  const publishedSourcePrefix = "./dist/package/src/";
  if (target.startsWith(publishedSourcePrefix)) {
    return resolve(sourcePackage.sourceRoot, target.slice(publishedSourcePrefix.length));
  }
  return resolve(sourcePackage.packageRoot, target);
}

/** 将文件系统路径转换为正则表达式字面量。 */
function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

export default defineAdminViteConfig();
