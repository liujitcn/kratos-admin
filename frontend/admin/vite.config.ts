import { realpathSync } from "node:fs";
import { createLogger, defineConfig, loadEnv, type ConfigEnv, type Logger, type UserConfig } from "vite";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { wrapperEnv } from "./build/getEnv";
import { createProxy } from "./build/proxy";
import { createVitePlugins } from "./build/plugins";
import pkg from "./package.json";
import dayjs from "dayjs";

const hostRoot = dirname(fileURLToPath(import.meta.url));
const { dependencies, devDependencies, name, version } = pkg;
const __APP_INFO__ = {
  pkg: { dependencies, devDependencies, name, version },
  lastBuildTime: dayjs().format("YYYY-MM-DD HH:mm:ss")
};

const viteLogger = createLogger();
const ignoredBuildWarningMatchers = ["[lightningcss minify] 'deep' is not recognized as a valid pseudo-class"];
const ignoredRolldownWarningSources = ["node_modules/.pnpm/@vueuse+core@"];

/**
 * 创建构建日志过滤器，仅隐藏升级 Vite 8 后第三方依赖产生的已知噪音。
 */
const createBuildLogger = (): Logger => {
  const warn = viteLogger.warn;
  const warnOnce = viteLogger.warnOnce;

  const shouldIgnoreWarning = (message: string) => {
    return ignoredBuildWarningMatchers.some(matcher => message.includes(matcher));
  };

  return {
    ...viteLogger,
    warn(message, options) {
      if (shouldIgnoreWarning(message)) {
        return;
      }
      warn.call(viteLogger, message, options);
    },
    warnOnce(message, options) {
      if (shouldIgnoreWarning(message)) {
        return;
      }
      warnOnce.call(viteLogger, message, options);
    }
  };
};

/**
 * 将文件系统路径转换为正则表达式字面量。
 */
const escapeRegExp = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

// @see: https://vitejs.dev/config/
export default defineConfig(({ mode }: ConfigEnv): UserConfig => {
  const root = process.cwd();
  const isCompositeHost = resolve(root) !== resolve(hostRoot);
  const env = loadEnv(mode, root);
  const viteEnv = wrapperEnv(env);
  const packageRoot = resolve(root, "node_modules/@liujitcn/kratos-admin");
  const packageSourceRoots = isCompositeHost ? [packageRoot, realpathSync(packageRoot)] : [];
  const escapedRoot = escapeRegExp(root);
  const compositeSourcePatterns = isCompositeHost
    ? [
        new RegExp(`${escapedRoot}[\\/\\\\]src[\\/\\\\].*\\.(?:vue|[jt]sx?)(?:\\?.*)?$`),
        ...[...new Set(packageSourceRoots)].map(
          sourceRoot =>
            new RegExp(`${escapeRegExp(sourceRoot)}[\\/\\\\]dist[\\/\\\\]package[\\/\\\\]src[\\/\\\\].*\\.(?:vue|[jt]sx?)(?:\\?.*)?$`)
        )
      ]
    : undefined;

  return {
    base: viteEnv.VITE_PUBLIC_PATH,
    root,
    resolve: {
      alias: isCompositeHost
        ? {}
        : {
            "@": resolve(hostRoot, "./src")
          }
    },
    define: {
      __APP_INFO__: JSON.stringify(__APP_INFO__)
    },
    customLogger: createBuildLogger(),
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: `@use "${isCompositeHost ? `${name}/styles/var.scss` : "@/styles/var.scss"}" as *;`
        }
      }
    },
    optimizeDeps: isCompositeHost
      ? {
          include: [
            "dayjs",
            "dayjs/plugin/customParseFormat",
            "@liujitcn/kratos-admin > nprogress",
            "@liujitcn/kratos-admin > qs",
            "@liujitcn/kratos-admin > swagger-ui-dist/swagger-ui-bundle.js"
          ],
          // 组合宿主源码时关闭自动依赖扫描，避免把 @ 开头的源码别名当作第三方包预构建。
          noDiscovery: true,
          // kratos-admin 通过包名加载发布源码，预构建会错误改写其 ESM 命名导入。
          exclude: ["@liujitcn/kratos-admin"]
        }
      : undefined,
    server: {
      host: "0.0.0.0",
      port: viteEnv.VITE_PORT,
      open: viteEnv.VITE_OPEN,
      cors: true,
      // Load proxy configuration from .env.development
      proxy: createProxy(viteEnv.VITE_PROXY)
    },
    plugins: createVitePlugins(viteEnv, { sourcePatterns: compositeSourcePatterns }),
    build: {
      outDir: resolve(hostRoot, "../../backend/data/admin"),
      emptyOutDir: true,
      minify: "terser",
      terserOptions: {
        compress: {
          drop_console: viteEnv.VITE_DROP_CONSOLE,
          drop_debugger: true
        }
      },
      sourcemap: false,
      // 禁用 gzip 压缩大小报告，可略微减少打包时间
      reportCompressedSize: false,
      // 规定触发警告的 chunk 大小
      chunkSizeWarningLimit: 2000,
      rolldownOptions: {
        onLog(level, log, defaultHandler) {
          const id = log.id ?? "";
          const shouldIgnoreInvalidAnnotation =
            level === "warn" &&
            log.code === "INVALID_ANNOTATION" &&
            ignoredRolldownWarningSources.some(source => id.includes(source));

          if (shouldIgnoreInvalidAnnotation || log.code === "PLUGIN_TIMINGS") {
            return;
          }

          defaultHandler(level, log);
        },
        output: {
          // Static resource classification and packaging
          chunkFileNames: "assets/js/[name]-[hash].js",
          entryFileNames: "assets/js/[name]-[hash].js",
          assetFileNames: "assets/[ext]/[name]-[hash].[ext]"
        }
      }
    }
  };
});
