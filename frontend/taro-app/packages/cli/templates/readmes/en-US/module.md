# @local/__MODULE_NAME__

`@local/__MODULE_NAME__` is a local Taro business module in the `__PROJECT_NAME__` workspace. It integrates with the host through the public core APIs.

- Put pages in `src/views` and register page configuration in `src/pages.ts`.
- Register stable view keys in `src/index.ts`; modules declared later take precedence when overriding static views.
- Use only public package entry points for requests, authentication, navigation, and state.
- `src/build.ts` describes the module to the build-time runner and must not contain runtime logic.
