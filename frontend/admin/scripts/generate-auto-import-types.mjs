import { readFileSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { build } from "vite";

const adminRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const coreRequire = createRequire(resolve(adminRoot, "packages/core/package.json"));
const { default: AutoImport } = await import(pathToFileURL(coreRequire.resolve("unplugin-auto-import/vite")).href);
const { ElementPlusResolver } = await import(pathToFileURL(coreRequire.resolve("unplugin-vue-components/resolvers")).href);
const imports = JSON.parse(readFileSync(resolve(adminRoot, "packages/core/build/auto-imports.json"), "utf8"));
const virtualEntry = "virtual:kratos-auto-import-types";

await build({
  configFile: false,
  logLevel: "silent",
  plugins: [
    {
      name: "kratos-auto-import-types-entry",
      resolveId(id) {
        return id === virtualEntry ? `\0${virtualEntry}` : undefined;
      },
      load(id) {
        return id === `\0${virtualEntry}` ? "export {};" : undefined;
      }
    },
    AutoImport({
      dts: resolve(adminRoot, "packages/core/types/generated/auto-imports.d.ts"),
      dtsMode: "overwrite",
      imports: ["vue", "vue-router", imports],
      resolvers: [ElementPlusResolver()]
    })
  ],
  build: {
    write: false,
    rollupOptions: { input: virtualEntry }
  }
});
