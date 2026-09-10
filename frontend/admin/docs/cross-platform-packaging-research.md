# 管理端 npm 包跨平台构建研究

调研日期：2026-09-10。范围：Windows 安装本项目 npm 包后，自动导入未生效的问题及发布方案。本文记录研究与本地模拟验证，未修改构建代码、安装依赖或发布 npm 包。上游链接为调研时的 `main` / `dev` 分支，不能视为项目当前锁定版本的实现保证。

## 结论

不需要逐个手动导入组件。本项目已经使用成熟的 `unplugin-auto-import` 与 `unplugin-vue-components`；优先规范传入插件的路径，再补真实 Windows 的 npm 包消费测试。当前仅凭 macOS/Linux 构建成功，不能保证 Windows 安装后的源码转换成功。

近期建议使用 Vite 已有的 `normalizePath` 处理当前主机解析出的路径，不为这一处问题额外引入完整打包工具。若需要在任意主机处理其他操作系统的路径字符串，可选插件同源的 `unplugin-utils.normalizePath`；若整个构建系统需要统一路径运算，再评估 `pathe`。更长期可借鉴 Element Plus 的预编译发布，但这是发布架构调整，需要先梳理模块和静态资源边界。

## 当前项目与故障边界

- [`scripts/build-package.mjs`](../scripts/build-package.mjs) 生成类型声明、复制源码并改写别名；[`system/package.json`](../packages/modules/system/package.json) 的 npm 导出仍指向 `dist/package/src/*.ts`。因此 npm 使用方还需运行 Vue 编译与自动导入转换；打包目录叫 `dist` 不代表已经是完整可运行的 JavaScript。
- 当前锁定安装的 `unplugin-auto-import@21.1.0` 与 `unplugin-vue-components@32.1.0` 使用 `unplugin-utils.createFilter`。实查 `unplugin-utils@0.3.2`：匹配前将待匹配 ID 的反斜杠转成 `/`，但不会重写配置者提供的 `RegExp`。若正则由 Windows 原生路径直接构造，即便手工转义正确，也可能与已标准化的 ID 不匹配。这与 [Rollup createFilter 源码](https://github.com/rollup/plugins/blob/master/packages/pluginutils/src/createFilter.ts) 对 RegExp 的处理原则一致。
- 正确顺序是先规范路径分隔符，再转义正则元字符，最后拼接范围和后缀规则。需要同时覆盖 `.ts`、`.vue` 及 Vue 带 query 的模块 ID；只给 `User` 手写 import 无法修复其他文件未被插件转换的问题。
- [现有发布 workflow](../../../.github/workflows/publish-npm.yml) 只在 Ubuntu / Node 24 执行 `make -C frontend package`，没有真实 Windows 的 tarball 消费验证。
- 截图中的 `ElMessage is not defined` 属于运行时导入缺失；HTTP 502 是另一条后端/代理链路，不能期待路径工具一并解决。

## 其他仓库采用什么方案

| 一手来源 | 已确认的实现 | 对本项目的启示 |
| --- | --- | --- |
| [Vite 插件文档](https://vite.dev/guide/api-plugin.html#path-normalization) | Vite 模块 ID 使用 `/`，文档要求比较路径前规范化，并导出 `normalizePath` 和 `createFilter` | 采用框架约定，避免自行猜测插件收到哪种分隔符 |
| [Element Plus 构建源码](https://github.com/element-plus/element-plus/blob/dev/internal/build/src/tasks/modules.ts)、[发布清单](https://github.com/element-plus/element-plus/blob/dev/packages/element-plus/package.json) | Vue/JSX 插件参与 Rolldown 构建，`preserveModules` 保留模块结构，导出 `es/*.mjs` / `lib/*.js`，另有类型与样式产物 | 消费方使用已编译模块，减少对库内部源码转换配置的依赖；不需要把所有组件打成一个大文件 |
| [VueUse 构建配置](https://github.com/vueuse/vueuse/blob/main/tsdown.config.ts) | 使用 tsdown，配置 ES / IIFE、声明产物、Vue 外部依赖，以及 `attw` 包类型解析检查 | 发布不仅是复制文件，还应检查 exports 与类型解析；VueUse 的工具函数构建不能直接证明复杂 Vue SFC、SCSS、动态页面也能零配置迁移 |
| [Vben Admin 库构建配置](https://github.com/vbenjs/vue-vben-admin/blob/main/internal/vite-config/src/config/library.ts)、[插件配置](https://github.com/vbenjs/vue-vben-admin/blob/main/internal/vite-config/src/plugins/index.ts) | 提供 Vite library 模式，输出 ES，按 dependencies / peerDependencies 外部化依赖，并支持声明插件 | 本项目已有 Vite，可以优先复用其库构建能力，而非立即替换整套构建系统；这只是该仓库的库构建配置，不能推断其所有包均这样发布 |

这些实例展示的是路径规则和产物设计，并不证明它们在所有 Windows 环境、包管理器与版本组合下都没有问题。

## 三方工具选型

| 工具 | 解决范围 | 建议 |
| --- | --- | --- |
| Vite `normalizePath` / `createFilter` | 规范实际主机解析出的模块路径、统一插件过滤习惯 | 当前项目首选；已经依赖 Vite。构造 RegExp 前仍需先规范根路径 |
| [unplugin-utils](https://github.com/sxzz/unplugin-utils) | 插件常用路径与过滤工具，当前安装版无条件转换反斜杠 | 需要跨主机处理路径字符串时合适；若源码直接导入，必须声明为 core 的直接依赖，不能依赖 pnpm 间接依赖可见性 |
| [pathe](https://github.com/unjs/pathe) | 对 resolve、join 等路径操作统一输出 `/` | 构建脚本存在广泛路径运算问题时采用；不为一处转换全面替换 `node:path` |
| [unplugin-auto-import](https://github.com/unplugin/unplugin-auto-import)、[unplugin-vue-components](https://github.com/unplugin/unplugin-vue-components) | 前者注入脚本 API/图标导入，后者解析模板组件，可配 `ElementPlusResolver` | 本项目已经具备核心工具，应修正作用范围。类型声明生成成功不等于运行时代码已被转换 |
| [cross-env](https://github.com/kentcdodds/cross-env) | npm scripts 中环境变量赋值的 Windows / POSIX 差异 | 只在存在 shell 环境变量兼容问题时引入，不能修复组件路径过滤 |
| Vite library / tsdown | 生成可发布的 JS、类型与相关产物 | 作为预编译发布候选；仍需配置 Vue 转换、样式、外部依赖、exports 与动态资源 |

Vite 文档示例表达的是跨平台路径规范。在本机安装的 Vite 8.2.2 中，`normalizePath` 含操作系统判断：macOS 上直接传模拟 Windows 反斜杠字符串不会得到预期替换。因此应区分“当前 Windows 主机产生的实际路径”与“在 macOS 测试中构造 Windows 字符串”。不能用后者的行为直接否定其在真实 Windows 上的用途。

## 本次验证结果与后续验证方案

本轮在 macOS 调用了实际安装的 `unplugin-auto-import` 转换实现：使用 `unplugin-utils.normalizePath(root)` 后进行正则转义，模拟 Windows npm 路径、带中文/空格/括号的 pnpm 路径以及 POSIX 路径，均成功注入 `ElMessage` 和 `User`；Vue query 的过滤也通过。这验证了路径规范与插件转换之间的因果关系，但不等于在真实 Windows 运行过安装、构建和浏览器。

建议将以下测试设为发布前门槛：

1. 在发布构建环境生成真实 npm tarball；消费测试必须安装同一批 tarball，避免 workspace 链接或直接引用仓库源码掩盖 `files` / `exports` 问题。
2. 使用 GitHub Actions 的 `windows-latest`、`ubuntu-latest`、`macos-latest`，固定受支持的 Node / pnpm 版本。消费项目覆盖 npm 和 pnpm 的安装布局。矩阵能力见 [GitHub 官方文档](https://docs.github.com/en/actions/using-jobs/using-a-matrix-for-your-jobs)。
3. 在隔离的最小项目里加载 core、system 和一个业务模块，执行类型检查与生产构建；验证自动导入的组件/API、菜单图标、懒加载页面、样式、SVG 和国际化资源。
4. 启动开发服务与生产预览，以浏览器冒烟测试访问登录页和动态页面，主动触发 `ElMessage` 错误提示分支；对接口做固定 mock，避免后端 502 干扰前端验证。
5. 覆盖中文、空格、括号路径和大小写；只把 Linux 当发布机、只检查构建返回码，都会漏掉部分消费环境问题。

如果还承诺用户能够在 Windows 上从源码制造 npm 包，应额外运行 Windows 的 `build:package`。它与“Linux 产出的 tarball 能被 Windows 消费”是两项不同测试；构建脚本的子进程调用、命令 shim 和 Make/shell 前置条件也需一起检查。

## 推荐执行顺序

先统一当前路径过滤并引入三系统 tarball 消费验证，保持自动导入开发体验；再根据验证结果处理剩余实际问题。若未来希望第三方应用无需参与本包内部的自动导入与 SFC 转换，再单独设计预编译发布：库构建阶段注入真实 import，输出 JS / d.ts / 样式，保留按需模块和动态入口，并明确 Vue、Element Plus 等依赖边界。该方案会涉及 core、system、脚手架和消费配置，本轮仅研究，不执行跨模块改造。
