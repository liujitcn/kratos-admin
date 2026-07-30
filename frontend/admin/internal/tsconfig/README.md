# @liujitcn/admin-tsconfig

管理端 workspace 的共享 TypeScript 基础配置包，仅供当前源码仓库使用，不发布到 npm。

## 文件

| 路径           | 作用                                                          |
| -------------- | ------------------------------------------------------------- |
| `base.json`    | 定义目标语法、Bundler 模块解析、Vue JSX、严格模式和通用类型。 |
| `package.json` | 声明内部包名以及该配置包包含的文件。                          |
| `README.md`    | 当前配置包的文件说明。                                        |

workspace 根 `tsconfig.json` 继承 `base.json`，再声明本仓库开发态路径映射；core、System、宿主和 Vite 配置包继续继承 workspace 根配置。CLI 使用独立的可输出 TypeScript 配置，因为它需要编译 Node.js 可执行文件。

`base.json` 只放所有管理端包共享的编译约束，不放具体包的 `include`、`exclude` 或源码别名。业务模块发布后的解析由各包 `package.json#exports` 负责，不能依赖 workspace 开发态 `paths` 才能工作。
