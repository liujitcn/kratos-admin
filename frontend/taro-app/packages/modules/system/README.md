# @liujitcn/kratos-taro-app-system

`@liujitcn/kratos-taro-app-system` 是默认应用业务模块，提供个人中心、设置、个人资料、MFA 绑定和 AI 助手页面。模块只依赖 core 与 UI 的公开入口，不持有宿主配置。

## 页面与视图键

| 页面 | 视图键 | 用途 |
| --- | --- | --- |
| `pages/my/my` | `PROFILE_HOME` | 我的 tab 与登录概览。 |
| `pagesMember/profile/profile` | `PROFILE` | 头像、昵称、性别、邮箱和证件资料。 |
| `pagesMember/settings/settings` | `SETTINGS` | MFA 状态、设置与退出登录。 |
| `pagesMember/ai/index` | `AI` | AI 会话、流式消息、附件和快捷入口。 |

页面配置统一登记在 `src/pages.ts`，运行时映射位于 `src/index.ts`，构建期描述位于 `src/build.ts`。页面私有组件放在页面目录的 `components` 中，不会被 runner 扫描为路由。

## 个人中心扩展

公共组件 `KratosMyPage` 保留顶部标题与设置、个人信息和 AI 助手入口；AI 卡片继续由应用的 `aiEnabled` 配置控制显隐。React `children` 作为业务扩展区，位于这些固定内容之后，与它们共用滚动区域、主题和卡片间距。未传入内容时不增加容器或留白，业务卡片、数据请求和权限由项目模块维护。

公共组件与样式位于 `src/components/KratosMyPage.tsx` 和 `KratosMyPage.scss`。内置 [我的页面](src/views/pages/my/my.tsx) 已使用该组件，并挂载 [MyExtensions 空实现](src/views/pages/my/components/MyExtensions.tsx)。空实现返回 `null`，不显示占位卡片；项目可以按这个结构组织自己的业务内容。

当前模块的 `src/views/pages/my/my.tsx` 通过公开组件入口装配个人中心：

```tsx
import KratosMyPage from '@liujitcn/kratos-taro-app-system/components/KratosMyPage'

/** MyPage 组合公共个人中心与项目业务卡片。 */
export default function MyPage() {
  return (
    <KratosMyPage />
  )
}
```

项目模块的 `pages` 声明 `'pages/my/my': { style: { navigationStyle: 'custom', navigationBarTitleText: '' } }`，`views` 声明 `PROFILE_HOME: 'pages/my/my'`，并在宿主 `module-manifest.ts` 中排在 system 模块之后。构建期 `buildModule.pages` 复用相同页面配置。使用相同的物理路由可以保留微信原生 tab 切换；页面 wrapper 和底部 tabBar 继续由 runner 生成，不在业务页面重复挂载。

当前页面保留空的 children 扩展区，宿主可以在本仓库的模块清单中装配业务内容；公共页面、顶部入口和导航逻辑仍由 System 维护。需要登录态、用户资料或导航时，通过 core 的公开 store 与导航 API 获取。

## AI 助手

AI 页面支持会话创建与切换、历史会话搜索、消息加载、附件上传、复制、回复信息、工具提示和快捷入口。Provider 与模型通过 `AiModelService` 加载，知识库通过 `AiKnowledgeService` 加载并可多选，发送消息时携带所选知识库编号。H5 使用 Fetch SSE，微信小程序使用 chunked 请求，两端共用增量 SSE 解析器和相同事件契约。

请求统一封装在 `src/api`，RPC 类型由 backend Buf 模板生成。页面不能直接写 `Taro.request`，也不能手工修改 `src/rpc`。

## MFA

设置页通过 `@liujitcn/kratos-taro-app-core/api/base/v1/mfa` 查询状态并发起绑定或禁用；页面私有的 `components/PasswordVerifyDialog.tsx` 提供 `setup`、`disable` 两种模式。未启用 MFA 时组件只显示密码输入，已启用 MFA 时在同一组件内同时完成密码和 MFA 校验，绑定成功展示一次性恢复码，禁用成功后清理会话并重新登录。

## 验证

```bash
pnpm --filter @liujitcn/kratos-taro-app-system tsc
pnpm test
pnpm build:h5
pnpm build:mp-weixin
```

AI 共用 RPC 同步人工确认消息类型；应用端会话仅开放读取工具，写操作确认由管理端完成。

## AI 助手只读工作区

AI 助手沿用移动端明亮/暗黑主题。导航栏历史按钮打开左侧抽屉，顶部依次为定时任务、文件、插件，下方为历史会话、新建对话和会话搜索；切换工作区保留普通对话草稿和流式状态。

- 定时任务先显示状态筛选，再显示搜索框，仅在任务卡片中提供执行记录入口；任务详情、冻结执行配置和结果均只读，确认请求仅展示。
- 文件支持全部、上传、生成、回收站分类、搜索与分页。活动文件可预览和下载，回收站只显示元数据。
- 插件支持目录、连接账号、工具三个只读页签，可查看可见范围、应用端可用状态、参数 Schema、授权范围和已有策略。
- 不提供任务新增/编辑/运行/停止、插件配置/连接/授权、文件管理上传/重命名/删除/恢复或移动端审批；历史消息仅复制与查看回复信息。
- 普通对话可新建、发送、停止生成，保留模型、知识库与附件入口，输入区紧贴底部导航；无需手动选择插件。

应用端新普通会话的空插件配置在服务端自动组装当前身份获准的只读工具，再由模型决定是否调用。已有显式引用、管理端会话和任务结果沿用原有工具配置，仍受租户、用户、终端、账号与审核策略约束。请求失败展示错误与重试，空态仅在成功查询后显示。

只读请求封装位于 System 的 `src/api/base/v1/ai_automation.ts`、`ai_file.ts`、`ai_plugin.ts`，类型由统一 Buf 配置生成。文件内容通过统一鉴权文件接口访问；文本以纯文本展示，图片使用平台预览，小程序文档使用系统查看器，H5 其他文档下载查看；浏览器不执行 HTML/SVG 附件内容。
