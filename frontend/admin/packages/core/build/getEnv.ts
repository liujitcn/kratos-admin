import path from "path";

/** 判断当前构建模式是否为开发环境。 */
export function isDevFn(mode: string): boolean {
  return mode === "development";
}

/** 判断当前构建模式是否为生产环境。 */
export function isProdFn(mode: string): boolean {
  return mode === "production";
}

/** 判断当前构建模式是否为测试环境。 */
export function isTestFn(mode: string): boolean {
  return mode === "test";
}

/**
 * Whether to generate package preview
 */
export function isReportMode(): boolean {
  return process.env.VITE_REPORT === "true";
}

// Read all environment variable configuration files to process.env
/** 归一化环境变量：换行转义还原、布尔/端口转类型、代理配置反序列化为对象。 */
export function wrapperEnv(envConf: Recordable): ViteEnv {
  const ret: any = {};

  for (const envName of Object.keys(envConf)) {
    let realName = envConf[envName].replace(/\\n/g, "\n");
    realName = realName === "true" ? true : realName === "false" ? false : realName;
    if (envName === "VITE_PORT") realName = Number(realName);
    if (envName === "VITE_PROXY") {
      try {
        realName = JSON.parse(realName);
      } catch {}
    }
    ret[envName] = realName;
  }
  return ret;
}

/**
 * Get user root directory
 * @param dir file path
 */
export function getRootPath(...dir: string[]) {
  return path.resolve(process.cwd(), ...dir);
}
