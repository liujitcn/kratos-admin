import { createLogger, defineConfig, loadEnv, type ConfigEnv, type Logger, type UserConfig } from "vite";
import { resolve } from "path";
import { wrapperEnv } from "./build/getEnv";
import { createProxy } from "./build/proxy";
import { createVitePlugins } from "./build/plugins";
import pkg from "./package.json";
import dayjs from "dayjs";

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

// @see: https://vitejs.dev/config/
export default defineConfig(({ mode }: ConfigEnv): UserConfig => {
  const root = process.cwd();
  const env = loadEnv(mode, root);
  const viteEnv = wrapperEnv(env);

  return {
    base: viteEnv.VITE_PUBLIC_PATH,
    root,
    resolve: {
      alias: {
        "@": resolve(__dirname, "./src")
      }
    },
    define: {
      __APP_INFO__: JSON.stringify(__APP_INFO__)
    },
    customLogger: createBuildLogger(),
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: `@use "@/styles/var.scss" as *;`
        }
      }
    },
    server: {
      host: "0.0.0.0",
      port: viteEnv.VITE_PORT,
      open: viteEnv.VITE_OPEN,
      cors: true,
      // Load proxy configuration from .env.development
      proxy: createProxy(viteEnv.VITE_PROXY)
    },
    plugins: createVitePlugins(viteEnv),
    build: {
      outDir: resolve(__dirname, "../../backend/data/admin"),
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
