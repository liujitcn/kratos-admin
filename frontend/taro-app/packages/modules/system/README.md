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

在项目业务模块的 `src/views/pages/my/my.tsx` 复用公开组件入口：

```tsx
import KratosMyPage from '@liujitcn/kratos-taro-app-system/components/KratosMyPage'
import OrderSection from './components/OrderSection'
import ServiceSection from './components/ServiceSection'

/** MyPage 组合公共个人中心与项目业务卡片。 */
export default function MyPage() {
  return (
    <KratosMyPage>
      <OrderSection />
      <ServiceSection />
    </KratosMyPage>
  )
}
```

项目模块的 `pages` 声明 `'pages/my/my': { style: { navigationStyle: 'custom', navigationBarTitleText: '' } }`，`views` 声明 `PROFILE_HOME: 'pages/my/my'`，并在宿主 `module-manifest.ts` 中排在 system 模块之后。构建期 `buildModule.pages` 复用相同页面配置。使用相同的物理路由可以保留微信原生 tab 切换；页面 wrapper 和底部 tabBar 继续由 runner 生成，不在业务页面重复挂载。

扩展区可以直接放多个业务卡片，不需要复制公共页面或给顶部入口提供替换内容。需要登录态、用户资料或导航时，通过 core 的公开 store 与导航 API 获取。

## AI 助手

AI 页面支持会话创建与切换、历史会话搜索、消息加载、附件上传、复制、删除、再生成、工具提示和快捷入口。Provider 与模型通过 `AiModelService` 加载，知识库通过 `AiKnowledgeService` 加载并可多选，发送消息时携带所选知识库编号。H5 使用 Fetch SSE，微信小程序使用 chunked 请求，两端共用增量 SSE 解析器和相同事件契约。

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
