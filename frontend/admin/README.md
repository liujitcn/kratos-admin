# frontend/admin

管理后台基于 Vue 3、Vite、TypeScript、Element Plus 和 Pinia，负责登录、AI 助手、系统配置、组织权限、日志、接口管理和代码生成。

## 目录职责

```text
frontend/admin
├── build              # Vite 环境变量、插件和代理
├── src/api            # 手写接口封装
├── src/components     # 通用组件
├── src/layouts        # 页面布局
├── src/routers        # 路由与动态菜单
├── src/rpc            # proto 生成的 TypeScript 类型与客户端
├── src/stores         # Pinia 状态
├── src/views/base     # 登录和公共能力页面
├── src/views/system   # 系统管理页面
└── types              # 自动导入类型
```

## 环境变量

- `.env`：应用标题、端口和构建选项。
- `.env.development`：开发环境接口地址和代理。
- `.env.production`：生产环境公共路径 `/admin/`。

开发代理：

```text
/api     -> http://localhost:7001
/events  -> http://localhost:7001
```

## 开发与构建

```bash
pnpm install
pnpm dev
pnpm type:check
pnpm lint:oxlint
pnpm build
```

默认开发地址为 `http://localhost:8848`，构建产物输出到 `backend/data/admin`。

接口类型由后端 `make ts` 生成到 `src/rpc`。新增或调整接口时，先修改 `backend/api/proto`，再执行生成命令，不要手写生成文件。
