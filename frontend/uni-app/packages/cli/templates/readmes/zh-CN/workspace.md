# __PROJECT_NAME__

`__PROJECT_NAME__` 是由 `@liujitcn/kratos-uni-app-cli` 创建的独立 pnpm workspace，支持 H5 和微信小程序。

## 目录结构

```text
__PROJECT_NAME__
├── apps/uni-app          # 私有 uni-app 宿主
├── packages/modules     # workspace 内本地业务模块
├── package.json         # 公共命令和开发依赖
├── pnpm-workspace.yaml  # workspace 包范围
└── tsconfig.json        # 共享 TypeScript 配置
```

宿主只负责入口、manifest、模块清单和启动；底座能力由 `@liujitcn/kratos-uni-app-core` 提供，默认业务页面由 `@liujitcn/kratos-uni-app-system` 提供。本地模块通过各自公开入口接入，不跨包相对引用源码。

## 开发

```bash
pnpm install
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
```

H5 通过局域网 IP 访问时，先在仓库根目录运行 `bash scripts/generate-dev-cert.sh 192.168.1.100` 生成证书，再在 `.env.development-h5.local` 中设置 `VITE_APP_HTTPS=true`；后端使用 HTTPS 时同时设置 `VITE_APP_API_URL=https://localhost:7001`。

模块装配入口是 `apps/uni-app/src/module-manifest.ts`。新增、删除或调整模块时同步维护该清单，并在模块自己的 README 中记录页面和接口职责。

生产构建前请在 `.env.production` 中填写实际 API 和静态资源地址；该文件默认留空，不应使用本机地址。
