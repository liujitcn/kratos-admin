import type { ProxyOptions } from "vite";

type ProxyItem = [string, string];

type ProxyList = ProxyItem[];

type ProxyTargetList = Record<string, ProxyOptions>;

/**
 * 创建代理，用于解析 .env.development 代理配置
 * @param list
 */
export function createProxy(list: ProxyList = []) {
  const ret: ProxyTargetList = {};
  for (const [prefix, target] of list) {
    const httpsRE = /^https:\/\//;
    const isHttps = httpsRE.test(target);

    // 当前后端接口本身就带 /api 前缀，开发代理不能再做路径裁剪。
    ret[prefix] = {
      target: target,
      changeOrigin: true,
      ws: true,
      configure(proxy) {
        proxy.on("proxyRes", (upstream, _, downstream) => {
          // 上游在响应体发送完毕前断开时，通知浏览器结束等待并进入恢复流程。
          upstream.once("close", () => {
            if (!upstream.complete) downstream.destroy();
          });
        });
      },
      // https is require secure=false
      ...(isHttps ? { secure: false } : {})
    };
  }
  return ret;
}
