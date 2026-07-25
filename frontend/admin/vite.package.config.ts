import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import vueJsx from "@vitejs/plugin-vue-jsx";
import { resolve } from "node:path";
import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";
import vueSetupExtend from "unplugin-vue-setup-extend-plus/vite";
import { createSvgIconsPlugin } from "vite-plugin-svg-icons";

/**
 * 构建可被 shop 组合的 kratos-admin 前端模块包。
 */
export default defineConfig({
  plugins: [
    vue(),
    vueJsx(),
    AutoImport({
      dts: false,
      resolvers: [ElementPlusResolver()],
      imports: ["vue", "vue-router"]
    }),
    Components({
      dts: false,
      dirs: [],
      resolvers: [ElementPlusResolver()]
    }),
    vueSetupExtend({}),
    createSvgIconsPlugin({
      iconDirs: [resolve(__dirname, "./src/assets/icons")],
      symbolId: "icon-[dir]-[name]"
    })
  ],
  resolve: {
    alias: {
      "@": resolve(__dirname, "./src")
    }
  },
  build: {
    outDir: resolve(__dirname, "./dist/package"),
    emptyOutDir: true,
    lib: {
      entry: resolve(__dirname, "./src/index.ts"),
      formats: ["es"],
      fileName: "index"
    },
    rollupOptions: {
      external: id =>
        id === "vue" ||
        id.startsWith("vue/") ||
        id === "vue-router" ||
        id === "pinia" ||
        id.startsWith("pinia/") ||
        id === "element-plus" ||
        id.startsWith("element-plus/") ||
        id === "@element-plus/icons-vue" ||
        id.startsWith("@vueuse/") ||
        id.startsWith("axios")
    }
  }
});
