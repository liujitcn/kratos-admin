# @liujitcn/kratos-uni-app

`apps/uni-app` 是 `__PROJECT_NAME__` 的私有 uni-app 宿主，负责组合模块并提供 H5、微信小程序的构建入口，不承载可复用业务实现。

## 目录结构

```text
apps/uni-app
├── src
│   ├── pages/bootstrap      # 固定启动页面
│   ├── App.vue              # 应用根组件和全局样式入口
│   ├── main.ts              # Vue 与 Kratos uni-app 启动入口
│   ├── manifest.json        # uni-app 平台配置
│   ├── module-manifest.ts   # 唯一模块清单
│   └── pages.json           # 固定 bootstrap 路由
├── index.html               # H5 HTML 入口
├── package.json             # 宿主命令和运行依赖
└── vite.config.ts           # 模块页面装配与 uni-app 插件
```

## 模块装配

`src/module-manifest.ts` 默认注册 core 和 system，并按声明顺序加载本地模块与已发布模块。业务页面、API 和状态应放在所属模块，宿主仅维护组合关系和平台配置。

## 命令

在 workspace 根目录执行：

```bash
pnpm dev:h5
pnpm dev:mp-weixin
pnpm build:h5
pnpm build:mp-weixin
```
