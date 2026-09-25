<!-- prettier-ignore -->
# __PROJECT_NAME__

An independent business admin workspace built on kratos-admin. It includes a thin host, the System module, and custom modules (__MODULE_NAMES__). Dependencies flow from `apps/admin` to business modules and then to `@liujitcn/kratos-admin-core`.

## Files and directories

```text
__PROJECT_NAME__
├── apps/admin
│   ├── src
│   ├── package.json
│   └── README.md
__MODULE_TREE__
├── scripts
│   └── build-package.mjs
├── .gitignore
├── package.json
├── pnpm-workspace.yaml
├── README.md
├── tsconfig.json
└── turbo.json
```

| Path | Purpose |
| --- | --- |
| `apps/admin/` | Runnable thin host that starts the app and selects enabled business modules. |
__MODULE_TABLE_ROWS__
| `scripts/build-package.mjs` | Produces publishable module source and TypeScript declarations. |
| `.gitignore` | Excludes dependencies, caches, and build artifacts. |
| `package.json` | Defines workspace commands, tooling, and runtime versions. |
| `pnpm-workspace.yaml` | Defines the host and business module workspace packages. |
| `README.md` | Documents this workspace's files and development workflow. |
| `tsconfig.json` | TypeScript configuration and source mappings for custom modules. |
| `turbo.json` | Defines development, build, package-build, and type-check tasks. |

Each directory containing a `package.json` has its own README with details about that package.

## Development

```bash
pnpm install
pnpm dev
pnpm type:check
pnpm build
pnpm build:package
pnpm package
```

| Command | Purpose |
| --- | --- |
| `pnpm dev` | Starts the `apps/admin` development server. |
| `pnpm type:check` | Checks types in the host and all business modules. |
| `pnpm build` | Builds the host application. |
| `pnpm build:package` | Creates npm publish directories for custom modules. |
| `pnpm package` | Builds custom modules and creates npm packages. |

Keep business APIs, RPCs, pages, and components in `packages/modules/<module>/src`. Backend menu component paths for business pages must use the module prefix, such as `shop/index/index`.

The host module manifest is `apps/admin/src/module-manifest.ts`. It loads `@liujitcn/kratos-admin-system` first, followed by custom modules in the order given at creation. `apps/admin/src/modules.ts` loads and exports the enabled modules. When adding another module, update the host dependencies and manifest.

Module registration order only controls explicit `staticViews` replacements: a later module replaces the same static view key. Ordinary business pages are isolated by `<module>/<view>` and cannot replace another module's page by reusing its name.

Use Vue Router for navigation between modules. Cross-module code may only use public npm subpaths; the host must not import module `src` directories.
