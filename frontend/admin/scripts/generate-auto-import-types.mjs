import { readFileSync, readdirSync } from "node:fs";
import { createRequire } from "node:module";
import { dirname, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { build, transformWithOxc } from "vite";

const adminRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const coreRequire = createRequire(resolve(adminRoot, "packages/core/package.json"));
const { default: AutoImport } = await import(pathToFileURL(coreRequire.resolve("unplugin-auto-import/vite")).href);
const { default: Components } = await import(pathToFileURL(coreRequire.resolve("unplugin-vue-components/vite")).href);
const { ElementPlusResolver } = await import(pathToFileURL(coreRequire.resolve("unplugin-vue-components/resolvers")).href);
const { parse, compileTemplate } = coreRequire("vue/compiler-sfc");
const imports = JSON.parse(readFileSync(resolve(adminRoot, "packages/core/build/auto-imports.json"), "utf8"));
const virtualEntry = "virtual:kratos-auto-import-types";
const templatePrefix = "virtual:kratos-component-types/";
const moduleRoot = resolve(adminRoot, "packages/modules");
const sourceRoots = [
  resolve(adminRoot, "apps/admin/src"),
  resolve(adminRoot, "packages/core/src"),
  ...readdirSync(moduleRoot, { withFileTypes: true })
    .filter(entry => entry.isDirectory())
    .map(entry => resolve(moduleRoot, entry.name, "src"))
];
const templates = sourceRoots.flatMap(root =>
  readdirSync(root, { recursive: true })
    .filter(name => name.endsWith(".vue"))
    .sort()
    .map(name => resolve(root, name))
);

await build({
  configFile: false,
  logLevel: "silent",
  plugins: [
    {
      name: "kratos-auto-import-types-entry",
      resolveId(id) {
        return id === virtualEntry || id.startsWith(templatePrefix) ? id : undefined;
      },
      async load(id) {
        if (id === virtualEntry) {
          return templates.map((_, index) => `import "${templatePrefix}${index}.vue";`).join("\n");
        }
        if (!id.startsWith(templatePrefix)) return;
        const filename = templates[Number(id.slice(templatePrefix.length, -4))];
        const parsed = parse(readFileSync(filename, "utf8"), { filename });
        if (parsed.errors.length) throw new Error(`${filename}: ${parsed.errors.join("\n")}`);
        if (!parsed.descriptor.template) return "export {};";
        // 只编译模板，让真实组件解析器生成声明，不打包业务代码或依赖。
        const compiled = compileTemplate({ source: parsed.descriptor.template.content, filename, id });
        if (compiled.errors.length) throw new Error(`${filename}: ${compiled.errors.join("\n")}`);
        return (await transformWithOxc(compiled.code, `${filename}.ts`, { lang: "ts" })).code;
      }
    },
    AutoImport({
      dts: resolve(adminRoot, "packages/core/types/generated/auto-imports.d.ts"),
      dtsMode: "overwrite",
      imports: ["vue", "vue-router", imports],
      resolvers: [ElementPlusResolver()]
    }),
    Components({
      dts: resolve(adminRoot, "packages/core/types/generated/components.d.ts"),
      syncMode: "overwrite",
      dirs: [],
      resolvers: [ElementPlusResolver()]
    })
  ],
  build: {
    write: false,
    rollupOptions: {
      input: virtualEntry,
      external: id => id !== virtualEntry && !id.startsWith(templatePrefix)
    }
  }
});
