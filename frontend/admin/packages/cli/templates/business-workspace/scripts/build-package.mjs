import { cp, mkdir, readFile, rm, writeFile } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import { resolve } from "node:path";

const packageRoot = resolve(process.cwd(), process.argv[2] ?? ".");
const declarationRoot = resolve(packageRoot, "dist/declarations");
const outputRoot = resolve(packageRoot, "dist/package");
const packageJson = JSON.parse(await readFile(resolve(packageRoot, "package.json"), "utf8"));

await rm(outputRoot, { recursive: true, force: true });
await rm(declarationRoot, { recursive: true, force: true });
execFileSync("pnpm", ["exec", "vue-tsc", "-p", "tsconfig.package.json"], {
  cwd: packageRoot,
  stdio: "inherit"
});
await mkdir(resolve(outputRoot, "src"), { recursive: true });
await cp(resolve(packageRoot, "src"), resolve(outputRoot, "src"), { recursive: true });
await cp(declarationRoot, resolve(outputRoot, "declarations"), { recursive: true });
await writeFile(resolve(outputRoot, "declarations/index.d.ts"), 'export * from "./src/index";\n');
await rm(declarationRoot, { recursive: true, force: true });
console.log(`已生成 ${packageJson.name} 发布文件`);
