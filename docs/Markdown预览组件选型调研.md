# Markdown 预览组件选型调研

## 结论

调研时间：2026-07-31。

最终采用 **`md-editor-v3@6.5.5` 的 `MdPreview`**，并封装为 core 的 `MarkdownPreview` 公共组件，统一替换项目文档、数据库迁移说明和 AI 聊天三个入口。它是候选中唯一同时提供 Vue 3 专用只读组件、完整文章主题、亮暗模式、代码高亮、代码行号和复制等阅读能力的方案，最接近“安装后即可得到类似 Typora 的完整阅读排版”。官方 README 明确列出 `MdPreview` 用法、6 套预览主题和亮暗主题，当前 6.x 也是官方安全支持分支。

迁移不能直接使用默认配置：`MdPreview` 内部的 `markdown-it` 开启了 `html: true`，而 `sanitize` 默认是恒等函数。必须显式传入 DOMPurify 清洗函数，或者在产品明确不支持 Markdown 内嵌 HTML 时，通过统一解析配置禁用 HTML。项目文档由仓库文件同步而来，也不能把“已登录”或“内容来自自己的仓库”当成可信 HTML 边界。

不推荐 `@kangc/v-md-editor`：Vue 3 版本仍挂在 npm 的 `next` 标签，最后一次发布是 2023-11-09，内置依赖停留在 `markdown-it 12`、`highlight.js 10`、`prismjs 1.23` 等旧代际，且官方主题没有完整暗色模式。

`vue-markdown-render` / `markdown-it` 组合适合需要完全自研渲染层的场景，不适合本次“尽快获得成熟 Markdown 阅读样式”的目标。它只负责把 Markdown 变成 HTML，主题、暗色、高亮、行号、复制、锚点规则与安全清洗都由项目长期维护。

## 当前页面约束

- 管理端使用 Vue `^3.5.35`，满足 `md-editor-v3@6.5.5` 的 Vue `^3.5.3` peer 约束。
- 原 `x-markdown-vue@0.0.200` 同时用于项目文档、数据库迁移内容和 AI 聊天，三个入口已经收敛到 core 公共组件，旧依赖和专用样式声明已删除。
- 项目文档页已经自行处理页内锚点、相对 Markdown 文档链接和外部链接；替换渲染器时这些行为要保留，不能只替换模板标签。
- 当前页面需要随管理端切换亮暗主题，并使用项目级 CSS 变量。候选组件的主题能力仍要和现有主题变量做一层适配。

## 候选对比

| 方案                           | 当前版本与活跃度                                                           | Vue 3 / 只读预览                             | 阅读样式与暗色                                                     | 代码能力                                                          | HTML / XSS                                                                                   | npm 包自身解包体积                                                  | 项目维护成本                                   |
| ------------------------------ | -------------------------------------------------------------------------- | -------------------------------------------- | ------------------------------------------------------------------ | ----------------------------------------------------------------- | -------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- | ---------------------------------------------- |
| `md-editor-v3` / `MdPreview`   | `6.5.5`，2026-07-30 发布；仓库同日仍有提交                                 | Vue `^3.5.3`；独立 `MdPreview`，支持按需引用 | 6 套成品预览主题；`light` / `dark`                                 | 内置 Highlight.js 接入、复制、折叠；代码行号默认开启              | **默认不安全**：解析器启用 HTML，`sanitize` 默认原样返回；需显式 DOMPurify                   | 519,975 B、42 文件、21 个直接依赖；预览入口可按需引用               | 低到中：主要维护主题变量、安全回调和链接行为   |
| `@kangc/v-md-editor` / preview | Vue 3 版 `2.3.18` 在 `next` 标签，2023-11-09 发布；仓库最后推送 2024-12-30 | `2.x` peer Vue `^3.0.0`；提供 `v-md-preview` | GitHub、VuePress 主题；发布 CSS 以固定浅色为主，无完整一等暗色切换 | GitHub 主题用 Highlight.js，VuePress 主题用 Prism；行号是额外插件 | 预览组件在写入 `v-html` 前调用内置 `xss.process`，默认比 `MdPreview` 稳妥                    | 2,183,674 B、382 文件、15 个直接依赖                                | 中到高：维护活跃度弱，暗色和旧依赖需要项目接管 |
| `vue-markdown-render`          | `2.3.1`，2026-06-06 发布；仓库同日有提交                                   | peer Vue `^3.3.4`；组件本身就是只读渲染      | 无成品样式、无暗色主题                                             | 无内置高亮或行号；透传 `markdown-it` 插件和配置                   | 默认 `markdown-it` 的 `html: false` 较安全；组件自身没有 sanitizer，启用 HTML 后必须自行清洗 | 80,237 B、23 文件、1 个直接依赖；还需安装/打包 `markdown-it` 及扩展 | 高：全部阅读体验由项目自研                     |
| 直接使用 `markdown-it`         | `15.0.0`，2026-07-30 发布；项目持续维护                                    | 与 Vue 无绑定；需自行封装组件和响应式更新    | 无成品样式、无暗色主题                                             | 官方文档给出 Highlight.js 接法；行号、复制等另选插件或自研        | 默认 `html: false`，官方标注 safe by default；一旦开启 HTML 仍需 sanitizer                   | 1,958,686 B、14 文件、6 个直接依赖                                  | 最高，但拥有最完整的渲染与 DOM 控制权          |

