# @liujitcn/kratos-uni-app-cli

`@liujitcn/kratos-uni-app-cli` 提供 `kratos-uni-app` 命令，用于创建独立 pnpm workspace、uni-app 宿主和可选本地业务模块。

## 目录结构

```text
packages/cli
├── assets/favicon.ico          # 随 npm 包发布的宿主图标
├── bin/kratos-uni-app.mjs       # 命令行参数解析与可执行入口
├── src/index.mjs               # workspace 文件生成逻辑
├── test
│   ├── scaffold.test.mjs       # 脚手架结构和参数测试
│   └── packed-scaffold.test.mjs # tarball 内 CLI 与图标复制回归测试
├── README.md                   # CLI 包职责、参数和生成约束
└── package.json
```

CLI 直接由 `src/index.mjs` 生成文件，不依赖外部模板目录。
宿主图标从包内 `assets/favicon.ico` 复制，`assets` 必须包含在 `package.json` 的 `files`
发布白名单中。

## 功能

- 创建包含 `apps/uni-app` 的独立 pnpm workspace。
- 默认接入 core 和 system 包。
- 使用 `--module` 创建 workspace 内本地模块。
- 使用 `--with` 安装额外的已发布模块。
- 校验模块名和包名，目标目录已存在时拒绝覆盖。
- core、system 和本地模块的 core 依赖版本统一跟随当前 CLI 包版本。
- 固定 uni-app、Vite 和 Vue 编译链版本，避免 `latest` 依赖漂移导致脚手架无法启动。
- 生成 H5 环境配置、页面事务恢复入口和独立 workspace 的 core/system 源码别名。
- 生成空的生产环境 API 配置，生产构建前必须填写实际后端地址。
- 生成的宿主在登录、退出和静默退出后重新加载当前身份的导航配置。
- 为 workspace 根目录、`apps/uni-app` 和每个本地模块生成同级 `README.md`；每个包含
  `package.json` 的生成目录都有对应的结构与职责说明。

## 使用

```bash
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app --module orders
pnpm dlx @liujitcn/kratos-uni-app-cli create my-app --with @acme/pay
```

`pnpm test` 包含发布包回归测试：使用 `pnpm pack` 生成 tarball，在临时目录解压后执行
`create --module app`，检查生成的宿主图标内容、HTML 图标引用和本地模块。测试使用本机
`pnpm` 与 `tar`，不安装生成项目的依赖，并在结束后清理临时目录。

包构建和脚手架验证命令见 [workspace 文档](../../README.md)。


### CLI 生成边界

CLI 直接生成完整宿主、本地业务模块、四种语言源文件与注册入口、类型检查、lint 和打包配置。
支持本地 `system` 与其他模块一起创建；管理端本地 System 继承内置能力，运行时只注册一次，
构建仍扫描内置源码。应用端本地模块使用独立导入别名，保留内置 System。

通过 `--kratos-project` 生成与 Go 后端配套的前端：H5 输出到 `backend/data/<terminal>`，
管理端 CLI 同时在 workspace 父目录创建共享 `Makefile` 与 `scripts`。Go 调用方仅传参执行 CLI。
新增语言或修改语言文件后运行 `pnpm i18n:sync`；`pnpm i18n:check` 校验注册文件是否同步。
