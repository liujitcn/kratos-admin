# __PROJECT_NAME__

`__PROJECT_NAME__` 是由 `@liujitcn/kratos-taro-app-cli` 创建的独立 pnpm workspace，使用 Taro、React 和 TypeScript，支持 H5 与微信小程序。

## 目录结构

```text
__PROJECT_NAME__
├── apps/taro-app             # 私有 Taro 宿主
├── packages/modules          # workspace 内本地业务模块
├── package.json              # 公共命令和开发依赖
├── pnpm-workspace.yaml       # workspace 包范围
└── tsconfig.json             # 共享 TypeScript 配置
```

宿主只负责入口、模块清单和平台构建配置。运行时底座、UI 主题和默认业务页分别由 `@liujitcn/kratos-taro-app-core`、`@liujitcn/kratos-taro-app-ui` 与 `@liujitcn/kratos-taro-app-system` 提供。

## 开发

```bash
pnpm install
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
pnpm tsc
```

开发产物位于 apps/taro-app/dist/dev/<平台>，生产产物位于 apps/taro-app/dist/build/<平台>（平台为 h5 或 mp-weixin）；Kratos 配套项目的 H5 生产产物输出到 backend/web/taro-app。微信开发者工具默认使用 dist/dev/mp-weixin，发布时导入 dist/build/mp-weixin。

模块装配入口是 `apps/taro-app/src/module-manifest.ts`。模块顺序决定静态视图覆盖优先级；新增页面时同步维护模块自己的 `src/pages.ts` 和视图映射。

环境文件位于 workspace 根目录，变量名与 uni-app 保持一致：`VITE_APP_PORT`、`VITE_APP_BASE_PATH`、`VITE_APP_BASE_API`、`VITE_APP_API_URL`、`VITE_APP_STATIC_API`、`VITE_APP_STATIC_URL`。H5 会在基础模式文件上叠加对应的 `*-h5` 文件。

H5 通过局域网 IP 访问时，先在仓库根目录运行 `bash scripts/generate-dev-cert.sh 192.168.1.100` 生成证书，再在 `.env.development-h5.local` 中设置 `VITE_APP_HTTPS=true`；后端使用 HTTPS 时同时设置 `VITE_APP_API_URL=https://localhost:7001`。
