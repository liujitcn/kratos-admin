/* eslint-disable @typescript-eslint/no-require-imports */
const vite = require('vite')
const uni = require('@dcloudio/vite-plugin-uni')

module.exports = {
  defineConfig: vite.defineConfig,
  loadConfigFromFile: vite.loadConfigFromFile,
  loadEnv: vite.loadEnv,
  createKratosUniPlugin: () => uni.default(),
}
