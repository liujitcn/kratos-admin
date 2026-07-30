# @liujitcn/admin-lint-config

管理端 workspace 的共享 Oxlint 配置包，仅供当前源码仓库使用，不发布到 npm，也不作为业务模块的运行依赖。

## 文件

| 路径           | 作用                                                      |
| -------------- | --------------------------------------------------------- |
| `oxlint.json`  | 定义 JavaScript、TypeScript、Vue 的正确性和可疑代码规则。 |
| `package.json` | 声明内部包名以及该配置包包含的文件。                      |
| `README.md`    | 当前配置包的文件说明。                                    |

workspace 根命令 `pnpm lint:oxlint` 通过 `--config ./internal/lint-config/oxlint.json` 使用此配置，并对 `apps`、`packages`、`internal`、`scripts` 执行自动修复。该命令不替代 TypeScript 类型检查；涉及类型改动时仍需执行 `pnpm type:check`。

规则调整只在 `oxlint.json` 完成，不在各包复制第二套配置。需要忽略生成目录或构建产物时，优先维护 workspace 根 `.oxlintignore`。