> 体积来自对应版本的 npm `dist.unpackedSize` 和 `dist.fileCount`，只表示包自身安装解包体积，不等于 gzip 后的浏览器产物，也不包含传递依赖。最终前端体积必须以项目构建后的 chunk 报告为准。

## 方案分析

### 1. `md-editor-v3` 的 `MdPreview`

适配点：

- 官方将它定义为 Vue 3 Markdown 编辑器，并提供独立 `MdPreview` 与 `lib/preview.css`，无需加载编辑器 UI。
- 官方列出 `default`、`vuepress`、`github`、`cyanosis`、`mk-cute`、`smart-blue` 六套预览主题；`theme` 类型为 `light | dark`。
- `showCodeRowNumber` 默认 `true`，本页可以关闭以贴近用户提供的 Typora 截图；代码块还内置语言标签、复制和可折叠结构。
- `noHighlight` 默认 `false`。默认高亮脚本和样式从 unpkg CDN 动态加载；内网、离线或严格 CSP 环境应通过全局 `config` 注入本地 Highlight.js 实例，不能依赖外网。
- 可以通过 `noMermaid`、`noKatex`、`noEcharts` 关闭文档页不需要的异步扩展，减少运行时请求和渲染副作用。
- npm 包自身约 508 KiB，并支持预览组件按需引用；但安装时仍会解析包含 CodeMirror 在内的 21 个直接依赖，最终是否比现状更小要以构建产物为准。

安全边界：

- 当前源码用 `markdown-it({ html: true, breaks: true, linkify: true })` 解析。
- 渲染结果仅调用传入的 `sanitize`，而该 prop 默认 `(html) => html`。类型注释也明确推荐 DOMPurify 或 `sanitize-html`。
- 因此 DOMPurify 必须由封装组件所属的 core 包声明为**直接依赖**，不能依赖其他渲染器带来的传递依赖。
- 若启用 Mermaid，`sanitizeMermaid` 也默认原样返回 Promise；本页不需要 Mermaid 时直接关闭更简单。

维护风险：

- 6.x 为官方当前安全支持版本，`SECURITY.md` 明确 `<6.x` 不再支持。
- 2026 年 3 月至 7 月仍有依赖安全、预览转义、渲染容器和只读行为修复，活跃度显著优于其他成品 Vue 候选。
- 主题 CSS 会通过 `@vavt/markdown-theme/css/all.css` 引入全部预览主题；页面仍需要少量作用域覆盖，隐藏非 Typora 风格的代码头或行号，并映射管理端主题变量。

一手来源：

