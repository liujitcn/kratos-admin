# @local/__MODULE_NAME__

`@local/__MODULE_NAME__` is a local business module in the `__PROJECT_NAME__` workspace. It integrates with the host through the public APIs of `@liujitcn/kratos-uni-app-core`.

## Directory Structure

```text
packages/modules/__MODULE_NAME__
├── src
│   ├── views               # Module pages; keep page-specific components nearby
│   ├── index.d.mts         # Module entry types
│   └── index.mjs           # Page, view key, and icon registration
├── package.json            # Module name, dependencies, and exports
└── README.md               # Module responsibilities and maintenance notes
```

## Development Rules

- Put pages in `src/views` and register pages and stable view keys in `src/index.mjs`.
- Use only the public exports of `@liujitcn/kratos-uni-app-core` for requests, authentication, navigation, and state.
- Replace pages through stable view keys; APIs must not return arbitrary component paths.
- After changing the module, run the relevant H5 or WeChat Mini Program command from the workspace root to verify assembly.
