# @local/__MODULE_NAME__

`@local/__MODULE_NAME__` 是 `__PROJECT_NAME__` workspace 内的本地 Taro 业务模块，通过 core 的公开接口接入宿主。

- 页面放在 `src/views`，页面配置登记在 `src/pages.ts`。
- 稳定视图键登记在 `src/index.ts`，后注册模块具有更高覆盖优先级。
- 请求、认证、导航和状态能力只能通过公开包入口使用。
- `src/build.ts` 是 runner 使用的构建期描述，不承载运行时逻辑。
