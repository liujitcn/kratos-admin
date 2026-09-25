<!-- prettier-ignore -->
# __MODULE_PACKAGE__

The __MODULE_PASCAL__ business module for the admin app in `__PROJECT_NAME__`. APIs, RPCs, pages, and business components are maintained together in this package, which can be published to npm and composed into different hosts.

## Files and directories

```text
packages/modules/__MODULE_NAME__
├── src
│   ├── api                       # Request files follow the complete Proto path and service file name
│   ├── rpc
│   │   └── README.md             # Proto-generated directory notes
│   ├── components                # Optional components shared inside this module
│   ├── views
│   │   └── index
│   │       └── index.vue
│   ├── index.ts
│   └── module.ts
├── package.json
├── README.md
├── tsconfig.json
└── tsconfig.package.json
```

| Path | Purpose |
| --- | --- |
| `src/index.ts` | npm entry point exporting `__MODULE_IDENTIFIER__`. |
| `src/module.ts` | Collects `src/views/**/*.vue` and declares the `__MODULE_NAME__` module. |
| `src/api/<proto-path>/*.ts` | Request wrappers matching the full Proto path and service filename. |
| `src/rpc/README.md` | Describes this module's RPC generation directory and public package subpath. |
| `src/views/index/index.vue` | Starter page included in the template. |
| `package.json` | Declares dependencies, module entry, API/RPC subpaths, and npm publish settings. |
| `README.md` | Documents this business module's files and integration workflow. |
| `tsconfig.json` | Development-time type-check configuration. |
| `tsconfig.package.json` | Generates declaration files for npm publishing. |

## Development guidelines

- Place APIs under `src/api/<proto-path>` to match the complete Proto path, for example `src/api/base/v1` and `src/api/system/admin/v1`. Keep filenames aligned with service filenames. Preserve full RPC paths under `src/rpc/<proto-domain>/<version>`; do not flatten them.
- Put pages under `src/views`. Keep page-specific components next to the page; create `src/components` only when multiple pages in this module share a component.
- Import core capabilities through public subpaths of `@liujitcn/kratos-admin-core`; do not depend on core source directories.
- Use Vue Router for navigation between modules. Cross-module code reuse requires an explicit public npm export from the other module.
- Prefix dynamic menu component paths with `__MODULE_NAME__/`. The starter page path is `__MODULE_NAME__/index/index`; `index/index` is not supported.
- Replace core static pages only by explicitly mapping them in `staticViews` using the `ADMIN_STATIC_VIEWS` export from core.

The module runtime interface is `__MODULE_IDENTIFIER__`, exported from `src/index.ts`. APIs and RPCs are public through `package.json#exports`; pages are registered only through `AdminModule.views` and are not exported as npm subpaths. Add explicit component exports only for genuine cross-module reuse. Files under `src` do not become public automatically.

Host integration:

```ts
export const adminModuleManifest = [
  {
    packageName: "__MODULE_PACKAGE__",
    load: async () => (await import("__MODULE_PACKAGE__")).__MODULE_IDENTIFIER__
  }
];
```

## Commands

```bash
pnpm --filter __MODULE_PACKAGE__ type:check
pnpm --filter __MODULE_PACKAGE__ build:package
pnpm --filter __MODULE_PACKAGE__ pack
```
