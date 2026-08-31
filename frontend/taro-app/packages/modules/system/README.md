# @liujitcn/kratos-taro-app-system

`@liujitcn/kratos-taro-app-system` 是默认应用业务模块，提供个人中心、设置、个人资料、MFA 绑定和 AI 助手页面。模块只依赖 core 与 UI 的公开入口，不持有宿主配置。

## 页面与视图键

| 页面 | 视图键 | 用途 |
| --- | --- | --- |
| `pages/my/my` | `PROFILE_HOME` | 我的 tab 与登录概览。 |
| `pagesMember/profile/profile` | `PROFILE` | 头像、昵称、性别和地区资料。 |
| `pagesMember/settings/settings` | `SETTINGS` | MFA 状态、设置与退出登录。 |
| `pagesMember/ai/index` | `AI` | AI 会话、流式消息、附件和快捷入口。 |

页面配置统一登记在 `src/pages.ts`，运行时映射位于 `src/index.ts`，构建期描述位于 `src/build.ts`。页面私有组件放在页面目录的 `components` 中，不会被 runner 扫描为路由。

## AI 助手

AI 页面支持会话创建与切换、历史会话搜索、消息加载、附件上传、复制、删除、再生成、工具提示和快捷入口。H5 使用 Fetch SSE，微信小程序使用 chunked 请求，两端共用增量 SSE 解析器和相同事件契约。

请求统一封装在 `src/api`，RPC 类型由 backend Buf 模板生成。页面不能直接写 `Taro.request`，也不能手工修改 `src/rpc`。

## MFA

设置页通过 `@liujitcn/kratos-taro-app-core/api/base/mfa` 查询状态并发起绑定或禁用；页面私有的 `components/PasswordVerifyDialog.tsx` 提供 `setup`、`disable` 两种模式。未启用 MFA 时组件只显示密码输入，已启用 MFA 时在同一组件内同时完成密码和 MFA 校验，绑定成功展示一次性恢复码，禁用成功后清理会话并重新登录。

## 验证

```bash
pnpm --filter @liujitcn/kratos-taro-app-system tsc
pnpm test
pnpm build:h5
pnpm build:mp-weixin
```