- [官方中文 README：功能、主题、只读用法](https://github.com/imzbf/md-editor-v3/blob/v6.5.5/README-CN.md)
- [官方 `MdPreview` 源码](https://github.com/imzbf/md-editor-v3/blob/v6.5.5/packages/MdPreview/MdPreview.tsx)
- [官方 props：主题、行号、sanitize 默认值](https://github.com/imzbf/md-editor-v3/blob/v6.5.5/packages/MdEditor/props.ts)
- [官方解析源码：`html: true` 与 sanitize 调用](https://github.com/imzbf/md-editor-v3/blob/v6.5.5/packages/MdEditor/layouts/Content/composition/useMarkdownIt.ts)
- [官方运行配置：Highlight.js 等 CDN 默认值](https://github.com/imzbf/md-editor-v3/blob/v6.5.5/packages/MdEditor/config.ts)
- [官方安全支持策略](https://github.com/imzbf/md-editor-v3/blob/v6.5.5/SECURITY.md)
- [官方更新记录](https://github.com/imzbf/md-editor-v3/blob/v6.5.5/CHANGELOG.md)
- [npm 6.5.5 元数据](https://registry.npmjs.org/md-editor-v3/6.5.5)
- [npm 发布页](https://www.npmjs.com/package/md-editor-v3/v/6.5.5)

### 2. `@kangc/v-md-editor`

适配点：

- 官方仓库主分支面向 Vue 3，安装说明要求 `@kangc/v-md-editor@next`；npm 的 `latest` 仍指向 Vue 2 的 `1.7.12`，容易误装。
- Vue 3 的 `2.3.18` peer 约束为 Vue `^3.0.0`，有独立 `v-md-preview`。
- 官方 GitHub 主题用 Highlight.js，VuePress 主题用 Prism；代码行号需要额外注册 `line-number` 插件。
- 预览源码在赋值给 `v-html` 前执行内置 `xss.process`，并提供白名单扩展接口。

不推荐原因：

- `2.3.18` 发布于 2023-11-09，Vue 3 版本长期未进入 `latest`；仓库最后代码推送为 2024-12-30。
- 直接依赖仍包含 `markdown-it ^12.3.2`、`highlight.js ^10.7.2`、`prismjs ^1.23.0`、`xss ^1.0.9`、CodeMirror 5 和 Vant 3。即便暂时没有确认漏洞，后续升级与兼容成本也会落到本项目。
- 发布的 GitHub / VuePress 主题 CSS 使用大量固定浅色值，官方没有与管理端暗色状态直接绑定的一等能力。
- 包自身约 2.08 MiB、382 个文件，比其他候选更重。

一手来源：

- [官方 README：Vue 3、`next` 安装方式](https://github.com/code-farmer-i/vue-markdown-editor/blob/master/README.md)
- [官方预览组件文档](https://github.com/code-farmer-i/vue-markdown-editor/blob/master/docs/zh/examples/preview-demo.md)
- [官方 GitHub 主题文档](https://github.com/code-farmer-i/vue-markdown-editor/blob/master/docs/zh/theme/github.md)
- [官方 VuePress 主题文档](https://github.com/code-farmer-i/vue-markdown-editor/blob/master/docs/zh/theme/vuepress.md)
- [官方代码行号插件文档](https://github.com/code-farmer-i/vue-markdown-editor/blob/master/docs/zh/plugins/line-number.md)
- [官方预览源码：内置 XSS 清洗](https://github.com/code-farmer-i/vue-markdown-editor/blob/master/src/preview.vue)
- [官方 XSS 实现](https://github.com/code-farmer-i/vue-markdown-editor/blob/master/src/utils/xss/index.js)
- [npm 2.3.18 元数据](https://registry.npmjs.org/@kangc%2fv-md-editor/2.3.18)
- [npm 发布页](https://www.npmjs.com/package/@kangc/v-md-editor/v/2.3.18)

### 3. `vue-markdown-render` / `markdown-it`

适配点：

- `vue-markdown-render` 是约 548 B 的发布 JS 包装层，将 `source` 交给 `markdown-it` 后通过 `innerHTML` 渲染；支持透传选项和简单插件。
- `2.3.1` peer 约束 Vue `^3.3.4`，2026-06-06 有新版本，维护状态尚可。
- `markdown-it` 默认 `html: false`、`highlight: null`，README 将默认行为描述为 safe by default；官方文档提供 Highlight.js 的接法。

不推荐作为本次首选的原因：

- 没有任何文章 CSS、暗色主题、代码头、复制、行号或锚点 slug 策略。
- 开启 `options.html = true` 后，包装组件不做额外清洗；安全责任完全在调用方。
- `vue-markdown-render` 本身很薄。若项目最终选择完全自定义，应考虑直接封装 `markdown-it`，避免为了几行 `computed + innerHTML` 再引入一层接口，同时统一 DOMPurify、slug、链接和插件注册。
- 这种方案能最精确复刻 Typora，但那意味着项目要拥有并长期维护一整套 Typora 风格 CSS，不符合本次快速替换成熟组件的目标。

一手来源：

- [`vue-markdown-render` 官方 README](https://github.com/cloudacy/vue-markdown-render/blob/master/README.md)
- [`vue-markdown-render` 全部核心源码](https://github.com/cloudacy/vue-markdown-render/blob/master/src/VueMarkdown.ts)
- [`vue-markdown-render` npm 2.3.1 元数据](https://registry.npmjs.org/vue-markdown-render/2.3.1)
- [`markdown-it` 官方 README](https://github.com/markdown-it/markdown-it/blob/master/README.md)
- [`markdown-it` 默认选项源码](https://github.com/markdown-it/markdown-it/blob/master/src/presets/default.ts)
- [`markdown-it` 高亮用法](https://github.com/markdown-it/markdown-it/blob/master/docs/usage.md#syntax-highlighting)
- [`markdown-it` npm 15.0.0 元数据](https://registry.npmjs.org/markdown-it/15.0.0)

## 推荐迁移配置

底层 `MdPreview` 的配置目标如下，业务页面通过 core 的 `MarkdownPreview` 间接使用：

```vue
<MdPreview
  :id="previewId"
  :model-value="selectedDocument.content"
  :theme="globalStore.isDark ? 'dark' : 'light'"
  preview-theme="github"
  code-theme="github"
  :show-code-row-number="false"
  :sanitize="sanitizeMarkdownHtml"
  :no-mermaid="true"
  :no-katex="true"
  :no-echarts="true"
  :no-img-zoom-in="true"
/>
```

上例是迁移方向，不是可以原样复制的最终实现。落地时还要完成：

1. 把 `md-editor-v3`、DOMPurify 和 Highlight.js 声明为 core 包直接依赖，并由项目命令更新 lockfile。
2. 使用 `MdPreview` 与 `md-editor-v3/lib/preview.css` 的按需入口，不引入完整编辑器样式或编辑器组件。
3. 明确文档是否允许原生 HTML。若不允许，优先禁用解析器 HTML；若需要兼容，使用显式 DOMPurify 策略，并为链接、图片、表格和代码块写安全回归用例。
4. 将高亮器改为本地实例，或明确关闭高亮。不能把默认 unpkg CDN 当成后台系统的稳定依赖。
5. 保留现有点击委托：页内锚点、项目内相对 `.md` 链接、外部链接新窗口与 `noopener,noreferrer`。
6. 为标题提供稳定的 `mdHeadingId`。需要覆盖中文标题、重复标题、显式 URL 编码和跨文档锚点。
7. 删除只服务于 `.x-md-*` DOM 的补丁，保留页面布局；再对 `.md-editor-preview`、`.md-editor-code` 做最少覆盖。若要贴近 Typora，可隐藏代码块头和行号，而不是修改代码正文高度。
8. 同时验证亮色、暗色、移动端、长表格、超长代码、单行/双行代码、图片、引用、任务列表和相对文档跳转。
9. 使用构建报告比较迁移前后的真实异步 chunk，而不是用 npm 解包体积推断浏览器下载量。

## 验收标准

- 单行与双行 fenced code block 高度只随真实内容行数增长，不出现模板空白行。
- 与用户提供的 Typora 参考相比，段落、列表、标题、表格、行内代码和代码块的密度接近；代码块不强制显示不需要的标题栏或行号。
- 亮暗主题切换后正文、边框、代码和链接均有足够对比度，无固定浅色残留。
- `<script>`、事件属性、`javascript:` 链接和恶意 SVG/HTML 不能执行。
- 页内锚点、重复中文标题、项目内相对 Markdown 链接和外部链接保持现有行为。
- 首次打开文档页不依赖公共 CDN；控制台无资源加载错误。
