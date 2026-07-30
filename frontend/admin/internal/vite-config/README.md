# @liujitcn/admin-vite-config

当前源码仓库的宿主 Vite 配置包。它把 core 提供的通用构建能力与本仓库的输出位置组合起来，模块扫描参数由宿主 manifest 传入；本包不作为公共 npm 包发布。

## 目录与文件

```text
internal/vite-config
├── src
│   └── admin.ts
├── package.json
├── README.md
└── tsconfig.json
```

| 路径            | 作用                                                   |
| --------------- | ------------------------------------------------------ |
| `src/admin.ts`  | 接收 module 扫描参数并固定默认宿主的生产输出目录。     |
| `package.json`  | 声明内部包、core 依赖、类型检查命令和 `./admin` 导出。 |
| `README.md`     | 当前配置包的职责和文件说明。                           |
| `tsconfig.json` | 配置 Vite 配置源码的 TypeScript 检查。                 |

`apps/admin/vite.config.ts` 调用本包的 `defineAdminAppViteConfig`，并传入从 `src/module-manifest.ts` 派生的参数：

- `modulePackages`：Vite 需要扫描的业务模块包。
- `optimizeDependencies`：业务模块需要预构建的依赖。
- `outputDirectory`：生产构建写入 `backend/data/admin`，供后端静态资源挂载。

前两项都来自宿主模块 manifest，不在本包维护第二份模块名单。新增宿主时创建独立配置函数和导出子路径，不把宿主特有模块或输出目录写回 core 的通用 Vite Interface。
